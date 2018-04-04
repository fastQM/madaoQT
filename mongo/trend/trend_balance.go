package mongotrend

import (
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type BalanceItemInfo struct {
	Coin    string  `json:"coin"`
	Balance float64 `json:"balance"`
}

type BalanceInfo struct {
	ID   bson.ObjectId     `bson:"_id,omitempty"`
	Time time.Time         `json:"time"`
	Item []BalanceItemInfo `json:"item"`
	// Start 资金起始节点
	Start bool `json:"start"`
	// Ration 资金使用率
	Ratio float64 `json:"ratio"`
}

type BalanceStruct struct {
	collection *mgo.Collection
}

func (p *BalanceStruct) LoadCollection(collection *mgo.Collection) {
	p.collection = collection
}

func (p *BalanceStruct) Insert(record BalanceInfo) error {
	record.Time = time.Now()
	err := p.collection.Insert(record)
	if err != nil {
		return err
	}
	return nil
}

func (p *BalanceStruct) Update(conditions map[string]interface{}, update map[string]interface{}) error {

	_, err := p.collection.UpdateAll(bson.M(conditions), bson.M{"$set": bson.M(update)})
	if err != nil {
		return err
	}
	return nil

}

func (p *BalanceStruct) FindAll(conditions map[string]interface{}, sort string) (error, []BalanceInfo) {
	var result []BalanceInfo
	err := p.collection.Find(bson.M(conditions)).Sort(sort).All(&result)
	if err != nil {
		return err, nil
	}
	return nil, result

}
