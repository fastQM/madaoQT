package task

import (
	"encoding/json"
	"log"
	Mongo "madaoQT/mongo"
	"testing"
)

func _TestConfigToJson(t *testing.T) {
	configJSON := "{\"area\":{\"ltc\":{\"open\":3, \"close\":1.5}}, \"limitclose\":0.03, \"limitopen\":0.005}"

	var config AnalyzerConfig
	err := json.Unmarshal([]byte(configJSON), &config)
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	log.Printf("Config:%v", config)
}

func TestRecordBalances(t *testing.T) {

	okexdiff := new(IAnalyzer)

	coins := []string{
		"eth", "ltc", "btc",
	}

	balanceDB := new(Mongo.Balances)
	if err := balanceDB.Connect(); err != nil {
		Logger.Errorf("Fail to connect BalanceDB:%v", err)
		return
	}
	defer balanceDB.Close()

	var coinInfos Mongo.BalanceInfo
	balances := okexdiff.GetBalances()
	if balances != nil {
		for _, coin := range coins {
			var coinInfo Mongo.CoinInfo
			coinInfo.Coin = coin
			for _, v := range balances["spots"].([]map[string]interface{}) {
				if v["name"] == coin {
					coinInfo.Balance += v["balance"].(float64)
					break
				}
			}

			for _, v := range balances["futures"].([]map[string]interface{}) {
				if v["name"] == coin {
					coinInfo.Balance += v["balance"].(float64)
					coinInfo.Balance += v["bond"].(float64)
					break
				}
			}

			coinInfos.Coins = append(coinInfos.Coins, coinInfo)
		}

		for _, v := range balances["spots"].([]map[string]interface{}) {
			var coinInfo Mongo.CoinInfo
			coinInfo.Coin = "usdt"
			if v["name"] == "usdt" {
				coinInfo.Balance += v["balance"].(float64)
				break
			}

			coinInfos.Coins = append(coinInfos.Coins, coinInfo)
		}
	}

	Logger.Infof("Balances:%v", coinInfos)

}
