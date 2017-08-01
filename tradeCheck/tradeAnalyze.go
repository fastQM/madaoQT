package tradeCheck

import (
	"log"
	"madaoqt/redis"
)

type Analyzer struct {
	tokenName string
	values    []redis.ChartItem
}

func (a *Analyzer) Init(tokenName string, array []redis.ChartItem, period int) {
	a.tokenName = tokenName

	for i := 1; i < len(array); i++ {
		if array[i].Date-array[i-1].Date != 900 {
			log.Printf("数据有误,名称:%v, 索引:%v", tokenName, i)
			return
		}
	}

	a.values = array
	log.Printf("%v数据加载成功,共有记录%v条", tokenName, len(array))
}

func (a *Analyzer) checkRatio() {
	var lowest float64
	var highest float64

	for _, item := range a.values {
		if lowest == 0 || item.Low < lowest {
			lowest = item.Low
		}

		if highest == 0 || item.High > highest {
			highest = item.High
		}
	}

	log.Printf("Name:%v, High:%v, Low:%v, ratio:%v", a.tokenName, highest, lowest, (highest-lowest)/lowest)
}

func (a *Analyzer) Analyze() {
	counter := 0
	values := a.values

	a.checkRatio()

	for i := 5; i < len(values); i++ {
		// i := len(values) - 1 {// 最新数据
		if (values[i].Volume > 3*values[i-1].Volume) &&
			(values[i].Close < values[i].Open) &&
			((values[i].Close-values[i].Low)/values[i].Low > 0.01) {
			for j := 5; j >= 0; j-- {
				if values[i-j].Close < values[i-j].Open {
					counter++
				}
			}

			if counter >= 4 {
				log.Printf("Name:%v, Date:%v is triggered.", a.tokenName, values[i].Hm)

				var order redis.OrderItem
				order.Pair = a.tokenName
				order.BuyLimitHigh = 1
				order.BuyLimitLow = 2
				order.SellLimitHigh = 1
				order.SellLimitLow = 2
				order.Trigger = values[i].Hm

				redisConn := new(redis.RedisOrder)
				err := redisConn.SaveOrder("poloniex", order)
				if err != nil {
					log.Printf("Error when saving order:%v", err.Error())
				}
				return
			}

		}
	}
}
