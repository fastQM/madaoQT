package rules

/*
	该策略用于在现货期货做差价
*/

import (
	"strings"
	"log"
	"math"
	"fmt"

	Exchange "madaoQT/exchange"
)

const Explanation = "To make profit from the difference between the contract`s price and the current`s"

type ExchangeHandler struct {
	Tag string
	Coin string
	Type Exchange.TradeType
	Exchange *Exchange.IExchange
}

type AnalyzeItem struct {
	value *Exchange.TickerValue
	exchange *Exchange.IExchange
}

type IAnalyzer struct {
	config *AnalyzerConfig
	exchanges []ExchangeHandler
	coins []string

	contracts map[string]AnalyzeItem
	currents map[string]AnalyzeItem

	event chan RulesEvent
}

type AnalyzerConfig struct {
	Trigger float64
}

func (a *IAnalyzer) GetExplanation() string {
	return Explanation
}

func (a *IAnalyzer) WatchEvent() chan RulesEvent {
	return a.event
}

func (a *IAnalyzer) triggerEvent(event EventType, msg interface{}) {
	a.event <- RulesEvent{EventType: event, Msg: msg}
}

func (a *IAnalyzer) defaultConfig() *AnalyzerConfig {
	return &AnalyzerConfig {
		Trigger: 0.6,
	}
}

func (a *IAnalyzer) Init(config *AnalyzerConfig) {
	if a.config == nil {
		a.config = a.defaultConfig()
	}

	a.event = make(chan RulesEvent)
	a.contracts = make(map[string]AnalyzeItem)
	a.currents = make(map[string]AnalyzeItem)
}

func (a *IAnalyzer) AddExchange(tag string, coin string, tradeType Exchange.TradeType, exchange *Exchange.IExchange) {
	
	exchangeHandler := ExchangeHandler {
		Tag: tag,
		Coin: strings.ToLower(coin),
		Type: tradeType,
		Exchange: exchange,
	}

	coin = strings.ToLower(coin)

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

		if exchange.Type == Exchange.TradeTypeContract {
			a.contracts[exchange.Coin] = AnalyzeItem {
				value: tmp,
				exchange: exchange.Exchange,
			}
		} else if exchange.Type == Exchange.TradeTypeCurrent {
			a.currents[exchange.Coin] = AnalyzeItem {
				value: tmp,
				exchange: exchange.Exchange,				
			}
		}
	}

	placeOrderQuan := map[string]float64 {
		"btc": 0.2,
		"ltc": 20,
	}

	for _, coin := range a.coins {
		valueContract := a.contracts[coin].value
		valueCurrent := a.currents[coin].value
		if valueContract != nil && valueCurrent != nil {

			a.triggerEvent(EventTypeTrigger, "===============================")

			difference := (valueContract.Last - valueCurrent.Last)*100/valueCurrent.Last
			msg := fmt.Sprintf("币种:%s, 合约价格：%.2f, 现货价格：%.2f, 价差：%.2f%%",
				coin, valueContract.Last, valueCurrent.Last, difference)

			log.Print(msg)
			a.triggerEvent(EventTypeTrigger, msg)

			if math.Abs(difference) > a.config.Trigger {
				if valueContract.Last > valueCurrent.Last {
					log.Printf("卖出合约，买入现货")
					
					// 期货判断bids深度
					exchange := *a.contracts[coin].exchange
					sell := exchange.GetDepthValue(coin, "", placeOrderQuan[coin])
					msg = fmt.Sprintf("[合约买单均格：%v 合约买单量:%v 操盘资金量：%v 下单深度均格：%v 下单价格:%v]", 
						sell.BidAverage, sell.BidQty, placeOrderQuan[coin], sell.BidByOrder, sell.BidPrice)

					log.Print(msg)
					a.triggerEvent(EventTypeTrigger, msg)

					exchange = *a.currents[coin].exchange
					buy := exchange.GetDepthValue(coin, "usdt", placeOrderQuan[coin])
					msg = fmt.Sprintf("[现货卖单均价：%v 现货卖单量:%v 操盘资金量:%v 下单深度均格：%v 下单价格:%v]",
						buy.AskAverage, buy.AskQty, placeOrderQuan[coin], buy.AskByOrder, buy.AskPrice)

					log.Print(msg)
					a.triggerEvent(EventTypeTrigger, msg)

					msg = fmt.Sprintf("[深度均价收益：%v%%, 限制资金收益：%v%%]",
						Exchange.GetRatio(sell.BidAverage, buy.AskAverage),
						Exchange.GetRatio(sell.BidByOrder, buy.AskByOrder))

					log.Print(msg)
					a.triggerEvent(EventTypeTrigger, msg)

				}else {
					log.Printf("买入合约, 卖出现货")

					exchange := *a.contracts[coin].exchange
					buy := exchange.GetDepthValue(coin, "", placeOrderQuan[coin])
					msg = fmt.Sprintf("[合约卖单均格：%v 合约卖单量:%v 操盘资金量：%v 下单深度均格：%v 下单价格:%v]", 
						buy.AskAverage, buy.AskQty, placeOrderQuan[coin], buy.AskByOrder, buy.AskPrice)

					log.Print(msg)
					a.triggerEvent(EventTypeTrigger, msg)

					exchange = *a.currents[coin].exchange
					sell := exchange.GetDepthValue(coin, "usdt", placeOrderQuan[coin])
					msg = fmt.Sprintf("[现货买单均价：%v 现货买单量:%v 操盘资金量:%v 下单深度均格：%v 下单价格:%v]",
						sell.BidAverage, sell.BidQty, placeOrderQuan[coin], sell.BidByOrder, sell.BidPrice)

					log.Print(msg)
					a.triggerEvent(EventTypeTrigger, msg)

					msg = fmt.Sprintf("[深度均价收益：%v%%, 限制资金收益：%v%%]",
						Exchange.GetRatio(buy.AskAverage, sell.BidAverage),
						Exchange.GetRatio(buy.AskByOrder, sell.BidByOrder))

					log.Print(msg)
					a.triggerEvent(EventTypeTrigger, msg)
				}

				a.triggerEvent(EventTypeTrigger, "===============================")
			}	
		}
	}

}
