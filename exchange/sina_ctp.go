package exchange

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

// RB0 螺纹钢
// AG0 白银
// AU0 黄金
// CU0 沪铜
// AL0 沪铝
// ZN0 沪锌
// PB0 沪铅
// RU0 橡胶
// FU0 燃油
// WR0 线材
// A0 大豆
// M0 豆粕
// Y0 豆油
// J0 焦炭
// C0 玉米
// L0 乙烯
// P0 棕油
// V0 PVC
// RS0 菜籽
// RM0 菜粕
// FG0 玻璃
// CF0 棉花
// WS0 强麦
// ER0 籼稻
// ME0 甲醇
// RO0 菜油
// TA0 甲酸

const SinaFuture15Min = "http://stock2.finance.sina.com.cn/futures/api/json.php/IndexService.getInnerFuturesMiniKLine15m?symbol="
const SinaFuture60Min = "http://stock2.finance.sina.com.cn/futures/api/json.php/IndexService.getInnerFuturesMiniKLine60m?symbol="
const SinaStockUrl = "http://stock2.finance.sina.com.cn/futures/api/json.php/CffexFuturesService.getCffexFuturesDailyKLine?symbol="
const SinaFutureUrl = "http://stock2.finance.sina.com.cn/futures/api/json.php/IndexService.getInnerFuturesDailyKLine?symbol="

type SinaCTP struct {
}

const (
	// Cotton 棉花
	Cotton = "CF0"
	// DeformedSteelBar 螺纹钢
	DeformedSteelBar = "RB0"
	// Silver 白银
	Silver = "AG0"
)

func (p *SinaCTP) marketRequest(name string) (error, []byte) {

	// logger.Debugf("Params:%v", bodystr)
	request, err := http.NewRequest("GET", SinaStockUrl+name, nil)
	if err != nil {
		return err, nil
	}

	// setup a http client
	httpTransport := &http.Transport{}
	httpClient := &http.Client{Transport: httpTransport}

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

func (p *SinaCTP) GetKline(pair string, start time.Time, end *time.Time, periodBySec int) []KlineValue {

	if err, response := p.marketRequest(pair); err != nil {
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
				kline[i].Time = value[0].(string)
				kline[i].Open, _ = strconv.ParseFloat(value[1].(string), 64)
				kline[i].High, _ = strconv.ParseFloat(value[2].(string), 64)
				kline[i].Low, _ = strconv.ParseFloat(value[3].(string), 64)
				kline[i].Close, _ = strconv.ParseFloat(value[4].(string), 64)
				kline[i].Volumn, _ = strconv.ParseFloat(value[5].(string), 64)

			}

			return kline
		}
	}

	return nil
}
