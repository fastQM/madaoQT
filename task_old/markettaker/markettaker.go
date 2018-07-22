// package markettaker
package main

import (
	"time"

	"github.com/kataras/golog"

	Global "madaoQT/config"
	Exchange "madaoQT/exchange"
)

var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.SetTimeFormat(Global.TimeFormat)
	Logger.SetPrefix("[PARA]")
}

type MarketTaker struct {
	binance Exchange.IExchange
}

func (p *MarketTaker) Start() error {
	// 测试
	p.binance = new(Exchange.Binance)

	for {
		select {
		case <-time.After(1 * time.Second):
			p.Watch()
		}
	}
	return nil
}

func (p *MarketTaker) Watch() {

	depths := p.binance.GetDepthValue("ltc/usdt")
	// log.Printf("Depth:%v", depths)
	ask := depths[Exchange.DepthTypeAsks][0]
	bid := depths[Exchange.DepthTypeBids][0]

	diff := (ask.Price - bid.Price) * 100 / bid.Price
	if diff > 0.3 {
		Logger.Debugf("Diff:%v", diff)
	}

}

func main() {
	marketMaker := new(MarketTaker)
	marketMaker.Start()
}
