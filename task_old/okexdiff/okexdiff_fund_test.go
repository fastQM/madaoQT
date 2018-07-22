package okexdiff

import (
	"container/list"
	"log"
	Mongo "madaoQT/mongo"
	"testing"
	"time"
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
	var records []Mongo.FundInfo
	var err error
	if err, records = fundManager.CheckPosition(); err != nil {
		log.Printf("Error:%v", err)
	} else {
	}

	var amout float64
	for _, record := range records {
		item := record.SpotOpen * record.SpotAmount
		log.Printf("records:%v item:%v", record, item)
		amout += item
	}

	log.Printf("amout：%v", amout)
}

func TestCheckProfit(t *testing.T) {
	fundManager := new(OkexFundManage)
	fundManager.Init()
	// fundManager.CheckProfit()
}

func TestCheckError(t *testing.T) {
	fundManager := new(OkexFundManage)
	fundManager.Init()
	var records []Mongo.FundInfo
	var err error

	if err, records = fundManager.GetFailedPositions(); err != nil {
		log.Printf("Error:%v", err)
	}

	for _, record := range records {
		log.Printf("record:%v", record)
	}

}

func TestFixError(t *testing.T) {
	fundManager := new(OkexFundManage)
	fundManager.Init()
	updates := map[string]interface{}{
		"batch":     "af73e2d651ff",
		"spotclose": 726.21486479,
	}

	fundManager.FixFailedPosition(updates)

}

func TestDailyProfit(t *testing.T) {
	fundManager := new(OkexFundManage)
	fundManager.Init()
	date := time.Date(2018, 2, 1, 0, 0, 0, 0, time.Local)
	today := time.Now()

	for {
		log.Printf("Date:%v", date.Format("2006-01-02"))
		fundManager.CheckDailyProfit(date)
		if date.AddDate(0, 0, 1).After(today) {
			break
		} else {
			date = date.AddDate(0, 0, 1)
		}
	}

}

func TestTimeCompare(t *testing.T) {
	format := "2006-01-02"
	today := time.Now()

	next := today
	for i := 0; i < 100; i++ {
		log.Printf("Date is %v", next.Format(format))
		next = next.Add(24 * time.Hour)
	}

}

func TestList(t *testing.T) {
	datas := list.New()
	datas.PushBack(1)
	datas.PushBack(2)
	datas.PushBack(3)
	log.Printf("Datas:%v", datas)
}
