package tradeCheck


import (
	"testing"
	Mongo "madaoqt/mongo"

)

func TestAnalyze(t *testing.T){
	tokenName := "USDT-ETH"
	mongo := new(Mongo.Charts);
	err := mongo.Connect();
	if err == nil {
		err = mongo.LoadCharts("Poloniex", tokenName, 15);
		if err == nil{
			analyzer := new(Analyzer)
			analyzer.Init(tokenName, mongo.Charts, 0)
			analyzer.Analyze()
		}
	}


}