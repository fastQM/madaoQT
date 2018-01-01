package exchange

import (
	Websocket "github.com/gorilla/websocket"
)

const EndPoint = "wss://stream.binance.com:9443/ws/bnbbtc@trade"

type Binance struct {
	websocket *Websocket.Conn
}

func (h *Binance) Start() {
	conn, _, err := Websocket.DefaultDialer.Dial(EndPoint, nil)
	if err != nil {
		logger.Errorf("Fail to dial: %v", err)
		// go h.triggerEvent(EventLostConnection)
		return
	}

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				logger.Errorf("Fail to read:%v", err)
				// go h.triggerEvent(EventError)
				return
			}

			logger.Debugf("message:%v", string(message))
		}
	}()

}

//
func (h *Binance) triggerEvent(event EventType) {

}
