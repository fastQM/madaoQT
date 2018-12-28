package exchange

import (
	"bytes"
	"compress/flate"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	Websocket "github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

const NameOKEXV3 = "OkexV3"

const OKEXV3WebsocketPath = "wss://real.okex.com:10442/ws/v3"

// event
const OKEXV3OpSubscribe = "subscribe"

const OKEXV3TableSpotDepthPrefix = "spot/depth"
const OKEXV3KeyTimestamp = "timestamp"

type InstrumentType int8

const (
	InstrumentTypeSpot InstrumentType = iota
	InstrumentTypeSwap
)

type OKEXV3API struct {
	instrumentType InstrumentType

	conn      *Websocket.Conn
	apiKey    string
	secretKey string
	proxy     string

	klines       map[string][]KlineValue
	ticker       int64
	lastTicker   int64
	errorCounter int

	event chan EventType

	/* Each channel has a depth */
	messageChannels sync.Map

	depthValues map[string]*sync.Map
}

func (o *OKEXV3API) WatchEvent() chan EventType {
	return o.event
}

func (o *OKEXV3API) triggerEvent(event EventType) {
	o.event <- event
}

func (o *OKEXV3API) SetConfigure(config Config) {

	o.event = make(chan EventType)
	o.apiKey = config.API
	o.secretKey = config.Secret
	o.proxy = config.Proxy

	if o.apiKey == "" || o.secretKey == "" {
		logger.Debug("The current connection doesn`t support trading without API")
	}

	go func() {
		for {
			select {
			case <-time.After(30 * time.Second):
				o.ping()
			}
		}
	}()

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

	newList[DepthTypeAsks] = newList[DepthTypeAsks][:200]

	lastPosition = 0

	quit = false
	for current, updateBid := range updateBids {
		for i := lastPosition; i < len(oldBids); i++ {
			// log.Printf("BID:Update:%v OldIndex[%v]%v", updateBid, i, oldBids[i])
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

	newList[DepthTypeBids] = newList[DepthTypeBids][:200]

	return newList
}

func (o *OKEXV3API) Start() error {

	o.klines = make(map[string][]KlineValue)
	// force to restart the command
	o.depthValues = make(map[string]*sync.Map)

	dialer := Websocket.DefaultDialer

	if o.proxy != "" {
		logger.Infof("Proxy:%s", o.proxy)
		values := strings.Split(o.proxy, ":")
		if values[0] == "SOCKS5" {
			proxy, err := proxy.SOCKS5("tcp", values[1]+":"+values[2], nil, proxy.Direct)
			if err != nil {
				return err
			}

			dialer = &Websocket.Dialer{NetDial: proxy.Dial}
		}

	}

	c, _, err := dialer.Dial(OKEXV3WebsocketPath, nil)
	if err != nil {
		logger.Errorf("Fail to dial:%v", err)
		go o.triggerEvent(EventLostConnection)
		return err
	}

	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				c.Close()
				logger.Errorf("Fail to read:%v", err)
				go o.triggerEvent(EventLostConnection)
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

			if table == OKEXV3TableSpotDepthPrefix {
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
								logger.Error("The crc32 is NOT the same")
								o.triggerEvent(EventLostConnection)
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
									logger.Error("The crc32 is NOT the same")
									o.triggerEvent(EventLostConnection)
									return
								} else {
									// logger.Infof("Update the depths")
								}

								o.depthValues[channel].Store("data", newList)
								o.depthValues[channel].Store(OKEXV3KeyTimestamp, data["timestamp"].(string))
							}

							// os.Exit(1)
						}
					}
				}
			}
		}
	}()

	o.conn = c

	go o.triggerEvent(EventConnected)

	return nil

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
	data := map[string]interface{}{
		"event": "ping",
	}
	return o.command(data, nil)
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
	o.command(data, nil)

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
	log.Printf("Values:%v value2:%v", valueCalc, valueOriginal)
	return (valueCalc == valueOriginal)
}

func (o *OKEXV3API) getCRC32Value(input string) uint32 {

	ieee := crc32.NewIEEE()
	io.WriteString(ieee, input)
	return ieee.Sum32()
}

func (o *OKEXV3API) GetDepthValue(coin string) [][]DepthPrice {

	coins := ParsePair(coin)
	channel := strings.ToUpper(coins[0]) + "-" + strings.ToUpper(coins[1])

	if o.instrumentType == InstrumentTypeSwap {
		// channel = o.StartDepth()
	} else if o.instrumentType == InstrumentTypeSpot {
		if o.depthValues[channel] == nil {
			o.depthValues[channel] = new(sync.Map)
			o.StartDepth(OKEXV3TableSpotDepthPrefix + ":" + channel)
		}
	}

	if o.depthValues[channel] != nil {
		now := time.Now()
		if timestamp, ok := o.depthValues[channel].Load(OKEXV3KeyTimestamp); ok {
			location, _ := time.LoadLocation("Asia/Shanghai")
			updateTime, _ := time.Parse(time.RFC3339Nano, timestamp.(string))
			logger.Infof("Now:%v Update:%v", now.String(), updateTime.In(location).String())
			if updateTime.Add(10 * time.Second).Before(now) {
				logger.Error("Invalid timestamp")
				return nil
			}
		}

		if recv, ok := o.depthValues[channel].Load("data"); ok {
			list := recv.([][]DepthPrice)
			if o.instrumentType == InstrumentTypeSwap {

				// if asks != nil && len(asks) > 0 {
				// 	askList := make([]DepthPrice, len(asks))
				// 	for i, ask := range asks {
				// 		values := ask.([]interface{})
				// 		askList[i].Price = values[UsdPriceIndex].(float64)
				// 		askList[i].Quantity = values[CoinQuantity].(float64)
				// 	}

				// 	list[DepthTypeAsks] = revertDepthArray(askList)
				// }

				// if bids != nil && len(bids) > 0 {
				// 	bidList := make([]DepthPrice, len(bids))
				// 	for i, bid := range bids {
				// 		values := bid.([]interface{})
				// 		bidList[i].Price = values[UsdPriceIndex].(float64)
				// 		bidList[i].Quantity = values[CoinQuantity].(float64)
				// 	}

				// 	list[DepthTypeBids] = bidList
				// }

			} else if o.instrumentType == InstrumentTypeSpot {
				// list[DepthTypeAsks] = revertDepthArray(list[DepthTypeAsks])
			}

			return list
		}
	}

	return nil
}

func (o *OKEXV3API) command(data map[string]interface{}, parameters map[string]string) error {
	if o.conn == nil {
		return errors.New("Connection is lost")
	}

	command := make(map[string]interface{})
	for k, v := range data {
		command[k] = v
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

		signPlain += ("secret_key=" + o.secretKey)

		// log.Printf("Plain:%v", signPlain)
		md5Value := fmt.Sprintf("%x", md5.Sum([]byte(signPlain)))
		// log.Printf("MD5:%v", md5Value)
		parameters["sign"] = strings.ToUpper(md5Value)
		command["parameters"] = parameters
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

/*

1. 【合约参数】
api_key: 用户申请的apiKey
sign: 请求参数的签名
symbol:btc_usd   ltc_usd
contract_type: 合约类型: this_week:当周 next_week:下周 quarter:季度
price: 价格
amount: 委托数量
type 1:开多 2:开空 3:平多 4:平空
match_price 是否为对手价： 0:不是 1:是 当取值为1时,price无效
lever_rate 杠杆倍数 value:10\20 默认10

【现货参数】

2. 返回：

错误或者order ID

*/
func (o *OKEXV3API) Trade(configs TradeConfig) *TradeResult {

	var channel string
	var parameters map[string]string

	coins := ParsePair(configs.Pair)

	if o.instrumentType == InstrumentTypeSwap {

		channel = ChannelFutureTrade

		parameters = map[string]string{
			"api_key": o.apiKey,
			"symbol":  coins[0] + "_usd",
			"price":   strconv.FormatFloat(configs.Price, 'f', 2, 64),
			// the exact amount orders is amount/level_rate
			"amount":      strconv.FormatFloat(configs.Amount, 'f', 2, 64),
			"type":        OkexGetTradeTypeString(configs.Type),
			"match_price": "0",
			"lever_rate":  "10",
		}

	} else if o.instrumentType == InstrumentTypeSpot {

		channel = ChannelSpotOrder

		parameters = map[string]string{
			"api_key": o.apiKey,
			"symbol":  coins[0] + "_" + coins[1],
			"type":    OkexGetTradeTypeString(configs.Type),
			"price":   strconv.FormatFloat(configs.Price, 'f', 4, 64),
			"amount":  strconv.FormatFloat(configs.Amount, 'f', 4, 64),
		}
	}

	data := map[string]interface{}{
		"event":   EventAddChannel,
		"channel": channel,
	}

	recvChan := make(chan interface{})
	o.messageChannels.Store(channel, recvChan)

	if err := o.command(data, parameters); err != nil {
		return &TradeResult{
			Error: err,
		}
	}

	select {
	case <-time.After(DefaultTimeoutSec * time.Second):
		go o.triggerEvent(EventLostConnection)
		return &TradeResult{
			Error: errors.New("Timeout to trade"),
		}
	case recv := <-recvChan:
		// log.Printf("message:%v", message)
		if recv != nil {
			result := recv.(map[string]interface{})["result"]
			if result != nil && result.(bool) {
				orderId := strconv.FormatFloat(recv.(map[string]interface{})["order_id"].(float64), 'f', 0, 64)
				return &TradeResult{
					Error:   nil,
					OrderID: orderId,
				}
			} else {
				errorCode := strconv.FormatFloat(recv.(map[string]interface{})["error_code"].(float64), 'f', 0, 64)
				return &TradeResult{
					Error: errors.New("errorCode:" + errorCode),
				}
			}
		}

		return &TradeResult{
			Error: errors.New("Invalid response"),
		}
	}

}

func (o *OKEXV3API) CancelOrder(order OrderInfo) *TradeResult {

	var channel string
	var parameters map[string]string

	coins := ParsePair(order.Pair)

	if o.instrumentType == InstrumentTypeSwap {

		channel = ChannelFutureCancelOrder

		parameters = map[string]string{
			"api_key":  o.apiKey,
			"order_id": order.OrderID,
			"symbol":   coins[0] + "_" + coins[1],
		}

	} else if o.instrumentType == InstrumentTypeSpot {
		channel = ChannelSpotCancelOrder

		parameters = map[string]string{
			"api_key":  o.apiKey,
			"order_id": order.OrderID,
			"symbol":   coins[0] + "_" + coins[1],
		}

	}

	data := map[string]interface{}{
		"event":   EventAddChannel,
		"channel": channel,
	}

	recvChan := make(chan interface{})
	o.messageChannels.Store(channel, recvChan)

	o.command(data, parameters)

	select {
	case <-time.After(DefaultTimeoutSec * time.Second):
		go o.triggerEvent(EventLostConnection)
		return nil
	case recv := <-recvChan:
		if recv != nil {
			result := recv.(map[string]interface{})["result"]
			if result != nil && result.(bool) {
				orderId := recv.(map[string]interface{})["order_id"].(string)
				return &TradeResult{
					Error:   nil,
					OrderID: orderId,
				}
			} else {
				errorCode := strconv.FormatFloat(recv.(map[string]interface{})["error_code"].(float64), 'f', 0, 64)
				return &TradeResult{
					Error: errors.New("errorCode:" + errorCode),
				}
			}
		}

		return &TradeResult{
			Error: errors.New("Invalid response"),
		}
	}

}

func (o *OKEXV3API) GetOrderInfo(filter OrderInfo) []OrderInfo {

	var channel string
	var parameters map[string]string

	pair := ParsePair(filter.Pair)

	if o.instrumentType == InstrumentTypeSwap {

		channel = ChannelFutureOrderInfo

		parameters = map[string]string{
			"api_key": o.apiKey,
			// "secret_key": constSecretKey,
			// "status":        "1",
			"current_page": "1",
			"page_length":  "1",
			"order_id":     filter.OrderID,
			"symbol":       pair[0] + "_" + pair[1],
		}

	} else if o.instrumentType == InstrumentTypeSpot {

		channel = ChannelSpotOrderInfo

		parameters = map[string]string{
			"api_key": o.apiKey,
			// "secret_key": constSecretKey,
			"order_id": filter.OrderID,
			"symbol":   pair[0] + "_" + pair[1],
		}

	}

	data := map[string]interface{}{
		"event":   EventAddChannel,
		"channel": channel,
	}

	recvChan := make(chan interface{})
	o.messageChannels.Store(channel, recvChan)

	o.command(data, parameters)

	select {
	case recv := <-recvChan:
		if recv != nil {
			result := recv.(map[string]interface{})["result"]
			if result != nil && result.(bool) {
				orders := recv.(map[string]interface{})["orders"].([]interface{})

				if len(orders) == 0 {
					return nil
				}

				result := make([]OrderInfo, len(orders))

				for i, tmp := range orders {
					order := tmp.(map[string]interface{})

					var orderType TradeType
					var avgPrice float64
					if o.instrumentType == InstrumentTypeSwap {
						orderType = OkexGetTradeTypeByFloat(order["type"].(float64))
						avgPrice = order["price_avg"].(float64)
					} else if o.instrumentType == InstrumentTypeSpot {
						orderType = o.getTradeTypeByString(order["type"].(string))
						avgPrice = order["avg_price"].(float64)
					}
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
		}

		log.Printf("Fail to get Order Info")
		return nil

	case <-time.After(DefaultTimeoutSec * time.Second):
		log.Printf("Timeout to get user info")
		go o.triggerEvent(EventLostConnection)
		return nil
	}
}

func (o *OKEXV3API) GetBalance() map[string]interface{} {

	var channel string
	var parameters map[string]string

	if o.instrumentType == InstrumentTypeSwap {
		channel = ChannelFutureUserInfo

	} else if o.instrumentType == InstrumentTypeSpot {
		channel = ChannelSpotUserInfo
	}

	parameters = map[string]string{
		"api_key": o.apiKey,
		// "secret_key": constSecretKey,
	}

	data := map[string]interface{}{
		"event":   EventAddChannel,
		"channel": channel,
	}

	recvChan := make(chan interface{})
	o.messageChannels.Store(channel, recvChan)

	o.command(data, parameters)

	select {
	case recv := <-recvChan:
		if o.instrumentType == InstrumentTypeSwap {
			if recv != nil {
				// values := recv.(map[string]interface{})[coin]
				// if values != nil {
				// 	balance := values.(map[string]interface{})["balance"]
				// 	contracts := values.(map[string]interface{})["contracts"]
				// 	var bond float64
				// 	if contracts != nil && len(contracts.([]interface{})) > 0 {
				// 		for _, contract := range contracts.([]interface{}) {
				// 			bond += contract.(map[string]interface{})["bond"].(float64)
				// 		}
				// 	}

				// 	if balance != nil {
				// 		return balance.(float64), bond
				// 	}
				// }
				result := make(map[string]interface{})
				balances := recv.(map[string]interface{})
				log.Printf("Balance:%v", balances)
				for coin, value := range balances {
					var bond float64
					balance := value.(map[string]interface{})["rights"]
					contracts := value.(map[string]interface{})["contracts"]
					if contracts != nil && len(contracts.([]interface{})) > 0 {
						for _, contract := range contracts.([]interface{}) {
							bond += contract.(map[string]interface{})["bond"].(float64)
						}
					}
					result[coin] = map[string]interface{}{
						"balance": balance,
						"bond":    bond,
					}
				}

				return result
			}

			return nil

		} else if o.instrumentType == InstrumentTypeSpot {
			if recv != nil {
				funds := recv.(map[string]interface{})["funds"]
				if funds != nil {
					balances := funds.(map[string]interface{})["free"].(map[string]interface{})
					// if balance != nil {
					// 	result, _ := strconv.ParseFloat(balance.(map[string]interface{})[coin].(string), 64)
					// 	return result, -1
					// }
					result := make(map[string]interface{})
					for coin, balance := range balances {
						value, _ := strconv.ParseFloat(balance.(string), 64)
						result[coin] = map[string]interface{}{
							"balance": value,
						}
					}
					return result
				}
			}
		}

		return nil
	case <-time.After(DefaultTimeoutSec * time.Second):
		log.Printf("Timeout to get user info")
		go o.triggerEvent(EventLostConnection)
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

func (p *OKEXV3API) GetKline(pair string, period int, limit int) []KlineValue {
	return nil
}
