package web

import (
    "github.com/kataras/iris"
	
    controllers "madaoQT/web/controllers"
    websocket "madaoQT/web/websocket"
)

type HttpServer struct {
    app *iris.Application
    ws *websocket.WebsocketServer
}

func (h *HttpServer)SetupHttpServer() {
	
    h.app = iris.New()

    // websocket.SetupWebsocket(app)
    h.ws = new(websocket.WebsocketServer)
    h.ws.SetupWebsocket(h.app)

    views := iris.HTML("./views", ".html")
    // views.Reload(true)  //开发模式，强制每次请求都更新页面
    views.Binary(Asset, AssetNames)
    
    h.app.RegisterView(views)
    
    h.app.Controller("/helloworld", new(controllers.HelloWorldController))

    h.app.Get("/", func(ctx iris.Context) {
        // Bind: {{.message}} with "Hello world!"
        // ctx.ViewData("message", "Hello world!")
        // Render template file: ./views/hello.html
        // ctx.View("websockets.html")
        if err := ctx.View("websockets.html"); err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Writef(err.Error())
		}
    })

    h.app.Run(iris.Addr(":8080"))
}

func (h *HttpServer)BroadcastByWebsocket(msg interface{}){
    h.ws.BroadcastAll(msg)
}