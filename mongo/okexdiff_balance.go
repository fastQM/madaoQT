package mongo

import (
	"errors"
	"fmt"
	"time"

	mgo "gopkg.in/mgo.v2"
)

type CoinInfo struct {
	Coin    string  `json:"coin"`
	Balance float64 `json:"balance"`
}

type BalanceInfo struct {
	Time  time.Time  `json:"time"`
	Coins []CoinInfo `json:"coins"`
}

type Balances struct {
	session    *mgo.Session
	collection *mgo.Collection

	Config *DBConfig
}

var defaultOrderDBConfig = &DBConfig{
	CollectionName: BalancesCollection,
}

func (t *Balances) Connect() error {
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

func (t *Balances) Close() {
	if t.session != nil {
		t.session.Close()
	}
}

func (t *Balances) Insert(record BalanceInfo) error {
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

func (t *Balances) FindAll() (error, []BalanceInfo) {
	var result []BalanceInfo
	if t.session != nil {
		err := t.collection.Find(nil).All(&result)
		if err != nil {
			return err, nil
		}

		return nil, result
	}

	return errors.New("Connection is lost"), nil
}
