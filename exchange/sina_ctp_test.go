package exchange

import (
	"fmt"
	"log"
	"testing"
	"time"
)

var instruments = []string{
	"RM1809",
	"AP1810",
	"rb1810",
	"CF1901",
	"m1809",
	"j1809",
	"bu1812",
	"MA1809",
	"SR1809",
	"FG1809",
	"hc1810",
}

func TestSinaCtp(t *testing.T) {
	var klines []KlineValue
	var logs []string

	for _, instrument := range instruments {

		filename := instrument + "-1day"
		sina := new(SinaCTP)
		if true {
			klines = sina.GetKline(instrument, time.Now(), nil, KlinePeriod1Day)
			SaveHistory(filename, klines)
			log.Printf("Init Done!!!")
		} else {
			klines = LoadHistory(filename)

			// ChangeOffset(0)
			// StrategyTrendTest(klines, true, true)
		}

		// for _, kline := range klines {
		// 	log.Printf("Time:%s value:%v", kline.Time, kline)
		// }

		value := 0.0
		// for value := 0.0; value < 0.6; value += 0.01 {
		// log.Printf("Klines:%v", klines)
		ChangeOffset(0.382)
		// result := StrategyTrendArea(klines, true, true)

		// // for waveLimit := 0.1; waveLimit < 1; waveLimit += 0.1 {
		// for interval = 6; interval < 20; interval++ {
		// SpliteSetWaveLimit(0.2)
		interval = 10
		result := CTPStrategyTrendSplit(klines, true, true, false)
		msg := fmt.Sprintf("[%s]Offset:%.2f Result:%s", instrument, value, result)
		logs = append(logs, msg)
		// }

		// for interval := 1; interval < 100; interval++ {
		// 	ChangeInterval(interval)
		// 	result := StrategyTrendArea(klines, true, true)
		// 	msg := fmt.Sprintf("Offset:%.2f Result:%s", value, result)
		// 	logs = append(logs, msg)
		// }

		// }
	}

	for _, msg := range logs {
		log.Printf(msg)
	}

}
