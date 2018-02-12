package exchange

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	Websocket "github.com/gorilla/websocket"
)

const EndPoint = "wss://stream.binance.com:9443/ws/"
const BinanceURL = "https://api.binance.com"
const BinanceExchange = "Binance"

type Binance struct {
	websocket *Websocket.Conn
	event     chan EventType
}

func (p *Binance) GetExchangeName() string {
	return HuobiExchangeName
}

// SetConfigure()
func (p *Binance) SetConfigure(config Config) {

}

// WatchEvent() return a channel which notified the application of the event triggered by exchange
func (p *Binance) WatchEvent() chan EventType {
	return p.event
}

func (h *Binance) Start() error {
	return nil
}

func (h *Binance) marketRequest(path string, params map[string]string) (error, map[string]interface{}) {
	var req http.Request
	req.ParseForm()
	for k, v := range params {
		req.Form.Add(k, v)
	}
	bodystr := strings.TrimSpace(req.Form.Encode())
	logger.Debugf("Params:%v", bodystr)
	request, err := http.NewRequest("GET", BinanceURL+path+"?"+bodystr, nil)
	if err != nil {
		return err, nil
	}

	// request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36")

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
	// log.Printf("Body:%v", string(body))
	var value map[string]interface{}
	if err = json.Unmarshal(body, &value); err != nil {
		return err, nil
	}

	return nil, value

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
	if err, response := p.marketRequest("/api/v1/depth", map[string]string{
		"symbol": strings.ToUpper(coins[0] + coins[1]),
		// "limit":  "100",
	}); err != nil {
		logger.Errorf("无效深度:%v", err)
		return nil
	} else {

		if response["code"] == nil {
			list := make([][]DepthPrice, 2)

			asks := response["asks"].([]interface{})
			bids := response["bids"].([]interface{})

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
