package exchange

import (
	"log"
	"testing"
	"time"
)

func _TestGetAveragePrice(t *testing.T) {
	values := []DepthPrice{
		{price: 155, qty: 10},
		{price: 165, qty: 10},
		{price: 155, qty: 10},
	}

	value1, value2 := GetDepthPriceByOrder(0, values, 25)
	log.Printf("Ave:%v%v", value1, value2)
}

func _TestGetContractDepth(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeContract)

	value := okex.GetDepthValue("btc", "", 1)
	log.Printf("Value:%v", value)
}

func _TestGetCurrentDepth(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeCurrent)

	value := okex.GetDepthValue("btc", "usdt", 1)
	Logger.Infof("Value:%v", value)
}

func _TestOKEXContractTicker(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeContract)

	okex.StartContractTicker("ltc", "this_week", "ltc_contract")

	counter := 3
	for {
		select {
		case <-time.After(1 * time.Second):
			values := okex.GetTickerValue("ltc_contract")
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

func _TestOKEXCurrentTicker(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeCurrent)

	okex.StartCurrentTicker("btc", "usdt", "btc_current")

	counter := 3
	for {
		select {
		case <-time.After(1 * time.Second):
			values := okex.GetTickerValue("btc_current")
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

func TestTrade(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeContract)

	// config := map[string]interface{} {
	//     "symbol": "ltc_usd",
	//     "contract_type": "this_week",
	//     "price": "80",
	//     "amount": "1",
	//     "type": "1",
	//     "match_price": "0",
	//     "lever_rate": "10",
	// }
	configs := TradeConfig{
		Coin:   "ltc_usd",
		Type:   OrderTypeOpenLong,
		Price:  60.01,
		Amount: 1000,
	}

	result := okex.Trade(configs)
	Logger.Debugf("Result:%v", result)

}

// func TestGetOrderInfo(t *testing.T) {
// 	okex := new(OKExAPI)
// 	okex.Init(TradeTypeContract)

// 	configs := map[string]interface{} {
// 		"symbol": "ltc_usd",
// 		"order_id": "-1",
// 		"contract_type": "this_week",
// 		"status": "2",
// 		"current_page": "1",
// 		"page_length": "1",
// 	}

// 	log.Printf("OrderInfo:%v", okex.GetOrderInfo(configs))
// }

func TestGetUserInfo(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeContract)
	log.Printf("UserInfo:%v", okex.GetBalance("ltc"))
}

func _TestCancelOrder(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeContract)

	// configs := map[string]interface{} {
	// 	"order_id": "14318387904",
	// 	"symbol": "ltc_usd",
	//     "contract_type": "this_week",
	// }
	order := OrderInfo{
		OrderID: "14566361108",
		Coin:    "ltc_usd",
	}

	log.Printf("CancelOrder:%v", okex.CancelOrder(order))
}

func _TestSpotCancelOrder(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeCurrent)

	// configs := map[string]interface{} {
	// 	"order_id": "58520149",
	// 	"symbol": "ltc_usdt",
	// }
	order := OrderInfo{
		OrderID: "60461596",
		Coin:    "ltc_usdt",
	}

	log.Printf("CancelOrder:%v", okex.CancelOrder(order))
}

func TestSpotOrder(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeCurrent)

	// configs := map[string]interface{} {
	// 	"symbol":"ltc_usdt",
	//     "type":"buy",
	//     "price":"70",
	//     "amount":"1",
	// }

	configs := TradeConfig{
		Coin:   "ltc_usdt",
		Type:   OrderTypeBuy,
		Price:  60,
		Amount: 1000,
	}

	result := okex.Trade(configs)
	Logger.Debugf("Result:%v", result)
}

func TestSpotGetOrderInfo(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeCurrent)
	configs := map[string]interface{}{
		"order_id": "-1",
		"symbol":   "ltc_usdt",
	}

	log.Printf("OrderInfo:%v", okex.GetOrderInfo(configs))
}

func TestSpotGetUserInfo(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeCurrent)
	log.Printf("UserInfo:%v", okex.GetBalance("usdt"))
}

func TestGetOrderType(t *testing.T) {
	okex := new(OKExAPI)
	log.Printf("Type:%d", okex.getOrderType("buy"))
}

func TestGetOrderStatus(t *testing.T) {
	okex := new(OKExAPI)
	log.Printf("Type:%d", okex.getStatus(1))
}
