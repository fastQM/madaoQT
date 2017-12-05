package controllers

import (
	"github.com/kataras/golog"
)

var Logger *golog.Logger

func init(){
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.Info("Web package init() finished")
}