package relay

import (
	"log/slog"
	"math/rand"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"lilly/proto/relay"
)

type Client interface {
	GetRelayClient(target string, port string) relay.RelayServiceClient
}

var _ Client = (*relayClient)(nil)

type relayClientMap map[string][]relay.RelayServiceClient

type relayClient struct {
	rcm relayClientMap
}

func NewRelayClient() Client {
	rcm := make(relayClientMap)
	return &relayClient{
		rcm: rcm,
	}
}

func (rc *relayClient) GetRelayClient(target string, port string) relay.RelayServiceClient {
	randIdx := rand.Intn(10)
	if !rc.rcm.contains(target) {
		rc.createRelayClient(target, port)
	}

	return rc.rcm[target][randIdx]
}

func (rc *relayClient) createRelayClient(target string, port string) {
	relayConn, err := grpc.Dial(target+":"+port,
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

	rc.rcm[target] = make([]relay.RelayServiceClient, 10)
	for i := 0; i < 10; i++ {
		rc.rcm[target][i] = relay.NewRelayServiceClient(relayConn)
		slog.Info("relay client connection", "i", i)
	}
}

func (m relayClientMap) contains(key string) bool {
	_, found := m[key]
	return found
}
