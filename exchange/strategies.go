package exchange

import (
	"fmt"
	"log"
	"time"
)

const StopLoss = 0.15
const TargetProfit = 0.05

func StrategyTrendTest(result []KlineValue) {

	var logs []string
	var changes []float64

	countProfit := 0
	countLoss := 0
	final := float64(1)
	var profitSum, lossSum float64

	for i := 30; i <= len(result)-1; i++ {
		// log.Printf("Current:%s", time.Unix(int64(result[i].OpenTime), 0).String())
		high, low, err := GetPeriodArea(result[:i])
		if err != nil {
			log.Printf("Error:%s", err.Error())
			continue
		}

		array5 := result[i-4 : i+1]
		array10 := result[i-9 : i+1]
		array20 := result[i-19 : i+1]

		avg5 := GetAverage(5, array5)
		avg10 := GetAverage(10, array10)
		avg20 := GetAverage(20, array20)

		// log.Printf("High:%.2f Low:%.2f Close:%.2f", high, low, result[i].Close)

		// 1. 三条均线要保持平行，一旦顺序乱则清仓
		// 2. 开仓后，价格柱破10日均线清仓;虽然可能只是下探均线，但是说明市场强势减弱，后续可以更轻松的建仓
		// 3. 开多时，开仓价格应该高于十日均线；开空时，开仓价格需要低于十日均线

		// log.Printf("Time:%s Avg5:%v Avg10:%v Avg20:%v Diff:%v", time.Unix(int64(result[i].OpenTime), 0), avg5, avg10, avg20, avg10-avg20)

		// if lastDiff > 0 && avg10-avg20 < 0 && (!isIncrease) {
		if avg10-avg20 < 0 && result[i].Close < low {
			if avg5 < avg10 {
				msg := fmt.Sprintf("卖出点:%s 卖出价格:%v", time.Unix(int64(result[i].OpenTime), 0), low)
				logs = append(logs, msg)
				// log.Printf("%s", msg)

				for j := i; j < len(result); j++ {
					if closeFlag, closePrice := checkTestClose(TradeTypeOpenShort, low, StopLoss, result[j-20:j+1]); closeFlag {
						change := (low - closePrice) * 100 / low
						changes = append(changes, change)
						msg = fmt.Sprintf("平仓日期:%v, 平仓价格:%v, 盈利：%v", time.Unix(int64(result[j].OpenTime), 0), closePrice, change)
						// log.Printf("%s", msg)
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
						final *= 0.999
						// log.Printf("当前净值:%f", final)
						i = j + 1
						logs = append(logs, msg)
						break
					}
				}
			}
			// } else if lastDiff < 0 && avg10-avg20 > 0 && isIncrease && result[i].Close > high {
		} else if avg10-avg20 > 0 && result[i].Close > high {
			if avg5 > avg10 {
				msg := fmt.Sprintf("买入点:%v 买入价格:%v", time.Unix(int64(result[i].OpenTime), 0), high)
				logs = append(logs, msg)
				// log.Printf("%s", msg)

				for j := i; j < len(result); j++ {

					if j+1 >= len(result) {
						break
					}

					if closeFlag, closePrice := checkTestClose(TradeTypeOpenLong, high, StopLoss, result[j-20:j+1]); closeFlag {
						change := (closePrice - high) * 100 / high
						changes = append(changes, change)
						msg = fmt.Sprintf("平仓日期:%v, 平仓价格:%v, 盈利：%v", time.Unix(int64(result[j].OpenTime), 0), closePrice, change)
						logs = append(logs, msg)
						// log.Printf("%s", msg)

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
						final *= 0.999
						// log.Printf("当前净值:%f", final)
						i = j + 1
						break
					}
				}
			}
		}

	}

	for _, msg := range logs {
		log.Printf("Log:%s", msg)
	}

	log.Printf("盈利次数：%d 亏损次数 ：%d", countProfit, countLoss)
	log.Printf("盈利求和：%f 亏损求和 ：%f 净值 ：%f", profitSum, lossSum, final)
}

func checkTestClose(tradeType TradeType, openPrice float64, lossLimit float64, values []KlineValue) (bool, float64) {
	var lossLimitPrice float64
	var openLongFlag bool
	debug := false
	if tradeType == TradeTypeBuy || tradeType == TradeTypeOpenLong {
		lossLimitPrice = openPrice * (1 - lossLimit)
		// targetProfitPrice = openPrice * (1 + profitLimit)
		openLongFlag = true
	} else {
		lossLimitPrice = openPrice * (1 + lossLimit)
		// targetProfitPrice = openPrice * (1 - lossLimit)
		openLongFlag = false
	}

	// log.Printf("开盘:%v 止损价格:%v", openPrice, lossLimit)

	if values != nil && len(values) >= 20 {
		length := len(values)
		highPrice := values[length-1].High
		lowPrice := values[length-1].Low
		closePrice := values[length-1].Close
		if openLongFlag {
			if lowPrice < lossLimitPrice {
				if debug {
					log.Printf("做多止损")
				}
				return true, closePrice
			}
		} else {
			if highPrice > lossLimitPrice {
				if debug {
					log.Printf("做空止损")
				}
				return true, closePrice
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
				if debug {
					log.Printf("做多趋势破坏平仓")
				}
				return true, closePrice
			}

			// if closePrice < avg10 {
			// if (avg10-lowPrice)/(highPrice-lowPrice) > (1 / 3) {
			// 低点到十日均线长于高点到十日均线
			if (closePrice < avg5) && (highPrice-avg5) < (avg5-lowPrice) {
				if debug {
					log.Printf("突破五日线平仓")
				}
				return true, closePrice
			}

			// if closePrice < avg10 {
			// 	if debug {
			// 		log.Printf("突破十日线平仓")
			// 	}
			// 	return true, closePrice
			// }
		} else {
			if avg5 < avg10 && avg10 < avg20 {

			} else {
				if debug {
					log.Printf("做空趋势破坏平仓")
				}
				return true, closePrice
			}

			// if closePrice > avg10 {
			// if (highPrice-avg10)/(highPrice-lowPrice) > (1 / 3) {
			if (closePrice > avg5) && (highPrice-avg5) > (avg5-lowPrice) {
				if debug {
					log.Printf("突破五日线平仓")
				}
				return true, closePrice
			}

			// if closePrice > avg10 {
			// 	if debug {
			// 		log.Printf("突破十日线平仓")
			// 	}
			// 	return true, closePrice
			// }
		}
	}

	return false, 0
}
