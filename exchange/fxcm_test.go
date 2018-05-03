package exchange

import (
	"log"
	"testing"
	"time"
)

func TestSocketConnection(t *testing.T) {
	fxcm := new(FXCM)
	err := fxcm.Start()
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	select {
	case <-time.After(10 * time.Second):
		log.Printf("Timeout")
		return
	}
}
