package exchange

import (
	"bytes"
	"compress/flate"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	Websocket "github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

const NameOKEXV3 = "OkexV3"

const OKEXV3RestAPIPath = "https://www.okex.com"
const OKEXV3WebsocketPath = "wss://real.okex.com:10442/ws/v3"

// event
const OKEXV3OpSubscribe = "subscribe"

const OKEXV3TableSpotDepthPrefix = "spot/depth"
const OKEXV3TableSwapDepthPrefix = "swap/depth"
const OKEXV3TableFutureDepthPrefix = "futures/depth"
const OKEXV3KeyTimestamp = "timestamp"

type InstrumentType int8

const (
	InstrumentTypeSpot InstrumentType = iota
	InstrumentTypeSwap
	InstrumentTypeFuture
)

type OKEXV3API struct {
	InstrumentType InstrumentType
	Proxy          string
	ApiKey         string
	SecretKey      string
	Passphare      string

	conn        *Websocket.Conn
	depthValues map[string]*sync.Map

	FutureIndex string // 季度合约的日期
}

func (o *OKEXV3API) mergeDepth(oldList [][]DepthPrice, updateList [][]DepthPrice) [][]DepthPrice {

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

func (o *OKEXV3API) Start() error {
	return nil
}

func (o *OKEXV3API) Start2(errChan chan EventType) error {

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

	connection, _, err := dialer.Dial(OKEXV3WebsocketPath, nil)
	if err != nil {
		logger.Errorf("Fail to dial:%v", err)
		// go o.triggerEvent(EventLostConnection)
		errChan <- EventLostConnection
		return err
	}

	go func() {
		counter := 0

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

			r := flate.NewReader(bytes.NewReader(message))
			out, err := ioutil.ReadAll(r)
			if err != nil {
				r.Close()
				logger.Errorf("Fail to decompress:%s\n", err)
				return
			}

			r.Close()

			message = out

			// to log the trade command
			if Debug {
				filters := []string{
					"depth",
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

			if response["event"] != nil && response["event"].(string) == "error" {
				if response["message"] != nil {
					logger.Errorf("Response Error:%v", response["message"].(string))
				}
				continue
			}

			var table string
			if response["event"] != nil {
				logger.Infof("Event:%v", response["event"].(string))
				continue
			}

			if response["table"] != nil {
				table = response["table"].(string)
			}

			if table == OKEXV3TableSpotDepthPrefix || table == OKEXV3TableSwapDepthPrefix || table == OKEXV3TableFutureDepthPrefix {
				action := response["action"].(string)
				datas := response["data"].([]interface{})
				for _, tmp := range datas {
					data := tmp.(map[string]interface{})
					channel := data["instrument_id"].(string)
					if action == "partial" {

						if o.depthValues[channel] != nil {
							asks := data["asks"].([]interface{})
							bids := data["bids"].([]interface{})

							compare := o.checkCRC32(asks, bids, uint32(data["checksum"].(float64)))
							if !compare {

								logger.Error("1. The crc32 is NOT the same")
								// o.triggerEvent(EventLostConnection)

								connection.Close()
								errChan <- EventLostConnection

								return
							}

							list := make([][]DepthPrice, 2)
							// log.Printf("Cr32:%x Result:%x", o.getCRC32Value(input), uint32(data["checksum"].(float64)))
							if asks != nil && len(asks) > 0 {
								askList := make([]DepthPrice, len(asks))
								for i, ask := range asks {
									values := ask.([]interface{})
									askList[i].Price, _ = strconv.ParseFloat(values[0].(string), 64)
									askList[i].Quantity, _ = strconv.ParseFloat(values[1].(string), 64)
								}

								// list[DepthTypeAsks] = revertDepthArray(askList)
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

							o.depthValues[channel].Store("data", list)
							o.depthValues[channel].Store(OKEXV3KeyTimestamp, data["timestamp"].(string))
						}
					} else if action == "update" {
						if o.depthValues[channel] != nil {

							if oldList, ok := o.depthValues[channel].Load("data"); ok {

								// log.Printf("oldList:%v", oldList)

								asks := data["asks"].([]interface{})
								bids := data["bids"].([]interface{})

								updateList := make([][]DepthPrice, 2)
								// log.Printf("Cr32:%x Result:%x", o.getCRC32Value(input), uint32(data["checksum"].(float64)))
								if asks != nil && len(asks) > 0 {
									askList := make([]DepthPrice, len(asks))
									for i, ask := range asks {
										values := ask.([]interface{})
										askList[i].Price, _ = strconv.ParseFloat(values[0].(string), 64)
										askList[i].Quantity, _ = strconv.ParseFloat(values[1].(string), 64)
									}

									// updateList[DepthTypeAsks] = revertDepthArray(askList)
									updateList[DepthTypeAsks] = askList
								}

								if bids != nil && len(bids) > 0 {
									bidList := make([]DepthPrice, len(bids))
									for i, bid := range bids {
										values := bid.([]interface{})
										bidList[i].Price, _ = strconv.ParseFloat(values[0].(string), 64)
										bidList[i].Quantity, _ = strconv.ParseFloat(values[1].(string), 64)
									}

									updateList[DepthTypeBids] = bidList
								}

								newList := o.mergeDepth(oldList.([][]DepthPrice), updateList)

								// log.Printf("NewList:%v", newList)

								var input string
								askList := newList[DepthTypeAsks]
								bidList := newList[DepthTypeBids]

								if len(bidList) < 25 || len(askList) < 25 {
									log.Printf("Invalid length of asklist/bidlist")
									continue
								}

								for i := 0; i < 25; i++ {
									if i != 24 {
										input += fmt.Sprintf("%v:%v:%v:%v:", bidList[i].Price, bidList[i].Quantity, askList[i].Price, askList[i].Quantity)
									} else {
										input += fmt.Sprintf("%v:%v:%v:%v", bidList[i].Price, bidList[i].Quantity, askList[i].Price, askList[i].Quantity)
									}
								}

								crc32Value := o.getCRC32Value(input)
								valueCalc := fmt.Sprintf("%x", crc32Value)
								valueOriginal := fmt.Sprintf("%x", uint32(data["checksum"].(float64)))
								if valueCalc != valueOriginal {

									logger.Error("2.The crc32 is NOT the same")
									// o.triggerEvent(EventLostConnection)
									if counter > 5 {
										connection.Close()
										errChan <- EventLostConnection
										return
									} else {
										counter++
									}

								} else {
									counter = 0
									// logger.Infof("Update the depths, CRC32 is ok")
								}

								o.depthValues[channel].Store("data", newList)
								o.depthValues[channel].Store(OKEXV3KeyTimestamp, data["timestamp"].(string))
							}
						}
					}
				}
			}
		}
	}()

	o.conn = connection

	errChan <- EventConnected

	return nil
}

func (p *OKEXV3API) orderRequest(method string, path string, params map[string]string) (error, []byte) {

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
func (o *OKEXV3API) Close() {
	if o.conn != nil {
		o.conn.Close()
		o.conn = nil
	}
}

func (o *OKEXV3API) StartTicker(pair string) {

}

func (o *OKEXV3API) SubKlines(pair string, period int, number int) {

}

func (o *OKEXV3API) ping() error {
	return o.conn.WriteMessage(Websocket.TextMessage, []byte("ping"))
}

// GetExchangeName get the name of the exchanges
func (o *OKEXV3API) GetExchangeName() string {
	return NameOKEXV3
}

func (o *OKEXV3API) StartDepth(channel string) {

	data := map[string]interface{}{
		"op": OKEXV3OpSubscribe,
		"args": []string{
			channel,
		},
	}
	o.command(data)

}

func (o *OKEXV3API) checkCRC32(asks []interface{}, bids []interface{}, crc32Original uint32) bool {

	var input string
	for i := 0; i < 25; i++ {
		value1 := bids[i].([]interface{})
		value2 := asks[i].([]interface{})
		input += value1[0].(string) + ":" + value1[1].(string) + ":"
		if i != 24 {
			input += value2[0].(string) + ":" + value2[1].(string) + ":"
		} else {
			input += value2[0].(string) + ":" + value2[1].(string)
		}

	}

	ieee := crc32.NewIEEE()
	io.WriteString(ieee, input)
	valueCalc := fmt.Sprintf("%x", ieee.Sum32())
	valueOriginal := fmt.Sprintf("%x", crc32Original)
	// log.Printf("Values:%v value2:%v", valueCalc, valueOriginal)
	return (valueCalc == valueOriginal)
}

func (o *OKEXV3API) getCRC32Value(input string) uint32 {

	ieee := crc32.NewIEEE()
	io.WriteString(ieee, input)
	return ieee.Sum32()
}

func (o *OKEXV3API) GetDepthValue(coin string) [][]DepthPrice {

	var channel string
	coins := ParsePair(coin)

	if o.InstrumentType == InstrumentTypeSwap {
		// channel = o.StartDepth()
		channel = strings.ToUpper(coins[0]) + "-USD-SWAP"
		if o.depthValues[channel] == nil {
			o.depthValues[channel] = new(sync.Map)
			o.StartDepth(OKEXV3TableSwapDepthPrefix + ":" + channel)
		}
	} else if o.InstrumentType == InstrumentTypeSpot {
		channel = strings.ToUpper(coins[0]) + "-" + strings.ToUpper(coins[1])
		if o.depthValues[channel] == nil {
			o.depthValues[channel] = new(sync.Map)
			o.StartDepth(OKEXV3TableSpotDepthPrefix + ":" + channel)
		}
	} else if o.InstrumentType == InstrumentTypeFuture {
		channel = strings.ToUpper(coins[0]) + "-USD-" + o.FutureIndex
		if o.depthValues[channel] == nil {
			o.depthValues[channel] = new(sync.Map)
			o.StartDepth(OKEXV3TableFutureDepthPrefix + ":" + channel)
		}
	}

	if o.depthValues[channel] != nil {
		now := time.Now()
		if timestamp, ok := o.depthValues[channel].Load(OKEXV3KeyTimestamp); ok {
			// location, _ := time.LoadLocation("Asia/Shanghai")
			updateTime, _ := time.Parse(time.RFC3339Nano, timestamp.(string))
			// logger.Infof("Now:%v Update:%v", now.String(), updateTime.In(location).String())
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

func (o *OKEXV3API) command(data map[string]interface{}) error {
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

func (o *OKEXV3API) Trade(configs TradeConfig) *TradeResult {

	var path string
	var parameters map[string]string

	coins := ParsePair(configs.Pair)

	if o.InstrumentType == InstrumentTypeSwap {

		path = "/api/swap/v3/order"

		parameters = map[string]string{
			"instrument_id": strings.ToUpper(coins[0]) + "-USD-SWAP",
			"price":         strconv.FormatFloat(configs.Price, 'f', 2, 64),
			// the exact amount orders is amount/level_rate
			"size": strconv.FormatFloat(configs.Amount, 'f', 0, 64),
			"type": OkexGetTradeTypeString(configs.Type),
			// "match_price": "0",
		}

	} else if o.InstrumentType == InstrumentTypeSpot {

		path = "/api/spot/v3/orders"

		if configs.Type == TradeTypeBuy {
			parameters = map[string]string{
				"instrument_id": strings.ToUpper(coins[0]) + "-" + strings.ToUpper(coins[1]),
				// "size":           strconv.FormatFloat(configs.Amount, 'f', 4, 64),
				"side":           OkexGetTradeTypeString(configs.Type),
				"margin_trading": "1",

				"type":     "market",
				"notional": strconv.FormatFloat(configs.Amount, 'f', 4, 64),
			}
		} else { // TradeTypeSell
			parameters = map[string]string{
				"instrument_id":  strings.ToUpper(coins[0]) + "-" + strings.ToUpper(coins[1]),
				"size":           strconv.FormatFloat(configs.Amount, 'f', 4, 64),
				"side":           OkexGetTradeTypeString(configs.Type),
				"margin_trading": "1",
				"type":           "market",
				// "notional": strconv.FormatFloat(configs.Price, 'f', 4, 64),
			}
		}

	}

	if err, response := o.orderRequest("POST", path, parameters); err != nil {
		logger.Errorf("无法获取余额:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		if o.InstrumentType == InstrumentTypeSwap {
			if values["error_code"].(string) != "0" {

				errorCode, _ := strconv.ParseInt(values["error_code"].(string), 10, 64)
				return &TradeResult{
					Error:     errors.New(values["error_message"].(string)),
					ErrorCode: int(errorCode),
				}

			} else {
				return &TradeResult{
					Error:   nil,
					OrderID: values["order_id"].(string),
				}
			}

			return nil
		} else {

			if values["result"] == nil || !values["result"].(bool) {

				errorCode := values["code"].(float64)
				return &TradeResult{
					Error:     errors.New(values["message"].(string)),
					ErrorCode: int(errorCode),
				}

			} else {
				return &TradeResult{
					Error:   nil,
					OrderID: values["order_id"].(string),
				}
			}
		}

	}

	return nil

}

func (o *OKEXV3API) CancelOrder(order OrderInfo) *TradeResult {

	var path string
	// var parameters map[string]string

	pair := ParsePair(order.Pair)

	if o.InstrumentType == InstrumentTypeSwap {

		path = "/api/swap/v3/cancel_order/" + strings.ToUpper(pair[0]) + "-USD-SWAP/" + order.OrderID

		// parameters = map[string]string{
		// 	"api_key":  o.ApiKey,
		// 	"order_id": order.OrderID,
		// 	"symbol":   coins[0] + "_" + coins[1],
		// }

	} else if o.InstrumentType == InstrumentTypeSpot {

		// parameters = map[string]string{
		// 	"api_key":  o.ApiKey,
		// 	"order_id": order.OrderID,
		// 	"symbol":   coins[0] + "_" + coins[1],
		// }

	}

	if err, response := o.orderRequest("POST", path, nil); err != nil {
		logger.Errorf("无法获取余额:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		if values["result"] != "true" {
			logger.Errorf("Fail to cancle the order:%v", values)
			return nil
		} else {
			return &TradeResult{
				Error:   nil,
				OrderID: values["order_id"].(string),
			}
		}
	}

	return nil

}

func (o *OKEXV3API) GetOrderInfo(filter OrderInfo) []OrderInfo {

	var path string

	pair := ParsePair(filter.Pair)

	if o.InstrumentType == InstrumentTypeSwap {

		path = "/api/swap/v3/orders/" + strings.ToUpper(pair[0]) + "-USD-SWAP/" + filter.OrderID

	} else if o.InstrumentType == InstrumentTypeSpot {

		// parameters = map[string]string{
		// 	"api_key": o.ApiKey,
		// 	// "secret_key": constSecretKey,
		// 	"order_id": filter.OrderID,
		// 	"symbol":   pair[0] + "_" + pair[1],
		// }
		path = "/api/spot/v3/orders/" + filter.OrderID + "?" + "instrument_id=" + strings.ToUpper(pair[0]) + "-" + strings.ToUpper(pair[1])

	}

	if err, response := o.orderRequest("GET", path, nil); err != nil {
		logger.Errorf("无法获取余额:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		if o.InstrumentType == InstrumentTypeSwap {

			if values["code"] != nil {
				logger.Errorf("Fail to get the order info:%v", values["message"].(string))
				return nil
			}

			result := make([]OrderInfo, 1)

			orderType, _ := strconv.ParseFloat(values["type"].(string), 64)
			placePrice, _ := strconv.ParseFloat(values["price"].(string), 64)
			avgPrice, _ := strconv.ParseFloat(values["price_avg"].(string), 64)
			amount, _ := strconv.ParseFloat(values["size"].(string), 64)
			dealAmount, _ := strconv.ParseFloat(values["filled_qty"].(string), 64)
			status, _ := strconv.ParseFloat(values["status"].(string), 64)

			item := OrderInfo{
				Pair:    values["instrument_id"].(string),
				OrderID: values["order_id"].(string),
				// OrderID: strconv.FormatInt(order["order_id"].(int64), 64),
				Price:      placePrice,
				Amount:     amount,
				Type:       OkexGetTradeTypeByFloat(orderType),
				Status:     OkexGetTradeStatus(status),
				DealAmount: dealAmount,
				AvgPrice:   avgPrice,
			}

			result[0] = item

			return result
		} else {

			result := make([]OrderInfo, 1)

			orderType := OkexGetTradeTypeByString(values["side"].(string))
			placePrice, _ := strconv.ParseFloat(values["price"].(string), 64)
			amount, _ := strconv.ParseFloat(values["size"].(string), 64)
			// all units are the base currency
			var dealAmount float64
			if orderType == TradeTypeBuy {
				// when buy, we need to know the size of the target
				dealAmount, _ = strconv.ParseFloat(values["filled_size"].(string), 64)
			} else {
				// when sell, we need to know the size of the base
				dealAmount, _ = strconv.ParseFloat(values["filled_notional"].(string), 64)
			}

			filledNotional, _ := strconv.ParseFloat(values["filled_notional"].(string), 64)
			avgPrice := filledNotional / dealAmount
			status := values["status"].(string)

			item := OrderInfo{
				Pair:    values["instrument_id"].(string),
				OrderID: values["order_id"].(string),
				// OrderID: strconv.FormatInt(order["order_id"].(int64), 64),
				Price:      placePrice,
				Amount:     amount,
				Type:       orderType,
				Status:     o.GetOrderStatus(status),
				DealAmount: dealAmount,
				AvgPrice:   avgPrice,
			}

			result[0] = item

			return result
		}

	}

	return nil
}

func (o *OKEXV3API) WatchEvent() chan EventType {
	return nil
}

func (o *OKEXV3API) SetConfigure(config Config) {

}

func (o *OKEXV3API) GetTicker(pair string) *TickerValue {
	return nil
}

func (o *OKEXV3API) GetBalance() map[string]interface{} {

	var path string

	if o.InstrumentType == InstrumentTypeSwap {
		path = "/api/swap/v3/accounts"

		if err, response := o.orderRequest("GET", path, map[string]string{}); err != nil {
			logger.Errorf("无法获取余额:%v", err)
			return nil
		} else {
			var values map[string]interface{}
			if err = json.Unmarshal(response, &values); err != nil {
				logger.Errorf("解析错误:%v", err)
				return nil
			}

			if values["info"] != nil {
				balance := values["info"].([]interface{})

				result := map[string]interface{}{}

				for _, temp := range balance {
					instrument := temp.(map[string]interface{})["instrument_id"].(string)
					key := strings.Split(instrument, "-")[0]
					value := temp.(map[string]interface{})["equity"].(string)
					if value == "" {
						result[key] = 0.0
					} else {
						result[key], _ = strconv.ParseFloat(value, 64)
					}
				}

				return result
			}
		}

	} else if o.InstrumentType == InstrumentTypeSpot {
		// channel = ChannelSpotUserInfo
		path = "/api/spot/v3/accounts"

		if err, response := o.orderRequest("GET", path, map[string]string{}); err != nil {
			logger.Errorf("无法获取余额:%v", err)
			return nil
		} else {
			var values []interface{}
			if err = json.Unmarshal(response, &values); err != nil {
				logger.Errorf("解析错误:%v", err)
				return nil
			}

			if len(values) == 0 {
				return nil
			}

			result := make(map[string]interface{})
			for _, temp := range values {
				instrument := temp.(map[string]interface{})
				key := instrument["currency"].(string)
				value := temp.(map[string]interface{})["available"].(string)
				if value == "" {
					result[key] = 0.0
				} else {
					result[key], _ = strconv.ParseFloat(value, 4)
				}
			}

			return result

		}
	}

	return nil
}
func (o *OKEXV3API) GetBalance2(instrument string) map[string]interface{} {

	var path string
	pair := ParsePair(instrument)
	instrument = strings.ToUpper(pair[0]) + "-" + strings.ToUpper(pair[1]) + "-SWAP"

	if o.InstrumentType == InstrumentTypeSwap {
		path = "/api/swap/v3/" + instrument + "/accounts"

	} else if o.InstrumentType == InstrumentTypeSpot {
		// channel = ChannelSpotUserInfo
	}

	if err, response := o.orderRequest("GET", path, map[string]string{}); err != nil {
		logger.Errorf("无法获取余额:%v", err)
		return nil
	} else {
		var values map[string]interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		return nil
	}

}

func (o *OKEXV3API) getTradeTypeByString(orderType string) TradeType {
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

func (o *OKEXV3API) GetKline(instrument string, period int, limit int) []KlineValue {
	var path string
	pair := ParsePair(instrument)

	granularity := period * 60

	if o.InstrumentType == InstrumentTypeSwap {
		instrument = strings.ToUpper(pair[0]) + "-USD-SWAP"
		path = "/api/swap/v3/instruments/" + instrument + "/candles?granularity=" + strconv.Itoa(granularity)

	} else if o.InstrumentType == InstrumentTypeSpot {
		// channel = ChannelSpotUserInfo
		instrument = strings.ToUpper(pair[0]) + "-" + strings.ToUpper(pair[1])
		path = "/api/spot/v3/instruments/" + instrument + "/candles?granularity=" + strconv.Itoa(granularity)
	}

	if err, response := o.orderRequest("GET", path, map[string]string{}); err != nil {
		logger.Errorf("无法获取余额:%v", err)
		return nil
	} else {
		var values []interface{}
		if err = json.Unmarshal(response, &values); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		// if o.InstrumentType == InstrumentTypeSwap {
		if values != nil && len(values) > 0 {
			klines := make([]KlineValue, len(values))
			for i, temp := range values {
				value := temp.([]interface{})
				updateTime, _ := time.Parse(time.RFC3339Nano, value[0].(string))
				klines[i].OpenTime = float64(updateTime.Unix())
				klines[i].Open, _ = strconv.ParseFloat(value[1].(string), 64)
				klines[i].High, _ = strconv.ParseFloat(value[2].(string), 64)
				klines[i].Low, _ = strconv.ParseFloat(value[3].(string), 64)
				klines[i].Close, _ = strconv.ParseFloat(value[4].(string), 64)
				klines[i].Volumn, _ = strconv.ParseFloat(value[5].(string), 64)
			}

			klines = RevertArray(klines)
			return klines
		}
		// } else {
		// 	if values != nil && len(values) > 0 {
		// 		klines := make([]KlineValue, len(values))
		// 		for i, temp := range values {
		// 			value := temp.(map[string]interface{})
		// 			updateTime, _ := time.Parse(time.RFC3339Nano, value["time"].(string))
		// 			klines[i].OpenTime = float64(updateTime.Unix())
		// 			klines[i].Open, _ = strconv.ParseFloat(value["open"].(string), 64)
		// 			klines[i].High, _ = strconv.ParseFloat(value["high"].(string), 64)
		// 			klines[i].Low, _ = strconv.ParseFloat(value["low"].(string), 64)
		// 			klines[i].Close, _ = strconv.ParseFloat(value["close"].(string), 64)
		// 			klines[i].Volumn, _ = strconv.ParseFloat(value["volume"].(string), 64)
		// 		}

		// 		klines = RevertArray(klines)
		// 		return klines
		// 	}
		// }

	}

	return nil
}

var OkexV3OrderStatusString = map[OrderStatusType]string{
	OrderStatusOpen:      "open",
	OrderStatusPartDone:  "part_filled",
	OrderStatusDone:      "filled",
	OrderStatusCanceling: "canceling",
	OrderStatusCanceled:  "cancelled",
	OrderStatusRejected:  "failure",
	OrderStatusOrdering:  "ordering",
}

func (o *OKEXV3API) GetOrderStatus(status string) OrderStatusType {
	for k, v := range OkexV3OrderStatusString {
		if v == status {
			return k
		}
	}

	return OrderStatusUnknown
}
