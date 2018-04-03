package mongotrend

import (
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const FundStatusOpen = "open"
const FundStatusClose = "close"
const FundStatusError = "error"

type FundInfo struct {
	Batch     string    `json:"batch"`
	Pair      string    `json:"pair"`
	OpenTime  time.Time `json:"opentime"`
	CloseTime time.Time `json:"closetime"`
	Status    string    `json:"status"`

	FutureType   string  `json:"futuretype"`
	FutureOpen   float64 `json:"futureopen"`
	FutureClose  float64 `json:"futureclose"`
	FutureAmount float64 `json:"futureamount"`
}

type Funds struct {
	collection *mgo.Collection
}

func (p *Funds) LoadCollection(collection *mgo.Collection) {
	p.collection = collection
}

func (p *Funds) Insert(record *FundInfo) error {
	err := p.collection.Insert(record)
	if err != nil {
		return err
	}
	return nil
}

func (p *Funds) Update(conditions map[string]interface{}, update map[string]interface{}) error {

	_, err := p.collection.UpdateAll(bson.M(conditions), bson.M{"$set": bson.M(update)})
	if err != nil {
		return err
	}
	return nil

}

func (p *Funds) GetDailySuccessRecords(date time.Time) (error, []FundInfo) {
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

	err = p.collection.Find(bson.M{
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

func (t *Funds) Find(conditions map[string]interface{}) (error, []FundInfo) {
	var result []FundInfo

	err := t.collection.Find(bson.M(conditions)).All(&result)
	if err != nil {
		return err, nil
	}

	return nil, result

}
