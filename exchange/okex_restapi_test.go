package exchange

import (
	"log"
	"testing"
)

func TestGetOkexRestAPIKline(t *testing.T) {

	okex := new(OkexRestAPI)
	okex.SetConfigure(Config{
		Proxy: "SOCKS5:127.0.0.1:1080",
	})
	result := okex.GetKline("eth/usdt", KlinePeriod2Hour, 1000)

	if len(result) != 0 {
		log.Printf("共有%d条", len(result))

		StrategyTrendTest(result, true, true)
	}

}
