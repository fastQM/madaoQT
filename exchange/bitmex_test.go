package exchange

import (
	"log"
	"testing"
)

func TestOffset(t *testing.T) {

	bitmex := new(ExchangeBitmex)
	bitmex.SetConfigure(Config{
		Proxy: "SOCKS5:127.0.0.1:1080",
	})

	err, result := bitmex.GetComposite(".BXBT", 200)
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	log.Printf("Result:%v", result)

}
