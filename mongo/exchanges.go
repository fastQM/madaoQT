package mongo

import (
	"errors"
	"fmt"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type ExchangeInfo struct {
	Name string
	/* User password encrypted */
	API string
	/* User password encrypted */
	Secret string
	/* corresponding to the user in the users database */
	User string
}

type ExchangeDB struct {
	session    *mgo.Session
	collection *mgo.Collection
}

func (t *ExchangeDB) Connect() error {
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

func (t *ExchangeDB) Close() {
	if t.session != nil {
		t.session.Close()
	}
}

func (t *ExchangeDB) Insert(record *ExchangeInfo) error {
	if t.session != nil {
		err := t.collection.Insert(record)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (t *ExchangeDB) FindOne(name string) (error, *ExchangeInfo) {
	result := &ExchangeInfo{}
	if t.session != nil {
		err := t.collection.Find(bson.M{"name": name}).One(&result)
		if err != nil {
			return err, nil
		}

		return nil, result
	}

	return errors.New("Connection is lost"), nil
}
