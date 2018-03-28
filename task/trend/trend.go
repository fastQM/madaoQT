package trend

import (
	"errors"
	"log"
	"math"
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

// 1. 只做和大趋势相同的方向，即上升通道不做空，下降通道不做多

// TrendTask 策略适用于在短期内(1-3天)出现大幅波动(10%-30%)的市场
type TrendTask struct {
	config TrendConfig

	okexKline Exchange.IExchange
	// binance Exchange.IExchange
	future Exchange.IExchange

	status      Task.StatusType
	database    *MongoTrend.TrendMongo
	fundManager *FundManager
	balance     float64

	klines []Exchange.KlineValue

	positions     map[uint]*TrendPosition
	positionIndex uint

	checkPeriodSec time.Duration

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
const globalPeriod = "5m"
const TradingPeriodMS = 1000
const CheckingPeriodMS = 3 * 1000

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

func (p *TrendTask) GetDescription() Task.Description {

	return Task.Description{
		Name:  "trend",
		Title: "趋势策略",
		Desc:  "该策略主要跟踪大幅上涨或者下跌的趋势",
	}
}

func (p *TrendTask) GetDefaultConfig() interface{} {
	return nil
}
func (p *TrendTask) GetBalances() map[string]interface{} {

	var futures map[string]interface{}

	if p.future != nil {
		if balances := p.future.GetBalance(); balances != nil {

			balance := balances["eth"]
			futures = map[string]interface{}{
				"name":    "eth",
				"balance": balance.(map[string]interface{})["balance"].(float64),
				"bond":    balance.(map[string]interface{})["bond"].(float64),
			}

		}
	}

	return map[string]interface{}{
		"futures": futures,
	}
}
func (p *TrendTask) GetTrades() []Mongo.TradesRecord {
	return nil
}
func (p *TrendTask) GetPositions() []map[string]interface{} {
	var positions []MongoTrend.FundInfo

	err1, closedPositions := p.fundManager.GetClosedPositions()
	if err1 != nil {
		Logger.Errorf("GetPositions:Fail to get closed positions %v", err1)
		return nil
	}
	positions = append(positions, closedPositions...)

	err2, openPositions := p.fundManager.GetOpenPositions()
	if err2 != nil {
		Logger.Errorf("GetPositions:Fail to get open positions %v", err2)
		return nil
	}

	positions = append(positions, openPositions...)

	return nil
}
func (p *TrendTask) GetFailedPositions() []map[string]interface{} {
	return nil
}
func (p *TrendTask) FixFailedPosition(updateJSON string) error {
	return nil
}
func (p *TrendTask) Close() {
	return
}
func (p *TrendTask) GetStatus() Task.StatusType {
	return p.status
}

func (p *TrendTask) Start(configJSON string) error {

	Logger.Infof("%s", trendTaskExplaination)

	p.config = TrendConfig{
		UnitAmount:      50,
		LimitCloseRatio: 0.06,
		LimitOpenRatio:  0.003,
	}

	mongo := new(Mongo.ExchangeDB)
	if mongo.Connect() != nil {
		return errors.New(Task.TaskErrorMsg[Task.TaskLostMongodb])
	}

	err, record := mongo.FindOne(Exchange.NameOKEXSpot)
	if err != nil {
		return errors.New(Task.TaskErrorMsg[Task.TaskAPINotFound])
	}

	p.status = Task.StatusProcessing
	p.checkPeriodSec = CheckingPeriodMS

	p.database = new(MongoTrend.TrendMongo)
	if err := p.database.Connect(); err != nil {
		Logger.Errorf("Error:%v", err)
		return err
	}

	p.fundManager = new(FundManager)
	p.fundManager.Init(p.database.FundCollection)
	p.positions = make(map[uint]*TrendPosition)
	p.loadPosition()

	// p.binance = new(Exchange.Binance)
	// p.binance.SetConfigure(Exchange.Config{
	// 	Proxy: "SOCKS5:127.0.0.1:1080",
	// })

	p.okexKline = new(Exchange.OkexRestAPI)
	p.okexKline.SetConfigure(Exchange.Config{
		Proxy: "SOCKS5:127.0.0.1:1080",
	})

	Logger.Info("启动OKEx合约监视程序")
	futureExchange := new(Exchange.OKExAPI)
	futureExchange.SetConfigure(Exchange.Config{
		API:    record.API,
		Secret: record.Secret,
		Custom: map[string]interface{}{
			"exchangeType": Exchange.ExchangeTypeFuture,
			"period":       "quarter",
		},
		Proxy: "SOCKS5:127.0.0.1:1080",
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
					p.future.GetDepthValue(pair)

				} else if event == Exchange.EventLostConnection {
					if p.status != Task.StatusNone && p.status != Task.StatusError {
						if p.future != nil {
							p.future.Close()
							p.future = nil
							go Task.Reconnect(futureExchange)
						}
					}
				}
			case <-time.After(p.checkPeriodSec * time.Millisecond):
				if p.status == Task.StatusError || p.status == Task.StatusNone {
					Logger.Debug("状态异常或退出")
					return
				}

				p.database.Refresh()
				p.Watch()
			}
		}
	}()

	return nil
}

func (p *TrendTask) loadPosition() {
	var records []MongoTrend.FundInfo
	var err error

	if err, records = p.fundManager.GetOpenPositions(); err != nil {
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
			position.TimeStamp = record.OpenTime.Unix()
			position.Amount = config.Amount
			position.config = config
			p.positions[p.positionIndex] = &position
			p.positionIndex++
			Logger.Infof("Position:%v", position)
		}
	}
}

func (p *TrendTask) checkFunds(coin string, latestPrice float64) float64 {

	var usedAmount float64
	for _, position := range p.positions {
		usedAmount += position.Amount
	}

	var balance map[string]interface{}

	if usedAmount == 0 && p.balance == 0 {
		balance = p.GetBalances()["futures"].(map[string]interface{})
		p.balance = balance["balance"].(float64) * latestPrice
		p.balance = float64(int(p.balance / 10))
	}

	Logger.Infof("余额:%v 开仓单位：%v 已开仓：%v 开仓总量:%v", p.balance, p.config.UnitAmount, usedAmount, p.balance)
	if (usedAmount + p.config.UnitAmount) > p.balance {
		if math.Floor(p.balance-usedAmount) > 0 {
			return math.Floor(p.balance - usedAmount)
		}

		return 0
	}
	return p.config.UnitAmount

}

func (p *TrendTask) Watch() {

	// kline := p.binance.GetKline(pair, globalPeriod, 200)
	if p.klines == nil || len(p.klines) == 0 {
		p.klines = p.okexKline.GetKline(pair, Exchange.KlinePeriod5Min, 200)
		if p.klines == nil {
			Logger.Errorf("未获取均线信息")
			return
		}
	} else {
		current := p.klines[len(p.klines)-1]

		if time.Now().Unix()-int64(current.OpenTime) > 400 {
			p.klines = nil
			Logger.Info("更新均线")
			return

		} else {
			err2, _, askFuturePlacePrice, _, bidFuturePlacePrice := Task.CalcDepthPrice(true, constContractRatio, p.future, pair, p.config.UnitAmount)
			if err2 != nil {
				Logger.Debugf("深度无效:%s", err2.Error())
				return
			}

			Logger.Infof("当前深度价格:%.2f %.2f", askFuturePlacePrice, bidFuturePlacePrice)
			// 挂的买单高于现有当前最高价才算最高价
			if bidFuturePlacePrice > current.High {
				p.klines[len(p.klines)-1].High = bidFuturePlacePrice
			}

			if askFuturePlacePrice < current.Low {
				p.klines[len(p.klines)-1].Low = askFuturePlacePrice
			}

			length := len(p.klines)
			array5 := p.klines[length-5 : length]
			array10 := p.klines[length-10 : length]
			array20 := p.klines[length-20 : length]

			avg5 := Exchange.GetAverage(5, array5)
			avg10 := Exchange.GetAverage(10, array10)
			avg20 := Exchange.GetAverage(20, array20)

			if avg5 > avg10 && avg10 > avg20 {
				Logger.Infof("做多使用买入价格作为收盘价")
				p.klines[len(p.klines)-1].Close = askFuturePlacePrice
			} else if avg20 > avg10 && avg10 > avg5 {
				Logger.Infof("卖空使用卖出价格作为收盘价")
				p.klines[len(p.klines)-1].Close = bidFuturePlacePrice
			}
		}
	}

	kline := p.klines
	if kline == nil || len(kline) < 20 {
		Logger.Errorf("无效K线数据")
		return
	}

	length := len(kline)
	current := kline[length-1]

	Logger.Infof("[High]%.2f [Open]%.2f [Close]%.2f [Low]%.2f [Volumn]%.2f", current.High, current.Open, current.Close, current.Low, current.Volumn)
	Logger.Infof("服务器时间:%d[%s] 当前时间:%d[%s]",
		int(current.OpenTime), time.Unix(int64(current.OpenTime), 0).Format(Global.TimeFormat),
		time.Now().Unix(), time.Now().Format(Global.TimeFormat))

	// var timeFlag bool
	// if time.Now().Unix()-int64(current.OpenTime) > 240 {
	// 	timeFlag = true
	// }

	// 是否需要减仓
	if p.CheckClosePosition(kline, int(current.OpenTime)) {
		p.adjustDuration(true)
		return
	}

	// 资金管理
	amount := p.checkFunds("eth", current.Close)
	if amount == 0 {
		Logger.Info("无可用仓位...不开仓")
		p.adjustDuration(false)
		return
	}

	if true {
		if p.checkBreakPosition(kline, amount) {
			p.adjustDuration(true)
		}
	}
}

func (p *TrendTask) checkBreakPosition(kline []Exchange.KlineValue, amount float64) bool {
	err, high, low := p.getLastPeriodArea(kline)
	if err != nil {
		Logger.Errorf("Error in getLastPeriodArea():%s", err.Error())
		return false
	}

	length := len(kline)
	current := kline[length-1]

	array5 := kline[length-5 : length]
	array10 := kline[length-10 : length]
	array20 := kline[length-20 : length]

	avg5 := Exchange.GetAverage(5, array5)
	avg10 := Exchange.GetAverage(10, array10)
	avg20 := Exchange.GetAverage(20, array20)

	Logger.Infof("前一个周期波动区间 High: %.2f Low: %.2f Avg5: %.2f Avg10: %.2f Avg20: %.2f", high, low, avg5, avg10, avg20)

	if (current.Close > high) && (avg10 > avg20) && (avg5 > avg10) {

		err2, _, askFuturePlacePrice, _, _ := Task.CalcDepthPrice(true, constContractRatio, p.future, pair, p.config.UnitAmount)
		if err2 != nil {
			Logger.Debugf("深度无效: %s", err2.Error())
			return false
		}

		Logger.Infof("突破前期高点,做多价格:%.2f", askFuturePlacePrice)
		batch := Utils.GetRandomHexString(12)
		timestamp := int64(current.OpenTime)
		p.openPosition(timestamp, Exchange.TradeConfig{
			Batch:  batch,
			Pair:   pair,
			Type:   Exchange.TradeTypeOpenLong,
			Price:  askFuturePlacePrice,
			Amount: amount,
			Limit:  p.config.LimitOpenRatio,
		})
		return true

	} else if (current.Close < low) && (avg10 < avg20) && (avg5 < avg10) {
		err2, _, _, _, bidFuturePlacePrice := Task.CalcDepthPrice(true, constContractRatio, p.future, pair, p.config.UnitAmount)
		if err2 != nil {
			Logger.Infof("深度无效")
			return false
		}

		Logger.Infof("突破前期低点加仓，做空价格:%.2f", bidFuturePlacePrice)

		batch := Utils.GetRandomHexString(12)
		timestamp := int64(current.OpenTime)
		p.openPosition(timestamp, Exchange.TradeConfig{
			Batch:  batch,
			Pair:   pair,
			Type:   Exchange.TradeTypeOpenShort,
			Price:  bidFuturePlacePrice,
			Amount: amount,
			Limit:  p.config.LimitOpenRatio,
		})
		return true
	}

	return false
}

func (p *TrendTask) adjustDuration(hasTrade bool) {

	if hasTrade {
		if p.checkPeriodSec == CheckingPeriodMS {
			Logger.Debugf("检测周期变成1秒")
			p.checkPeriodSec = TradingPeriodMS
		}
	} else {
		if p.checkPeriodSec == TradingPeriodMS {
			// 5 minutes without trading
			Logger.Debugf("检测周期变成300毫秒")
			p.checkPeriodSec = CheckingPeriodMS
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
		position.TimeStamp = timestamp
		position.Amount = config.Amount
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
func (p *TrendTask) CheckClosePosition(values []Exchange.KlineValue, currentKlineStart int) bool {

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

		var timeFlag bool
		if time.Now().Unix()-int64(currentKlineStart) > 240 {
			timeFlag = true
		}

		if openLongFlag {
			// 还要考虑瞬时价格突变的保护措施
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
			if (closePrice < avg5) && (highPrice-avg5) < (avg5-lowPrice) && timeFlag {
				Logger.Debugf("突破五日线平仓")
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
			if (closePrice > avg5) && (highPrice-avg5) > (avg5-lowPrice) && timeFlag {
				log.Printf("突破五日线平仓")
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

				if len(p.positions) == 0 {
					Logger.Infof("全部平仓,无仓位")
					p.adjustDuration(false)
					p.balance = 0
				}
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

func (p *TrendTask) getLastPeriodArea(kline []Exchange.KlineValue) (err error, high float64, low float64) {

	var start int
	found := false

	length := len(kline)
	array10 := kline[length-10 : length]
	array20 := kline[length-20 : length]

	avg10 := Exchange.GetAverage(10, array10)
	avg20 := Exchange.GetAverage(20, array20)

	var isOpenLong bool
	if avg10 > avg20 {
		isOpenLong = true
	} else {
		isOpenLong = false
	}

	if isOpenLong {

		step := 0
		for i := len(kline) - 1; i >= 0; i-- {

			if i-20 < 0 {
				start = i
				found = true
				break
			}

			array10 := kline[i-10 : i]
			array20 := kline[i-20 : i]

			avg10 := Exchange.GetAverage(10, array10)
			avg20 := Exchange.GetAverage(20, array20)

			if step == 0 {
				if avg10 < avg20 {
					step = 1
					continue
				}
			} else if step == 1 {
				if avg10 > avg20 {
					step = 2
					continue
				}
			} else if step == 2 {
				if avg10 < avg20 {
					start = i
					found = true
					break
				}
			}
		}

	} else {
		step := 0
		for i := len(kline) - 1; i >= 0; i-- {

			if i-20 < 0 {
				start = i
				found = true
				break
			}

			array10 := kline[i-10 : i]
			array20 := kline[i-20 : i]

			avg10 := Exchange.GetAverage(10, array10)
			avg20 := Exchange.GetAverage(20, array20)

			if step == 0 {
				if avg10 > avg20 {
					step = 1
					continue
				}
			} else if step == 1 {
				if avg10 < avg20 {
					step = 2
					continue
				}
			} else if step == 2 {
				if avg10 > avg20 {
					start = i
					found = true
					break
				}
			}
		}
	}

	if found {
		high = 0
		low = 0
		// Logger.Infof("区间起点:%v", time.Unix(int64(kline[start].OpenTime), 0))
		for i := start; i < len(kline)-1; i++ {
			if high == 0 {
				high = kline[i].High
			} else if high < kline[i].High {
				high = kline[i].High
			}

			if low == 0 {
				low = kline[i].Low
			} else if low > kline[i].Low {
				low = kline[i].Low
			}
		}

		return nil, high, low

	}

	return errors.New("Perios is not Found"), 0, 0

}
