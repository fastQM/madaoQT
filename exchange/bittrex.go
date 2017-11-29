package exchange

import (
	"time"
	"encoding/json"
	"strings"

	Utils "madaoQT/utils"
)

const BittrexMarketUrl = "https://bittrex.com/api/v1.1/public/getticker?market="

type BittrexAPI struct {
	tickerList []TickerListItem
}

func (b *BittrexAPI) Init() {

	var counter int
	// get ticker
	go func(){
		for {
			select {
			case <-time.After(1 * time.Second):
				if counter < len(b.tickerList) {
					b.ticker(b.tickerList[counter].Name)
					counter++
				} else {
					counter = 0
				}
			}
		}
	}()
}

func (b *BittrexAPI) ticker(pair string) {
	url := BittrexMarketUrl + pair
	// log.Printf("URL:%s", url)
	data, err := Utils.HttpGet(url, nil)
	if err != nil {
		Logger.Errorf("fail to http request:%v", err);
		return
	}

	var records map[string]interface{}
	if err = json.Unmarshal(data, &records); err != nil {
		Logger.Errorf("Fail to Unmarshal:%v", err)
		return
	}

	// log.Printf("record:%v", records)

	if !records["success"].(bool) {
		Logger.Error("Fail to get ticker")
		return
	}

	values := records["result"].(map[string]interface{})

	// log.Printf("Recv:%v", records)

	if b.tickerList != nil {
		for i, ticker := range b.tickerList {
			if ticker.Name == pair {
				b.tickerList[i].Value = values;
				break
			}
		}
	}

}


// USDT-BTC
func (b *BittrexAPI) AddTicker(coinA string, coinB string, tag string) {
	pair := (strings.ToUpper(coinA) + "-" + strings.ToUpper(coinB))

	// log.Printf("Pair:%v", pair)
	ticker := TickerListItem{
		Tag: tag,
		Name: pair,
	}

	b.tickerList = append(b.tickerList, ticker)
}

func (b *BittrexAPI) GetExchangeName() string {
	return "Bittrex";
}

func (b *BittrexAPI) GetTickerValue(tag string) map[string]interface{} {
	for _, ticker := range b.tickerList {
		if ticker.Tag == tag {
			if ticker.Value != nil {
				return ticker.Value.(map[string]interface{})
			}
		}
	}

	return nil
}