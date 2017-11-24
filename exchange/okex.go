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
}

func (o *OKExAPI) WatchEvent() chan EventType {
	return o.event
}

func (o *OKExAPI) triggerEvent(event EventType){
	o.event <- event
}

func (o *OKExAPI)Init(tradeType TradeType){

	o.tickerList = nil
	o.depthList = nil
	o.event = make(chan EventType)
	o.tradeType = tradeType

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

			if response[0]["channel"].(string) == EventAddChannel {

			}else{

				// 处理期货价格深度
				if o.depthList != nil {
					data := response[0]["data"].(map[string]interface{})
					for i, item := range o.depthList {
						if item.Name == response[0]["channel"] {

							o.depthList[i].Asks = data["asks"].([]interface{})
							o.depthList[i].Bids = data["bids"].([]interface{})
							unitTime := time.Unix(int64(data["timestamp"].(float64))/1000, 0)
							timeHM := unitTime.Format("2006-01-02 03:04:05 PM")
							o.depthList[i].Time = timeHM
							log.Printf("Result:%s", o.depthList[i])

							goto END
						}
					}
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

				unitTime := time.Unix(int64(tmp["timestamp"].(float64))/1000, 0)
				timeHM := unitTime.Format("2006-01-02 03:04:05 PM")

				tickerValue := &TickerValue {
					Last: lastValue,
					Time: timeHM,
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
func (o *OKExAPI) GetContractDepth(coin string, period string, depth string) {

	channel := strings.Replace(ChannelContractDepth, "X", coin, 1)
	channel = strings.Replace(channel, "Y", period, 1)
	channel = strings.Replace(channel, "Z", depth, 1)

	depthItem := DepthListItem {
		Name: channel,
	}
	o.depthList = append(o.depthList, depthItem)

	data := map[string]string {
		"event": EventAddChannel,
		"channel": channel,
	}

	o.command(data,nil)	
}

/*
X值为：ltc_btc eth_btc etc_btc bch_btc btc_usdt eth_usdt 
ltc_usdt etc_usdt bch_usdt etc_eth bt1_btc bt2_btc btg_btc 
qtum_btc hsr_btc neo_btc gas_btc qtum_usdt hsr_usdt neo_usdt gas_usdt
Y值为: 5, 10, 20(获取深度条数)
*/
func (o *OKExAPI) GetCurrentDepth(pair string, depth string) {
	channel := strings.Replace(ChannelCurrentDepth, "X", pair, 1)
	channel = strings.Replace(channel, "Y", depth, 1)

	depthItem := DepthListItem {
		Name: channel,
	}
	o.depthList = append(o.depthList, depthItem)

	data := map[string]string {
		"event": EventAddChannel,
		"channel": channel,
	}

	o.command(data,nil)	
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