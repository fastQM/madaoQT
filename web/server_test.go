package web

import (
	"time"
	"testing"
	// utils "madaoQT/utils"
)

func TestCreateServer(t *testing.T) {
	
	server := new(HttpServer)
	go server.SetupHttpServer()

	for{
		select{
		case <-time.After(3*time.Second):
			server.BroadcastByWebsocket("hello, world")
		}
	}
}