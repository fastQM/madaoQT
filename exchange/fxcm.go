package exchange

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	Global "madaoQT/config"

	socketio "github.com/graarh/golang-socketio"
	"github.com/graarh/golang-socketio/transport"
	"golang.org/x/net/proxy"
)

const ExchangeFXCM = "FXCM"

const WssFXCMUrl = "wss://api.fxcm.com"
const HttpFXCMUrl = "https://api.fxcm.com"

var shareFxcm *FXCM

// var once sync.Once

type FXCM struct {
	config Config
	event  chan EventType
	socket *socketio.Client
	mutex  sync.Mutex
}

// func GetFxcmInstance(config Config) *FXCM {

// 	shareFxcm = &FXCM{}
// 	shareFxcm.SetConfigure(config)
// 	if err := shareFxcm.Start(); err != nil {
// 		log.Printf("Fail to start fxcm instance:%v", err)
// 		shareFxcm = nil
// 	}

// 	return shareFxcm
// }

func (p *FXCM) GetExchangeName() string {
	return ExchangeFXCM
}

// SetConfigure()
func (p *FXCM) SetConfigure(config Config) {
	p.config = config

	if p.config.Proxy != "" {
		logger.Infof("使用代理:%s", p.config.Proxy)
	}
}

// WatchEvent() return a channel which notified the application of the event triggered by exchange
func (p *FXCM) WatchEvent() chan EventType {
	return p.event
}

// Close() close the connection to the exchange and other handles
func (p *FXCM) Close() {
	if p.socket != nil {
		p.socket.Close()
	}
}

// StartTicker() send message to the exchange to start the ticker of the given pairs
func (p *FXCM) StartTicker(pair string) {
}

// CancelOrder() cancel the order as the order information
func (p *FXCM) CancelOrder(order OrderInfo) *TradeResult {
	return nil
}

// GetOrderInfo() get the information with order filter
func (p *FXCM) GetOrderInfo(filter OrderInfo) []OrderInfo {
	return nil
}

func (p *FXCM) GetDepthValue(pair string) [][]DepthPrice {
	return nil
}

func (p *FXCM) Start() error {

	logger.Infof("启动socket")
	p.event = make(chan EventType)
	token := p.config.Custom["token"].(string)
	socket, err := socketio.Dial(WssFXCMUrl+"/socket.io/?EIO=3&transport=websocket&access_token="+token,
		transport.GetDefaultWebsocketTransport())
	if err != nil {
		return err
	}

	socket.On(socketio.OnConnection, func(c *socketio.Channel, args interface{}) {
		log.Printf("Socket Connected[%s]", c.Id())
		p.socket = socket
		go p.triggerEvent(EventConnected)
	})

	socket.On(socketio.OnDisconnection, func(c *socketio.Channel, args interface{}) {
		log.Printf("Socket Disconnected:%v", args)
		p.socket = nil
		go p.triggerEvent(EventLostConnection)
	})

	socket.On(socketio.OnError, func(c *socketio.Channel) {
		log.Printf("Error occurs")
		go p.triggerEvent(EventLostConnection)
	})

	return nil
}

func (o *FXCM) triggerEvent(event EventType) {
	o.event <- event
}

func (p *FXCM) marketRequest(method, path string, params map[string]string) (error, []byte) {

	if p.socket == nil {
		return errors.New("Socket is not connected"), nil
	}

	// log.Printf("Path:%s", path)

	var bodystr string
	for k, v := range params {
		if bodystr == "" {
			bodystr += (k + "=" + v)
		} else {
			bodystr += ("&" + k + "=" + v)
		}

	}
	// logger.Debugf("Params:%s auth[%s]", bodystr, "Bearer "+p.socket.Id()+p.config.Custom["token"].(string))

	p.mutex.Lock()
	defer p.mutex.Unlock()

	var request *http.Request
	var err error
	if method == "GET" {
		request, err = http.NewRequest(method, HttpFXCMUrl+path+"?"+string(bodystr), nil)
		if err != nil {
			return err, nil
		}
	} else if method == "POST" {
		request, err = http.NewRequest(method, HttpFXCMUrl+path, strings.NewReader(bodystr))
		if err != nil {
			return err, nil
		}
	}

	request.Header.Add("User-Agent", "request")
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Authorization", "Bearer "+p.socket.Id()+p.config.Custom["token"].(string))

	// setup a http client
	httpTransport := &http.Transport{}
	httpClient := &http.Client{Transport: httpTransport}

	if p.config.Proxy != "" {
		values := strings.Split(p.config.Proxy, ":")
		if values[0] == "SOCKS5" {
			dialer, err := proxy.SOCKS5("tcp", values[1]+":"+values[2], nil, proxy.Direct)
			if err != nil {
				return err, nil
			}

			httpTransport.Dial = dialer.Dial
		}

	}

	var resp *http.Response
	resp, err = httpClient.Do(request)
	if err != nil {
		return err, nil
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err, nil
	}

	keywords := []string{
		"candles",
		"pairs",
	}

	filtered := false

	for _, keyword := range keywords {
		if strings.Contains(string(body), keyword) {
			filtered = true
			break
		}
	}

	if !filtered {
		log.Printf("Body:%v", string(body))
	}

	// var value map[string]interface{}
	// if err = json.Unmarshal(body, &value); err != nil {
	// 	return err, nil
	// }

	return nil, body

}

// GetBalance() get the balances of all the coins
func (p *FXCM) GetBalance() map[string]interface{} {

	if err, response := p.marketRequest("GET", "/trading/get_model", map[string]string{
		"models": "Account",
	}); err != nil {
		logger.Errorf("无法获取余额:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		if values["response"] != nil && values["response"].(map[string]interface{})["executed"].(bool) != true {
			logger.Errorf("无法获取余额:%v", err)
			return nil
		}

		accounts := values["accounts"].([]interface{})
		for _, account := range accounts {
			if account.(map[string]interface{})["accountId"].(string) == p.config.Custom["account"].(string) {
				return map[string]interface{}{
					"balance": account.(map[string]interface{})["balance"].(float64),
				}
			}
		}

	}

	return nil
}

func (p *FXCM) GetTicker(pair string) *TickerValue {

	if err, response := p.marketRequest("POST", "/subscribe", map[string]string{
		"pairs": pair,
	}); err != nil {
		logger.Errorf("无法获取ticker值:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		if values["response"] != nil && values["response"].(map[string]interface{})["executed"].(bool) != true {
			logger.Errorf("无法获取ticker值:%v", err)
			return nil
		}

		pairs := values["pairs"].([]interface{})
		if pairs != nil && len(pairs) > 0 {
			rates := pairs[len(pairs)-1].(map[string]interface{})["Rates"].([]interface{})
			return &TickerValue{
				High: rates[2].(float64),
				Low:  rates[3].(float64),
				Last: (rates[0].(float64) + rates[1].(float64)) / 2,
				Time: time.Unix(int64(pairs[len(pairs)-1].(map[string]interface{})["Updated"].(float64)/1000), 0).Format(Global.TimeFormat),
			}
		}

		p.socket.On(pair, func(c *socketio.Channel, args interface{}) {
			log.Printf("[IGNORE]Values:%v", args)
		})

	}

	return nil
}

var FxcmTradeTypeMap = map[TradeType]string{
	TradeTypeOpenLong:  "true",
	TradeTypeOpenShort: "false",
}

func (p *FXCM) openTrade(configs TradeConfig) *TradeResult {

	if err, response := p.marketRequest("POST", "/trading/open_trade", map[string]string{
		"symbol":     configs.Pair,
		"account_id": p.config.Custom["account"].(string),
		// "trade_id":   "32992577",
		"is_buy": FxcmTradeTypeMap[configs.Type],
		"amount": strconv.FormatFloat(configs.Amount, 'f', 2, 64),
		// "rate":             strconv.FormatFloat(configs.Price, 'f', 4, 64),
		"at_market":     "0",
		"order_type":    "AtMarket",
		"time_in_force": "FOK",
	}); err != nil {
		logger.Errorf("下单失败:%v", err)
		return &TradeResult{
			Error: err,
		}
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return &TradeResult{
				Error: err,
			}
		}

		if values["response"] != nil && values["response"].(map[string]interface{})["executed"].(bool) != true {
			return &TradeResult{
				Error: errors.New("Fail to executed"),
			}
		}

		info := &OrderInfo{
			// OrderID:    values["clientOrderId"].(string),
			Pair:   configs.Pair,
			Price:  configs.Price,
			Amount: configs.Amount,
		}

		return &TradeResult{
			Error: nil,
			// OrderID: values["clientOrderId"].(string),
			Info: info,
		}
	}
}

func (p *FXCM) closeTrade(configs TradeConfig) *TradeResult {

	if err, response := p.marketRequest("POST", "/trading/close_trade", map[string]string{
		"trade_id": configs.Batch,
		"amount":   strconv.FormatFloat(configs.Amount, 'f', 2, 64),
		// "rate":             strconv.FormatFloat(configs.Price, 'f', 4, 64),
		"at_market":     "0",
		"order_type":    "AtMarket",
		"time_in_force": "FOK",
	}); err != nil {
		logger.Errorf("下单失败:%v", err)
		return &TradeResult{
			Error: err,
		}
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return &TradeResult{
				Error: err,
			}
		}

		if values["response"] != nil && values["response"].(map[string]interface{})["executed"].(bool) != true {
			return &TradeResult{
				Error: errors.New("Fail to executed"),
			}
		}

		id := int(values["data"].(map[string]interface{})["orderId"].(float64))
		info := &OrderInfo{
			OrderID: strconv.Itoa(id),
			Pair:    configs.Pair,
			Price:   configs.Price,
			Amount:  configs.Amount,
		}

		return &TradeResult{
			Error:   nil,
			OrderID: strconv.Itoa(id),
			Info:    info,
		}
	}
}

func (p *FXCM) Trade(configs TradeConfig) *TradeResult {

	if configs.Type == TradeTypeOpenLong || configs.Type == TradeTypeOpenShort {
		return p.openTrade(configs)
	} else if configs.Type == TradeTypeCloseLong || configs.Type == TradeTypeCloseShort {
		return p.closeTrade(configs)
	} else {
		log.Printf("Invalid trade type")
	}

	return nil
}

func (p *FXCM) GetOpenPositions() []OrderInfo {
	if err, response := p.marketRequest("GET", "/trading/get_model", map[string]string{
		"models": "OpenPosition",
	}); err != nil {
		logger.Errorf("获取仓位失败:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		if values["response"] != nil && values["response"].(map[string]interface{})["executed"].(bool) != true {
			logger.Error("获取仓位失败")
			return nil
		}

		positions := values["open_positions"].([]interface{})

		if positions != nil && len(positions) > 0 {
			orders := make([]OrderInfo, len(positions))
			for i, position := range positions {
				orders[i].Pair = position.(map[string]interface{})["currency"].(string)
				orders[i].AvgPrice = position.(map[string]interface{})["open"].(float64)
				orders[i].DealAmount = position.(map[string]interface{})["amountK"].(float64)
				orders[i].OrderID = position.(map[string]interface{})["tradeId"].(string)
			}

			return orders
		}

	}

	return nil
}

func (p *FXCM) GetClosePositions() {
	if err, response := p.marketRequest("GET", "/trading/get_model", map[string]string{
		"models": "ClosedPosition",
	}); err != nil {
		logger.Errorf("下单失败:%v", err)
		return
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return
		}
		log.Printf("values:%v", values)
	}
}

func (p *FXCM) GetOffers() {
	if err, response := p.marketRequest("GET", "/trading/get_model", map[string]string{
		"models": "Offer",
	}); err != nil {
		logger.Errorf("获取货币对列表失败:%v", err)
		return
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return
		}
		offers := values["offers"].([]interface{})

		for _, offer := range offers {
			values = offer.(map[string]interface{})
			log.Printf("OfferID:%d Pair:%s", int(values["offerId"].(float64)), values["currency"])
		}
	}
}

type FxcmKlineValue struct {
	Time     float64
	BidOpen  float64
	BidClose float64
	BidHigh  float64
	BidLow   float64
	AskOpen  float64
	AskClose float64
	AskHigh  float64
	AskLow   float64
	TickQty  float64
}

/*
2018/05/03 10:04:49 OfferID:1 Pair:EUR/USD
2018/05/03 10:04:49 OfferID:2 Pair:USD/JPY
2018/05/03 10:04:49 OfferID:3 Pair:GBP/USD
2018/05/03 10:04:49 OfferID:4 Pair:USD/CHF
2018/05/03 10:04:49 OfferID:7 Pair:USD/CAD
2018/05/03 10:04:49 OfferID:8 Pair:NZD/USD
2018/05/03 10:04:49 OfferID:10 Pair:EUR/JPY
2018/05/03 10:04:49 OfferID:11 Pair:GBP/JPY
2018/05/03 10:04:49 OfferID:16 Pair:AUD/CAD
2018/05/03 10:04:49 OfferID:17 Pair:AUD/JPY
2018/05/03 10:04:49 OfferID:19 Pair:NZD/JPY
2018/05/03 10:04:49 OfferID:22 Pair:GBP/AUD
2018/05/03 10:04:49 OfferID:28 Pair:AUD/NZD
2018/05/03 10:04:49 OfferID:1004 Pair:GER30
2018/05/03 10:04:49 OfferID:1005 Pair:HKG33
2018/05/03 10:04:49 OfferID:1013 Pair:US30
2018/05/03 10:04:49 OfferID:2001 Pair:USOil
2018/05/03 10:04:49 OfferID:4001 Pair:XAU/USD
*/

const (
	FxcmPairEURUSD = "EUR/USD"
	FxcmPairUS30   = "US30"
	FxcmPairCHN50  = "CHN50"
	FxcmPairUSOil  = "USOil"
)

var MapOfferID = map[string]string{
	FxcmPairEURUSD: "1",
	"USD/JPY":      "2",
	"GBP/USD":      "3",
	"USD/CHF":      "4",
	"USD/CAD":      "7",
	"GER30":        "1004",
	"HKG33":        "1005",
	FxcmPairUS30:   "1013",
	FxcmPairUSOil:  "2001",
	"XAU/USD":      "4001",
	FxcmPairCHN50:  "1020",
}

var MapDeposit = map[string]float64{
	FxcmPairEURUSD: 13,
	FxcmPairUS30:   13.5,
	FxcmPairCHN50:  65,
	FxcmPairUSOil:  20,
}

func (p *FXCM) GetKline(pair string, period int, limit int) []KlineValue {

	// log.Printf("OfferID:%s", MapOfferID[pair])
	var interval string
	switch period {
	case KlinePeriod5Min:
		interval = "m5"
	case KlinePeriod15Min:
		interval = "m15"
	case KlinePeriod30Min:
		interval = "m30"
	case KlinePeriod1Hour:
		interval = "H1"
	case KlinePeriod2Hour:
		interval = "H2"
	case KlinePeriod1Day:
		interval = "D1"
	}

	if err, response := p.marketRequest("GET", "/candles/"+MapOfferID[pair]+"/"+interval, map[string]string{
		"num": strconv.Itoa(limit), // max:10000
	}); err != nil {
		logger.Errorf("获取K线失败:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		arrays := values["candles"].([]interface{})

		kline := make([]KlineValue, len(arrays))
		for i, item := range arrays {
			value := item.([]interface{})
			kline[i].OpenTime = value[0].(float64)
			kline[i].Open = (value[1].(float64) + value[5].(float64)) / 2
			kline[i].Close = (value[2].(float64) + value[6].(float64)) / 2
			kline[i].High = (value[3].(float64) + value[7].(float64)) / 2
			kline[i].Low = (value[4].(float64) + value[8].(float64)) / 2
			kline[i].Volumn = value[9].(float64)
		}

		return kline
	}
}
