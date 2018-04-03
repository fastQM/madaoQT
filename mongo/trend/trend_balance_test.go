package mongotrend

import (
	"log"
	"testing"
)

func TestSave(t *testing.T) {

	db := TrendMongo{
		BalanceCollectionName: "balanceTest",
	}
	if err := db.Connect(); err != nil {
		log.Printf("Invalid mongodatabase,%v", err)
		return
	}

	eth := BalanceInfo{
		Coin:    "eth",
		Balance: 100,
	}

	usdt := BalanceInfo{
		Coin:    "usdt",
		Balance: 1000,
	}

	var balances Balance
	balances.Item = make([]BalanceInfo, 2)
	balances.Item[0] = eth
	balances.Item[1] = usdt
	db.BalanceCollection.Insert(balances)

}
