package server

import (
	"sync"
	"time"

	"github.com/kataras/iris/view"

	"github.com/gorilla/securecookie"
	"github.com/kataras/golog"
	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	"github.com/kataras/iris/sessions"

	Global "madaoQT/config"
	Exchange "madaoQT/exchange"
	Controllers "madaoQT/server/controllers"
	Websocket "madaoQT/server/websocket"
	Utils "madaoQT/utils"

	// task
	OkexDiff "madaoQT/task/okexdiff"
	Trend "madaoQT/task/trend"
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
		// "/profit": "profit.html",
		"/test": "test.html",
	}

	for k, _ := range routers {
		h.app.Get(k, func(ctx iris.Context) {
			if err := ctx.View(routers[ctx.Path()]); err != nil {
				ctx.StatusCode(iris.StatusInternalServerError)
				ctx.Writef(err.Error())
			}
		})
	}

	h.app.OnErrorCode(iris.StatusNotFound, func(ctx iris.Context) {
		ctx.View("index.html")
	})
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

	// prefix := "/api/v1/"
	prefix := ""
	mvc.New(h.app.Party(prefix + "helloworld")).Handle(new(Controllers.HelloWorldController))
	mvc.New(h.app.Party(prefix + "charts")).Handle(new(Controllers.ChartsController))
	mvc.New(h.app.Party(prefix + "user")).Handle(&Controllers.UserController{Sessions: h.sess})
	mvc.New(h.app.Party(prefix + "task")).Handle(&Controllers.TaskController{Sessions: h.sess, Tasks: h.Tasks})
	mvc.New(h.app.Party(prefix + "exchange")).Handle(&Controllers.ExchangeController{Sessions: h.sess, Exchanges: h.exchanges})

}

func (h *HttpServer) SetupHttpServer() {

	h.app = iris.New()

	// Websocket.SetupWebsocket(app)
	h.ws = new(Websocket.WebsocketServer)
	h.ws.SetupWebsocket(h.app)

	var views *view.HTMLEngine

	if Global.ProductionEnv {
		views = iris.HTML("./www/www", ".html").Binary(Asset, AssetNames)
		h.app.StaticEmbedded("/bower_components", "./www/bower_components", Asset, AssetNames)
		h.app.StaticEmbedded("/elements", "./www/elements", Asset, AssetNames)
		h.app.StaticEmbedded("/images", "./www/images", Asset, AssetNames)
		h.app.StaticEmbedded("/assets", "./www/assets", Asset, AssetNames)

	} else {
		views = iris.HTML("./www/www", ".html")
		views.Reload(true) //开发模式，强制每次请求都更新页面
		h.app.StaticWeb("/bower_components", "./www/bower_components")
		h.app.StaticWeb("/elements", "./www/elements")
		h.app.StaticWeb("/images", "./www/images")
		h.app.StaticWeb("/assets", "./www/assets")

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

	okexdiff := new(OkexDiff.IAnalyzer)
	h.Tasks.Store(okexdiff.GetDescription().Name, okexdiff)

	trend := new(Trend.TrendTask)
	h.Tasks.Store(trend.GetDescription().Name, trend)

}
