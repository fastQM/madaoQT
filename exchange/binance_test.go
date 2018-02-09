package exchange

import (
	"testing"
)

func TestBinanceStreamTrade(t *testing.T) {
	binance := new(Binance)

	binance.Start()

	select {}

}
