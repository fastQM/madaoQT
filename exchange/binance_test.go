package exchange

import (
	"log"
	"testing"
)

func TestBinanceStreamTrade(t *testing.T) {
	binance := new(Binance)

	result := binance.GetDepthValue("eth/usdt")
	log.Printf("Result:%v", result)

}
