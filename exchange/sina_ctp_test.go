package exchange

import (
	"fmt"
	"log"
	"testing"
	"time"
)

var instruments = []string{
	// "RM1809",
	// "AP1810",
	// "rb1810",
	// "CF1901",
	// "m1809",
	// "j1809",
	// "bu1812",
	// "MA1809",
	// "SR1809",
	// "FG1809",
	// "hc1810",

	"rb0",
	// "RM0", //波动不活跃不操作
	"AP0",
	"CF0",
	// "m0", //波动不活跃不操作
	"j0",
	// "bu0",
	"MA0",
	"SR0",
	// "FG0",	// 亏损
	// "hc0",
	"ta0",
	"l0",
	// "pp0",	// 	亏损
	"i0",
	"ru0",

	// "v0", // 收益太低
	// "y0", // 豆油
	// "p0", //棕榈

	// "cu0",
	// "au0",
	// "jd0",
	// "pb0",
	// "sn0",
	// "fu0",
	// "sf0",
	// "sm0",
	// "zc0",
}

func TestSinaCtp(t *testing.T) {
	var klines []KlineValue
	var logs []string

	for _, instrument := range instruments {
		log.Printf("当前种类:%v", instrument)
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

		klines = CTPDailyKlinesToWeek(klines)
		// for _, kline := range klinesByWeek {
		// 	log.Printf("Time:%s High:%v Low:%v Open:%v Close:%v", kline.Time, kline.High, kline.Low, kline.Open, kline.Close)
		// }
		// return

		// klinesByMonth := CTPDailyKlinesToMonth(klines)

		// klinesByYears := CTPDailyKlinesSplitToYears(klines)
		// for _, kline := range klinesByWeek {
		// 	log.Printf("Time:%s High:%v Low:%v Open:%v Close:%v", kline.Time, kline.High, kline.Low, kline.Open, kline.Close)
		// }
		// return

		// // for waveLimit := 0.1; waveLimit < 1; waveLimit += 0.1 {
		// for interval = 6; interval < 20; interval++ {
		// SpliteSetWaveLimit(0.2)
		interval = 10
		// for _, klines := range klinesByYears {
		result := CTPStrategyTrendSplit(klines, true, true, false)
		msg := fmt.Sprintf("[%v][%s]Result:%s", klines[0].Time, instrument, result)
		logs = append(logs, msg)
		// }

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
