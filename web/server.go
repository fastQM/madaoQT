package main

import (
    "github.com/kataras/iris"
	
    controllers "madaoqt/web/controllers"
    websocket "madaoqt/web/websocket"
)

func createServer() {
	
    app := iris.New()

    websocket.SetupWebsocket(app)

    app.RegisterView(iris.HTML("./views", ".html"))
    app.Controller("/helloworld", new(controllers.HelloWorldController))

    app.Get("/", func(ctx iris.Context) {
        // Bind: {{.message}} with "Hello world!"
        // ctx.ViewData("message", "Hello world!")
        // Render template file: ./views/hello.html
        ctx.View("websockets.html")
    })

    app.Run(iris.Addr(":8080"))
}

func main(){
    createServer()
}