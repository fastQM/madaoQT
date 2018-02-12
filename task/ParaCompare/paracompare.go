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

	exchanges []Exchange.IExchange
}

func (p *ParaCompare) Start() error {
	// 测试
	p.huobi = new(Exchange.Huobi)
	p.binance = new(Exchange.Binance)

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

			} else if event == Exchange.EventLostConnection {
				go Task.Reconnect(spotExchange)
			}
		case <-time.After(3 * time.Second):
			p.Watch()
		}
	}

	return nil

}

func (p *ParaCompare) Watch() {

	pair := "eth/usdt"
	uintAmount := float64(100)

	err2, askOkex, _, bidOkex, _ := Task.CalcDepthPrice(false, map[string]float64{}, p.okex, pair, uintAmount)
	err3, askHuobi, _, bidHuobi, _ := Task.CalcDepthPrice(false, map[string]float64{}, p.huobi, pair, uintAmount)
	err4, askBinance, _, bidBinance, _ := Task.CalcDepthPrice(false, map[string]float64{}, p.binance, pair, uintAmount)
	if err2 == nil && err3 == nil && err4 == nil {
		Logger.Infof("币种:%s, OKEX可买入价格：%.2f, OKEX可卖出价格：%.2f", pair, askOkex, bidOkex)
		Logger.Infof("币种:%s, 火币可买入价格：%.2f, 火币可卖出价格：%.2f", pair, askHuobi, bidHuobi)
		Logger.Infof("币种:%s, 币安可买入价格：%.2f, 币安可卖出价格：%.2f", pair, askBinance, bidBinance)
		maxBid := GetMax(bidOkex, bidHuobi, bidBinance) // 可以卖出
		minAsk := GetMin(askOkex, askHuobi, askBinance) // 可以买入

		Logger.Infof("最高卖出价格:%.2f 最低买入价格:%.2f 最大利差:%.2f%%", maxBid, minAsk, (maxBid-minAsk)*100/minAsk)
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
