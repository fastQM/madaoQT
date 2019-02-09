package exchange

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const BittrexMarketUrl = "https://bittrex.com/api/v1.1"

const BittrexExchangeName = "Bittrex"

type Bittrex struct {
	event  chan EventType
	config Config
}

func (p *Bittrex) GetExchangeName() string {
	return BittrexExchangeName
}

// SetConfigure()
func (p *Bittrex) SetConfigure(config Config) {
	p.config = config

	if p.config.Proxy != "" {
		logger.Infof("Proxy:%s", p.config.Proxy)
	}
}

// WatchEvent() return a channel which notified the application of the event triggered by exchange
func (p *Bittrex) WatchEvent() chan EventType {
	return p.event
}

// Start() prepare the connection to the exchange
func (p *Bittrex) Start() error {
	return nil
}

func (p *Bittrex) marketRequest(path string, params map[string]string, authen bool) (error, map[string]interface{}) {

	var req http.Request
	req.ParseForm()
	for k, v := range params {
		req.Form.Add(k, v)
	}

	nonce := time.Now().Unix()
	req.Form.Add("nonce", fmt.Sprintf("%d", nonce))

	bodystr := strings.TrimSpace(req.Form.Encode())
	logger.Debugf("Params:%v", bodystr)
	path = BittrexMarketUrl + path + "?" + bodystr
	logger.Debugf("Path:%s", path)

	request, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return err, nil
	}

	if authen {
		h := hmac.New(sha512.New, []byte(p.config.Secret))
		io.WriteString(h, path)
		request.Header.Set("apisign", fmt.Sprintf("%x", h.Sum(nil)))
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

	log.Printf("body:%v", value)
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

func (p *Bittrex) parsePair(pair string) string {
	coins := ParsePair(pair)
	return strings.ToUpper(coins[1]) + "-" + strings.ToUpper(coins[0])
}

// GetDepthValue() get the depth of the assigned price area and quantity
// GetDepthValue(pair string, price float64, limit float64, orderQuantity float64, tradeType TradeType) []DepthPrice
func (p *Bittrex) GetDepthValue(pair string) [][]DepthPrice {

	if err, response := p.marketRequest("/public/getorderbook", map[string]string{
		"market": p.parsePair(pair),
		"type":   "both",
	}, false); err != nil {
		logger.Errorf("Fail to get Orderbook:%v", err)
		return nil
	} else {
		if response["success"].(bool) {
			list := make([][]DepthPrice, 2)
			data := response["result"].(map[string]interface{})
			if data["sell"] == nil || data["buy"] == nil {
				logger.Error("Invlaid orderbook")
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
	if err, response := p.marketRequest("/account/getbalances", map[string]string{
		"apikey": p.config.API,
	}, true); err != nil {
		logger.Errorf("Fail to get balance:%v", err)
		return nil
	} else {

		if response == nil || response["success"] != true {
			logger.Errorf("Fail to get the balances:%v", response["message"])
			return nil
		}

		balances := make(map[string]interface{})
		result := response["result"].([]interface{})
		if result != nil {
			for _, item := range result {
				balance := item.(map[string]interface{})
				balances[balance["Currency"].(string)] = balance["Available"].(float64)
			}

			return balances
		}

	}

	return nil
}

// Trade() trade as the configs
func (p *Bittrex) Trade(configs TradeConfig) *TradeResult {
	var path string
	if configs.Type == TradeTypeBuy {
		path = "/market/buylimit"
	} else {
		path = "/market/selllimit"
	}

	if err, response := p.marketRequest(path, map[string]string{
		"market":   p.parsePair(configs.Pair),
		"apikey":   p.config.API,
		"quantity": strconv.FormatFloat(configs.Amount, 'f', 4, 64),
		"rate":     strconv.FormatFloat(configs.Price, 'f', 8, 64),
	}, true); err != nil {
		logger.Errorf("Fail to trade:%v", err)
		return &TradeResult{
			Error: err,
		}
	} else {

		if response == nil || response["success"] != true {
			logger.Errorf("Fail to trade:%v", response["message"])
			return &TradeResult{
				Error: errors.New(response["message"].(string)),
			}
		}

		uuid := response["result"].(map[string]interface{})["uuid"].(string)
		return &TradeResult{
			OrderID: uuid,
			Error:   nil,
			Info:    nil,
		}

	}

	return nil
}

// CancelOrder() cancel the order as the order information
func (p *Bittrex) CancelOrder(order OrderInfo) *TradeResult {
	return nil
}

// GetOrderInfo() get the information with order filter
func (p *Bittrex) GetOrderInfo(filter OrderInfo) []OrderInfo {
	if err, response := p.marketRequest("/account/getorder", map[string]string{
		"uuid":   filter.OrderID,
		"apikey": p.config.API,
	}, true); err != nil {
		logger.Errorf("Fail to trade:%v", err)
		return nil
	} else {

		if response == nil || response["success"] != true {
			logger.Errorf("Fail to trade:%v", response["message"])
			return nil
		}

		result := response["result"].(map[string]interface{})
		orderInfo := make([]OrderInfo, 1)
		if result["IsOpen"].(bool) {
			orderInfo[0].Status = OrderStatusOpen
		} else {
			orderInfo[0].Pair = filter.Pair
			orderInfo[0].Price = filter.Price
			orderInfo[0].Type = filter.Type
			orderInfo[0].Amount = filter.Amount
			orderInfo[0].OrderID = filter.OrderID
			orderInfo[0].Status = OrderStatusDone
			orderInfo[0].AvgPrice = result["PricePerUnit"].(float64)
			orderInfo[0].DealAmount = result["Quantity"].(float64) - result["QuantityRemaining"].(float64)

			return orderInfo
		}
	}

	return nil
}

func (p *Bittrex) GetKline(pair string, period int, limit int) []KlineValue {
	return nil
}
