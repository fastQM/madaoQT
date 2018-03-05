package main

import (
	"log"
	MongoTrend "madaoQT/mongo/trend"
	"testing"
	"time"
)

func TestCheckProfit(t *testing.T) {
	db := new(MongoTrend.TrendMongo)
	if err := db.Connect(); err != nil {
		log.Printf("Error:%v", err)
		return
	}

	fundManager := new(FundManager)
	fundManager.Init(db.FundCollection)

	date := time.Date(2018, 3, 1, 0, 0, 0, 0, time.Local)
	today := time.Now()

	for {
		log.Printf("Date:%v", date.Format("2006-01-02"))
		fundManager.CheckDailyProfit(date)
		if date.AddDate(0, 0, 1).After(today) {
			break
		} else {
			date = date.AddDate(0, 0, 1)
		}
	}
}
