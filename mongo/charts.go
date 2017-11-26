package mongo

import (
	"fmt"
	"errors"
	"log"

	mgo "gopkg.in/mgo.v2"
	bson "gopkg.in/mgo.v2/bson"
)

type Charts struct {
	session *mgo.Session
	collection *mgo.Collection

	Charts []ChartItem
}

type ChartItem struct {
	Name			string `json:"name"`
	Date            int64   `json:"date"`
	Hm              string  `json:"hm"`
	High            float64 `json:"high"`
	Low             float64 `json:"low"`
	Open            float64 `json:"open"`
	Close           float64 `json:"close"`
	Volume          float64 `json:"volume"`
	QuoteVolume     float64 `json:"quoteVolume"`
	WeightedAverage float64 `json:"weightedAverage"`
	Exchange		string `json:"exchange"`
}

func (t *Charts) Connect() error {
	session, err := mgo.Dial(MongoURL)
	if err != nil {
		fmt.Println("Connect to redis error", err)
		return err
	}
	session.SetMode(mgo.Monotonic, true)
	c := session.DB(Database).C(ChartCollectin)

	t.session = session
	t.collection = c

	return nil
}

func (t *Charts) Close() {
	if t.session != nil {
		t.session.Close()
	}
}

func (t *Charts) LoadCharts(exchange string, name string, period int) error {

	if t.collection == nil {
		return errors.New("Mongo is not connected")
	}

	t.Charts = []ChartItem{}
	t.collection.Find(bson.M{"exchange": exchange, "name": name}).All(&t.Charts);

	for i := 0; i < len(t.Charts); i++{
		log.Printf("chart:%v",t.Charts[i])
	}

	return nil;
}