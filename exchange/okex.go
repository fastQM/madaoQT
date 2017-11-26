package exchange


import (
	"log"
	"crypto/md5"
	"errors"
	"sort"
	"fmt"
	"encoding/json"
	"strings"
	"time"
	"strconv"

	websocket "github.com/gorilla/websocket"
)

const contractUrl = "wss://real.okex.com:10440/websocket/okexapi"
const currentUrl = "wss://real.okex.com:10441/websocket"

const constApiKey = "a982120e-8505-41db-9ae3-0c62dd27435c"
const constSecretKey = "71430C7FA63A067724FB622FB3031970"

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
const ChannelCancelOrder = "ok_futureusd_cancel_order"
const ChannelUserInfo = "ok_futureusd_userinfo"
const ChannelOrderInfo = "ok_futureusd_orderinfo"
const ChannelTrades = "ok_sub_futureusd_trades"
const ChannelSubUserInfo = "ok_sub_futureusd_userinfo"
const ChannelSubPositions = "ok_sub_futureusd_positions"

// 现货行情API
const ChannelCurrentChannelTicker = "ok_sub_spot_X_ticker"
const ChannelCurrentDepth = "ok_sub_spot_X_depth_Y"

type ContractItemValueIndex int8

const (
	UsdPriceIndex ContractItemValueIndex = iota
	ContractQuantity
	CoinQuantity
	TotalCoinQuantity
	TotalContractQuantity
)

type OKExAPI struct{
	conn *websocket.Conn
	tickerList []TickerListItem
	depthList []DepthListItem
	event chan EventType
	tradeType TradeType

	/* Each channel has a depth */
	messageChannels map[string]chan interface{}

	/* 建仓数量 */
	qty float64
}

func formatTimeOKEX() string {
	timeFormat := "2006-01-02 06:04:05"
	location,_ := time.LoadLocation("Local")
	// unixTime := time.Unix(timestamp/1000, 0)
	unixTime := time.Now()
	return unixTime.In(location).Format(timeFormat)
}

func (o *OKExAPI) WatchEvent() chan EventType {
	return o.event
}

func (o *OKExAPI) triggerEvent(event EventType){
	o.event <- event
}


func (o *OKExAPI) SetQty(quantity float64) {
	o.qty = quantity
}

func (o *OKExAPI) GetQty() float64 {
	return o.qty;
}

func (o *OKExAPI)Init(tradeType TradeType){

	o.tickerList = nil
	o.depthList = nil
	o.event = make(chan EventType)
	o.tradeType = tradeType

	o.messageChannels = make(map[string]chan interface{})

	o.qty = 100

	var url string
	if tradeType == TradeTypeContract{
		url = contractUrl
	} else if tradeType == TradeTypeCurrent {
		url = currentUrl
	}

	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Printf("Fail to dial: %v", err)
		go o.triggerEvent(EventError)
		return
	}

	go func(){
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Printf("read:%v", err)
				go o.triggerEvent(EventError)
				return
			}

			// log.Printf("recv: %s", message)

			var response []map[string]interface{}
			if err = json.Unmarshal([]byte(message), &response); err != nil {
				log.Println("Unmarshal:", err)
				continue
			}

			if response[0]["channel"].(string) == EventAddChannel || response[0]["channel"].(string) == EventRemoveChannel  {

			}else if response[0]["channel"].(string) == ChannelUserInfo {
				if o.messageChannels[ChannelUserInfo] != nil {
					data := response[0]["data"].(map[string]interface{})
					if data != nil && data["result"] == true {
						info := data["info"]
						go func() {
							o.messageChannels[ChannelUserInfo] <- info
							close(o.messageChannels[ChannelUserInfo])
							delete(o.messageChannels, ChannelUserInfo)
						}()
					}
				}
			} else if response[0]["channel"].(string) == ChannelTrades {
				log.Printf("trades: %v", response[0]["data"])
				go func() {
					o.messageChannels[ChannelTrades] <- response[0]["data"]
					close(o.messageChannels[ChannelTrades])
					delete(o.messageChannels, ChannelTrades)
				}()
			} else{

				// 处理期货价格深度
				if o.messageChannels[response[0]["channel"].(string)] != nil {

					depth := new(DepthValue)
					data := response[0]["data"].(map[string]interface{})

					// unitTime := time.Unix(int64(data["timestamp"].(float64))/1000, 0)
					// timeHM := unitTime.Format("2006-01-02 03:04:05")
					// o.depthList[i].Time = timeHM
					if data["asks"] == nil || data["bids"] == nil {
						log.Printf("Invalid data")
						goto END
					}

					depth.Time = formatTimeOKEX()

					asks := data["asks"].([]interface{})
					bids := data["bids"].([]interface{})

					if o.tradeType == TradeTypeContract {
						if asks != nil && len(asks) > 0 {
							askList := make([]DepthPrice, len(asks))
							for i, ask := range asks {
								values := ask.([]interface{})
								askList[i].price, _ = strconv.ParseFloat(values[UsdPriceIndex].(string), 64)
								askList[i].qty, _ = strconv.ParseFloat(values[CoinQuantity].(string), 64)
							}

							depth.AskAverage, depth.AskQty = GetDepthAveragePrice(askList)
							depth.AskByOrder = GetDepthPriceByOrder(DepthTypeAsks, askList, o.qty)
						}

						if bids != nil && len(bids) > 0 {
							bidList := make([]DepthPrice, len(bids))
							for i, bid := range bids {
								values := bid.([]interface{})
								bidList[i].price, _ = strconv.ParseFloat(values[UsdPriceIndex].(string), 64)
								bidList[i].qty, _ = strconv.ParseFloat(values[CoinQuantity].(string), 64)
							}

							depth.BidAverage, depth.BidQty = GetDepthAveragePrice(bidList)
							depth.BidByOrder = GetDepthPriceByOrder(DepthTypeBids, bidList, o.qty)
						}

					} else if o.tradeType == TradeTypeCurrent {
						if asks != nil && len(asks) > 0 {
							askList := make([]DepthPrice, len(asks))
							for i, ask := range asks {
								values := ask.([]interface{})
								askList[i].price, _ = strconv.ParseFloat(values[0].(string), 64)
								askList[i].qty, _ = strconv.ParseFloat(values[1].(string), 64)
							}

							depth.AskAverage, depth.AskQty = GetDepthAveragePrice(askList)
							depth.AskByOrder = GetDepthPriceByOrder(DepthTypeAsks, askList, o.qty)
						}

						if bids != nil && len(bids) > 0 {
							bidList := make([]DepthPrice, len(bids))
							for i, bid := range bids {
								values := bid.([]interface{})
								bidList[i].price, _ = strconv.ParseFloat(values[0].(string), 64)
								bidList[i].qty, _ = strconv.ParseFloat(values[1].(string), 64)
							}

							depth.BidAverage, depth.BidQty = GetDepthAveragePrice(bidList)
							depth.BidByOrder = GetDepthPriceByOrder(DepthTypeBids, bidList, o.qty)
						}
					}

					log.Printf("Result:%v", depth)

					go func(){
						o.messageChannels[response[0]["channel"].(string)] <- depth
						close(o.messageChannels[response[0]["channel"].(string)])
						delete(o.messageChannels, response[0]["channel"].(string))
					}()

					goto END

				}

				// 处理现货价格
				if o.tickerList != nil {
					for i, ticker := range o.tickerList {
						if ticker.Name == response[0]["channel"] {
							// o.tickerList[i].Time = timeHM
							o.tickerList[i].Value = response[0]["data"]
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

func (o *OKExAPI)Close(){
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
		Tag: tag,
		Name: channel,
	}

	o.tickerList = append(o.tickerList, ticker)

	data := map[string]string{
		"event": "addChannel",
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
		Tag: tag,
		Name: channel,
	}

	o.tickerList = append(o.tickerList, ticker)

	data := map[string]string{
		"event": "addChannel",
		"channel": channel,
	}

	o.command(data, nil)	
}

func (o *OKExAPI) GetExchangeName() string {
	return "OKEX";
}

func (o *OKExAPI)GetTickerValue(tag string) *TickerValue {
	for _, ticker := range o.tickerList {
		if ticker.Tag == tag {
			if ticker.Value != nil {
				// return ticker.Value.(map[string]interface{})
				var lastValue float64
				tmp := ticker.Value.(map[string]interface{})
				if o.tradeType == TradeTypeContract {
					lastValue = tmp["last"].(float64)
				} else if o.tradeType == TradeTypeCurrent {
					value, _ := strconv.ParseFloat(tmp["last"].(string), 64)
					lastValue = value
				}

				// unitTime := time.Unix(int64(tmp["timestamp"].(float64))/1000, 0)
				// timeHM := unitTime.Format("2006-01-02 06:04:05")

				tickerValue := &TickerValue {
					Last: lastValue,
					Time: formatTimeOKEX(),
				}
				
				return tickerValue
			}
		}
	}

	return nil
}

func (o *OKExAPI)ping() {
	data := map[string]string{
		"event":"ping",
	}

	o.command(data,nil)
}

func (o *OKExAPI) Login() {
	data := map[string]string{
		"event":"login",
	}

	parameters := map[string]string {
		"api_key": constApiKey,
		"secret_key": constSecretKey,
	}
	o.command(data,parameters)	
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
		o.messageChannels[channel] = make(chan interface{})

	} else {
		event = EventRemoveChannel
		delete(o.messageChannels, channel)
	}

	data := map[string]string {
		"event": event,
		"channel": channel,
	}

	o.command(data,nil)	

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
		o.messageChannels[channel] = make(chan interface{})
	} else {
		event = EventRemoveChannel
		delete(o.messageChannels, channel)
	}

	data := map[string]string {
		"event": event,
		"channel": channel,
	}

	o.command(data,nil)	
	return channel
	
}

func (o *OKExAPI)GetDepthValue(coinA string, coinB string) *DepthValue {

	var channel string

	if o.tradeType == TradeTypeContract {
		channel = o.SwithContractDepth(true, coinA, "this_week", "20")
		// defer o.SwithContractDepth(false, coinA, "this_week", "20")
	} else if o.tradeType == TradeTypeCurrent {
		channel = o.SwitchCurrentDepth(true, coinA, coinB, "20")
		// defer o.SwitchCurrentDepth(false, coinA, coinB, "20")
	}

	select{
	case <-time.After(1*time.Second):
		log.Print("timeout to wait for the depths")
		return nil	
	case value := <- o.messageChannels[channel]:
		return value.(*DepthValue)
	}
}	

func (o *OKExAPI)command(data map[string]string, parameters map[string]string) error{
	if o.conn == nil {
		return errors.New("Connection is lost")
	}

	command := make(map[string]interface{})
	for k, v := range data{
		command[k] = v
	}

	if parameters != nil {
		var keys []string
		var signPlain string

		for k, _ := range parameters {
			keys = append(keys, k)
		}

		sort.Strings(keys)
		
		for i, key := range keys {
			if key == "sign"{
				continue
			}
			signPlain += (key + "=" + parameters[key])
			if i != (len(keys)-1) {
				signPlain += "&"
			}
		}

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
	
	log.Printf("Cmd:%v", string(cmd))
	o.conn.WriteMessage(websocket.TextMessage, cmd)

	return nil
}


/* 合约交易接口 */

func (o *OKExAPI) Trade() {

}

func (o *OKExAPI) CancelTrade() {

}

func (o *OKExAPI) GetUserInfo() map[string]interface{} {
	data := map[string]string{
		"event":EventAddChannel,
		"channel": ChannelUserInfo,
	}

	parameters := map[string]string {
		"api_key": constApiKey,
		"secret_key": constSecretKey,
	}

	o.messageChannels[ChannelUserInfo] = make(chan interface{})	

	o.command(data,parameters)

	select{
	case <- time.After(1 * time.Second):
		log.Printf("Timeout to get user account info")
		return nil
	case msg := <- o.messageChannels[ChannelUserInfo]:
		return msg.(map[string]interface{})
	}
}


// 没有响应
func (o *OKExAPI) GetTradesInfo() map[string]interface{} {
	data := map[string]string{
		"event":EventAddChannel,
		"channel": ChannelTrades,
	}

	parameters := map[string]string {
		"api_key": constApiKey,
		"secret_key": constSecretKey,
	}

	o.messageChannels[ChannelTrades] = make(chan interface{})	

	o.command(data,parameters)

	select{
	case <- time.After(5 * time.Second):
		log.Printf("Timeout to get user trades info")
		return nil
	case msg := <- o.messageChannels[ChannelUserInfo]:
		return msg.(map[string]interface{})
	}
}
