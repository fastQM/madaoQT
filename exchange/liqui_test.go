package exchange

import (
	"log"
	"testing"
)

func TestGetLiquiDepth(t *testing.T) {
	liqui := new(Liqui)
	result := liqui.GetDepthValue("eth/usdt")
	log.Printf("result:%v", result)
}
