package mongo

import (
	"errors"
	"log"
	"net"
	"strings"

	"golang.org/x/net/proxy"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type ExchangeInfo struct {
	Name string
	/* User password encrypted */
	API []byte
	/* User password encrypted */
	Secret []byte
	/* corresponding to the user in the users database */
	User string
}

type ExchangeDB struct {
	session    *mgo.Session
	collection *mgo.Collection

	Server     string
	Sock5Proxy string
}

func (p *ExchangeDB) Connect() error {

	if p.Sock5Proxy == "" {
		session, err := mgo.Dial(MongoURL)
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
	c := p.session.DB(Database).C(ExchangeCollection)

	p.collection = c

	return nil
}

func (t *ExchangeDB) Close() {
	if t.session != nil {
		t.session.Close()
	}
}

func (t *ExchangeDB) Insert(record *ExchangeInfo) error {
	if t.session != nil {
		err := t.collection.Insert(record)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("Connection is lost")
}

func (t *ExchangeDB) FindOne(name string) (error, *ExchangeInfo) {
	result := &ExchangeInfo{}
	if t.session != nil {
		err := t.collection.Find(bson.M{"name": name}).One(&result)
		if err != nil {
			return err, nil
		}

		return nil, result
	}

	return errors.New("Connection is lost"), nil
}
