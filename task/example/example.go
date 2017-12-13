package task

import (
	Websocket "github.com/gorilla/websocket"
)

const websocketServer = "ws://localhost:8080/websocket"

type ChangeDetect struct {
	// websocket *Websocket.conne
}

func (c *ChangeDetect) Start() {
	c.connectTicker()
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
}
