package exchange

import (
	"log"
	Mongo "madaoQT/mongo"
	"testing"
)

func TestOffset(t *testing.T) {

	mongo := &Mongo.ExchangeDB{
		Server:     "mongodb://54.212.224.28:28017",
		Sock5Proxy: "SOCKS5:127.0.0.1:1080",
	}
	err, key := GetExchangeKey(mongo, NameBitmex, []byte(""), []byte(""))
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	bitmex := new(ExchangeBitmex)
	bitmex.SetConfigure(Config{
		API:    key.API,
		Secret: key.Secret,
		Proxy:  "SOCKS5:127.0.0.1:1080",
	})

	err, result := bitmex.GetComposite(".BXBT", 50)
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	log.Printf("Result:%v", result)

}

func TestSign(t *testing.T) {
	bitmex := new(ExchangeBitmex)

	result := bitmex.sign("POST", "/api/v1/order", "1518064238", "{\"symbol\":\"XBTM15\",\"price\":219.0,\"clOrdID\":\"mm_bitmex_1a/oemUeQ4CAJZgP3fjHsA\",\"orderQty\":98}")

	log.Printf("SIGN:%s", result)
}

func TestBitmexGetBalance(t *testing.T) {

	mongo := &Mongo.ExchangeDB{
		Server:     "mongodb://54.212.224.28:28017",
		Sock5Proxy: "SOCKS5:127.0.0.1:1080",
	}
	err, key := GetExchangeKey(mongo, NameBitmex, []byte(""), []byte(""))
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	bitmex := new(ExchangeBitmex)
	bitmex.SetConfigure(Config{
		API:    key.API,
		Secret: key.Secret,
		Proxy:  "SOCKS5:127.0.0.1:1080",
	})

	log.Printf("Balance：%v", bitmex.GetBalance())
}

func TestBitmexTrade(t *testing.T) {

	mongo := &Mongo.ExchangeDB{
		Server:     "mongodb://54.212.224.28:28017",
		Sock5Proxy: "SOCKS5:127.0.0.1:1080",
	}
	err, key := GetExchangeKey(mongo, NameBitmex, []byte(""), []byte(""))
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	bitmex := new(ExchangeBitmex)
	bitmex.SetConfigure(Config{
		API:    key.API,
		Secret: key.Secret,
		Proxy:  "SOCKS5:127.0.0.1:1080",
	})

	result := bitmex.Trade(TradeConfig{
		Pair:   "XBTUSD",
		Type:   TradeTypeBuy,
		Amount: 50,
		Price:  8880.72})

	log.Printf("Result:%v", result)
}

func TestBitmexGetDepth(t *testing.T) {
	mongo := &Mongo.ExchangeDB{
		Server:     "mongodb://54.212.224.28:28017",
		Sock5Proxy: "SOCKS5:127.0.0.1:1080",
	}
	err, key := GetExchangeKey(mongo, NameBitmex, []byte(""), []byte(""))
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	bitmex := new(ExchangeBitmex)
	bitmex.SetConfigure(Config{
		API:    key.API,
		Secret: key.Secret,
		Proxy:  "SOCKS5:127.0.0.1:1080",
	})

	log.Printf("Depth:%v", bitmex.GetDepthValue(""))
}
