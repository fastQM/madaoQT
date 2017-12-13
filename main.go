package main

import (
	"fmt"
	"os"
	"os/signal"

	Config "madaoQT/config"
	Server "madaoQT/server"
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

func main() {

	go handleCmd()

	http := new(Server.HttpServer)
	go http.SetupHttpServer()

	if Config.ProductionEnv {
		go Utils.OpenBrowser("http://localhost:" + Config.ServerPort)
	}

	kill := make(chan os.Signal, 1)
	signal.Notify(kill, os.Interrupt, os.Kill)

	for {
		select {
		// case event := <-analyzer.WatchEvent():
		// 	if event.EventType == Task.EventTypeTrigger {
		// 		http.BroadcastByWebsocket(event.Msg)
		// 	}
		case <-kill:
			Logger.Infof("interrupt")
			Utils.SleepAsyncBySecond(3)
			os.Exit(0)

		}
	}

}
