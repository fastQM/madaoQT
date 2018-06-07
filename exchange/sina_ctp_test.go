package exchange

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func RevertArray(array []KlineValue) []KlineValue {
	var tmp KlineValue
	var length int

	if len(array)%2 != 0 {
		length = len(array) / 2
	} else {
		length = len(array)/2 - 1
	}
	for i := 0; i <= length; i++ {
		tmp = array[i]
		array[i] = array[len(array)-1-i]
		array[len(array)-1-i] = tmp

	}
	return array
}
func TestSinaCtp(t *testing.T) {
	var klines []KlineValue
	var logs []string

	name := "rb0"
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
