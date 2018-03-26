package main

import (
	"log"
	"madaoQT/task/trend"
)

func main() {

	trendTask := new(trend.TrendTask)
	err := trendTask.Start("")
	if err != nil {
		log.Printf("Error:%v", err)
	}
	select {}
}
