package exchange

import (
	"log"
	Mongo "madaoQT/mongo"
	"testing"
	"time"
)

// 阈值：2018/07/28 20:46:57 Result:盈利次数：24 亏损次数 ：17 盈利求和：125.697068 亏损求和 ：-54.244807 净值 ：1.826487 阈值比例:40
// 反转：2018/07/28 18:00:47 Result:盈利次数：30 亏损次数 ：30 盈利求和：181.087415 亏损求和 ：-70.059811 净值 ：2.590557 阈值比例:0.3800
// 开盘突破：2018/07/28 18:02:27 Result:盈利次数：30 亏损次数 ：39 盈利求和：179.961559 亏损求和 ：-84.617132 净值 ：2.201622 阈值比例:0.3800
// 低点突破:2018/07/28 18:01:33 Result:盈利次数：36 亏损次数 ：42 盈利求和：176.474276 亏损求和 ：-67.213361 净值 ：2.540028 阈值比例:0.3800
func TestGetOkexRestAPIKline(t *testing.T) {

	okex := new(OkexRestAPI)
	okex.SetConfigure(Config{
		Proxy: "SOCKS5:127.0.0.1:1080",
	})

	var klines []KlineValue
	// file := "okex-ethusdt-1h-tmp"
	file := "okex-ethusdt-1h"
	// file := "okex-btcusdt-2h"
	if false {
		klines = okex.GetKline("btc/usdt", KlinePeriod2Hour, 1400)
		SaveHistory(file, klines)
	} else {
		klines = LoadHistory(file)
	}

	for _, kline := range klines {
		log.Printf("Time:%v %v", time.Unix(int64(kline.OpenTime), 0).String(), kline)
	}

	log.Printf("%v", time.Now())

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
