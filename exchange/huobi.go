package exchange

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	Websocket "github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

const HuobiMarketUrl = "https://api.huobi.pro/market"
const HuobiTradeUrl = "https://api.huobi.pro/v1"

const HuobiWebsocketSpot = "wss://api.huobi.pro/ws"
const HuobiWebsocketFuture = "wss://www.hbdm.com/ws"
const HuobiTimestamp = "ts"

const HuobiExchangeName = "Huobi"

const TickerDelaySecond = 1

type Huobi struct {
	event chan EventType

	InstrumentType InstrumentType
	Proxy          string
	ApiKey         string
	SecretKey      string
	Passphare      string

	conn   *Websocket.Conn
	klines map[string][]KlineValue

	depthValues map[string]*sync.Map
}

func (p *Huobi) GetExchangeName() string {
	return HuobiExchangeName
}

// SetConfigure()
func (p *Huobi) SetConfigure(config Config) {

}

// WatchEvent() return a channel which notified the application of the event triggered by exchange
func (p *Huobi) WatchEvent() chan EventType {
	return p.event
}

// Start() prepare the connection to the exchange
func (p *Huobi) Start() error {
	go func() {
		for {
			select {
			case <-time.After(TickerDelaySecond * time.Second):
			}
		}
	}()

	return nil
}

func (p *Huobi) ping(value uint64) error {
	pongMessage := map[string]uint64{
		"pong": value,
	}
	message, _ := json.Marshal(pongMessage)
	// logger.Debugf("%v Pong:%v", time.Now(), message)
	return p.conn.WriteMessage(Websocket.TextMessage, message)
}

func (p *Huobi) Start2(errChan chan EventType) error {

	p.klines = make(map[string][]KlineValue)
	// force to restart the command
	p.depthValues = make(map[string]*sync.Map)

	dialer := Websocket.DefaultDialer

	if p.Proxy != "" {
		logger.Infof("Proxy:%s", p.Proxy)
		values := strings.Split(p.Proxy, ":")
		if values[0] == "SOCKS5" {
			proxy, err := proxy.SOCKS5("tcp", values[1]+":"+values[2], nil, proxy.Direct)
			if err != nil {
				return err
			}

			dialer = &Websocket.Dialer{NetDial: proxy.Dial}
		}

	}

	var path string
	if p.InstrumentType == InstrumentTypeSpot {
		path = HuobiWebsocketSpot
	} else {
		path = HuobiWebsocketFuture
	}

	connection, _, err := dialer.Dial(path, nil)
	if err != nil {
		logger.Errorf("Fail to dial:%v", err)
		// go o.triggerEvent(EventLostConnection)
		errChan <- EventLostConnection
		return err
	}

	go func() {
		// counter := 0

		for {
			_, message, err := connection.ReadMessage()
			if err != nil {
				connection.Close()
				logger.Errorf("Fail to read:%v", err)
				// go o.triggerEvent(EventLostConnection)
				errChan <- EventLostConnection
				return
			}

			r, err := gzip.NewReader(bytes.NewReader(message))
			if err != nil {
				logger.Errorf("Fail to create reader:%s\n", err)
				return
			}

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
					"ping",
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

			if strings.Contains(string(message), "ping") {
				var pingMessage map[string]interface{}
				if err := json.Unmarshal(message, &pingMessage); err != nil {
					logger.Errorf("Invalid ping message:%v", err)
					continue
				}

				if pingMessage != nil && pingMessage["ping"] != nil {
					p.ping(uint64(pingMessage["ping"].(float64)))
					continue
				}
			}

			var response map[string]interface{}
			if err = json.Unmarshal([]byte(message), &response); err != nil {
				logger.Errorf("Fail to Unmarshal:%v", err)
				continue
			}

			if response["subbed"] != nil {
				if response["status"] != nil && response["status"].(string) == "ok" {
					logger.Infof("Subjected:%v", response["subbed"])
				} else {
					logger.Infof("Failed to subject:%v", response["subbed"])
				}
				continue
			}

			if response["ch"] != nil {
				channel := response["ch"].(string)
				timestamp := response["ts"].(float64)
				data := response["tick"].(map[string]interface{})
				if p.depthValues[channel] != nil {
					asks := data["asks"].([]interface{})
					bids := data["bids"].([]interface{})

					list := make([][]DepthPrice, 2)
					// log.Printf("Cr32:%x Result:%x", o.getCRC32Value(input), uint32(data["checksum"].(float64)))
					if asks != nil && len(asks) > 0 {
						askList := make([]DepthPrice, len(asks))
						for i, ask := range asks {
							values := ask.([]interface{})
							askList[i].Price = values[0].(float64)
							askList[i].Quantity = values[1].(float64)
						}

						// list[DepthTypeAsks] = revertDepthArray(askList)
						list[DepthTypeAsks] = askList
					}

					if bids != nil && len(bids) > 0 {
						bidList := make([]DepthPrice, len(bids))
						for i, bid := range bids {
							values := bid.([]interface{})
							bidList[i].Price = values[0].(float64)
							bidList[i].Quantity = values[1].(float64)
						}

						list[DepthTypeBids] = bidList
					}

					p.depthValues[channel].Store("data", list)
					p.depthValues[channel].Store(HuobiTimestamp, timestamp)
				}
			}
		}
	}()

	p.conn = connection

	errChan <- EventConnected

	return nil
}

func (p *Huobi) marketRequest(path string, params map[string]string) (error, map[string]interface{}) {

	var req http.Request
	req.ParseForm()
	for k, v := range params {
		req.Form.Add(k, v)
	}
	bodystr := strings.TrimSpace(req.Form.Encode())
	logger.Debugf("Params:%v", bodystr)
	request, err := http.NewRequest("GET", HuobiMarketUrl+path, strings.NewReader(bodystr))
	if err != nil {
		return err, nil
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36")

	var resp *http.Response
	resp, err = http.DefaultClient.Do(request)
	if err != nil {
		return err, nil
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err, nil
	}

	var value map[string]interface{}
	if err = json.Unmarshal(body, &value); err != nil {
		return err, nil
	}

	return nil, value

}

// Close() close the connection to the exchange and other handles
func (p *Huobi) Close() {

}

// StartTicker() send message to the exchange to start the ticker of the given pairs
func (p *Huobi) StartTicker(pair string) {
}

// GetTicker(), better to use the ITicker to notify the ticker information
func (p *Huobi) GetTicker(pair string) *TickerValue {
	return nil
}

func (p *Huobi) StartDepth(subject string) {

	data := map[string]interface{}{
		"sub": subject,
		"id":  "madaoQT",
	}

	p.command(data)

}

// GetDepthValue() get the depth of the assigned price area and quantity
// GetDepthValue(pair string, price float64, limit float64, orderQuantity float64, tradeType TradeType) []DepthPrice
func (p *Huobi) GetDepthValue(pair string) [][]DepthPrice {

	var channel string
	coins := ParsePair(pair)

	if p.InstrumentType == InstrumentTypeSwap {
		// channel = o.StartDepth()
		channel = "market." + coins[0] + coins[1] + ".depth.step0"
		if p.depthValues[channel] == nil {
			p.depthValues[channel] = new(sync.Map)
			p.StartDepth(channel)
		}
	} else if p.InstrumentType == InstrumentTypeSpot {
		channel = "market." + coins[0] + coins[1] + ".depth.step0"
		if p.depthValues[channel] == nil {
			p.depthValues[channel] = new(sync.Map)
			p.StartDepth(channel)
		}
	}

	if p.depthValues[channel] != nil {
		now := time.Now()
		if timestamp, ok := p.depthValues[channel].Load(HuobiTimestamp); ok {
			// location, _ := time.LoadLocation("Asia/Shanghai")
			updateTime := time.Unix(int64(timestamp.(float64))/1000, 0)
			// logger.Infof("Now:%v Update:%v", now.String(), updateTime.In(location).String())
			if updateTime.Add(10 * time.Second).Before(now) {
				logger.Error("Invalid timestamp")
				return nil
			}
		}

		if recv, ok := p.depthValues[channel].Load("data"); ok {
			list := recv.([][]DepthPrice)
			return list
		}
	}

	return nil
}

func (p *Huobi) command(data map[string]interface{}) error {
	if p.conn == nil {
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

	p.conn.WriteMessage(Websocket.TextMessage, cmd)

	return nil
}

// GetBalance() get the balances of all the coins
func (p *Huobi) GetBalance() map[string]interface{} {
	return map[string]interface{}{
		"helo": "wolrd",
	}
}

// Trade() trade as the configs
func (p *Huobi) Trade(configs TradeConfig) *TradeResult {
	return nil
}

// CancelOrder() cancel the order as the order information
func (p *Huobi) CancelOrder(order OrderInfo) *TradeResult {
	return nil
}

// GetOrderInfo() get the information with order filter
func (p *Huobi) GetOrderInfo(filter OrderInfo) []OrderInfo {
	return nil
}

func (p *Huobi) GetKline(pair string, period int, limit int) []KlineValue {
	return nil
}
