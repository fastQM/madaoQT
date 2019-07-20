package exchange

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	Websocket "github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

const NameInteractiveBrokers = "InteractiveBrokers"
const InteractiveBrokersURL = "https://localhost:5000/v1/portal"

type InteractiveBrokers struct {
	Proxy     string
	websocket *Websocket.Conn
	event     chan EventType
	config    Config

	uid string
}

func (p *InteractiveBrokers) GetExchangeName() string {
	return NameInteractiveBrokers
}

// SetConfigure()
func (p *InteractiveBrokers) SetConfigure(config Config) {
	p.config = config

	if p.config.Proxy != "" {
		logger.Infof("使用代理:%s", p.config.Proxy)
	}
}

// WatchEvent() return a channel which notified the application of the event triggered by exchange
func (p *InteractiveBrokers) WatchEvent() chan EventType {
	return p.event
}

func (h *InteractiveBrokers) Start() error {
	return nil
}

func (p *InteractiveBrokers) marketRequest(path string, params map[string]string) (error, []byte) {
	var req http.Request
	req.ParseForm()
	for k, v := range params {
		req.Form.Add(k, v)
	}
	bodystr := strings.TrimSpace(req.Form.Encode())
	logger.Debugf("Params:%v", bodystr)
	request, err := http.NewRequest("GET", InteractiveBrokersURL+path+"?"+bodystr, nil)
	if err != nil {
		return err, nil
	}

	// setup a http client
	httpTransport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	httpClient := &http.Client{
		Transport: httpTransport,
		Timeout:   10 * time.Second,
	}

	if p.Proxy != "" {
		values := strings.Split(p.Proxy, ":")
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

func (p *InteractiveBrokers) orderRequest(path string, params map[string]interface{}) (error, []byte) {

	postBody, _ := json.Marshal(params)
	request, err := http.NewRequest("POST", InteractiveBrokersURL+path, bytes.NewReader(postBody))
	if err != nil {
		return err, nil
	}

	request.Header.Add("Content-Type", "application/json")

	httpTransport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: httpTransport}

	if p.Proxy != "" {
		values := strings.Split(p.Proxy, ":")
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
func (p *InteractiveBrokers) Close() {

}

// StartTicker() send message to the exchange to start the ticker of the given pairs
func (p *InteractiveBrokers) StartTicker(pair string) {
}

// GetTicker(), better to use the ITicker to notify the ticker information
func (p *InteractiveBrokers) GetTicker(pair string) *TickerValue {
	return nil
}

func (p *InteractiveBrokers) getSymbol(pair string) string {
	coins := ParsePair(pair)
	return strings.ToUpper(coins[0] + coins[1])
}

func (p *InteractiveBrokers) GetDepthValue(pair string) [][]DepthPrice {
	return nil
}

// GetBalance() get the balances of all the coins
func (p *InteractiveBrokers) GetBalance() map[string]interface{} {
	return p.GetPortfolio(p.uid)
}

// Trade() trade as the configs
func (p *InteractiveBrokers) Trade(configs TradeConfig, conType string) *TradeResult {

	path := "/iserver/account/" + p.uid + "/order"
	// ticker := time.Now().Unix()

	conid, _ := strconv.ParseInt(configs.Pair, 10, 64)
	parameters := map[string]interface{}{
		// "acctId":    "",
		"conid":     conid,
		"secType":   configs.Pair + ":" + conType,
		"cOID":      configs.Batch,
		"orderType": "MKT",
		// "listingExchange", "",
		"outsideRTH": false,
		// "price":      "",
		"side":     OkexGetTradeTypeString(configs.Type),
		"ticker":   "",
		"tif":      "DAY",
		"referrer": "QuickTrade",
		"quantity": configs.Amount,
	}

	if err, response := p.orderRequest(path, parameters); err != nil {
		logger.Errorf("无法获取余额:%v", err)
		return nil
	} else {

		var errMessage map[string]interface{}
		if err = json.Unmarshal(response, &errMessage); err == nil { // 有错误才会被成功解析
			logger.Errorf("订单错误:%v", errMessage["error"].(string))
			return nil
		}

		var values []interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		return &TradeResult{
			Error:   nil,
			OrderID: values[0].(map[string]interface{})["id"].(string),
		}
	}

	return nil
}

func (p *InteractiveBrokers) TradeConfirm(replyID string, confirm bool) *OrderInfo {

	path := "/iserver/reply/" + replyID
	// ticker := time.Now().Unix()

	parameters := map[string]interface{}{
		"confirmed": confirm,
	}

	if err, response := p.orderRequest(path, parameters); err != nil {
		logger.Errorf("无法获取余额:%v", err)
		return nil
	} else {

		var errMessage map[string]interface{}
		if err = json.Unmarshal(response, &errMessage); err == nil { // 有错误才会被成功解析
			logger.Errorf("订单错误:%v", errMessage["error"].(string))
			return nil
		}

		var values []interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		// logger.Infof("Result:%v", values)
		return &OrderInfo{
			OrderID: values[0].(map[string]interface{})["order_id"].(string),
			Status:  p.getStatusType(values[0].(map[string]interface{})["order_status"].(string)),
		}
	}

	return nil
}

// CancelOrder() cancel the order as the order information
func (p *InteractiveBrokers) CancelOrder(order OrderInfo) *TradeResult {
	return nil
}

// GetOrderInfo() get the information with order filter
func (p *InteractiveBrokers) GetOrderInfo(filter OrderInfo) []OrderInfo {
	positions := p.GetPositionByConid(filter.Pair)
	if positions != nil {
		result := make([]OrderInfo, 1)

		// orderType := values["type"].(string)
		// placePrice, _ := strconv.ParseFloat(values["price"].(string), 64)
		// amount, _ := strconv.ParseFloat(values["field-cash-amount"].(string), 64)
		dealAmount := positions[0].(map[string]interface{})["position"].(float64)
		avgPrice := positions[0].(map[string]interface{})["avgPrice"].(float64)
		// status := values["state"].(string)

		item := OrderInfo{
			// Pair:       values["symbol"].(string),
			// OrderID:    filter.OrderID,
			// Price:      placePrice,
			// Amount:     amount,
			// Type:       p.GetTradeType(orderType),
			// Status:     p.GetOrderStatus(status),
			DealAmount: dealAmount,
			AvgPrice:   avgPrice,
		}

		result[0] = item
		return result
	}
	return nil
}

func (p *InteractiveBrokers) GetKline(pair string, period int, limit int) []KlineValue {
	return nil
}

var InteractiveBrokersOrderStatusMap = map[OrderStatusType]string{
	OrderStatusOpen: "Submitted",
	// OrderStatusPartDone:  "PARTIALLY_FILLED",
	OrderStatusDone:      "Filled",
	OrderStatusCanceling: "PendingCancel",
	OrderStatusCanceled:  "Cancelled",
	// OrderStatusRejected:  "REJECTED",
	// OrderStatusExpired:   "EXPIRED",
}

func (p *InteractiveBrokers) getStatusType(key string) OrderStatusType {
	for k, v := range InteractiveBrokersOrderStatusMap {
		if v == key {
			return k
		}
	}
	return OrderStatusUnknown
}

var InteractiveBrokersTradeTypeMap = map[TradeType]string{
	TradeTypeBuy:  "BUY",
	TradeTypeSell: "SELL",
}

func (p *InteractiveBrokers) GetAccountUID() (error, string) {
	if err, response := p.marketRequest("/iserver/accounts", map[string]string{}); err != nil {
		logger.Errorf("Invalid request:%v", err)
		return err, ""
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse:%v", err)
			return err, ""
		}

		if values["accounts"] != nil {
			accounts := values["accounts"].([]interface{})
			if len(accounts) > 0 {
				p.uid = accounts[0].(string)
				return nil, accounts[0].(string)
			}

		}

		return errors.New("Invalid account"), ""

	}
}

func (p *InteractiveBrokers) GetPortfolio(uid string) map[string]interface{} {
	if err, response := p.marketRequest("/portfolio/"+uid+"/ledger", map[string]string{}); err != nil {
		logger.Errorf("Invalid request:%v", err)
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

func (p *InteractiveBrokers) GetAccountInformation(uid string) map[string]interface{} {
	if err, response := p.marketRequest("/portfolio/"+uid+"/meta", map[string]string{}); err != nil {
		logger.Errorf("Invalid request:%v", err)
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

func (p *InteractiveBrokers) GetAccountSummary(uid string) map[string]interface{} {
	if err, response := p.marketRequest("/portfolio/"+uid+"/summary", map[string]string{}); err != nil {
		logger.Errorf("Invalid request:%v", err)
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

func (p *InteractiveBrokers) GetAccountLedger(uid string) map[string]interface{} {
	if err, response := p.marketRequest("/portfolio/"+uid+"/ledger", map[string]string{}); err != nil {
		logger.Errorf("Invalid request:%v", err)
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

func (p *InteractiveBrokers) GetMarketData(conids []string) []interface{} {

	var list string
	for _, conid := range conids {
		if list == "" {
			list = conid
		} else {
			list += ("," + conid)
		}
	}

	if err, response := p.marketRequest("/iserver/marketdata/snapshot", map[string]string{
		"conids": list,
	}); err != nil {
		logger.Errorf("Invalid request:%v", err)
		return nil
	} else {
		var values []interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse:%v", err)
			return nil
		}

		return values
	}
}

func (p *InteractiveBrokers) GetLiveOrders() map[string]interface{} {
	if err, response := p.marketRequest("/iserver/account/orders", map[string]string{}); err != nil {
		logger.Errorf("Invalid request:%v", err)
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

func (p *InteractiveBrokers) SearchBySymbol(symbol string) []interface{} {
	if err, response := p.marketRequest("/iserver/secdef/search", map[string]string{
		"symbol": symbol,
	}); err != nil {
		logger.Errorf("Invalid request:%v", err)
		return nil
	} else {
		var values []interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse:%v", err)
			return nil
		}

		return values
	}
}

func (p *InteractiveBrokers) GetPositionByConid(conid string) []interface{} {
	if err, response := p.marketRequest("/portfolio/"+p.uid+"/position/"+conid, map[string]string{}); err != nil {
		logger.Errorf("Invalid request:%v", err)
		return nil
	} else {
		var values []interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse:%v", err)
			return nil
		}

		return values
	}
}

func (p *InteractiveBrokers) GetAllPositions() []interface{} {
	if err, response := p.marketRequest("/portfolio/"+p.uid+"/positions/0", map[string]string{}); err != nil {
		logger.Errorf("Invalid request:%v", err)
		return nil
	} else {
		var values []interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse:%v", err)
			return nil
		}

		return values
	}
}

func (p *InteractiveBrokers) Validate() map[string]interface{} {
	if err, response := p.marketRequest("/sso/validate", map[string]string{}); err != nil {
		logger.Errorf("Invalid request:%v", err)
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

func (p *InteractiveBrokers) OneUser() map[string]interface{} {
	if err, response := p.marketRequest("/one/user", map[string]string{}); err != nil {
		logger.Errorf("Invalid request:%v", err)
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

func (p *InteractiveBrokers) PortfolioAccounts() []interface{} {
	if err, response := p.marketRequest("/portfolio/accounts", map[string]string{}); err != nil {
		logger.Errorf("Invalid request:%v", err)
		return nil
	} else {
		var values []interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("Fail to parse:%v", err)
			return nil
		}

		return values
	}
}

func (p *InteractiveBrokers) GetStatus() map[string]interface{} {
	if err, response := p.marketRequest("/iserver/auth/status", map[string]string{}); err != nil {
		logger.Errorf("Invalid request:%v", err)
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

func (p *InteractiveBrokers) Reauthenticate() map[string]interface{} {
	if err, response := p.marketRequest("/iserver/reauthenticate", map[string]string{}); err != nil {
		logger.Errorf("Invalid request:%v", err)
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
