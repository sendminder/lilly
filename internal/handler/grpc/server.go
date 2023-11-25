package grpc

import (
	"context"
	"log/slog"
	"net"
	"sync"

	gr "google.golang.org/grpc"
	"lilly/internal/config"
	"lilly/internal/util"
	"lilly/proto/relay"
)

type RelayServer interface {
	StartRelayServer(wg *sync.WaitGroup)
	RelayMessage(ctx context.Context, req *relay.RequestRelayMessage) (*relay.ResponseRelayMessage, error)
}

var _ RelayServer = (*relayServer)(nil)

type relayServer struct {
	relay.UnimplementedRelayServiceServer
	relayPort string
}

func NewRelayServer() RelayServer {
	relayPort := config.GetString("relay.port")
	return &relayServer{
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
	srv := gr.NewServer()
	relay.RegisterRelayServiceServer(srv, s)
	slog.Info("gRPC server is listening on port", "relayPort", s.relayPort)
	if err := srv.Serve(listener); err != nil {
		slog.Error("failed to serve", "error", err)
	}
}
