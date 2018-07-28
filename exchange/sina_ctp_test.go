package exchange

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestSinaCtp(t *testing.T) {
	var klines []KlineValue
	var logs []string

	name := "rb0"
	filename := name + "-1day"
	sina := new(SinaCTP)
	if false {
		klines = sina.GetKline(name, time.Now(), nil, KlinePeriod1Day)
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
	// SpliteSetWaveLimit(0.2)
	result := CTPStrategyTrendSplit(klines, true, true, true)
	msg := fmt.Sprintf("Offset:%.2f Result:%s", value, result)
	logs = append(logs, msg)
	// // }

	// for interval := 1; interval < 100; interval++ {
	// 	ChangeInterval(interval)
	// 	result := StrategyTrendArea(klines, true, true)
	// 	msg := fmt.Sprintf("Offset:%.2f Result:%s", value, result)
	// 	logs = append(logs, msg)
	// }

	// }
	for _, msg := range logs {
		log.Printf(msg)
	}

}
