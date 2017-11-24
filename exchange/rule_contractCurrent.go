package exchange

/*
	该策略用于在现货期货做差价
*/

import (
	"strings"
	"log"
)

type ExchangeHandler struct {
	Tag string
	Coin string
	Type TradeType
	Exchange *IExchange
}

type IAnalyzer struct {
	exchanges []ExchangeHandler
	coins []string
}

func (a *IAnalyzer) AddExchange(tag string, coin string, tradeType TradeType, exchange *IExchange) {
	
	exchangeHandler := ExchangeHandler {
		Tag: tag,
		Coin: coin,
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

func (a *IAnalyzer) Analyze() {
	for _, exchange := range a.exchanges {
		log.Printf("Type:%v, Coin:%v Value:%v", exchange.Type, exchange.Coin, 
			(*exchange.Exchange).GetTickerValue(exchange.Tag))
	}
}