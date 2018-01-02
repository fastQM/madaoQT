package exchange

import (
	"encoding/json"
	"strings"
	"time"

	Utils "madaoQT/utils"
)

const BittrexMarketUrl = "https://bittrex.com/api/v1.1/public/getticker?market="

type BittrexAPI struct {
	tickerList []TickerListItem
}

func (b *BittrexAPI) Init() {

	var counter int
	// get ticker
	go func() {
		for {
			select {
			case <-time.After(1 * time.Second):
				if counter < len(b.tickerList) {
					b.ticker(b.tickerList[counter].Pair)
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
		logger.Errorf("fail to http request:%v", err)
		return
	}

	var records map[string]interface{}
	if err = json.Unmarshal(data, &records); err != nil {
		logger.Errorf("Fail to Unmarshal:%v", err)
		return
	}

	// log.Printf("record:%v", records)

	if !records["success"].(bool) {
		logger.Error("Fail to get ticker")
		return
	}

	values := records["result"].(map[string]interface{})

	// log.Printf("Recv:%v", records)

	if b.tickerList != nil {
		for i, ticker := range b.tickerList {
			if ticker.Pair == pair {
				b.tickerList[i].Value = values
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
		Pair:   tag,
		Symbol: pair,
	}

	b.tickerList = append(b.tickerList, ticker)
}

func (b *BittrexAPI) GetExchangeName() string {
	return "Bittrex"
}

func (b *BittrexAPI) GetTicker(pair string) map[string]interface{} {
	for _, ticker := range b.tickerList {
		if ticker.Pair == pair {
			if ticker.Value != nil {
				return ticker.Value.(map[string]interface{})
			}
		}
	}

	return nil
}
