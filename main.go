package main

import (
	"lilly/handler"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(2)
	go handler.StartRelayServer(&wg)
	go handler.StartWebSocketServer(&wg)

	wg.Wait()
}
