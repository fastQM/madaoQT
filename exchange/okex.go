package exchange

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	websocket "github.com/gorilla/websocket"
)

const NameOKEX = "okex"

const contractUrl = "wss://real.okex.com:10440/websocket/okexapi"
const currentUrl = "wss://real.okex.com:10441/websocket"

const X_BTC = "btc"
const X_LTC = "ltc"
const X_ETH = "eth"

const Y_THIS_WEEK = "this_week"
const Y_NEXT_WEEK = "next_week"
const Y_QUARTER = "quarter"

const Z_1min = "1min"
const Z_3min = "3min"
const Z_5min = "5min"
const Z_15min = "15min"
const Z_30min = "30min"
const Z_1hour = "1hour"
const Z_2hour = "2hour"
const Z_4hour = "4hour"
const Z_6hour = "6hour"
const Z_12hour = "12hour"
const Z_day = "day"
const Z_3day = "3day"
const Z_week = "week"

// event
const EventAddChannel = "addChannel"
const EventRemoveChannel = "removeChannel"

// 合约行情API
const ChannelContractTicker = "ok_sub_futureusd_X_ticker_Y"
const ChannelContractDepth = "ok_sub_futureusd_X_depth_Y_Z"

const ChannelLogin = "login"
const ChannelContractTrade = "ok_futureusd_trade"
const ChannelContractTradeCancel = "ok_futureusd_cancel_order"
const ChannelUserInfo = "ok_futureusd_userinfo"
const ChannelOrderInfo = "ok_futureusd_orderinfo"
const ChannelSubTradesInfo = "ok_sub_futureusd_trades"
const ChannelSubUserInfo = "ok_sub_futureusd_userinfo"
const ChannelSubPositions = "ok_sub_futureusd_positions"

// 现货行情API
const ChannelCurrentChannelTicker = "ok_sub_spot_X_ticker"
const ChannelCurrentDepth = "ok_sub_spot_X_depth_Y"

// 现货交易API
const ChannelSpotOrder = "ok_spot_order"
const ChannelSpotCancelOrder = "ok_spot_cancel_order"
const ChannelSpotUserInfo = "ok_spot_userinfo"
const ChannelSpotOrderInfo = "ok_spot_orderinfo"

const Debug = false
const DefaultTimeoutSec = 3

type ContractItemValueIndex int8

const (
	UsdPriceIndex ContractItemValueIndex = iota
	ContractQuantity
	CoinQuantity
	TotalCoinQuantity
	TotalContractQuantity
)

type OKExAPI struct {
	conn      *websocket.Conn
	apiKey    string
	secretKey string

	tickerList []TickerListItem
	depthList  []DepthListItem
	event      chan EventType
	tradeType  TradeType

	/* Each channel has a depth */
	messageChannels sync.Map
}

func formatTimeOKEX() string {
	timeFormat := "2006-01-02 06:04:05"
	location, _ := time.LoadLocation("Local")
	// unixTime := time.Unix(timestamp/1000, 0)
	unixTime := time.Now()
	return unixTime.In(location).Format(timeFormat)
}

var handlderOkexFuture *OKExAPI
var handlerOkexSpot *OKExAPI

func NewOKExFutureApi(config *InitConfig) *OKExAPI {

	if handlderOkexFuture == nil && config != nil {
		handlderOkexFuture := new(OKExAPI)
		futureConfig := InitConfig{
			Api:    config.Api,
			Secret: config.Secret,
			Custom: map[string]interface{}{"tradeType": TradeTypeFuture},
		}
		handlderOkexFuture.Init(futureConfig)

		return handlderOkexFuture
	}

	return handlderOkexFuture
}

func NewOKExSpotApi(config *InitConfig) *OKExAPI {
	if handlerOkexSpot == nil && config != nil {
		handlerOkexSpot := new(OKExAPI)
		spotConfig := InitConfig{
			Api:    config.Api,
			Secret: config.Secret,
			Custom: map[string]interface{}{"tradeType": TradeTypeSpot},
		}
		handlerOkexSpot.Init(spotConfig)

		return handlerOkexSpot
	}

	return handlerOkexSpot

}

func (o *OKExAPI) WatchEvent() chan EventType {
	return o.event
}

func (o *OKExAPI) triggerEvent(event EventType) {
	o.event <- event
}

func (o *OKExAPI) Init(config InitConfig) {

	o.tickerList = nil
	o.depthList = nil
	o.event = make(chan EventType)
	o.apiKey = config.Api
	o.secretKey = config.Secret

	o.tradeType = config.Custom["tradeType"].(TradeType)

}

func (o *OKExAPI) Start() {

	var url string

	if o.tradeType == TradeTypeFuture {
		url = contractUrl
	} else if o.tradeType == TradeTypeSpot {
		url = currentUrl
	} else {
		Logger.Error("%s")
		return
	}

	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		Logger.Errorf("Fail to dial: %v", err)
		go o.triggerEvent(EventError)
		return
	}

	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				Logger.Errorf("Fail to read:%v", err)
				go o.triggerEvent(EventError)
				return
			}

			if Debug {
				Logger.Debugf("recv: %s", message)
			}

			var response []map[string]interface{}
			if err = json.Unmarshal([]byte(message), &response); err != nil {
				Logger.Errorf("Fail to Unmarshal:%v", err)
				continue
			}

			channel := response[0]["channel"].(string)

			if channel == EventAddChannel || channel == EventRemoveChannel {

			} else if channel == ChannelUserInfo || channel == ChannelSpotUserInfo {
				if recvChan, ok := o.messageChannels.Load(channel); recvChan != nil && ok {
					data := response[0]["data"].(map[string]interface{})
					if data != nil && data["result"] == true {
						info := data["info"]
						go func() {
							recvChan.(chan interface{}) <- info
							close(recvChan.(chan interface{}))
							o.messageChannels.Delete(channel)
						}()
					} else if data["result"] == false {
						Logger.Errorf("Response Error: %v", message)
						goto END
					}
				}
			} else {
				// 处理下单取消订单
				acceptChannels := []string{
					ChannelOrderInfo,
					// ChannelUserInfo,
					ChannelSubTradesInfo,
					ChannelContractTrade,
					ChannelContractTradeCancel,
					ChannelSpotOrder,
					ChannelSpotCancelOrder,
					// ChannelSpotUserInfo,
					ChannelSpotOrderInfo,
				}

				for _, accept := range acceptChannels {
					if accept == channel {
						go func() {
							if recvChan, ok := o.messageChannels.Load(channel); recvChan != nil && ok {
								recvChan.(chan interface{}) <- response[0]["data"]
								close(recvChan.(chan interface{}))
								o.messageChannels.Delete(channel)
							}
						}()

						goto END
					}
				}

				// 处理期货价格深度
				if recvChan, ok := o.messageChannels.Load(channel); recvChan != nil && ok {

					// depth := new(DepthValue)
					data := response[0]["data"].(map[string]interface{})

					// unitTime := time.Unix(int64(data["timestamp"].(float64))/1000, 0)
					// timeHM := unitTime.Format("2006-01-02 03:04:05")
					// o.depthList[i].Time = timeHM
					if data["asks"] == nil {
						log.Printf("Recv Invalid data from %s:%v",
							o.GetExchangeName(), response)
						goto END
					}

					go func() {
						recvChan.(chan interface{}) <- response[0]["data"]
						close(recvChan.(chan interface{}))
						o.messageChannels.Delete(channel)
					}()

					goto END

				}

				// 处理现货价格
				if o.tickerList != nil {
					for i, ticker := range o.tickerList {
						if ticker.Name == channel {
							// o.tickerList[i].Time = timeHM
							o.tickerList[i].Value = response[0]["data"]
							o.tickerList[i].ticket++
							goto END
						}
					}
				}
			}

		END:
		}

	}()

	o.conn = c

	go o.triggerEvent(EventConnected)

}

func (o *OKExAPI) Close() {
	if o.conn != nil {
		o.conn.Close()
	}
}

/*
① X值为：btc, ltc
② Y值为：this_week, next_week, quarter
*/
func (o *OKExAPI) StartContractTicker(coin string, period string, tag string) {
	channel := strings.Replace(ChannelContractTicker, "X", coin, 1)
	channel = strings.Replace(channel, "Y", period, 1)

	ticker := TickerListItem{
		Tag:  tag,
		Name: channel,
	}

	o.tickerList = append(o.tickerList, ticker)

	data := map[string]string{
		"event":   "addChannel",
		"channel": channel,
	}

	o.command(data, nil)
}

/*
① X值为：ltc_btc eth_btc etc_btc bch_btc btc_usdt
eth_usdt ltc_usdt etc_usdt bch_usdt etc_eth bt1_btc
bt2_btc btg_btc qtum_btc hsr_btc neo_btc gas_btc
qtum_usdt hsr_usdt neo_usdt gas_usdt
*/
func (o *OKExAPI) StartCurrentTicker(coinA string, coinB string, tag string) {
	pair := (coinA + "_" + coinB)

	channel := strings.Replace(ChannelCurrentChannelTicker, "X", pair, 1)

	ticker := TickerListItem{
		Tag:  tag,
		Name: channel,
	}

	o.tickerList = append(o.tickerList, ticker)

	data := map[string]string{
		"event":   "addChannel",
		"channel": channel,
	}

	o.command(data, nil)
}

func (o *OKExAPI) GetExchangeName() string {
	return NameOKEX
}

func (o *OKExAPI) GetTickerValue(tag string) *TickerValue {
	for _, ticker := range o.tickerList {
		if ticker.Tag == tag {
			if ticker.Value != nil {
				// return ticker.Value.(map[string]interface{})
				if ticker.oldticket == ticker.ticket {
					Logger.Errorf("[%s][%s]Ticker数据未更新", o.GetExchangeName(), ticker.Name)
				} else {
					ticker.oldticket = ticker.ticket
				}

				var lastValue float64
				tmp := ticker.Value.(map[string]interface{})
				if o.tradeType == TradeTypeFuture {
					lastValue = tmp["last"].(float64)
				} else if o.tradeType == TradeTypeSpot {
					value, _ := strconv.ParseFloat(tmp["last"].(string), 64)
					lastValue = value
				}

				// unitTime := time.Unix(int64(tmp["timestamp"].(float64))/1000, 0)
				// timeHM := unitTime.Format("2006-01-02 06:04:05")

				tickerValue := &TickerValue{
					Last: lastValue,
					Time: formatTimeOKEX(),
				}

				return tickerValue
			}
		}
	}

	return nil
}

/*
	① X值为：btc, ltc
	② Y值为：this_week, next_week, quarter
	③ Z值为：5, 10, 20(获取深度条数)
*/
func (o *OKExAPI) SwithContractDepth(open bool, coin string, period string, depth string) string {

	channel := strings.Replace(ChannelContractDepth, "X", coin, 1)
	channel = strings.Replace(channel, "Y", period, 1)
	channel = strings.Replace(channel, "Z", depth, 1)

	var event string
	if open {
		event = EventAddChannel
		o.messageChannels.Store(channel, make(chan interface{}))

	} else {
		event = EventRemoveChannel
		o.messageChannels.Delete(channel)
	}

	data := map[string]string{
		"event":   event,
		"channel": channel,
	}

	o.command(data, nil)

	return channel
}

/*
X值为：ltc_btc eth_btc etc_btc bch_btc btc_usdt eth_usdt
ltc_usdt etc_usdt bch_usdt etc_eth bt1_btc bt2_btc btg_btc
qtum_btc hsr_btc neo_btc gas_btc qtum_usdt hsr_usdt neo_usdt gas_usdt
Y值为: 5, 10, 20(获取深度条数)
*/
func (o *OKExAPI) SwitchCurrentDepth(open bool, coinA string, coinB string, depth string) string {
	pair := (coinA + "_" + coinB)
	channel := strings.Replace(ChannelCurrentDepth, "X", pair, 1)
	channel = strings.Replace(channel, "Y", depth, 1)

	var event string
	if open {
		event = EventAddChannel
		o.messageChannels.Store(channel, make(chan interface{}))
	} else {
		event = EventRemoveChannel
		o.messageChannels.Delete(channel)
	}

	data := map[string]string{
		"event":   event,
		"channel": channel,
	}

	o.command(data, nil)
	return channel

}

func (o *OKExAPI) GetDepthValue(coinA string, coinB string, orderQuantity float64) *DepthValue {

	var channel string

	if o.tradeType == TradeTypeFuture {
		channel = o.SwithContractDepth(true, coinA, "this_week", "20")
		// defer o.SwithContractDepth(false, coinA, "this_week", "20")
	} else if o.tradeType == TradeTypeSpot {
		channel = o.SwitchCurrentDepth(true, coinA, coinB, "20")
		// defer o.SwitchCurrentDepth(false, coinA, coinB, "20")
	}

	recvChan, _ := o.messageChannels.Load(channel)

	select {
	case <-time.After(DefaultTimeoutSec * time.Second):
		log.Print("timeout to wait for the depths")
		return nil
	case recv := <-recvChan.(chan interface{}):
		depth := new(DepthValue)

		data := recv.(map[string]interface{})
		depth.Time = formatTimeOKEX()

		asks := data["asks"].([]interface{})
		bids := data["bids"].([]interface{})

		if o.tradeType == TradeTypeFuture {
			if asks != nil && len(asks) > 0 {
				askList := make([]DepthPrice, len(asks))
				for i, ask := range asks {
					values := ask.([]interface{})
					askList[i].price, _ = strconv.ParseFloat(values[UsdPriceIndex].(string), 64)
					askList[i].qty, _ = strconv.ParseFloat(values[CoinQuantity].(string), 64)
				}

				depth.AskAverage, depth.AskQty = GetDepthAveragePrice(askList)
				depth.AskByOrder, depth.AskPrice = GetDepthPriceByOrder(DepthTypeAsks, askList, orderQuantity)
			}

			if bids != nil && len(bids) > 0 {
				bidList := make([]DepthPrice, len(bids))
				for i, bid := range bids {
					values := bid.([]interface{})
					bidList[i].price, _ = strconv.ParseFloat(values[UsdPriceIndex].(string), 64)
					bidList[i].qty, _ = strconv.ParseFloat(values[CoinQuantity].(string), 64)
				}

				depth.BidAverage, depth.BidQty = GetDepthAveragePrice(bidList)
				depth.BidByOrder, depth.BidPrice = GetDepthPriceByOrder(DepthTypeBids, bidList, orderQuantity)
			}

		} else if o.tradeType == TradeTypeSpot {
			if asks != nil && len(asks) > 0 {
				askList := make([]DepthPrice, len(asks))
				for i, ask := range asks {
					values := ask.([]interface{})
					askList[i].price, _ = strconv.ParseFloat(values[0].(string), 64)
					askList[i].qty, _ = strconv.ParseFloat(values[1].(string), 64)
				}

				depth.AskAverage, depth.AskQty = GetDepthAveragePrice(askList)
				depth.AskByOrder, depth.AskPrice = GetDepthPriceByOrder(DepthTypeAsks, askList, orderQuantity)
			}

			if bids != nil && len(bids) > 0 {
				bidList := make([]DepthPrice, len(bids))
				for i, bid := range bids {
					values := bid.([]interface{})
					bidList[i].price, _ = strconv.ParseFloat(values[0].(string), 64)
					bidList[i].qty, _ = strconv.ParseFloat(values[1].(string), 64)
				}

				depth.BidAverage, depth.BidQty = GetDepthAveragePrice(bidList)
				depth.BidByOrder, depth.BidPrice = GetDepthPriceByOrder(DepthTypeBids, bidList, orderQuantity)
			}
		}

		// log.Printf("Result:%v", depth)
		return depth
	}
}

func (o *OKExAPI) command(data map[string]string, parameters map[string]string) error {
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
		log.Printf("Cmd:%v", string(cmd))
	}

	o.conn.WriteMessage(websocket.TextMessage, cmd)

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
func (o *OKExAPI) Trade(configs TradeConfig) *TradeResult {

	var channel string
	var data, parameters map[string]string

	if o.tradeType == TradeTypeFuture {

		channel = ChannelContractTrade

		parameters = map[string]string{
			"api_key":       o.apiKey,
			"symbol":        configs.Coin + "_usd",
			"contract_type": "this_week",
			"price":         strconv.FormatFloat(configs.Price, 'f', 2, 64),
			// the exact amount orders is amount/level_rate
			"amount":      strconv.FormatFloat(configs.Amount, 'f', 2, 64),
			"type":        o.getOrderTypeString(configs.Type),
			"match_price": "0",
			"lever_rate":  "10",
		}

	} else if o.tradeType == TradeTypeSpot {

		channel = ChannelSpotOrder

		parameters = map[string]string{
			"api_key": o.apiKey,
			"symbol":  configs.Coin,
			"type":    o.getOrderTypeString(configs.Type),
			"price":   strconv.FormatFloat(configs.Price, 'f', 2, 64),
			"amount":  strconv.FormatFloat(configs.Amount, 'f', 2, 64),
		}
	}

	data = map[string]string{
		"event":   EventAddChannel,
		"channel": channel,
	}

	recvChan := make(chan interface{})
	o.messageChannels.Store(channel, recvChan)

	o.command(data, parameters)

	select {
	case <-time.After(DefaultTimeoutSec * time.Second):
		return &TradeResult{
			Error: errors.New("Timeout"),
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

func (o *OKExAPI) CancelOrder(order OrderInfo) map[string]interface{} {

	var channel string
	var data, parameters map[string]string

	if o.tradeType == TradeTypeFuture {

		channel = ChannelContractTradeCancel

		parameters = map[string]string{
			"api_key":       o.apiKey,
			"order_id":      order.OrderID,
			"symbol":        order.Coin,
			"contract_type": "this_week",
		}

	} else if o.tradeType == TradeTypeSpot {
		channel = ChannelSpotCancelOrder

		parameters = map[string]string{
			"api_key":  o.apiKey,
			"order_id": order.OrderID,
			"symbol":   order.Coin,
		}

	}

	data = map[string]string{
		"event":   EventAddChannel,
		"channel": channel,
	}

	recvChan := make(chan interface{})
	o.messageChannels.Store(channel, recvChan)

	o.command(data, parameters)

	select {
	case <-time.After(DefaultTimeoutSec * time.Second):
		return nil
	case message := <-recvChan:
		// log.Printf("message:%v", message)
		return message.(map[string]interface{})
	}

}

func (o *OKExAPI) GetOrderInfo(configs map[string]interface{}) []OrderInfo {

	var channel string
	var data, parameters map[string]string

	if o.tradeType == TradeTypeFuture {

		channel = ChannelOrderInfo

		parameters = map[string]string{
			"api_key": o.apiKey,
			// "secret_key": constSecretKey,
		}

		for k, v := range configs {
			parameters[k] = v.(string)
		}

	} else if o.tradeType == TradeTypeSpot {

		channel = ChannelSpotOrderInfo

		parameters = map[string]string{
			"api_key": o.apiKey,
			// "secret_key": constSecretKey,
		}

		for k, v := range configs {
			parameters[k] = v.(string)
		}
	}

	data = map[string]string{
		"event":   EventAddChannel,
		"channel": channel,
	}

	recvChan := make(chan interface{})
	o.messageChannels.Store(channel, recvChan)

	o.command(data, parameters)

	select {
	case recv := <-recvChan:
		orders := recv.(map[string]interface{})["orders"].([]interface{})

		if len(orders) == 0 {
			return nil
		}

		result := make([]OrderInfo, len(orders))

		for i, tmp := range orders {
			order := tmp.(map[string]interface{})
			item := OrderInfo{
				Coin:    order["symbol"].(string),
				OrderID: strconv.FormatFloat(order["order_id"].(float64), 'f', 0, 64),
				// OrderID: strconv.FormatInt(order["order_id"].(int64), 64),
				Price:  order["price"].(float64),
				Amount: order["amount"].(float64),
				Type:   o.getOrderType(order["type"].(string)),
				Status: o.getStatus(order["status"].(float64)),
			}
			result[i] = item
		}

		return result
	case <-time.After(DefaultTimeoutSec * time.Second):
		log.Printf("Timeout to get user info")
		return nil
	}
}

func (o *OKExAPI) GetBalance(coin string) float64 {

	var channel string
	var data, parameters map[string]string

	if o.tradeType == TradeTypeFuture {
		channel = ChannelUserInfo

	} else if o.tradeType == TradeTypeSpot {
		channel = ChannelSpotUserInfo
	}

	parameters = map[string]string{
		"api_key": o.apiKey,
		// "secret_key": constSecretKey,
	}

	data = map[string]string{
		"event":   EventAddChannel,
		"channel": channel,
	}

	recvChan := make(chan interface{})
	o.messageChannels.Store(channel, recvChan)

	o.command(data, parameters)

	select {
	case recv := <-recvChan:
		if o.tradeType == TradeTypeFuture {
			if recv != nil {
				values := recv.(map[string]interface{})[coin]
				if values != nil {
					balance := values.(map[string]interface{})["balance"]
					if balance != nil {
						return balance.(float64)
					}
				}
			}
			return -1

		} else if o.tradeType == TradeTypeSpot {
			if recv != nil {
				funds := recv.(map[string]interface{})["funds"]
				if funds != nil {
					balance := funds.(map[string]interface{})["free"]
					if balance != nil {
						result, _ := strconv.ParseFloat(balance.(map[string]interface{})[coin].(string), 64)
						return result
					}
				}
			}
		}

		return -1
	case <-time.After(DefaultTimeoutSec * time.Second):
		log.Printf("Timeout to get user info")
		return -1
	}

}

func (o *OKExAPI) getStatus(status float64) OrderStatusType {
	switch status {
	case 0:
		return OrderStatusOpen
	case 1:
		return OrderStatusPartDone
	case 2:
		return OrderStatusDone
	case 3:
		return OrderStatusCanceling
	case 4:
		return OrderStatusCanceled
	}

	return OrderStatusUnknown
}

func (o *OKExAPI) getOrderType(orderType string) OrderType {
	switch orderType {
	case "1":
		return OrderTypeOpenLong
	case "2":
		return OrderTypeOpenShort
	case "3":
		return OrderTypeCloseLong
	case "4":
		return OrderTypeCloseShort
	case "buy":
		return OrderTypeBuy
	case "sell":
		return OrderTypeSell
	}

	return OrderTypeUnknown
}

func (o *OKExAPI) getOrderTypeString(orderType OrderType) string {

	switch orderType {
	case OrderTypeOpenLong:
		return "1"
	case OrderTypeOpenShort:
		return "2"
	case OrderTypeCloseLong:
		return "3"
	case OrderTypeCloseShort:
		return "4"
	case OrderTypeBuy:
		return "buy"
	case OrderTypeSell:
		return "sell"
	}

	Logger.Errorf("[%s]getOrderType: Invalid type", o.GetExchangeName())
	return ""
}
