package controllers

import (
	"github.com/kataras/golog"
)

type errorCode int

const (
	errorCodeSuccess errorCode = iota
	errorCodeInvalidSession
	errorCodeInvalidParameters
)

var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.SetTimeFormat("2006-01-02 06:04:05")
}
