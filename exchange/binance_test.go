package exchange

import (
	"crypto/hmac"
	"crypto/sha256"
	"io"
	"log"
	"testing"
	"time"
)

func TestBinanceStreamTrade(t *testing.T) {
	binance := new(Binance)

	result := binance.GetDepthValue("eth/usdt")
	log.Printf("Result:%v", result)

}

func TestGetUnixTime(t *testing.T) {

	binance := new(Binance)
	binance.SetConfigure(Config{
		Proxy: "SOCKS5:127.0.0.1:1080",
	})

	for {
		select {
		case <-time.After(5 * time.Second):
			kline := binance.GetKline("eth/usdt", KlinePeriod5Min, 50)
			length := len(kline)
			if length != 0 {
				log.Printf("kline:%f %d", kline[length-1].OpenTime, time.Now().Unix())
				log.Printf("kline:%s current:%s", time.Unix(int64(kline[length-1].OpenTime), 0).String(), time.Now().String())
			}

		}
	}

}

func TestPeriodArea(t *testing.T) {

	binance := new(Binance)
	binance.SetConfigure(Config{
		Proxy: "SOCKS5:127.0.0.1:1080",
	})

	kline := binance.GetKline("eth/usdt", KlinePeriod5Min, 500)

	length := len(kline)
	array10 := kline[length-10 : length]
	array20 := kline[length-20 : length]

	avg10 := GetAverage(10, array10)
	avg20 := GetAverage(20, array20)

	var isOpenLong bool
	if avg10 > avg20 {
		isOpenLong = true
	} else {
		isOpenLong = false
	}

	var start int
	found := false
	if isOpenLong {

		step := 0
		for i := len(kline) - 1; i >= 0; i-- {
			array10 := kline[i-10 : i]
			array20 := kline[i-20 : i]

			avg10 := GetAverage(10, array10)
			avg20 := GetAverage(20, array20)

			if step == 0 {
				if avg10 < avg20 {
					step = 1
					continue
				}
			} else if step == 1 {
				if avg10 > avg20 {
					step = 2
					continue
				}
			} else if step == 2 {
				if avg10 < avg20 {
					start = i
					found = true
					break
				}
			}
		}

	} else {
		step := 0
		for i := len(kline) - 1; i >= 0; i-- {
			array10 := kline[i-10 : i]
			array20 := kline[i-20 : i]

			avg10 := GetAverage(10, array10)
			avg20 := GetAverage(20, array20)

			if step == 0 {
				if avg10 > avg20 {
					step = 1
					continue
				}
			} else if step == 1 {
				if avg10 < avg20 {
					step = 2
					continue
				}
			} else if step == 2 {
				if avg10 > avg20 {
					start = i
					found = true
					break
				}
			}
		}
	}

	if found {
		var high, low float64
		log.Printf("Start is %v", time.Unix(int64(kline[start].OpenTime), 0))
		for i := start; i < len(kline)-1; i++ {
			if high == 0 {
				high = kline[i].High
			} else if high < kline[i].High {
				high = kline[i].High
			}

			if low == 0 {
				low = kline[i].Low
			} else if low > kline[i].Low {
				low = kline[i].Low
			}
		}

	}

}

func TestGetKlines(t *testing.T) {

	binance := new(Binance)
	binance.SetConfigure(Config{
		Proxy: "SOCKS5:127.0.0.1:1080",
	})

	result := binance.GetKline("eth/usdt", KlinePeriod5Min, 500)
	// log.Printf("Result:%v", result)
	StrategyTrendTest(result)
}
func TestCheckBottomSupport(t *testing.T) {
	var logs []string
	binance := new(Binance)
	binance.SetConfigure(Config{
		Proxy: "SOCKS5:127.0.0.1:1080",
	})

	result := binance.GetKline("eth/usdt", KlinePeriod5Min, 500)

	logs = CheckBottomSupport("eth", result)

	for _, msg := range logs {
		log.Printf("Log:%s", msg)
	}
}

func TestKlineRatio(t *testing.T) {

	// var logs []string

	binance := new(Binance)
	binance.SetConfigure(Config{
		Proxy: "SOCKS5:127.0.0.1:1080",
	})

	result := binance.GetKline("eth/usdt", KlinePeriod5Min, 500)

	var lastRatio float64

	pre10 := result[0:10]
	preAvg10 := GetAverage(10, pre10)

	for i := 10; i <= len(result)-1; i++ {
		array10 := result[i-9 : i+1]
		avg10 := GetAverage(10, array10)

		ratio := (avg10 - preAvg10) / 1
		log.Printf("[%s] Ratio:%.2f", time.Unix(int64(result[i].OpenTime), 0).String(), ratio)
		lastRatio = ratio

		// 发生逆转，重新选择起点
		if ratio > 0 && lastRatio < 0 {

		} else if ratio < 0 && lastRatio > 0 {

		}
	}
}

func TestSha256(t *testing.T) {
	h := hmac.New(sha256.New, []byte("NhqPtmdSJYdKjVHjA7PZj4Mge3R5YNiP1e3UZjInClVN65XAbvqqM6A7H5fATj0j"))
	io.WriteString(h, "symbol=LTCBTC&side=BUY&type=LIMIT&timeInForce=GTC&quantity=1&price=0.1&recvWindow=5000&timestamp=1499827319559")
	log.Printf("%x", h.Sum(nil))
}

func TestGetBalance(t *testing.T) {
	err, key := GetExchangeKey(NameBinance)
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	binance := new(Binance)
	binance.SetConfigure(Config{
		API:    key.API,
		Secret: key.Secret,
		Proxy:  "SOCKS5:127.0.0.1:1080",
	})

	log.Printf("BALANCES:%v", binance.GetBalance())
}

func TestTrade(t *testing.T) {
	err, key := GetExchangeKey(NameBinance)
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	binance := new(Binance)
	binance.SetConfigure(Config{
		API:    key.API,
		Secret: key.Secret,
		Proxy:  "SOCKS5:127.0.0.1:1080",
	})

	result := binance.Trade(TradeConfig{
		Pair:   "eth/usdt",
		Type:   TradeTypeSell,
		Amount: 0.05,
		Price:  408,
	})

	log.Printf("result:%v", result)
}

func TestGetOrderInfo(t *testing.T) {
	err, key := GetExchangeKey(NameBinance)
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	binance := new(Binance)
	binance.SetConfigure(Config{
		API:    key.API,
		Secret: key.Secret,
		Proxy:  "SOCKS5:127.0.0.1:1080",
	})

	result := binance.GetOrderInfo(OrderInfo{
		Pair:    "eth/usdt",
		OrderID: "A8LQi9x4zPQDiiJ2dbVlwp",
	})

	log.Printf("Reulst:%v", result)

}
