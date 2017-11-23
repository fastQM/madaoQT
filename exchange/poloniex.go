package exchange

import (
	"time"
	"log"
	"encoding/json"
	"strings"

	Utils "madaoqt/utils"
)

type PoloniexAPI struct {
	tickerList []tickerValue
}

func (p *PoloniexAPI) Init() {

	// get ticker
	go func(){
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
		log.Printf("Fail to get ticker");
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
				if ticker.Name == k {
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
	ticker := tickerValue {
		Tag: tag,
		Name: pair,
	}

	p.tickerList = append(p.tickerList, ticker)
}

func (p *PoloniexAPI) GetExchangeName() string {
	return "Poloniex";
}

func (p *PoloniexAPI) GetTickerValue(tag string) map[string]interface{} {
	for _, ticker := range p.tickerList {
		if ticker.Tag == tag {
			if ticker.Value != nil {
				return ticker.Value.(map[string]interface{})
			}
		}
	}

	return nil
}