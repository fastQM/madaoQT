package redis

import "testing"

func TestLoadCharts(t *testing.T) {
	conn := new(ChartsHistory)

	conn.Connect()

	conn.LoadCharts("charts-poloniex-BTC-ETH", 1)
}
