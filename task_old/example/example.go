package main

import (
	"time"

	"github.com/kataras/golog"

	Message "madaoQT/server/websocket"

	Websocket "github.com/gorilla/websocket"
)

const websocketServer = "ws://localhost:8080/websocket"

var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.SetTimeFormat("2006-01-02 06:04:05")
}

type ChangeDetect struct {
	conn *Websocket.Conn
}

func (c *ChangeDetect) Start() {
	c.connectTicker()
	for {
		select {
		case <-time.After(3 * time.Second):
			msg := Message.PackageRequestMsg(0, Message.CmdTypePublish, "iamhere", "je;;p")
			Logger.Infof("Send:%v", string(msg))
			c.conn.WriteMessage(Websocket.TextMessage, []byte(msg))
		}
	}
}

func (c *ChangeDetect) connectTicker() {

	conn, _, err := Websocket.DefaultDialer.Dial(websocketServer, nil)
	if err != nil {
		Logger.Errorf("Fail to dial: %v", err)
		return
	}

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				Logger.Errorf("Fail to read:%v", err)
				return
			}

			Logger.Infof("message:%v", string(message))
		}
	}()

	c.conn = conn
}

func main() {
	task := ChangeDetect{}
	task.Start()
}
