package exchange

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const BittrexMarketUrl = "https://bittrex.com/api/v1.1/public"
const BittrexTradeUrl = "https://api.huobi.pro/v1"

const BittrexExchangeName = "Bittrex"

type Bittrex struct {
	event chan EventType
}

func (p *Bittrex) GetExchangeName() string {
	return BittrexExchangeName
}

// SetConfigure()
func (p *Bittrex) SetConfigure(config Config) {

}

// WatchEvent() return a channel which notified the application of the event triggered by exchange
func (p *Bittrex) WatchEvent() chan EventType {
	return p.event
}

// Start() prepare the connection to the exchange
func (p *Bittrex) Start() error {
	go func() {
		for {
			select {
			case <-time.After(TickerDelaySecond * time.Second):
			}
		}
	}()

	return nil
}

func (p *Bittrex) marketRequest(path string, params map[string]string) (error, map[string]interface{}) {

	var req http.Request
	req.ParseForm()
	for k, v := range params {
		req.Form.Add(k, v)
	}
	bodystr := strings.TrimSpace(req.Form.Encode())
	logger.Debugf("Params:%v", bodystr)
	request, err := http.NewRequest("GET", BittrexMarketUrl+path, strings.NewReader(bodystr))
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

	var value map[string]interface{}
	if err = json.Unmarshal(body, &value); err != nil {
		return err, nil
	}

	return nil, value

}

// Close() close the connection to the exchange and other handles
func (p *Bittrex) Close() {

}

// StartTicker() send message to the exchange to start the ticker of the given pairs
func (p *Bittrex) StartTicker(pair string) {
}

// GetTicker(), better to use the ITicker to notify the ticker information
func (p *Bittrex) GetTicker(pair string) *TickerValue {
	return nil
}

// GetDepthValue() get the depth of the assigned price area and quantity
// GetDepthValue(pair string, price float64, limit float64, orderQuantity float64, tradeType TradeType) []DepthPrice
func (p *Bittrex) GetDepthValue(pair string) [][]DepthPrice {
	coins := ParsePair(pair)
	if err, response := p.marketRequest("/getorderbook", map[string]string{
		"market": strings.ToUpper(coins[1]) + "-" + strings.ToUpper(coins[0]),
		"type":   "both",
	}); err != nil {
		logger.Errorf("无效深度:%v", err)
		return nil
	} else {
		if response["success"].(bool) {
			list := make([][]DepthPrice, 2)
			data := response["result"].(map[string]interface{})
			if data["sell"] == nil || data["buy"] == nil {
				logger.Error("无效深度信息")
				return nil
			}
			asks := data["sell"].([]interface{})
			bids := data["buy"].([]interface{})

			if asks != nil && len(asks) > 0 {
				askList := make([]DepthPrice, len(asks))
				for i, ask := range asks {
					values := ask.(map[string]interface{})
					askList[i].Price = values["Rate"].(float64)
					askList[i].Quantity = values["Quantity"].(float64)
				}

				list[DepthTypeAsks] = askList
			}

			if bids != nil && len(bids) > 0 {
				bidList := make([]DepthPrice, len(bids))
				for i, bid := range bids {
					values := bid.(map[string]interface{})
					bidList[i].Price = values["Rate"].(float64)
					bidList[i].Quantity = values["Quantity"].(float64)
				}

				list[DepthTypeBids] = bidList
			}

			return list
		}
	}

	return nil
}

// GetBalance() get the balances of all the coins
func (p *Bittrex) GetBalance() map[string]interface{} {
	return map[string]interface{}{
		"helo": "wolrd",
	}
}

// Trade() trade as the configs
func (p *Bittrex) Trade(configs TradeConfig) *TradeResult {
	return nil
}

// CancelOrder() cancel the order as the order information
func (p *Bittrex) CancelOrder(order OrderInfo) *TradeResult {
	return nil
}

// GetOrderInfo() get the information with order filter
func (p *Bittrex) GetOrderInfo(filter OrderInfo) []OrderInfo {
	return nil
}

func (p *Bittrex) GetKline(pair string, period int, limit int) []KlineValue {
	return nil
}
