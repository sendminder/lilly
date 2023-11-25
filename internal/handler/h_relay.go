package handler

import (
	"context"
	"log/slog"
	"math/rand"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"lilly/internal/config"
	"lilly/internal/protocol"
	relay "lilly/proto/relay"
)

type relayServer struct {
	relay.UnimplementedRelayServiceServer
}

func (s *relayServer) RelayMessage(ctx context.Context, req *relay.RequestRelayMessage) (*relay.ResponseRelayMessage, error) {
	slog.Info("RelayMessage", "text", req.Message.Text)

	jsonData, err := createJsonData("message", req.Message)
	if err != nil {
		slog.Error("Failed to marshal Json", "error", err)
		return nil, err
	}

	broadcastEvent := protocol.BroadcastEvent{
		Event:       "message",
		Payload:     jsonData,
		JoinedUsers: req.JoinedUsers,
	}

	broadcast <- broadcastEvent

	return &relay.ResponseRelayMessage{}, nil
}

type relayClientMap map[string][]relay.RelayServiceClient

var (
	relayClients = make(relayClientMap)
	relayPort    string
)

func StartRelayServer(wg *sync.WaitGroup) {
	defer wg.Done()
	relayPort = config.GetString("relay.port")
	listener, err := net.Listen("tcp", ":"+relayPort)
	if err != nil {
		slog.Error("failed to listen", "error", err)
	}
	srv := grpc.NewServer()
	relay.RegisterRelayServiceServer(srv, &relayServer{})
	slog.Info("gRPC server is listening on port", "relayPort", relayPort)
	if err := srv.Serve(listener); err != nil {
		slog.Error("failed to serve", "error", err)
	}
}

func GetRelayClient(target string) relay.RelayServiceClient {
	randIdx := rand.Intn(10)
	// mutexes[randIdx].Lock()
	// mutexes[randIdx].Unlock()
	if !relayClients.contains(target) {
		createRelayClient(target)
	}

	return relayClients[target][randIdx]
}

func createRelayClient(target string) {
	relayConn, err := grpc.Dial(target+":"+relayPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(
			keepalive.ClientParameters{
				Time:                3600 * time.Second,
				Timeout:             3 * time.Second,
				PermitWithoutStream: true,
			},
		))
	if err != nil {
		slog.Error("failed to connect", "error", err)
	}

	relayClients[target] = make([]relay.RelayServiceClient, 10)
	for i := 0; i < 10; i++ {
		relayClients[target][i] = relay.NewRelayServiceClient(relayConn)
		slog.Info("relay client connection", "i", i)
	}
}

func (m relayClientMap) contains(key string) bool {
	_, found := m[key]
	return found
}
