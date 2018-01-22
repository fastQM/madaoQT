package task

import (
	"log"
	"testing"
)

const constApiKey = "a982120e-8505-41db-9ae3-0c62dd27435c"
const constSecretKey = "71430C7FA63A067724FB622FB3031970"

// func _TestSpotFund(t *testing.T) {
// 	batch := "111112"
// 	fundManager := new(OkexFundManage)
// 	fundManager.Init()

// 	spotExchange := Exchange.NewOKExSpotApi(&Exchange.Config{
// 		API:    constApiKey,
// 		Secret: constSecretKey,
// 	})
// 	spotExchange.Start()

// 	fundManager.OpenPosition(Exchange.ExchangeTypeSpot, spotExchange, batch, "eth", Exchange.TradeTypeBuy)

// 	spotExchange.Trade(Exchange.TradeConfig{
// 		Pair:   "eth/usdt",
// 		Type:   Exchange.TradeTypeBuy,
// 		Amount: 0.01,
// 		Price:  700,
// 	})

// 	Utils.SleepAsyncBySecond(10)

// 	spotExchange.Trade(Exchange.TradeConfig{
// 		Pair:   "eth/usdt",
// 		Type:   Exchange.TradeTypeSell,
// 		Amount: 0.01,
// 		Price:  690,
// 	})

// 	Utils.SleepAsyncBySecond(10)

// 	fundManager.ClosePosition(Exchange.ExchangeTypeSpot, spotExchange, batch, "eth")

// 	fundManager.CalcRatio()
// }

// func _TestFutureFund(t *testing.T) {
// 	batch := "111113"
// 	fundManager := new(OkexFundManage)
// 	fundManager.Init()

// 	spotExchange := Exchange.NewOKExFutureApi(&Exchange.Config{
// 		API:    constApiKey,
// 		Secret: constSecretKey,
// 	})
// 	spotExchange.Start()

// 	fundManager.OpenPosition(Exchange.ExchangeTypeFuture, spotExchange, batch, "eth", Exchange.TradeTypeOpenLong)

// 	spotExchange.Trade(Exchange.TradeConfig{
// 		Pair:   "eth/usdt",
// 		Type:   Exchange.TradeTypeOpenLong,
// 		Amount: 1,
// 		Price:  690,
// 	})

// 	Utils.SleepAsyncBySecond(10)

// 	spotExchange.Trade(Exchange.TradeConfig{
// 		Pair:   "eth/usdt",
// 		Type:   Exchange.TradeTypeCloseLong,
// 		Amount: 1,
// 		Price:  670,
// 	})

// 	Utils.SleepAsyncBySecond(10)

// 	fundManager.ClosePosition(Exchange.ExchangeTypeFuture, spotExchange, batch, "eth")

// 	fundManager.CalcRatio()
// }

func TestCheckPosition(t *testing.T) {
	fundManager := new(OkexFundManage)
	fundManager.Init()
	if err, records := fundManager.CheckPosition(); err != nil {
		log.Printf("Error:%v", err)
	} else {
		log.Printf("records:%v", records)
	}
}
