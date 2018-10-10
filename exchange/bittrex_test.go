package exchange

import (
	"log"
	"testing"
)

const bittrexAPI = ""
const bittrexSecret = ""

func TestBittrexGetDepth(t *testing.T) {

	bittrex := new(Bittrex)
	result := bittrex.GetDepthValue("eth/usdt")
	log.Printf("result:%v", result)
}

func TestBittrexGetBalance(t *testing.T) {

	bittrex := new(Bittrex)
	bittrex.SetConfigure(Config{
		API:    bittrexAPI,
		Secret: bittrexSecret,
	})
	result := bittrex.GetBalance()
	log.Printf("result:%v", result)
}

func TestBittrexTrade(t *testing.T) {

	bittrex := new(Bittrex)
	bittrex.SetConfigure(Config{
		API:    bittrexAPI,
		Secret: bittrexSecret,
	})
	result := bittrex.Trade(TradeConfig{
		Pair:   "storj/btc",
		Price:  0.00004064,
		Amount: 100,
		Type:   TradeTypeSell,
	})
	log.Printf("result:%v", result)
}

func TestBittrexGetOrderInfo(t *testing.T) {

	bittrex := new(Bittrex)
	bittrex.SetConfigure(Config{
		API:    bittrexAPI,
		Secret: bittrexSecret,
	})
	result := bittrex.GetOrderInfo(OrderInfo{
		OrderID: "b9b92fdb-383a-486c-bf7b-d201206e04f0",
	})
	log.Printf("result:%v", result)
}
