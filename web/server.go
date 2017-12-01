package web

import (
    "github.com/kataras/iris"
    "github.com/kataras/golog"

	Config "madaoQT/config"
	controllers "madaoQT/web/controllers"
	websocket "madaoQT/web/websocket"
)

type HttpServer struct {
	app *iris.Application
	ws  *websocket.WebsocketServer
}

var Logger *golog.Logger

func init(){
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.Info("Rules package init() finished")
}

func (h *HttpServer)setupRoutes(){

    routers := map[string]string{
        "/": "index.html",
        "/login": "login.html",
        "/test": "test.html",
    }

    for k,_ := range routers {
        h.app.Get(k, func(ctx iris.Context) {
            if err := ctx.View(routers[ctx.Path()]); err != nil {
                ctx.StatusCode(iris.StatusInternalServerError)
                ctx.Writef(err.Error())
            }
        })
    }
}

func (h *HttpServer)setupControllers() {

    h.app.Controller("/helloworld", new(controllers.HelloWorldController))
}

func (h *HttpServer)SetupHttpServer() {
	
    h.app = iris.New()

    // websocket.SetupWebsocket(app)
    h.ws = new(websocket.WebsocketServer)
    h.ws.SetupWebsocket(h.app)

    views := iris.HTML("./www/www", ".html")
    views.Reload(true)  //开发模式，强制每次请求都更新页面
    

    if Config.PRODUCTION_ENV {
        // h.app.StaticEmbedded("/static", "./views/node_modules", Asset, AssetNames)

    } else {
        h.app.StaticWeb("/bower_components", "./www/bower_components")
        h.app.StaticWeb("/elements", "./www/elements")
        h.app.StaticWeb("/images", "./www/images")

    }
    
    h.app.RegisterView(views)
    
    h.setupControllers()
    h.setupRoutes()

	h.app.Run(iris.Addr(":8080"))
}

func (h *HttpServer) BroadcastByWebsocket(msg interface{}) {
	h.ws.BroadcastAll(msg)
}
