package trend

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

func TestMapItemOperation(t *testing.T) {
	op := make(map[string]*int)
	log.Printf("Value:%v", op["test"])
	op["test"] = new(int)
	log.Printf("Value:%d", *op["test"])
	*op["test"]++
	log.Printf("Value:%d", *op["test"])

	op2 := make(map[string]int)
	log.Printf("Value:%d", op2["test"])
	op2["test"]++
	log.Printf("Value:%d", op2["test"])
}
