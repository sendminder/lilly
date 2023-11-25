package main

import (
	"context"
	"log/slog"
	"sync"

	"github.com/spf13/viper"
	"lilly/internal/cache"
	"lilly/internal/config"
	"lilly/internal/handler"
)

func main() {
	viper.AutomaticEnv()
	ctx := context.Background()
	if err := run(ctx); err != nil {
		test := 1
		slog.Error("failed to run", "test", test)
	}
	slog.Info("server terminated")
}

func run(ctx context.Context) error {
	config.Init()
	go cache.CreateRedisConnection()

	var wg sync.WaitGroup
	wg.Add(2)
	go handler.StartRelayServer(&wg)
	go handler.StartWebSocketServer(&wg)

	wg.Wait()

	return nil
}
