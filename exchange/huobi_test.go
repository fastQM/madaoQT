package exchange

import (
	"log"
	"testing"
)

func TestHttpRequest(t *testing.T) {
	huobi := new(Huobi)
	err, result := huobi.marketRequest("/depth", map[string]string{
		"symbol": "ethusdt",
		"type":   "step0",
	})

	if err != nil {
		log.Printf("Error:%v", err)
	} else {
		log.Printf("Rsp:%v", result)
	}
}

func TestGetDepth(t *testing.T) {
	huobi := new(Huobi)
	result := huobi.GetDepthValue("eth/usdt")
	if result != nil {
		log.Printf("Depth:%v", result)
	}
}
