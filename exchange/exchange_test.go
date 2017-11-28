package exchange

import (
	"testing"
	"time"
	"log"
)

func _TestGetAveragePrice(t *testing.T) {
	values := []DepthPrice{
		{price: 155, qty:10},
		{price: 165, qty:10},
		{price: 155, qty:10},
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
	log.Printf("Value:%v", value)
}

func _TestOKEXContractTicker(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeContract)

	okex.StartContractTicker("ltc", "this_week", "ltc_contract")	

	counter := 3
	for {
		select{
		case <-time.After(1*time.Second):
			values := okex.GetTickerValue("ltc_contract")
			if values != nil {
				log.Printf("Value:%v", values)
			}
			if counter > 0{
				counter--
			}else{
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
		select{
		case <-time.After(1*time.Second):
			values := okex.GetTickerValue("btc_current")
			if values != nil {
				log.Printf("Value:%v", values)
			}
			if counter > 0{
				counter--
			}else{
				return
			}
		}
	}
}

func _TestTrade(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeContract)
	okex.Login()

	config := map[string]interface{} {
        "symbol": "ltc_usd",
        "contract_type": "this_week",
        "price": "80",
        "amount": "1",
        "type": "1",
        "match_price": "0",
        "lever_rate": "10",
	}

	err, message := okex.PlaceOrder(config)
	if err != nil {
		log.Printf("Error:%v", err)
		return
	}

	log.Printf("message:%v", message)

	select{
	case <- time.After(1 * time.Second):
		return
	}

}

func TestGetOrderInfo(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeContract)
	
	configs := map[string]interface{} {
		"symbol": "ltc_usd",
		"order_id": "-1",
		"contract_type": "this_week",
		"status": "2",
		"current_page": "1",
		"page_length": "1",
	}

	log.Printf("OrderInfo:%v", okex.GetOrderInfo(configs))	
}

// func TestGetTradesInfo(t *testing.T) {
// 	okex := new(OKExAPI)
// 	okex.Init(TradeTypeContract)
// 	log.Printf("UserInfo:%v", okex.GetTradesInfo())	
// }

func TestGetUserInfo(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeContract)
	log.Printf("UserInfo:%v", okex.GetUserInfo())	
}

func TestCancelOrder(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeContract)

	configs := map[string]interface{} {
		"order_id": "14318387904",
		"symbol": "ltc_usd",
        "contract_type": "this_week",
	}
	log.Printf("CancelOrder:%v", okex.CancelOrder(configs))
}

func _TestBittrexTicker(t *testing.T) {

	bittrex := new(BittrexAPI)
	bittrex.Init()
	bittrex.AddTicker("USDT", "BTC", "USDT-BTC")

	for {
		select{
		case <-time.After(3*time.Second):
			values := bittrex.GetTickerValue("USDT-BTC")
			if values != nil {
				log.Printf("Value:%v %v", values, values["Last"].(float64))
			}
		}
	}
}