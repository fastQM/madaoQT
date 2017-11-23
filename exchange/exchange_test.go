package exchange

import (
	"testing"
	"time"
	"log"
	"strconv"
)

func Compare(current float64, 
		currentExchange string,
		contract float64,
		contractExchange string) {

	log.Printf("现货(%s):%f, 期货(%s):%f， 价差[(期货-现货）/现货]:%f%%", 
	currentExchange, current, 
	contractExchange, contract,
	(contract - current) * 100/current)
}

func TestExchangesTicker(t *testing.T) {
	tag1 := "contract_BTC"

	okex := new(OKExAPI)
	okex.Init(tradeTypeContract)
	// okex.Login()

	tag2 := "currect_btc_usdt"

	okex2 := new(OKExAPI)
	okex2.Init(tradeTypeCurrent)

	polo := new(PoloniexAPI)
	polo.Init()
	polo.AddTicker("USDT", "BTC", "USDT_BTC")


	bittrex := new(BittrexAPI)
	bittrex.Init()
	bittrex.AddTicker("USDT", "BTC", "USDT-BTC")

	for {
		select{
			case <- time.After(3*time.Second):

				contractBTC := okex.GetTickerValue(tag1)
				currentBTC := okex2.GetTickerValue(tag2)
				if contractBTC != nil && currentBTC != nil {
					contractLast := contractBTC["last"].(float64)
					currentLast, _ := strconv.ParseFloat(currentBTC["last"].(string), 64)
					Compare(currentLast, okex2.GetExchangeName(), contractLast, okex.GetExchangeName())
				}

				value := polo.GetTickerValue("USDT_BTC")
				if value != nil && currentBTC != nil {
					contractLast := contractBTC["last"].(float64)
					current, _ := strconv.ParseFloat(value["last"].(string), 64)
					// log.Printf("%s: 现货: %v", polo.GetExchangeName(), value["last"])
					Compare(current, polo.GetExchangeName(), contractLast, okex.GetExchangeName())
				}

				values := bittrex.GetTickerValue("USDT-BTC")

				if value != nil && values != nil {
					last := values["Last"].(float64)
					contractLast := contractBTC["last"].(float64)

					Compare(last, bittrex.GetExchangeName(), contractLast, okex.GetExchangeName())
				}


			case event := <- okex.WatchEvent():
				if event == EventConnected{
					log.Printf("connected")
					okex.StartContractTicker(X_BTC, Y_THIS_WEEK, tag1)
				}else if event == EventError {
					log.Printf("reconnnect")
					okex.Init(tradeTypeContract)
				}
			case event := <- okex2.WatchEvent():
				if event == EventConnected{
					log.Printf("connected")
					okex2.StartCurrentTicker("btc", "usdt", tag2)
				}else if event == EventError {
					log.Printf("reconnnect")
					okex2.Init(tradeTypeCurrent)
				}


		}
	}
}

func _TestBittrexTicker(t *testing.T) {

	bittrex := new(BittrexAPI)
	bittrex.Init()
	bittrex.AddTicker("USDT", "BTC", "USDT-BTC")

	for {
		select{
		case <-time.After(3*time.Second):
			values := bittrex.GetTickerValue("USDT-BTC")
			if values != nil {
				log.Printf("Value:%v %v", values, values["Last"].(float64))
			}
		}
	}
}