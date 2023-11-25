package main

import (
	"context"
	rc "lilly/client/relay"
	"lilly/internal/handler/broadcast"
	"log/slog"
	"sync"

	"github.com/spf13/viper"
	"lilly/client/message"
	"lilly/internal/cache"
	"lilly/internal/config"
	rs "lilly/internal/handler/relay"
	"lilly/internal/handler/ws"
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
	broadcaster := broadcast.NewBroadcaster()

	relayServer := rs.NewRelayServer(broadcaster)
	go relayServer.StartRelayServer(&wg)

	relayClient := rc.NewRelayClient()
	messageClient := message.NewMessageClient(10)
	webSocketServer := ws.NewWebSocketServer(broadcaster, relayClient, messageClient)
	go webSocketServer.StartWebSocketServer(&wg, config.GetInt("websocket.port"))

	wg.Wait()

	return nil
}
