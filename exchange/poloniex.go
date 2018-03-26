package exchange

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const PoloniexURL = "https://poloniex.com/public?"

type PoloniexAPI struct {
	tickerList []TickerListItem
}

type PoloniexKlineValue struct {
	Date   float64 `json:"date"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Volumn float64 `json:"volumn"`
}

func (p *PoloniexAPI) Init() {

}

func (p *PoloniexAPI) marketRequest(params map[string]string) (error, []byte) {

	var req http.Request
	req.ParseForm()
	for k, v := range params {
		req.Form.Add(k, v)
	}
	bodystr := strings.TrimSpace(req.Form.Encode())
	logger.Debugf("Params:%v", bodystr)
	request, err := http.NewRequest("GET", PoloniexURL+bodystr, nil)
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

	return nil, body

}

func (p *PoloniexAPI) GetExchangeName() string {
	return "Poloniex"
}

func (p *PoloniexAPI) GetKline(pair string, start time.Time, end *time.Time, periodBySec int) []KlineValue {
	coins := ParsePair(pair)
	symbol := strings.ToUpper(coins[1] + "_" + coins[0])

	endDate := "9999999999"
	if end != nil {
		endDate = strconv.Itoa(int(end.Unix()))
	}

	if err, response := p.marketRequest(map[string]string{
		"command":      "returnChartData",
		"currencyPair": symbol,
		"start":        strconv.Itoa(int(start.Unix())),
		"end":          endDate,
		"period":       strconv.Itoa(periodBySec),
	}); err != nil {
		logger.Errorf("无效数据:%v", err)
		return nil
	} else {
		var values []PoloniexKlineValue
		if response != nil {
			if err = json.Unmarshal(response, &values); err != nil {
				logger.Errorf("Fail to Unmarshal:%v", err)
				return nil
			}

			kline := make([]KlineValue, len(values))
			for i, value := range values {
				// kline[i].OpenTime = time.Unix((int64)(value[0].(float64)/1000), 0).Format(Global.TimeFormat)
				kline[i].OpenTime = value.Date
				kline[i].Open = value.Open
				kline[i].High = value.High
				kline[i].Low = value.Low
				kline[i].Close = value.Close
				kline[i].Volumn = value.Volumn
			}

			return kline
		}

		return nil
	}
}

func (p *PoloniexAPI) GetTickerValue(tag string) map[string]interface{} {
	for _, ticker := range p.tickerList {
		if ticker.Pair == tag {
			if ticker.Value != nil {
				return ticker.Value.(map[string]interface{})
			}
		}
	}

	return nil
}
