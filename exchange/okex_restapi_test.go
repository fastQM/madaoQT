package exchange

import (
	"log"
	"testing"
)

func TestGetOkexRestAPIKline(t *testing.T) {

	okex := new(OkexRestAPI)
	okex.SetConfigure(Config{
	// Proxy: "SOCKS5:127.0.0.1:1080",
	})
	klines := okex.GetKline("eth/usdt", KlinePeriod2Hour, 600)

	if len(klines) != 0 {
		log.Printf("共有%d条", len(klines))

		ChangeOffset(0.382)
		result := StrategyTrendArea(klines, true, true)
		log.Printf("Result:%v", result)
	}

}
