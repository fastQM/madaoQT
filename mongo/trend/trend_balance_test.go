package mongotrend

import (
	"log"
	"testing"
)

func TestSave(t *testing.T) {

	db := TrendMongo{
		BalanceCollectionName: "balanceTest",
		Sock5Proxy:            "SOCKS5:127.0.0.1:1080",
	}
	if err := db.Connect(); err != nil {
		log.Printf("Invalid mongodatabase,%v", err)
		return
	}

	eth := BalanceItemInfo{
		Coin:    "eth",
		Balance: 100,
	}

	usdt := BalanceItemInfo{
		Coin:    "usdt",
		Balance: 1000,
	}

	var balances BalanceInfo
	balances.Item = make([]BalanceItemInfo, 2)
	balances.Item[0] = eth
	balances.Item[1] = usdt
	db.BalanceCollection.Insert(balances)

}
