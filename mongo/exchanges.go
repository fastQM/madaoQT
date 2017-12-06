package mongo

import (
	"fmt"

	mgo "gopkg.in/mgo.v2"
)

type ExchangeRecord struct {
	Name string
	/* User password encrypted */
	API string
	/* User password encrypted */
	Secret string
	/* corresponding to the user in the users database */
	User string
}

type Exchange struct {
	session    *mgo.Session
	collection *mgo.Collection
}

func (t *Exchange) Connect() error {
	session, err := mgo.Dial(MongoURL)
	if err != nil {
		fmt.Println("Connect to redis error", err)
		return err
	}
	session.SetMode(mgo.Monotonic, true)
	c := session.DB(Database).C(ExchangeCollection)

	t.session = session
	t.collection = c

	return nil
}

func (t *Exchange) Insert(record *ExchangeRecord) error {
	if t.session != nil {
		err := t.collection.Insert(record)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}
