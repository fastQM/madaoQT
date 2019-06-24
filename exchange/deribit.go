package exchange

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	Websocket "github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

const NameDeribit = "deribitV2"
const DeribitWebsocketPath = "wss://www.deribit.com/ws/api/v2"

type DeribitInstrumentType string

const (
	DeribitInstrumentBTC DeribitInstrumentType = "BTC"
	DeribitInstrumentETH DeribitInstrumentType = "ETH"
)

type DeribitV2API struct {
	Proxy     string
	ApiKey    string
	SecretKey string
	Passphare string

	accessToken  string
	refreshToken string

	conn        *Websocket.Conn
	depthValues map[string]*sync.Map
	channelMap  sync.Map

	FutureIndex string // 季度合约的日期

	// command
	commandID int
}

func (o *DeribitV2API) mergeDepth(oldList [][]DepthPrice, updateList [][]DepthPrice) [][]DepthPrice {

	newList := make([][]DepthPrice, 2)

	oldAsks := oldList[DepthTypeAsks]
	oldBids := oldList[DepthTypeBids]

	updateAsks := updateList[DepthTypeAsks]
	updateBids := updateList[DepthTypeBids]

	var lastPosition int
	quit := false
	for current, updateAsk := range updateAsks {
		for i := lastPosition; i < len(oldAsks); i++ {
			// log.Printf("ASK[%d] Update:%v OldIndex[%v]%v", len(oldAsks), updateAsk, i, oldAsks[i])
			if updateAsk.Price == oldAsks[i].Price && updateAsk.Quantity != 0 {
				// 非0替换
				newAsk := DepthPrice{
					Price:    updateAsk.Price,
					Quantity: updateAsk.Quantity,
				}
				newList[DepthTypeAsks] = append(newList[DepthTypeAsks], newAsk)
				lastPosition = i + 1
				break
			} else if updateAsk.Price == oldAsks[i].Price && updateAsk.Quantity == 0 {
				// 0删除
				lastPosition = i + 1
				break // 退出初始深度的判断
			} else if updateAsk.Price < oldAsks[i].Price && updateAsk.Quantity != 0 {
				newList[DepthTypeAsks] = append(newList[DepthTypeAsks], updateAsk)
				lastPosition = i
				break // 退出初始深度的判断
			} else if updateAsk.Price > oldAsks[i].Price {
				newList[DepthTypeAsks] = append(newList[DepthTypeAsks], oldAsks[i])
				if i < len(oldAsks)-1 {
					continue // 与初始深度的下一个值进行比较
				} else {
					newList[DepthTypeAsks] = append(newList[DepthTypeAsks], updateAsks[current:]...)
					quit = true
				}
			}
		}

		if quit {
			break
		}

	}

	if lastPosition != len(oldAsks)-1 {
		newList[DepthTypeAsks] = append(newList[DepthTypeAsks], oldAsks[lastPosition:]...)
	}

	if len(newList[DepthTypeAsks]) > 200 {
		newList[DepthTypeAsks] = newList[DepthTypeAsks][:200]
	}

	lastPosition = 0

	quit = false
	for current, updateBid := range updateBids {
		for i := lastPosition; i < len(oldBids); i++ {
			// log.Printf("BID[%d]Update:%v OldIndex[%v]%v", len(oldBids), updateBid, i, oldBids[i])
			if updateBid.Price == oldBids[i].Price && updateBid.Quantity != 0 {
				// 非0替换
				newBid := DepthPrice{
					Price:    updateBid.Price,
					Quantity: updateBid.Quantity,
				}
				newList[DepthTypeBids] = append(newList[DepthTypeBids], newBid)
				lastPosition = i + 1
				break
			} else if updateBid.Price == oldBids[i].Price && updateBid.Quantity == 0 {
				// 0删除
				lastPosition = i + 1
				break // 退出初始深度的判断
			} else if updateBid.Price > oldBids[i].Price && updateBid.Quantity != 0 {
				newList[DepthTypeBids] = append(newList[DepthTypeBids], updateBid)
				lastPosition = i
				break // 退出初始深度的判断
			} else if updateBid.Price < oldBids[i].Price {
				newList[DepthTypeBids] = append(newList[DepthTypeBids], oldBids[i])
				if i < len(oldBids)-1 {
					continue // 与初始深度的下一个值进行比较
				} else { // i == len(oldBids)-1
					newList[DepthTypeBids] = append(newList[DepthTypeBids], updateBids[current:]...)
					quit = true
				}
			}
		}

		if quit {
			break
		}

	}

	if lastPosition != len(oldBids)-1 {
		newList[DepthTypeBids] = append(newList[DepthTypeBids], oldBids[lastPosition:]...)
	}

	if len(newList[DepthTypeBids]) > 200 {
		newList[DepthTypeBids] = newList[DepthTypeBids][:200]
	}

	return newList
}

func (o *DeribitV2API) Start() error {
	return nil
}

func (o *DeribitV2API) Start2(errChan chan EventType) error {

	o.depthValues = make(map[string]*sync.Map)

	dialer := Websocket.DefaultDialer

	if o.Proxy != "" {
		logger.Infof("Proxy:%s", o.Proxy)
		values := strings.Split(o.Proxy, ":")
		if values[0] == "SOCKS5" {
			proxy, err := proxy.SOCKS5("tcp", values[1]+":"+values[2], nil, proxy.Direct)
			if err != nil {
				return err
			}

			dialer = &Websocket.Dialer{NetDial: proxy.Dial}
		}

	}

	connection, _, err := dialer.Dial(DeribitWebsocketPath, nil)
	if err != nil {
		logger.Errorf("Fail to dial:%v", err)
		// go o.triggerEvent(EventLostConnection)
		errChan <- EventLostConnection
		return err
	}

	go func() {

		cancle := make(chan struct{})
		go func() {
			for {
				select {
				case <-time.After(30 * time.Second):
					o.ping()
				case <-cancle:
					return
				}
			}
		}()

		defer close(cancle)

		for {
			_, message, err := connection.ReadMessage()
			if err != nil {
				connection.Close()
				logger.Errorf("Fail to read:%v", err)
				// go o.triggerEvent(EventLostConnection)
				errChan <- EventLostConnection
				return
			}

			// to log the trade command
			if Debug {
				filters := []string{
					"subscription",
					"ticker",
					"pong",
					"userinfo",
				}

				var filtered = false
				for _, filter := range filters {
					if strings.Contains(string(message), filter) {
						filtered = true
					}
				}

				if !filtered {
					logger.Debugf("[RECV]%s", message)
				}

			}

			if strings.Contains(string(message), "pong") {
				continue
			}

			var response map[string]interface{}
			if err = json.Unmarshal([]byte(message), &response); err != nil {
				logger.Errorf("Fail to Unmarshal:%v", err)
				continue
			}

			var method string
			if response["error"] != nil {
				logger.Infof("Error:%v", response["error"])
				// continue
			}

			if response["method"] != nil {
				method = response["method"].(string)
			}

			if method == "subscription" {

				params := response["params"].(map[string]interface{})
				datas := params["data"].(map[string]interface{})
				channel := params["channel"].(string)
				changeID := datas["change_id"].(float64)

				if prevList, ok := o.depthValues[channel].Load("data"); !ok {
					asks := datas["asks"].([]interface{})
					bids := datas["bids"].([]interface{})

					list := make([][]DepthPrice, 2)

					if asks != nil && len(asks) > 0 {
						askList := make([]DepthPrice, len(asks))
						for i, ask := range asks {
							values := ask.([]interface{})
							askList[i].Price = values[1].(float64)
							askList[i].Quantity = values[2].(float64)
						}

						// list[DepthTypeAsks] = revertDepthArray(askList)
						list[DepthTypeAsks] = askList
					}

					if bids != nil && len(bids) > 0 {
						bidList := make([]DepthPrice, len(bids))
						for i, bid := range bids {
							values := bid.([]interface{})
							bidList[i].Price = values[1].(float64)
							bidList[i].Quantity = values[2].(float64)
						}

						list[DepthTypeBids] = bidList
					}

					o.depthValues[channel].Store("data", list)
					o.depthValues[channel].Store("changeid", changeID)
					o.depthValues[channel].Store("timestamp", datas["timestamp"].(float64))
				} else {
					// errChan <- EventLostConnection
					if lastID, ok := o.depthValues[channel].Load("changeid"); ok {
						previousID := datas["prev_change_id"].(float64)
						if lastID.(float64) != previousID {
							logger.Error("changeID is lost")
							errChan <- EventLostConnection
							return
						}

					} else {
						logger.Error("ChangeID is not found, reset the connection")
						errChan <- EventLostConnection
						return

					}

					asks := datas["asks"].([]interface{})
					bids := datas["bids"].([]interface{})

					updateList := make([][]DepthPrice, 2)
					// log.Printf("Cr32:%x Result:%x", o.getCRC32Value(input), uint32(data["checksum"].(float64)))
					if asks != nil && len(asks) > 0 {
						askList := make([]DepthPrice, len(asks))
						for i, ask := range asks {
							values := ask.([]interface{})
							askList[i].Price = values[1].(float64)
							askList[i].Quantity = values[2].(float64)
						}

						updateList[DepthTypeAsks] = askList
					}

					if bids != nil && len(bids) > 0 {
						bidList := make([]DepthPrice, len(bids))
						for i, bid := range bids {
							values := bid.([]interface{})
							bidList[i].Price = values[1].(float64)
							bidList[i].Quantity = values[2].(float64)
						}
						updateList[DepthTypeBids] = bidList
					}

					newList := o.mergeDepth(prevList.([][]DepthPrice), updateList)

					o.depthValues[channel].Store("data", newList)
					o.depthValues[channel].Store("changeid", changeID)
					o.depthValues[channel].Store("timestamp", datas["timestamp"].(float64))
				}
			} else if method == "heartbeat" {
				go o.Test()
			} else {
				if response["id"] != nil {
					id := strconv.Itoa(int(response["id"].(float64)))
					if channel, ok := o.channelMap.Load(id); ok {
						go func() {
							channel.(chan interface{}) <- response
							close(channel.(chan interface{}))
							o.channelMap.Delete(id)
						}()
					}
				}
			}

		}
	}()

	o.conn = connection

	errChan <- EventConnected

	return nil
}

func (p *DeribitV2API) Authen(isRefresh bool) error {

	id := p.commandID
	p.commandID++
	method := "public/auth"
	var data map[string]interface{}
	if isRefresh && p.refreshToken != "" { // 一年有效期，可以不主动refresh
		data = map[string]interface{}{
			"json":   "2.0",
			"method": method,
			"id":     id,
			"params": map[string]interface{}{
				"grant_type":    "refresh_token",
				"refresh_token": p.refreshToken,
			},
		}
	} else {
		data = map[string]interface{}{
			"json":   "2.0",
			"method": method,
			"id":     id,
			"params": map[string]interface{}{
				"grant_type":    "client_credentials",
				"client_id":     p.ApiKey,
				"client_secret": p.SecretKey,
			},
		}
	}

	p.command(data)

	channel := strconv.Itoa(id)
	response := make(chan interface{})
	p.channelMap.Store(channel, response)

	select {
	case <-time.After(10 * time.Second):
		return errors.New("Timeout to authen")
	case result := <-response:
		rsp := result.(map[string]interface{})
		if rsp["error"] != nil {
			return errors.New(rsp["error"].(map[string]interface{})["message"].(string))
		}
		values := rsp["result"].(map[string]interface{})
		p.refreshToken = values["refresh_token"].(string)
		p.accessToken = values["access_token"].(string)
		return nil
	}
}

func (p *DeribitV2API) Test() error {

	id := p.commandID
	p.commandID++
	method := "public/test"

	data := map[string]interface{}{
		"json":   "2.0",
		"method": method,
		"id":     id,
	}

	p.command(data)

	channel := strconv.Itoa(id)
	response := make(chan interface{})
	p.channelMap.Store(channel, response)

	select {
	case <-time.After(10 * time.Second):
		return errors.New("Timeout to Test")
	case result := <-response:
		logger.Infof("the result of Test:%v", result)
		return nil
	}
}

func (p *DeribitV2API) startHeartBeat() error {

	id := p.commandID
	p.commandID++
	method := "public/set_heartbeat"

	data := map[string]interface{}{
		"json":   "2.0",
		"method": method,
		"id":     id,
		"params": map[string]interface{}{
			"interval": 10,
		},
	}

	p.command(data)

	channel := strconv.Itoa(id)
	response := make(chan interface{})
	p.channelMap.Store(channel, response)

	select {
	case <-time.After(10 * time.Second):
		return errors.New("Timeout to startHeartBeat")
	case result := <-response:
		logger.Infof("the result of authen:%v", result)
		return nil
	}
}

func (p *DeribitV2API) orderRequest(method string, path string, params map[string]string) (error, []byte) {

	logger.Infof("Path:%v", path)
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var input string
	var request *http.Request
	var err error

	if method == "GET" {
		input = timestamp + method + path
		request, err = http.NewRequest(method, OKEXV3RestAPIPath+path, nil)
	} else { // POST

		if params != nil {
			body, _ := json.Marshal(params)
			input = timestamp + method + path + string(body)
			logger.Infof("Input:%v", input)
			// reader = strings.NewReader(string(body))
			request, err = http.NewRequest(method, OKEXV3RestAPIPath+path, strings.NewReader(string(body)))
		} else {
			input = timestamp + method + path
			logger.Infof("Input:%v", input)
			// reader = strings.NewReader(string(body))
			request, err = http.NewRequest(method, OKEXV3RestAPIPath+path, nil)
		}

	}

	h := hmac.New(sha256.New, []byte(p.SecretKey))
	io.WriteString(h, input)
	// signature := fmt.Sprintf("%x", h.Sum(nil))
	// log.Printf("Sign:%v", timestamp+method+path)
	base64signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	if err != nil {
		return err, nil
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("OK-ACCESS-KEY", p.ApiKey)
	request.Header.Add("OK-ACCESS-SIGN", base64signature)
	request.Header.Add("OK-ACCESS-TIMESTAMP", timestamp)
	request.Header.Add("OK-ACCESS-PASSPHRASE", p.Passphare)

	// setup a http client
	httpTransport := &http.Transport{}
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

// Close close all handles and free the resources
func (o *DeribitV2API) Close() {
	if o.conn != nil {
		o.conn.Close()
		o.conn = nil
	}
}

func (o *DeribitV2API) StartTicker(pair string) {

}

func (o *DeribitV2API) SubKlines(pair string, period int, number int) {

}

func (o *DeribitV2API) ping() error {
	// return o.conn.WriteMessage(Websocket.TextMessage, []byte("ping"))
	return nil
}

// GetExchangeName get the name of the exchanges
func (o *DeribitV2API) GetExchangeName() string {
	return NameOKEXV3
}

func (o *DeribitV2API) StartDepth(channel string) {

	o.commandID++

	data := map[string]interface{}{
		"json":   "2.0",
		"method": "public/subscribe",
		"id":     o.commandID,
		"params": map[string]interface{}{
			"channels": []string{channel},
		},
	}
	o.command(data)

}

func (o *DeribitV2API) GetDepthValue(instrument DeribitInstrumentType) [][]DepthPrice {

	var channel string

	channel = "book." + string(instrument) + "-PERPETUAL.100ms"

	if o.depthValues[channel] == nil {
		o.depthValues[channel] = new(sync.Map)
		o.StartDepth(channel)
	}

	if o.depthValues[channel] != nil {
		now := time.Now()
		if timestamp, ok := o.depthValues[channel].Load("timestamp"); ok {
			location, _ := time.LoadLocation("Asia/Shanghai")
			updateTime := time.Unix(int64(timestamp.(float64))/1000, 0)
			logger.Infof("Now:%v Update:%v", now.String(), updateTime.In(location).String())
			if updateTime.Add(10 * time.Second).Before(now) {
				logger.Error("Invalid timestamp")
				return nil
			}
		}

		if recv, ok := o.depthValues[channel].Load("data"); ok {
			list := recv.([][]DepthPrice)
			return list
		}
	}

	return nil
}

func (o *DeribitV2API) command(data map[string]interface{}) error {
	if o.conn == nil {
		return errors.New("Connection is lost")
	}

	command := make(map[string]interface{})
	for k, v := range data {
		command[k] = v
	}

	cmd, err := json.Marshal(command)
	if err != nil {
		return errors.New("Marshal failed")
	}

	if Debug {

		filters := []string{
			"ping",
		}

		found := false

		for _, filter := range filters {
			if strings.Contains(string(cmd), filter) {
				found = true
				break
			}
		}

		if !found {
			logger.Debugf("Command[%s]", string(cmd))
		}
	}

	o.conn.WriteMessage(Websocket.TextMessage, cmd)

	return nil
}

func (p *DeribitV2API) Trade(configs TradeConfig) *TradeResult {

	var method string

	id := p.commandID
	p.commandID++

	coins := ParsePair(configs.Pair)

	if configs.Type == TradeTypeOpenLong || configs.Type == TradeTypeCloseShort {
		method = "private/buy"
	} else if configs.Type == TradeTypeOpenShort || configs.Type == TradeTypeCloseLong {
		method = "private/sell"
	} else {
		return &TradeResult{
			Error: errors.New("Invalid Trade Type"),
		}
	}

	data := map[string]interface{}{
		"json":   "2.0",
		"method": method,
		"id":     id,
		"params": map[string]interface{}{
			"instrument_name": strings.ToUpper(coins[0]) + "-PERPETUAL",
			// "price":           strconv.FormatFloat(configs.Price, 'f', 2, 64),
			"amount": strconv.FormatFloat(configs.Amount, 'f', 0, 64),
			"type":   "market",
			// "time_in_force": "fill_or_kill",
			"access_token": p.accessToken,
		},
	}

	p.command(data)

	channel := strconv.Itoa(id)
	response := make(chan interface{})
	p.channelMap.Store(channel, response)

	select {
	case <-time.After(10 * time.Second):
		return &TradeResult{
			Error: errors.New("Timeout to Trade"),
		}
	case result := <-response:
		rsp := result.(map[string]interface{})
		if rsp["error"] != nil {
			return &TradeResult{
				Error: errors.New(rsp["error"].(map[string]interface{})["message"].(string)),
			}
		}
		values := rsp["result"].(map[string]interface{})
		order := values["order"].(map[string]interface{})

		info := &OrderInfo{
			OrderID:    order["order_id"].(string),
			Pair:       configs.Pair,
			AvgPrice:   order["average_price"].(float64),
			DealAmount: order["filled_amount"].(float64),
		}

		return &TradeResult{
			Error:   nil,
			OrderID: order["order_id"].(string),
			Info:    info,
		}
	}

	return nil

}

func (p *DeribitV2API) GetPosition(pair string) (error, map[string]interface{}) {

	var method string

	id := p.commandID
	p.commandID++

	coins := ParsePair(pair)
	method = "private/get_position"

	data := map[string]interface{}{
		"json":   "2.0",
		"method": method,
		"id":     id,
		"params": map[string]interface{}{
			"instrument_name": strings.ToUpper(coins[0]) + "-PERPETUAL",
		},
	}

	p.command(data)

	channel := strconv.Itoa(id)
	response := make(chan interface{})
	p.channelMap.Store(channel, response)

	select {
	case <-time.After(10 * time.Second):
		return errors.New("Timeout to Trade"), nil
	case result := <-response:
		rsp := result.(map[string]interface{})
		if rsp["error"] != nil {
			return errors.New(rsp["error"].(map[string]interface{})["message"].(string)), nil
		}
		values := rsp["result"].(map[string]interface{})

		return nil, map[string]interface{}{
			"direction": values["direction"].(string),
			"size":      math.Abs(values["size"].(float64)),
		}
	}

}

func (p *DeribitV2API) CancelOrder(order OrderInfo) *TradeResult {
	return nil
}

func (p *DeribitV2API) GetOrderInfo(filter OrderInfo) []OrderInfo {
	return nil
}

func (o *DeribitV2API) WatchEvent() chan EventType {
	return nil
}

func (o *DeribitV2API) SetConfigure(config Config) {

}

func (o *DeribitV2API) GetTicker(pair string) *TickerValue {
	return nil
}

func (p *DeribitV2API) GetBalance(currency string) (error, float64) {

	id := p.commandID
	p.commandID++
	method := "private/get_account_summary"

	data := map[string]interface{}{
		"json":   "2.0",
		"method": method,
		"id":     id,
		"params": map[string]interface{}{
			"access_token": p.accessToken,
			"currency":     currency,
		},
	}

	p.command(data)

	channel := strconv.Itoa(id)
	response := make(chan interface{})
	p.channelMap.Store(channel, response)

	select {
	case <-time.After(10 * time.Second):
		return errors.New("Timeout to GetBalance"), 0
	case result := <-response:
		rsp := result.(map[string]interface{})
		if rsp["error"] != nil {
			return errors.New(rsp["error"].(map[string]interface{})["message"].(string)), 0
		}
		values := rsp["result"].(map[string]interface{})
		balance := values["balance"].(float64)
		return nil, balance
	}
}

func (o *DeribitV2API) getTradeTypeByString(orderType string) TradeType {
	switch orderType {
	case "1":
		return TradeTypeOpenLong
	case "2":
		return TradeTypeOpenShort
	case "3":
		return TradeTypeCloseLong
	case "4":
		return TradeTypeCloseShort
	case "buy":
		return TradeTypeBuy
	case "sell":
		return TradeTypeSell
	}

	return TradeTypeUnknown
}

func (o *DeribitV2API) GetKline(instrument string, period int, limit int) []KlineValue {
	return nil
}

// var OkexV3OrderStatusString = map[OrderStatusType]string{
// 	OrderStatusOpen:      "open",
// 	OrderStatusPartDone:  "part_filled",
// 	OrderStatusDone:      "filled",
// 	OrderStatusCanceling: "canceling",
// 	OrderStatusCanceled:  "cancelled",
// 	OrderStatusRejected:  "failure",
// 	OrderStatusOrdering:  "ordering",
// }

func (o *DeribitV2API) GetOrderStatus(status string) OrderStatusType {
	for k, v := range OkexV3OrderStatusString {
		if v == status {
			return k
		}
	}

	return OrderStatusUnknown
}
