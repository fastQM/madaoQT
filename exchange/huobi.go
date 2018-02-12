package exchange

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const HuobiMarketUrl = "https://api.huobi.pro/market"
const HuobiTradeUrl = "https://api.huobi.pro/v1"

const HuobiExchangeName = "Huobi"

const TickerDelaySecond = 1

type Huobi struct {
	event chan EventType
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

// GetDepthValue() get the depth of the assigned price area and quantity
// GetDepthValue(pair string, price float64, limit float64, orderQuantity float64, tradeType TradeType) []DepthPrice
func (p *Huobi) GetDepthValue(pair string) [][]DepthPrice {
	coins := ParsePair(pair)
	if err, response := p.marketRequest("/depth", map[string]string{
		"symbol": coins[0] + coins[1],
		"type":   "step0",
	}); err != nil {
		logger.Errorf("无效深度:%v", err)
		return nil
	} else {

		if response["status"].(string) == "ok" {
			list := make([][]DepthPrice, 2)
			data := response["tick"].(map[string]interface{})
			asks := data["asks"].([]interface{})
			bids := data["bids"].([]interface{})

			if asks != nil && len(asks) > 0 {
				askList := make([]DepthPrice, len(asks))
				for i, ask := range asks {
					values := ask.([]interface{})
					askList[i].Price = values[0].(float64)
					askList[i].Quantity = values[1].(float64)
				}

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

			return list
		}
	}

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
