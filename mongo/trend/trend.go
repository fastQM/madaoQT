package mongotrend

import (
	"errors"
	"log"
	"net"
	"strings"

	Mongo "madaoQT/mongo"

	"golang.org/x/net/proxy"
	mgo "gopkg.in/mgo.v2"
)

const TrendDataBase = "TrendDB"

type TrendMongo struct {
	session *mgo.Session

	FundCollectionName string
	FundCollection     Funds

	BalanceCollectionName string
	BalanceCollection     Balances

	Sock5Proxy string
}

func (p *TrendMongo) Connect() error {

	if p.Sock5Proxy == "" {
		session, err := mgo.Dial(Mongo.MongoURL)
		if err != nil {
			log.Printf("Connect to Mongo error:%v", err)
			return err
		}
		p.session = session
	} else {

		dialInfo, err := mgo.ParseURL(Mongo.MongoServer)
		if err != nil {
			return err
		}

		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			values := strings.Split(p.Sock5Proxy, ":")
			if values[0] == "SOCKS5" {
				dialer, err := proxy.SOCKS5("tcp", values[1]+":"+values[2], nil, proxy.Direct)
				if err != nil {
					return nil, err
				}

				log.Printf("Server:%v", addr)
				return dialer.Dial("tcp", addr.String())
			}

			return nil, errors.New("Invalid protocal")
		}

		session, err := mgo.DialWithInfo(dialInfo)
		if err != nil {
			return err
		}
		p.session = session
	}

	p.session.SetMode(mgo.Monotonic, true)

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
