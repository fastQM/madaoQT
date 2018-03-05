package main

import (
	"log"
	"sync"
	"time"

	Global "madaoQT/config"
	Exchange "madaoQT/exchange"
	Mongo "madaoQT/mongo"
	MongoTrend "madaoQT/mongo/trend"
	Task "madaoQT/task"
	Utils "madaoQT/utils"

	"github.com/kataras/golog"
)

const trendTaskExplaination = "该策略适用于可能在短期内(1-3天)出现大幅波动(10%-30%)的市场"

// TrendTask 策略适用于在短期内(1-3天)出现大幅波动(10%-30%)的市场
type TrendTask struct {
	config TrendConfig

	binance Exchange.IExchange
	future  Exchange.IExchange

	status      Task.StatusType
	database    *MongoTrend.TrendMongo
	fundManager *FundManager

	positions     map[uint]*TrendPosition
	positionIndex uint

	errorCounter int
}

type TrendConfig struct {
	UnitAmount      float64
	LimitCloseRatio float64
	LimitOpenRatio  float64
}

type TrendPosition struct {
	TimeStamp int64
	Amount    float64
	config    Exchange.TradeConfig
}

const pair = "eth/usdt"
const globalPeriod = "2h"

var constContractRatio = map[string]float64{
	"btc": 100,
	"ltc": 10,
	"eth": 10,
}

var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.SetTimeFormat(Global.TimeFormat)
	Logger.SetPrefix("[TREN]")
}

func (p *TrendTask) Start(api string, secret string, configJSON string) error {

	Logger.Infof("%s", trendTaskExplaination)

	p.config = TrendConfig{
		UnitAmount:      250,
		LimitCloseRatio: 0.03,
		LimitOpenRatio:  0.003,
	}

	p.database = new(MongoTrend.TrendMongo)
	if err := p.database.Connect(); err != nil {
		Logger.Errorf("Error:%v", err)
		return err
	}

	p.fundManager = new(FundManager)
	p.fundManager.Init(p.database.FundCollection)

	p.binance = new(Exchange.Binance)

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

	go func() {
		for {
			select {
			case event := <-futureExchange.WatchEvent():
				if event == Exchange.EventConnected {
					p.future = Exchange.IExchange(futureExchange)

				} else if event == Exchange.EventLostConnection {
					if p.status != Task.StatusNone && p.status != Task.StatusError {
						go Task.Reconnect(futureExchange)
					}
				}
			case <-time.After(10 * time.Second):
				if p.status == Task.StatusError || p.status == Task.StatusNone {
					Logger.Debug("状态异常或退出")
					return
				}

				if p.status == Task.StatusOrdering {
					Logger.Debug("交易中...")
					continue
				}

				p.Watch()
			}
		}
	}()

	p.status = Task.StatusProcessing
	p.positions = make(map[uint]*TrendPosition)
	p.loadPosition()
	return nil
}

func (p *TrendTask) loadPosition() {
	var records []MongoTrend.FundInfo
	var err error

	if err, records = p.fundManager.CheckPosition(); err != nil {
		Logger.Errorf("Fail to load positions:%v", err)
		return
	}

	if records != nil && len(records) > 0 {
		for _, record := range records {
			var position TrendPosition
			var config Exchange.TradeConfig
			config.Batch = record.Batch
			config.Amount = record.FutureAmount
			config.Limit = p.config.LimitOpenRatio
			config.Pair = record.Pair
			config.Type = Exchange.TradeTypeInt(record.FutureType)
			config.Price = record.FutureOpen
			coin := Exchange.ParsePair(config.Pair)[0]
			position.TimeStamp = record.OpenTime.Unix()
			position.Amount = config.Amount * constContractRatio[coin]
			position.config = config
			p.positions[p.positionIndex] = &position
			p.positionIndex++
			Logger.Infof("Position:%v", position)
		}
	}
}

func (p *TrendTask) checkFunds() float64 {
	funds := float64(5000)
	var usedAmount float64
	for _, position := range p.positions {
		usedAmount += position.Amount
	}

	Logger.Infof("开仓单位：%v 已开仓：%v 开仓总量:%v", p.config.UnitAmount, usedAmount, funds)
	if (usedAmount + p.config.UnitAmount) > funds {
		return 0
	}

	return p.config.UnitAmount

}

func (p *TrendTask) Watch() {

	result := p.binance.GetKline(pair, globalPeriod, 50)
	var lastDiff float64

	if result == nil || len(result) < 20 {
		Logger.Errorf("无效K线数据")
		return
	}

	length := len(result)
	current := result[length-1]
	Logger.Infof("最新[High]%.2f [Open]%.2f [Close]%.2f [Low]%.2f [Volumn]%.2f", current.High, current.Open, current.Close, current.Low, current.Volumn)

	// 是否需要减仓
	if p.CheckClosePosition(result) {
		return
	}

	// 资金管理
	amount := p.checkFunds()
	if amount == 0 {
		Logger.Info("无可用仓位...不开仓")
		return
	}

	// 之前的数据判断均线
	array10 := result[length-11 : length-1]
	array20 := result[length-21 : length-1]
	// array10 := result[length-12 : length-2]
	// array20 := result[length-22 : length-2]

	// avg5 := GetAverage(5, array5)
	avg10 := Exchange.GetAverage(10, array10)
	avg20 := Exchange.GetAverage(20, array20)

	lastDiff = avg10 - avg20

	array5 := result[length-5 : length]
	array10 = result[length-10 : length]
	array20 = result[length-20 : length]

	avg5 := Exchange.GetAverage(5, array5)
	avg10 = Exchange.GetAverage(10, array10)
	avg20 = Exchange.GetAverage(20, array20)

	// 1. 三条均线要保持平行，一旦顺序乱则清仓
	// 2. 开仓后，价格柱破10日均线清仓;虽然可能只是下探均线，但是说明市场强势减弱，后续可以更轻松的建仓
	// 3. 开多时，开仓价格应该高于十日均线；开空时，开仓价格需要低于十日均线
	timestamp := int64(current.OpenTime)
	time := time.Unix(int64(timestamp), 0).Format(Global.TimeFormat)
	Logger.Infof("[Time]%s [Last]%.2f [Avg5]%.2f [Avg10]%.2f [Avg20]%.2f [Diff]%.2f", time, lastDiff, avg5, avg10, avg20, avg10-avg20)
	// Logger.Infof("Current Middle:%v", (current.High+current.Low)/2)
	// 10日均线从高于20日均线变成低于20日均线
	if lastDiff > 0 && avg10-avg20 < 0 {
		// 需要保证5日均线低于10日均线，并且当日均线中间值低于五日均线
		if avg5 < avg10 && (current.High+current.Low)/2 < avg5 {
			err2, _, _, _, bidSpotPlacePrice := Task.CalcDepthPrice(true, constContractRatio, p.future, pair, p.config.UnitAmount)
			if err2 != nil {
				Logger.Infof("深度无效")
				return
			}

			// 价格低于均线
			if bidSpotPlacePrice < avg5 {
				Logger.Infof("执行做空，做空价格:%.2f", bidSpotPlacePrice)
				batch := Utils.GetRandomHexString(12)
				p.openPosition(timestamp, Exchange.TradeConfig{
					Batch:  batch,
					Pair:   pair,
					Type:   Exchange.TradeTypeOpenShort,
					Price:  bidSpotPlacePrice,
					Amount: amount / constContractRatio["eth"],
					Limit:  p.config.LimitOpenRatio,
				})
			}

		}
	} else if lastDiff < 0 && avg10-avg20 > 0 {
		if avg5 > avg10 && (current.High+current.Low)/2 > avg5 {
			err2, _, askSpotPlacePrice, _, _ := Task.CalcDepthPrice(true, constContractRatio, p.future, pair, p.config.UnitAmount)
			if err2 != nil {
				Logger.Infof("深度无效")
				return
			}

			if askSpotPlacePrice > avg5 {
				Logger.Infof("执行做多,做多价格:%.2f", askSpotPlacePrice)
				batch := Utils.GetRandomHexString(12)
				p.openPosition(timestamp, Exchange.TradeConfig{
					Batch:  batch,
					Pair:   pair,
					Type:   Exchange.TradeTypeOpenLong,
					Price:  askSpotPlacePrice,
					Amount: amount / constContractRatio["eth"],
					Limit:  p.config.LimitOpenRatio,
				})
			}
		}
	}

}

func (p *TrendTask) openPosition(timestamp int64, tradeConfig Exchange.TradeConfig) {

	channelFuture := Task.ProcessTradeRoutine(p.future, tradeConfig, nil)

	var waitGroup sync.WaitGroup
	var futureResult Task.TradeResult

	waitGroup.Add(1)
	go func() {
		select {
		case futureResult = <-channelFuture:
			Logger.Debugf("交易结果:%v", futureResult)
			waitGroup.Done()
		}
	}()

	waitGroup.Wait()

	if err := p.fundManager.OpenPosition(tradeConfig.Batch,
		timestamp,
		tradeConfig.Pair,
		tradeConfig.Type,
		futureResult.AvgPrice,
		futureResult.DealAmount); err != nil {
		Logger.Error("Fail to save fund info")
	}

	if futureResult.Error == Task.TaskErrorSuccess {
		var position TrendPosition
		var config Exchange.TradeConfig
		config.Batch = tradeConfig.Batch
		config.Amount = futureResult.DealAmount
		config.Limit = tradeConfig.Limit
		config.Pair = tradeConfig.Pair
		config.Type = tradeConfig.Type
		config.Price = tradeConfig.Price
		coin := Exchange.ParsePair(config.Pair)[0]
		position.TimeStamp = timestamp
		position.Amount = config.Amount * constContractRatio[coin]
		position.config = config
		p.positions[p.positionIndex] = &position
		p.positionIndex++
	} else {
		// 开仓失败，手工检查
		p.fundManager.ClosePosition(tradeConfig.Batch, 0, MongoTrend.FundStatusError)
		p.errorCounter++
		Logger.Errorf("Trade Error:%v", futureResult.Error)
		if p.errorCounter > 100 {
			p.status = Task.StatusError
		}
	}

}

// 如果需要平仓，则返回true，后续不再开仓；否则返回false，后续可能开仓
func (p *TrendTask) CheckClosePosition(values []Exchange.KlineValue) bool {

	if p.positions == nil || len(p.positions) == 0 {
		return false
	}

	length := len(values)
	current := values[length-1]
	highPrice := current.High
	lowPrice := current.Low
	closePrice := current.Close

	for index, position := range p.positions {
		var lossLimitPrice, placeClosePrice float64
		var openLongFlag bool
		var closeFlag bool
		config := position.config
		Logger.Debugf("仓位配置:%v", config)

		if int64(current.OpenTime) == position.TimeStamp {
			Logger.Info("忽略开仓期间的价格波动")
			return false
		}

		if config.Type == Exchange.TradeTypeBuy || config.Type == Exchange.TradeTypeOpenLong {
			lossLimitPrice = config.Price * (1 - p.config.LimitCloseRatio)
			// targetProfitPrice = openPrice * (1 + profitLimit)
			openLongFlag = true
		} else if config.Type == Exchange.TradeTypeSell || config.Type == Exchange.TradeTypeOpenShort {
			lossLimitPrice = config.Price * (1 + p.config.LimitCloseRatio)
			// targetProfitPrice = openPrice * (1 - lossLimit)
			openLongFlag = false
		} else {
			Logger.Errorf("无效的交易类型")
			continue
		}

		err2, _, askSpotPlacePrice, _, bidSpotPlacePrice := Task.CalcDepthPrice(true, constContractRatio, p.future, pair, p.config.UnitAmount)
		if err2 != nil {
			return false
		}

		if openLongFlag {
			if lowPrice < lossLimitPrice {
				Logger.Debugf("做多止损,止损价格:%v", lossLimitPrice)
				placeClosePrice = bidSpotPlacePrice
				closeFlag = true
			}
		} else {
			if highPrice > lossLimitPrice {
				Logger.Debugf("做空止损,止损价格:%v", lossLimitPrice)
				placeClosePrice = askSpotPlacePrice
				closeFlag = true
			}
		}

		array5 := values[length-5 : length]
		array10 := values[length-10 : length]
		array20 := values[length-20 : length]

		avg5 := Exchange.GetAverage(5, array5)
		avg10 := Exchange.GetAverage(10, array10)
		avg20 := Exchange.GetAverage(20, array20)

		Logger.Debugf("[Avg5]%.2f [Avg10]%.2f [Avg20]%.2f", avg5, avg10, avg20)

		if openLongFlag {
			if avg5 > avg10 && avg10 > avg20 {

			} else {
				Logger.Debugf("做多趋势破坏平仓")
				placeClosePrice = bidSpotPlacePrice
				closeFlag = true
				goto __DONE
			}

			// if closePrice < avg10 {
			// 价格柱三分之一突破十日均线平仓
			if (closePrice < avg10) && (highPrice-avg10) < (avg10-closePrice) {
				Logger.Debugf("突破十日线平仓")
				placeClosePrice = bidSpotPlacePrice
				closeFlag = true
				goto __DONE
			}
		} else {
			if avg5 < avg10 && avg10 < avg20 {

			} else {
				log.Printf("做空趋势破坏平仓")
				placeClosePrice = askSpotPlacePrice
				closeFlag = true
				goto __DONE
			}

			// if closePrice > avg10 {
			// 当前价格高于十日均线并且突出长度大于当天价格柱的1/3
			if (closePrice > avg10) && (closePrice-avg10) > (avg10-lowPrice) {
				log.Printf("突破十日线平仓")
				placeClosePrice = askSpotPlacePrice
				closeFlag = true
				goto __DONE
			}
		}
	__DONE:
		if closeFlag {
			config := position.config
			config.Price = placeClosePrice
			config.Type = Exchange.RevertTradeType(config.Type)
			channelFuture := Task.ProcessTradeRoutine(p.future, config, nil)

			var waitGroup sync.WaitGroup
			var futureResult Task.TradeResult

			waitGroup.Add(1)
			go func() {
				select {
				case futureResult = <-channelFuture:
					Logger.Debugf("交易结果:%v", futureResult)
					waitGroup.Done()
				}
			}()

			waitGroup.Wait()

			if futureResult.Error == Task.TaskErrorSuccess {
				Logger.Infof("平仓成功")
				delete(p.positions, index)
				p.fundManager.ClosePosition(config.Batch, futureResult.AvgPrice, MongoTrend.FundStatusClose)
			} else {
				Logger.Infof("平仓失败")
				p.fundManager.ClosePosition(config.Batch, futureResult.AvgPrice, MongoTrend.FundStatusError)
				p.errorCounter++
				Logger.Errorf("Trade Error:%v", futureResult.Error)
				if p.errorCounter > 100 {
					p.status = Task.StatusError
				}
			}

			return true
		}

		return false
	}

	return false
}

func main() {
	mongo := new(Mongo.ExchangeDB)
	if err := mongo.Connect(); err != nil {
		Logger.Errorf("Error:%v", err)
		return
	}

	err, record := mongo.FindOne("OkexSpot")
	if err != nil || record == nil {
		Logger.Errorf("Invalid record")
		return
	}

	trendTask := new(TrendTask)
	trendTask.Start(record.API, record.Secret, "")
	select {}
}
