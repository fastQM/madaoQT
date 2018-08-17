package exchange

import (
	"io/ioutil"
	"log"
	"strings"
	"testing"

	Utils "madaoQT/utils"
)

// Get the list from the link: http://www.bestopview.com/stocklist.html
func TestGetPoloniexKlines(t *testing.T) {

	var stocks []string

	// 获取上海股票列表
	shlist, err := ioutil.ReadFile("./shstocks.txt")
	if err != nil {
		log.Printf("error:%v", err)
		return
	}

	shstocks := strings.Split(string(shlist), "\n")
	for _, stock := range shstocks {
		stocks = append(stocks, "sh"+stock)
	}

	// 获取深圳股票列表
	szlist, err := ioutil.ReadFile("./szstocks.txt")
	if err != nil {
		log.Printf("error:%v", err)
		return
	}

	szstocks := strings.Split(string(szlist), "\n")
	for _, stock := range szstocks {
		stocks = append(stocks, "sz"+stock)
	}

	stocks = []string{"sz002008"}

	for _, stock := range stocks {

		Utils.SleepAsyncByMillisecond(300)
		server := new(TencentStock)
		result := server.GetDialyKlines(2017, stock)

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

			// log.Printf("Time:%f Avg5:%v Avg10:%v Avg20:%v Diff:%v", result[i].OpenTime, avg5, avg10, avg20, avg10-avg20)
			if lastDiff != 0 {
				if lastDiff > 0 && avg10-avg20 < 0 {
					// if avg5 < avg10 && result[i].Open < avg5 {
					// 	log.Printf("卖出点:%f 卖出价格:%.2f", result[i].OpenTime, result[i].Open)
					// }
				} else if lastDiff < 0 && avg10-avg20 > 0 {
					if avg5 > avg10 && result[i].Open > avg5 {

						high, _, err := GetLastPeriodArea(result)
						if err != nil {
							log.Printf("error:%v", err)
							continue
						}

						if result[i].High > high {
							log.Printf("突破前期高点 买入点:%f 买入价格:%v.2f", result[i].OpenTime, result[i].Open)
						}
					}
				}
			}

			lastDiff = avg10 - avg20
		}
	}

}

func TestGetLastest(t *testing.T) {
	stock := "sz000001"
	server := new(TencentStock)
	result := server.GetLast(stock)
	log.Printf("Result:%v", result)
}
