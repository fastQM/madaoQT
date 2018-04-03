package mongotrend

import (
	"log"
	"testing"
	"time"

	Global "madaoQT/config"
	Exchange "madaoQT/exchange"
)

func TestFunds(t *testing.T) {

	db := new(TrendMongo)
	db.Connect()

	openTime, _ := time.Parse(Global.TimeFormat, "2018-03-06 08:00:00")
	openTime = openTime.Add(-8 * time.Hour)
	close1Time, _ := time.Parse(Global.TimeFormat, "2018-03-07 10:00:00")
	close1Time = close1Time.Add(-8 * time.Hour)
	close2Time, _ := time.Parse(Global.TimeFormat, "2018-03-08 16:00:00")
	close2Time = close2Time.Add(-8 * time.Hour)

	log.Printf("open time:%s, close time:%s", openTime.Format(Global.TimeFormat), close1Time.Format(Global.TimeFormat))

	openArray := []float64{850.547, 850.188, 850.000, 849.895, 849.503, 849.483, 849.432, 848.417, 848.002, 847.988, 847.987,
		847.880, 847.993, 848.887, 849.210, 849.321, 848.652, 848.520, 848.628, 848.594}

	close1Array := []float64{822.909, 822.909, 822.910, 822.910, 823.510, 823.303, 823.606, 823.529, 823.529, 823.490, 823.490,
		823.490, 823.490, 823.490}

	close2Array := []float64{754.977, 754.900, 754.850, 754.850, 754.850, 754.850}

	for i := 0; i < len(close1Array); i++ {
		info := &FundInfo{
			Batch:        "add-after",
			Pair:         "eth/usdt",
			FutureType:   Exchange.TradeTypeString[Exchange.TradeTypeOpenShort],
			FutureOpen:   openArray[i],
			FutureAmount: 25,
			FutureClose:  close1Array[i],
			OpenTime:     openTime,
			CloseTime:    close1Time,
			Status:       FundStatusClose,
		}

		if err := db.FundCollection.Insert(info); err != nil {
			log.Printf("Error:%v", err)
		}
	}

	for j := 0; j < len(close2Array); j++ {
		info := &FundInfo{
			Batch:        "add-after",
			Pair:         "eth/usdt",
			FutureType:   Exchange.TradeTypeString[Exchange.TradeTypeOpenShort],
			FutureOpen:   openArray[len(close1Array)+j],
			FutureAmount: 25,
			FutureClose:  close2Array[j],
			OpenTime:     openTime,
			CloseTime:    close2Time,
			Status:       FundStatusClose,
		}

		if err := db.FundCollection.Insert(info); err != nil {
			log.Printf("Error:%v", err)
		}
	}

}
