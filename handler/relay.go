package handler

import (
	"context"
	"lilly/config"
	relay "lilly/proto/relay"
	"log"
	"math/rand"
	"net"
	"sync"

	"google.golang.org/grpc"
)

type relayServer struct {
	relay.UnimplementedRelayServiceServer
}

func (s *relayServer) RelayMessage(ctx context.Context, req *relay.RequestRelayMessage) (*relay.ResponseRelayMessage, error) {
	log.Println("RelayMessage : text=", req.Message.Text)
	return &relay.ResponseRelayMessage{}, nil
}

type relayClientMap map[string][]relay.RelayServiceClient

var (
	relayClients = make(relayClientMap)
	relayPort    string
)

func StartRelayServer(wg *sync.WaitGroup) {
	defer wg.Done()
	listener, err := net.Listen("tcp", ":"+relayPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	srv := grpc.NewServer()
	relay.RegisterRelayServiceServer(srv, &relayServer{})
	log.Printf("gRPC server is listening on port %s...\n", relayPort)
	if err := srv.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func init() {
	relayPort = config.GetString("relay.port")
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
	relayConn, err := grpc.Dial(target+":"+relayPort, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	relayClients[target] = make([]relay.RelayServiceClient, 10)
	for i := 0; i < 10; i++ {
		relayClients[target][i] = relay.NewRelayServiceClient(relayConn)
		log.Println("relay client connection", i)
	}
}

func (m relayClientMap) contains(key string) bool {
	_, found := m[key]
	return found
}
