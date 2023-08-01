package main

import (
	"lilly/handler"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(3)
	go handler.StartRelayServer(&wg)
	go handler.CreateRelayConnection(&wg)

	go handler.StartWebSocketServer(&wg)
	go handler.CreateMessageConnection(&wg)

	wg.Wait()
}
