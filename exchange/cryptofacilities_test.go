package exchange

import (
	"log"
	Mongo "madaoQT/mongo"
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

	mongo := &Mongo.ExchangeDB{
		Server: "mongodb://localhost:27017",
	}
	err, key := GetExchangeKey(mongo, NameCryptoFacilities, []byte(""), []byte(""))
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	handle := new(CryptoFacilities)
	handle.SetConfigure(Config{
		API:    key.API,
		Secret: key.Secret,
	})
	result := handle.GetBalance()

	log.Printf("===Result:%v", result)
}

func TestCFTrade(t *testing.T) {

	mongo := &Mongo.ExchangeDB{
		Server: "mongodb://localhost:27017",
	}
	err, key := GetExchangeKey(mongo, NameCryptoFacilities, []byte(""), []byte(""))
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	handle := new(CryptoFacilities)
	handle.SetConfigure(Config{
		API:    key.API,
		Secret: key.Secret,
	})
	result := handle.Trade(TradeConfig{
		Amount: 1,
		Price:  220,
		Pair:   "ETH/USD",
		Type:   TradeTypeBuy,
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

func TestCFGetOrder(t *testing.T) {
	mongo := &Mongo.ExchangeDB{
		Server: "mongodb://localhost:27017",
	}
	err, key := GetExchangeKey(mongo, NameCryptoFacilities, []byte(""), []byte(""))
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	handle := new(CryptoFacilities)
	handle.SetConfigure(Config{
		API:    key.API,
		Secret: key.Secret,
	})
	result := handle.GetOrderInfo(OrderInfo{
		OrderID: "b27a343d-e254-4fe4-8d4e-70d8a9558e2c",
	})

	log.Printf("===Result:%v", result)
}

func TestCFGetPositions(t *testing.T) {

	mongo := &Mongo.ExchangeDB{
		Server: "mongodb://localhost:27017",
	}
	err, key := GetExchangeKey(mongo, NameCryptoFacilities, []byte(""), []byte(""))
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	handle := new(CryptoFacilities)
	handle.SetConfigure(Config{
		API:    key.API,
		Secret: key.Secret,
	})
	result := handle.GetPositions("eth/usd")

	log.Printf("===Result:%v", result)
}
