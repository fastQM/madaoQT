package mongo

import (
	"errors"
	"fmt"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type DiffValue struct {
	Coin         string    `json:"coin"`
	SpotPrice    float64   `json:"spotprice"`
	SpotVolume   float64   `json:"spotvolume"`
	FuturePrice  float64   `json:"futureprice"`
	FutureVolume float64   `json:"futurevolume"`
	Diff         float64   `json:"diff"`
	Time         time.Time `json:"time"`
}

type OKExDiff struct {
	session    *mgo.Session
	collection *mgo.Collection

	Config *DBConfig
}

var defaultOKExDiffDBConfig = &DBConfig{
	CollectionName: OkexDiffHistory,
}

func (t *OKExDiff) Connect() error {
	session, err := mgo.Dial(MongoURL)
	if err != nil {
		fmt.Println("Connect to Mongo error", err)
		return err
	}
	session.SetMode(mgo.Monotonic, true)
	if t.Config == nil {
		t.Config = defaultOKExDiffDBConfig
	}

	c := session.DB(Database).C(t.Config.CollectionName)

	t.session = session
	t.collection = c

	return nil
}

func (t *OKExDiff) Close() {
	if t.session != nil {
		t.session.Close()
		t.session = nil
	}
}

func (t *OKExDiff) Insert(record DiffValue) error {
	if t.session != nil {
		err := t.collection.Insert(record)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("Connection is lost")
}

func (t *OKExDiff) FindAll(filter map[string]interface{}) ([]DiffValue, error) {
	var result []DiffValue
	if t.session != nil {
		err := t.collection.Find(bson.M(filter)).All(&result)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return nil, errors.New("Connection is lost")
}
