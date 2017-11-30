package main

import (
	"time"

	"github.com/kataras/golog"

	Exchange "madaoQT/exchange"
	Rules "madaoQT/rules"
	Web "madaoQT/web"
	Utils "madaoQT/utils"
)

var Logger *golog.Logger

func init(){
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
}

func main(){

	analyzer := new(Rules.IAnalyzer)
	analyzer.Init(nil)

	Logger.Info("启动OKEx合约监视程序")
	okexContract := new (Exchange.OKExAPI)
	okexContract.Init(Exchange.TradeTypeContract)

	Logger.Info("启动OKEx现货监视程序")
	okexCurrent := new (Exchange.OKExAPI)
	okexCurrent.Init(Exchange.TradeTypeCurrent)

	http := new(Web.HttpServer)
	go http.SetupHttpServer()
	go Utils.OpenBrowser("http://localhost:8080")

	go func(){
		
		for{
			select{
			case event := <-analyzer.WatchEvent():
				if event.EventType == Rules.EventTypeTrigger {
					http.BroadcastByWebsocket(event.Msg)
				}
			}
		}
	}()

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
		case <- time.After(5 * time.Second): 
			analyzer.Watch()
		}
	}

}