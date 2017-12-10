package task

/*
	该策略用于在现货期货做差价
*/

import (
	"fmt"
	"math"
	"sync"
	"time"

	Exchange "madaoQT/exchange"
)

const Explanation = "To make profit from the difference between the future`s price and the current`s"

type StatusType int

const (
	StatusNone StatusType = iota
	StatusProcessing
	StatusOrdering
	StatusError
)

type IAnalyzer struct {
	config *AnalyzerConfig
	// exchanges []ExchangeHandler
	coins map[string]float64

	// futures  map[string]AnalyzeItem
	// currents map[string]AnalyzeItem
	future Exchange.IExchange
	spot   Exchange.IExchange

	event chan RulesEvent

	status StatusType

	ops []OperationItem
}

type OperationItem struct {
	futureConfig Exchange.TradeConfig
	spotConfig   Exchange.TradeConfig
}

type AnalyzerConfig struct {
	Trigger    map[string]float64
	Close      map[string]float64
	LimitArea  float64
	LimitClose float64
}

var defaultConfig = AnalyzerConfig{
	Trigger: map[string]float64{
		"btc": 1.6,
		"ltc": 3,
	},
	Close: map[string]float64{
		"btc": 0.5,
		"ltc": 1.5,
	},
	LimitClose: 0.03,  // 止损幅度
	LimitArea:  0.005, // 允许操作价格的波动范围
}

func (a *IAnalyzer) GetExplanation() string {
	return Explanation
}

func (a *IAnalyzer) WatchEvent() chan RulesEvent {
	return a.event
}

func (a *IAnalyzer) triggerEvent(event EventType, msg interface{}) {
	a.event <- RulesEvent{EventType: event, Msg: msg}
}

func (a *IAnalyzer) defaultConfig() *AnalyzerConfig {
	return &defaultConfig
}

func (a *IAnalyzer) Init(config *AnalyzerConfig) {

	if a.config == nil {
		a.config = a.defaultConfig()
	}

	// 监视币种以及余额
	a.coins = map[string]float64{
		// "btc": 0,
		"ltc/usdt": 1,
	}

	a.status = StatusProcessing
	a.event = make(chan RulesEvent)

	Logger.Info("启动OKEx合约监视程序")
	futureExchange := new(Exchange.OKExAPI)
	futureExchange.Init(Exchange.InitConfig{
		Api:    constOKEXApiKey,
		Secret: constOEXSecretKey,
		Custom: map[string]interface{}{"tradeType": Exchange.TradeTypeFuture},
	})
	futureExchange.Start()

	Logger.Info("启动OKEx现货监视程序")
	spotExchange := new(Exchange.OKExAPI)
	spotExchange.Init(Exchange.InitConfig{
		Api:    constOKEXApiKey,
		Secret: constOEXSecretKey,
		Custom: map[string]interface{}{"tradeType": Exchange.TradeTypeSpot},
	})
	spotExchange.Start()

	go func() {
		for {
			select {
			case event := <-futureExchange.WatchEvent():
				if event == Exchange.EventConnected {
					for k := range a.coins {
						futureExchange.StartContractTicker(k, "this_week", k+"future")
					}
					a.future = Exchange.IExchange(futureExchange)

				} else if event == Exchange.EventError {
					futureExchange.Start()
				}
			case event := <-spotExchange.WatchEvent():
				if event == Exchange.EventConnected {

					for k := range a.coins {
						spotExchange.StartCurrentTicker(k, k+"spot")
					}

					a.spot = spotExchange
				} else if event == Exchange.EventError {
					spotExchange.Start()
				}
			case <-time.After(10 * time.Second):
				if !a.Watch() {
					return
				}
			}
		}
	}()
}

func (a *IAnalyzer) Watch() bool {

	for coinName, _ := range a.coins {

		if a.status == StatusError {
			Logger.Debug("状态异常")
			return false
		}

		valuefuture := a.future.GetTickerValue(coinName + "future")
		valueCurrent := a.spot.GetTickerValue(coinName + "spot")

		difference := (valuefuture.Last - valueCurrent.Last) * 100 / valueCurrent.Last
		msg := fmt.Sprintf("币种:%s, 合约价格：%.2f, 现货价格：%.2f, 价差：%.2f%%",
			coinName, valuefuture.Last, valueCurrent.Last, difference)

		Logger.Info(msg)

		a.triggerEvent(EventTypeTrigger, msg)

		if a.checkPosition(coinName, valuefuture.Last, valueCurrent.Last) {
			Logger.Info("持仓中...不做交易")
			continue
		}

		if valuefuture != nil && valueCurrent != nil {

			a.triggerEvent(EventTypeTrigger, "===============================")

			if math.Abs(difference) > a.config.Trigger[coinName] {
				if valuefuture.Last > valueCurrent.Last {
					Logger.Info("卖出合约，买入现货")

					a.placeOrdersByQuantity(a.future, Exchange.TradeConfig{
						Coin:   coinName,
						Type:   Exchange.OrderTypeOpenShort,
						Price:  valuefuture.Last,
						Amount: 5,
						Limit:  a.config.LimitArea,
					},
						a.spot, Exchange.TradeConfig{
							Coin:   coinName,
							Type:   Exchange.OrderTypeBuy,
							Price:  valueCurrent.Last,
							Amount: 50 / valueCurrent.Last,
							Limit:  a.config.LimitArea,
						})

				} else {
					Logger.Info("买入合约, 卖出现货")

					a.placeOrdersByQuantity(a.future, Exchange.TradeConfig{
						Coin:   coinName,
						Type:   Exchange.OrderTypeOpenLong,
						Price:  valuefuture.Last,
						Amount: 5,
						Limit:  a.config.LimitArea,
					},
						a.spot, Exchange.TradeConfig{
							Coin:   coinName,
							Type:   Exchange.OrderTypeSell,
							Price:  valueCurrent.Last,
							Amount: 50 / valueCurrent.Last,
							Limit:  a.config.LimitArea,
						})
				}

				a.triggerEvent(EventTypeTrigger, "===============================")
			}
		}
	}

	return true

}

/*
	根据持仓量限价买入
*/
func (a *IAnalyzer) placeOrdersByQuantity(future Exchange.IExchange, futureConfig Exchange.TradeConfig,
	spot Exchange.IExchange, spotConfig Exchange.TradeConfig) {

	if a.status != StatusProcessing {
		Logger.Infof("Invalid Status %v", a.status)
		return
	}

	channelFuture := ProcessTradeRoutine(future, futureConfig)
	channelSpot := ProcessTradeRoutine(spot, spotConfig)

	var waitGroup sync.WaitGroup
	var futureResult, spotResult TradeResult

	waitGroup.Add(1)
	go func() {
		select {
		case futureResult = <-channelFuture:
			Logger.Debugf("合约交易结果:%v", futureResult)
			waitGroup.Done()
		}
	}()

	waitGroup.Add(1)
	go func() {
		select {
		case spotResult = <-channelSpot:
			Logger.Debugf("现货交易结果:%v", spotResult)
			waitGroup.Done()
		}
	}()

	waitGroup.Wait()
	operation := OperationItem{}

	if futureConfig.Type == Exchange.OrderTypeOpenLong {
		futureConfig.Type = Exchange.OrderTypeCloseLong
	} else if futureConfig.Type == Exchange.OrderTypeOpenShort {
		futureConfig.Type = Exchange.OrderTypeCloseShort
	} else {
		Logger.Error("Invalid Operation for the future")
		return
	}

	if spotConfig.Type == Exchange.OrderTypeSell {
		spotConfig.Type = Exchange.OrderTypeBuy
	} else if spotConfig.Type == Exchange.OrderTypeBuy {
		spotConfig.Type = Exchange.OrderTypeSell
	} else {
		Logger.Error("Invalid Operation for the future")
		return
	}

	// futureConfig.Amount = futureResult.Balance
	spotConfig.Amount = spotResult.Balance

	operation.futureConfig = futureConfig
	operation.spotConfig = spotConfig

	if futureResult.Error == nil && spotResult.Error == nil {
		Logger.Debug("锁仓成功")
		a.ops = append(a.ops, operation)
		return
	} else if futureResult.Error == nil && spotResult.Error != nil {
		channelSpot = ProcessTradeRoutine(spot, operation.spotConfig)

		select {

		case spotResult = <-channelSpot:
			Logger.Debugf("现货平仓结果:%v", spotResult)
			if spotResult.Error != nil {
				Logger.Errorf("平仓失败，请手工检查:%v", spotResult)
				a.status = StatusError
			}
		}
	} else if futureResult.Error != nil && spotResult.Error == nil {
		channelFuture = ProcessTradeRoutine(spot, operation.futureConfig)

		select {

		case futureResult = <-channelFuture:
			Logger.Debugf("合约平仓结果:%v", futureResult)
			if futureResult.Error != nil {
				Logger.Errorf("平仓失败，请手工检查:%v", futureResult)
				a.status = StatusError
			}
		}
	} else {
		Logger.Errorf("无法建立仓位：%v, %v", futureResult, spotResult)
		a.status = StatusError
	}

	return

}

func (a *IAnalyzer) checkPosition(coin string, futurePrice float64, spotPrice float64) bool {
	if a.ops != nil {
		for _, op := range a.ops {

			closeConditions := []bool{
				!InPriceArea(futurePrice, op.futureConfig.Price, a.config.LimitClose), // 防止爆仓
				// !InPriceArea(spotPrice, op.spotConfig.Price, a.config.LimitClose),
				math.Abs((futurePrice-spotPrice)*100/spotPrice) < a.config.Close[coin],
			}

			Logger.Debugf("Conditions:%v", closeConditions)

			for _, condition := range closeConditions {

				if condition {
					Logger.Error("条件平仓...")
					op.futureConfig.Price = futurePrice
					op.spotConfig.Price = spotPrice
					channelFuture := ProcessTradeRoutine(a.future, op.futureConfig)
					channelSpot := ProcessTradeRoutine(a.spot, op.spotConfig)

					var waitGroup sync.WaitGroup
					var futureResult, spotResult TradeResult

					waitGroup.Add(1)
					go func() {
						select {
						case futureResult = <-channelFuture:
							Logger.Debugf("合约交易结果:%v", futureResult)
							waitGroup.Done()
						}
					}()

					waitGroup.Add(1)
					go func() {
						select {
						case spotResult = <-channelSpot:
							Logger.Debugf("现货交易结果:%v", spotResult)
							waitGroup.Done()
						}
					}()

					waitGroup.Wait()

					if futureResult.Error == nil && spotResult.Error == nil {
						Logger.Info("平仓完成")
					} else {
						Logger.Error("平仓失败，请手工检查")
						a.status = StatusError
					}
				}

			}
		}

		return true
	}

	return false
}
