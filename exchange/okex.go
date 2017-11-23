package exchange


import (
	"log"
	"os"
	"os/signal"
	"crypto/md5"
	"errors"
	"sort"
	"fmt"
	"encoding/json"
	"strings"

	websocket "github.com/gorilla/websocket"
)

const contractUrl = "wss://real.okex.com:10440/websocket/okexapi"
const currentUrl = "wss://real.okex.com:10441/websocket"

const constApiKey = "a982120e-8505-41db-9ae3-0c62dd27435c"
const constSecretKey = "71430C7FA63A067724FB622FB3031970"

const tradeTypeContract = 0
const tradeTypeCurrent = 1

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
const ChannelTicker = "ok_sub_futureusd_X_ticker_Y"


const ChannelLogin = "login"
const ChannelTrade = "ok_futureusd_trade"
const ChannelCancelOrder = "ok_futureusd_cancel_order"
const ChannelUserInfo = "ok_futureusd_userinfo"
const ChannelOrderInfo = "ok_futureusd_orderinfo"
const ChannelTrades = "ok_sub_futureusd_trades"
const ChannelSubUserInfo = "ok_sub_futureusd_userinfo"
const ChannelSubPositions = "ok_sub_futureusd_positions"

// 现货行情API
const CurrentChannelTicker = "ok_sub_spot_X_ticker"

const EventConnected = 0
const EventError = 1

type OKExAPI struct{
	conn *websocket.Conn
	tickerList []tickerValue
	event chan int
}

func (o *OKExAPI) WatchEvent() chan int {
	return o.event
}

func (o *OKExAPI) triggerEvent(event int){
	o.event <- event
}

func (o *OKExAPI)Init(tradeType int){

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	o.tickerList = nil
	o.event = make(chan int)

	var url string
	if tradeType == tradeTypeContract{
		url = contractUrl
	} else if tradeType == tradeTypeCurrent {
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
			var records []map[string]interface{}
			if err = json.Unmarshal([]byte(message), &records); err != nil {
				log.Println("Unmarshal:", err)
				continue
			}

			if records[0]["channel"].(string) == EventAddChannel {

			}else{
				if o.tickerList != nil {
					for i, ticker := range o.tickerList {
						if ticker.Name == records[0]["channel"] {
							o.tickerList[i].Value = records[0]["data"]
							break
						}
					}
				}
			}

			// record := records[0]["data"].(map[string]interface{})
			// log.Printf("record: %v", record)
			// if record["timestamp"] != nil {
			// 	unitTime := time.Unix(int64(record["timestamp"].(float64))/1000, 0)
			// 	timeHM := unitTime.Format("2006-01-02 03:04:05 PM")
			// 	log.Printf("recv: %v", timeHM)
			// }
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

func (o *OKExAPI) StartContractTicker(coin string, period string, tag string) {
	channel := strings.Replace(ChannelTicker, "X", coin, 1)
	channel = strings.Replace(channel, "Y", period, 1)

	ticker := tickerValue{
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

func (o *OKExAPI) StartCurrentTicker(coinA string, coinB string, tag string) {
	pair := (coinA + "_" + coinB)
	
	channel := strings.Replace(CurrentChannelTicker, "X", pair, 1)

	ticker := tickerValue{
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

func (o *OKExAPI)GetTickerValue(tag string) map[string]interface{} {
	for _, ticker := range o.tickerList {
		if ticker.Tag == tag {
			if ticker.Value != nil {
				return ticker.Value.(map[string]interface{})
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