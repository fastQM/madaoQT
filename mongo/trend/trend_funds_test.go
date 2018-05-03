package mongotrend

import (
	"log"
	"strings"
	"testing"
	"time"

	Exchange "madaoQT/exchange"
	Mongo "madaoQT/mongo"
	Task "madaoQT/task"
)

const MongoServer = "mongodb://54.212.224.28:28017"

const K1 = ""
const K2 = ""

func TestShowFunds(t *testing.T) {

	mongo := &Mongo.ExchangeDB{
		Server:     MongoServer,
		Sock5Proxy: "SOCKS5:127.0.0.1:1080",
	}
	err, key := Exchange.GetExchangeKey(mongo, Exchange.NameBinance, []byte(K1), []byte(K2))
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	binance := new(Exchange.Binance)
	binance.SetConfigure(Exchange.Config{
		API:    key.API,
		Secret: key.Secret,
		Proxy:  "SOCKS5:127.0.0.1:1080",
	})

	balances := binance.GetBalance()

	if balances == nil {
		log.Printf("Fail to get the balances")
		return
	}

	database := &TrendMongo{
		FundCollectionName: Task.TrendFundBinance,
		Server:             MongoServer,
		Sock5Proxy:         "SOCKS5:127.0.0.1:1080",
	}
	if err := database.Connect(); err != nil {
		log.Printf("Error:%v", err)
		return
	}

	defer database.Disconnect()

	coin := "usdt"
	balance := balances[strings.ToUpper(coin)].(float64)

	err, records := database.FundCollection.FindAll(map[string]interface{}{
		"name":   coin,
		"action": ActionAdd,
	})
	if err != nil {
		log.Printf("Error:%v", err)
		return
	}

	total := 0.0
	for _, record := range records {
		log.Printf("[%s]账户名:%s 申购价格:%.4f 申购日期:%v", record.Name, record.Owner, record.Price, record.Date)
		total += (record.Quantity * record.Price)
	}

	log.Printf("净值:%.4f", balance/total)

}

// 增加资金
func TestBinanceAddNewClient(t *testing.T) {
	mongo := &Mongo.ExchangeDB{
		Server:     MongoServer,
		Sock5Proxy: "SOCKS5:127.0.0.1:1080",
	}
	err, key := Exchange.GetExchangeKey(mongo, Exchange.NameBinance, []byte(K1), []byte(K2))
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	binance := new(Exchange.Binance)
	binance.SetConfigure(Exchange.Config{
		API:    key.API,
		Secret: key.Secret,
		Proxy:  "SOCKS5:127.0.0.1:1080",
	})

	balances := binance.GetBalance()

	if balances == nil {
		log.Printf("Fail to get the balances")
		return
	}

	database := &TrendMongo{
		FundCollectionName: Task.TrendFundBinance,
		Server:             MongoServer,
		Sock5Proxy:         "SOCKS5:127.0.0.1:1080",
	}
	if err := database.Connect(); err != nil {
		log.Printf("Error:%v", err)
		return
	}

	defer database.Disconnect()

	// config
	username := "me"
	coin := "usdt"
	balance := balances[strings.ToUpper(coin)].(float64)

	err, records := database.FundCollection.FindAll(map[string]interface{}{
		"name":   coin,
		"action": ActionAdd,
	})
	if err != nil {
		log.Printf("Error:%v", err)
		return
	}

	total := 0.0
	for _, record := range records {
		total += (record.Price + record.Quantity)
	}

	// !!!以下操作需保证资金在操作后打进账户

	added := 1000.0
	price := balance / total
	quantity := added / price
	log.Printf("当前净值:%.4f 申购数量:%.4f", price, quantity)

	if err := database.FundCollection.Insert(&FundInfo{
		Name:     coin,
		Action:   ActionAdd,
		Quantity: 1389.08952,
		Owner:    username,
		Price:    1.0,
		Date:     time.Now(),
	}); err != nil {
		log.Printf("Error:%v", err)
		return
	}

	log.Printf("添加资金成功")
}

func TestBinanceRemoveFund(t *testing.T) {
	mongo := &Mongo.ExchangeDB{
		Server:     MongoServer,
		Sock5Proxy: "SOCKS5:127.0.0.1:1080",
	}
	err, key := Exchange.GetExchangeKey(mongo, Exchange.NameBinance, []byte(K1), []byte(K2))
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	binance := new(Exchange.Binance)
	binance.SetConfigure(Exchange.Config{
		API:    key.API,
		Secret: key.Secret,
		Proxy:  "SOCKS5:127.0.0.1:1080",
	})

	balances := binance.GetBalance()

	if balances == nil {
		log.Printf("Fail to get the balances")
		return
	}

	database := &TrendMongo{
		FundCollectionName: Task.TrendFundBinance,
		Server:             MongoServer,
		Sock5Proxy:         "SOCKS5:127.0.0.1:1080",
	}
	if err := database.Connect(); err != nil {
		log.Printf("Error:%v", err)
		return
	}

	defer database.Disconnect()

	coin := "usdt"
	// balance := balances[strings.ToUpper(coin)].(float64)

	err, _ = database.FundCollection.FindAll(map[string]interface{}{
		"name":   coin,
		"action": ActionAdd,
	})
	if err != nil {
		log.Printf("Error:%v", err)
		return
	}

	// add after
	// total := 0.0
	// for _, record := range records {
	// 	total += record.Quantity
	// }

	// 以下操作需保证资金在操作后打进账户

	added := 1000.0

	if err := database.FundCollection.Insert(&FundInfo{
		Name:     coin,
		Action:   ActionAdd,
		Quantity: added,
		Owner:    "me",
	}); err != nil {
		log.Printf("Error:%v", err)
		return
	}

}

func TestOkexAddFund2(t *testing.T) {
	coin := "eth"

	database := &TrendMongo{
		FundCollectionName: Task.TrendFundOKEX,
		Server:             MongoServer,
		Sock5Proxy:         "SOCKS5:127.0.0.1:1080",
	}
	if err := database.Connect(); err != nil {
		log.Printf("Error:%v", err)
		return
	}

	defer database.Disconnect()

	if err := database.FundCollection.Insert(&FundInfo{
		Name:     coin,
		Action:   ActionAdd,
		Quantity: 16.25005514,
		Owner:    "me",
	}); err != nil {
		log.Printf("Error:%v", err)
		return
	}
}

func TestOkexRemoveFund(t *testing.T) {

}
