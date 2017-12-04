package web

import (
    "time"

	"github.com/kataras/iris/sessions"
    "github.com/kataras/iris"
    "github.com/kataras/golog"
    "github.com/gorilla/securecookie"

	Config "madaoQT/config"
	controllers "madaoQT/web/controllers"
    websocket "madaoQT/web/websocket"
    Utils "madaoQT/utils"
)

type HttpServer struct {
	app *iris.Application
    ws  *websocket.WebsocketServer
    sess *sessions.Sessions
}

const CookiesName = "madao-sessions"

var Logger *golog.Logger


func init(){
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.Info("Web package init() finished")
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

func (h *HttpServer)setupSessions() {

    cookieName := CookiesName
	// AES only supports key sizes of 16, 24 or 32 bytes.
	// You either need to provide exactly that amount or you derive the key from what you type in.
	hashKey := []byte(Utils.GetRandomHexString(32))
	blockKey := []byte(Utils.GetRandomHexString(32))
	secureCookie := securecookie.New(hashKey, blockKey)

	h.sess = sessions.New(sessions.Config{
		Cookie: cookieName,
		Encode: secureCookie.Encode,
        Decode: secureCookie.Decode,
        Expires: time.Minute * 10,
	})
}

func (h *HttpServer)setupControllers() {

    h.app.Controller("/helloworld", new(controllers.HelloWorldController))
    h.app.Controller("/user", &controllers.UserController{Sessions: h.sess})
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
    
    h.setupSessions()
    h.setupRoutes()
    h.setupControllers()

	h.app.Run(iris.Addr(":8080"))
}

func (h *HttpServer) BroadcastByWebsocket(msg interface{}) {
	h.ws.BroadcastAll(msg)
}
