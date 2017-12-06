package mongo

import (
	"fmt"
	"time"

	mgo "gopkg.in/mgo.v2"
)

type UserRecord struct {
	Name string
	// SHA256
	Password string
	Email    string
	Time     time.Time
}

type User struct {
	session    *mgo.Session
	collection *mgo.Collection
}

func (t *User) Connect() error {
	session, err := mgo.Dial(MongoURL)
	if err != nil {
		fmt.Println("Connect to redis error", err)
		return err
	}
	session.SetMode(mgo.Monotonic, true)
	c := session.DB(Database).C(TradeRecordCollection)

	t.session = session
	t.collection = c

	return nil
}
