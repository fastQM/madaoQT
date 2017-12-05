package mongo

import (
	"fmt"

	mgo "gopkg.in/mgo.v2"
)

type ExchangeRecord struct {
	Name   string
	API    string
	Secret string
	/* corresponding to the user in the users database */
	User string
}

type Exchange struct {
}

func (t *Exchange) Connect() error {
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
