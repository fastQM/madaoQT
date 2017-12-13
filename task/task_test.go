package task

import (
	"log"
	"math"
	"testing"

	Exchange "madaoQT/exchange"
	Utils "madaoQT/utils"
)

const pair = "ltc/usdt"

func _TestProcessFutureTrade(t *testing.T) {
	okexFuture := new(Exchange.OKExAPI)
	okexFuture.Init(Exchange.InitConfig{
		Api:    constOKEXApiKey,
		Secret: constOEXSecretKey,
		Custom: map[string]interface{}{"exchangeType": Exchange.ExchangeTypeFuture},
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
		Type:   Exchange.TradeTypeOpenLong,
		Price:  tickerValue.Last,
		Amount: 1,
		Limit:  0.003,
	}, nil, nil)

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
		Custom: map[string]interface{}{"exchangeType": Exchange.ExchangeTypeSpot},
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
		Type:   Exchange.TradeTypeBuy,
		Price:  tickerValue.Last,
		Amount: 0.01,
		Limit:  0.003,
	}, nil, nil)

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

func TestCheckStruct(t *testing.T) {
	type Test struct {
		S1 string
		S2 string
	}

	var value = Test{
		"hello",
		"world",
	}

	log.Printf("1:%v", value.S1)
	log.Printf("2:%v", value.S2)
}

func TestStartTask(t *testing.T) {
	task := Task{}
	err := task.InstallTaskAndRun("okexdiff", "monrnig")
	if err != nil {
		log.Printf("Error:%v", err)
		return
	}

	Utils.SleepAsyncBySecond(10)

	task.ExitTask()
}
