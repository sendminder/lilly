package main

import (
	"lilly/cache"
	"lilly/handler"
	"sync"
)

func main() {
	go cache.CreateRedisConnection()

	var wg sync.WaitGroup
	wg.Add(2)
	go handler.StartRelayServer(&wg)
	go handler.StartWebSocketServer(&wg)

	wg.Wait()
}
