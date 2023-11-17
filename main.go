package main

import (
	"lilly/internal/cache"
	handler2 "lilly/internal/handler"
	"sync"
)

func main() {
	go cache.CreateRedisConnection()

	var wg sync.WaitGroup
	wg.Add(2)
	go handler2.StartRelayServer(&wg)
	go handler2.StartWebSocketServer(&wg)

	wg.Wait()
}
