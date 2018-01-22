package controllers

import (
	"github.com/kataras/golog"
	Global "madaoQT/config"
)

type errorCode int

const (
	errorCodeSuccess errorCode = iota
	errorCodeInvalidSession
	errorCodeInvalidParameters
	errorCodeAPINotSet
	errorCodeMongoDisconnect
)

var errorMessage = map[errorCode]string{
	errorCodeSuccess:           "success",
	errorCodeInvalidSession:    "Invalid session",
	errorCodeInvalidParameters: "Invalid parameters",
	errorCodeAPINotSet:         "API isn`t set",
	errorCodeMongoDisconnect:   "mongodb is not connected",
}

var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.SetTimeFormat(Global.TimeFormat)
}
