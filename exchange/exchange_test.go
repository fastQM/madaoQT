package exchange

import (
	"testing"
	"time"
	"log"
)

func TestGetAveragePrice(t *testing.T) {
	values := []DepthPrice{
		{price: 155, qty:10},
		{price: 165, qty:10},
		{price: 155, qty:10},
	}

	log.Printf("Ave:%v", GetDepthPriceByOrder(0, values, 25))
}

func _TestGetContractDepth(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeContract)

	value := okex.GetDepthValue("btc", "")
	log.Printf("Value:%v", value)

	// for{
	// 	select{
	// 		case event := <- okex.WatchEvent():
	// 			if event == EventConnected{
	// 				log.Printf("connected")
	// 				okex.GetContractDepth("btc", "this_week", "20")
	// 				// okex.StartContractTicker("btc", Y_THIS_WEEK, "test")
	// 			}else if event == EventError {
	// 				log.Printf("reconnnect")
	// 				okex.Init(TradeTypeContract)
	// 			}
	// 	}
	// }
}

func TestGetCurrentDepth(t *testing.T) {
	okex := new(OKExAPI)
	okex.Init(TradeTypeCurrent)

	value := okex.GetDepthValue("btc", "usdt")
	log.Printf("Value:%v", value)	

	// for{
	// 	select{
	// 		case event := <- okex.WatchEvent():
	// 			if event == EventConnected{
	// 				log.Printf("connected")
	// 				okex.GetCurrentDepth("btc_usdt", "5")
	// 				// okex.StartContractTicker("btc", Y_THIS_WEEK, "test")
	// 			}else if event == EventError {
	// 				log.Printf("reconnnect")
	// 				okex.Init(TradeTypeCurrent)
	// 			}
	// 	}
	// }
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