package mongo

import (
	"log"
	"testing"
	"time"
)

func _TestLoadingCharts(t *testing.T) {
	mongo := new(Charts)
	err := mongo.Connect()
	if err == nil {
		mongo.LoadCharts("Poloniex", "USDT-ETH", 15)
	}
}

func _TestInsertTradeRecord(t *testing.T) {
	tradesDB := new(Trades)
	err := tradesDB.Connect()
	if err == nil {
		record := &TradesRecord{
			Time:     time.Now(),
			Oper:     "buy",
			Exchange: "okex",
			Coin:     "btc",
			Quantity: 123.45,
		}
		tradesDB.Insert(record)
	}

	err, records := tradesDB.FindAll()
	if err != nil {
		log.Printf("Error:%v", err)
		return
	}

	log.Printf("Records:%v", records)
}
