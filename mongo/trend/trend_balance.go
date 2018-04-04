package mongotrend

import (
	"time"

	mgo "gopkg.in/mgo.v2"
)

type BalanceInfo struct {
	Coin    string  `json:"coin"`
	Balance float64 `json:"balance"`
}

type Balance struct {
	Time time.Time     `json:"time"`
	Item []BalanceInfo `json:"item"`
	// Start 资金起始节点
	Start bool `json:"start"`
	// Ration 资金使用率
	Ratio float64 `json:"ratio"`
}

type Balances struct {
	collection *mgo.Collection
}

func (p *Balances) LoadCollection(collection *mgo.Collection) {
	p.collection = collection
}

func (p *Balances) Insert(record Balance) error {
	record.Time = time.Now()
	err := p.collection.Insert(record)
	if err != nil {
		return err
	}
	return nil
}

func (p *Balances) FindAll() (error, []Balance) {
	var result []Balance
	err := p.collection.Find(nil).All(&result)
	if err != nil {
		return err, nil
	}
	return nil, result

}
