package exchange

/*
	该策略用于在现货期货做差价
*/

import (
	"strings"
	"log"
	"math"
)

type ExchangeHandler struct {
	Tag string
	Coin string
	Type TradeType
	Exchange *IExchange
}

type IAnalyzer struct {
	config *AnalyzerConfig
	exchanges []ExchangeHandler
	coins []string

	contracts map[string]*TickerValue
	currents map[string]*TickerValue
}

type AnalyzerConfig struct {
	Trigger float64
}

func (a *IAnalyzer) GetDefaultConfig() *AnalyzerConfig {
	return &AnalyzerConfig {
		Trigger: 1,
	}
}

func (a *IAnalyzer) Init(config *AnalyzerConfig) {
	if a.config == nil {
		a.config = a.GetDefaultConfig()
	}

	a.contracts = make(map[string]*TickerValue)
	a.currents = make(map[string]*TickerValue)
}

func (a *IAnalyzer) AddExchange(tag string, coin string, tradeType TradeType, exchange *IExchange) {
	
	exchangeHandler := ExchangeHandler {
		Tag: tag,
		Coin: strings.ToUpper(coin),
		Type: tradeType,
		Exchange: exchange,
	}

	coin = strings.ToUpper(coin)

	found := false
	a.exchanges = append(a.exchanges, exchangeHandler)
	if a.coins != nil {
		for _, coinName := range a.coins {
			if coinName == coin {
				found = true
				 break
			}
		}
	}
	if !found {
		a.coins = append(a.coins, coin)
	}

}

func (a *IAnalyzer) Watch() {


	for _, exchange := range a.exchanges {
		tmp := (*exchange.Exchange).GetTickerValue(exchange.Tag)

		// log.Printf("Type:%v, Coin:%v Value:%v", exchange.Type, exchange.Coin, tmp)

		if exchange.Type == TradeTypeContract {
			a.contracts[exchange.Coin] = tmp
		} else if exchange.Type == TradeTypeCurrent {
			a.currents[exchange.Coin] = tmp
		}
	}

	for _, coin := range a.coins {
		if a.contracts[coin] != nil && a.currents[coin] != nil {
			difference := (a.contracts[coin].Last - a.currents[coin].Last)*100/a.currents[coin].Last
			log.Printf("币种:%s, 合约价格：%.2f, 现货价格：%.2f, 价差：%.2f%%",
				coin, a.contracts[coin].Last, a.currents[coin].Last, difference)

			if math.Abs(difference) > a.config.Trigger {
				if a.contracts[coin].Last > a.currents[coin].Last {
					log.Printf("卖出合约，买入现货")
				}else {
					log.Printf("卖出现货，买入合约")
				}
			}	
		}
	}

}
