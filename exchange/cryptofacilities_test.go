package exchange

import (
	"log"
	"testing"
)

func TestCFGetInstruments(t *testing.T) {
	handle := new(CryptoFacilities)
	result := handle.GetInstruments()

	log.Printf("Result:%v", result)
}

func TestCFGetOrderBook(t *testing.T) {
	handle := new(CryptoFacilities)
	result := handle.GetDepthValue("ETH/USD")

	log.Printf("Result:%v", result)
}

func TestCFGetBalance(t *testing.T) {
	handle := new(CryptoFacilities)
	handle.SetConfigure(Config{
		API:    "",
		Secret: "",
	})
	result := handle.GetBalance()

	log.Printf("===Result:%v", result)
}

func TestCFTrade(t *testing.T) {
	handle := new(CryptoFacilities)
	handle.SetConfigure(Config{
		API:    "",
		Secret: "",
	})
	result := handle.Trade(TradeConfig{
		Amount: 2,
		Price:  195,
		Pair:   "ETH/USD",
		Type:   TradeTypeSell,
	})

	log.Printf("===Result:%v", result)
}

func TestCFCancelOrder(t *testing.T) {
	handle := new(CryptoFacilities)
	handle.SetConfigure(Config{
		API:    "",
		Secret: "",
	})
	result := handle.CancelOrder(OrderInfo{
		OrderID: "b27a343d-e254-4fe4-8d4e-70d8a9558e2c",
	})

	log.Printf("===Result:%v", result)
}

func TestCFGetPositions(t *testing.T) {
	handle := new(CryptoFacilities)
	handle.SetConfigure(Config{
		API:    "",
		Secret: "",
	})
	result := handle.GetPositions("eth/usd")

	log.Printf("===Result:%v", result)
}
