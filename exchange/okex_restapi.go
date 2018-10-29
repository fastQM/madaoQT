package exchange

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

const OkexURL = "https://www.okex.com/api/v1/"
const OkexRest = "OkexRest"

type OkexRestAPI struct {
	event  chan EventType
	config Config

	apiKey    string
	secretKey string
}

func (p *OkexRestAPI) GetExchangeName() string {
	return OkexRest
}

// WatchEvent() return a channel which notified the application of the event triggered by exchange
func (p *OkexRestAPI) WatchEvent() chan EventType {
	return p.event
}

func (h *OkexRestAPI) Start() error {
	return nil
}

// SetConfigure()
func (p *OkexRestAPI) SetConfigure(config Config) {

	p.config = config
	p.apiKey = config.API
	p.secretKey = config.Secret

	if p.config.Proxy != "" {
		logger.Infof("Proxy:%s", p.config.Proxy)
	}
}

func (p *OkexRestAPI) marketRequest(path string, params map[string]string) (error, []byte) {

	var req http.Request
	req.ParseForm()
	for k, v := range params {
		req.Form.Add(k, v)
	}
	bodystr := strings.TrimSpace(req.Form.Encode())
	// logger.Debugf("Params:%v", bodystr)
	request, err := http.NewRequest("GET", OkexURL+path+"?"+bodystr, nil)
	if err != nil {
		return err, nil
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36")

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

	return nil, body
}

func (p *OkexRestAPI) tradeRequest(path string, params map[string]string) (error, []byte) {

	var req http.Request
	req.ParseForm()
	for k, v := range params {
		req.Form.Add(k, v)
	}
	bodystr := strings.TrimSpace(req.Form.Encode())
	logger.Debugf("Params:%v", bodystr)
	request, err := http.NewRequest("POST", OkexURL+path, strings.NewReader(bodystr))
	if err != nil {
		return err, nil
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36")

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

	logger.Infof("Body:%v", string(body))

	return nil, body
}

func (p *OkexRestAPI) GetPosition(pair string, contract_type string) map[string]float64 {

	coins := ParsePair(pair)
	symbol := coins[0] + "_" + coins[1]

	parameters := p.sign(map[string]string{
		"symbol":        symbol,
		"contract_type": contract_type,
		"api_key":       p.apiKey,
	})

	if err, response := p.tradeRequest("future_position_4fix.do", parameters); err != nil {
		logger.Errorf("Invalid response:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if response != nil {
			if err = json.Unmarshal(response, &values); err != nil {
				logger.Errorf("Fail to Unmarshal:%v", err)
				return nil
			}
		}

		if !values["result"].(bool) || values["holding"] == nil || len(values["holding"].([]interface{})) == 0 {
			logger.Error("Fail to get position")
			return nil
		}

		return map[string]float64{
			"long":     values["holding"].([]interface{})[0].(map[string]interface{})["buy_amount"].(float64),
			"short":    values["holding"].([]interface{})[0].(map[string]interface{})["sell_amount"].(float64),
			"buy_avg":  values["holding"].([]interface{})[0].(map[string]interface{})["buy_price_avg"].(float64),
			"sell_avg": values["holding"].([]interface{})[0].(map[string]interface{})["sell_price_avg"].(float64),
		}
	}
}

func (p *OkexRestAPI) GetKline(pair string, period int, limit int) []KlineValue {
	pair = strings.Replace(pair, "usdt", "usd", 1)
	coins := ParsePair(pair)
	symbol := coins[0] + "_" + coins[1]

	var interval string
	switch period {
	case KlinePeriod5Min:
		interval = "5min"
	case KlinePeriod15Min:
		interval = "15min"
	case KlinePeriod30Min:
		interval = "30min"
	case KlinePeriod1Hour:
		interval = "1hour"
	case KlinePeriod2Hour:
		interval = "2hour"
	case KlinePeriod4Hour:
		interval = "4hour"
	case KlinePeriod1Day:
		interval = "1day"
	}

	params := map[string]string{
		"symbol":        symbol,
		"type":          interval,
		"contract_type": "quarter",
	}

	if limit != 0 {
		params["size"] = strconv.Itoa(limit)
	}

	if err, response := p.marketRequest("future_kline.do", params); err != nil {
		logger.Errorf("Invalid response:%v", err)
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
				kline[i].OpenTime = value[0].(float64) / 1000
				kline[i].Open = value[1].(float64)
				kline[i].High = value[2].(float64)
				kline[i].Low = value[3].(float64)
				kline[i].Close = value[4].(float64)
				kline[i].Volumn = value[5].(float64)
			}

			return kline
		}

		return nil
	}
}

// Close() close the connection to the exchange and other handles
func (p *OkexRestAPI) Close() {

}

// StartTicker() send message to the exchange to start the ticker of the given pairs
func (p *OkexRestAPI) StartTicker(pair string) {
}

// GetTicker(), better to use the ITicker to notify the ticker information
func (p *OkexRestAPI) GetTicker(pair string) *TickerValue {
	return nil
}

// GetDepthValue() get the depth of the assigned price area and quantity
// GetDepthValue(pair string, price float64, limit float64, orderQuantity float64, tradeType TradeType) []DepthPrice
func (p *OkexRestAPI) GetDepthValue(pair string) [][]DepthPrice {
	return nil
}

func (p *OkexRestAPI) sign(parameters map[string]string) map[string]string {

	if parameters != nil {
		var keys []string
		var signPlain string

		for k := range parameters {
			keys = append(keys, k)
		}

		sort.Strings(keys)

		for _, key := range keys {
			if key == "sign" {
				continue
			}
			signPlain += (key + "=" + parameters[key])
			signPlain += "&"
		}

		signPlain += ("secret_key=" + p.secretKey)

		// log.Printf("Plain:%v", signPlain)
		md5Value := fmt.Sprintf("%x", md5.Sum([]byte(signPlain)))
		// log.Printf("MD5:%v", md5Value)
		parameters["sign"] = strings.ToUpper(md5Value)

		return parameters
	}

	return nil
}

// GetBalance() get the balances of all the coins
func (p *OkexRestAPI) GetBalance() map[string]interface{} {

	parameters := p.sign(map[string]string{
		"api_key": p.apiKey,
	})

	if err, response := p.tradeRequest("future_userinfo_4fix.do", parameters); err != nil {
		logger.Errorf("Invalid response:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if response != nil {
			if err = json.Unmarshal(response, &values); err != nil {
				logger.Errorf("Fail to Unmarshal:%v", err)
				return nil
			}
		}

		if values["result"] != nil && !values["result"].(bool) {
			logger.Error("Fail to get order info")
			return nil
		}

		if values["info"] != nil {
			balance := values["info"].(map[string]interface{})

			result := map[string]interface{}{}

			for key, value := range balance {
				result[key] = value.(map[string]interface{})["rights"].(float64)
			}

			return result
		}

		return nil

	}
}

// Trade() trade as the configs
func (p *OkexRestAPI) Trade(configs TradeConfig) *TradeResult {
	coins := ParsePair(configs.Pair)

	parameters := p.sign(map[string]string{
		"symbol":        coins[0] + "_usd",
		"contract_type": "quarter",
		"api_key":       p.apiKey,
		"price":         strconv.FormatFloat(configs.Price, 'f', 4, 64),
		"amount":        strconv.FormatFloat(configs.Amount, 'f', 4, 64),
		"type":          OkexGetTradeTypeString(configs.Type),
		// "match_price":   "1",
	})

	if err, response := p.tradeRequest("future_trade.do", parameters); err != nil {
		logger.Errorf("Invalid response:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if response != nil {
			if err = json.Unmarshal(response, &values); err != nil {
				logger.Errorf("Fail to Unmarshal:%v", err)
				return nil
			}
		}

		if !values["result"].(bool) {
			logger.Error("Fail to get position")
			errMsg := fmt.Sprintf("Error code:%f", values["error_code"].(float64))
			return &TradeResult{
				Error: errors.New(errMsg),
			}
		}

		orderId := strconv.FormatFloat(values["order_id"].(float64), 'f', 0, 64)
		return &TradeResult{
			Error:   nil,
			OrderID: orderId,
		}

	}
}

// CancelOrder() cancel the order as the order information
func (p *OkexRestAPI) CancelOrder(order OrderInfo) *TradeResult {
	return nil
}

// GetOrderInfo() get the information with order filter
func (p *OkexRestAPI) GetOrderInfo(filter OrderInfo) []OrderInfo {
	coins := ParsePair(filter.Pair)

	parameters := p.sign(map[string]string{
		"symbol":        coins[0] + "_usd",
		"contract_type": "quarter",
		"api_key":       p.apiKey,
		"order_id":      filter.OrderID,
	})

	if err, response := p.tradeRequest("future_order_info.do", parameters); err != nil {
		logger.Errorf("Invalid response:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if response != nil {
			if err = json.Unmarshal(response, &values); err != nil {
				logger.Errorf("Fail to Unmarshal:%v", err)
				return nil
			}
		}

		if values["result"] != nil && !values["result"].(bool) {
			logger.Error("Fail to get order info")
			return nil
		}

		if values["orders"] != nil {
			orders := values["orders"].([]interface{})

			if len(orders) == 0 {
				logger.Error("The order info is not found")
				return nil
			}

			result := make([]OrderInfo, len(orders))

			for i, tmp := range orders {
				order := tmp.(map[string]interface{})

				var orderType TradeType
				var avgPrice float64

				orderType = OkexGetTradeTypeByFloat(order["type"].(float64))
				avgPrice = order["price_avg"].(float64)

				item := OrderInfo{
					Pair:    order["symbol"].(string),
					OrderID: strconv.FormatFloat(order["order_id"].(float64), 'f', 0, 64),
					// OrderID: strconv.FormatInt(order["order_id"].(int64), 64),
					Price:      order["price"].(float64),
					Amount:     order["amount"].(float64),
					Type:       orderType,
					Status:     OkexGetTradeStatus(order["status"].(float64)),
					DealAmount: order["deal_amount"].(float64),
					AvgPrice:   avgPrice,
				}
				result[i] = item
			}

			return result
		}

		logger.Error("Invalid order info")
		return nil

	}
}

func SwithMinutesToHourKlines(klines []KlineValue) []KlineValue {
	var KlinesByHour []KlineValue

	var high, low, open, close float64
	// var klineTime time.Time

	location, _ := time.LoadLocation("Asia/Shanghai")
	var first time.Time

	for i, kline := range klines {
		first = time.Unix(int64(kline.OpenTime), 0).In(location)
		if first.Minute() == 0 { // start from 01:33
			klines = klines[i:]
			break
		}

	}

	for i, kline := range klines {

		// klineTime = time.Unix(int64(kline.OpenTime), 0).In(location)
		if open == 0 {
			open = kline.Open
		}
		// log.Printf("Time:%v", klineTime)
		if high == 0 || high < kline.High {
			high = kline.High
		}

		if low == 0 || low > kline.Low {
			low = kline.Low
		}

		close = kline.Close

		if i+1 < len(klines) {
			nextTime := time.Unix(int64(klines[i+1].OpenTime), 0).In(location)
			if nextTime.Hour() != first.Hour() {
				if high != 0 && low != 0 && close != 0 {
					lastKline := KlineValue{
						High:     high,
						Low:      low,
						Open:     open,
						Close:    close,
						OpenTime: float64(time.Date(first.Year(), first.Month(), first.Day(), first.Hour(), 0, 0, 0, location).Unix()),
					}
					KlinesByHour = append(KlinesByHour, lastKline)
				}
				first = nextTime
				open = 0
				high = 0
				low = 0
				close = 0
			}
		} else {
			if high != 0 && low != 0 && close != 0 {
				lastKline := KlineValue{
					High:     high,
					Low:      low,
					Open:     open,
					Close:    close,
					OpenTime: float64(time.Date(first.Year(), first.Month(), first.Day(), first.Hour(), 0, 0, 0, location).Unix()),
				}
				KlinesByHour = append(KlinesByHour, lastKline)
			}
		}

	}

	return KlinesByHour
}

func Swith1HourToDialyKlines(klines []KlineValue) []KlineValue {
	var KlinesByDate []KlineValue

	var high, low, open, close float64
	// var klineTime time.Time

	location, _ := time.LoadLocation("Asia/Shanghai")
	var first time.Time

	for i, kline := range klines {
		first = time.Unix(int64(kline.OpenTime), 0).In(location)
		if first.Hour() == 0 { // start from 01:33
			klines = klines[i:]
			break
		}

	}

	for i, kline := range klines {

		// klineTime = time.Unix(int64(kline.OpenTime), 0).In(location)
		if open == 0 {
			open = kline.Open
		}
		// log.Printf("Time:%v", klineTime)
		if high == 0 || high < kline.High {
			high = kline.High
		}

		if low == 0 || low > kline.Low {
			low = kline.Low
		}

		close = kline.Close

		if i+1 < len(klines) {
			nextTime := time.Unix(int64(klines[i+1].OpenTime), 0).In(location)
			if nextTime.Day() != first.Day() {
				if high != 0 && low != 0 && close != 0 {
					lastKline := KlineValue{
						High:     high,
						Low:      low,
						Open:     open,
						Close:    close,
						OpenTime: float64(time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, location).Unix()),
					}
					KlinesByDate = append(KlinesByDate, lastKline)
				}
				first = nextTime
				open = 0
				high = 0
				low = 0
				close = 0
			}
		} else {
			if high != 0 && low != 0 && close != 0 {
				lastKline := KlineValue{
					High:     high,
					Low:      low,
					Open:     open,
					Close:    close,
					OpenTime: float64(time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, location).Unix()),
				}
				KlinesByDate = append(KlinesByDate, lastKline)
			}
		}

	}

	return KlinesByDate
}

func Swith1HourToHoursKlines(hours int, klines []KlineValue) []KlineValue {
	var KlinesByHours []KlineValue

	var high, low, open, close float64
	var klineTime time.Time

	location, _ := time.LoadLocation("Asia/Shanghai")
	var first time.Time

	if hours == 0 {
		return nil
	}

	for i, kline := range klines {
		first = time.Unix(int64(kline.OpenTime), 0).In(location)
		if first.Hour() == 0 { // start from 01:33
			klines = klines[i:]
			break
		}

	}

	for i, kline := range klines {
		klineTime = time.Unix(int64(kline.OpenTime), 0).In(location)
		if open == 0 {
			open = kline.Open
		}
		// log.Printf("Time:%v", klineTime)
		if high == 0 || high < kline.High {
			high = kline.High
		}

		if low == 0 || low > kline.Low {
			low = kline.Low
		}

		close = kline.Close

		if klineTime.Hour()%hours == (hours - 1) {
			if high != 0 && low != 0 && close != 0 {
				lastKline := KlineValue{
					High:     high,
					Low:      low,
					Open:     open,
					Close:    close,
					OpenTime: float64(time.Date(first.Year(), first.Month(), first.Day(), first.Hour(), 0, 0, 0, location).Unix()),
				}
				KlinesByHours = append(KlinesByHours, lastKline)
			}
			if i+1 < len(klines) {
				first = time.Unix(int64(klines[i+1].OpenTime), 0).In(location)
			}

			open = 0
			high = 0
			low = 0
			close = 0

		}

	}

	// okex 包含了最新的k线数据，所以不用担心有时间但没有数据的问题
	if high != 0 && low != 0 && close != 0 {
		lastKline := KlineValue{
			High:     high,
			Low:      low,
			Open:     open,
			Close:    close,
			OpenTime: float64(time.Date(first.Year(), first.Month(), first.Day(), first.Hour(), 0, 0, 0, location).Unix()),
		}
		KlinesByHours = append(KlinesByHours, lastKline)
	}

	return KlinesByHours
}

func SwithDialyToWeekKlines(klines []KlineValue) []KlineValue {
	var KlinesByWeek []KlineValue

	var high, low, open, close float64
	var klineTime time.Time

	location, _ := time.LoadLocation("Asia/Shanghai")
	var first time.Time

	if klines == nil || len(klines) == 0 {
		return nil
	}

	for i, kline := range klines {
		first = time.Unix(int64(kline.OpenTime), 0).In(location)
		if first.Weekday() == time.Monday { // start from 01:33
			klines = klines[i:]
			break
		}

	}

	for i, kline := range klines {
		klineTime = time.Unix(int64(kline.OpenTime), 0).In(location)
		if open == 0 {
			open = kline.Open
		}
		// log.Printf("Time:%v", klineTime)
		if high == 0 || high < kline.High {
			high = kline.High
		}

		if low == 0 || low > kline.Low {
			low = kline.Low
		}

		close = kline.Close

		if klineTime.Weekday() == time.Friday {
			if high != 0 && low != 0 && close != 0 {
				lastKline := KlineValue{
					High:     high,
					Low:      low,
					Open:     open,
					Close:    close,
					OpenTime: float64(time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, location).Unix()),
				}
				KlinesByWeek = append(KlinesByWeek, lastKline)
			}
			if i+1 < len(klines) {
				first = time.Unix(int64(klines[i+1].OpenTime), 0).In(location)
			}

			open = 0
			high = 0
			low = 0
			close = 0

		}

	}

	// okex 包含了最新的k线数据，所以不用担心有时间但没有数据的问题
	if high != 0 && low != 0 && close != 0 {
		lastKline := KlineValue{
			High:     high,
			Low:      low,
			Open:     open,
			Close:    close,
			OpenTime: float64(time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, location).Unix()),
		}
		KlinesByWeek = append(KlinesByWeek, lastKline)
	}

	return KlinesByWeek
}
