package exchange

import (
	"log"
	Mongo "madaoQT/mongo"
	"testing"
	"time"
)

const constAPIKey = "a982120e-8505-41db-9ae3-0c62dd27435c"
const constSecretKey = "71430C7FA63A067724FB622FB3031970"

func TestGetFutureDepth(t *testing.T) {
	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		// API:    constAPIKey,
		// Secret: constSecretKey,
		Custom: map[string]interface{}{
			"exchangeType": ExchangeTypeFuture,
			"period":       "this_week",
		},
	})

	okex.Start()

	counter := 10

	for {
		select {
		case <-time.After(1 * time.Second):
			value := okex.GetDepthValue("eth/usdt")
			log.Printf("Value:%v", value)
			if counter > 0 {
				counter--
			} else {
				return
			}
		}
	}

}

func TestGetSpotDepth(t *testing.T) {
	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		// API:    constAPIKey,
		// Secret: constSecretKey,
		Custom: map[string]interface{}{"exchangeType": ExchangeTypeSpot},
	})

	okex.Start()

	counter := 10

	for {
		select {
		case <-time.After(1 * time.Second):
			value := okex.GetDepthValue("eth/usdt")
			log.Printf("Value:%v", value)
			if counter > 0 {
				counter--
			} else {
				return
			}
		}
	}
}

func TestOKEXContractTicker(t *testing.T) {
	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		Custom: map[string]interface{}{
			"exchangeType": ExchangeTypeFuture,
			"period":       "this_week",
		},
	})

	okex.Start()

	okex.StartTicker("eth/usdt")

	counter := 10
	for {
		select {
		case <-time.After(1 * time.Second):
			values := okex.GetTicker("eth/usdt")
			if values != nil {
				log.Printf("Value:%v", values)
			}
			if counter > 0 {
				counter--
			} else {
				return
			}
		}
	}
}

func TestOKEXCurrentTicker(t *testing.T) {
	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		Custom: map[string]interface{}{"exchangeType": ExchangeTypeSpot},
	})

	okex.Start()
	okex.StartTicker("ltc/usdt")

	counter := 3
	for {
		select {
		case <-time.After(1 * time.Second):
			values := okex.GetTicker("ltc/usdt")
			if values != nil {
				log.Printf("Value:%v", values)
			}
			if counter > 0 {
				counter--
			} else {
				return
			}
		}
	}
}

func TestFutureTrade(t *testing.T) {
	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		API:    constAPIKey,
		Secret: constSecretKey,
		Custom: map[string]interface{}{
			"exchangeType": ExchangeTypeFuture,
			"period":       "this_week",
		},
	})

	okex.Start()

	configs := TradeConfig{
		Pair:   "eth/usd",
		Type:   TradeTypeOpenLong,
		Price:  1177,
		Amount: 1,
	}

	result := okex.Trade(configs)
	logger.Debugf("Result:%v", result)

}

func TestGetUserInfo(t *testing.T) {
	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		API:    constAPIKey,
		Secret: constSecretKey,
		Custom: map[string]interface{}{
			"exchangeType": ExchangeTypeFuture,
			"period":       "this_week",
		},
	})
	okex.Start()
	log.Printf("balance:%v", okex.GetBalance())
}

func TestCancelFutureOrder(t *testing.T) {
	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		API:    constAPIKey,
		Secret: constSecretKey,
		Custom: map[string]interface{}{
			"exchangeType": ExchangeTypeFuture,
			"period":       "this_week",
		},
	})
	okex.Start()

	order := OrderInfo{
		OrderID: "19124409771",
		Pair:    "eth/usd",
	}

	log.Printf("CancelOrder:%v", okex.CancelOrder(order))
}

func _TestSpotCancelOrder(t *testing.T) {
	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		API:    constAPIKey,
		Secret: constSecretKey,
		Custom: map[string]interface{}{"exchangeType": ExchangeTypeSpot},
	})

	order := OrderInfo{
		OrderID: "64274385",
		Pair:    "ltc_usdt",
	}

	log.Printf("CancelOrder:%v", okex.CancelOrder(order))
}

func TestSpotOrder(t *testing.T) {
	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		API:    constAPIKey,
		Secret: constSecretKey,
		Custom: map[string]interface{}{"exchangeType": ExchangeTypeSpot},
	})

	okex.Start()

	configs := TradeConfig{
		Pair:   "eth/usdt",
		Type:   TradeTypeBuy,
		Price:  1155,
		Amount: 0.011,
	}

	result := okex.Trade(configs)
	logger.Debugf("Result:%v", result)
}

func TestSpotGetOrderInfo(t *testing.T) {
	okex := new(OKExAPI)

	okex.SetConfigure(Config{
		API:    constAPIKey,
		Secret: constSecretKey,
		Custom: map[string]interface{}{"exchangeType": ExchangeTypeSpot},
	})
	okex.Start()

	log.Printf("OrderInfo:%v", okex.GetOrderInfo(OrderInfo{
		OrderID: "79863957",
		Pair:    "eth/usdt",
	}))
}

func TestFutureGetOrderInfo(t *testing.T) {
	okex := new(OKExAPI)

	okex.SetConfigure(Config{
		API:    constAPIKey,
		Secret: constSecretKey,
		Custom: map[string]interface{}{
			"exchangeType": ExchangeTypeFuture,
			"period":       "this_week",
		},
	})

	okex.Start()

	log.Printf("OrderInfo:%v", okex.GetOrderInfo(OrderInfo{
		OrderID: "21144549502",
		Pair:    "eth/usd",
	}))
}

func TestSpotGetUserInfo(t *testing.T) {

	mongo := &Mongo.ExchangeDB{
		Server: "mongodb://localhost:27017",
	}
	err, key := GetExchangeKey(mongo, NameOKEX, []byte(""), []byte(""))
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		API:    key.API,
		Secret: key.Secret,
		Custom: map[string]interface{}{"exchangeType": ExchangeTypeFuture},
	})

	okex.Start()

	log.Printf("Balance:%v", okex.GetBalance())
}

func _TestGetTradeType(t *testing.T) {
	log.Printf("Type:%d", OkexGetTradeTypeByString("buy"))
}

func _TestGetOrderStatus(t *testing.T) {
	log.Printf("Type:%d", OkexGetTradeStatus(1))
}

func TestGetFutureKline(t *testing.T) {
	futureExchange := new(OKExAPI)
	futureExchange.SetConfigure(Config{
		Custom: map[string]interface{}{
			"exchangeType": ExchangeTypeFuture,
			"period":       "quarter",
		},
		Proxy: "SOCKS5:127.0.0.1:1080",
	})

	futureExchange.Start()
	futureExchange.SubKlines("eth/usd", KlinePeriod5Min, 100)

	select {}
}
