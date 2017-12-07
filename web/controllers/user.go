package controllers

import (
	"fmt"

	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"

	"github.com/kataras/iris/sessions"

	Mongo "madaoQT/mongo"
)

type UserController struct {
	mvc.C
	Sessions *sessions.Sessions `iris:"persistence"`
	// [ Your fields here ]
	// Request lifecycle data
	// Models
	// Database
	// Global properties
}

type UserControllerLoginInfo struct {
	Name     string
	Password string
	LastPage string
}

type ExchangeInfo struct {
	Name      string
	APIKey    string
	SecretKey string
}

// basic user interfaces

func (c *UserController) GetLoginBy(lastPage string) mvc.Result {
	Logger.Debugf("Last Page:%s", lastPage)
	return mvc.View{
		Name: "login.html",
		Data: map[string]interface{}{
			"lastPage": lastPage,
		},
	}
}

func (c *UserController) PostLogin() iris.Map {

	var errMsg string
	var session *sessions.Session
	info := UserControllerLoginInfo{}
	err := c.Ctx.ReadForm(&info)
	if err != nil {
		c.Ctx.StatusCode(iris.StatusInternalServerError)
		errMsg = err.Error()
		goto _ERROR
	}

	Logger.Debugf("Info:%v", info)

	session = c.Sessions.Start(c.Ctx)
	session.Set("name", info.Name)
	session.Set("password", info.Password)

	return iris.Map{
		"result": true,
		"last":   info.LastPage,
	}

_ERROR:
	return iris.Map{
		"result": false,
		"error":  errMsg,
	}
}

func (c *UserController) GetInfo() string {
	s := c.Sessions.Start(c.Ctx)
	name := s.Get("name")
	// password := s.Get("password")
	// return fmt.Sprintf("name:%s, password:%s", name, password)
	return fmt.Sprintf("Hellow, %s!", name)
}

// user's exchanges interfaces

// POST: /user/exchange/addkey

func (c *UserController) PostExchangeAddkey() iris.Map {
	var errMsg string
	var err error
	s := c.Sessions.Start(c.Ctx)
	username := s.Get("name")

	exchangesDB := Mongo.ExchangeDB{}

	info := ExchangeInfo{}

	if username == nil {
		errMsg = "Invalid Session"
		goto _ERROR
	}

	err = c.Ctx.ReadForm(&info)
	if err != nil {
		errMsg = err.Error()
		goto _ERROR
	}

	Logger.Debugf("KEY:%v", info)

	if err = exchangesDB.Connect(); err != nil {
		errMsg = err.Error()
		goto _ERROR
	}

	if err = exchangesDB.Insert(&Mongo.ExchangeRecord{
		Name:   info.Name,
		API:    info.APIKey,
		Secret: info.SecretKey,
		User:   username.(string),
	}); err != nil {
		errMsg = "Fail to insert record into database"
		goto _ERROR
	}

	return iris.Map{
		"result": true,
	}

_ERROR:
	return iris.Map{
		"result": false,
		"error":  errMsg,
	}
}

//
// GET: /helloworld/welcome/{name:string}/{numTimes:int}

func (c *UserController) GetWelcomeBy(name string, numTimes int) {
	// Access to the low-level Context,
	// output arguments are optional of course so we don't have to use them here.
	c.Ctx.Writef("Hello %s, NumTimes is: %d", name, numTimes)
}
