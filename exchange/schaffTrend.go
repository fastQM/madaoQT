package exchange

import (
	"errors"
	"log"
)

//https://www.investopedia.com/articles/forex/10/schaff-trend-cycle-indicator.asp

type EMAIndicator struct {
	EMAWeightY uint
	last       float64
	a          float64
}

func (p *EMAIndicator) Add(today float64) float64 {

	if p.EMAWeightY == 0 {
		log.Fatal("EMA Weight is zero")
		return 0
	} else if p.a == 0 {
		p.a = 2.0 / (1.0 + float64(p.EMAWeightY))
		// log.Printf("EMA A:%v", p.a)
	}

	if p.last == 0 {
		p.last = today
		// log.Printf("The first or the invalid value")
		return today
	}

	// p.last = today*a + p.last*(1-a)
	p.last = p.a*(today-p.last) + p.last
	// log.Printf("Current EMA:%v", p.last)
	return p.last
}

func (p *EMAIndicator) AddTest(today float64) float64 {

	// if p.EMAWeightY == 0 {
	// 	log.Fatal("EMA Weight is zero")
	// 	return 0
	// } else if p.a == 0 {
	// 	p.a = 2.0 / (1.0 + float64(p.EMAWeightY))
	// 	log.Printf("EMA A:%v", p.a)
	// }

	// if p.last == 0 {
	// 	p.last = today
	// 	log.Printf("The first or the invalid value")
	// 	return today
	// }

	last := p.a*(today-p.last) + p.last
	return last
}

type SchaffTrend struct {
	Period     int
	FastLength uint
	SlowLength uint
	Factor     float64

	OverSell float64
	OverBuy  float64

	TrionePeriod int

	fastEMA *EMAIndicator
	slowEMA *EMAIndicator

	macd []float64
	f1   []float64
	pf   []float64
	f2   []float64
	pff  []float64
}

func (p *SchaffTrend) update(value float64) float64 {
	fastMA := p.fastEMA.Add(value)
	slowMA := p.slowEMA.Add(value)
	// log.Printf("Fast:%v Slow:%v", fastMA, slowMA)
	return fastMA - slowMA
}

func (p *SchaffTrend) updateTest(value float64) float64 {
	fastMA := p.fastEMA.AddTest(value)
	slowMA := p.slowEMA.AddTest(value)
	// log.Printf("Fast:%v Slow:%v", fastMA, slowMA)
	return fastMA - slowMA
}

func (p *SchaffTrend) lowest(values []float64) float64 {
	lowest := values[0]
	for _, value := range values {
		if value < lowest {
			lowest = value
		}
	}
	return lowest
}

func (p *SchaffTrend) highest(values []float64) float64 {
	highest := values[0]
	for _, value := range values {
		if value > highest {
			highest = value
		}
	}

	return highest
}

// type KlineValue struct {
// 	Time      string
// 	OpenTime  float64
// 	Open      float64
// 	High      float64
// 	Low       float64
// 	Close     float64
// 	Volumn    float64
// 	CloseTime float64
// }

// The length of klines should be more than 200
func (p *SchaffTrend) SchaffIndicatorInit(klines []KlineValue) bool {
	if p.FastLength == 0 || p.SlowLength == 0 || p.Period == 0 || p.Factor == 0 || len(klines) < 200 || p.OverBuy == 0 || p.OverSell == 0 {
		log.Printf("Invalid Paramters")
		return false
	} else {
		p.fastEMA = nil
		p.fastEMA = &EMAIndicator{
			EMAWeightY: p.FastLength,
		}

		p.slowEMA = nil
		p.slowEMA = &EMAIndicator{
			EMAWeightY: p.SlowLength,
		}

		p.f1 = nil
		p.f2 = nil
		p.pf = nil
		p.pff = nil
	}

	for _, kline := range klines {
		p.UpdateSchaff(kline)
	}

	return true
}

func (p *SchaffTrend) GetLastSchaffValue() float64 {
	return p.pff[len(p.pff)-1]
}

func (p *SchaffTrend) GetTironeArea(klines []KlineValue) (float64, float64) {

	if p.TrionePeriod == 0 {
		log.Printf("Invalid TrionePeriod")
		return 0, 0
	}

	klines = klines[len(klines)-p.TrionePeriod:]

	var high, low float64
	for _, kline := range klines {
		if high == 0 || high < kline.Close {
			high = kline.Close
		}

		if low == 0 || low > kline.Close {
			low = kline.Close
		}
	}

	return high - (high-low)/3, low + (high-low)/3
}

func (p *SchaffTrend) UpdateSchaff(kline KlineValue) {

	var length int
	m := p.update(kline.Close)
	p.macd = append(p.macd, m)
	length = len(p.macd)
	if length < p.Period {
		return
	}
	v1 := p.lowest(p.macd[length-p.Period:])
	v2 := p.highest(p.macd[length-p.Period:]) - v1

	var currentF1, currentPF, currentF2, currentPFF float64
	if v2 > 0 {
		currentF1 = (m - v1) / v2 * 100
		// p.f1 = append(p.f1, currentF1)
	} else {
		if p.f1 == nil {
			currentF1 = 0.0
			// p.f1 = append(p.f1, currentF1)
		} else {
			length = len(p.f1)
			// p.f1 = append(p.f1, p.f1[length-1])
			currentF1 = p.f1[length-1]
		}
	}

	p.f1 = append(p.f1, currentF1)

	if p.pf == nil {
		currentPF = currentF1

	} else {
		length = len(p.pf)
		currentPF = p.pf[length-1] + p.Factor*(currentF1-p.pf[length-1])
	}

	p.pf = append(p.pf, currentPF)

	length = len(p.pf)
	if length < p.Period {
		// log.Printf("Invalid PF length")
		return
	}
	v3 := p.lowest(p.pf[length-p.Period:])
	v4 := p.highest(p.pf[length-p.Period:]) - v3

	if v4 > 0 {
		currentF2 = (currentPF - v3) / v4 * 100
	} else {
		if p.f2 == nil {
			currentF2 = 0
		} else {
			length := len(p.f2)
			currentF2 = p.f2[length-1]
		}
	}

	p.f2 = append(p.f2, currentF2)

	if p.pff == nil {
		currentPFF = currentF2
	} else {
		length := len(p.pff)
		currentPFF = p.pff[length-1] + p.Factor*(currentF2-p.pff[length-1])
	}

	p.pff = append(p.pff, currentPFF)

	// log.Printf("Time:%v macd:%v Pff:%v", time.Unix(int64(kline.OpenTime), 0), m, currentPFF)
}

func (p *SchaffTrend) GetThreshValue(direction TradeType, last KlineValue, accuracy float64) (error, float64, int) {

	// var longTrend bool
	// length := len(p.pff)
	// lastPFF := p.pff[length-1]

	if p.f1 == nil || p.f2 == nil || p.pf == nil || p.pff == nil {
		return errors.New("Not init?"), 0, 0
	}

	// if lastPFF < p.OverSell { // 小与超卖，等待买点
	// 	log.Printf("小与超卖，等待买点")
	// 	longTrend = false
	// } else if lastPFF > p.OverBuy { // 大于超买，等待卖点
	// 	log.Printf("大于超买，等待卖点")
	// 	longTrend = true
	// } else {
	// 	return errors.New("Invalid Trend"), 0, 0
	// }

	if direction == TradeTypeOpenLong {
		lastValue := last.Close
		for tmp := last.Close; tmp >= 0; {
			// log.Printf("TMP:%v", tmp)
			tmp -= accuracy * 100
			if p.calcSchaff(tmp) < p.OverBuy {
				for i := lastValue; i > tmp; {
					// log.Printf("I:%v >tmp:%v", i, tmp)
					i -= accuracy * 10
					if p.calcSchaff(i) < p.OverBuy {
						for j := lastValue; j > i; {
							// log.Printf("J:%v >i:%v", j, i)
							j -= accuracy
							if p.calcSchaff(j) < p.OverBuy {
								return nil, j, 1
							}
						}

					} else {
						lastValue = i
					}
				}

			} else {
				lastValue = tmp
			}
		}
	} else if direction == TradeTypeOpenShort {
		lastValue := last.Close
		for tmp := last.Close; tmp < last.Close*10; {
			// log.Printf("TMP:%v", tmp)
			tmp += accuracy * 100
			if p.calcSchaff(tmp) > p.OverSell {
				for i := lastValue; i < tmp; {
					// log.Printf("I:%v <tmp:%v", i, tmp)
					i += accuracy * 10
					if p.calcSchaff(i) > p.OverSell {
						for j := lastValue; j < i; {
							// log.Printf("J:%v <i:%v", j, i)
							j += accuracy
							if p.calcSchaff(j) > p.OverSell {
								return nil, j, 2
							}
						}

					} else {
						lastValue = i
					}
				}

			} else {
				lastValue = tmp
			}
		}
	}

	return errors.New("Invalid Trade Type"), 0, 0

}

func (p *SchaffTrend) calcSchaff(value float64) float64 {

	var length int

	tmpMACD := p.macd[:]
	length = len(tmpMACD)
	m := p.updateTest(value)
	tmpMACD = append(tmpMACD, m)

	v1 := p.lowest(tmpMACD[length-p.Period:])
	v2 := p.highest(tmpMACD[length-p.Period:]) - v1

	var currentF1, currentPF, currentF2, currentPFF float64
	tmpF1 := p.f1[:]
	if v2 > 0 {
		currentF1 = (m - v1) / v2 * 100
		// p.f1 = append(p.f1, currentF1)
	} else {
		if tmpF1 == nil {
			currentF1 = 0.0
			// p.f1 = append(p.f1, currentF1)
		} else {
			length = len(tmpF1)
			// p.f1 = append(p.f1, p.f1[length-1])
			currentF1 = tmpF1[length-1]
		}
	}

	// tmpF1 = append(tmpF1, currentF1)

	tmpPF := p.pf[:]
	if tmpPF == nil {
		currentPF = currentF1

	} else {
		length = len(tmpPF)
		currentPF = tmpPF[length-1] + p.Factor*(currentF1-tmpPF[length-1])
	}

	tmpPF = append(tmpPF, currentPF)

	length = len(tmpPF)
	v3 := p.lowest(tmpPF[length-p.Period:])
	v4 := p.highest(tmpPF[length-p.Period:]) - v3

	tmpF2 := p.f2[:]
	if v4 > 0 {
		currentF2 = (currentPF - v3) / v4 * 100
	} else {
		if tmpF2 == nil {
			currentF2 = 0
		} else {
			length := len(tmpF2)
			currentF2 = tmpF2[length-1]
		}
	}

	// tmpF2 = append(tmpF2, currentF2)

	tmpPFF := p.pff[:]
	if tmpPFF == nil {
		currentPFF = currentF2
	} else {
		length := len(tmpPFF)
		currentPFF = tmpPFF[length-1] + p.Factor*(currentF2-tmpPFF[length-1])
	}

	tmpF1 = nil
	tmpF2 = nil
	tmpMACD = nil
	tmpPF = nil
	tmpPFF = nil
	return currentPFF

}
