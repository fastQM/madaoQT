package main

import (
	"log"
	"madaoqt/redis"
)

func checkRatio(values []redis.ChartItem) float64 {
	var lowest float64
	var highest float64

	start := values[0].Open

	for _, item := range values {
		if lowest == 0 || item.Low < lowest {
			lowest = item.Low
		}

		if highest == 0 || item.High > highest {
			highest = item.High
		}
	}

	ratio := (highest - start) / start

	return ratio
}

func main() {

	orderConn := new(redis.RedisOrder)
	orders, err := orderConn.LoadOrders("poloniex")
	if err != nil {
		log.Printf("Error:%v", err.Error())
	}
	log.Printf("orders:%v", orders)

	var counter int
	length := len(orders)

	for i := 0; i < len(orders); i++ {
		historyConn := new(redis.ChartsHistory)
		err := historyConn.LoadCharts(orders[i].Pair, 0)
		if err != nil {
			log.Printf("Error:%v", err.Error())
			return
		}
		log.Printf("Current pair:%v", orders[i].Pair)

		charts := historyConn.Charts[:]

		for j := 0; j < len(charts); j++ {
			if charts[j].Hm == orders[i].Trigger {

				nextDays := charts[i+1 : i+5]
				ratio := checkRatio(nextDays)
				log.Printf("Index:%v, Close:%v, Ratio:%v", i, charts[i].Close, ratio)
				if ratio > 0.01 {
					counter++
				}

			}
		}
	}

	log.Printf("Total:%v Up:%v", length, counter)

}
