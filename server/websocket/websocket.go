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

func (w *WebsocketServer) Broadcast(topic string, msg interface{}) {

	connections := w.ws.GetConnections()
	if connections != nil && len(connections) != 0 {
		if topic == "" {
			connections[0].To(websocket.All).Emit("chat", msg)
		} else {
			connections[0].To(topic).Emit(topic, msg)
		}
	}

	return
}

func (w *WebsocketServer) Ticker(exchange string, tickerValue Exchange.TickerValue) {
	w.Broadcast("", fmt.Sprintf("%v", tickerValue))
}

func (w *WebsocketServer) Publish(topic string, msg string) {
	w.Broadcast(topic, msg)
}

func (w *WebsocketServer) handleConnection(c websocket.Connection) {

	// Read events from browser
	// c.On("chat", func(msg string) {
	// 	// Print the message to the console, c.Context() is the iris's http context.
	// 	fmt.Printf("%s sent: %s\n", c.Context().RemoteAddr(), msg)
	// 	// Write message back to the client message owner:
	// 	// c.Emit("chat", msg)
	// 	c.To(websocket.Broadcast).Emit("chat", msg)
	// })

	// c.On(MsgCmdPublish, func(msg string) {
	// 	Logger.Debugf("recv publish msg:%s", msg)
	// 	data := parseRequestMsg(msg)
	// 	if data != nil {
	// 		rsp := packageResponseMsg(data.Seq, true, ErrorTypeNone, nil)
	// 		Logger.Debugf("Response:%s", rsp)
	// 		c.To(c.ID()).Emit(MsgCmdPublish, string(rsp)) // 发送方相应
	// 		c.To(data.Cmd).Emit(data.Cmd, data.Data)      // 订阅方发送
	// 		return
	// 	}
	// })

	c.OnMessage(func(msg []byte) {
		Logger.Debugf("recv message:%s", msg)
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
				channel := data.Channel
				c.To(channel).EmitMessage([]byte(data.Data.(string)))

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
