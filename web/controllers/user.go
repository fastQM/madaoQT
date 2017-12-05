package controllers

import (
	"fmt"

    "github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	
	"github.com/kataras/iris/sessions"
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
	Name string
	Password string
}

func (c *UserController) PostLogin() string {

	info := UserControllerLoginInfo{}
	err := c.Ctx.ReadForm(&info)
	if err != nil {
		c.Ctx.StatusCode(iris.StatusInternalServerError)
		return err.Error()
	}
	
	Logger.Debugf("Info:%v", info)

	s := c.Sessions.Start(c.Ctx)	
	s.Set("name", info.Name)
	s.Set("password", info.Password)
    return "This is my default action..."
}

func (c *UserController) GetBy(name string) string {
    return "Hello " + name
}

func (c *UserController) GetInfo() string {
	s := c.Sessions.Start(c.Ctx)	
	name := s.Get("name")
	password := s.Get("password")
	return fmt.Sprintf("name:%s, password:%s", name, password)
}

//
// GET: /helloworld/welcome/{name:string}/{numTimes:int}

func (c *UserController) GetWelcomeBy(name string, numTimes int) {
    // Access to the low-level Context,
    // output arguments are optional of course so we don't have to use them here.
    c.Ctx.Writef("Hello %s, NumTimes is: %d", name, numTimes)
}