package mongotrend

import (
	"log"

	Mongo "madaoQT/mongo"

	mgo "gopkg.in/mgo.v2"
)

const TrendDataBase = "TrendDB"

type TrendMongo struct {
	session *mgo.Session

	FundCollectionName string
	FundCollection     Funds

	BalanceCollectionName string
	BalanceCollection     Balances
}

func (p *TrendMongo) Connect() error {

	session, err := mgo.Dial(Mongo.MongoURL)
	if err != nil {
		log.Printf("Connect to Mongo error:%v", err)
		return err
	}

	session.SetMode(mgo.Monotonic, true)
	p.session = session

	if p.FundCollectionName == "" {
		log.Printf("FundCollectionName is not assgined, and the collection is not valid")
	} else {
		p.FundCollection.LoadCollection(p.AddCollection(p.FundCollectionName))
	}

	if p.BalanceCollectionName == "" {
		log.Printf("BalanceCollectionName is not assgined, and the collection is not valid")
	} else {
		p.BalanceCollection.LoadCollection(p.AddCollection(p.BalanceCollectionName))
	}

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
