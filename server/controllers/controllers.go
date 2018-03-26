package controllers

import (
	Global "madaoQT/config"

	"github.com/kataras/golog"
)

type errorCode int

const (
	errorCodeSuccess errorCode = iota
	errorCodeInvalidSession
	errorCodeInvalidParameters
	errorCodeAPINotSet
	errorCodeMongoDisconnect
	errorCodeTaskNotRunning
)

var errorMessage = map[errorCode]string{
	errorCodeSuccess:           "success",
	errorCodeInvalidSession:    "Invalid session",
	errorCodeInvalidParameters: "Invalid parameters",
	errorCodeAPINotSet:         "API isn`t set",
	errorCodeMongoDisconnect:   "mongodb is not connected",
	errorCodeTaskNotRunning:    "Task is not running",
}

var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.SetTimeFormat(Global.TimeFormat)
}
