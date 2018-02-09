package task

/*
	该策略用于在现货期货做差价
*/

import (
	"container/list"
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
	Cron "github.com/robfig/cron"
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
	diffDB      *Mongo.OKExDiff
	diffList    *list.List
	diffBalance float64
	forceClose  bool

	conn *Websocket.Conn
	cron *Cron.Cron

	errorCount int
}

type OperationItem struct {
	Amount       float64
	futureConfig Exchange.TradeConfig
	spotConfig   Exchange.TradeConfig
}

type AnalyzerConfig struct {
	API        string
	Secret     string
	Area       map[string]*TriggerArea `json:"area"`
	LimitOpen  float64                 `json:"limitopen"`
	LimitClose float64                 `json:"limitclose"`
	UnitAmount float64                 `json:"unitamount"`
	AutoAdjust bool                    `json:"autoadjust"`
	StepValue  float64                 `json:"stepvalue"`
}

type TriggerArea struct {
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	Amount float64 `json:"amount"`
}

// const CheckPeriodSec = 10
// const UnitAmount = 50
const TradingPeriod = 1
const CheckingPeriod = 10

var defaultConfig = AnalyzerConfig{
	// Trigger: map[string]float64{
	// 	"btc": 1.6,
	// 	"ltc": 3,
	// },
	// Close: map[string]float64{
	// 	"btc": 0.5,
	// 	"ltc": 1.5,
	// },
	Area: map[string]*TriggerArea{
		"btc": {1.6, 0.5, 10},
		"ltc": {3, 1.5, 10},
	},
	LimitClose: 0.03,  // 止损幅度
	LimitOpen:  0.005, // 允许操作价格的波动范围
	UnitAmount: 50,
	StepValue:  0.8,
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

func (a *IAnalyzer) RecordBalances() {
	balanceDB := new(Mongo.Balances)
	if err := balanceDB.Connect(); err != nil {
		Logger.Errorf("Fail to connect BalanceDB:%v", err)
		return
	}
	defer balanceDB.Close()

	var coinInfos Mongo.BalanceInfo
	balances := a.GetBalances()
	if balances != nil {
		for coin := range a.config.Area {
			var coinInfo Mongo.CoinInfo
			coinInfo.Coin = coin
			for _, v := range balances["spots"].([]map[string]interface{}) {
				if v["name"] == coin {
					coinInfo.Balance += v["balance"].(float64)
					break
				}
			}

			for _, v := range balances["futures"].([]map[string]interface{}) {
				if v["name"] == coin {
					coinInfo.Balance += v["balance"].(float64)
					coinInfos.Coins = append(coinInfos.Coins, coinInfo)
					break
				}
			}
		}

		for _, v := range balances["spots"].([]map[string]interface{}) {
			var coinInfo Mongo.CoinInfo
			coinInfo.Coin = "usdt"
			if v["name"] == "usdt" {
				coinInfo.Balance += v["balance"].(float64)
				coinInfos.Coins = append(coinInfos.Coins, coinInfo)
				break
			}
		}
	}

	if len(coinInfos.Coins) != 0 {
		balanceDB.Insert(coinInfos)
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

func (a *IAnalyzer) GetFailedPositions() []map[string]interface{} {

	var records []Mongo.FundInfo
	var err error

	if a.fund == nil {
		Logger.Error("MongoDB is not connected")
		return nil
	}

	if err, records = a.fund.GetFailedPositions(); err != nil {
		log.Printf("Error:%v", err)
	}

	var positions []map[string]interface{}
	for _, record := range records {

		if math.IsNaN(record.FutureOpen) {
			record.FutureOpen = 0
		}

		if math.IsNaN(record.FutureClose) {
			record.FutureClose = 0
		}

		if math.IsNaN(record.SpotOpen) {
			record.SpotOpen = 0
		}

		if math.IsNaN(record.SpotClose) {
			record.SpotClose = 0
		}

		position := map[string]interface{}{
			"time":         record.OpenTime,
			"batch":        record.Batch,
			"pair":         record.Pair,
			"futuretype":   record.FutureType,
			"futureopen":   record.FutureOpen,
			"futureclose":  record.FutureClose,
			"futureamount": record.FutureAmount,
			"spottype":     record.SpotType,
			"spotopen":     record.SpotOpen,
			"spotclose":    record.SpotClose,
			"spotamount":   record.SpotAmount,
		}
		positions = append(positions, position)
	}

	return positions
}

func (a *IAnalyzer) FixFailedPosition(updateJson string) error {

	if updateJson != "" {
		var config map[string]interface{}
		err := json.Unmarshal([]byte(updateJson), &config)
		if err != nil {
			log.Printf("Fail to get config:%v", err)
			return errors.New(TaskErrorMsg[TaskInvalidConfig])
		}

		return a.fund.FixFailedPosition(config)

	}

	return errors.New(TaskErrorMsg[TaskInvalidInput])

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
		if a.checkPeriodSec == CheckingPeriod {
			Logger.Debugf("检测周期变成1秒")
			a.checkPeriodSec = TradingPeriod
		}
		a.checkNoTradeCounter = 0
	} else {
		if a.checkPeriodSec == TradingPeriod {
			// 5 minutes without trading
			if a.checkNoTradeCounter >= 300 {
				Logger.Debugf("检测周期变成10秒")
				a.checkPeriodSec = CheckingPeriod
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
	a.diffList = list.New()
	a.checkPeriodSec = CheckingPeriod
	a.forceClose = false
	a.errorCount = 0
	a.tradeDB = &Mongo.Trades{
		Config: &Mongo.DBConfig{
			CollectionName: "DiffOKExFutureSpotTrade",
		},
	}

	a.cron = Cron.New()
	a.cron.AddFunc("@daily", a.RecordBalances)
	a.cron.Start()

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
	a.loadDiffHistory()

	go func() {
		for {
			select {
			case event := <-futureExchange.WatchEvent():
				if event == Exchange.EventConnected {
					// 提早发送命令
					for k := range a.config.Area {
						pair := (k + "/usdt")
						futureExchange.GetDepthValue(pair)
					}
					a.future = Exchange.IExchange(futureExchange)

				} else if event == Exchange.EventLostConnection {
					if a.status != StatusNone && a.status != StatusError {
						go a.reconnect(futureExchange)
					}
				}
			case event := <-spotExchange.WatchEvent():
				if event == Exchange.EventConnected {
					for k := range a.config.Area {
						pair := (k + "/usdt")
						spotExchange.GetDepthValue(pair)
					}

					a.spot = Exchange.IExchange(spotExchange)

				} else if event == Exchange.EventLostConnection {
					if a.status != StatusNone && a.status != StatusError {
						go a.reconnect(spotExchange)
					}
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

func (a *IAnalyzer) loadDiffHistory() {
	now := time.Now()
	start := now.Add(-12 * time.Hour)
	log.Printf("开始时间:%v 结束时间:%v", start, now)
	records, err := a.diffDB.FindAll("eth", start, now)

	if err != nil {
		log.Printf("Error:%v", err)
		return
	}

	var totalDiff float64
	for _, record := range records {
		totalDiff += record.Diff
		a.diffList.PushBack(record.Diff)
	}

	a.diffBalance = totalDiff / float64(len(records))
	log.Printf("波动中值:%v", a.diffBalance)
}

func (a *IAnalyzer) updateDiffArea(coin string, currentDiff float64) {
	if a.diffList == nil || a.diffList.Len() < 4096 {
		a.diffList.PushBack(currentDiff)
		length := a.diffList.Len()
		var total float64
		for e := a.diffList.Front(); e != nil; e = e.Next() {
			total += e.Value.(float64)
		}

		a.diffBalance = total / float64(length)

	} else {
		front := a.diffList.Front()
		total := a.diffBalance * float64(a.diffList.Len())
		total = (total - front.Value.(float64) + currentDiff)

		a.diffList.Remove(front)
		a.diffList.PushBack(currentDiff)
		a.diffBalance = total / float64(a.diffList.Len())
	}

	if a.config.AutoAdjust {
		min := a.diffBalance
		if min < 1.0 {
			min = 1.0
		}

		a.config.Area[coin].Open = (min + a.config.StepValue)

		// 平仓差价一般不修改
		// close := min - DiffStep
		// if close > 0.5 {
		// 	close = 0.5
		// }
		// a.config.Area[coin].Close = close
	}

	Logger.Infof("波动中值:%.2f 开仓价差:%v 平仓价差:%v 自动调整价差:%v", a.diffBalance, a.config.Area[coin].Open, a.config.Area[coin].Close, a.config.AutoAdjust)
}

func (a *IAnalyzer) Watch() {

	// var orderAmount = 50
	// var orderFutureQuantity = orderAmount / 10

	for coin := range a.config.Area {
		pair := coin + "/usdt"

		var diff1, diff2 float64
		err1, askFuture, askFuturePlacePrice, bidFuture, bidFuturePlacePrice := CalcDepthPrice(true, a.future, pair, a.config.UnitAmount)
		err2, askSpot, askSpotPlacePrice, bidSpot, bidSpotPlacePrice := CalcDepthPrice(false, a.spot, pair, a.config.UnitAmount)

		if err1 == nil && err2 == nil {
			diff1 = (bidSpot - askFuture) * 100 / bidSpot
			msg1 := fmt.Sprintf("币种:%s, 合约可买入价格：%.2f, 现货可卖出价格：%.2f, 价差：%.2f%%", pair, askFuture, bidSpot, diff1)

			Logger.Info(msg1)
			// a.wsPublish("okexdiff", msg)
			diff2 = (bidFuture - askSpot) * 100 / askSpot
			msg2 := fmt.Sprintf("币种:%s, 合约可卖出价格：%.2f, 现货可买入价格：%.2f, 价差：%.2f%%", pair, bidFuture, askSpot, diff2)

			Logger.Info(msg2)
			if diff1 > 0 {
				a.updateDiffArea(coin, diff1)
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
				a.updateDiffArea(coin, diff2)
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
		var diff float64
		if diff1 > 0 {
			diff = diff1
		} else {
			diff = diff2
		}

		placeAmount := a.checkFunds(pair, diff)
		if placeAmount == 0 {
			Logger.Info("无可用仓位...不开仓")
			a.adjustDuration(false)
			continue
		}

		if !a.checkOpenTime() {
			Logger.Info("周五10点至17点不开仓")
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
				Price:  askFuturePlacePrice,
				Amount: placeAmount / constContractRatio[coin],
				Limit:  a.config.LimitOpen,
			},
				a.spot, Exchange.TradeConfig{
					Batch:  batch,
					Pair:   pair,
					Type:   Exchange.TradeTypeBuy,
					Price:  bidSpotPlacePrice,
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
				Price:  bidFuturePlacePrice,
				Amount: placeAmount / constContractRatio[coin],
				Limit:  a.config.LimitOpen,
			},
				a.spot, Exchange.TradeConfig{
					Batch:  batch,
					Pair:   pair,
					Type:   Exchange.TradeTypeSell,
					Price:  askSpotPlacePrice,
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

	Logger.Info("关闭任务")

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
	spotConfig.Amount = spotResult.DealAmount

	Logger.Debugf("spotConfig.Amount:%v", spotConfig.Amount)

	operation.futureConfig = futureConfig
	operation.spotConfig = spotConfig

	if futureResult.Error == TaskErrorSuccess && spotResult.Error == TaskErrorSuccess {

		Logger.Debug("锁仓成功")
		a.ops[a.opIndex] = &operation
		a.ops[a.opIndex].Amount = spotResult.AvgPrice * spotResult.DealAmount
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
						// a.status = StatusError
						a.countError()
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
					// a.status = StatusError
					a.countError()
				}
				waitGroup.Done()
			}
		}

		waitGroup.Wait()
		if spotResult.Error == TaskErrorSuccess && futureResult.Error == TaskErrorSuccess {
			a.fund.ClosePosition(spotConfig.Batch, spotResult.AvgPrice, futureResult.AvgPrice, Mongo.FundStatusClose)
		} else {
			a.fund.ClosePosition(spotConfig.Batch, spotResult.AvgPrice, futureResult.AvgPrice, Mongo.FundStatusError)
		}

	}

	return
}

func (a *IAnalyzer) checkOpenTime() bool {
	now := time.Now()
	if now.Weekday() == time.Friday && (now.Hour() > 10 && now.Hour() < 17) {
		return false
	}

	return true
}

func (a *IAnalyzer) checkCloseTime() bool {
	now := time.Now()
	if now.Weekday() == time.Friday && now.Hour() > 16 {
		return false
	}

	return true
}

func (a *IAnalyzer) checkFunds(pair string, diff float64) float64 {
	var usedAmount, ratio float64

	for _, op := range a.ops {
		if op != nil && op.futureConfig.Pair == pair {
			usedAmount += op.Amount
		}
	}
	coin := Exchange.ParsePair(pair)[0]

	// base := a.config.Area[coin].Open
	// if diff > base && diff < base+step {
	// 	ratio = 0.3
	// } else if diff >= (base+step) && diff < (base+2*step) {
	// 	ratio = 0.6
	// } else if diff >= (base + 2*step) {
	// 	ratio = 1
	// }

	base := a.config.Area[coin].Open
	if diff > base && diff < (base+a.config.StepValue) {
		ratio = 0.6
	} else if diff >= (base + a.config.StepValue) {
		ratio = 1
	}
	// } else if diff >= (base+a.config.StepValue) && diff < (base+2*a.config.StepValue) {
	// 	ratio = 0.8
	// } else if diff >= (base + 2*a.config.StepValue) {
	// 	ratio = 1
	// }

	Logger.Infof("开仓单位：%v 已开仓：%v 开仓总量:%v 当前价差:%.2f 可开仓比例:%v", a.config.UnitAmount, usedAmount, a.config.Area[coin].Amount, diff, ratio)

	if (usedAmount + a.config.UnitAmount) > (a.config.Area[coin].Amount * ratio) {
		return 0
	}

	return a.config.UnitAmount

}

func (a *IAnalyzer) checkPosition(coin string, askFuturePrice float64, bidFuturePrice float64, askSpotPrice float64, bidSpotPrice float64) bool {
	if len(a.ops) != 0 {
		for index, op := range a.ops {

			if op == nil {
				Logger.Debug("Invalid operation")
				continue
			}

			closeConditions := []bool{
				a.forceClose,
				// OutFuturePriceArea(op.futureConfig, askFuturePrice, bidFuturePrice, a.config.LimitClose), // 防止爆仓
				CheckPriceDiff(op.spotConfig, op.futureConfig, askFuturePrice, bidFuturePrice, askSpotPrice, bidSpotPrice, a.config.Area[coin].Close),
			}

			// Logger.Debugf("Conditions:%v", closeConditions)

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
						if len(a.ops) == 0 {
							a.forceClose = false
						}

					} else {
						Logger.Error("平仓失败，请手工检查")
						a.fund.ClosePosition(op.spotConfig.Batch, spotResult.AvgPrice, futureResult.AvgPrice, Mongo.FundStatusError)
						//a.status = StatusError
						a.countError()
					}

					return true
				}

			}
		}

		// return false
	}

	return false
}

func (a *IAnalyzer) countError() {
	Logger.Errorf("Error Counter:%v", a.errorCount)
	if a.errorCount < 10 {
		a.errorCount++
	} else {
		a.errorCount = 0
		a.status = StatusError
	}
}
