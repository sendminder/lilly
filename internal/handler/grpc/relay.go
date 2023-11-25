package grpc

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
	"lilly/internal/util"
	"lilly/proto/relay"
)

type RelayServer interface {
	StartRelayServer(wg *sync.WaitGroup)
	RelayMessage(ctx context.Context, req *relay.RequestRelayMessage) (*relay.ResponseRelayMessage, error)
	GetRelayClient(target string) relay.RelayServiceClient
}

var _ RelayServer = (*relayServer)(nil)

type relayClientMap map[string][]relay.RelayServiceClient

type relayServer struct {
	relay.UnimplementedRelayServiceServer
	rcm       relayClientMap
	relayPort string
}

func NewRelayServer() RelayServer {
	rcm := make(relayClientMap)
	relayPort := config.GetString("relay.port")
	return &relayServer{
		rcm:       rcm,
		relayPort: relayPort,
	}
}

func (s *relayServer) RelayMessage(ctx context.Context, req *relay.RequestRelayMessage) (*relay.ResponseRelayMessage, error) {
	slog.Info("RelayMessage", "text", req.Message.Text)

	_, err := util.CreateJsonData("message", req.Message)
	if err != nil {
		slog.Error("Failed to marshal Json", "error", err)
		return nil, err
	}

	//broadcastEvent := protocol.BroadcastEvent{
	//	Event:       "message",
	//	Payload:     jsonData,
	//	JoinedUsers: req.JoinedUsers,
	//}

	// TODO: broadcast 어케하냐..
	// broadcast <- broadcastEvent

	return &relay.ResponseRelayMessage{}, nil
}

func (s *relayServer) StartRelayServer(wg *sync.WaitGroup) {
	defer wg.Done()
	listener, err := net.Listen("tcp", ":"+s.relayPort)
	if err != nil {
		slog.Error("failed to listen", "error", err)
	}
	srv := grpc.NewServer()
	relay.RegisterRelayServiceServer(srv, s)
	slog.Info("gRPC server is listening on port", "relayPort", s.relayPort)
	if err := srv.Serve(listener); err != nil {
		slog.Error("failed to serve", "error", err)
	}
}

func (s *relayServer) GetRelayClient(target string) relay.RelayServiceClient {
	randIdx := rand.Intn(10)
	// mutexes[randIdx].Lock()
	// mutexes[randIdx].Unlock()
	if !s.rcm.contains(target) {
		s.createRelayClient(target)
	}

	return s.rcm[target][randIdx]
}

func (s *relayServer) createRelayClient(target string) {
	relayConn, err := grpc.Dial(target+":"+s.relayPort,
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

	s.rcm[target] = make([]relay.RelayServiceClient, 10)
	for i := 0; i < 10; i++ {
		s.rcm[target][i] = relay.NewRelayServiceClient(relayConn)
		slog.Info("relay client connection", "i", i)
	}
}

func (m relayClientMap) contains(key string) bool {
	_, found := m[key]
	return found
}
