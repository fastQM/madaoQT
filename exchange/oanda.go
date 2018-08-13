package exchange

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/net/proxy"
)

const OandaURL = "https://api-fxtrade.oanda.com"
const NameOdanda = "Oanda"

//REST API
//120 requests per second. Excess requests will receive HTTP 429 error. This restriction is applied against the requesting IP address.

type OandaAPI struct {
	event  chan EventType
	config Config
}

func (p *OandaAPI) GetExchangeName() string {
	return NameBinance
}

// SetConfigure()
func (p *OandaAPI) SetConfigure(config Config) {
	p.config = config

	if p.config.Proxy != "" {
		logger.Infof("Proxy:%s", p.config.Proxy)
	}
}

// WatchEvent() return a channel which notified the application of the event triggered by exchange
func (p *OandaAPI) WatchEvent() chan EventType {
	return p.event
}

func (h *OandaAPI) Start() error {
	return nil
}

func (h *OandaAPI) marketRequest(method, path string, params map[string]string) (error, []byte) {

	// log.Printf("Path:%s", path)
	var bodystr string
	for k, v := range params {
		if bodystr == "" {
			bodystr += (k + "=" + v)
		} else {
			bodystr += ("&" + k + "=" + v)
		}

	}
	// logger.Debugf("Params:%s auth[%s]", bodystr, "Bearer "+h.config.Custom["token"].(string))

	var request *http.Request
	var err error
	if method == "GET" {
		request, err = http.NewRequest(method, OandaURL+path+"?"+string(bodystr), nil)
		if err != nil {
			return err, nil
		}
	} else if method == "POST" || method == "PUT" {
		request, err = http.NewRequest(method, OandaURL+path, strings.NewReader(params["data"]))
		if err != nil {
			return err, nil
		}
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept-Datetime-Format", "UNIX")
	request.Header.Add("Authorization", "Bearer "+h.config.Custom["token"].(string))

	// setup a http client
	httpTransport := &http.Transport{}
	httpClient := &http.Client{Transport: httpTransport}

	if h.config.Proxy != "" {
		values := strings.Split(h.config.Proxy, ":")
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
		"prices",
	}

	filtered := false

	for _, keyword := range keywords {
		if strings.Contains(string(body), keyword) {
			filtered = true
			break
		}
	}

	if !filtered {
		logger.Infof("Body:%v", string(body))
	}

	// var value map[string]interface{}
	// if err = json.Unmarshal(body, &value); err != nil {
	// 	return err, nil
	// }

	return nil, body

}

// Close() close the connection to the exchange and other handles
func (p *OandaAPI) Close() {

}

// StartTicker() send message to the exchange to start the ticker of the given pairs
func (p *OandaAPI) StartTicker(pair string) {
}

// GetTicker(), better to use the ITicker to notify the ticker information
func (p *OandaAPI) GetTicker(pair string) *TickerValue {
	coins := ParsePair(pair)
	symbol := strings.ToUpper(coins[0] + "_" + coins[1])
	if err, response := p.marketRequest("GET", "/v3/accounts/"+p.config.Custom["account"].(string)+"/pricing", map[string]string{
		"instruments": symbol,
	}); err != nil {
		logger.Errorf("Fail to get ticker info:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse the json:%v", err)
			return nil
		}

		if values["prices"] != nil {
			prices := values["prices"].([]interface{})

			for _, tmp := range prices {
				price := tmp.(map[string]interface{})
				if price["instrument"].(string) == symbol {
					asks := price["asks"].([]interface{})
					bids := price["bids"].([]interface{})
					// updateTime, _ := strconv.ParseFloat(price["time"].(string), 64)
					askPrice, _ := strconv.ParseFloat(asks[0].(map[string]interface{})["price"].(string), 64)
					bidPrice, _ := strconv.ParseFloat(bids[0].(map[string]interface{})["price"].(string), 64)
					return &TickerValue{
						High: askPrice,
						Low:  bidPrice,
						Last: (askPrice + bidPrice) / 2,
						Time: price["time"].(string),
					}
				}
			}
		}
	}

	return nil
}

func (p *OandaAPI) getSymbol(pair string) string {
	coins := ParsePair(pair)
	return strings.ToUpper(coins[0] + coins[1])
}

// GetDepthValue() get the depth of the assigned price area and quantity
// GetDepthValue(pair string, price float64, limit float64, orderQuantity float64, tradeType TradeType) []DepthPrice
func (p *OandaAPI) GetDepthValue(pair string) [][]DepthPrice {
	return nil
}

// GetBalance() get the balances of all the coins
func (p *OandaAPI) GetBalance() map[string]interface{} {

	if err, response := p.marketRequest("GET", "/v3/accounts/"+p.config.Custom["account"].(string), map[string]string{}); err != nil {
		logger.Errorf("Fail to get account info:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse the json:%v", err)
			return nil
		}

		if values["account"] != nil {
			balance, _ := strconv.ParseFloat(values["account"].(map[string]interface{})["balance"].(string), 64)
			return map[string]interface{}{
				"balance": balance,
			}
		}

	}

	return nil
}

func (p *OandaAPI) openTrade(configs TradeConfig) *TradeResult {
	coins := ParsePair(configs.Pair)
	symbol := strings.ToUpper(coins[0] + "_" + coins[1])

	var amount string
	if configs.Type == TradeTypeOpenLong {
		amount = strconv.Itoa(int(configs.Amount))
	} else {
		amount = strconv.Itoa(int(configs.Amount) * -1)
	}
	body := map[string]interface{}{
		"order": map[string]string{
			"units":        amount,
			"instrument":   symbol,
			"timeInForce":  "FOK",
			"type":         "MARKET",
			"positionFill": "DEFAULT",
		},
	}

	data, _ := json.Marshal(body)
	log.Printf("%s", string(data))

	if err, response := p.marketRequest("POST", "/v3/accounts/"+p.config.Custom["account"].(string)+"/orders", map[string]string{
		"data": string(data),
	}); err != nil {
		logger.Errorf("Fail to open trade:%v", err)
		return &TradeResult{
			Error: err,
		}
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse json:%v", err)
			return &TradeResult{
				Error: err,
			}
		}

		if values["orderFillTransaction"] != nil {
			result := values["orderFillTransaction"].(map[string]interface{})
			price, _ := strconv.ParseFloat(result["price"].(string), 64)
			amount, _ := strconv.ParseFloat(result["units"].(string), 64)
			info := &OrderInfo{
				Pair:       result["instrument"].(string),
				OrderID:    result["orderID"].(string),
				DealAmount: amount,
				AvgPrice:   math.Abs(price),
			}

			return &TradeResult{
				Error: nil,
				// OrderID: values["clientOrderId"].(string),
				Info: info,
			}
		}
	}

	return &TradeResult{
		Error: errors.New("Invalid response"),
	}
}

func (p *OandaAPI) closeTrade(configs TradeConfig) *TradeResult {

	body := map[string]interface{}{
		"units": "ALL",
	}

	data, _ := json.Marshal(body)
	log.Printf("%s", string(data))

	if err, response := p.marketRequest("PUT",
		"/v3/accounts/"+p.config.Custom["account"].(string)+"/trades/"+configs.Batch+"/close", map[string]string{
			"data": string(data),
		}); err != nil {
		logger.Errorf("Fail to open trade:%v", err)
		return &TradeResult{
			Error: err,
		}
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse json:%v", err)
			return &TradeResult{
				Error: err,
			}
		}

		if values["orderFillTransaction"] != nil {
			result := values["orderFillTransaction"].(map[string]interface{})
			price, _ := strconv.ParseFloat(result["units"].(string), 64)
			amount, _ := strconv.ParseFloat(result["price"].(string), 64)
			info := &OrderInfo{
				Pair:       result["instrument"].(string),
				OrderID:    result["orderID"].(string),
				DealAmount: amount,
				AvgPrice:   price,
			}

			return &TradeResult{
				Error: nil,
				// OrderID: values["clientOrderId"].(string),
				Info: info,
			}
		}
	}

	return &TradeResult{
		Error: errors.New("Invalid response"),
	}
}

// Trade() trade as the configs
func (p *OandaAPI) Trade(configs TradeConfig) *TradeResult {
	if configs.Type == TradeTypeOpenLong || configs.Type == TradeTypeOpenShort {
		return p.openTrade(configs)
	} else if configs.Type == TradeTypeCloseLong || configs.Type == TradeTypeCloseShort {
		return p.closeTrade(configs)
	} else {
		log.Printf("Invalid trade type")
	}

	return nil
}

// CancelOrder() cancel the order as the order information
func (p *OandaAPI) CancelOrder(order OrderInfo) *TradeResult {
	return nil
}

// GetOrderInfo() get the information with order filter
func (p *OandaAPI) GetPositionInfo(filter OrderInfo) *OrderInfo {

	if err, response := p.marketRequest("GET",
		"/v3/accounts/"+p.config.Custom["account"].(string)+"/trades/"+filter.OrderID, map[string]string{}); err != nil {
		logger.Errorf("Fail to get account info:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse:%v", err)
			return nil
		}

		if values["trade"] != nil {
			trade := values["trade"].(map[string]interface{})
			order := &OrderInfo{}

			order.Pair = trade["instrument"].(string)
			order.AvgPrice, _ = strconv.ParseFloat(trade["price"].(string), 64)
			order.DealAmount, _ = strconv.ParseFloat(trade["currentUnits"].(string), 64)
			order.OrderID = trade["id"].(string)

			return order
		}

	}
	return nil
}

var oandaOrderStatus = map[OrderStatusType]string{
	OrderStatusDone:     "FILLED",
	OrderStatusCanceled: "CANCELLED",
	OrderStatusOpen:     "PENDING",
}

func oandaGetStatusFromString(status string) OrderStatusType {
	for key, value := range oandaOrderStatus {
		if value == status {
			return key
		}
	}
	return OrderStatusUnknown
}

func (p *OandaAPI) GetOrderInfo(filter OrderInfo) *OrderInfo {
	// get the status of the order and the transation id, so we can check the status of the trade/position
	if err, response := p.marketRequest("GET",
		"/v3/accounts/"+p.config.Custom["account"].(string)+"/orders/"+filter.OrderID, map[string]string{}); err != nil {
		logger.Errorf("Fail to get account info:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse:%v", err)
			return nil
		}

		if values["order"] != nil {
			trade := values["order"].(map[string]interface{})

			order := &OrderInfo{}
			order.Pair = trade["instrument"].(string)
			order.OrderID = trade["fillingTransactionID"].(string)
			order.Status = oandaGetStatusFromString(trade["state"].(string))

			return order

		}

	}
	return nil
}

func (p *OandaAPI) GetKline(pair string, period int, limit int) []KlineValue {
	coins := ParsePair(pair)
	symbol := strings.ToUpper(coins[0] + "_" + coins[1])

	var interval string

	switch period {
	case KlinePeriod5Min:
		interval = "M5"
	case KlinePeriod15Min:
		interval = "M15"
	case KlinePeriod1Hour:
		interval = "H1"
	case KlinePeriod2Hour:
		interval = "H2"
	case KlinePeriod6Hour:
		interval = "H6"
	case KlinePeriod1Day:
		interval = "D"
	}

	// The Price component(s) to get candlestick data for.
	// Can contain any combination of the characters “M” (midpoint candles) “B” (bid candles) and “A” (ask candles).
	// [default=M]

	// The number of candlesticks to return in the reponse.
	// Count should not be specified if both the start and end parameters are provided,
	// as the time range combined with the graularity will determine the number of candlesticks to return.
	// [default=500, maximum=5000]

	if err, response := p.marketRequest("GET", "/v3/instruments/"+symbol+"/candles", map[string]string{
		"granularity": interval,
		"count":       strconv.Itoa(limit),
	}); err != nil {
		logger.Errorf("Invalid klines:%v", err)
		return nil
	} else {
		if response != nil {
			var values map[string]interface{}
			if err = json.Unmarshal(response, &values); err != nil {
				logger.Errorf("Fail to Unmarshal:%v", err)
				return nil
			}

			if values["candles"] == nil {
				logger.Errorf("Invalid kline datas")
				return nil
			}

			datas := values["candles"].([]interface{})

			kline := make([]KlineValue, len(datas))
			for i, data := range datas {
				// 	// kline[i].OpenTime = time.Unix((int64)(value[0].(float64)/1000), 0).Format(Global.TimeFormat)
				kline[i].OpenTime, _ = strconv.ParseFloat(data.(map[string]interface{})["time"].(string), 64)
				prices := data.(map[string]interface{})["mid"].(map[string]interface{})
				kline[i].Open, _ = strconv.ParseFloat(prices["o"].(string), 64)
				kline[i].High, _ = strconv.ParseFloat(prices["h"].(string), 64)
				kline[i].Low, _ = strconv.ParseFloat(prices["l"].(string), 64)
				kline[i].Close, _ = strconv.ParseFloat(prices["c"].(string), 64)
				kline[i].Volumn = data.(map[string]interface{})["volume"].(float64)
				// 	// kline[i].CloseTime = time.Unix((int64)(value[6].(float64)/1000), 0).Format(Global.TimeFormat)

			}

			return kline
		}

		return nil
	}
}

func (p *OandaAPI) getStatusType(key string) OrderStatusType {
	for k, v := range BinanceOrderStatusMap {
		if v == key {
			return k
		}
	}
	return OrderStatusUnknown
}

func (p *OandaAPI) GetAccountInfo() map[string]interface{} {
	if err, response := p.marketRequest("GET", "/v3/accounts/"+p.config.Custom["account"].(string), map[string]string{}); err != nil {
		logger.Errorf("Fail to get account info:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse:%v", err)
			return nil
		}

		return values

	}
}

func (p *OandaAPI) GetInstruments() map[string]interface{} {
	if err, response := p.marketRequest("GET", "/v3/accounts/"+p.config.Custom["account"].(string)+"/instruments", map[string]string{}); err != nil {
		logger.Errorf("无法获取账户信息:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		return values

	}
}
