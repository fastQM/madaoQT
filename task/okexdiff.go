package task

/*
	该策略用于在现货期货做差价
*/

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
	// "time"

	Exchange "madaoQT/exchange"
	Mongo "madaoQT/mongo"
	Utils "madaoQT/utils"

	Message "madaoQT/server/websocket"

	Websocket "github.com/gorilla/websocket"
)

const Explanation = "To make profit from the difference between the future`s price and the current`s"

type StatusType int

const (
	StatusNone StatusType = iota
	StatusProcessing
	StatusOrdering
	StatusQuit
	StatusError
)

type IAnalyzer struct {
	config AnalyzerConfig
	// exchanges []ExchangeHandler
	coins map[string]float64

	// futures  map[string]AnalyzeItem
	// currents map[string]AnalyzeItem
	future Exchange.IExchange
	spot   Exchange.IExchange

	status StatusType

	ops     map[uint]*OperationItem
	opIndex uint

	tradeDB *Mongo.Trades
	orderDB *Mongo.Orders

	conn *Websocket.Conn
}

type OperationItem struct {
	futureConfig Exchange.TradeConfig
	spotConfig   Exchange.TradeConfig
}

type AnalyzerConfig struct {
	// APIKey     string
	// SecretKey  string
	Area       map[string]TriggerArea `json:"area"`
	LimitOpen  float64                `json:"limitopen"`
	LimitClose float64                `json:"limitclose"`
}

type TriggerArea struct {
	Open  float64 `json:"open"`
	Close float64 `json:"close"`
}

var defaultConfig = AnalyzerConfig{
	// Trigger: map[string]float64{
	// 	"btc": 1.6,
	// 	"ltc": 3,
	// },
	// Close: map[string]float64{
	// 	"btc": 0.5,
	// 	"ltc": 1.5,
	// },
	Area: map[string]TriggerArea{
		"btc": {1.6, 0.5},
		"ltc": {3, 1.5},
	},
	LimitClose: 0.03,  // 止损幅度
	LimitOpen:  0.005, // 允许操作价格的波动范围
}

func (a *IAnalyzer) GetTaskName() string {
	return "okexdiff"
}

func (a *IAnalyzer) GetTaskExplanation() *TaskExplanation {
	return &TaskExplanation{"okexdiff", "okexdiff"}
}

func (a *IAnalyzer) GetDefaultConfig() interface{} {
	return defaultConfig
}

func (a *IAnalyzer) wsConnect() {
	url := "ws://localhost:8080/websocket"
	c, _, err := Websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		Logger.Errorf("Fail to dial: %v", err)
		return
	}

	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				Logger.Errorf("Fail to read:%v", err)
				return
			}

			Logger.Infof("message:%v", string(message))
		}
	}()

	a.conn = c
}

func (a *IAnalyzer) wsPublish(topic string, message string) {
	if a.conn != nil {
		message := Message.PackageRequestMsg(0, Message.CmdTypePublish, topic, message)
		if message != nil {
			if err := a.conn.WriteMessage(Websocket.TextMessage, message); err != nil {
				Logger.Errorf("Fail to write message:%v", err)
			}
		}
	}
}

func (a *IAnalyzer) Start(configJSON string) {

	if a.status != StatusNone {
		return
	}

	a.status = StatusProcessing

	Logger.Debugf("Configure:%v", configJSON)
	// config := a.GetDefaultConfig().(AnalyzerConfig)
	var config AnalyzerConfig
	if configJSON != "" {
		err := json.Unmarshal([]byte(configJSON), &config)
		if err != nil {
			log.Printf("Fail to get config:%v", err)
			return
		}

		config = a.GetDefaultConfig().(AnalyzerConfig)
	}
	Logger.Debugf("Configure:%v", config)
	a.config = config

	Logger.Infof("Config:%v", a.config)
	// 监视币种以及余额
	a.coins = map[string]float64{
		// "btc": 0,
		"ltc/usdt": 1,
	}

	a.ops = make(map[uint]*OperationItem)

	a.tradeDB = &Mongo.Trades{
		Config: &Mongo.DBConfig{
			CollectionName: "DiffOKExFutureSpotTrade",
		},
	}

	err := a.tradeDB.Connect()
	if err != nil {
		Logger.Errorf("tradeDB error:%v", err)
		return
	}

	a.orderDB = &Mongo.Orders{
		Config: &Mongo.DBConfig{
			CollectionName: "DiffOKExFutureSpotOrder",
		},
	}

	err = a.orderDB.Connect()
	if err != nil {
		Logger.Errorf("orderDB error:%v", err)
		return
	}

	Logger.Info("启动OKEx合约监视程序")
	futureExchange := Exchange.NewOKExFutureApi(nil) // 交易需要api
	futureExchange.Start()

	Logger.Info("启动OKEx现货监视程序")
	spotExchange := Exchange.NewOKExSpotApi(nil)
	spotExchange.Start()

	a.wsConnect()

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
				if a.status == StatusError || a.status == StatusQuit {
					Logger.Debug("状态异常")
					return
				}

				a.Watch()
			}
		}
	}()
}

func (a *IAnalyzer) Watch() {

	for coinName, _ := range a.coins {

		valuefuture := a.future.GetTickerValue(coinName + "future")
		valueCurrent := a.spot.GetTickerValue(coinName + "spot")

		difference := (valuefuture.Last - valueCurrent.Last) * 100 / valueCurrent.Last
		msg := fmt.Sprintf("币种:%s, 合约价格：%.2f, 现货价格：%.2f, 价差：%.2f%%",
			coinName, valuefuture.Last, valueCurrent.Last, difference)

		Logger.Info(msg)

		a.wsPublish("test", msg)

		if a.checkPosition(coinName, valuefuture.Last, valueCurrent.Last) {
			Logger.Info("持仓中...不做交易")
			continue
		}

		if valuefuture != nil && valueCurrent != nil {

			if math.Abs(difference) > a.config.Area[coinName].Open {
				if valuefuture.Last > valueCurrent.Last {
					Logger.Info("卖出合约，买入现货")

					batch := Utils.GetRandomHexString(12)

					a.placeOrdersByQuantity(a.future, Exchange.TradeConfig{
						Batch:  batch,
						Coin:   coinName,
						Type:   Exchange.TradeTypeOpenShort,
						Price:  valuefuture.Last,
						Amount: 5,
						Limit:  a.config.LimitOpen,
					},
						a.spot, Exchange.TradeConfig{
							Batch:  batch,
							Coin:   coinName,
							Type:   Exchange.TradeTypeBuy,
							Price:  valueCurrent.Last,
							Amount: 50 / valueCurrent.Last,
							Limit:  a.config.LimitOpen,
						})

				} else {
					Logger.Info("买入合约, 卖出现货")

					batch := Utils.GetRandomHexString(12)

					a.placeOrdersByQuantity(a.future, Exchange.TradeConfig{
						Batch:  batch,
						Coin:   coinName,
						Type:   Exchange.TradeTypeOpenLong,
						Price:  valuefuture.Last,
						Amount: 5,
						Limit:  a.config.LimitOpen,
					},
						a.spot, Exchange.TradeConfig{
							Batch:  batch,
							Coin:   coinName,
							Type:   Exchange.TradeTypeSell,
							Price:  valueCurrent.Last,
							Amount: 50 / valueCurrent.Last,
							Limit:  a.config.LimitOpen,
						})
				}
			}
		}
	}

	return

}

func (a *IAnalyzer) Close() {
	a.status = StatusQuit

	a.future.Close()
	a.spot.Close()

	if a.conn != nil {
		a.conn.Close()
	}

	if a.orderDB != nil {
		a.orderDB.Close()
	}

	if a.tradeDB != nil {
		a.tradeDB.Close()
	}
}

/*
	根据持仓量限价买入
*/
func (a *IAnalyzer) placeOrdersByQuantity(future Exchange.IExchange, futureConfig Exchange.TradeConfig,
	spot Exchange.IExchange, spotConfig Exchange.TradeConfig) {

	if true {
		return
	}

	if a.status != StatusProcessing {
		Logger.Infof("Invalid Status %v", a.status)
		return
	}

	channelFuture := ProcessTradeRoutine(future, futureConfig, a.tradeDB, a.orderDB)
	channelSpot := ProcessTradeRoutine(spot, spotConfig, a.tradeDB, a.orderDB)

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

	futureConfig.Type = Exchange.RevertTradeType(futureConfig.Type)
	spotConfig.Type = Exchange.RevertTradeType(spotConfig.Type)

	// futureConfig.Amount = futureResult.Balance
	spotConfig.Amount = math.Trunc(spotResult.Balance*100) / 100

	Logger.Debugf("spotConfig.Amount:%v", spotConfig.Amount)

	operation.futureConfig = futureConfig
	operation.spotConfig = spotConfig

	if futureResult.Error == TradeErrorSuccess && spotResult.Error == TradeErrorSuccess {

		Logger.Debug("锁仓成功")
		a.ops[a.opIndex] = &operation
		a.opIndex++
		return

	} else if futureResult.Error == TradeErrorSuccess && spotResult.Error != TradeErrorSuccess {

		channelSpot = ProcessTradeRoutine(spot, operation.spotConfig, a.tradeDB, a.orderDB)
		select {

		case spotResult = <-channelSpot:
			Logger.Debugf("现货平仓结果:%v", spotResult)
			if spotResult.Error != TradeErrorSuccess {
				Logger.Errorf("平仓失败，请手工检查:%v", spotResult)
				a.status = StatusError
			}
		}

	} else if futureResult.Error != TradeErrorSuccess && spotResult.Error == TradeErrorSuccess {

		channelFuture = ProcessTradeRoutine(spot, operation.futureConfig, a.tradeDB, a.orderDB)
		select {
		case futureResult = <-channelFuture:
			Logger.Debugf("合约平仓结果:%v", futureResult)
			if futureResult.Error != TradeErrorSuccess {
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
	if len(a.ops) != 0 {
		for index, op := range a.ops {

			if op == nil {
				Logger.Debug("Invalid operation")
				continue
			}

			closeConditions := []bool{
				!InPriceArea(futurePrice, op.futureConfig.Price, a.config.LimitClose), // 防止爆仓
				// !InPriceArea(spotPrice, op.spotConfig.Price, a.config.LimitClose),
				math.Abs((futurePrice-spotPrice)*100/spotPrice) < a.config.Area[coin].Close,
				// a.status ==
			}

			Logger.Debugf("Conditions:%v", closeConditions)

			for _, condition := range closeConditions {

				if condition {

					Logger.Error("条件平仓...")

					op.futureConfig.Price = futurePrice
					op.spotConfig.Price = spotPrice

					channelFuture := ProcessTradeRoutine(a.future, op.futureConfig, a.tradeDB, a.orderDB)
					channelSpot := ProcessTradeRoutine(a.spot, op.spotConfig, a.tradeDB, a.orderDB)

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

					if futureResult.Error == TradeErrorSuccess && spotResult.Error == TradeErrorSuccess {
						Logger.Info("平仓完成")
						delete(a.ops, index)
					} else {
						Logger.Error("平仓失败，请手工检查")
						a.status = StatusError
					}

					break
				}

			}
		}

		return true
	}

	return false
}
