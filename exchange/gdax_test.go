package exchange

import (
	"log"
	"testing"
)

func TestGdaxGetKline(t *testing.T) {

	// date1 := time.Date(2018, 1, 10, 0, 0, 0, 0, time.Local)
	// date2 := time.Date(2018, 4, 1, 0, 0, 0, 0, time.Local)

	gdax := new(ExchangeGdax)
	gdax.SetConfigure(Config{
		Proxy: "SOCKS5:127.0.0.1:1080",
	})

	result := gdax.GetKline("btc/usdt", Period15Min, 0)
	log.Printf("Length:%d", len(result))
	StrategyTrendTest(result, true, true)
	// count := 13

	// for {
	// 	select {
	// 	case <-time.After(2 * time.Second):
	// 		log.Printf("Last:%v %v", time.Unix(int64(result[len(result)-1].OpenTime), 0), result[len(result)-1])
	// 		// StrategyTrendTest(result, true, true)
	// 		if count > 0 {
	// 			count--
	// 		} else {
	// 			return
	// 		}
	// 	}
	// }

}

func TestGdaxGetTicker(t *testing.T) {

	// date1 := time.Date(2018, 1, 10, 0, 0, 0, 0, time.Local)
	// date2 := time.Date(2018, 4, 1, 0, 0, 0, 0, time.Local)

	gdax := new(ExchangeGdax)
	gdax.SetConfigure(Config{
		Proxy: "SOCKS5:127.0.0.1:1080",
	})

	log.Printf("Ticker:%v", gdax.GetTicker("eth/usdt"))
}
