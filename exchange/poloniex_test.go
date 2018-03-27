package exchange

import (
	"fmt"
	"log"
	"testing"
	"time"
)

const Period5Min = 5 * 60
const Period15Min = 15 * 60
const Period30Min = 30 * 60
const Period2H = 120 * 60
const Period4H = 240 * 60
const Period1Day = 86400

/*
[eth]
1. Period5Min
2018/03/26 14:27:51 盈利次数：1039 亏损次数 ：666
2018/03/26 14:27:51 盈利求和：1179.919932 亏损求和 ：-284.852370
胜率 60%


2. Period15Min
2018/03/26 14:28:14 盈利次数：347 亏损次数 ：205
2018/03/26 14:28:14 盈利求和：698.230341 亏损求和 ：-144.313904
胜率 63%

3. Period30Min
2018/03/26 14:28:37 盈利次数：183 亏损次数 ：98
2018/03/26 14:28:37 盈利求和：638.228734 亏损求和 ：-89.056961
胜率 65%


4. Period2H
2018/03/26 14:29:39 盈利次数：55 亏损次数 ：17
2018/03/26 14:29:39 盈利求和：500.722605 亏损求和 ：-39.249346
胜率 76%

5. Period4H
2018/03/26 14:29:59 盈利次数：29 亏损次数 ：9
2018/03/26 14:29:59 盈利求和：502.975330 亏损求和 ：-29.038794

6. Period1Day
2018/03/26 14:30:23 盈利次数：10 亏损次数 ：1
2018/03/26 14:30:23 盈利求和：574.010105 亏损求和 ：-10.860438


[btc]
1.Period5Min
2018/03/26 14:31:37 盈利次数：1109 亏损次数 ：652
2018/03/26 14:31:37 盈利求和：839.472408 亏损求和 ：-202.113165
胜率 62%

2. Period15Min
2018/03/26 14:32:35 盈利次数：387 亏损次数 ：237
2018/03/26 14:32:35 盈利求和：504.385143 亏损求和 ：-136.847186
胜率 62%

3.Period30Min
2018/03/26 14:33:01 盈利次数：219 亏损次数 ：129
2018/03/26 14:33:01 盈利求和：411.806138 亏损求和 ：-113.672792
胜率 63%

4. Period2H
2018/03/26 14:33:24 盈利次数：56 亏损次数 ：27
2018/03/26 14:33:24 盈利求和：294.995289 亏损求和 ：-47.830697

5.Period4H
2018/03/26 14:33:44 盈利次数：36 亏损次数 ：11
2018/03/26 14:33:44 盈利求和：237.889534 亏损求和 ：-20.047389

5. Period1Day
2018/03/26 14:34:06 盈利次数：6 亏损次数 ：2
2018/03/26 14:34:06 盈利求和：226.815438 亏损求和 ：-6.503738
*/

func TestGetKline(t *testing.T) {
	var logs []string

	countProfit := 0
	countLoss := 0
	final := float64(1)
	var profitSum, lossSum float64

	// date1 := time.Date(2017, 1, 14, 0, 0, 0, 0, time.Local)
	date2 := time.Date(2018, 3, 1, 0, 0, 0, 0, time.Local)

	polo := new(PoloniexAPI)
	// result := polo.GetKline("eth/usdt", date2, nil, Period15Min)
	result := polo.GetKline("eth/usdt", date2, nil, Period5Min)

	// isIncrease := true
	var lastDiff float64
	for i := 30; i <= len(result)-1; i++ {

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

		// 1. 三条均线要保持平行，一旦顺序乱则清仓
		// 2. 开仓后，价格柱破10日均线清仓;虽然可能只是下探均线，但是说明市场强势减弱，后续可以更轻松的建仓
		// 3. 开多时，开仓价格应该高于十日均线；开空时，开仓价格需要低于十日均线

		// log.Printf("Time:%s Avg5:%v Avg10:%v Avg20:%v Diff:%v", time.Unix(int64(result[i].OpenTime), 0), avg5, avg10, avg20, avg10-avg20)
		if lastDiff != 0 {
			// if lastDiff > 0 && avg10-avg20 < 0 && (!isIncrease) {
			if avg10-avg20 < 0 && result[i].Close < low {
				if avg5 < avg10 {
					msg := fmt.Sprintf("卖出点:%s 卖出价格:%v", time.Unix(int64(result[i].OpenTime), 0), low)
					logs = append(logs, msg)
					log.Printf("%s", msg)

					for j := i; j < len(result); j++ {
						if closeFlag, closePrice := CheckTestClose(TradeTypeOpenShort, low, StopLoss, result[j-20:j+1]); closeFlag {
							change := (low - closePrice) * 100 / low
							msg = fmt.Sprintf("平仓日期:%v, 平仓价格:%v, 盈利：%v", time.Unix(int64(result[j].OpenTime), 0), closePrice, change)
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
							final *= 0.999
							log.Printf("当前净值:%f", final)
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
					log.Printf("%s", msg)

					for j := i; j < len(result); j++ {

						if j+1 >= len(result) {
							break
						}

						if closeFlag, closePrice := CheckTestClose(TradeTypeOpenLong, high, StopLoss, result[j-20:j+1]); closeFlag {
							change := (closePrice - high) * 100 / high
							msg = fmt.Sprintf("平仓日期:%v, 平仓价格:%v, 盈利：%v", time.Unix(int64(result[j].OpenTime), 0), closePrice, change)
							logs = append(logs, msg)
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
							final *= 0.999
							log.Printf("当前净值:%f", final)
							i = j + 1
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

	log.Printf("盈利次数：%d 亏损次数 ：%d", countProfit, countLoss)
	log.Printf("盈利求和：%f 亏损求和 ：%f 净值 ：%f", profitSum, lossSum, final)
}
