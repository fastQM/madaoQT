package main

import (
	"testing"
	utils "madaoqt/utils"
)

func TestCreateServer(t *testing.T) {
	createServer();
	utils.SleepAsyncBySecond(30)
}