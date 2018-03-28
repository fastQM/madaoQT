package mongotrend

import (
	"log"

	Mongo "madaoQT/mongo"

	mgo "gopkg.in/mgo.v2"
)

const TrendDataBase = "TrendDB"
const TrendFundsCollectionName = "TrendFunds2"

type TrendMongo struct {
	session        *mgo.Session
	FundCollection Funds
}

func (p *TrendMongo) Connect() error {
	session, err := mgo.Dial(Mongo.MongoURL)
	if err != nil {
		log.Printf("Connect to Mongo error:%v", err)
		return err
	}
	session.SetMode(mgo.Monotonic, true)
	p.session = session

	p.FundCollection.LoadCollection(p.AddCollection(TrendFundsCollectionName))

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
