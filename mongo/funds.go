package mongo

import (
	"errors"
	"fmt"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const FundStatusOpen = "open"
const FundStatusOpened = "opened"
const FundStatusClose = "close"
const FundStatusClosed = "closed"

type FundInfo struct {
	Type         string    `json:"type"`
	Batch        string    `json:"batch"`
	OpenType     string    `json:"opentype"`
	Coin         string    `json:"coin"`
	OpenTime     time.Time `json:"opentime"`
	OpenBalance  float64   `json:"openbalance"`
	CloseTime    time.Time `json:"closetime"`
	CloseBalance float64   `json:"closebalance"`
	Exchange     string    `json:"exchange"`
	Status       string    `json:"status"`
}

type Funds struct {
	session    *mgo.Session
	collection *mgo.Collection
	Config     *DBConfig
}

var defaultFundDBConfig = &DBConfig{
	CollectionName: FundCollection,
}

func (t *Funds) Connect() error {
	session, err := mgo.Dial(MongoURL)
	if err != nil {
		fmt.Println("Connect to Mongo error", err)
		return err
	}
	session.SetMode(mgo.Monotonic, true)
	if t.Config == nil {
		t.Config = defaultFundDBConfig
	}

	c := session.DB(Database).C(t.Config.CollectionName)

	t.session = session
	t.collection = c

	return nil
}

func (t *Funds) Close() {
	if t.session != nil {
		t.session.Close()
	}
}

func (t *Funds) Insert(record *FundInfo) error {
	if t.session != nil {
		err := t.collection.Insert(record)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("Connection is lost")
}

func (t *Funds) Update(conditions map[string]interface{}, update map[string]interface{}) error {
	if t.session != nil {
		_, err := t.collection.UpdateAll(bson.M(conditions), bson.M{"$set": bson.M(update)})
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("Connection is lost")
}

func (t *Funds) FindAll() (error, []FundInfo) {
	var result []FundInfo
	if t.session != nil {
		err := t.collection.Find(nil).All(&result)
		if err != nil {
			return err, nil
		}

		return nil, result
	}

	return errors.New("Connection is lost"), nil
}
