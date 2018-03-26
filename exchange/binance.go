package exchange

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	Websocket "github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

const EndPoint = "wss://stream.binance.com:9443/ws/"
const BinanceURL = "https://api.binance.com"
const NameBinance = "Binance"

type Binance struct {
	websocket *Websocket.Conn
	event     chan EventType
	config    Config
}

func (p *Binance) GetExchangeName() string {
	return NameBinance
}

// SetConfigure()
func (p *Binance) SetConfigure(config Config) {
	p.config = config

	if p.config.Proxy != "" {
		logger.Infof("使用代理:%s", p.config.Proxy)
	}
}

// WatchEvent() return a channel which notified the application of the event triggered by exchange
func (p *Binance) WatchEvent() chan EventType {
	return p.event
}

func (h *Binance) Start() error {
	return nil
}

func (p *Binance) marketRequest(path string, params map[string]string) (error, []byte) {
	var req http.Request
	req.ParseForm()
	for k, v := range params {
		req.Form.Add(k, v)
	}
	bodystr := strings.TrimSpace(req.Form.Encode())
	// logger.Debugf("Params:%v", bodystr)
	request, err := http.NewRequest("GET", BinanceURL+path+"?"+bodystr, nil)
	if err != nil {
		return err, nil
	}

	// setup a http client
	httpTransport := &http.Transport{}
	httpClient := &http.Client{Transport: httpTransport}

	if p.config.Proxy != "" {
		values := strings.Split(p.config.Proxy, ":")
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
	// log.Printf("Body:%v", string(body))
	// var value map[string]interface{}
	// if err = json.Unmarshal(body, &value); err != nil {
	// 	return err, nil
	// }

	return nil, body

}

// Close() close the connection to the exchange and other handles
func (p *Binance) Close() {

}

// StartTicker() send message to the exchange to start the ticker of the given pairs
func (p *Binance) StartTicker(pair string) {
}

// GetTicker(), better to use the ITicker to notify the ticker information
func (p *Binance) GetTicker(pair string) *TickerValue {
	return nil
}

// GetDepthValue() get the depth of the assigned price area and quantity
// GetDepthValue(pair string, price float64, limit float64, orderQuantity float64, tradeType TradeType) []DepthPrice
func (p *Binance) GetDepthValue(pair string) [][]DepthPrice {
	//ethusdt@depth20
	coins := ParsePair(pair)
	symbol := strings.ToUpper(coins[0] + coins[1])
	if err, response := p.marketRequest("/api/v1/depth", map[string]string{
		"symbol": symbol,
		// "limit":  "100",
	}); err != nil {
		logger.Errorf("无效深度:%v", err)
		return nil
	} else {

		var value map[string]interface{}
		if err = json.Unmarshal(response, &value); err != nil {
			logger.Errorf("解析错误:%v", err)
			return nil
		}

		if value["code"] == nil {
			list := make([][]DepthPrice, 2)

			asks := value["asks"].([]interface{})
			bids := value["bids"].([]interface{})

			if asks != nil && len(asks) > 0 {
				askList := make([]DepthPrice, len(asks))
				for i, ask := range asks {
					values := ask.([]interface{})
					askList[i].Price, _ = strconv.ParseFloat(values[0].(string), 64)
					askList[i].Quantity, _ = strconv.ParseFloat(values[1].(string), 64)
				}

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

			return list
		}
	}

	return nil
}

// GetBalance() get the balances of all the coins
func (p *Binance) GetBalance() map[string]interface{} {
	return map[string]interface{}{
		"helo": "wolrd",
	}
}

// Trade() trade as the configs
func (p *Binance) Trade(configs TradeConfig) *TradeResult {
	return nil
}

// CancelOrder() cancel the order as the order information
func (p *Binance) CancelOrder(order OrderInfo) *TradeResult {
	return nil
}

// GetOrderInfo() get the information with order filter
func (p *Binance) GetOrderInfo(filter OrderInfo) []OrderInfo {
	return nil
}

func (p *Binance) GetKline(pair string, period string, limit int) []KlineValue {
	coins := ParsePair(pair)
	symbol := strings.ToUpper(coins[0] + coins[1])
	if err, response := p.marketRequest("/api/v1/klines", map[string]string{
		"symbol":   symbol,
		"interval": period,
		"limit":    strconv.Itoa(limit),
	}); err != nil {
		logger.Errorf("无效数据:%v", err)
		return nil
	} else {
		var values [][]interface{}
		if response != nil {
			if err = json.Unmarshal(response, &values); err != nil {
				logger.Errorf("Fail to Unmarshal:%v", err)
				return nil
			}

			kline := make([]KlineValue, len(values))
			for i, value := range values {
				// kline[i].OpenTime = time.Unix((int64)(value[0].(float64)/1000), 0).Format(Global.TimeFormat)
				kline[i].OpenTime = value[0].(float64) / 1000
				kline[i].Open, _ = strconv.ParseFloat(value[1].(string), 64)
				kline[i].High, _ = strconv.ParseFloat(value[2].(string), 64)
				kline[i].Low, _ = strconv.ParseFloat(value[3].(string), 64)
				kline[i].Close, _ = strconv.ParseFloat(value[4].(string), 64)
				kline[i].Volumn, _ = strconv.ParseFloat(value[5].(string), 64)
				// kline[i].CloseTime = time.Unix((int64)(value[6].(float64)/1000), 0).Format(Global.TimeFormat)
				kline[i].CloseTime = value[6].(float64) / 1000

			}

			return kline
		}

		return nil
	}
}
