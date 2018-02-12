package exchange

import (
	"log"
	"testing"
)

func TestBittrexGetDepth(t *testing.T) {

	bittrex := new(Bittrex)
	result := bittrex.GetDepthValue("eth/usdt")
	log.Printf("result:%v", result)
}
