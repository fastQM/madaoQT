package mongo

import (
	"errors"
	"fmt"
	"time"

	mgo "gopkg.in/mgo.v2"
	//   bson "gopkg.in/mgo.v2/bson"
)

type TradesRecord struct {
	Batch    string    `json:"batch"`
	Time     time.Time `json:"time"`
	Oper     string    `json:"oper"`
	Exchange string    `json:"exchange"`
	Coin     string    `json:"coin"`
	Price    float64   `json:"price"`
	Quantity float64   `json:"quantity"`
	OrderID  string    `json:"orderid"`
	Details  string    `json:"detail"`
}

type Trades struct {
	session    *mgo.Session
	collection *mgo.Collection

	Config *DBConfig
}

var defaultTradeDBConfig = &DBConfig{
	CollectionName: TradeRecordCollection,
}

func (t *Trades) Connect() error {
	session, err := mgo.Dial(MongoURL)
	if err != nil {
		fmt.Println("Connect to Mongo error", err)
		return err
	}
	session.SetMode(mgo.Monotonic, true)
	if t.Config == nil {
		t.Config = defaultTradeDBConfig
	}

	c := session.DB(Database).C(t.Config.CollectionName)

	t.session = session
	t.collection = c

	return nil
}

func (t *Trades) Close() {
	if t.session != nil {
		t.session.Close()
	}
}

func (t *Trades) Insert(record *TradesRecord) error {
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

func (t *Trades) FindAll() (error, []TradesRecord) {
	var result []TradesRecord
	if t.session != nil {
		err := t.collection.Find(nil).All(&result)
		if err != nil {
			return err, nil
		}
		return nil, result
	}

	return errors.New("Connection is lost"), nil
}
