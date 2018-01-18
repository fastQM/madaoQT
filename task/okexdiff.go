package task

/*
	该策略用于在现货期货做差价
*/

import (
	"madaoQT/utils"
	"encoding/json"
	"errors"
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
	StatusError
)

type IAnalyzer struct {
	config AnalyzerConfig
	// exchanges []ExchangeHandler
	// coins map[string]float64

	// futures  map[string]AnalyzeItem
	// currents map[string]AnalyzeItem
	future Exchange.IExchange
	spot   Exchange.IExchange
	fund   *OkexFundManage

	status StatusType

	ops     map[uint]*OperationItem
	opIndex uint

	tradeDB *Mongo.Trades
	// orderDB *Mongo.Orders
	diffDB *Mongo.OKExDiff

	conn *Websocket.Conn
}

type OperationItem struct {
	futureConfig Exchange.TradeConfig
	spotConfig   Exchange.TradeConfig
}

type AnalyzerConfig struct {
	API        string
	Secret     string
	Area       map[string]TriggerArea `json:"area"`
	LimitOpen  float64                `json:"limitopen"`
	LimitClose float64                `json:"limitclose"`
}

type TriggerArea struct {
	Open     float64 `json:"open"`
	Close    float64 `json:"close"`
	Position float64 `json:"position"`
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
		"btc": {1.6, 0.5, 10},
		"ltc": {3, 1.5, 10},
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
		Logger.Debugf("WriteMessage:%s", string(message))
		if message != nil {
			if err := a.conn.WriteMessage(Websocket.TextMessage, message); err != nil {
				Logger.Errorf("Fail to write message:%v", err)
			}
		}
	}
}

func (a *IAnalyzer) GetBalances() map[string]interface{} {

	var spots []map[string]interface{}
	var futures []map[string]interface{}

	if a.spot != nil {
		if balances := a.spot.GetBalance();balances != nil{
			for coin := range a.config.Area {
				balance := balances[coin]
				spots = append(spots, map[string]interface{}{
					"name":    coin,
					"balance": balance.(map[string]interface{})["balance"].(float64),
				})
			}

			balance := balances["usdt"]
			spots = append(spots, map[string]interface{}{
				"name":    "usdt",
				"balance": balance.(map[string]interface{})["balance"].(float64),
			})
		}
	}

	if a.future != nil {
		if balances := a.future.GetBalance();balances != nil{
			for coin := range a.config.Area {
				balance := balances[coin]
				futures = append(futures, map[string]interface{}{
					"name":    coin,
					"balance": balance.(map[string]interface{})["balance"].(float64),
					"bond":    balance.(map[string]interface{})["bond"].(float64),
				})
			}
		}
	}

	return map[string]interface{}{
		"spots":   spots,
		"futures": futures,
	}
}

func (a *IAnalyzer) GetPositions() {

}

func (a *IAnalyzer) GetTrades() []Mongo.TradesRecord {
	if a.tradeDB != nil {
		err, records := a.tradeDB.FindAll()
		if err != nil {
			Logger.Errorf("Fail to get trades:%v", err)
			return nil
		}

		return records
	}

	return nil
}

// func (a *IAnalyzer) GetOrders() []Mongo.OrderInfo {
// 	if a.orderDB != nil {
// 		err, orders := a.orderDB.FindAll()
// 		if err != nil {
// 			Logger.Errorf("Fail to get orders:%v", err)
// 			return nil
// 		}
// 		return orders
// 	}
// 	return nil
// }

func (a *IAnalyzer) Start(api string, secret string, configJSON string) error {

	if a.status != StatusNone {
		return errors.New(TaskErrorMsg[TaskErrorStatus])
	}

	Logger.Debugf("Configure:%v", configJSON)

	if configJSON != "" {
		var config AnalyzerConfig
		err := json.Unmarshal([]byte(configJSON), &config)
		if err != nil {
			log.Printf("Fail to get config:%v", err)
			return errors.New(TaskErrorMsg[TaskInvalidConfig])
		}
		a.config = config
	} else {
		a.config = a.GetDefaultConfig().(AnalyzerConfig)
	}

	Logger.Infof("Config:%v", a.config)

	a.ops = make(map[uint]*OperationItem)

	a.tradeDB = &Mongo.Trades{
		Config: &Mongo.DBConfig{
			CollectionName: "DiffOKExFutureSpotTrade",
		},
	}

	err := a.tradeDB.Connect()
	if err != nil {
		Logger.Errorf("tradeDB error:%v", err)
		return errors.New(TaskErrorMsg[TaskLostMongodb])
	}

	a.diffDB = &Mongo.OKExDiff{
		Config: &Mongo.DBConfig{
			CollectionName: "DiffOKExHistory",
		},
	}

	err = a.diffDB.Connect()
	if err != nil {
		Logger.Errorf("tradeDB error:%v", err)
		return errors.New(TaskErrorMsg[TaskLostMongodb])
	}

	// a.orderDB = &Mongo.Orders{
	// 	Config: &Mongo.DBConfig{
	// 		CollectionName: "DiffOKExFutureSpotOrder",
	// 	},
	// }

	// err = a.orderDB.Connect()
	// if err != nil {
	// 	Logger.Errorf("orderDB error:%v", err)
	// 	return errors.New(TaskErrorMsg[TaskLostMongodb])
	// }

	Logger.Info("启动OKEx合约监视程序")
	// futureExchange := Exchange.NewOKExFutureApi(&Exchange.Config{
	// 	API:    api,
	// 	Secret: secret,
	// })
	futureExchange := new(Exchange.OKExAPI)

	futureExchange.SetConfigure(Exchange.Config{
		API:    api,
		Secret: secret,
		Custom: map[string]interface{}{
			"exchangeType": Exchange.ExchangeTypeFuture,
			"period": "this_week",
		},
	})
	futureExchange.Start()

	Logger.Info("启动OKEx现货监视程序")
	spotExchange := Exchange.NewOKExSpotApi(&Exchange.Config{
		API:    api,
		Secret: secret,
	})
	spotExchange.Start()

	a.fund = new(OkexFundManage)
	a.fund.Init()

	a.wsConnect()

	a.status = StatusProcessing

	go func() {
		for {
			select {
			case event := <-futureExchange.WatchEvent():
				if event == Exchange.EventConnected {
					utils.SleepAsyncBySecond(3)
					for k := range a.config.Area {
						k = (k + "/usdt")
						futureExchange.StartTicker(k)
					}
					a.future = Exchange.IExchange(futureExchange)

				} else if event == Exchange.EventLostConnection {
					futureExchange.Start()
				}
			case event := <-spotExchange.WatchEvent():
				if event == Exchange.EventConnected {
					utils.SleepAsyncBySecond(3)
					for k := range a.config.Area {
						k = (k + "/usdt")
						spotExchange.StartTicker(k)
					}

					a.spot = spotExchange
				} else if event == Exchange.EventLostConnection {
					spotExchange.Start()
				}
			case <-time.After(10 * time.Second):
				if a.status == StatusError || a.status == StatusNone {
					Logger.Debug("状态异常或退出")
					return
				}

				a.Watch()
			}
		}
	}()

	return nil
}

func (a *IAnalyzer) Watch() {

	// var orderAmount = 50
	// var orderFutureQuantity = orderAmount / 10

	for coin := range a.config.Area {
		pair := coin + "/usdt"

		valueFuture := a.future.GetTicker(pair)
		valueCurrent := a.spot.GetTicker(pair)

		if valueFuture == nil || valueCurrent == nil {
			Logger.Errorf("not valid ticker")
			return
		}

		Logger.Debugf("Current Coin:%v spot:%v future:%v", pair, valueCurrent, valueFuture)
		difference := (valueFuture.Last - valueCurrent.Last) * 100 / valueCurrent.Last
		msg := fmt.Sprintf("币种:%s, 合约价格：%.2f, 现货价格：%.2f, 价差：%.2f%%",
			pair, valueFuture.Last, valueCurrent.Last, difference)

		a.diffDB.Insert(Mongo.DiffValue{
			Coin:         coin,
			SpotPrice:    valueCurrent.Last,
			SpotVolume:   valueCurrent.Volume,
			FuturePrice:  valueFuture.Last,
			FutureVolume: valueFuture.Volume,
			Diff:         difference,
			Time:         time.Now(),
		})

		Logger.Info(msg)

		a.wsPublish("okexdiff", msg)

		if a.checkPosition(coin, valueFuture.Last, valueCurrent.Last) {
			Logger.Info("持仓中...不做交易")
			continue
		}

		if valueFuture != nil && valueCurrent != nil {
			if math.Abs(difference) > a.config.Area[coin].Open {
				if valueFuture.Last > valueCurrent.Last {
					Logger.Info("卖出合约，买入现货")

					batch := Utils.GetRandomHexString(12)

					a.placeOrdersByQuantity(a.future, Exchange.TradeConfig{
						Batch:  batch,
						Pair:   pair,
						Type:   Exchange.TradeTypeOpenShort,
						Price:  valueFuture.Last,
						Amount: 5,
						Limit:  a.config.LimitOpen,
					},
						a.spot, Exchange.TradeConfig{
							Batch:  batch,
							Pair:   pair,
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
						Pair:   pair,
						Type:   Exchange.TradeTypeOpenLong,
						Price:  valueFuture.Last,
						Amount: 5,
						Limit:  a.config.LimitOpen,
					},
						a.spot, Exchange.TradeConfig{
							Batch:  batch,
							Pair:   pair,
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

	a.status = StatusNone

	if a.future != nil {
		a.future.Close()
	}

	if a.spot != nil {
		a.spot.Close()
	}

	if a.conn != nil {
		a.conn.Close()
	}

	// if a.orderDB != nil {
	// 	a.orderDB.Close()
	// }

	if a.tradeDB != nil {
		a.tradeDB.Close()
	}

	if a.fund != nil {
		a.fund.Close()
	}
}

/*
	根据持仓量限价买入
*/
func (a *IAnalyzer) placeOrdersByQuantity(future Exchange.IExchange, futureConfig Exchange.TradeConfig,
	spot Exchange.IExchange, spotConfig Exchange.TradeConfig) {

	// if true {
	// 	return
	// }

	if a.status != StatusProcessing {
		Logger.Infof("Invalid Status %v", a.status)
		return
	}

	pair := Exchange.ParsePair(futureConfig.Pair)
	a.fund.OpenPosition(Exchange.ExchangeTypeFuture, future, futureConfig.Batch, pair[0], futureConfig.Type)
	a.fund.OpenPosition(Exchange.ExchangeTypeSpot, spot, spotConfig.Batch, pair[0], spotConfig.Type)

	channelFuture := ProcessTradeRoutine(future, futureConfig, a.tradeDB)
	channelSpot := ProcessTradeRoutine(spot, spotConfig, a.tradeDB)

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

	futureConfig.Amount = futureResult.DealAmount
	spotConfig.Amount = math.Trunc(spotResult.DealAmount*100) / 100 //如果数值是0.299,可交易数量是0.29

	Logger.Debugf("spotConfig.Amount:%v", spotConfig.Amount)

	operation.futureConfig = futureConfig
	operation.spotConfig = spotConfig

	if futureResult.Error == TaskErrorSuccess && spotResult.Error == TaskErrorSuccess {

		Logger.Debug("锁仓成功")
		a.ops[a.opIndex] = &operation
		a.opIndex++
		return

	} else {

		if spotResult.DealAmount > 0 {
			channelSpot = ProcessTradeRoutine(spot, operation.spotConfig, a.tradeDB)
			waitGroup.Add(1)
			go func() {
				select {
				case spotResult = <-channelSpot:
					Logger.Debugf("现货平仓结果:%v", spotResult)
					if spotResult.Error != TaskErrorSuccess {
						Logger.Errorf("平仓失败，请手工检查:%v", spotResult)
						a.status = StatusError
					}
					waitGroup.Done()
				}
			}()
		}

		if futureResult.DealAmount > 0 {
			channelFuture = ProcessTradeRoutine(spot, operation.futureConfig, a.tradeDB)
			waitGroup.Add(1)
			select {
			case futureResult = <-channelFuture:
				Logger.Debugf("合约平仓结果:%v", futureResult)
				if futureResult.Error != TaskErrorSuccess {
					Logger.Errorf("平仓失败，请手工检查:%v", futureResult)
					a.status = StatusError
				}
				waitGroup.Done()
			}
		}

		waitGroup.Wait()
		a.fund.ClosePosition(Exchange.ExchangeTypeFuture, future, futureConfig.Batch, pair[0])
		a.fund.ClosePosition(Exchange.ExchangeTypeSpot, spot, spotConfig.Batch, pair[0])
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

					Logger.Info("条件平仓...")

					op.futureConfig.Price = futurePrice
					op.spotConfig.Price = spotPrice

					channelFuture := ProcessTradeRoutine(a.future, op.futureConfig, a.tradeDB)
					channelSpot := ProcessTradeRoutine(a.spot, op.spotConfig, a.tradeDB)

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

					if futureResult.Error == TaskErrorSuccess && spotResult.Error == TaskErrorSuccess {
						Logger.Info("平仓完成")
						delete(a.ops, index)

						pair := Exchange.ParsePair(op.futureConfig.Pair)
						a.fund.ClosePosition(Exchange.ExchangeTypeFuture, a.future, op.futureConfig.Batch, pair[0])
						a.fund.ClosePosition(Exchange.ExchangeTypeSpot, a.spot, op.spotConfig.Batch, pair[0])

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
