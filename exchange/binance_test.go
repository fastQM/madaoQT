package exchange

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestBinanceStreamTrade(t *testing.T) {
	binance := new(Binance)

	result := binance.GetDepthValue("eth/usdt")
	log.Printf("Result:%v", result)

}

const StopLoss = 0.1
const TargetProfit = 0.05

func TestPeriodArea(t *testing.T) {

	binance := new(Binance)
	binance.SetConfigure(Config{
		Proxy: "SOCKS5:127.0.0.1:1080",
	})

	kline := binance.GetKline("eth/usdt", "2h", 500)

	length := len(kline)
	array10 := kline[length-10 : length]
	array20 := kline[length-20 : length]

	avg10 := GetAverage(10, array10)
	avg20 := GetAverage(20, array20)

	var isOpenLong bool
	if avg10 > avg20 {
		isOpenLong = true
	} else {
		isOpenLong = false
	}

	var start int
	found := false
	if isOpenLong {

		step := 0
		for i := len(kline) - 1; i >= 0; i-- {
			array10 := kline[i-10 : i]
			array20 := kline[i-20 : i]

			avg10 := GetAverage(10, array10)
			avg20 := GetAverage(20, array20)

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
			array10 := kline[i-10 : i]
			array20 := kline[i-20 : i]

			avg10 := GetAverage(10, array10)
			avg20 := GetAverage(20, array20)

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
		var high, low float64
		log.Printf("Start is %v", time.Unix(int64(kline[start].OpenTime), 0))
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

	}

}
func TestGetKlines(t *testing.T) {

	var logs []string
	binance := new(Binance)
	binance.SetConfigure(Config{
		Proxy: "SOCKS5:127.0.0.1:1080",
	})

	isIncrease := false
	result := binance.GetKline("eth/usdt", "1h", 500)
	// log.Printf("Result:%v", result)

	var lastDiff float64
	for i := 30; i <= len(result)-1; i++ {
		array5 := result[i-4 : i+1]
		array10 := result[i-9 : i+1]
		array20 := result[i-19 : i+1]

		avg5 := GetAverage(5, array5)
		avg10 := GetAverage(10, array10)
		avg20 := GetAverage(20, array20)

		// 1. 三条均线要保持平行，一旦顺序乱则清仓
		// 2. 开仓后，价格柱破10日均线清仓;虽然可能只是下探均线，但是说明市场强势减弱，后续可以更轻松的建仓
		// 3. 开多时，开仓价格应该高于十日均线；开空时，开仓价格需要低于十日均线

		log.Printf("Time:%s Avg5:%v Avg10:%v Avg20:%v Diff:%v", time.Unix(int64(result[i].OpenTime), 0), avg5, avg10, avg20, avg10-avg20)
		if lastDiff != 0 {
			if lastDiff > 0 && avg10-avg20 < 0 && (!isIncrease) {
				if avg5 < avg10 {
					msg := fmt.Sprintf("卖出点:%s 卖出价格:%v", time.Unix(int64(result[i].OpenTime), 0), result[i].Open)
					logs = append(logs, msg)
					log.Printf("%s", msg)

					for j := i; j < len(result); j++ {
						if CheckTestClose(TradeTypeOpenShort, result[i].Open, StopLoss, result[j-20:j+1]) {
							msg = fmt.Sprintf("平仓日期:%v, 平仓价格:%v, 盈利：%v", time.Unix(int64(result[j].OpenTime), 0), result[j+1].Open, (result[i].Open-result[j+1].Open)*100/result[i].Open)
							log.Printf("%s", msg)
							logs = append(logs, msg)
							break
						}
					}
				}
			} else if lastDiff < 0 && avg10-avg20 > 0 && isIncrease {
				if avg5 > avg10 {
					msg := fmt.Sprintf("买入点:%v 买入价格:%v", time.Unix(int64(result[i].OpenTime), 0), result[i].Open)
					logs = append(logs, msg)
					log.Printf("%s", msg)

					for j := i; j < len(result); j++ {
						if CheckTestClose(TradeTypeOpenLong, result[i].Open, StopLoss, result[j-20:j+1]) {
							msg = fmt.Sprintf("平仓日期:%v, 平仓价格:%v, 盈利：%v", time.Unix(int64(result[j].OpenTime), 0), result[j+1].Open, (result[j+1].Open-result[i].Open)*100/result[i].Open)
							logs = append(logs, msg)
							log.Printf("%s", msg)
							break
						}
					}
				}
			}
		}

		lastDiff = avg10 - avg20

	}

	for _, msg := range logs {
		log.Printf("Log:%s", msg)
	}

}

func CheckTestClose(tradeType TradeType, openPrice float64, lossLimit float64, values []KlineValue) bool {
	var lossLimitPrice float64
	var openLongFlag bool
	if tradeType == TradeTypeBuy || tradeType == TradeTypeOpenLong {
		lossLimitPrice = openPrice * (1 - lossLimit)
		// targetProfitPrice = openPrice * (1 + profitLimit)
		openLongFlag = true
	} else {
		lossLimitPrice = openPrice * (1 + lossLimit)
		// targetProfitPrice = openPrice * (1 - lossLimit)
		openLongFlag = false
	}

	if values != nil && len(values) >= 20 {
		length := len(values)
		highPrice := values[length-1].High
		lowPrice := values[length-1].Low
		closePrice := values[length-1].Close
		if openLongFlag {
			if lowPrice < lossLimitPrice {
				log.Printf("做多止损")
				return true
			}
		} else {
			if highPrice > lossLimitPrice {
				log.Printf("做空止损")
				return true
			}
		}

		array5 := values[length-5 : length]
		array10 := values[length-10 : length]
		array20 := values[length-20 : length]

		avg5 := GetAverage(5, array5)
		avg10 := GetAverage(10, array10)
		avg20 := GetAverage(20, array20)

		if openLongFlag {
			if avg5 > avg10 && avg10 > avg20 {

			} else {
				log.Printf("做多趋势破坏平仓")
				return true
			}

			// if closePrice < avg10 {
			// if (avg10-lowPrice)/(highPrice-lowPrice) > (1 / 3) {
			// 低点到十日均线长于高点到十日均线
			if (closePrice < avg10) && (highPrice-avg10) < (avg10-lowPrice) {
				log.Printf("突破十日线平仓")
				return true
			}
		} else {
			if avg5 < avg10 && avg10 < avg20 {

			} else {
				log.Printf("做空趋势破坏平仓")
				return true
			}

			// if closePrice > avg10 {
			// if (highPrice-avg10)/(highPrice-lowPrice) > (1 / 3) {
			if (closePrice > avg10) && (highPrice-avg10) > (avg10-lowPrice) {
				log.Printf("突破十日线平仓")
				return true
			}
		}
	}

	return false
}

func TestCheckBottomSupport(t *testing.T) {

	var logs []string
	binance := new(Binance)
	binance.SetConfigure(Config{
		Proxy: "SOCKS5:127.0.0.1:1080",
	})

	result := binance.GetKline("eth/usdt", "2h", 500)

	logs = CheckBottomSupport("eth", result)

	for _, msg := range logs {
		log.Printf("Log:%s", msg)
	}
}

func TestKlineRatio(t *testing.T) {

	// var logs []string

	binance := new(Binance)
	binance.SetConfigure(Config{
		Proxy: "SOCKS5:127.0.0.1:1080",
	})

	result := binance.GetKline("eth/usdt", "1d", 500)

	var lastRatio float64

	pre10 := result[0:10]
	preAvg10 := GetAverage(10, pre10)

	for i := 10; i <= len(result)-1; i++ {
		array10 := result[i-9 : i+1]
		avg10 := GetAverage(10, array10)

		ratio := (avg10 - preAvg10) / 1
		log.Printf("[%s] Ratio:%.2f", time.Unix(int64(result[i].OpenTime), 0).String(), ratio)
		lastRatio = ratio

		// 发生逆转，重新选择起点
		if ratio > 0 && lastRatio < 0 {

		} else if ratio < 0 && lastRatio > 0 {

		}
	}
}
