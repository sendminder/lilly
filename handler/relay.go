package handler

import (
	"context"
	relay "lilly/proto/relay"
	"log"
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

var (
	relayConn   *grpc.ClientConn
	RelayClient = make([]relay.RelayServiceClient, 10)
)

func StartRelayServer(wg *sync.WaitGroup) {
	defer wg.Done()
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	srv := grpc.NewServer()
	relay.RegisterRelayServiceServer(srv, &relayServer{})
	log.Println("gRPC server is listening on port 50051...")
	if err := srv.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func init() {
	relayConn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	for i := 0; i < 10; i++ {
		RelayClient[i] = relay.NewRelayServiceClient(relayConn)
		log.Println("relay client connection", i)
	}
}
