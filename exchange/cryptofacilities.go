package exchange

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

const CryptoFacilitiesURL = "https://www.cryptofacilities.com/derivatives"
const NameCryptoFacilities = "CryptoFacilities"

//The number of API calls is limited to 10 per second for non-whitelisted users.
//If the API limit is exceeded, the API  will return error equal to apiLimitExeeded.

type CryptoFacilities struct {
	event  chan EventType
	config Config

	nonce int64
}

func (p *CryptoFacilities) GetExchangeName() string {
	return NameBinance
}

// SetConfigure()
func (p *CryptoFacilities) SetConfigure(config Config) {
	p.config = config

	if p.config.Proxy != "" {
		logger.Infof("使用代理:%s", p.config.Proxy)
	}
}

// WatchEvent() return a channel which notified the application of the event triggered by exchange
func (p *CryptoFacilities) WatchEvent() chan EventType {
	return p.event
}

func (h *CryptoFacilities) Start() error {
	return nil
}

func (p *CryptoFacilities) marketRequest(path string, params map[string]string) (error, []byte) {
	var req http.Request
	req.ParseForm()
	for k, v := range params {
		req.Form.Add(k, v)
	}
	bodystr := strings.TrimSpace(req.Form.Encode())
	// logger.Debugf("Params:%v", bodystr)
	request, err := http.NewRequest("GET", CryptoFacilitiesURL+path+"?"+bodystr, nil)
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

func (p *CryptoFacilities) orderRequest(method string, path string, params map[string]string) (error, []byte) {

	var req http.Request
	var bodystr string

	if path == "/api/v3/sendorder" {
		// for sendorder, the sequence of the params should be the same
		bodystr = params["params"]
	} else {
		req.ParseForm()
		for k, v := range params {
			req.Form.Add(k, v)
		}
		bodystr = strings.TrimSpace(req.Form.Encode())
	}

	p.nonce++
	now := time.Now().Unix()*1000 + p.nonce
	nonce := strconv.Itoa(int(now))
	// nonce = "1536541236001"
	input := bodystr + nonce + path
	log.Printf("Input:%s", input)
	h := sha256.New()
	h.Write([]byte(input))
	hash := h.Sum(nil)
	// logger.Debugf("Path:%s", CryptoFacilitiesURL+path)
	// logger.Debugf("Hash:%v", hex.EncodeToString(hash))

	base64apisecret, err := base64.StdEncoding.DecodeString(p.config.Secret)
	if err != nil {
		log.Printf("Fail to decode:%v", err)
		return nil, nil
	}
	// logger.Debugf("Decoded:%v", hex.EncodeToString(base64apisecret))

	hmac := hmac.New(sha512.New, base64apisecret)
	io.WriteString(hmac, string(hash))
	signature := hmac.Sum(nil)
	// logger.Debugf("signature:%v", hex.EncodeToString(signature))

	base64signature := base64.StdEncoding.EncodeToString(signature)
	// base64signature = base64signature + "hello"
	// logger.Debugf("base64signature:%v", base64signature)

	request, err := http.NewRequest(method, CryptoFacilitiesURL+path, strings.NewReader(bodystr))
	if err != nil {
		return err, nil
	}

	request.Header.Add("APIKey", p.config.API)
	request.Header.Add("Nonce", nonce)
	request.Header.Add("Authent", string(base64signature))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
func (p *CryptoFacilities) Close() {

}

// StartTicker() send message to the exchange to start the ticker of the given pairs
func (p *CryptoFacilities) StartTicker(pair string) {
}

// GetTicker(), better to use the ITicker to notify the ticker information
func (p *CryptoFacilities) GetTicker(pair string) *TickerValue {
	return nil
}

func (p *CryptoFacilities) getSymbol(pair string) string {
	coins := ParsePair(pair)
	return "pi_" + strings.ToLower(coins[0]+coins[1])
}

// GetDepthValue() get the depth of the assigned price area and quantity
// GetDepthValue(pair string, price float64, limit float64, orderQuantity float64, tradeType TradeType) []DepthPrice
func (p *CryptoFacilities) GetDepthValue(pair string) [][]DepthPrice {
	//ethusdt@depth20
	symbol := p.getSymbol(pair)
	if err, response := p.marketRequest("/api/v3/orderbook", map[string]string{
		"symbol": symbol,
		// "limit":  "100",
	}); err != nil {
		logger.Errorf("Fail to get the orderbook:%v", err)
		return nil
	} else {

		var value map[string]interface{}
		if err = json.Unmarshal(response, &value); err != nil {
			logger.Errorf("Fail to parse:%v", err)
			return nil
		}

		if value["result"].(string) != "success" {
			logger.Errorf("Fail to execute the command:%v", value["error"])
			return nil
		}

		orderBook := value["orderBook"].(map[string]interface{})
		if orderBook != nil {
			list := make([][]DepthPrice, 2)

			asks := orderBook["asks"].([]interface{})
			bids := orderBook["bids"].([]interface{})

			if asks != nil && len(asks) > 0 {
				askList := make([]DepthPrice, len(asks))
				for i, ask := range asks {
					values := ask.([]interface{})
					askList[i].Price = values[0].(float64)
					askList[i].Quantity = values[1].(float64)
				}

				list[DepthTypeAsks] = askList
			}

			if bids != nil && len(bids) > 0 {
				bidList := make([]DepthPrice, len(bids))
				for i, bid := range bids {
					values := bid.([]interface{})
					bidList[i].Price = values[0].(float64)
					bidList[i].Quantity = values[1].(float64)
				}

				list[DepthTypeBids] = bidList
			}

			return list
		}

	}

	return nil
}

// GetBalance() get the balances of all the coins
func (p *CryptoFacilities) GetBalance() map[string]interface{} {

	if err, response := p.orderRequest("GET", "/api/v3/accounts", map[string]string{}); err != nil {
		logger.Errorf("Fail to get balance:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse:%v", err)
			return nil
		}

		if values["result"].(string) == "success" {
			account := values["accounts"].(map[string]interface{})
			balances := make(map[string]interface{})
			log.Printf("Account %v", account)
			for key, value := range account {
				if value == nil || value.(map[string]interface{})["auxiliary"] == nil {
					// log.Printf("KEy:%v", key)
					continue
				}
				balances[key] = value.(map[string]interface{})["auxiliary"].(map[string]interface{})["af"].(float64)
			}
			return balances
		} else {
			logger.Errorf("Fail to get balance:%v", values["error"].(string))
		}

	}

	return nil
}

// Trade() trade as the configs
func (p *CryptoFacilities) Trade(configs TradeConfig) *TradeResult {
	symbol := p.getSymbol(configs.Pair)

	if configs.Type == TradeTypeCloseLong {
		configs.Type = TradeTypeSell
	} else if configs.Type == TradeTypeCloseShort {
		configs.Type = TradeTypeBuy
	}

	if err, response := p.orderRequest("POST", "/api/v3/sendorder", map[string]string{
		"params": "orderType=lmt&symbol=" + symbol +
			"&side=" + CryptoFacilitiesTradeTypeMap[configs.Type] +
			"&size=" + strconv.FormatInt(int64(configs.Amount), 10) +
			"&limitPrice=" + strconv.FormatFloat(configs.Price, 'f', 2, 64),
	}); err != nil {
		logger.Errorf("Fail to trade:%v", err)
		return &TradeResult{
			Error: err,
		}
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse:%v", err)
			return &TradeResult{
				Error: err,
			}
		}

		if values["result"].(string) != "success" {
			return &TradeResult{
				Error: errors.New(values["error"].(string)),
			}
		}

		status := values["sendStatus"].(map[string]interface{})
		if status["status"].(string) != "placed" && status["status"].(string) != "attempted" {
			logger.Errorf("Send status:%v", values)
			return &TradeResult{
				Error: errors.New(status["status"].(string)),
				Info:  nil,
			}

		} else {
			return &TradeResult{
				Error:   nil,
				OrderID: status["order_id"].(string),
			}
		}
	}
}

// CancelOrder() cancel the order as the order information
func (p *CryptoFacilities) CancelOrder(order OrderInfo) *TradeResult {
	if err, response := p.orderRequest("POST", "/api/v3/cancelorder", map[string]string{
		"order_id": order.OrderID,
	}); err != nil {
		logger.Errorf("Fail to get instruments info:%v", err)
		return &TradeResult{
			Error: err,
		}
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse:%v", err)
			return &TradeResult{
				Error: err,
			}
		}

		if values["result"].(string) != "success" {
			logger.Errorf("Fail to get the result:%v", values["error"].(string))
			return &TradeResult{
				Error: errors.New(values["error"].(string)),
			}
		}

		return &TradeResult{
			Error: nil,
		}

	}
}

// GetOrderInfo() get the information with order filter
func (p *CryptoFacilities) GetOrderInfo(filter OrderInfo) []OrderInfo {
	if err, response := p.orderRequest("GET", "/api/v3/openorders", map[string]string{}); err != nil {
		logger.Errorf("Fail to get open Orders:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse:%v", err)
			return nil
		}

		if values["result"].(string) != "success" {
			logger.Errorf("Fail to get the result:%v", values["error"].(string))
			return nil
		}

		orders := values["openOrders"].([]interface{})
		if orders != nil {
			infos := make([]OrderInfo, len(orders))
			for i, order := range orders {
				// infos[i].Status = order.(map[string]string)["status"]
				infos[i].OrderID = order.(map[string]string)["order_id"]
			}
		}

		return nil
	}
}

func (p *CryptoFacilities) GetKline(pair string, period int, limit int) []KlineValue {
	coins := ParsePair(pair)
	symbol := strings.ToUpper(coins[0] + coins[1])

	var interval string
	// if period == KlinePeriod5Min {
	// 	interval = "5m"
	// } else if period == KlinePeriod15Min {
	// 	interval = "15m"
	// } else if period == KlinePeriod1Day {
	// 	interval = "1d"
	// }
	switch period {
	case KlinePeriod5Min:
		interval = "5m"
	case KlinePeriod15Min:
		interval = "15m"
	case KlinePeriod1Hour:
		interval = "1h"
	case KlinePeriod2Hour:
		interval = "2h"
	case KlinePeriod4Hour:
		interval = "4h"
	case KlinePeriod6Hour:
		interval = "6h"
	case KlinePeriod1Day:
		interval = "1d"
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

func (p *CryptoFacilities) GetInstruments() []interface{} {
	if err, response := p.marketRequest("/api/v3/instruments", map[string]string{}); err != nil {
		logger.Errorf("Fail to get instruments info:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse:%v", err)
			return nil
		}

		if values["result"].(string) != "success" {
			logger.Errorf("Fail to get the result:%v", values["error"].(string))
			return nil
		}

		return values["instruments"].([]interface{})

	}
}

func (p *CryptoFacilities) GetPositions(pair string) []map[string]interface{} {
	symbol := p.getSymbol(pair)
	if err, response := p.orderRequest("GET", "/api/v3/openpositions", map[string]string{}); err != nil {
		logger.Errorf("Fail to get instruments info:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse:%v", err)
			return nil
		}

		if values["result"].(string) != "success" {
			logger.Errorf("Fail to get the result:%v", values["error"].(string))
			return nil
		}

		positions := values["openPositions"].([]interface{})
		var results []map[string]interface{}
		for _, position := range positions {
			value := position.(map[string]interface{})
			if value["symbol"].(string) == symbol {
				value["symbol"] = pair
				results = append(results, value)
			}
		}

		return results

	}
}

var CryptoFacilitiesOrderStatusMap = map[OrderStatusType]string{
	OrderStatusOpen:      "NEW",
	OrderStatusPartDone:  "PARTIALLY_FILLED",
	OrderStatusDone:      "FILLED",
	OrderStatusCanceling: "PENDING_CANCEL",
	OrderStatusCanceled:  "CANCELED",
	OrderStatusRejected:  "REJECTED",
	OrderStatusExpired:   "EXPIRED",
}

func (p *CryptoFacilities) getStatusType(key string) OrderStatusType {
	for k, v := range BinanceOrderStatusMap {
		if v == key {
			return k
		}
	}
	return OrderStatusUnknown
}

var CryptoFacilitiesTradeTypeMap = map[TradeType]string{
	TradeTypeBuy:  "buy",
	TradeTypeSell: "sell",
}
