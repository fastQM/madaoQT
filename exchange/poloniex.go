package exchange

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	Utils "madaoQT/utils"
)

type PoloniexAPI struct {
	tickerList []TickerListItem
}

func (p *PoloniexAPI) Init() {

	go func() {
		for {
			select {
			case <-time.After(5 * time.Second):
				p.ticker()
			}
		}
	}()
}

func (p *PoloniexAPI) ticker() {
	data, err := Utils.HttpGet("https://poloniex.com/public?command=returnTicker", nil)
	if err != nil {
		log.Printf("Fail to get ticker")
		return
	}

	var records map[string]interface{}
	if err = json.Unmarshal(data, &records); err != nil {
		log.Println("Fail to Unmarshal:", err)
		return
	}

	// log.Printf("Recv:%v", records)

	if p.tickerList != nil {
		for i, ticker := range p.tickerList {
			for k, v := range records {
				if ticker.Symbol == k {
					// log.Printf("COMP:%v|%v", k, v)
					p.tickerList[i].Value = v
					break
				}
			}
		}
	}

}

func (p *PoloniexAPI) AddTicker(coinA string, coinB string, tag string) {
	pair := (strings.ToUpper(coinA) + "_" + strings.ToUpper(coinB))

	// log.Printf("Pair:%v", pair)
	ticker := TickerListItem{
		Pair:   tag,
		Symbol: pair,
	}

	p.tickerList = append(p.tickerList, ticker)
}

func (p *PoloniexAPI) GetExchangeName() string {
	return "Poloniex"
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
