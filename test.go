package main

import (
	"log"
	"time"

	Exchange "madaoQT/exchange"
)

func main(){

	analyzer := new(Exchange.IAnalyzer)

	log.Print("启动OKEx合约监视程序")
	okexContract := new (Exchange.OKExAPI)
	okexContract.Init(Exchange.TradeTypeContract)

	log.Printf("启动OKEx现货监视程序")
	okexCurrent := new (Exchange.OKExAPI)
	okexCurrent.Init(Exchange.TradeTypeCurrent)

	for{
		select{
		case event := <-okexContract.WatchEvent():
			if event == Exchange.EventConnected {
				okexContract.StartContractTicker("btc", "this_week", "btc_contract_this_week")
				p := Exchange.IExchange(okexContract)
				analyzer.AddExchange("btc_contract_this_week", "btc",
					Exchange.TradeTypeContract, &p)
				okexContract.StartContractTicker("ltc", "this_week", "ltc_contract_this_week")
				analyzer.AddExchange("ltc_contract_this_week", "ltc", 
					Exchange.TradeTypeContract, &p)

			} else if event == Exchange.EventError {
				okexContract.Init(Exchange.TradeTypeContract)
			}
		case event := <-okexCurrent.WatchEvent():
			if event == Exchange.EventConnected {
				okexCurrent.StartCurrentTicker("btc", "usdt", "current_btc_usdt")
				p := Exchange.IExchange(okexCurrent)
				analyzer.AddExchange("current_btc_usdt", "btc", 
					Exchange.TradeTypeCurrent, &p)
				okexCurrent.StartCurrentTicker("ltc", "usdt", "current_ltc_usdt")
				analyzer.AddExchange("current_ltc_usdt", "ltc", 
					Exchange.TradeTypeCurrent, &p)
			} else if event == Exchange.EventError {
				okexCurrent.Init(Exchange.TradeTypeCurrent)
			}
		case <- time.After(3 * time.Second): 
			analyzer.Analyze()
		}
	}

}