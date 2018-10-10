package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	"madaoQT/exchange"
	Utils "madaoQT/utils"
)

// Get the list from the link: http://www.bestopview.com/stocklist.html

const IsFromInternet = true

type StocksTrend struct {
	stocks  []string
	last    map[string]exchange.KlineValue
	tencent *exchange.TencentStock
	klines  map[string][]exchange.KlineValue

	UpdateKlinesFlag bool
	TestModeFlag     bool
}

type ResultValues struct {
	stock  string
	profit int
	loss   int
	final  float64
}

func (p *StocksTrend) InitStockList() bool {

	if p.stocks == nil || len(p.stocks) == 0 {
		list, err := ioutil.ReadFile("./hongkong.txt")
		if err != nil {
			log.Printf("error:%v", err)
			return false
		}

		lines := strings.Split(string(list), "\r\n")
		for _, line := range lines {
			values := strings.Split(line, " ")
			p.stocks = append(p.stocks, "hk0"+values[0])
		}
	}
	// p.stocks = []string{"hk00342"}

	p.last = make(map[string]exchange.KlineValue)
	p.tencent = new(exchange.TencentStock)
	p.klines = make(map[string][]exchange.KlineValue)

	return true
}

func (p *StocksTrend) UpdateKlines() {
	log.Printf("Updating the klines......")

	for _, stock := range p.stocks {
		Utils.SleepAsyncByMillisecond(100)
		klines := p.tencent.GetDialyKlines(2017, stock)
		exchange.SaveHistory(stock, klines)
	}

	log.Printf("Klines are updated!")
}

func (p *StocksTrend) UpdatePrices() {
	var lists []string
	var combinedList string
	var counter int

	for _, stock := range p.stocks {
		if combinedList == "" {
			combinedList = stock
		} else {
			combinedList = combinedList + "," + stock
		}
		if counter < 500 {
			counter++
		} else {
			lists = append(lists, combinedList)
			combinedList = ""
			counter = 0
		}

	}

	if combinedList != "" {
		lists = append(lists, combinedList)
	}

	for _, combined := range lists {
		Utils.SleepAsyncByMillisecond(100)
		prices := p.tencent.GetHKMultipleLast(combined)
		for code, price := range prices {
			p.last[code] = price
		}
	}

	log.Printf("Update prices successfully!")
	// for key, price := range p.last {
	// 	log.Printf("Stock:%v Price:%v", key, price)
	// }

	// for {
	// 	select {
	// 	case <-time.After(5 * time.Second):
	// 		for _, combined := range lists {
	// 			Utils.SleepAsyncByMillisecond(100)
	// 			prices := p.tencent.GetMultipleLast(combined)
	// 			for code, price := range prices {
	// 				p.last[code] = price
	// 			}
	// 		}

	// 	}
	// }
}

func checkTimeValid() bool {

	location, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Now()
	periodStart1 := time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, location)
	periodEnd1 := time.Date(now.Year(), now.Month(), now.Day(), 11, 30, 0, 0, location)

	periodStart2 := time.Date(now.Year(), now.Month(), now.Day(), 13, 00, 0, 0, location)
	periodEnd2 := time.Date(now.Year(), now.Month(), now.Day(), 15, 00, 0, 0, location)

	// log.Printf("%v %v", now.After(periodStart3), now.Before(periodEnd3))

	if now.After(periodStart1) && now.Before(periodEnd1) ||
		(now.After(periodStart2) && now.Before(periodEnd2)) {
		return true

	} else {
		return false
	}
}

func (p *StocksTrend) Start() {
	if !p.InitStockList() {
		log.Printf("fail to init stock list")
		return
	}

	if p.UpdateKlinesFlag {
		p.UpdateKlines()
	}

	if p.TestModeFlag {
		p.Test()
	} else {
		for {
			select {
			case <-time.After(5 * time.Second):
				if checkTimeValid() {
					// if true {
					p.UpdatePrices()
					p.Watch()
				} else {
					log.Printf("非工作时间")
					Utils.SleepAsyncBySecond(5)
				}
			}
		}
	}

}

func FormatTime(klines []exchange.KlineValue) []exchange.KlineValue {
	var updateKlines []exchange.KlineValue

	for _, kline := range klines {

		openTime := strconv.FormatFloat(kline.OpenTime, 'f', 0, 64)
		openTimeArray := strings.Split(openTime, "")
		year := "20" + openTimeArray[0] + openTimeArray[1]
		month := openTimeArray[2] + openTimeArray[3]
		date := openTimeArray[4] + openTimeArray[5]
		kline.Time = year + "-" + month + "-" + date
		// log.Printf("TestTime2:%s", kline.Time)
		updateKlines = append(updateKlines, kline)
	}

	return updateKlines
}

func (p *StocksTrend) Test() {
	var results []ResultValues
	var klines []exchange.KlineValue
	for _, stock := range p.stocks {
		// get klines from internet
		// log.Printf("当前股票:%s", stock)

		if p.klines[stock] == nil {
			klines = exchange.LoadHistory(stock)
			if len(klines) < 120 {
				// log.Printf("[%s]Invalid length of the klines", stock)
				continue
			}

			klines = FormatTime(klines)
			p.klines[stock] = klines
		} else {
			klines = p.klines[stock]
		}

		// log.Printf("Klines:%v", klines[len(klines)-50:])

		win, loss, ratio := StrategyTrendTest(stock, klines)
		results = append(results, ResultValues{
			stock:  stock,
			profit: win,
			loss:   loss,
			final:  ratio,
		})
	}

	var winCounter, lossCounter int
	final := 1.0
	for _, result := range results {
		log.Printf("Result:%v Final:%v", result, final)
		if result.profit > 0 {
			winCounter += result.profit
		}
		if result.loss > 0 {
			lossCounter += result.loss
		}
		if result.final != 0 && result.final != 1 {
			final *= result.final
		}
	}
	log.Printf("WIN:%d Loss:%d Final:%f", winCounter, lossCounter, final)
}

func (p *StocksTrend) Watch() {

	var klines []exchange.KlineValue
	for _, stock := range p.stocks {
		// get klines from internet
		// log.Printf("当前股票:%s", stock)

		if p.klines[stock] == nil {
			klines = exchange.LoadHistory(stock)
			if len(klines) < 120 {
				// log.Printf("[%s]Invalid length of the klines", stock)
				continue
			}

			klines = FormatTime(klines)
			p.klines[stock] = klines
		} else {
			klines = p.klines[stock]
		}

		stock = strings.TrimPrefix(stock, "sh")
		stock = strings.TrimPrefix(stock, "sz")
		if p.last[stock].Open == 0 {
			// log.Printf("[%s]Fail to get the last price", stock)
			// log.Printf("Last:%v", p.last)
			continue
		} else {
			klines = append(klines, p.last[stock])
		}

		// log.Printf("%v", klines)
		StrategyTrend(stock, klines)

	}
}

func TestCheckWeeklyBreak(dayklines []exchange.KlineValue) (bool, float64) {
	weeklines := exchange.CTPDailyKlinesToWeek(dayklines)

	// for _, tmp := range weeklines {
	// 	log.Printf("week:%v", tmp)
	// }

	length := len(weeklines)
	high, low, err := exchange.GetLastDaysArea(10, weeklines)
	if err != nil {
		log.Printf("Error:%s", err.Error())
		return false, 0
	}

	array5 := weeklines[length-5 : length]
	array10 := weeklines[length-10 : length]

	avg5 := exchange.GetAverage(5, array5)
	avg10 := exchange.GetAverage(10, array10)

	thresh := exchange.GetThreshHold(array5[0].Close, avg5, array10[0].Close, avg10)

	if weeklines[length-1].Low < thresh && weeklines[length-1].Low < low {
		// } else if lastDiff < 0 && avg10-avg20 > 0 && isIncrease && result[i].Close > high {
	} else if weeklines[length-1].High > thresh && weeklines[length-1].High > high {
		// for _, kline := range dayklines {
		// 	log.Printf("day:%v", kline)
		// }

		// for _, kline := range weeklines {
		// 	log.Printf("week:%v", kline)
		// }
		if high < thresh {
			high = thresh
		}
		return true, high
	}

	return false, 0
}

func CheckWeeklyBreak(dayklines []exchange.KlineValue) (bool, float64) {
	weeklines := exchange.CTPDailyKlinesToWeek(dayklines)

	// for _, tmp := range weeklines {
	// 	log.Printf("week:%v", tmp)
	// }

	length := len(weeklines)
	high, low, err := exchange.GetLastDaysArea(10, weeklines)
	if err != nil {
		log.Printf("Error:%s", err.Error())
		return false, 0
	}

	array5 := weeklines[length-5 : length]
	array10 := weeklines[length-10 : length]

	avg5 := exchange.GetAverage(5, array5)
	avg10 := exchange.GetAverage(10, array10)

	thresh := exchange.GetThreshHold(array5[0].Close, avg5, array10[0].Close, avg10)

	// log.Printf("Thresh:%.2f", thresh)

	if weeklines[length-1].Close < thresh && weeklines[length-1].Close < low {
		// } else if lastDiff < 0 && avg10-avg20 > 0 && isIncrease && result[i].Close > high {
	} else if weeklines[length-1].Close > thresh && weeklines[length-1].Close > high {
		// for _, kline := range dayklines {
		// 	log.Printf("day:%v", kline)
		// }

		// for _, kline := range weeklines {
		// 	log.Printf("week:%v", kline)
		// }
		if high < thresh {
			high = thresh
		}
		return true, high
	}

	return false, 0
}

func StrategyTrend(stock string, result []exchange.KlineValue) {

	stopLoss := 0.03
	last := result[len(result)-1]

	for i := len(result) - 3; i <= len(result)-1; i++ {

		high, low, err := exchange.GetLastDaysArea(10, result[:i+1])
		if err != nil {
			log.Printf("Error:%s", err.Error())
			continue
		}

		array5 := result[i-5 : i]
		array10 := result[i-10 : i]

		avg5 := exchange.GetAverage(5, array5)
		avg10 := exchange.GetAverage(10, array10)

		thresh := exchange.GetThreshHold(array5[0].Close, avg5, array10[0].Close, avg10)

		// weekbreak, weekbreakprice := CheckWeeklyBreak(result[:i+1])

		// log.Printf("High:%.2f Low:%.2f", high, low)

		today := time.Now().Format("2006-01-02")
		if result[i].Close < thresh && result[i].Close < low {

		} else if result[i].Close > thresh && result[i].Close > high {
			if thresh < high {
				thresh = high
			}

			msg := fmt.Sprintf("[%s]买入点:%s 买入价格:%v 最新价格:%v", stock, result[i].Time, thresh, last.Close)
			if today == result[i].Time {
				msg = fmt.Sprintf("===========>%s", msg)
			}

			log.Printf("%s", msg)

			for j := i + 1; j < len(result); j++ {

				if j+1 >= len(result) {
					break
				}

				// if closeFlag, closePrice := checkTestClose(exchange.TradeTypeOpenLong, high, StopLoss, result[j-20:j+1]); closeFlag {
				if closed, closePrice := CheckAreaClose(exchange.TradeTypeOpenLong, high, stopLoss, result[0:j+1]); closed {
					log.Printf("[%s]平仓点:%.2f", stock, closePrice)
				}
			}
		}

	}
}

func StrategyTrendTest(stock string, result []exchange.KlineValue) (int, int, float64) {

	var changes []float64

	countProfit := 0
	countLoss := 0
	stopLoss := 0.03
	final := float64(1)
	var profitSum, lossSum float64
	last := result[len(result)-1]

	for i := len(result) - 100; i <= len(result)-1; i++ {

		// high, low, err := exchange.GetLastDaysArea(10, result[:i+1])
		// if err != nil {
		// 	log.Printf("Error:%s", err.Error())
		// 	continue
		// }

		high, low, err := exchange.GetLastPeriodArea(result[:i+1])
		if err != nil {
			log.Printf("Error:%s", err.Error())
			continue
		}

		array5 := result[i-5 : i]
		array10 := result[i-10 : i]

		avg5 := exchange.GetAverage(5, array5)
		avg10 := exchange.GetAverage(10, array10)

		thresh := exchange.GetThreshHold(array5[0].Close, avg5, array10[0].Close, avg10)

		// weekbreak, weekbreakprice := TestCheckWeeklyBreak(result[:i+1])

		// log.Printf("High:%.2f Low:%.2f WeekBreak:%.2f", high, low, weekbreakprice)

		today := time.Now().Format("2006-01-02")
		if result[i].Low < thresh && result[i].Low < low {

		} else if result[i].High > thresh && result[i].High > high {
			if thresh < high {
				thresh = high
			}
			// if thresh < weekbreakprice {
			// 	thresh = weekbreakprice
			// }
			msg := fmt.Sprintf("[%s]买入点:%s 买入价格:%v 最新价格:%v", stock, result[i].Time, thresh, last.Close)
			if today == result[i].Time {
				log.Printf("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
			}

			log.Printf("%s", msg)

			for j := i + 1; j < len(result); j++ {

				if j+1 >= len(result) {
					break
				}

				// if closeFlag, closePrice := checkTestClose(exchange.TradeTypeOpenLong, high, StopLoss, result[j-20:j+1]); closeFlag {
				if closeFlag, closePrice := TestCheckAreaClose(exchange.TradeTypeOpenLong, high, stopLoss, result[0:j+1]); closeFlag {
					change := (closePrice - high) * 100 / high

					changes = append(changes, change)
					msg = fmt.Sprintf("[%s]平仓日期:%s, 平仓价格:%v, 盈利：%v", stock, result[j].Time, closePrice, change)
					log.Printf("%s", msg)

					if change > 0 {
						countProfit++
						profitSum += change
						// change *= 0.994
					} else {
						countLoss++
						lossSum += change
						// change *= 1.006
					}
					final *= (1.0 + change/100)
					final *= 0.9997
					i = j
					break
				}

			}
		}

	}

	// if len(logs) > 0 {
	// 	for _, msg := range logs {
	// 		log.Printf("Log:%s", msg)
	// 	}
	// }
	// log.Printf("盈利次数：%d 亏损次数 ：%d", countProfit, countLoss)
	// log.Printf("盈利求和：%f 亏损求和 ：%f 净值 ：%f", profitSum, lossSum, final)
	// exchange.CheckChange(changes)

	return countProfit, countLoss, final
}

func TestCheckAreaClose(tradeType exchange.TradeType, openPrice float64, lossLimit float64, values []exchange.KlineValue) (bool, float64) {
	var lossLimitPrice float64
	var openLongFlag bool

	debug := false

	if tradeType == exchange.TradeTypeBuy || tradeType == exchange.TradeTypeOpenLong {
		lossLimitPrice = openPrice * (1 - lossLimit)
		openLongFlag = true
	} else {
		lossLimitPrice = openPrice * (1 + lossLimit)
		openLongFlag = false
	}

	length := len(values)
	current := values[length-1]
	highPrice := values[length-1].High
	lowPrice := values[length-1].Low

	if openLongFlag {
		if lowPrice < lossLimitPrice {
			if debug {
				log.Printf("做多止损")
			}

			return true, lossLimitPrice
		}
	} else {
		if highPrice > lossLimitPrice {
			if debug {
				log.Printf("做空止损")
			}
			return true, lossLimitPrice
		}
	}

	array5 := values[length-5 : length]
	array10 := values[length-10 : length]

	avg5 := exchange.GetAverage(5, array5)
	avg10 := exchange.GetAverage(10, array10)

	thresh := exchange.GetThreshHold(array5[0].Close, avg5, array10[0].Close, avg10)

	// log.Printf("Time:%v Avg5:%.2f Avg10:%.2f 阈值:%.2f", time.Unix(int64(current.OpenTime), 0), avg5, avg10, thresh)

	if openLongFlag {

		if current.Low < thresh {

			if debug {
				log.Printf("趋势破坏平仓")
			}

			if thresh > current.High {
				return true, current.Open * (1 - 0.005)
			}

			return true, thresh * (1 - 0.005)
		}
	} else {

		if current.High > thresh {
			if debug {
				log.Printf("趋势破坏平仓")
			}

			if thresh < current.Low {
				return true, current.Open * (1 + 0.005)
			}

			return true, thresh * (1 + 0.005)
		}
	}

	return false, 0
}

func CheckAreaClose(tradeType exchange.TradeType, openPrice float64, lossLimit float64, values []exchange.KlineValue) (bool, float64) {
	var lossLimitPrice float64
	var openLongFlag bool

	debug := true

	if tradeType == exchange.TradeTypeBuy || tradeType == exchange.TradeTypeOpenLong {
		lossLimitPrice = openPrice * (1 - lossLimit)
		openLongFlag = true
	} else {
		lossLimitPrice = openPrice * (1 + lossLimit)
		openLongFlag = false
	}

	length := len(values)
	current := values[length-1]
	highPrice := values[length-1].High
	lowPrice := values[length-1].Low

	if openLongFlag {
		if lowPrice < lossLimitPrice {
			if debug {
				log.Printf("做多止损")
			}

			return true, lossLimitPrice
		}
	} else {
		if highPrice > lossLimitPrice {
			if debug {
				log.Printf("做空止损")
			}
			return true, lossLimitPrice
		}
	}

	array5 := values[length-5 : length]
	array10 := values[length-10 : length]

	avg5 := exchange.GetAverage(5, array5)
	avg10 := exchange.GetAverage(10, array10)

	thresh := exchange.GetThreshHold(array5[0].Close, avg5, array10[0].Close, avg10)

	// log.Printf("Time:%v Avg5:%.2f Avg10:%.2f 阈值:%.2f", time.Unix(int64(current.OpenTime), 0), avg5, avg10, thresh)

	if openLongFlag {

		if current.Close < thresh {

			if debug {
				log.Printf("趋势破坏平仓")
			}

			if thresh > current.Close {
				return true, current.Close
			}

			return true, thresh
		}
	} else {

		if current.Close > thresh {
			if debug {
				log.Printf("趋势破坏平仓")
			}

			if thresh < current.Close {
				return true, current.Open
			}

			return true, thresh
		}
	}

	return false, 0
}
