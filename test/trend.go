package main

import (
	"log"
	Mongo "madaoQT/mongo"
	"madaoQT/task/trend"
)

func main() {
	mongo := new(Mongo.ExchangeDB)
	if err := mongo.Connect(); err != nil {
		log.Printf("Error:%v", err)
		return
	}

	err, record := mongo.FindOne("OkexSpot")
	if err != nil || record == nil {
		log.Printf("Cannot load API/APKEY")
		return
	}

	trendTask := new(trend.TrendTask)
	trendTask.Start(record.API, record.Secret, "")
	select {}
}
