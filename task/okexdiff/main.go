package main

/*
	该策略用于在现货期货做差价
*/

import (
	"flag"
	"fmt"
	"math"
	"sync"
	"time"
	// "time"

	Exchange "madaoQT/exchange"
	Mongo "madaoQT/mongo"
	Task "madaoQT/task"
	Utils "madaoQT/utils"

	Websocket "github.com/gorilla/websocket"
	"github.com/kataras/golog"
)

const Explanation = "To make profit from the difference between the future`s price and the current`s"

/*
	初始化日志句柄
*/
var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.SetTimeFormat("2006-01-02 06:04:05")
}

type StatusType int

const (
	StatusNone StatusType = iota
	StatusProcessing
	StatusOrdering
	StatusQuit
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
	APIKey     string
	SecretKey  string
	Area       map[string]TriggerArea
	LimitArea  float64
	LimitClose float64
}

type TriggerArea struct {
	Start float64
	Close float64
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
	LimitArea:  0.005, // 允许操作价格的波动范围
}

func (a *IAnalyzer) GetTaskExplanation() *Task.TaskExplanation {
	return &Task.TaskExplanation{"OkexDiff", "OkexDiff"}
}

func (a *IAnalyzer) GetDefaultConfig() *AnalyzerConfig {
	return &defaultConfig
}

func (a *IAnalyzer) connectTicker() {
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

func (a *IAnalyzer) websocketPulish(topic string, message string) {
	if a.conn != nil {
		msg, err := Task.WebsocketMessageSerialize(topic, message)
		if err != nil {
			Logger.Errorf("Fail to serialize: %v", err)
		}

		Logger.Infof("Send:%v", msg)
		a.conn.WriteMessage(Websocket.TextMessage, []byte(msg))
	}
}

func (a *IAnalyzer) Init(config *AnalyzerConfig) {

	if a.config == nil {
		a.config = a.GetDefaultConfig()
	}

	// 监视币种以及余额
	a.coins = map[string]float64{
		// "btc": 0,
		"ltc/usdt": 1,
	}

	a.ops = make(map[uint]*OperationItem)
	a.status = StatusProcessing

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

	// a.connectTicker()

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

		a.websocketPulish("test", msg)

		if a.checkPosition(coinName, valuefuture.Last, valueCurrent.Last) {
			Logger.Info("持仓中...不做交易")
			continue
		}

		if valuefuture != nil && valueCurrent != nil {

			if math.Abs(difference) > a.config.Area[coinName].Start {
				if valuefuture.Last > valueCurrent.Last {
					Logger.Info("卖出合约，买入现货")

					batch := Utils.GetRandomHexString(12)

					a.placeOrdersByQuantity(a.future, Exchange.TradeConfig{
						Batch:  batch,
						Coin:   coinName,
						Type:   Exchange.TradeTypeOpenShort,
						Price:  valuefuture.Last,
						Amount: 5,
						Limit:  a.config.LimitArea,
					},
						a.spot, Exchange.TradeConfig{
							Batch:  batch,
							Coin:   coinName,
							Type:   Exchange.TradeTypeBuy,
							Price:  valueCurrent.Last,
							Amount: 50 / valueCurrent.Last,
							Limit:  a.config.LimitArea,
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
						Limit:  a.config.LimitArea,
					},
						a.spot, Exchange.TradeConfig{
							Batch:  batch,
							Coin:   coinName,
							Type:   Exchange.TradeTypeSell,
							Price:  valueCurrent.Last,
							Amount: 50 / valueCurrent.Last,
							Limit:  a.config.LimitArea,
						})
				}
			}
		}
	}

	return true

}

func (a *IAnalyzer) Close() {
	a.status = StatusQuit
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

	channelFuture := Task.ProcessTradeRoutine(future, futureConfig, a.tradeDB, a.orderDB)
	channelSpot := Task.ProcessTradeRoutine(spot, spotConfig, a.tradeDB, a.orderDB)

	var waitGroup sync.WaitGroup
	var futureResult, spotResult Task.TradeResult

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

	if futureResult.Error == Task.TradeErrorSuccess && spotResult.Error == Task.TradeErrorSuccess {

		Logger.Debug("锁仓成功")
		a.ops[a.opIndex] = &operation
		a.opIndex++
		return

	} else if futureResult.Error == Task.TradeErrorSuccess && spotResult.Error != Task.TradeErrorSuccess {

		channelSpot = Task.ProcessTradeRoutine(spot, operation.spotConfig, a.tradeDB, a.orderDB)
		select {

		case spotResult = <-channelSpot:
			Logger.Debugf("现货平仓结果:%v", spotResult)
			if spotResult.Error != Task.TradeErrorSuccess {
				Logger.Errorf("平仓失败，请手工检查:%v", spotResult)
				a.status = StatusError
			}
		}

	} else if futureResult.Error != Task.TradeErrorSuccess && spotResult.Error == Task.TradeErrorSuccess {

		channelFuture = Task.ProcessTradeRoutine(spot, operation.futureConfig, a.tradeDB, a.orderDB)
		select {
		case futureResult = <-channelFuture:
			Logger.Debugf("合约平仓结果:%v", futureResult)
			if futureResult.Error != Task.TradeErrorSuccess {
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
				!Task.InPriceArea(futurePrice, op.futureConfig.Price, a.config.LimitClose), // 防止爆仓
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

					channelFuture := Task.ProcessTradeRoutine(a.future, op.futureConfig, a.tradeDB, a.orderDB)
					channelSpot := Task.ProcessTradeRoutine(a.spot, op.spotConfig, a.tradeDB, a.orderDB)

					var waitGroup sync.WaitGroup
					var futureResult, spotResult Task.TradeResult

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

					if futureResult.Error == Task.TradeErrorSuccess && spotResult.Error == Task.TradeErrorSuccess {
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

/*
	Parameters:
	config AnalyzerConfig
*/
func main() {

	// flag.Parse()

	// kill := make(chan os.Signal, 1)
	// signal.Notify(kill, os.Interrupt, os.Kill)

	// go func() {
	// 	for {
	// 		select {
	// 		case <-kill:
	// 			Logger.Infof("interrupt")
	// 			Utils.SleepAsyncBySecond(3)
	// 			os.Exit(0)
	// 		}
	// 	}
	// }()

	config := flag.String("config", "test", "configuration")
	flag.Parse()
	Logger.Infof("start with paramters:%v, and I will sleep for a while", *config)

	Utils.SleepAsyncBySecond(5)

	Logger.Info("I am awake")

	for {
		select {
		case <-time.After(1 * time.Second):
			Logger.Info("1....")
		}
	}
}
