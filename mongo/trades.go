package mongo

import (
  "fmt"
  "time"

  mgo "gopkg.in/mgo.v2"
//   bson "gopkg.in/mgo.v2/bson"
)

type TradesRecord struct {
	Time time.Time
	Oper	string	// buy,sell
	Exchange string
	Coin string
	Quantity float64
	OrderID string
	Details string
}

type Trades struct {
	session *mgo.Session
	collection *mgo.Collection
}

func (t *Trades) Connect() error {
	session, err := mgo.Dial(MongoURL)
	if err != nil {
		fmt.Println("Connect to redis error", err)
		return err
	}
	session.SetMode(mgo.Monotonic, true)
	c := session.DB(Database).C(TradeRecordCollection)

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
		err := t.collection.Insert(record)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

