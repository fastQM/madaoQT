package main

import (
	"flag"
	"log"
)

func main() {

	update := flag.Bool("update", true, "update klines")
	test := flag.Bool("test", false, "test mode")
	flag.Parse()

	log.Printf("Update:%v Test:%v", *update, *test)

	trend := StocksTrend{
		UpdateKlinesFlag: *update,
		TestModeFlag:     *test,
	}
	trend.Start()
}
