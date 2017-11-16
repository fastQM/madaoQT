package mongo

import "testing"

func TestLoadingCharts(t *testing.T) {
	mongo := new(Charts);
	err := mongo.Connect();
	if err == nil {
		mongo.LoadCharts("Poloniex", "USDT-ETH", 15);
	}
}