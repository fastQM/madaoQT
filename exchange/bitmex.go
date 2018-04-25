package exchange

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	Websocket "github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

const BitmexAPIRoot = "https://www.bitmex.com/api/v1"
const NameBitmex = "Bitmex"

//对我们的 REST API 的请求是限于每 5 分钟 300 次的速率。此计数器持续重设。如果您没有登录，您的频率限制是每 5 分钟 150 次。

type ExchangeBitmex struct {
	websocket *Websocket.Conn
	event     chan EventType
	config    Config
}

var BitmexTradeTypeMap = map[TradeType]string{
	TradeTypeBuy:  "Buy",
	TradeTypeSell: "Sell",
}

func (p *ExchangeBitmex) GetExchangeName() string {
	return NameBitmex
}

// WatchEvent() return a channel which notified the application of the event triggered by exchange
func (p *ExchangeBitmex) WatchEvent() chan EventType {
	return p.event
}

func (p *ExchangeBitmex) Start() error {
	return nil
}

// Close() close the connection to the exchange and other handles
func (p *ExchangeBitmex) Close() {

}

// StartTicker() send message to the exchange to start the ticker of the given pairs
func (p *ExchangeBitmex) StartTicker(pair string) {
}

// CancelOrder() cancel the order as the order information
func (p *ExchangeBitmex) CancelOrder(order OrderInfo) *TradeResult {
	return nil
}

// GetOrderInfo() get the information with order filter
func (p *ExchangeBitmex) GetOrderInfo(filter OrderInfo) []OrderInfo {
	return nil
}

func (p *ExchangeBitmex) GetKline(pair string, period int, limit int) []KlineValue {
	return nil
}

func (p *ExchangeBitmex) GetTicker(pair string) *TickerValue {
	return nil
}

// SetConfigure()
func (p *ExchangeBitmex) SetConfigure(config Config) {
	p.config = config

	if p.config.Proxy != "" {
		logger.Infof("使用代理:%s", p.config.Proxy)
	}
}

func (p *ExchangeBitmex) orderRequest(method string, path string, params map[string]string) (error, []byte) {
	var req http.Request
	req.ParseForm()
	for k, v := range params {
		req.Form.Add(k, v)
	}
	bodystr := strings.TrimSpace(req.Form.Encode())
	logger.Debugf("Params:%v Path:%s", params, BitmexAPIRoot+path+"?"+bodystr)

	var request *http.Request
	var err error

	if method == "GET" {
		request, err = http.NewRequest(method, BitmexAPIRoot+path+"?"+bodystr, nil)
		if err != nil {
			return err, nil
		}
	} else if method == "POST" {
		request, err = http.NewRequest(method, BitmexAPIRoot+path, strings.NewReader(bodystr))
		if err != nil {
			return err, nil
		}
	}

	expire := strconv.Itoa(int(time.Now().Add(5 * time.Second).Unix()))
	request.Header.Add("api-expires", expire)
	request.Header.Add("api-key", p.config.API)

	if method == "GET" {
		request.Header.Add("api-signature", p.sign(method, "/api/v1"+path+"?"+bodystr, expire, ""))
	} else if method == "POST" {
		request.Header.Add("api-signature", p.sign(method, "/api/v1"+path, expire, bodystr))
		request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

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
	// log.Printf("Body:%v", string(body))
	// var value map[string]interface{}
	// if err = json.Unmarshal(body, &value); err != nil {
	// 	return err, nil
	// }

	// log.Printf("Header:%v", resp.Header)
	limit, _ := strconv.ParseInt(resp.Header["X-Ratelimit-Remaining"][0], 10, 64)
	logger.Infof("Access Limit:%d", limit)

	return nil, body
}

func (p *ExchangeBitmex) marketRequest(path string, params map[string]string) (error, []byte) {
	var req http.Request
	req.ParseForm()
	for k, v := range params {
		req.Form.Add(k, v)
	}
	bodystr := strings.TrimSpace(req.Form.Encode())
	logger.Debugf("Params:%v Path:%s", bodystr, BitmexAPIRoot+path+"?"+bodystr)
	request, err := http.NewRequest("GET", BitmexAPIRoot+path+"?"+bodystr, nil)
	if err != nil {
		return err, nil
	}

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
	// log.Printf("Body:%v", string(body))
	// var value map[string]interface{}
	// if err = json.Unmarshal(body, &value); err != nil {
	// 	return err, nil
	// }

	return nil, body
}

// GetBalance() get the balances of all the coins
func (p *ExchangeBitmex) GetBalance() map[string]interface{} {

	if err, response := p.orderRequest("GET", "/user/wallet", map[string]string{
		"currency": "XBt",
	}); err != nil {
		logger.Errorf("无法获取余额:%v", err)
		return nil
	} else {
		// log.Printf("balance:%v", string(response))
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		if values["code"] == nil {
			balance := values["amount"].(float64) / 1e8
			return map[string]interface{}{
				"btc": balance,
			}
		}

	}

	return nil
}

// Trade() trade as the configs
func (p *ExchangeBitmex) Trade(configs TradeConfig) *TradeResult {

	symbol := "XBTUSD"
	if err, response := p.orderRequest("POST", "/order", map[string]string{
		"symbol":      symbol,
		"side":        BitmexTradeTypeMap[configs.Type],
		"orderType":   "Limit",
		"orderQty":    strconv.FormatFloat(configs.Amount, 'f', 4, 64),
		"price":       strconv.FormatFloat(configs.Price, 'f', 0, 64),
		"timeInForce": "ImmediateOrCancel",
	}); err != nil {
		logger.Errorf("下单失败:%v", err)
		return &TradeResult{
			Error: err,
		}
	} else {
		var values map[string]interface{}
		log.Printf("response:%s", string(response))
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return &TradeResult{
				Error: err,
			}
		}

		if values["error"] != nil {
			return &TradeResult{
				Error: errors.New(values["error"].(map[string]interface{})["message"].(string)),
			}
		}

		if p.getStatusType(values["ordStatus"].(string)) != OrderStatusDone {
			return &TradeResult{
				Error: nil,
				Info:  nil,
			}

		} else {

			executedQty := values["cumQty"].(float64)
			avgPrice := values["avgPx"].(float64)

			info := &OrderInfo{
				OrderID:    values["orderID"].(string),
				Pair:       symbol,
				Price:      configs.Price,
				Amount:     configs.Amount,
				AvgPrice:   avgPrice,
				DealAmount: executedQty,
			}

			return &TradeResult{
				Error:   nil,
				OrderID: values["orderID"].(string),
				Info:    info,
			}
		}
	}
}

func (p *ExchangeBitmex) GetComposite(symbol string, count int) (error, float64) {

	if err, response := p.orderRequest("GET", "/instrument/compositeIndex", map[string]string{
		"symbol":  symbol,
		"count":   strconv.Itoa(count),
		"reverse": "true",
	}); err != nil {
		logger.Errorf("无效深度:%v", err)
		return err, 0
	} else {
		var datas []map[string]interface{}
		if err = json.Unmarshal(response, &datas); err != nil {
			logger.Errorf("解析错误:%v", err)
			return err, 0
		}

		var gdaxDatas []map[string]interface{}
		var bitstampDatas []map[string]interface{}

		for _, data := range datas {
			if data["reference"] != nil && data["reference"].(string) == "BSTP" {
				bitstampDatas = append(bitstampDatas, data)
			} else if data["reference"] != nil && data["reference"].(string) == "GDAX" {
				gdaxDatas = append(gdaxDatas, data)
			}
		}

		// for i, data := range gdaxDatas {
		// 	log.Printf("%d Gdax:%v", i, data)
		// }
		// for i, data := range bitstampDatas {
		// 	log.Printf("%d Bitstamp:%v", i, data)
		// }

		if len(gdaxDatas) == 0 || len(bitstampDatas) == 0 {
			return errors.New("数据长度不匹配"), 0
		}

		length := len(gdaxDatas)
		if len(gdaxDatas) > len(bitstampDatas) {
			length = len(bitstampDatas)
		}

		var offset float64
		for i := 0; i < length; i++ {
			if gdaxDatas[i]["timestamp"].(string) != bitstampDatas[i]["timestamp"].(string) {
				return errors.New("数据不匹配"), 0
			}

			weightPrice := (gdaxDatas[i]["lastPrice"].(float64) + bitstampDatas[i]["lastPrice"].(float64)) / 2
			// log.Printf("offset:%.6f", gdaxDatas[i]["lastPrice"].(float64)/bitstampDatas[i]["lastPrice"].(float64))
			offset += weightPrice / gdaxDatas[i]["lastPrice"].(float64)
		}

		return nil, offset / float64(length)
	}
}

func (p *ExchangeBitmex) GetDepthValue(pair string) [][]DepthPrice {
	//ethusdt@depth20

	if err, response := p.orderRequest("GET", "/orderBook/L2", map[string]string{
		"symbol": "XBT",
		"depth":  "100",
	}); err != nil {
		logger.Errorf("无效深度:%v", err)
		return nil
	} else {

		var values []map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		list := make([][]DepthPrice, 2)
		var askList, bidList []DepthPrice
		for _, value := range values {
			if value["side"].(string) == "Sell" {
				var askItem DepthPrice
				askItem.Price = value["price"].(float64)
				askItem.Quantity = value["size"].(float64)
				askList = append(askList, askItem)
			}

			if value["side"].(string) == "Buy" {
				var bidItem DepthPrice
				bidItem.Price = value["price"].(float64)
				bidItem.Quantity = value["size"].(float64)
				bidList = append(bidList, bidItem)
			}
		}

		list[DepthTypeAsks] = revertDepthArray(askList)
		list[DepthTypeBids] = bidList

		return list
	}

	return nil
}

func (p *ExchangeBitmex) sign(method string, path string, expire string, data string) string {
	plain := method + path + expire + data
	// log.Printf("Plain:%s", plain)
	h := hmac.New(sha256.New, []byte(p.config.Secret))
	io.WriteString(h, plain)

	return hex.EncodeToString(h.Sum(nil))
}

var BitmexOrderStatusMap = map[OrderStatusType]string{
	OrderStatusOpen:      "NEW",
	OrderStatusPartDone:  "PARTIALLY_FILLED",
	OrderStatusDone:      "Filled",
	OrderStatusCanceling: "PENDING_CANCEL",
	OrderStatusCanceled:  "Canceled",
	OrderStatusRejected:  "REJECTED",
	OrderStatusExpired:   "EXPIRED",
}

func (p *ExchangeBitmex) getStatusType(key string) OrderStatusType {
	for k, v := range BitmexOrderStatusMap {
		if v == key {
			return k
		}
	}
	return OrderStatusUnknown
}
