package exchange

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	socketio "github.com/graarh/golang-socketio"
	"github.com/graarh/golang-socketio/transport"
	"golang.org/x/net/proxy"
)

const ExchangeFXCM = "FXCM"

const WssFXCMUrl = "wss://api.fxcm.com"
const HttpFXCMUrl = "https://api.fxcm.com"

type FXCM struct {
	config Config
	event  chan EventType
	socket *socketio.Client
}

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

	token := p.config.Custom["token"].(string)
	socket, err := socketio.Dial(WssFXCMUrl+"/socket.io/?EIO=3&transport=websocket&access_token="+token,
		transport.GetDefaultWebsocketTransport())
	if err != nil {
		return err
	}

	socket.On(socketio.OnConnection, func(c *socketio.Channel, args interface{}) {
		log.Printf("Socket Connected[%s]", c.Id())
		p.socket = socket
	})

	socket.On(socketio.OnError, func(c *socketio.Channel) {
		log.Printf("Error occurs")
	})

	return nil
}

func (p *FXCM) marketRequest(method, path string, params map[string]string) (error, []byte) {

	if p.socket == nil {
		return errors.New("Socket is not connected"), nil
	}

	var bodystr string
	for k, v := range params {
		if bodystr == "" {
			bodystr += (k + "=" + v)
		} else {
			bodystr += ("&" + k + "=" + v)
		}

	}
	logger.Debugf("Params:%s auth[%s]", bodystr, "Bearer "+p.socket.Id()+p.config.Custom["token"].(string))

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
	log.Printf("Body:%v", string(body))
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

		if values["response"] != nil && values["response"].(map[string]interface{})["executed"].(bool) == true {
			logger.Errorf("无法获取余额:%v", err)
			return nil
		}

		return map[string]interface{}{
			"balance": values["balance"].(float64),
		}

	}

	return nil
}

func (p *FXCM) GetTicker(pair string) *TickerValue {

	if err, response := p.marketRequest("POST", "/subscribe", map[string]string{
		"pairs": pair,
	}); err != nil {
		logger.Errorf("无法获取当前价格:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		log.Printf("values:%v", values)
		if values["code"] == nil {
			// log.Printf("Val:%v", values)
		}

		p.socket.On(pair, func(c *socketio.Channel, args interface{}) {
			log.Printf("Values:%v", args)
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

		if values["response"] != nil && values["response"].(map[string]interface{})["executed"].(bool) == true {
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

		if values["response"] != nil && values["response"].(map[string]interface{})["executed"].(bool) == true {
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

func (p *FXCM) GetOpenPositions() {
	if err, response := p.marketRequest("GET", "/trading/get_model", map[string]string{
		"models": "OpenPosition",
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

func (p *FXCM) GetKline(pair string, period int, limit int) []KlineValue {
	if err, response := p.marketRequest("GET", "/candles/1/D1", map[string]string{
		"num": strconv.Itoa(limit), // max:10000
	}); err != nil {
		logger.Errorf("下单失败:%v", err)
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
