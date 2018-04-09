package exchange

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	Websocket "github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

const NameBitmex = "https://www.bitmex.com/api/v1"

//对我们的 REST API 的请求是限于每 5 分钟 300 次的速率。此计数器持续重设。如果您没有登录，您的频率限制是每 5 分钟 150 次。

type ExchangeBitmex struct {
	websocket *Websocket.Conn
	event     chan EventType
	config    Config
}

// SetConfigure()
func (p *ExchangeBitmex) SetConfigure(config Config) {
	p.config = config

	if p.config.Proxy != "" {
		logger.Infof("使用代理:%s", p.config.Proxy)
	}
}

func (p *ExchangeBitmex) marketRequest(path string, params map[string]string) (error, []byte) {
	var req http.Request
	req.ParseForm()
	for k, v := range params {
		req.Form.Add(k, v)
	}
	bodystr := strings.TrimSpace(req.Form.Encode())
	logger.Debugf("Params:%v Path:%s", bodystr, NameBitmex+path+"?"+bodystr)
	request, err := http.NewRequest("GET", NameBitmex+path+"?"+bodystr, nil)
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

func (p *ExchangeBitmex) GetComposite(symbol string, limit int) (error, float64) {

	if err, response := p.marketRequest("/instrument/compositeIndex", map[string]string{
		"symbol":  symbol,
		"limit":   strconv.Itoa(limit),
		"reverse": "true",
	}); err != nil {
		logger.Errorf("无效深度:%v", err)
		return err, 0
	} else {
		var datas []map[string]interface{}
		if err = json.Unmarshal(response, &datas); err != nil {
			logger.Errorf("解析错误:%v", err)
			return err, 0
		}

		var gdaxDatas []map[string]interface{}
		var bitstampDatas []map[string]interface{}

		for _, data := range datas {
			if data["reference"] != nil && data["reference"].(string) == "BSTP" {
				bitstampDatas = append(bitstampDatas, data)
			} else if data["reference"] != nil && data["reference"].(string) == "GDAX" {
				gdaxDatas = append(gdaxDatas, data)
			}
		}

		// for i, data := range gdaxDatas {
		// 	log.Printf("%d Gdax:%v", i, data)
		// }
		// for i, data := range bitstampDatas {
		// 	log.Printf("%d Bitstamp:%v", i, data)
		// }

		if len(gdaxDatas) == 0 || len(bitstampDatas) == 0 {
			return errors.New("数据长度不匹配"), 0
		}

		length := len(gdaxDatas)
		if len(gdaxDatas) > len(bitstampDatas) {
			length = len(bitstampDatas)
		}

		var offset float64
		for i := 0; i < length; i++ {
			if gdaxDatas[i]["timestamp"].(string) != bitstampDatas[i]["timestamp"].(string) {
				return errors.New("数据不匹配"), 0
			}

			weightPrice := (gdaxDatas[i]["lastPrice"].(float64) + bitstampDatas[i]["lastPrice"].(float64)) / 2
			log.Printf("offset:%.6f", gdaxDatas[i]["lastPrice"].(float64)/bitstampDatas[i]["lastPrice"].(float64))
			offset += weightPrice / gdaxDatas[i]["lastPrice"].(float64)
		}

		return nil, offset / float64(length)
	}
}

func (p *ExchangeBitmex) Trade(configs TradeConfig) *TradeResult {
	return nil
}
