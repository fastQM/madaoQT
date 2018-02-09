package mongo

import (
	"errors"
	"fmt"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const FundStatusOpen = "open"
const FundStatusClose = "close"
const FundStatusError = "error"

// type FundInfo struct {
// 	Type         string    `json:"type"`
// 	Batch        string    `json:"batch"`
// 	OpenType     string    `json:"opentype"`
// 	Coin         string    `json:"coin"`
// 	OpenTime     time.Time `json:"opentime"`
// 	OpenBalance  float64   `json:"openbalance"`
// 	CloseTime    time.Time `json:"closetime"`
// 	CloseBalance float64   `json:"closebalance"`
// 	Exchange     string    `json:"exchange"`
// 	Status       string    `json:"status"`
// }

type FundInfo struct {
	Batch     string    `json:"batch"`
	Pair      string    `json:"pair"`
	OpenTime  time.Time `json:"opentime"`
	CloseTime time.Time `json:"closetime"`
	Status    string    `json:"status"`

	SpotType     string  `json:"spottype"`
	FutureType   string  `json:"futuretype"`
	SpotOpen     float64 `json:"spotopen"`
	SpotClose    float64 `json:"spotclose"`
	SpotAmount   float64 `json:"spotamount"`
	FutureOpen   float64 `json:"futureopen"`
	FutureClose  float64 `json:"futureclose"`
	FutureAmount float64 `json:"futureamount"`
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

func (t *Funds) Refresh() {
	if t.session != nil {
		t.session.Refresh()
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

func (t *Funds) GetDailySuccessRecords(date time.Time) (error, []FundInfo) {
	var result []FundInfo
	var start, end time.Time
	var err error

	format := "2006-01-02"
	temp := date.Format(format)
	start, err = time.Parse(format, temp)
	if err != nil {
		return err, nil
	}

	end = start.AddDate(0, 0, 1).Add(-8 * time.Hour)
	start = start.Add(-8 * time.Hour)

	// log.Printf("Start:%v End:%v", start.Format(config.TimeFormat), end.Format(config.TimeFormat))

	if t.session != nil {
		err := t.collection.Find(bson.M{
			"closetime": bson.M{
				"$gte": start,
				"$lt":  end,
			},
			"status": FundStatusClose,
		}).All(&result)
		if err != nil {
			return err, nil
		}

		return nil, result
	}

	return errors.New("Connection is lost"), nil
}

func (t *Funds) Find(conditions map[string]interface{}) (error, []FundInfo) {
	var result []FundInfo
	if t.session != nil {
		err := t.collection.Find(bson.M(conditions)).All(&result)
		if err != nil {
			return err, nil
		}

		return nil, result
	}

	return errors.New("Connection is lost"), nil
}
