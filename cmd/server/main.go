package main

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"sync"

	"github.com/spf13/viper"
	"lilly/client/message"
	rc "lilly/client/relay"
	"lilly/internal/cache"
	"lilly/internal/config"
	"lilly/internal/handler/broadcast"
	rs "lilly/internal/handler/relay"
	"lilly/internal/handler/ws"
)

var (
	errFailedToStartServer = errors.New("failed to start server")
)

func main() {
	viper.AutomaticEnv()
	ctx := context.Background()
	if err := run(ctx); err != nil {
		slog.Error("failed to run", "error", err)
	}
	slog.Info("server terminated")
}

func run(ctx context.Context) error {
	config.Init()

	var wg sync.WaitGroup
	wg.Add(2)
	broadcaster := broadcast.NewBroadcaster()

	relayServer := rs.NewRelayServer(broadcaster)
	if relayServer == nil {
		return errFailedToStartServer
	}
	go relayServer.StartRelayServer(&wg)

	relayClient := rc.NewRelayClient(ctx)
	if relayClient == nil {
		return errFailedToStartServer
	}
	messageClient := message.NewMessageClient(10)
	if messageClient == nil {
		return errFailedToStartServer
	}
	redisClient := cache.NewRedisClient(ctx, config.GetString("redis.host")+":"+strconv.Itoa(config.GetInt("redis.port")))
	if redisClient == nil {
		return errFailedToStartServer
	}
	webSocketServer := ws.NewWebSocketServer(ctx, broadcaster, relayClient, messageClient, redisClient)
	if webSocketServer == nil {
		return errFailedToStartServer
	}
	go webSocketServer.StartWebSocketServer(&wg, config.GetInt("websocket.port"))

	wg.Wait()

	return nil
}
