package rules

import (
	"github.com/kataras/golog"
)

/*
	初始化日志句柄
*/
var Logger *golog.Logger

func init(){
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.Info("Rules package init() finished")
}

type EventType int8

const (
	EventTypeError EventType = iota
	EventTypeTrigger
)

type RulesEvent struct {
	EventType EventType
	Msg interface{}
}