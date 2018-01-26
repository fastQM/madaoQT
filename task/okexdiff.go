package task

/*
	该策略用于在现货期货做差价
*/

import (
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

type IAnalyzer struct {
	config AnalyzerConfig
	// exchanges []ExchangeHandler
	// coins map[string]float64

	// futures  map[string]AnalyzeItem
	// currents map[string]AnalyzeItem
	future Exchange.IExchange
	spot   Exchange.IExchange
	fund   *OkexFundManage

	status              StatusType
	checkPeriodSec      time.Duration
	checkNoTradeCounter int
	ops                 map[uint]*OperationItem
	opIndex             uint

	tradeDB *Mongo.Trades
	// orderDB *Mongo.Orders
	diffDB *Mongo.OKExDiff

	conn *Websocket.Conn
}

type OperationItem struct {
	Amount       float64
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
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	Amount float64 `json:"amount"`
}

// const CheckPeriodSec = 10
const UnitAmount = 50

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

var constContractRatio = map[string]float64{
	"btc": 100,
	"ltc": 10,
	"eth": 10,
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
		if balances := a.spot.GetBalance(); balances != nil {
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
		if balances := a.future.GetBalance(); balances != nil {
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

func (a *IAnalyzer) GetPositions() []map[string]interface{} {

	var positions []map[string]interface{}
	if a.ops != nil {
		for _, op := range a.ops {
			if op != nil {
				position := map[string]interface{}{
					"amount":     op.Amount,
					"batch":      op.futureConfig.Batch,
					"pair":       op.futureConfig.Pair,
					"futuretype": Exchange.TradeTypeString[op.futureConfig.Type],
					"spottype":   Exchange.TradeTypeString[op.spotConfig.Type],
				}
				positions = append(positions, position)
			}
		}

		return positions
	}
	return nil
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

// GetStatus get the status of the task
func (a *IAnalyzer) GetStatus() int {
	return int(a.status)
}

func (a *IAnalyzer) adjustDuration(hasTrade bool) {
	if hasTrade {
		if a.checkPeriodSec == 10 {
			Logger.Debugf("检测周期变成3秒")
			a.checkPeriodSec = 3
		}
		a.checkNoTradeCounter = 0
	} else {
		if a.checkPeriodSec == 3 {

			if a.checkNoTradeCounter >= 10 {
				Logger.Debugf("检测周期变成10秒")
				a.checkPeriodSec = 10
				a.checkNoTradeCounter = 0
			} else {
				a.checkNoTradeCounter++
			}
		}
	}
}

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
	a.checkPeriodSec = 10
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

	Logger.Info("启动OKEx合约监视程序")

	futureExchange := new(Exchange.OKExAPI)
	futureExchange.SetConfigure(Exchange.Config{
		API:    api,
		Secret: secret,
		Custom: map[string]interface{}{
			"exchangeType": Exchange.ExchangeTypeFuture,
			"period":       "this_week",
		},
	})

	if err := futureExchange.Start(); err != nil {
		Logger.Errorf("Fail to start:%v", err)
		return err
	}

	Logger.Info("启动OKEx现货监视程序")
	spotExchange := Exchange.NewOKExSpotApi(&Exchange.Config{
		API:    api,
		Secret: secret,
	})

	if err := spotExchange.Start(); err != nil {
		Logger.Errorf("Fail to start:%v", err)
		return err
	}

	a.fund = new(OkexFundManage)
	a.fund.Init()

	a.wsConnect()
	a.status = StatusProcessing

	// load current positions
	a.loadPosition()

	go func() {
		for {
			select {
			case event := <-futureExchange.WatchEvent():
				if event == Exchange.EventConnected {
					// for k := range a.config.Area {
					// 	k = (k + "/usdt")
					// 	futureExchange.StartTicker(k)
					// }
					a.future = Exchange.IExchange(futureExchange)

				} else if event == Exchange.EventLostConnection {
					go a.reconnect(futureExchange)
				}
			case event := <-spotExchange.WatchEvent():
				if event == Exchange.EventConnected {
					// for k := range a.config.Area {
					// 	k = (k + "/usdt")
					// 	spotExchange.StartTicker(k)
					// }

					a.spot = Exchange.IExchange(spotExchange)

				} else if event == Exchange.EventLostConnection {
					go a.reconnect(spotExchange)
				}
			case <-time.After(a.checkPeriodSec * time.Second):
				if a.status == StatusError || a.status == StatusNone {
					Logger.Debug("状态异常或退出")
					return
				}

				if a.status == StatusOrdering {
					Logger.Debug("交易中...")
					continue
				}

				a.Watch()
			}
		}
	}()

	return nil
}

func (a *IAnalyzer) reconnect(exchange Exchange.IExchange) {
	Logger.Debug("Reconnecting......")
	// utils.SleepAsyncBySecond(60)
	if err := exchange.Start(); err != nil {
		Logger.Errorf("Fail to start exchange %v with error:%v", exchange.GetExchangeName(), err)
		return
	}
}

func (a *IAnalyzer) loadPosition() {
	var records []Mongo.FundInfo
	var err error

	if err, records = a.fund.CheckPosition(); err != nil {
		Logger.Errorf("Fail to load positions:%v", err)
		return
	}

	if records != nil && len(records) > 0 {
		for _, record := range records {

			var futureConfig, spotConfig Exchange.TradeConfig
			operation := OperationItem{}

			spotConfig.Batch = record.Batch
			spotConfig.Amount = record.SpotAmount
			spotConfig.Pair = record.Pair
			spotConfig.Price = record.SpotOpen
			spotConfig.Type = Exchange.RevertTradeType(Exchange.TradeTypeInt(record.SpotType))
			spotConfig.Limit = a.config.LimitOpen

			futureConfig.Batch = record.Batch
			futureConfig.Amount = record.FutureAmount
			futureConfig.Pair = record.Pair
			futureConfig.Price = record.FutureOpen
			futureConfig.Type = Exchange.RevertTradeType(Exchange.TradeTypeInt(record.FutureType))
			futureConfig.Limit = a.config.LimitOpen

			operation.futureConfig = futureConfig
			operation.spotConfig = spotConfig
			a.ops[a.opIndex] = &operation
			a.ops[a.opIndex].Amount = spotConfig.Amount * spotConfig.Price
			a.opIndex++
		}
	}
}

func (a *IAnalyzer) Watch() {

	// var orderAmount = 50
	// var orderFutureQuantity = orderAmount / 10

	for coin := range a.config.Area {
		pair := coin + "/usdt"

		var diff1, diff2 float64
		err1, askFuture, bidFuture := CalcDepthPrice(true, a.future, pair, UnitAmount)
		err2, askSpot, bidSpot := CalcDepthPrice(false, a.spot, pair, UnitAmount)

		if err1 == nil && err2 == nil {
			diff1 = (bidSpot - askFuture) * 100 / bidSpot
			msg1 := fmt.Sprintf("币种:%s, 合约可买入价格：%.2f, 现货可卖出价格：%.2f, 价差：%.2f%%", pair, askFuture, bidSpot, diff1)

			Logger.Info(msg1)
			// a.wsPublish("okexdiff", msg)
			diff2 = (bidFuture - askSpot) * 100 / askSpot
			msg2 := fmt.Sprintf("币种:%s, 合约可卖出价格：%.2f, 现货可买入价格：%.2f, 价差：%.2f%%", pair, bidFuture, askSpot, diff2)

			Logger.Info(msg2)
			if diff1 > 0 {
				a.diffDB.Insert(Mongo.DiffValue{
					Coin:      coin,
					SpotPrice: bidSpot,
					// SpotVolume:   valueCurrent.Volume,
					FuturePrice: askFuture,
					// FutureVolume: valueFuture.Volume,
					Diff: diff1,
					Time: time.Now(),
				})
			} else {
				a.diffDB.Insert(Mongo.DiffValue{
					Coin:      coin,
					SpotPrice: askSpot,
					// SpotVolume:   valueCurrent.Volume,
					FuturePrice: bidFuture,
					// FutureVolume: valueFuture.Volume,
					Diff: diff2,
					Time: time.Now(),
				})
			}

		} else {
			Logger.Errorf("无效深度不操作")
			return
		}

		// 检测是否需要平仓
		if a.checkPosition(coin, askFuture, bidFuture, askSpot, bidSpot) {
			a.adjustDuration(true)
			Logger.Info("已执行平仓操作...不做交易")
			continue
		}

		// 检测是否有足够的开仓资金
		placeAmount := a.checkFunds(pair)
		if placeAmount == 0 {
			Logger.Info("无可用仓位...不开仓")
			a.adjustDuration(false)
			continue
		}

		// 检测是否需要开仓
		if diff2 > a.config.Area[coin].Open {

			Logger.Info("卖出合约，买入现货")
			a.adjustDuration(true)
			batch := Utils.GetRandomHexString(12)

			a.placeOrdersByQuantity(a.future, Exchange.TradeConfig{
				Batch:  batch,
				Pair:   pair,
				Type:   Exchange.TradeTypeOpenShort,
				Price:  askFuture,
				Amount: placeAmount / constContractRatio[coin],
				Limit:  a.config.LimitOpen,
			},
				a.spot, Exchange.TradeConfig{
					Batch:  batch,
					Pair:   pair,
					Type:   Exchange.TradeTypeBuy,
					Price:  bidSpot,
					Amount: placeAmount / bidSpot,
					Limit:  a.config.LimitOpen,
				})

		} else if diff1 > a.config.Area[coin].Open {
			Logger.Info("买入合约, 卖出现货")
			a.adjustDuration(true)
			batch := Utils.GetRandomHexString(12)

			a.placeOrdersByQuantity(a.future, Exchange.TradeConfig{
				Batch:  batch,
				Pair:   pair,
				Type:   Exchange.TradeTypeOpenLong,
				Price:  bidFuture,
				Amount: placeAmount / constContractRatio[coin],
				Limit:  a.config.LimitOpen,
			},
				a.spot, Exchange.TradeConfig{
					Batch:  batch,
					Pair:   pair,
					Type:   Exchange.TradeTypeSell,
					Price:  askSpot,
					Amount: placeAmount / askSpot,
					Limit:  a.config.LimitOpen,
				})
		} else {
			a.adjustDuration(false)
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

	// record the result
	if err := a.fund.OpenPosition(spotConfig.Batch, spotConfig.Pair,
		spotConfig.Type,
		futureConfig.Type,
		spotResult.AvgPrice,
		spotResult.DealAmount,
		futureResult.AvgPrice,
		futureResult.DealAmount); err != nil {
		Logger.Error("Fail to save fund info")
	}

	operation := OperationItem{}

	futureConfig.Type = Exchange.RevertTradeType(futureConfig.Type)
	spotConfig.Type = Exchange.RevertTradeType(spotConfig.Type)

	futureConfig.Amount = futureResult.DealAmount
	spotConfig.Amount = math.Trunc(spotResult.DealAmount*100) / 100 // 后台的限制, 如果数值是0.299,可交易数量是0.29

	Logger.Debugf("spotConfig.Amount:%v", spotConfig.Amount)

	operation.futureConfig = futureConfig
	operation.spotConfig = spotConfig

	if futureResult.Error == TaskErrorSuccess && spotResult.Error == TaskErrorSuccess {

		Logger.Debug("锁仓成功")
		a.ops[a.opIndex] = &operation
		a.ops[a.opIndex].Amount = spotConfig.Amount * spotConfig.Price
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
			channelFuture = ProcessTradeRoutine(future, operation.futureConfig, a.tradeDB)
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
		if spotResult.Error == TaskErrorSuccess && futureResult.Error == TaskErrorSuccess {
			a.fund.ClosePosition(spotConfig.Batch, 0, 0, Mongo.FundStatusClose)
		} else {
			a.fund.ClosePosition(spotConfig.Batch, 0, 0, Mongo.FundStatusError)
		}

	}

	return
}

func (a *IAnalyzer) checkFunds(pair string, diff float64) float64 {
	var usedAmount, ratio float64
	step := 0.5
	for _, op := range a.ops {
		if op != nil && op.futureConfig.Pair == pair {
			usedAmount += op.Amount
		}
	}
	coin := Exchange.ParsePair(pair)[0]

	base := a.config.Area[coin].Open
	if diff > base && diff < base+step {
		ratio = 0.3
	} else if diff >= (base+step) && diff < (base+2*step) {
		ratio = 0.6
	} else if diff >= (base + 2*step) {
		ratio = 1
	}

	Logger.Debugf("已开仓：%v 开仓总量:%v 当前价差(%v)可开仓比例:%v", usedAmount, a.config.Area[coin].Amount, diff, ratio)

	if (usedAmount + UnitAmount) > a.config.Area[coin].Amount {
		return 0
	}

	return UnitAmount

}

func (a *IAnalyzer) checkPosition(coin string, askFuturePrice float64, bidFuturePrice float64, askSpotPrice float64, bidSpotPrice float64) bool {
	if len(a.ops) != 0 {
		for index, op := range a.ops {

			if op == nil {
				Logger.Debug("Invalid operation")
				continue
			}

			closeConditions := []bool{
				OutFuturePriceArea(op.futureConfig, askFuturePrice, bidFuturePrice, a.config.LimitClose), // 防止爆仓
				// !InPriceArea(spotPrice, op.spotConfig.Price, a.config.LimitClose),
				// math.Abs((futurePrice-spotPrice)*100/spotPrice) < a.config.Area[coin].Close,
				CheckPriceDiff(op.spotConfig, op.futureConfig, askFuturePrice, bidFuturePrice, askSpotPrice, bidSpotPrice, a.config.Area[coin].Close),
			}

			Logger.Debugf("Conditions:%v", closeConditions)

			for _, condition := range closeConditions {

				if condition {

					Logger.Infof("条件平仓...期货配置:%v 现货配置:%v", op.futureConfig, op.spotConfig)

					if op.futureConfig.Type == Exchange.TradeTypeCloseLong {
						op.futureConfig.Price = askFuturePrice
					} else if op.futureConfig.Type == Exchange.TradeTypeCloseShort {
						op.futureConfig.Price = bidFuturePrice
					} else {
						Logger.Errorf("Invalid trade type")
					}

					if op.spotConfig.Type == Exchange.TradeTypeBuy {
						op.spotConfig.Price = askSpotPrice
					} else if op.spotConfig.Type == Exchange.TradeTypeSell {
						op.spotConfig.Price = bidSpotPrice
					} else {
						Logger.Errorf("Invalid trade type")
					}

					// op.futureConfig.Price = futurePrice
					// op.spotConfig.Price = spotPrice

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
						a.fund.ClosePosition(op.spotConfig.Batch, spotResult.AvgPrice, futureResult.AvgPrice, Mongo.FundStatusClose)

					} else {
						Logger.Error("平仓失败，请手工检查")
						a.fund.ClosePosition(op.spotConfig.Batch, 0, 0, Mongo.FundStatusError)
						a.status = StatusError
					}

					return true
				}

			}
		}

		// return false
	}

	return false
}
