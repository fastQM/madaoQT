package exchange

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	Websocket "github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

const NameGdax = "Gdax"
const GdaxURL = "https://api.gdax.com/"

type ExchangeGdax struct {
	websocket *Websocket.Conn
	event     chan EventType
	config    Config
}

func (p *ExchangeGdax) GetExchangeName() string {
	return NameGdax
}

// SetConfigure()
func (p *ExchangeGdax) SetConfigure(config Config) {
	p.config = config

	if p.config.Proxy != "" {
		logger.Infof("使用代理:%s", p.config.Proxy)
	}
}

func (p *ExchangeGdax) marketRequest(path string, params map[string]string) (error, []byte) {
	var req http.Request
	req.ParseForm()
	for k, v := range params {
		req.Form.Add(k, v)
	}
	bodystr := strings.TrimSpace(req.Form.Encode())
	logger.Debugf("Params:%v Path:%s", bodystr, GdaxURL+path+"?"+bodystr)
	request, err := http.NewRequest("GET", GdaxURL+path+"?"+bodystr, nil)
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

func (p *ExchangeGdax) GetKline(pair string, startUnixTime *time.Time, endUnixTime *time.Time, interval int, limit int) []KlineValue {

	pair = strings.Replace(pair, "usdt", "usd", 1)
	coins := ParsePair(pair)
	symbol := strings.ToUpper(coins[0]) + "-" + strings.ToUpper(coins[1])

	params := map[string]string{
		// "start":       start,
		// "end":         end,
		"granularity": strconv.Itoa(interval),
	}

	// 2014-11-06T10:34:47.123456Z
	var start, end string
	if startUnixTime != nil {
		start = startUnixTime.UTC().Format(time.RFC3339)
		params["start"] = start
	}

	if endUnixTime != nil {
		end = endUnixTime.UTC().Format(time.RFC3339)
		params["end"] = end
	}

	if err, response := p.marketRequest("/products/"+symbol+"/candles", params); err != nil {
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
				kline[i].OpenTime = value[0].(float64)
				kline[i].Low = value[1].(float64)
				kline[i].High = value[2].(float64)
				kline[i].Open = value[3].(float64)
				kline[i].Close = value[4].(float64)
				kline[i].Volumn = value[5].(float64)
				// kline[i].CloseTime = time.Unix((int64)(value[6].(float64)/1000), 0).Format(Global.TimeFormat)
			}

			tmp := make([]KlineValue, len(values))
			copy(tmp, kline)

			j := 0
			for i := len(tmp) - 1; i >= 0; i-- {
				kline[j] = tmp[i]
				j++
			}

			return kline
		}

		return nil
	}
}
