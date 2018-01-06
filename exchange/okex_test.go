package exchange

import (
	"log"
	"testing"
	"time"
)

const constAPIKey = "a982120e-8505-41db-9ae3-0c62dd27435c"
const constSecretKey = "71430C7FA63A067724FB622FB3031970"

func _TestGetAveragePrice(t *testing.T) {
	values := []DepthPrice{
		{price: 155, qty: 10},
		{price: 165, qty: 10},
		{price: 155, qty: 10},
	}

	value1, value2 := GetDepthPriceByOrder(values, 25)
	log.Printf("Ave:%v%v", value1, value2)
}

func TestGetContractDepth(t *testing.T) {
	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		// API:    constAPIKey,
		// Secret: constSecretKey,
		Custom: map[string]interface{}{"exchangeType": ExchangeTypeFuture},
	})

	okex.Start()
	value := okex.GetDepthValue("eth/usdt", 884, 0.005, 3, TradeTypeOpenLong)
	log.Printf("Value:%v", value)
}

func TestGetCurrentDepth(t *testing.T) {
	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		// API:    constAPIKey,
		// Secret: constSecretKey,
		Custom: map[string]interface{}{"exchangeType": ExchangeTypeSpot},
	})

	okex.Start()

	value := okex.GetDepthValue("eth/usdt", 877, 0.005, 3, TradeTypeBuy)
	logger.Infof("Value:%v", value)
}

func TestOKEXContractTicker(t *testing.T) {
	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		// API:    constAPIKey,
		// Secret: constSecretKey,
		Custom: map[string]interface{}{"exchangeType": ExchangeTypeFuture},
	})

	okex.Start()

	okex.StartTicker("ltc/usdt", map[string]interface{}{
		"period": "this_week",
	})

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

func TestOKEXCurrentTicker(t *testing.T) {
	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		// API:    constAPIKey,
		// Secret: constSecretKey,
		Custom: map[string]interface{}{"exchangeType": ExchangeTypeSpot},
	})

	okex.Start()
	okex.StartTicker("ltc/usdt", nil)

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

func _TestFutureTrade(t *testing.T) {
	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		API:    constAPIKey,
		Secret: constSecretKey,
		Custom: map[string]interface{}{"exchangeType": ExchangeTypeFuture},
	})

	configs := TradeConfig{
		Pair:   "ltc/usd",
		Type:   TradeTypeOpenLong,
		Price:  60.01,
		Amount: 1,
	}

	result := okex.Trade(configs)
	logger.Debugf("Result:%v", result)

}

func _TestGetUserInfo(t *testing.T) {
	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		API:    constAPIKey,
		Secret: constSecretKey,
		Custom: map[string]interface{}{"exchangeType": ExchangeTypeFuture},
	})
	okex.Start()
	log.Printf("balance:%v", okex.GetBalance())
}

func _TestCancelFutureOrder(t *testing.T) {
	okex := new(OKExAPI)

	okex.SetConfigure(Config{
		API:    constAPIKey,
		Secret: constSecretKey,
		Custom: map[string]interface{}{"exchangeType": ExchangeTypeFuture},
	})

	order := OrderInfo{
		OrderID: "14922014209",
		Pair:    "ltc_usd",
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

func _TestSpotOrder(t *testing.T) {
	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		API:    constAPIKey,
		Secret: constSecretKey,
		Custom: map[string]interface{}{"exchangeType": ExchangeTypeSpot},
	})

	configs := TradeConfig{
		Pair:   "ltc/usdt",
		Type:   TradeTypeBuy,
		Price:  60,
		Amount: 1,
	}

	result := okex.Trade(configs)
	logger.Debugf("Result:%v", result)
}

func _TestSpotGetOrderInfo(t *testing.T) {
	okex := new(OKExAPI)

	okex.SetConfigure(Config{
		API:    constAPIKey,
		Secret: constSecretKey,
		Custom: map[string]interface{}{"exchangeType": ExchangeTypeSpot},
	})

	// configs := map[string]interface{}{
	// 	"order_id": "-1",
	// 	"symbol":   "ltc_usdt",
	// }

	log.Printf("OrderInfo:%v", okex.GetOrderInfo(OrderInfo{
		OrderID: "-1",
		Pair:    "ltc/usdt",
	}))
}

func _TestFutureGetOrderInfo(t *testing.T) {
	okex := new(OKExAPI)

	okex.SetConfigure(Config{
		API:    constAPIKey,
		Secret: constSecretKey,
		Custom: map[string]interface{}{"exchangeType": ExchangeTypeFuture},
	})

	okex.Start()

	// configs := map[string]interface{}{
	// 	"order_id": "15188890865",
	// 	"symbol":   "ltc_usd",
	// }

	log.Printf("OrderInfo:%v", okex.GetOrderInfo(OrderInfo{
		OrderID: "-1",
		Pair:    "ltc/usd",
	}))
}

func TestSpotGetUserInfo(t *testing.T) {
	okex := new(OKExAPI)
	okex.SetConfigure(Config{
		API:    constAPIKey,
		Secret: constSecretKey,
		Custom: map[string]interface{}{"exchangeType": ExchangeTypeSpot},
	})

	okex.Start()

	log.Printf("Balance:%v", okex.GetBalance())
}

func _TestGetTradeType(t *testing.T) {
	okex := new(OKExAPI)
	log.Printf("Type:%d", okex.getTradeTypeByString("buy"))
}

func _TestGetOrderStatus(t *testing.T) {
	okex := new(OKExAPI)
	log.Printf("Type:%d", okex.getStatus(1))
}
