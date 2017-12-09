package main

import (
	"fmt"
	"os"

	Http "madaoQT/http"
	Task "madaoQT/task"
	Utils "madaoQT/utils"

	"github.com/kataras/golog"
)

var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
}

func handleCmd() {
	var cmd string
	for {
		fmt.Scanln(&cmd)
		switch cmd {
		case "q":
			Logger.Info("Exiting...")
			os.Exit(0)
		}
	}

}

const constOKEXApiKey = "a982120e-8505-41db-9ae3-0c62dd27435c"
const constOEXSecretKey = "71430C7FA63A067724FB622FB3031970"

func main() {

	go handleCmd()

	analyzer := new(Task.IAnalyzer)
	analyzer.Init(nil)

	http := new(Http.HttpServer)
	go http.SetupHttpServer()
	go Utils.OpenBrowser("http://localhost:8080")

	for {
		select {
		case event := <-analyzer.WatchEvent():
			if event.EventType == Task.EventTypeTrigger {
				http.BroadcastByWebsocket(event.Msg)
			}
		}
	}

}
