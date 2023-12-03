package relay

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"lilly/proto/relay"
)

type Client interface {
	GetRelayClient(target string, port string) (relay.RelayServiceClient, error)
}

var (
	ErrNotReady = errors.New("gRPC connection is not ready")
)

var _ Client = (*relayClient)(nil)

type relayClientMap map[string][]relay.RelayServiceClient

type relayClient struct {
	ctx context.Context
	rcm relayClientMap
}

func NewRelayClient(ctx context.Context) Client {
	rcm := make(relayClientMap)
	return &relayClient{
		ctx: ctx,
		rcm: rcm,
	}
}

func (rc *relayClient) GetRelayClient(target string, port string) (relay.RelayServiceClient, error) {
	randIdx := rand.Intn(10)
	if !rc.rcm.contains(target) {
		err := rc.createRelayClient(target, port)
		if err != nil {
			slog.Error("failed to create relay client", "error", err)
			return nil, err
		}
	}

	return rc.rcm[target][randIdx], nil
}

func (rc *relayClient) createRelayClient(target string, port string) error {
	ctx, cancel := context.WithTimeout(rc.ctx, 3*time.Second)
	defer cancel()

	relayConn, err := grpc.DialContext(ctx, target+":"+port,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(
			keepalive.ClientParameters{
				Time:                60 * time.Second,
				Timeout:             3 * time.Second,
				PermitWithoutStream: true,
			},
		), grpc.WithBlock())
	if err != nil {
		slog.Error("failed to connect", "error", err)
		return err
	}

	state := relayConn.GetState()
	if state == connectivity.Ready || state == connectivity.Idle || state == connectivity.Connecting {
		slog.Info("gRPC connection is ready")
	} else {
		slog.Error("gRPC connection is not ready", "state", state)
		return ErrNotReady
	}

	rc.rcm[target] = make([]relay.RelayServiceClient, 10)
	for i := 0; i < 10; i++ {
		rc.rcm[target][i] = relay.NewRelayServiceClient(relayConn)
		slog.Info("relay client connection", "i", i)
	}
	return nil
}

func (m relayClientMap) contains(key string) bool {
	_, found := m[key]
	return found
}
