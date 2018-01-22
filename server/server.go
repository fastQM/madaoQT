package server

import (
	"sync"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/kataras/golog"
	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	"github.com/kataras/iris/sessions"

	Global "madaoQT/config"
	Exchange "madaoQT/exchange"
	Controllers "madaoQT/server/controllers"
	Websocket "madaoQT/server/websocket"
	Task "madaoQT/task"
	Utils "madaoQT/utils"
)

type HttpServer struct {
	app       *iris.Application
	ws        *Websocket.WebsocketServer
	sess      *sessions.Sessions
	exchanges []Exchange.IExchange

	Tasks *sync.Map
}

const CookiesName = "madao-sessions"

var Logger *golog.Logger

func init() {
	logger := golog.New()
	Logger = logger
	Logger.SetLevel("debug")
	Logger.SetTimeFormat(Global.TimeFormat)
}

func (h *HttpServer) setupRoutes() {

	routers := map[string]string{
		"/":      "index.html",
		"/login": "login.html",
		"/test":  "test.html",
	}

	for k, _ := range routers {
		h.app.Get(k, func(ctx iris.Context) {
			if err := ctx.View(routers[ctx.Path()]); err != nil {
				ctx.StatusCode(iris.StatusInternalServerError)
				ctx.Writef(err.Error())
			}
		})
	}
}

func (h *HttpServer) setupSessions() {

	cookieName := CookiesName
	// AES only supports key sizes of 16, 24 or 32 bytes.
	// You either need to provide exactly that amount or you derive the key from what you type in.
	hashKey := []byte(Utils.GetRandomHexString(32))
	blockKey := []byte(Utils.GetRandomHexString(32))
	secureCookie := securecookie.New(hashKey, blockKey)

	h.sess = sessions.New(sessions.Config{
		Cookie:  cookieName,
		Encode:  secureCookie.Encode,
		Decode:  secureCookie.Decode,
		Expires: time.Minute * 10,
	})
}

func (h *HttpServer) setupControllers() {

	// h.app.Controller("/helloworld", new(Controllers.HelloWorldController))
	// h.app.Controller("/charts", new(Controllers.ChartsController))
	// h.app.Controller("/user", &Controllers.UserController{Sessions: h.sess})
	// h.app.Controller("/task", &Controllers.TaskController{Sessions: h.sess, Tasks: h.Tasks})
	// h.app.Controller("/exchange", &Controllers.ExchangeController{Sessions: h.sess, Exchanges: h.exchanges})
	mvc.New(h.app.Party("/helloworld")).Handle(new(Controllers.HelloWorldController))
	mvc.New(h.app.Party("/charts")).Handle(new(Controllers.ChartsController))
	mvc.New(h.app.Party("/user")).Handle(&Controllers.UserController{Sessions: h.sess})
	mvc.New(h.app.Party("/task")).Handle(&Controllers.TaskController{Sessions: h.sess, Tasks: h.Tasks})
	mvc.New(h.app.Party("/exchange")).Handle(&Controllers.ExchangeController{Sessions: h.sess, Exchanges: h.exchanges})

}

func (h *HttpServer) SetupHttpServer() {

	h.app = iris.New()

	// Websocket.SetupWebsocket(app)
	h.ws = new(Websocket.WebsocketServer)
	h.ws.SetupWebsocket(h.app)

	views := iris.HTML("./www/www", ".html")
	views.Reload(true) //开发模式，强制每次请求都更新页面

	if Global.ProductionEnv {
		// h.app.StaticEmbedded("/static", "./views/node_modules", Asset, AssetNames)

	} else {
		h.app.StaticWeb("/bower_components", "./www/bower_components")
		h.app.StaticWeb("/elements", "./www/elements")
		h.app.StaticWeb("/images", "./www/images")

	}

	h.app.RegisterView(views)

	// task
	h.setupExchanges()
	h.setupTasks()

	// http
	h.setupSessions()
	h.setupRoutes()
	h.setupControllers()

	h.app.Run(iris.Addr(":8080"))
}

// func (h *HttpServer) BroadcastByWebsocket(msg interface{}) {
// 	h.ws.BroadcastAll(msg)
// }

func (h *HttpServer) setupExchanges() {
	okexspot := Exchange.NewOKExSpotApi(&Exchange.Config{
		Ticker: Exchange.ITicker(h.ws),
	})

	okexfuture := Exchange.NewOKExFutureApi(nil)

	h.exchanges = append(h.exchanges, okexspot)
	h.exchanges = append(h.exchanges, okexfuture)

	okexspot.Start()

	go func() {
		for {
			select {
			case event := <-okexspot.WatchEvent():
				if event == Exchange.EventConnected {
					okexspot.StartTicker("ltc/usdt")

				} else if event == Exchange.EventLostConnection {
					okexspot.Start()
				}
			}
		}
	}()
}

func (h *HttpServer) setupTasks() {

	if h.Tasks == nil {
		h.Tasks = &sync.Map{}
	}
	// load default task
	// h.Tasks.Store("okexdiff", &Task.Task{
	// 	Name: "okexdiff",
	// })
	tasks := Task.LoadStaticTask()
	for _, task := range tasks {
		h.Tasks.Store(task.GetTaskName(), task)
	}

}
