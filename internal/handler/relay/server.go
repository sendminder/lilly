package relay

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"time"

	gr "google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"lilly/internal/config"
	"lilly/internal/handler/broadcast"
	"lilly/internal/protocol"
	"lilly/internal/util"
	"lilly/proto/relay"
)

type Server interface {
	StartRelayServer(wg *sync.WaitGroup)
	RelayMessage(ctx context.Context, req *relay.RequestRelayMessage) (*relay.ResponseRelayMessage, error)
}

var _ Server = (*relayServer)(nil)

type relayServer struct {
	relay.UnimplementedRelayServiceServer
	relayPort   string
	broadcaster broadcast.Broadcaster
}

func NewRelayServer(broadcaster broadcast.Broadcaster) Server {
	relayPort := config.GetString("relay.port")
	return &relayServer{
		relayPort:   relayPort,
		broadcaster: broadcaster,
	}
}

func (s *relayServer) RelayMessage(ctx context.Context, req *relay.RequestRelayMessage) (*relay.ResponseRelayMessage, error) {
	slog.Info("RelayMessage", "text", req.Message.Text)

	jsonData, err := util.CreateJsonData("message", req.Message)
	if err != nil {
		slog.Error("Failed to marshal Json", "error", err)
		return nil, err
	}

	broadcastEvent := protocol.BroadcastEvent{
		Event:       "message",
		Payload:     jsonData,
		JoinedUsers: req.JoinedUsers,
	}

	s.broadcaster.Broadcast <- broadcastEvent

	return &relay.ResponseRelayMessage{}, nil
}

func (s *relayServer) StartRelayServer(wg *sync.WaitGroup) {
	defer wg.Done()
	listener, err := net.Listen("tcp", ":"+s.relayPort)
	if err != nil {
		slog.Error("failed to listen", "error", err)
	}
	srv := gr.NewServer(
		gr.KeepaliveParams(
			keepalive.ServerParameters{
				MaxConnectionAge:      24 * time.Hour,
				MaxConnectionAgeGrace: 10 * time.Second,
				Time:                  60 * time.Second,
				Timeout:               10 * time.Second,
			},
		),
		gr.KeepaliveEnforcementPolicy(
			keepalive.EnforcementPolicy{
				MinTime: 60 * time.Second,
			},
		),
	)
	relay.RegisterRelayServiceServer(srv, s)
	slog.Info("gRPC server is listening on port", "relayPort", s.relayPort)
	if err := srv.Serve(listener); err != nil {
		slog.Error("failed to serve", "error", err)
	}
}
