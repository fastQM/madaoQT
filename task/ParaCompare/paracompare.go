package main

import (
	"time"

	"github.com/kataras/golog"

	Global "madaoQT/config"
	Exchange "madaoQT/exchange"
	Task "madaoQT/task"
)

var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.SetTimeFormat(Global.TimeFormat)
	Logger.SetPrefix("[PARA]")
}

type ParaCompare struct {
	huobi   Exchange.IExchange
	binance Exchange.IExchange
	okex    Exchange.IExchange
	liqui   Exchange.IExchange
	bittrex Exchange.IExchange

	exchanges map[string]Exchange.IExchange
}

func (p *ParaCompare) Start() error {
	// 测试
	p.huobi = new(Exchange.Huobi)
	p.binance = new(Exchange.Binance)
	p.liqui = new(Exchange.Liqui)
	p.bittrex = new(Exchange.Bittrex)

	p.exchanges = make(map[string]Exchange.IExchange)
	p.exchanges[p.huobi.GetExchangeName()] = p.huobi
	p.exchanges[p.binance.GetExchangeName()] = p.binance
	p.exchanges[p.liqui.GetExchangeName()] = p.liqui
	// p.exchanges[p.bittrex.GetExchangeName()] = p.bittrex

	spotExchange := Exchange.NewOKExSpotApi(&Exchange.Config{
	// API:    api,
	// Secret: secret,
	})

	if err := spotExchange.Start(); err != nil {
		Logger.Errorf("Fail to start:%v", err)
		return err
	}

	for {
		select {
		case event := <-spotExchange.WatchEvent():
			if event == Exchange.EventConnected {
				// for k := range a.config.Area {
				// 	pair := (k + "/usdt")
				// 	spotExchange.GetDepthValue(pair)
				// }

				p.okex = Exchange.IExchange(spotExchange)
				p.exchanges[p.okex.GetExchangeName()] = p.okex

			} else if event == Exchange.EventLostConnection {

				go Task.Reconnect(spotExchange)
			}
		case <-time.After(10 * time.Second):
			p.Watch()
		}
	}

	return nil

}

func (p *ParaCompare) Watch() {

	pairs := []string{"eth/usdt", "btc/usdt", "ltc/usdt"}
	for _, pair := range pairs {

		uintAmount := float64(100)
		// err, ask, _, bid, _ := Task.CalcDepthPrice(false, map[string]float64{}, p.bittrex, pair, uintAmount)

		// if err == nil {
		// 	Logger.Infof("币种[%s][%s]可买入价格：%.2f, 可卖出价格：%.2f", pair, p.bittrex.GetExchangeName(), ask, bid)
		// }
		var askList []float64
		var bidList []float64
		for _, exchange := range p.exchanges {
			err, ask, _, bid, _ := Task.CalcDepthPrice(false, map[string]float64{}, exchange, pair, uintAmount)
			if err == nil {
				askList = append(askList, ask)
				bidList = append(bidList, bid)
				// Logger.Infof("币种[%s][%s]可买入价格：%.2f, 可卖出价格：%.2f", pair, exchange.GetExchangeName(), ask, bid)
			} else {
				Logger.Infof("[%s]获取深度失败", exchange.GetExchangeName())
			}
		}

		maxBid := GetMax(bidList...) // 可以卖出
		minAsk := GetMin(askList...) // 可以买入
		Logger.Infof("[%s]最高卖出价格:%.2f 最低买入价格:%.2f 最大利差:%.2f%%", pair, maxBid, minAsk, (maxBid-minAsk)*100/minAsk)
	}

}

func GetMin(list ...float64) float64 {
	var min float64
	for index, item := range list {
		if index == 0 {
			min = item
		} else {
			if item < min {
				min = item
			}
		}
	}

	return min
}

func GetMax(list ...float64) float64 {
	var max float64
	for index, item := range list {
		if index == 0 {
			max = item
		} else {
			if item > max {
				max = item
			}
		}
	}

	return max
}

func main() {
	compare := new(ParaCompare)
	compare.Start()
}
