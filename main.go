package main

import (
	"fmt"
	"os"
	"time"

	Exchange "madaoQT/exchange"
	Http "madaoQT/http"
	Task "madaoQT/task"
	Utils "madaoQT/utils"

	"github.com/kataras/golog"
)

var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
}

func handleCmd() {
	var cmd string
	for {
		fmt.Scanln(&cmd)
		switch cmd {
		case "q":
			Logger.Info("Exiting...")
			os.Exit(0)
		}
	}

}

const constOKEXApiKey = "a982120e-8505-41db-9ae3-0c62dd27435c"
const constOEXSecretKey = "71430C7FA63A067724FB622FB3031970"

func main() {

	go handleCmd()

	analyzer := new(Task.IAnalyzer)
	analyzer.Init(nil)

	Logger.Info("启动OKEx合约监视程序")
	okexContract := new(Exchange.OKExAPI)
	okexContract.Init(Exchange.InitConfig{
		Api:    constOKEXApiKey,
		Secret: constOEXSecretKey,
		Custom: map[string]interface{}{"tradeType": Exchange.TradeTypeFuture},
	})
	okexContract.Start()

	Logger.Info("启动OKEx现货监视程序")
	okexCurrent := new(Exchange.OKExAPI)
	okexCurrent.Init(Exchange.InitConfig{
		Api:    constOKEXApiKey,
		Secret: constOEXSecretKey,
		Custom: map[string]interface{}{"tradeType": Exchange.TradeTypeSpot},
	})
	okexCurrent.Start()

	http := new(Http.HttpServer)
	go http.SetupHttpServer()
	go Utils.OpenBrowser("http://localhost:8080")

	go func() {

		for {
			select {
			case event := <-analyzer.WatchEvent():
				if event.EventType == Task.EventTypeTrigger {
					http.BroadcastByWebsocket(event.Msg)
				}
			}
		}
	}()

	for {
		select {
		case event := <-okexContract.WatchEvent():
			if event == Exchange.EventConnected {
				okexContract.StartContractTicker("btc", "this_week", "btc_contract_this_week")
				p := Exchange.IExchange(okexContract)
				analyzer.AddExchange("btc_contract_this_week", "btc",
					Exchange.TradeTypeFuture, &p)
				okexContract.StartContractTicker("ltc", "this_week", "ltc_contract_this_week")
				analyzer.AddExchange("ltc_contract_this_week", "ltc",
					Exchange.TradeTypeFuture, &p)

			} else if event == Exchange.EventError {
				okexContract.Start()
			}
		case event := <-okexCurrent.WatchEvent():
			if event == Exchange.EventConnected {
				okexCurrent.StartCurrentTicker("btc", "usdt", "current_btc_usdt")
				p := Exchange.IExchange(okexCurrent)
				analyzer.AddExchange("current_btc_usdt", "btc",
					Exchange.TradeTypeSpot, &p)
				okexCurrent.StartCurrentTicker("ltc", "usdt", "current_ltc_usdt")
				analyzer.AddExchange("current_ltc_usdt", "ltc",
					Exchange.TradeTypeSpot, &p)
			} else if event == Exchange.EventError {
				okexCurrent.Start()
			}
		case <-time.After(10 * time.Second):
			analyzer.Watch()
		}
	}

}
