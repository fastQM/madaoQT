package exchange

import (
	"log"
	"testing"
	"time"
)

const Period5Min = 5 * 60
const Period15Min = 15 * 60
const Period30Min = 30 * 60
const Period2H = 120 * 60
const Period4H = 240 * 60
const Period1Day = 86400

/*
[eth]
1. Period5Min
2018/03/26 14:27:51 盈利次数：1039 亏损次数 ：666
2018/03/26 14:27:51 盈利求和：1179.919932 亏损求和 ：-284.852370
胜率 60%


2. Period15Min
2018/03/26 14:28:14 盈利次数：347 亏损次数 ：205
2018/03/26 14:28:14 盈利求和：698.230341 亏损求和 ：-144.313904
胜率 63%

3. Period30Min
2018/03/26 14:28:37 盈利次数：183 亏损次数 ：98
2018/03/26 14:28:37 盈利求和：638.228734 亏损求和 ：-89.056961
胜率 65%


4. Period2H
2018/03/26 14:29:39 盈利次数：55 亏损次数 ：17
2018/03/26 14:29:39 盈利求和：500.722605 亏损求和 ：-39.249346
胜率 76%

5. Period4H
2018/03/26 14:29:59 盈利次数：29 亏损次数 ：9
2018/03/26 14:29:59 盈利求和：502.975330 亏损求和 ：-29.038794

6. Period1Day
2018/03/26 14:30:23 盈利次数：10 亏损次数 ：1
2018/03/26 14:30:23 盈利求和：574.010105 亏损求和 ：-10.860438


[btc]
1.Period5Min
2018/03/26 14:31:37 盈利次数：1109 亏损次数 ：652
2018/03/26 14:31:37 盈利求和：839.472408 亏损求和 ：-202.113165
胜率 62%

2. Period15Min
2018/03/26 14:32:35 盈利次数：387 亏损次数 ：237
2018/03/26 14:32:35 盈利求和：504.385143 亏损求和 ：-136.847186
胜率 62%

3.Period30Min
2018/03/26 14:33:01 盈利次数：219 亏损次数 ：129
2018/03/26 14:33:01 盈利求和：411.806138 亏损求和 ：-113.672792
胜率 63%

4. Period2H
2018/03/26 14:33:24 盈利次数：56 亏损次数 ：27
2018/03/26 14:33:24 盈利求和：294.995289 亏损求和 ：-47.830697

5.Period4H
2018/03/26 14:33:44 盈利次数：36 亏损次数 ：11
2018/03/26 14:33:44 盈利求和：237.889534 亏损求和 ：-20.047389

5. Period1Day
2018/03/26 14:34:06 盈利次数：6 亏损次数 ：2
2018/03/26 14:34:06 盈利求和：226.815438 亏损求和 ：-6.503738
*/

func TestGetKline(t *testing.T) {

	// date1 := time.Date(2017, 8, 10, 0, 0, 0, 0, time.Local)
	date2 := time.Date(2018, 4, 1, 0, 0, 0, 0, time.Local)

	polo := new(PoloniexAPI)
	// result := polo.GetKline("eth/usdt", date1, &date2, Period5Min)
	var result []KlineValue

	filename := "poloniex-2hour"

	if true {
		result = polo.GetKline("eth/usdt", date2, nil, Period2H)
		SaveHistory(filename, result)
	} else {
		result = LoadHistory(filename)
	}

	StrategyTrendTest(result, true, true)
}

func TestMapArray(t *testing.T) {
	array := []KlineValue{
		KlineValue{
			OpenTime: 1,
			High:     100,
		},
	}
	maps := make(map[string][]KlineValue)

	maps["test"] = array

	log.Printf("1. Array:%v", maps["test"])

	maps["test"][0].OpenTime = 3
	maps["test"][0].High = 3

	log.Printf("2. Array:%v", maps["test"])

	maps["test"] = append(maps["test"], KlineValue{
		OpenTime: 2,
		High:     200,
	})

	log.Printf("3. Array:%v", maps["test"])

}
