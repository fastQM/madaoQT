package exchange

import (
	"log"
	"testing"
	"time"
)

const deribitkey = ""
const deribitsecret = ""

func TestDeribitGetDepth(t *testing.T) {

	deribit := DeribitV2API{
		Proxy: "SOCKS5:127.0.0.1:1080",
	}

	eventChannel := make(chan EventType)
	status := 0

	go func() {
		for {
			select {
			case <-time.After(1 * time.Second):
				if status == 1 {
					log.Printf("%v", deribit.GetDepthValue("btc/usdt"))
				}

			case event := <-eventChannel:
				if event == EventConnected {
					log.Printf("connected")
					status = 1
				} else if event == EventLostConnection {
					log.Printf("The connection lost")
					status = 0
				}
			}
		}
	}()

	if err := deribit.Start2(eventChannel); err != nil {
		log.Printf("Fail to start:%v", err)
		return
	}

	select {
	case <-time.After(15 * time.Second):
		return
	}
}

func TestDeribitAuthen(t *testing.T) {

	deribit := DeribitV2API{
		Proxy:     "SOCKS5:127.0.0.1:1080",
		ApiKey:    deribitkey,
		SecretKey: deribitsecret,
	}

	eventChannel := make(chan EventType)

	go func() {
		for {
			select {
			case event := <-eventChannel:
				if event == EventConnected {
					log.Printf("connected")
					log.Printf("Authen result:%v", deribit.Authen(false))
					_, balance := deribit.GetBalance2("ETH")
					log.Printf("Authen result:%v", balance)
				} else if event == EventLostConnection {
					log.Printf("The connection lost")
				}
			}
		}
	}()

	if err := deribit.Start2(eventChannel); err != nil {
		log.Printf("Fail to start:%v", err)
		return
	}

	select {
	case <-time.After(15 * time.Second):
		return
	}
}

func TestDeribitHeartBeat(t *testing.T) {

	deribit := DeribitV2API{
		Proxy:     "SOCKS5:127.0.0.1:1080",
		ApiKey:    deribitkey,
		SecretKey: deribitsecret,
	}

	eventChannel := make(chan EventType)

	go func() {
		for {
			select {
			case event := <-eventChannel:
				if event == EventConnected {
					log.Printf("connected")
					deribit.startHeartBeat()
				} else if event == EventLostConnection {
					log.Printf("The connection lost")
				}
			}
		}
	}()

	if err := deribit.Start2(eventChannel); err != nil {
		log.Printf("Fail to start:%v", err)
		return
	}

	select {
	case <-time.After(30 * time.Second):
		return
	}
}

func TestDeribitTrade(t *testing.T) {

	deribit := DeribitV2API{
		Proxy:     "SOCKS5:127.0.0.1:1080",
		ApiKey:    deribitkey,
		SecretKey: deribitsecret,
	}

	eventChannel := make(chan EventType)

	go func() {
		for {
			select {
			case event := <-eventChannel:
				if event == EventConnected {
					log.Printf("connected")
					log.Printf("Authen result:%v", deribit.Authen(false))
					tradeResult := deribit.Trade(TradeConfig{
						Pair:   "eth/usdt",
						Type:   TradeTypeCloseShort,
						Amount: 20,
					})
					log.Printf("tradeResult result:%v %v", tradeResult, tradeResult.Info)
				} else if event == EventLostConnection {
					log.Printf("The connection lost")
				}
			}
		}
	}()

	if err := deribit.Start2(eventChannel); err != nil {
		log.Printf("Fail to start:%v", err)
		return
	}

	select {
	case <-time.After(15 * time.Second):
		return
	}
}

func TestDeribitGetPosition(t *testing.T) {

	deribit := DeribitV2API{
		Proxy:     "SOCKS5:127.0.0.1:1080",
		ApiKey:    deribitkey,
		SecretKey: deribitsecret,
	}

	eventChannel := make(chan EventType)

	go func() {
		for {
			select {
			case event := <-eventChannel:
				if event == EventConnected {
					log.Printf("connected")
					log.Printf("Authen result:%v", deribit.Authen(false))
					if err, result := deribit.GetPosition("eth/usdt"); err == nil {
						log.Printf("result:%v", result)
					}

				} else if event == EventLostConnection {
					log.Printf("The connection lost")
				}
			}
		}
	}()

	if err := deribit.Start2(eventChannel); err != nil {
		log.Printf("Fail to start:%v", err)
		return
	}

	select {
	case <-time.After(15 * time.Second):
		return
	}
}
