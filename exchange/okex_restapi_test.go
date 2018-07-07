package exchange

import (
	"log"
	Mongo "madaoQT/mongo"
	"testing"
	"time"
)

func TestGetOkexRestAPIKline(t *testing.T) {

	okex := new(OkexRestAPI)
	okex.SetConfigure(Config{
		// Proxy: "SOCKS5:127.0.0.1:1080",
	})
	klines := okex.GetKline("eth/usdt", KlinePeriod2Hour, 100)
	for _, kline := range klines {
		log.Printf("%.2f Time:%v", kline.OpenTime, time.Unix(int64(kline.OpenTime), 0).String())
	}

	log.Printf("%v", time.Now())

	if len(klines) != 0 {
		log.Printf("共有%d条", len(klines))

		ChangeOffset(0.382)
		result := StrategyTrendArea(klines, true, true)
		log.Printf("Result:%v", result)
	}

}

func TestGetPosition(t *testing.T) {

	mongo := &Mongo.ExchangeDB{
		Server: "mongodb://localhost:27017",
	}
	err, key := GetExchangeKey(mongo, NameOKEX, []byte(""), []byte(""))
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	okex := new(OkexRestAPI)
	okex.SetConfigure(Config{
		API:    key.API,
		Secret: key.Secret,
	})

	okex.Start()

	log.Printf("Balance:%v", okex.GetPosition("eth/usd", "quarter"))
}
