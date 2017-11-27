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

func TestGetContractDepth(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeContract)

	value := okex.GetDepthValue("btc", "", 1)
	log.Printf("Value:%v", value)
}

func TestGetCurrentDepth(t *testing.T) {
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
				log.Printf("Value:%v %v", values)
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
				log.Printf("Value:%v %v", values)
			}
			if counter > 0{
				counter--
			}else{
				return
			}
		}
	}
}

func _TestGetUserInfo(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeContract)

	log.Printf("UserInfo:%v", okex.GetUserInfo())
}

func _TestGetTrades(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeContract)

	log.Printf("TradesInfo:%v", okex.GetTradesInfo())
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