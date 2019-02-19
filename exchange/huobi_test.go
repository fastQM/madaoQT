package exchange

import (
	"log"
	"testing"
	"time"
)

const HuobiAPI = ""
const HuobiSecret = ""

func TestHuobiGetBalance(t *testing.T) {
	huobi := Huobi{
		InstrumentType: InstrumentTypeSpot,
		Proxy:          "SOCKS5:127.0.0.1:1080",
		ApiKey:         HuobiAPI,
		SecretKey:      HuobiSecret,
	}

	log.Printf("Balance:%v", huobi.GetBalance())
}

func TestHuobiTrade(t *testing.T) {
	huobi := Huobi{
		InstrumentType: InstrumentTypeSpot,
		Proxy:          "SOCKS5:127.0.0.1:1080",
		ApiKey:         HuobiAPI,
		SecretKey:      HuobiSecret,
	}

	result := huobi.Trade(TradeConfig{
		Amount: 0.1,
		Pair:   "eth/usdt",
		Type:   TradeTypeSell,
	})

	log.Printf("Result:%v", result)
}

func TestHuobiGetOrderInfo(t *testing.T) {
	huobi := Huobi{
		InstrumentType: InstrumentTypeSpot,
		Proxy:          "SOCKS5:127.0.0.1:1080",
		ApiKey:         HuobiAPI,
		SecretKey:      HuobiSecret,
	}

	log.Printf("result:%v", huobi.GetOrderInfo(OrderInfo{
		OrderID: "21620207770",
	}))
}

func TestHuobiGetKlines(t *testing.T) {
	huobi := Huobi{
		InstrumentType: InstrumentTypeSpot,
		Proxy:          "SOCKS5:127.0.0.1:1080",
		ApiKey:         HuobiAPI,
		SecretKey:      HuobiSecret,
	}
	location, _ := time.LoadLocation("Asia/Shanghai")
	klines := huobi.GetKline("eth/usdt", KlinePeriod1Hour, 200)
	for _, kline := range klines {
		log.Printf("TIme:%v %v", time.Unix(int64(kline.OpenTime), 0).In(location), kline)
	}
}

func TestHuobiGetDepth(t *testing.T) {

	huobi := Huobi{
		InstrumentType: InstrumentTypeSpot,
		Proxy:          "SOCKS5:127.0.0.1:1080",
	}

	eventChan := make(chan EventType)
	go func() {
		huobi.Start2(eventChan)
	}()

	counter := 5
	status := false

	for {
		select {
		case event := <-eventChan:
			if event == EventConnected {
				log.Printf("connected")
				status = true
			} else if event == EventLostConnection {
				log.Printf("The connection lost")
			}
		case <-time.After(1 * time.Second):
			if status {
				value := huobi.GetDepthValue("eth/usdt")
				log.Printf("Value:%v", value)
				if counter > 0 {
					counter--
				} else {
					return
				}
			}

		}
	}
}

func TestHuobiDMGetDepth(t *testing.T) {

	huobi := Huobi{
		InstrumentType: InstrumentTypeSwap,
		Proxy:          "SOCKS5:127.0.0.1:1080",
	}

	eventChan := make(chan EventType)
	go func() {
		huobi.Start2(eventChan)
	}()

	counter := 5
	status := false

	for {
		select {
		case event := <-eventChan:
			if event == EventConnected {
				log.Printf("connected")
				status = true
			} else if event == EventLostConnection {
				log.Printf("The connection lost")
			}
		case <-time.After(1 * time.Second):
			if status {
				value := huobi.GetDepthValue("BTC_CQ")
				log.Printf("Value:%v", value)
				if counter > 0 {
					counter--
				} else {
					return
				}
			}

		}
	}
}

func TestHuobiDMGetKlines(t *testing.T) {
	huobi := Huobi{
		InstrumentType: InstrumentTypeSwap,
		Proxy:          "SOCKS5:127.0.0.1:1080",
	}
	location, _ := time.LoadLocation("Asia/Shanghai")
	klines := huobi.GetKline("BTC_CQ", KlinePeriod1Hour, 200)
	for _, kline := range klines {
		log.Printf("TIme:%v %v", time.Unix(int64(kline.OpenTime), 0).In(location), kline)
	}
}
