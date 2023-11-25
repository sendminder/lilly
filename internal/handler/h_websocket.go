package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"lilly/internal/cache"
	"lilly/internal/config"
	"lilly/internal/protocol"
	"lilly/proto/message"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 모든 origin을 허용하도록 설정합니다.
		return true
	},
}

type client struct {
	conn   *websocket.Conn
	send   chan []byte
	userId int64
}

type clientMap map[int64]*client

var (
	activeClients = make(clientMap)
	broadcast     = make(chan protocol.BroadcastEvent)
	register      = make(chan *client)
	unregister    = make(chan *client)
	messageConn   *grpc.ClientConn
	messageClient = make([]message.MessageServiceClient, 10)
	mutexes       = make([]sync.Mutex, 10)
	clientLock    = make([]sync.Mutex, 10)
)

func handleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Error upgrading connection", "error", err)
		return
	}
	defer conn.Close()

	newClient := &client{
		conn: conn,
		send: make(chan []byte),
	}
	newClient.userId = getClientUserId(r)

	register <- newClient
	defer func() {
		unregister <- newClient
	}()

	// 클라이언트와의 웹소켓 연결이 성공적으로 이루어졌을 때 로직을 작성합니다.
	slog.Info("Client connected", "userId", newClient.userId, "localIP", config.LocalIP)

	go newClient.writePump()

	for {
		// 클라이언트로부터 메시지를 읽습니다.
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			slog.Error("Error reading message", "err", err, "msgType", messageType)
			break
		}

		handleWebSocketMessage(conn, msg)
	}
}

func handleWebSocketMessage(conn *websocket.Conn, message []byte) {
	var clientRequest struct {
		Event   string          `json:"event"`
		Payload json.RawMessage `json:"payload"`
	}

	jsonErr := json.Unmarshal(message, &clientRequest)
	if jsonErr != nil {
		slog.Error("Json Error", "error", jsonErr)
		return
	}

	switch clientRequest.Event {
	case "create_message":
		HandleCreateMessage(clientRequest.Payload)

	case "read_message":
		HandleReadMessage(clientRequest.Payload)

	case "decrypt_channel":
		HandleDecryptChannel(clientRequest.Payload)

	case "finish_channel":
		HandleFinishChannel(clientRequest.Payload)

	case "register_role":
		HandleRegisterRole(clientRequest.Payload)

	default:
		slog.Error("Unknown event", "event", clientRequest.Event)
	}
}

func (c *client) writePump() {
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				// 채널이 닫힌 경우
				slog.Error("Error acquired")
				err := c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				if err != nil {
					return
				}
				return
			}

			// 클라이언트로 메시지를 보냅니다.
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				slog.Error("Error writing message", "error", err)
				return
			}
		}
	}
}

func handleMessages() {
	for {
		select {
		case msg := <-broadcast:
			// 메시지를 받아서 모든 클라이언트에게 보냅니다.
			for _, joinedUserId := range msg.JoinedUsers {
				clientLock[joinedUserId%10].Lock()
				if activeClients.contains(joinedUserId) {
					client := activeClients[joinedUserId]

					data := map[string]interface{}{
						"event":   msg.Event,
						"payload": msg.Payload,
					}

					// JSON 마샬링
					jsonData, err := json.Marshal(data)
					if err != nil {
						slog.Error("Failed to marshal Json", "error", err)
						return
					}

					select {
					case client.send <- jsonData:
					default:
						// 보내기 실패한 경우 클라이언트를 제거합니다.
						slog.Error("broadcast Error acquired")
						close(client.send)
						activeClients.delete(joinedUserId)
					}
				}
				clientLock[joinedUserId%10].Unlock()
			}
		case client := <-register:
			// 새로운 클라이언트를 activeClients에 등록합니다.
			location := config.LocalIP + ":" + strconv.Itoa(config.WebSocketPort)
			err := cache.SetUserLocation(client.userId, location)
			if err != nil {
				slog.Error("register error", "error", err)
				close(client.send)
				activeClients.delete(client.userId)
			} else {
				slog.Info("registered", "userId", client.userId, "location", location)
				activeClients[client.userId] = client
			}
		case client := <-unregister:
			// 연결이 끊긴 클라이언트를 activeClients에서 제거합니다.
			err := cache.DeleteUserLocation(client.userId)
			if err != nil {
				slog.Error("unregister error", "error", err)
			}
			slog.Info("unregistered", "userId", client.userId)
			if _, ok := activeClients[client.userId]; ok {
				close(client.send)
				activeClients.delete(client.userId)
			}
		}
	}
}

func (m clientMap) contains(key int64) bool {
	_, found := m[key]
	return found
}

func (m clientMap) delete(key int64) bool {
	_, found := m[key]
	if found {
		delete(m, key)
		return true
	}
	return false
}

// 클라이언트의 식별자(예: 유저 이름 또는 ID)를 추출하는 함수
func getClientUserId(r *http.Request) int64 {
	// 클라이언트로부터 전달된 식별자 값을 추출하는 로직을 구현해야 합니다.
	// 예를 들어, URL 쿼리 매개변수로 식별자를 전달받는다고 가정하면 다음과 같이 구현할 수 있습니다.
	userId := r.URL.Query().Get("user_id")
	num, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		slog.Error("getClientUserId ParseInt error", "error", err)
		return -1
	}
	return num
}

func StartWebSocketServer(wg *sync.WaitGroup, port int) {
	defer wg.Done()
	createMessageConnection()

	http.HandleFunc("/", handleWebSocketConnection)

	// 웹소켓 핸들러를 등록하고 서버를 8080 포트에서 실행합니다.
	slog.Info("WebSocket server listening on" + strconv.Itoa(port))

	go handleMessages()

	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
	if err != nil {
		slog.Error("Error starting WebSocket server", "error", err)
	}
}

func createMessageConnection() {
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
		slog.Error("failed to connect grpc", "error", err)
	}
	for i := 0; i < 10; i++ {
		messageClient[i] = message.NewMessageServiceClient(messageConn)
		slog.Info("mocha client connection", "mocha", mochaIP+mochaPort+"["+strconv.Itoa(i)+"]")
	}
}
