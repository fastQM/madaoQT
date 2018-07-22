package task

import (
	"log"
	"math"
	"testing"

	Exchange "madaoQT/exchange"
	Utils "madaoQT/utils"
)

const constOKEXApiKey = "a982120e-8505-41db-9ae3-0c62dd27435c"
const constOEXSecretKey = "71430C7FA63A067724FB622FB3031970"

const pair = "ltc/usdt"

func _TestProcessFutureTrade(t *testing.T) {
	okexFuture := new(Exchange.OKExAPI)
	okexFuture.SetConfigure(Exchange.Config{
		API:    constOKEXApiKey,
		Secret: constOEXSecretKey,
		Custom: map[string]interface{}{
			"exchangeType": Exchange.ExchangeTypeFuture,
			"period":       "this_week",
		},
	})

	okexFuture.Start()

	okexFuture.StartTicker(pair)

	var tickerValue *Exchange.TickerValue
	for {
		tickerValue = okexFuture.GetTicker(pair)
		if tickerValue != nil {
			Logger.Debugf("Ticker Value:%v", tickerValue)
			break
		}
	}

	resultChan := ProcessTradeRoutine(okexFuture, Exchange.TradeConfig{
		Pair:   pair,
		Type:   Exchange.TradeTypeOpenLong,
		Price:  tickerValue.Last,
		Amount: 1,
		Limit:  0.003,
	}, nil)

	select {
	case result := <-resultChan:
		Logger.Debugf("result:%v", result)
	}
}

func _TestProcessSpotTrade(t *testing.T) {
	okexSpot := new(Exchange.OKExAPI)
	okexSpot.SetConfigure(Exchange.Config{
		API:    constOKEXApiKey,
		Secret: constOEXSecretKey,
		Custom: map[string]interface{}{"exchangeType": Exchange.ExchangeTypeSpot},
	})

	okexSpot.Start()

	okexSpot.StartTicker(pair)

	var tickerValue *Exchange.TickerValue
	for {
		tickerValue = okexSpot.GetTicker(pair)
		if tickerValue != nil {
			Logger.Debugf("Ticker Value:%v", tickerValue)
			break
		}
	}

	resultChan := ProcessTradeRoutine(okexSpot, Exchange.TradeConfig{
		Pair:   pair,
		Type:   Exchange.TradeTypeBuy,
		Price:  tickerValue.Last,
		Amount: 0.01,
		Limit:  0.003,
	}, nil)

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

func _TestStartTask(t *testing.T) {
	task := TaskHotLoad{}
	err := task.InstallTaskAndRun("okexdiff", "monrnig")
	if err != nil {
		log.Printf("Error:%v", err)
		return
	}

	Utils.SleepAsyncBySecond(10)

	task.ExitTask()
}

func TestCalcPrice(t *testing.T) {
	var totalQuantity, totalAmount, askPrice float64
	isFuture := true
	var amount float64
	var quantity float64
	var askFlag bool
	amount = 5000
	quantity = 5

	asks := []Exchange.DepthPrice{
		{983.497, 0.3152}, {983.186, 0.2237}, {982.471, 1.2723}, {982.292, 0.6413}, {980.897, 4.0779},
	}
	for _, depth := range asks {

		if isFuture {
			if (totalQuantity + depth.Quantity) >= quantity {

				totalAmount += depth.Price * (quantity - totalQuantity)
				totalQuantity = quantity
				log.Printf("totalQuantity %v totalAmount %v", totalQuantity, totalAmount)
				askFlag = true
				askPrice = totalAmount / totalQuantity
				break
			} else {

				totalQuantity += depth.Quantity
				totalAmount += depth.Quantity * depth.Price
				log.Printf("totalQuantity %v totalAmount %v", totalQuantity, totalAmount)
			}
		} else {
			if (totalAmount + depth.Price*depth.Quantity) >= amount {

				totalQuantity += (amount - totalAmount) / depth.Price
				totalAmount = amount
				log.Printf("totalQuantity %v totalAmount %v amount %v", totalQuantity, totalAmount, (amount - totalAmount))
				askFlag = true
				askPrice = totalAmount / totalQuantity
				break
			} else {
				totalQuantity += depth.Quantity
				totalAmount += depth.Quantity * depth.Price
				log.Printf("totalQuantity %v totalAmount %v", totalQuantity, totalAmount)
			}
		}

	}

	log.Printf("Flag %v Price %v", askFlag, askPrice)
}
