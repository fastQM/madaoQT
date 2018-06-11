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

	name := "ih0"
	sina := new(SinaCTP)
	if true {
		klines = sina.GetKline(name, time.Now(), nil, 0)
		SaveHistory(name, klines)
		log.Printf("Init Done!!!")
	} else {
		klines = LoadHistory(name)

		// ChangeOffset(0)
		// StrategyTrendTest(klines, true, true)
	}

	// klines = RevertArray(klines)

	value := 0.0
	// for value := 0.0; value < 0.6; value += 0.01 {
	// log.Printf("Klines:%v", klines)
	ChangeOffset(0.382)
	result := StrategyTrendArea(klines, true, true)
	msg := fmt.Sprintf("Offset:%.2f Result:%s", value, result)
	logs = append(logs, msg)

	// }
	for _, msg := range logs {
		log.Printf(msg)
	}

}
