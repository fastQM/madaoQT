package task

import (
	"log"
	Exchange "madaoQT/exchange"
	"math"
	"testing"
)

const pair = "ltc/usdt"

func _TestProcessFutureTrade(t *testing.T) {
	okexFuture := new(Exchange.OKExAPI)
	okexFuture.Init(Exchange.InitConfig{
		Api:    constOKEXApiKey,
		Secret: constOEXSecretKey,
		Custom: map[string]interface{}{"tradeType": Exchange.TradeTypeFuture},
	})

	okexFuture.Start()

	okexFuture.StartContractTicker(pair, "this_week", "ltc_contract_this_week")

	var tickerValue *Exchange.TickerValue
	for {
		tickerValue = okexFuture.GetTickerValue("ltc_contract_this_week")
		if tickerValue != nil {
			Logger.Debugf("Ticker Value:%v", tickerValue)
			break
		}
	}

	resultChan := ProcessTradeRoutine(okexFuture, Exchange.TradeConfig{
		Coin:   pair,
		Type:   Exchange.OrderTypeOpenLong,
		Price:  tickerValue.Last,
		Amount: 1,
		Limit:  0.003,
	})

	select {
	case result := <-resultChan:
		Logger.Debugf("result:%v", result)
	}
}

func _TestProcessSpotTrade(t *testing.T) {
	okexSpot := new(Exchange.OKExAPI)
	okexSpot.Init(Exchange.InitConfig{
		Api:    constOKEXApiKey,
		Secret: constOEXSecretKey,
		Custom: map[string]interface{}{"tradeType": Exchange.TradeTypeSpot},
	})

	okexSpot.Start()

	okexSpot.StartCurrentTicker(pair, "ltc_spot_this_week")

	var tickerValue *Exchange.TickerValue
	for {
		tickerValue = okexSpot.GetTickerValue("ltc_spot_this_week")
		if tickerValue != nil {
			Logger.Debugf("Ticker Value:%v", tickerValue)
			break
		}
	}

	resultChan := ProcessTradeRoutine(okexSpot, Exchange.TradeConfig{
		Coin:   pair,
		Type:   Exchange.OrderTypeBuy,
		Price:  tickerValue.Last,
		Amount: 0.01,
		Limit:  0.003,
	})

	select {
	case result := <-resultChan:
		Logger.Debugf("result:%v", result)
	}
}

func TestMathTrunc(t *testing.T) {
	tmp1 := math.Trunc(1.234)
	tmp2 := math.Trunc(1.834)

	log.Printf("Value:%v %v", tmp1, tmp2)
}
