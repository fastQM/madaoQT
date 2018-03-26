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

func TestGetTicker(t *testing.T) {
	var logs []string

	countProfit := 0
	countLoss := 0
	var profitSum, lossSum float64

	// date1 := time.Date(2017, 1, 14, 0, 0, 0, 0, time.Local)
	date2 := time.Date(2018, 1, 14, 0, 0, 0, 0, time.Local)

	polo := new(PoloniexAPI)
	result := polo.GetKline("eth/usdt", date2, nil, Period2H)

	isIncrease := false
	var lastDiff float64
	for i := 30; i <= len(result)-1; i++ {

		high, _, err := GetPeriodArea(result[:i])
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
			if lastDiff > 0 && avg10-avg20 < 0 && (!isIncrease) {
				if avg5 < avg10 {
					msg := fmt.Sprintf("卖出点:%s 卖出价格:%v", time.Unix(int64(result[i].OpenTime), 0), result[i].Open)
					logs = append(logs, msg)
					log.Printf("%s", msg)

					for j := i; j < len(result); j++ {
						if CheckTestClose(TradeTypeOpenShort, result[i].Open, StopLoss, result[j-20:j+1]) {
							msg = fmt.Sprintf("平仓日期:%v, 平仓价格:%v, 盈利：%v", time.Unix(int64(result[j].OpenTime), 0), result[j+1].Open, (result[i].Open-result[j+1].Open)*100/result[i].Open)
							log.Printf("%s", msg)
							if (result[i].Open-result[j+1].Open)*100/result[i].Open > 0 {
								countProfit++
								profitSum += (result[i].Open - result[j+1].Open) * 100 / result[i].Open
							} else {
								countLoss++
								lossSum += (result[i].Open - result[j+1].Open) * 100 / result[i].Open
							}
							logs = append(logs, msg)
							break
						}
					}
				}
			} else if lastDiff < 0 && avg10-avg20 > 0 && isIncrease && result[i].Close > high {
				if avg5 > avg10 {
					msg := fmt.Sprintf("买入点:%v 买入价格:%v", time.Unix(int64(result[i].OpenTime), 0), result[i].Open)
					logs = append(logs, msg)
					log.Printf("%s", msg)

					for j := i; j < len(result); j++ {

						if j+1 >= len(result) {
							break
						}

						if CheckTestClose(TradeTypeOpenLong, result[i].Open, StopLoss, result[j-20:j+1]) {
							msg = fmt.Sprintf("平仓日期:%v, 平仓价格:%v, 盈利：%v", time.Unix(int64(result[j].OpenTime), 0), result[j+1].Open, (result[j+1].Open-result[i].Open)*100/result[i].Open)
							logs = append(logs, msg)
							log.Printf("%s", msg)

							if (result[j+1].Open-result[i].Open)*100/result[i].Open > 0 {
								countProfit++
								profitSum += (result[j+1].Open - result[i].Open) * 100 / result[i].Open
							} else {
								countLoss++
								lossSum += (result[j+1].Open - result[i].Open) * 100 / result[i].Open
							}

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
	log.Printf("盈利求和：%f 亏损求和 ：%f", profitSum, lossSum)
}

func TestFloatCompare(t *testing.T) {

}
