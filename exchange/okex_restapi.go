package exchange

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

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

	log.Printf("Body:%v", string(body))

	return nil, body
}

func (p *OkexRestAPI) GetPosition(pair string, contract_type string) map[string]float64 {

	coins := ParsePair(pair)
	symbol := coins[0] + "_" + coins[1]

	parameters := map[string]string{
		"symbol":        symbol,
		"contract_type": contract_type,
		"api_key":       p.apiKey,
	}

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

	}

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

		if !values["result"].(bool) {
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

// GetBalance() get the balances of all the coins
func (p *OkexRestAPI) GetBalance() map[string]interface{} {
	return map[string]interface{}{
		"helo": "wolrd",
	}
}

// Trade() trade as the configs
func (p *OkexRestAPI) Trade(configs TradeConfig) *TradeResult {
	return nil
}

// CancelOrder() cancel the order as the order information
func (p *OkexRestAPI) CancelOrder(order OrderInfo) *TradeResult {
	return nil
}

// GetOrderInfo() get the information with order filter
func (p *OkexRestAPI) GetOrderInfo(filter OrderInfo) []OrderInfo {
	return nil
}
