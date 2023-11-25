package message

import (
	"log/slog"
	"math/rand"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"lilly/internal/config"
	"lilly/proto/message"
)

type Client interface {
	GetMessageClient() message.MessageServiceClient
	CreateMessageConnection()
}

var _ Client = (*messageClient)(nil)

type messageClient struct {
	clients    []message.MessageServiceClient
	numClients int
}

func NewMessageClient(numClients int) Client {
	clients := make([]message.MessageServiceClient, 10)
	return &messageClient{
		clients:    clients,
		numClients: numClients,
	}
}

func (mc *messageClient) GetMessageClient() message.MessageServiceClient {
	randIdx := rand.Intn(mc.numClients)
	return mc.clients[randIdx]
}

func (mc *messageClient) CreateMessageConnection() {
	mochaIP := config.GetString("mocha.ip")
	mochaPort := config.GetString("mocha.port")

	messageConn, err := grpc.Dial(mochaIP+":"+mochaPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(
			keepalive.ClientParameters{
				Time:                3600 * time.Second,
				Timeout:             3 * time.Second,
				PermitWithoutStream: true,
			},
		))
	if err != nil {
		slog.Error("failed to connect relay", "error", err)
	}
	for i := 0; i < mc.numClients; i++ {
		mc.clients[i] = message.NewMessageServiceClient(messageConn)
		slog.Info("mocha client connection", "mocha", mochaIP+":"+mochaPort+"["+strconv.Itoa(i)+"]")
	}
}
