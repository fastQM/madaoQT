package main

import (
	"io/ioutil"
	"log"
	"madaoQT/exchange"
	"strings"
	"testing"
)

func TestWrite(t *testing.T) {
	var datas []exchange.KlineValue

	data := exchange.KlineValue{
		OpenTime: 111,
		Open:     111,
	}

	datas = append(datas, data)
	datas = append(datas, data)
	datas = append(datas, data)
	exchange.SaveHistory("123456", datas)

	klines := exchange.LoadHistory("123456")
	if klines != nil {
		log.Printf("klines:%v", klines)
	}
}

func TestCombine(t *testing.T) {
	var stocks []string
	// 获取上海股票列表
	shlist, err := ioutil.ReadFile("./shstocks.txt")
	if err != nil {
		log.Printf("error:%v", err)
		return
	}

	shstocks := strings.Split(string(shlist), "\r\n")
	for _, stock := range shstocks {
		stocks = append(stocks, "sh"+stock)
	}

	var combinedList string
	var counter int
	for _, stock := range stocks {
		if combinedList == "" {
			combinedList = stock
		} else {
			combinedList = combinedList + "," + stock
		}
		if counter < 500 {
			counter++
		} else {
			break
		}

	}
	server := new(exchange.TencentStock)

	prices := server.GetMultipleLast(combinedList)
	if prices != nil {
		for _, price := range prices {
			log.Printf("%v", price)
		}
	}

}

func TestEmptyMap(t *testing.T) {
	type TestStruct struct {
		name string
		age  int
	}
	hello := map[string]TestStruct{
		"1": TestStruct{
			name: "qiu",
			age:  188,
		},
		"2": TestStruct{
			name: "min",
			age:  12,
		},
	}

	log.Printf("%v", hello["3"])
}
