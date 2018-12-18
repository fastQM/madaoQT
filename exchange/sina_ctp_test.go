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

	"rb1901",
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

type InstrumentInfo struct {
	name      string
	lossLimit float64
	strategy  int
	base      float64
	costUnit  float64
}

func TestSinaCtp(t *testing.T) {
	var klines []KlineValue
	var logs []string

	// instrument := InstrumentInfo{
	// 	name:      "rb1901",
	// 	lossLimit: 0.1,
	// 	base:      15000,
	// 	costUnit:  10,
	// 	strategy:  StrategyIntervalBreak,
	// }

	// instrument := InstrumentInfo{
	// 	name:      "hc0",
	// 	lossLimit: 0.1,
	// 	strategy:  StrategyAreaBreak,
	// 	base:      15000,
	// 	costUnit:  10,
	// }

	// instrument := InstrumentInfo{
	// 	name:      "cf0",
	// 	lossLimit: 0.06,
	// 	base:      90000,
	// 	costUnit:  25,
	// 	strategy:  StrategyAreaBreak,
	// }

	// instrument := InstrumentInfo{
	// 	name:      "ma1901",
	// 	lossLimit: 0.1,
	// 	strategy:  StrategyAreaBreak,
	// 	base:      10000,
	// 	costUnit:  10,
	// }

	// instrument := InstrumentInfo{
	// 	name:      "ta1901",
	// 	lossLimit: 0.08,
	// 	base:      16000,
	// 	costUnit:  5,
	// 	strategy:  StrategyAreaBreak,
	// }

	// instrument := InstrumentInfo{
	// 	name:      "sr0",
	// 	lossLimit: 0.1,
	// 	strategy:  StrategyAreaBreak,
	// 	base:      20000,
	// 	costUnit:  20,
	// }

	// instrument := InstrumentInfo{
	// 	name:      "ru0",
	// 	lossLimit: 0.03,
	// 	strategy:  StrategyAreaBreak,
	// }

	// instrument := InstrumentInfo{
	// 	name:      "l0",
	// 	lossLimit: 0.07,
	// 	strategy:  StrategyAreaBreak,
	// }

	instrument := InstrumentInfo{
		name:      "j1901",
		lossLimit: 0.05,
		base:      80000,
		strategy:  StrategyAreaBreak,
		costUnit:  50,
	}

	// instrument := InstrumentInfo{
	// 	name:      "ag0",
	// 	lossLimit: 0.09,
	// 	strategy:  StrategyAreaBreak,
	// 	base:      20000,
	// 	costUnit:  15,
	// }

	// instrument := InstrumentInfo{
	// 	name:      "pp0",
	// 	lossLimit: 0.1,
	// 	strategy:  StrategyAreaBreak,
	// 	base:      10000,
	// 	costUnit:  10,
	// }

	log.Printf("当前种类:%v", instrument.name)
	filename := instrument.name + "-1day"
	sina := new(SinaCTP)
	if true {
		klines = sina.GetKline(instrument.name, time.Now(), nil, KlinePeriod1Day)
		SaveHistory(filename, klines)
		log.Printf("Init Done!!!")
	} else {
		klines = LoadHistory(filename)
	}

	var checkedKlines []KlineValue
	for _, kline := range klines {
		if kline.Open == 0 || kline.Close == 0 || kline.High == 0 || kline.Low == 0 {
			continue
		}
		checkedKlines = append(checkedKlines, kline)
		log.Printf("Time:%s value:%v", kline.Time, kline)
	}

	klines = checkedKlines

	// klines = CTPDailyKlinesToWeek(klines)
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

	interval = 30
	lossLimit = instrument.lossLimit
	SetBaseAmount(instrument.base)
	SetUnitCost(instrument.costUnit)
	// for lossLimit = 0.01; lossLimit < 0.20; lossLimit += 0.01 {
	// for interval = 15; interval < 50; interval++ {
	// SpliteSetWaveLimit(0.2)

	// for _, klines := range klinesByYears {
	var result string
	if instrument.strategy == StrategyAreaBreak {
		result = CTPStrategyTrendAreaBreak(klines, true, true, true)
	} else if instrument.strategy == StrategyIntervalBreak {
		result = CTPStrategyTrendIntervalBreak(klines, true, true, true)
	}

	msg := fmt.Sprintf("[%v][%s]Result:%s", klines[0].Time, instrument.name, result)
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
	// }

	for _, msg := range logs {
		log.Printf(msg)
	}

}
