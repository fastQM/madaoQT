package exchange

import (
	"fmt"
	"log"
	"math"
)

func CheckBottomSupport(name string, kline []KlineValue) []string {

	var logs []string
	if len(kline) < 11 {
		log.Printf("无效数据长度")
		return nil
	}

	// log.Printf("kline:%v", kline)

	for i := len(kline) - 10; i < len(kline); i++ {
		if kline[i].High <= kline[i-1].Low &&
			math.Abs(kline[i].Open-kline[i].Close)*100/(kline[i].High-kline[i].Low) < (100/3) &&
			kline[i].Close > ((kline[i].High+kline[i].Low)/2) &&
			kline[i].Volumn > 2*kline[i-1].Volumn {

			if i+5 >= len(kline)-1 {
				log := fmt.Sprintf("[%s]触发条件:%2f", name, kline[i].OpenTime)
				logs = append(logs, log)
				continue
			}

			var high, low float64
			for j := 1; j <= 5; j++ {
				if high < kline[i+j].High {
					high = kline[i+j].High
				}

				if low == 0 {
					low = kline[i+j].Low
				} else if low > kline[i+j].Low {
					low = kline[i+j].Low
				}
			}

			log := fmt.Sprintf("[%s]历史触发:%f profit:%.2f%% loss:%.2f%%", name, kline[i].OpenTime, (high-kline[i].Close)*100/kline[i].Close, (low-kline[i].Close)*100/kline[i].Close)
			logs = append(logs, log)
		}

		if kline[i].Low >= kline[i-1].High &&
			math.Abs(kline[i].Open-kline[i].Close)*100/(kline[i].High-kline[i].Low) < (100/3) &&
			kline[i].Close < ((kline[i].High+kline[i].Low)/2) &&
			kline[i].Volumn > 2*kline[i-1].Volumn {

			log := fmt.Sprintf("[%s]卖点触发:%f", name, kline[i].OpenTime)
			logs = append(logs, log)
		}
	}

	return logs
}
