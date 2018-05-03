package mongotrend

import (
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const ActionAdd = "add"
const ActionRemove = "remove"

type FundInfo struct {
	ID       bson.ObjectId `bson:"_id,omitempty"`
	Name     string        `json:"name"`
	Action   string        `json:"action"`
	Quantity float64       `json:"quantity"`
	Owner    string        `json:"owner"`
	Price    float64       `json:"price"`
	Date     time.Time     `json:"date"`
}

type Funds struct {
	collection *mgo.Collection
}

func (p *Funds) LoadCollection(collection *mgo.Collection) {
	p.collection = collection
}

func (p *Funds) Insert(record *FundInfo) error {
	err := p.collection.Insert(record)
	if err != nil {
		return err
	}
	return nil
}

func (p *Funds) FindAll(conditions map[string]interface{}) (error, []FundInfo) {
	var result []FundInfo
	err := p.collection.Find(bson.M(conditions)).All(&result)
	if err != nil {
		return err, nil
	}
	return nil, result

}

func (p *Funds) Update(conditions map[string]interface{}, update map[string]interface{}) error {

	_, err := p.collection.UpdateAll(bson.M(conditions), bson.M{"$set": bson.M(update)})
	if err != nil {
		return err
	}
	return nil

}
