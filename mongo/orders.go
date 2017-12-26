package mongo

import (
	"errors"
	"fmt"
	"time"

	mgo "gopkg.in/mgo.v2"
)

// type OrderItem struct {
// 	Pair          string  `json:"pair"`
// 	Trigger       string  `json:"trigger"`
// 	SellLimitHigh float64 `json:"sellhigh"`
// 	SellLimitLow  float64 `json:"selllow"`
// 	BuyLimitHigh  float64 `json:"buyhigh"`
// 	BuyLimitLow   float64 `json:"buylow"`
// 	// priority
// }

type OrderInfo struct {
	Batch    string    `json:"batch"`
	Time     time.Time `json:"time"`
	Exchange string    `json:"exchange"`
	Coin     string    `json:"coin"`
	OrderID  string    `json:"orderid"`
	Status   string    `json:"status"`
	Details  string    `json:"detail"`
}

type Orders struct {
	session    *mgo.Session
	collection *mgo.Collection

	Config *DBConfig
}

var defaultOrderDBConfig = &DBConfig{
	CollectionName: OrderCollection,
}

func (t *Orders) Connect() error {
	session, err := mgo.Dial(MongoURL)
	if err != nil {
		fmt.Println("Connect to Mongo error", err)
		return err
	}
	session.SetMode(mgo.Monotonic, true)
	if t.Config == nil {
		t.Config = defaultOrderDBConfig
	}

	c := session.DB(Database).C(t.Config.CollectionName)

	t.session = session
	t.collection = c

	return nil
}

func (t *Orders) Close() {
	if t.session != nil {
		t.session.Close()
	}
}

func (t *Orders) Insert(record *OrderInfo) error {
	if t.session != nil {
		record.Time = time.Now()
		err := t.collection.Insert(record)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (t *Orders) FindAll() (error, []OrderInfo) {
	var result []OrderInfo
	if t.session != nil {
		err := t.collection.Find(nil).All(&result)
		if err != nil {
			return err, nil
		}

		return nil, result
	}

	return errors.New("Connection is lost"), nil
}
