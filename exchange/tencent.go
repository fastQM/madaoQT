package exchange

import (
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const TencentPrefix = "http://data.gtimg.cn/flashdata/hushen/daily/"
const TencentLatestPrefix = "http://qt.gtimg.cn/q="
const TencentURL = "http://data.gtimg.cn/flashdata/hushen/daily/[year]/[stock].js"

type TencentStock struct {
}

func (p *TencentStock) marketRequest(path string) (error, []byte) {

	request, err := http.NewRequest("GET", path, nil)
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

	// info, _ := ioutil.ReadAll(transform.NewReader(bytes.NewReader(body), simplifiedchinese.HZGB2312.NewEncoder()))
	// fmt.Printf("Body:%s", string(info))

	return nil, body

}

func (p *TencentStock) GetDialyKlines(startyear int, code string) []KlineValue {

	var klines []KlineValue

	currentyear := time.Now().Year()
	start := strings.Replace(strconv.Itoa(startyear), "20", "", 1)
	end := strings.Replace(strconv.Itoa(currentyear), "20", "", 1)

	url := strings.Join([]string{TencentPrefix, start, "/", code, ".js"}, "")
	// logger.Infof("URL:%s", url)

	err, rsp := p.marketRequest(url)
	if err != nil {
		logger.Errorf("Error:%v", err)
		return nil
	}

	lists := strings.Split(string(rsp), "\\n\\")
	for i := 1; i < len(lists)-1; i++ {
		var kline KlineValue
		lists[i] = strings.Replace(lists[i], "\n", " ", 1)
		item := strings.Split(lists[i], " ")
		kline.OpenTime, _ = strconv.ParseFloat(item[1], 64)
		kline.Open, _ = strconv.ParseFloat(item[2], 64)
		kline.Close, _ = strconv.ParseFloat(item[3], 64)
		kline.High, _ = strconv.ParseFloat(item[4], 64)
		kline.Low, _ = strconv.ParseFloat(item[5], 64)
		kline.Volumn, _ = strconv.ParseFloat(item[6], 64)
		klines = append(klines, kline)
	}

	if end != start {
		url = strings.Join([]string{TencentPrefix, end, "/", code, ".js"}, "")
		// logger.Infof("URL:%s", url)
		p.marketRequest(url)

		err, rsp = p.marketRequest(url)
		if err != nil {
			logger.Errorf("Error:%v", err)
			return nil
		}

		lists = strings.Split(string(rsp), "\\n\\")
		for i := 1; i < len(lists)-1; i++ {
			var kline KlineValue
			lists[i] = strings.Replace(lists[i], "\n", " ", 1)
			// log.Printf("%d value:%v", i, lists[i])
			item := strings.Split(lists[i], " ")
			kline.OpenTime, _ = strconv.ParseFloat(item[1], 64)
			kline.Open, _ = strconv.ParseFloat(item[2], 64)
			kline.Close, _ = strconv.ParseFloat(item[3], 64)
			kline.High, _ = strconv.ParseFloat(item[4], 64)
			kline.Low, _ = strconv.ParseFloat(item[5], 64)
			kline.Volumn, _ = strconv.ParseFloat(item[6], 64)
			klines = append(klines, kline)
		}
	}

	return klines
}

func (p *TencentStock) GetLast(code string) *KlineValue {
	url := strings.Join([]string{TencentLatestPrefix, code}, "")
	log.Printf("URL:%v", url)
	err, rsp := p.marketRequest(url)
	if err != nil {
		logger.Errorf("Error:%v", err)
		return nil
	}

	data := strings.Split(string(rsp), "~")

	var kline KlineValue
	if len(data) < 35 {
		return nil
	}

	kline.OpenTime, _ = strconv.ParseFloat(data[30], 64)
	kline.Open, _ = strconv.ParseFloat(data[5], 64)
	kline.High, _ = strconv.ParseFloat(data[33], 64)
	kline.Low, _ = strconv.ParseFloat(data[34], 64)
	kline.Close, _ = strconv.ParseFloat(data[3], 64)
	kline.Volumn, _ = strconv.ParseFloat(data[6], 64)

	return &kline
}

func (p *TencentStock) formatTime(openTime float64) string {
	openTimeString := strconv.FormatFloat(openTime, 'f', 0, 64)
	openTimeArray := strings.Split(openTimeString, "")
	year := openTimeArray[0] + openTimeArray[1] + openTimeArray[2] + openTimeArray[3]
	month := openTimeArray[4] + openTimeArray[5]
	date := openTimeArray[6] + openTimeArray[7]
	return year + "-" + month + "-" + date
}

func (p *TencentStock) GetMultipleLast(code string) map[string]KlineValue {
	url := strings.Join([]string{TencentLatestPrefix, code}, "")
	// log.Printf("URL:%v", url)
	err, rsp := p.marketRequest(url)
	if err != nil {
		logger.Errorf("Error:%v", err)
		return nil
	}

	prices := make(map[string]KlineValue)
	stocks := strings.Split(string(rsp), ";")
	for _, stock := range stocks {
		// log.Printf("Stock:%s", stock)
		data := strings.Split(string(stock), "~")
		var kline KlineValue
		if len(data) < 35 {
			// log.Printf("Error:%v", data)
			continue
		}

		code := data[2]
		kline.OpenTime, _ = strconv.ParseFloat(data[30], 64)
		kline.Time = p.formatTime(kline.OpenTime)
		kline.Open, _ = strconv.ParseFloat(data[5], 64)
		kline.High, _ = strconv.ParseFloat(data[33], 64)
		kline.Low, _ = strconv.ParseFloat(data[34], 64)
		kline.Close, _ = strconv.ParseFloat(data[3], 64)
		kline.Volumn, _ = strconv.ParseFloat(data[6], 64)

		prices[code] = kline
	}

	return prices
}
