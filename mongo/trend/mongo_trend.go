package mongotrend

import (
	"errors"
	"log"

	Mongo "madaoQT/mongo"

	mgo "gopkg.in/mgo.v2"
)

const TrendDataBase = "TrendDB"

type TrendMongo struct {
	session            *mgo.Session
	FundCollection     Funds
	FundCollectionName string
}

func (p *TrendMongo) Connect() error {

	if p.FundCollectionName == "" {
		return errors.New("Please assign the collection name")
	}

	session, err := mgo.Dial(Mongo.MongoURL)
	if err != nil {
		log.Printf("Connect to Mongo error:%v", err)
		return err
	}

	session.SetMode(mgo.Monotonic, true)
	p.session = session

	p.FundCollection.LoadCollection(p.AddCollection(p.FundCollectionName))

	return nil
}

func (p *TrendMongo) AddCollection(name string) *mgo.Collection {

	if p.session != nil {
		c := p.session.DB(TrendDataBase).C(name)
		return c
	}

	log.Printf("Error:%s", Mongo.ErrorNotConnected)
	return nil

}

func (p *TrendMongo) Disconnect() {
	if p.session != nil {
		p.session.Close()
	}
}

func (p *TrendMongo) Refresh() {
	if p.session != nil {
		p.session.Refresh()
	}
}
