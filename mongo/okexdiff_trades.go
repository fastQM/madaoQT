package mongo

import (
	"errors"
	"fmt"
	"time"

	mgo "gopkg.in/mgo.v2"
	bson "gopkg.in/mgo.v2/bson"
)

const TradeStatusOpen = "open"
const TradeStatusCanceled = "canceled"
const TradeStatusDone = "done"

type TradesRecord struct {
	Batch    string    `json:"batch"`
	Time     time.Time `json:"time"`
	Oper     string    `json:"oper"`
	Exchange string    `json:"exchange"`
	Pair     string    `json:"pair"`
	Price    float64   `json:"price"`
	Quantity float64   `json:"quantity"`
	OrderID  string    `json:"orderid"`
	Status   string    `json:"status"`
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

func (t *Trades) Refresh() {
	if t.session != nil {
		t.session.Refresh()
	}
}

func (t *Trades) Insert(record *TradesRecord) error {
	if t.session != nil {
		record.Time = time.Now()
		record.Status = TradeStatusOpen
		err := t.collection.Insert(record)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (t *Trades) SetCanceled(orderID string) error {
	return t.updateStatus(orderID, TradeStatusCanceled)
}

func (t *Trades) SetDone(orderID string) error {
	return t.updateStatus(orderID, TradeStatusDone)
}

func (t *Trades) updateStatus(orderID string, status string) error {
	if t.session != nil {
		_, err := t.collection.UpdateAll(bson.M{"orderid": orderID}, bson.M{"$set": bson.M{"status": status}})
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
