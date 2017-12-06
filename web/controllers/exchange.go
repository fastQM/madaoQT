package controllers

import (
	"github.com/kataras/iris/mvc"
	"github.com/kataras/iris/sessions"
)

type ExchangeController struct {
	mvc.C
	Sessions *sessions.Sessions `iris:"persistence"`

	// [ Your fields here ]
	// Request lifecycle data
	// Models
	// Database
	// Global properties
}

func (c *ExchangeController) authen() {
	s := c.Sessions.Start(c.Ctx)
	name := s.Get("name")

	if name == nil || name == "" {
		c.Ctx.Redirect("/")
	}
}

//
// GET: /exchange

func (c *ExchangeController) Get() string {
	c.authen()
	return "This is my default action..."
}
