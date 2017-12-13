package websocket

import (
	"fmt"
	// "log"

	"github.com/kataras/iris"
	"github.com/kataras/iris/websocket"

	Exchange "madaoQT/exchange"
)

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

func (w *WebsocketServer) BroadcastAll(msg interface{}) {

	connections := w.ws.GetConnections()
	if connections != nil && len(connections) != 0 {
		connections[0].To(websocket.All).Emit("chat", msg)
	}

	return
}

func (w *WebsocketServer) Ticker(exchange string, tickerValue Exchange.TickerValue) {
	w.BroadcastAll(fmt.Sprintf("%v", tickerValue))
}

func (w *WebsocketServer) handleConnection(c websocket.Connection) {

	// Read events from browser
	c.On("chat", func(msg string) {
		// Print the message to the console, c.Context() is the iris's http context.
		fmt.Printf("%s sent: %s\n", c.Context().RemoteAddr(), msg)
		// Write message back to the client message owner:
		// c.Emit("chat", msg)
		c.To(websocket.Broadcast).Emit("chat", msg)
	})

	c.On(MsgCmdPublish, func(msg string) {
		// not allowed by connections
	})

	c.On(MsgCmdSubscribe, func(msg string) {

	})
}
