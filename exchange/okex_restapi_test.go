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
		Proxy: "SOCKS5:127.0.0.1:1080",
	})

	var klines []KlineValue
	file := "okex-ethusdt-2h-tmp"
	// file := "okex-ethusdt-1h"
	// file := "okex-btcusdt-2h"
	if true {
		klines = okex.GetKline("eth/usdt", KlinePeriod2Hour, 10000)
		SaveHistory(file, klines)
	} else {
		klines = LoadHistory(file)
	}

	for _, kline := range klines {
		log.Printf("Hour Time:%v %v", time.Unix(int64(kline.OpenTime), 0).String(), kline)
	}

	// klines = Swith1HourToDialyKlines(klines)
	// klines = Swith1HourToHoursKlines(12, klines)

	// for _, kline := range klines {
	// 	log.Printf("Day Time:%v %v", time.Unix(int64(kline.OpenTime), 0).String(), kline)
	// }
	// return

	var results []string
	if len(klines) != 0 {
		log.Printf("共有%d条", len(klines))

		// for i := 0.01; i < 0.2; i += 0.01 {
		// for i := 1; i < 45; i++ {
		// ChangeOffset(i)
		// ChangeInterval(i)
		// ChangeLoss(i)
		result := StrategyTrendArea(klines, true, true)
		results = append(results, result)
		// }

		// for i := 0.1; i < 0.8; i += 0.01 {
		// 	SpliteSetWaveLimit(i)
		// result := CTPStrategyTrendSplit(klines, true, true, true)
		// results = append(results, result)
		// }

		for _, result := range results {
			log.Printf("Result:%s", result)
		}
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

func TestOKEXRestTrade(t *testing.T) {

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

	log.Printf("Result:%v", okex.Trade(TradeConfig{
		Type:   TradeTypeCloseLong,
		Amount: 1,
		Price:  300,
		Pair:   "eth/usdt",
	}))
}

func TestOKEXRestGetOrderInfo(t *testing.T) {

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

	//1208688209050624
	log.Printf("Result:%v", okex.GetOrderInfo(OrderInfo{
		Pair:    "eth/usdt",
		OrderID: "1208772889369600",
	}))
}

func TestOKEXRestGetBalance(t *testing.T) {

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

	//1208688209050624
	log.Printf("Result:%v", okex.GetBalance())
}
