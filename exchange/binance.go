package exchange

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	Websocket "github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

const EndPoint = "wss://stream.binance.com:9443/ws/"
const BinanceURL = "https://api.binance.com"
const NameBinance = "Binance"

type Binance struct {
	websocket *Websocket.Conn
	event     chan EventType
	config    Config
}

func (p *Binance) GetExchangeName() string {
	return NameBinance
}

// SetConfigure()
func (p *Binance) SetConfigure(config Config) {
	p.config = config

	if p.config.Proxy != "" {
		logger.Infof("使用代理:%s", p.config.Proxy)
	}
}

// WatchEvent() return a channel which notified the application of the event triggered by exchange
func (p *Binance) WatchEvent() chan EventType {
	return p.event
}

func (h *Binance) Start() error {
	return nil
}

func (p *Binance) marketRequest(path string, params map[string]string) (error, []byte) {
	var req http.Request
	req.ParseForm()
	for k, v := range params {
		req.Form.Add(k, v)
	}
	bodystr := strings.TrimSpace(req.Form.Encode())
	// logger.Debugf("Params:%v", bodystr)
	request, err := http.NewRequest("GET", BinanceURL+path+"?"+bodystr, nil)
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

func (p *Binance) orderRequest(method string, path string, params map[string]string) (error, []byte) {

	// add the common parameters
	params["timestamp"] = strconv.FormatInt(time.Now().Unix()*1000, 10)

	var req http.Request

	req.ParseForm()
	for k, v := range params {
		req.Form.Add(k, v)
	}
	bodystr := strings.TrimSpace(req.Form.Encode())

	h := hmac.New(sha256.New, []byte(p.config.Secret))
	io.WriteString(h, bodystr)
	signature := "&signature=" + fmt.Sprintf("%x", h.Sum(nil))
	logger.Debugf("Path:%s", BinanceURL+path+"?"+bodystr+signature)

	request, err := http.NewRequest(method, BinanceURL+path+"?"+bodystr+signature, nil)
	if err != nil {
		return err, nil
	}
	request.Header.Add("X-MBX-APIKEY", p.config.API)

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

	logger.Debugf("RSP:%s", string(body))
	return nil, body

}

// Close() close the connection to the exchange and other handles
func (p *Binance) Close() {

}

// StartTicker() send message to the exchange to start the ticker of the given pairs
func (p *Binance) StartTicker(pair string) {
}

// GetTicker(), better to use the ITicker to notify the ticker information
func (p *Binance) GetTicker(pair string) *TickerValue {
	return nil
}

func (p *Binance) getSymbol(pair string) string {
	coins := ParsePair(pair)
	return strings.ToUpper(coins[0] + coins[1])
}

// GetDepthValue() get the depth of the assigned price area and quantity
// GetDepthValue(pair string, price float64, limit float64, orderQuantity float64, tradeType TradeType) []DepthPrice
func (p *Binance) GetDepthValue(pair string) [][]DepthPrice {
	//ethusdt@depth20
	symbol := p.getSymbol(pair)
	if err, response := p.marketRequest("/api/v1/depth", map[string]string{
		"symbol": symbol,
		// "limit":  "100",
	}); err != nil {
		logger.Errorf("无效深度:%v", err)
		return nil
	} else {

		var value map[string]interface{}
		if err = json.Unmarshal(response, &value); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		if value["code"] == nil {
			list := make([][]DepthPrice, 2)

			asks := value["asks"].([]interface{})
			bids := value["bids"].([]interface{})

			if asks != nil && len(asks) > 0 {
				askList := make([]DepthPrice, len(asks))
				for i, ask := range asks {
					values := ask.([]interface{})
					askList[i].Price, _ = strconv.ParseFloat(values[0].(string), 64)
					askList[i].Quantity, _ = strconv.ParseFloat(values[1].(string), 64)
				}

				list[DepthTypeAsks] = askList
			}

			if bids != nil && len(bids) > 0 {
				bidList := make([]DepthPrice, len(bids))
				for i, bid := range bids {
					values := bid.([]interface{})
					bidList[i].Price, _ = strconv.ParseFloat(values[0].(string), 64)
					bidList[i].Quantity, _ = strconv.ParseFloat(values[1].(string), 64)
				}

				list[DepthTypeBids] = bidList
			}

			return list
		}
	}

	return nil
}

// GetBalance() get the balances of all the coins
func (p *Binance) GetBalance() map[string]interface{} {

	if err, response := p.orderRequest("GET", "/api/v3/account", map[string]string{}); err != nil {
		logger.Errorf("无法获取余额:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		// log.Printf("Val:%v", values)
		balances := make(map[string]interface{})
		assets := values["balances"].([]interface{})
		for _, asset := range assets {
			key := asset.(map[string]interface{})["asset"].(string)
			value := asset.(map[string]interface{})["free"].(string)
			balances[key], _ = strconv.ParseFloat(value, 64)
		}

		return balances

	}

}

// Trade() trade as the configs
func (p *Binance) Trade(configs TradeConfig) *TradeResult {
	symbol := p.getSymbol(configs.Pair)

	if err, response := p.orderRequest("POST", "/api/v3/order", map[string]string{
		"symbol":           symbol,
		"side":             BinanceTradeTypeMap[configs.Type],
		"type":             "LIMIT",
		"quantity":         strconv.FormatFloat(configs.Amount, 'f', 4, 64),
		"price":            strconv.FormatFloat(configs.Price, 'f', 4, 64),
		"timeInForce":      "IOC",
		"newOrderRespType": "FULL",
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

		if values["code"] != nil || values["msg"] != nil {
			return &TradeResult{
				Error: errors.New(values["msg"].(string)),
			}
		}

		if p.getStatusType(values["status"].(string)) != OrderStatusDone {
			return &TradeResult{
				Error: nil,
				Info:  nil,
			}

		} else {

			fills := values["fills"].([]interface{})

			var avgPrice, totalCost float64

			for _, fill := range fills {
				price, _ := strconv.ParseFloat(fill.(map[string]interface{})["price"].(string), 64)
				qty, _ := strconv.ParseFloat(fill.(map[string]interface{})["qty"].(string), 64)
				totalCost += (price * qty)
			}

			executedQty, _ := strconv.ParseFloat(values["executedQty"].(string), 64)
			avgPrice = totalCost / executedQty

			info := &OrderInfo{
				OrderID:    values["clientOrderId"].(string),
				Pair:       symbol,
				Price:      configs.Price,
				Amount:     configs.Amount,
				AvgPrice:   avgPrice,
				DealAmount: executedQty,
			}

			return &TradeResult{
				Error:   nil,
				OrderID: values["clientOrderId"].(string),
				Info:    info,
			}
		}
	}
}

// CancelOrder() cancel the order as the order information
func (p *Binance) CancelOrder(order OrderInfo) *TradeResult {
	return nil
}

// GetOrderInfo() get the information with order filter
func (p *Binance) GetOrderInfo(filter OrderInfo) []OrderInfo {
	symbol := p.getSymbol(filter.Pair)
	if err, response := p.orderRequest("GET", "/api/v3/order", map[string]string{
		"symbol":            symbol,
		"origClientOrderId": filter.OrderID,
	}); err != nil {
		logger.Errorf("无法获取订单信息:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		info := make([]OrderInfo, 1)
		info[0].Amount, _ = strconv.ParseFloat(values["origQty"].(string), 64)
		info[0].Pair = symbol
		info[0].DealAmount, _ = strconv.ParseFloat(values["executedQty"].(string), 64)
		info[0].Status = p.getStatusType(values["status"].(string))
		info[0].OrderID = filter.OrderID
		info[0].AvgPrice, _ = strconv.ParseFloat(values["stopPrice"].(string), 64)

		return info

	}
}

func (p *Binance) GetKline(pair string, period int, limit int) []KlineValue {
	coins := ParsePair(pair)
	symbol := strings.ToUpper(coins[0] + coins[1])

	var interval string
	if period == KlinePeriod5Min {
		interval = "5m"
	} else if period == KlinePeriod15Min {
		interval = "15m"
	}

	if err, response := p.marketRequest("/api/v1/klines", map[string]string{
		"symbol":   symbol,
		"interval": interval,
		"limit":    strconv.Itoa(limit),
	}); err != nil {
		logger.Errorf("无效数据:%v", err)
		return nil
	} else {
		var values [][]interface{}
		if response != nil {
			if err = json.Unmarshal(response, &values); err != nil {
				logger.Errorf("Fail to Unmarshal:%v", err)
				return nil
			}

			kline := make([]KlineValue, len(values))
			for i, value := range values {
				// kline[i].OpenTime = time.Unix((int64)(value[0].(float64)/1000), 0).Format(Global.TimeFormat)
				kline[i].OpenTime = value[0].(float64) / 1000
				kline[i].Open, _ = strconv.ParseFloat(value[1].(string), 64)
				kline[i].High, _ = strconv.ParseFloat(value[2].(string), 64)
				kline[i].Low, _ = strconv.ParseFloat(value[3].(string), 64)
				kline[i].Close, _ = strconv.ParseFloat(value[4].(string), 64)
				kline[i].Volumn, _ = strconv.ParseFloat(value[5].(string), 64)
				// kline[i].CloseTime = time.Unix((int64)(value[6].(float64)/1000), 0).Format(Global.TimeFormat)
				kline[i].CloseTime = value[6].(float64) / 1000

			}

			return kline
		}

		return nil
	}
}

var BinanceOrderStatusMap = map[OrderStatusType]string{
	OrderStatusOpen:      "NEW",
	OrderStatusPartDone:  "PARTIALLY_FILLED",
	OrderStatusDone:      "FILLED",
	OrderStatusCanceling: "PENDING_CANCEL",
	OrderStatusCanceled:  "CANCELED",
	OrderStatusRejected:  "REJECTED",
	OrderStatusExpired:   "EXPIRED",
}

func (p *Binance) getStatusType(key string) OrderStatusType {
	for k, v := range BinanceOrderStatusMap {
		if v == key {
			return k
		}
	}
	return OrderStatusUnknown
}

var BinanceTradeTypeMap = map[TradeType]string{
	TradeTypeBuy:  "BUY",
	TradeTypeSell: "SELL",
}
