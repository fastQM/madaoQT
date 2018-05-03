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

	TradeCollectionName string
	TradeCollection     Trades

	BalanceCollectionName string
	BalanceCollection     BalanceStruct

	FundCollectionName string
	FundCollection     Funds

	Server     string
	Sock5Proxy string
}

func (p *TrendMongo) Connect() error {

	if p.Sock5Proxy == "" {
		session, err := mgo.Dial(p.Server)
		if err != nil {
			log.Printf("Connect to Mongo error:%v", err)
			return err
		}
		p.session = session
	} else {

		dialInfo, err := mgo.ParseURL(p.Server)
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

	if p.TradeCollectionName == "" {
		log.Printf("TradeCollectionName is not assgined, and the collection is not valid")
	} else {
		p.TradeCollection.LoadCollection(p.AddCollection(p.TradeCollectionName))
	}

	if p.BalanceCollectionName == "" {
		log.Printf("BalanceCollectionName is not assgined, and the collection is not valid")
	} else {
		p.BalanceCollection.LoadCollection(p.AddCollection(p.BalanceCollectionName))
	}

	if p.FundCollectionName == "" {
		log.Printf("FundCollectionName is not assgined, and the collection is not valid")
	} else {
		p.FundCollection.LoadCollection(p.AddCollection(p.FundCollectionName))
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
