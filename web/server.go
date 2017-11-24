package main

import (
    "github.com/kataras/iris"
	
    controllers "madaoqt/web/controllers"
    websocket "madaoqt/web/websocket"
)

func setupHttpServer() {
	
    app := iris.New()

    websocket.SetupWebsocket(app)

    // views := iris.HTML("./views", ".html")
    // views.Reload(true)  //开发模式，强制每次请求都更新页面

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
    setupHttpServer()
}