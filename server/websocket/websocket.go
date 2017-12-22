package websocket

import (
	"fmt"
	// "log"

	"github.com/kataras/golog"
	"github.com/kataras/iris"
	"github.com/kataras/iris/websocket"

	Exchange "madaoQT/exchange"
)

var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.SetTimeFormat("2006-01-02 06:04:05")
	Logger.SetPrefix("[SOCK]")
}

type WebsocketServer struct {
	ws *websocket.Server
}

func (w *WebsocketServer) SetupWebsocket(app *iris.Application) {
	// create our echo websocket server
	w.ws = websocket.New(websocket.Config{
	// ReadBufferSize:  1024,
	// WriteBufferSize: 1024,
	})

	w.ws.OnConnection(w.handleConnection)

	// register the server on an endpoint.
	// see the inline javascript code in the websockets.html, this endpoint is used to connect to the server.
	app.Get("/websocket", w.ws.Handler())

	// serve the javascript built'n client-side library,
	// see weboskcets.html script tags, this path is used.
	app.Any("/iris-ws.js", func(ctx iris.Context) {
		ctx.Write(websocket.ClientSource)
	})
}

func (w *WebsocketServer) Broadcast(room string, msg interface{}) {

	connections := w.ws.GetConnectionsByRoom(room)
	if connections != nil && len(connections) > 0 {
		connections[0].To(room).EmitMessage([]byte(msg.(string)))
	}
}

func (w *WebsocketServer) Ticker(exchange string, tickerValue Exchange.TickerValue) {
	w.Broadcast("", fmt.Sprintf("%v", tickerValue))
}

func (w *WebsocketServer) Publish(topic string, msg string) {
	w.Broadcast(topic, msg)
}

func (w *WebsocketServer) handleConnection(c websocket.Connection) {

	c.OnMessage(func(msg []byte) {
		Logger.Debugf("recv:%s from:%s", msg, c.ID())
		data := ParseRequestMsg(string(msg))
		if data != nil {
			if data.Cmd == CmdTypeSubscribe && data.Data != nil {
				topic := data.Data.(map[string]interface{})["topic"].(string)
				c.Join(topic)
				rsp := PackageResponseMsg(data.Seq, true, ErrorTypeNone, nil)
				Logger.Debugf("Response:%s", rsp)
				c.To(c.ID()).EmitMessage(rsp)

			} else if data.Cmd == CmdTypeUnsubscribe && data.Data != nil {
				topic := data.Data.(map[string]interface{})["topic"].(string)
				c.Leave(topic)
				rsp := PackageResponseMsg(data.Seq, true, ErrorTypeNone, nil)
				Logger.Debugf("Response:%s", rsp)
				c.To(c.ID()).EmitMessage(rsp)

			} else if data.Cmd == CmdTypePublish {
				topic := data.Topic
				if connections := w.ws.GetConnectionsByRoom(data.Topic); connections == nil && len(connections) > 0 {
					Logger.Debug("room not found")
				} else {
					// Logger.Debugf("Room:%v", connections[0])
					c.To(topic).EmitMessage([]byte(msg))
				}

			} else {
				goto __INVALID_CMD
			}

			return

		}

	__INVALID_CMD:
		rsp := PackageResponseMsg(data.Seq, false, ErrorTypeInvalidCmd, nil)
		c.To(c.ID()).EmitMessage(rsp)
		return
	})
}
