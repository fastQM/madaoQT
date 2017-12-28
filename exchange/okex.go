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

const NameOKEXSpot = "OkexSpot"
const NameOKEXFuture = "OkexFuture"

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

const Debug = true
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
	Ticker ITicker

	conn      *websocket.Conn
	apiKey    string
	secretKey string

	tickerList   []TickerListItem
	depthList    []DepthListItem
	event        chan EventType
	exchangeType ExchangeType

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

const constOKEXApiKey = "a982120e-8505-41db-9ae3-0c62dd27435c"
const constOEXSecretKey = "71430C7FA63A067724FB622FB3031970"

func NewOKExFutureApi(config *InitConfig) *OKExAPI {

	if config == nil {
		config = &InitConfig{}
	}

	config.Custom = map[string]interface{}{"exchangeType": ExchangeTypeFuture}
	future := new(OKExAPI)
	future.Init(*config)
	return future
}

func NewOKExSpotApi(config *InitConfig) *OKExAPI {

	if config == nil {
		config = &InitConfig{}
	}

	config.Custom = map[string]interface{}{"exchangeType": ExchangeTypeSpot}
	spot := new(OKExAPI)
	spot.Init(*config)

	return spot

}

func (o *OKExAPI) WatchEvent() chan EventType {
	return o.event
}

func (o *OKExAPI) triggerEvent(event EventType) {
	o.event <- event
}

func (o *OKExAPI) Init(config InitConfig) {

	o.Ticker = config.Ticker
	o.tickerList = nil
	o.depthList = nil
	o.event = make(chan EventType)
	o.apiKey = config.Api
	o.secretKey = config.Secret
	o.exchangeType = config.Custom["exchangeType"].(ExchangeType)

	if o.apiKey == "" || o.secretKey == "" {
		Logger.Debug("没有配置API，不可执行交易操作")
	}
}

func (o *OKExAPI) Start() {

	var url string

	if o.exchangeType == ExchangeTypeFuture {
		url = contractUrl
	} else if o.exchangeType == ExchangeTypeSpot {
		url = currentUrl
	} else {
		Logger.Error("Invalid type")
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
				filters := []string{
					"depth",
					"ticker",
				}

				var filtered = false
				for _, filter := range filters {
					if strings.Contains(string(message), filter) {
						filtered = true
					}
				}

				if !filtered {
					Logger.Debugf("recv: %s", message)
				}

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
						Logger.Errorf("Response Error: %s", message)
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

					/*
						1. OKEX will send the message periodly, and if we remove the channel, the socket will be closed;
						2. There is possiblity that more than single routine will be called at the same time, so we will need to remove the channel here
					*/
					o.messageChannels.Delete(channel)

					data := response[0]["data"].(map[string]interface{})

					// unitTime := time.Unix(int64(data["timestamp"].(float64))/1000, 0)
					// timeHM := unitTime.Format("2006-01-02 03:04:05")
					// o.depthList[i].Time = timeHM
					if data["asks"] == nil {
						log.Printf("Recv Invalid data from %s:%v", o.GetExchangeName(), response)
						goto END
					}

					go func() {
						recvChan.(chan interface{}) <- response[0]["data"]
						close(recvChan.(chan interface{}))
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

							tmp := o.tickerList[i].Value.(map[string]interface{})
							lastValue, _ := strconv.ParseFloat(tmp["last"].(string), 64)
							tickerValue := TickerValue{
								Last: lastValue,
								Time: formatTimeOKEX(),
							}
							if o.Ticker != nil {
								o.Ticker.Ticker(o.GetExchangeName(), tickerValue)
							}
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
func (o *OKExAPI) StartContractTicker(pair string, period string, tag string) {

	coins := ParsePair(pair)

	channel := strings.Replace(ChannelContractTicker, "X", coins[0], 1)
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
func (o *OKExAPI) StartCurrentTicker(pair string, tag string) {
	coins := ParsePair(pair)
	channel := strings.Replace(ChannelCurrentChannelTicker, "X", coins[0]+"_"+coins[1], 1)

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
	if o.exchangeType == ExchangeTypeFuture {
		return NameOKEXFuture
	} else if o.exchangeType == ExchangeTypeSpot {
		return NameOKEXSpot
	}

	return "Invalid Exchange type"
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

				tmp := ticker.Value.(map[string]interface{})
				// if o.exchangeType == ExchangeTypeFuture {
				// 	lastValue = tmp["last"].(float64)
				// } else if o.exchangeType == ExchangeTypeSpot {
				// 	value, _ := strconv.ParseFloat(tmp["last"].(string), 64)
				// 	lastValue = value
				// }
				lastValue, _ := strconv.ParseFloat(tmp["last"].(string), 64)
				volume, _ := strconv.ParseFloat(tmp["vol"].(string), 64)

				tickerValue := &TickerValue{
					Last:   lastValue,
					Time:   formatTimeOKEX(),
					Volume: volume,
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
func (o *OKExAPI) SwithContractDepth(open bool, coin string, period string, depth string) chan interface{} {
	coin = strings.TrimSuffix(coin, "_usd")
	channel := strings.Replace(ChannelContractDepth, "X", coin, 1)
	channel = strings.Replace(channel, "Y", period, 1)
	channel = strings.Replace(channel, "Z", depth, 1)

	var event string
	recvChan := make(chan interface{})

	if open {
		event = EventAddChannel
		o.messageChannels.Store(channel, recvChan)

	} else {
		event = EventRemoveChannel
		o.messageChannels.Delete(channel)
	}

	data := map[string]string{
		"event":   event,
		"channel": channel,
	}

	o.command(data, nil)

	return recvChan
}

/*
X值为：ltc_btc eth_btc etc_btc bch_btc btc_usdt eth_usdt
ltc_usdt etc_usdt bch_usdt etc_eth bt1_btc bt2_btc btg_btc
qtum_btc hsr_btc neo_btc gas_btc qtum_usdt hsr_usdt neo_usdt gas_usdt
Y值为: 5, 10, 20(获取深度条数)
*/
func (o *OKExAPI) SwitchCurrentDepth(open bool, pair string, depth string) chan interface{} {
	channel := strings.Replace(ChannelCurrentDepth, "X", pair, 1)
	channel = strings.Replace(channel, "Y", depth, 1)

	var event string
	recvChan := make(chan interface{})

	if open {
		event = EventAddChannel
		o.messageChannels.Store(channel, recvChan)
	} else {
		event = EventRemoveChannel
		o.messageChannels.Delete(channel)
	}

	data := map[string]string{
		"event":   event,
		"channel": channel,
	}

	o.command(data, nil)
	return recvChan

}

func (o *OKExAPI) GetDepthValue(coin string, price float64, limit float64, orderQuantity float64, tradeType TradeType) *DepthValue {

	var recvChan chan interface{}
	coins := ParsePair(coin)

	if o.exchangeType == ExchangeTypeFuture {
		recvChan = o.SwithContractDepth(true, coins[0], "this_week", "20")
		// defer o.SwithContractDepth(false, coinA, "this_week", "20")
	} else if o.exchangeType == ExchangeTypeSpot {
		recvChan = o.SwitchCurrentDepth(true, coins[0]+"_"+coins[1], "20")
		// defer o.SwitchCurrentDepth(false, coinA, coinB, "20")
	}

	// o.depthList = append(o.depthList, DepthListItem{
	// 	Name: channel,
	// })

	select {
	case <-time.After(DefaultTimeoutSec * time.Second):
		log.Print("timeout to wait for the depths")
		return nil
	case recv := <-recvChan:
		depth := new(DepthValue)

		data := recv.(map[string]interface{})
		depth.Time = formatTimeOKEX()

		asks := data["asks"].([]interface{})
		bids := data["bids"].([]interface{})

		var list []DepthPrice

		if o.exchangeType == ExchangeTypeFuture {

			if asks != nil && len(asks) > 0 {
				askList := make([]DepthPrice, len(asks))
				for i, ask := range asks {
					values := ask.([]interface{})
					// askList[i].price, _ = strconv.ParseFloat(values[UsdPriceIndex].(string), 64)
					// askList[i].qty, _ = strconv.ParseFloat(values[CoinQuantity].(string), 64)
					askList[i].price = values[UsdPriceIndex].(float64)
					askList[i].qty = values[CoinQuantity].(float64)
				}

				depth.AskAverage, depth.AskQty = GetDepthAveragePrice(askList)
				depth.AskByOrder, depth.AskPrice = GetDepthPriceByOrder(askList, orderQuantity)
				if tradeType == TradeTypeOpenLong || tradeType == TradeTypeCloseShort {
					list = RevertDepthArray(askList)
				}
			}

			if bids != nil && len(bids) > 0 {
				bidList := make([]DepthPrice, len(bids))
				for i, bid := range bids {
					values := bid.([]interface{})
					// bidList[i].price, _ = strconv.ParseFloat(values[UsdPriceIndex].(string), 64)
					// bidList[i].qty, _ = strconv.ParseFloat(values[CoinQuantity].(string), 64)
					bidList[i].price = values[UsdPriceIndex].(float64)
					bidList[i].qty = values[CoinQuantity].(float64)
				}

				depth.BidAverage, depth.BidQty = GetDepthAveragePrice(bidList)
				depth.BidByOrder, depth.BidPrice = GetDepthPriceByOrder(bidList, orderQuantity)

				if tradeType == TradeTypeOpenShort || tradeType == TradeTypeCloseLong {
					list = bidList
				}
			}

		} else if o.exchangeType == ExchangeTypeSpot {
			if asks != nil && len(asks) > 0 {
				askList := make([]DepthPrice, len(asks))
				for i, ask := range asks {
					values := ask.([]interface{})
					askList[i].price, _ = strconv.ParseFloat(values[0].(string), 64)
					askList[i].qty, _ = strconv.ParseFloat(values[1].(string), 64)
				}

				depth.AskAverage, depth.AskQty = GetDepthAveragePrice(askList)
				depth.AskByOrder, depth.AskPrice = GetDepthPriceByOrder(askList, orderQuantity)

				if tradeType == TradeTypeBuy {
					list = RevertDepthArray(askList)
				}

			}

			if bids != nil && len(bids) > 0 {
				bidList := make([]DepthPrice, len(bids))
				for i, bid := range bids {
					values := bid.([]interface{})
					bidList[i].price, _ = strconv.ParseFloat(values[0].(string), 64)
					bidList[i].qty, _ = strconv.ParseFloat(values[1].(string), 64)
				}

				depth.BidAverage, depth.BidQty = GetDepthAveragePrice(bidList)
				depth.BidByOrder, depth.BidPrice = GetDepthPriceByOrder(bidList, orderQuantity)

				if tradeType == TradeTypeSell {
					list = bidList
				}
			}
		}

		depth.LimitTradePrice, depth.LimitTradeAmount = GetDepthPriceByPrice(list, price, limit, orderQuantity)
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

	coins := ParsePair(configs.Coin)

	if o.exchangeType == ExchangeTypeFuture {

		channel = ChannelContractTrade

		parameters = map[string]string{
			"api_key":       o.apiKey,
			"symbol":        coins[0] + "_usd",
			"contract_type": "this_week",
			"price":         strconv.FormatFloat(configs.Price, 'f', 2, 64),
			// the exact amount orders is amount/level_rate
			"amount":      strconv.FormatFloat(configs.Amount, 'f', 2, 64),
			"type":        o.getTradeTypeString(configs.Type),
			"match_price": "0",
			"lever_rate":  "10",
		}

	} else if o.exchangeType == ExchangeTypeSpot {

		channel = ChannelSpotOrder

		parameters = map[string]string{
			"api_key": o.apiKey,
			"symbol":  coins[0] + "_" + coins[1],
			"type":    o.getTradeTypeString(configs.Type),
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

func (o *OKExAPI) CancelOrder(order OrderInfo) *TradeResult {

	var channel string
	var data, parameters map[string]string

	coins := ParsePair(order.Pair)

	if o.exchangeType == ExchangeTypeFuture {

		channel = ChannelContractTradeCancel

		parameters = map[string]string{
			"api_key":       o.apiKey,
			"order_id":      order.OrderID,
			"symbol":        coins[0] + "_" + coins[1],
			"contract_type": "this_week",
		}

	} else if o.exchangeType == ExchangeTypeSpot {
		channel = ChannelSpotCancelOrder

		parameters = map[string]string{
			"api_key":  o.apiKey,
			"order_id": order.OrderID,
			"symbol":   coins[0] + "_" + coins[1],
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

func (o *OKExAPI) GetOrderInfo(filter OrderInfo) []OrderInfo {

	var channel string
	var data, parameters map[string]string

	pair := ParsePair(filter.Pair)

	if o.exchangeType == ExchangeTypeFuture {

		channel = ChannelOrderInfo

		parameters = map[string]string{
			"api_key": o.apiKey,
			// "secret_key": constSecretKey,
			"contract_type": "this_week",
			// "status":        "1",
			"current_page": "1",
			"page_length":  "1",
			"order_id":     filter.OrderID,
			"symbol":       pair[0] + "_" + pair[1],
		}

	} else if o.exchangeType == ExchangeTypeSpot {

		channel = ChannelSpotOrderInfo

		parameters = map[string]string{
			"api_key": o.apiKey,
			// "secret_key": constSecretKey,
			"order_id": filter.OrderID,
			"symbol":   pair[0] + "_" + pair[1],
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

			var orderType TradeType
			var avgPrice float64
			if o.exchangeType == ExchangeTypeFuture {
				orderType = o.getTradeTypeByFloat(order["type"].(float64))
				avgPrice = order["price_avg"].(float64)
			} else if o.exchangeType == ExchangeTypeSpot {
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
				Status:     o.getStatus(order["status"].(float64)),
				DealAmount: order["deal_amount"].(float64),
				AvgPrice:   avgPrice,
			}
			result[i] = item
		}

		return result
	case <-time.After(DefaultTimeoutSec * time.Second):
		log.Printf("Timeout to get user info")
		return nil
	}
}

func (o *OKExAPI) GetBalance() map[string]interface{} {

	var channel string
	var data, parameters map[string]string

	if o.exchangeType == ExchangeTypeFuture {
		channel = ChannelUserInfo

	} else if o.exchangeType == ExchangeTypeSpot {
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
		if o.exchangeType == ExchangeTypeFuture {
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
				for coin, value := range balances {
					var bond float64
					balance := value.(map[string]interface{})["balance"]
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

		} else if o.exchangeType == ExchangeTypeSpot {
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
		return nil
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

func (o *OKExAPI) getTradeTypeByString(orderType string) TradeType {
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

func (o *OKExAPI) getTradeTypeByFloat(orderType float64) TradeType {
	switch orderType {
	case 1:
		return TradeTypeOpenLong
	case 2:
		return TradeTypeOpenShort
	case 3:
		return TradeTypeCloseLong
	case 4:
		return TradeTypeCloseShort
	}

	return TradeTypeUnknown
}

func (o *OKExAPI) getTradeTypeString(orderType TradeType) string {

	switch orderType {
	case TradeTypeOpenLong:
		return "1"
	case TradeTypeOpenShort:
		return "2"
	case TradeTypeCloseLong:
		return "3"
	case TradeTypeCloseShort:
		return "4"
	case TradeTypeBuy:
		return "buy"
	case TradeTypeSell:
		return "sell"
	}

	Logger.Errorf("[%s]getTradeType: Invalid type", o.GetExchangeName())
	return ""
}
