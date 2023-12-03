package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
	"lilly/client/message"
	"lilly/client/relay"
	"lilly/internal/cache"
	"lilly/internal/config"
	"lilly/internal/handler/broadcast"
	"lilly/internal/handler/socket"
	"lilly/internal/util"
)

type WebSocketServer interface {
	MessageHandler
	ChannelHandler
	MatchHandler
	StartWebSocketServer(wg *sync.WaitGroup, port int)
	handleMessages()
}

var _ WebSocketServer = (*webSocketServer)(nil)

type clientMap map[int64]*socket.Socket

type webSocketServer struct {
	activeClients clientMap
	broadcaster   broadcast.Broadcaster
	clientLock    []sync.Mutex
	upgrade       websocket.Upgrader
	relayClient   relay.Client
	messageClient message.Client
}

func NewWebSocketServer(broadcaster broadcast.Broadcaster, relayClient relay.Client, messageClient message.Client) WebSocketServer {
	activeClients := make(clientMap)
	clientLock := make([]sync.Mutex, 10)
	upgrade := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// 모든 origin을 허용하도록 설정합니다.
			return true
		},
	}
	return &webSocketServer{
		activeClients: activeClients,
		broadcaster:   broadcaster,
		clientLock:    clientLock,
		upgrade:       upgrade,
		relayClient:   relayClient,
		messageClient: messageClient,
	}
}

func (wv *webSocketServer) StartWebSocketServer(wg *sync.WaitGroup, port int) {
	defer wg.Done()
	wv.messageClient.CreateMessageConnection()

	http.HandleFunc("/", wv.handleWebSocketConnection)

	// 웹소켓 핸들러를 등록하고 서버를 8080 포트에서 실행합니다.
	slog.Info("WebSocket server listening on", "port", strconv.Itoa(port))

	go wv.handleMessages()

	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
	if err != nil {
		slog.Error("Error starting WebSocket server", "error", err)
	}
}

func (wv *webSocketServer) handleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := wv.upgrade.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Error upgrading connection", "error", err)
		return
	}
	defer conn.Close()

	newClient := &socket.Socket{
		Conn: conn,
		Send: make(chan []byte),
	}
	newClient.UserID = util.GetClientUserID(r)

	wv.broadcaster.Register <- newClient
	defer func() {
		wv.broadcaster.Unregister <- newClient
	}()

	// 클라이언트와의 웹소켓 연결이 성공적으로 이루어졌을 때 로직을 작성합니다.
	slog.Info("Client connected", "userID", newClient.UserID, "localIP", config.LocalIP)

	go newClient.WritePump()

	for {
		// 클라이언트로부터 메시지를 읽습니다.
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			slog.Error("Error reading message", "err", err, "msgType", messageType)
			break
		}

		wv.handleWebSocketMessage(conn, msg)
	}
}

func (wv *webSocketServer) handleWebSocketMessage(_ *websocket.Conn, message []byte) {
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
		wv.handleCreateMessage(clientRequest.Payload)

	case "read_message":
		wv.handleReadMessage(clientRequest.Payload)

	case "decrypt_channel":
		wv.handleDecryptChannel(clientRequest.Payload)

	case "finish_channel":
		wv.handleFinishChannel(clientRequest.Payload)

	case "register_role":
		wv.handleRegisterRole(clientRequest.Payload)

	default:
		slog.Error("Unknown event", "event", clientRequest.Event)
	}
}

func (wv *webSocketServer) handleMessages() {
	for {
		select {
		case msg := <-wv.broadcaster.Broadcast:
			// 메시지를 받아서 모든 클라이언트에게 보냅니다.
			for _, joinedUserID := range msg.JoinedUsers {
				wv.clientLock[joinedUserID%10].Lock()
				if wv.activeClients.contains(joinedUserID) {
					client := wv.activeClients[joinedUserID]

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
					case client.Send <- jsonData:
					default:
						// 보내기 실패한 경우 클라이언트를 제거합니다.
						slog.Error("broadcast Error acquired")
						close(client.Send)
						wv.activeClients.delete(joinedUserID)
					}
				}
				wv.clientLock[joinedUserID%10].Unlock()
			}
		case newSocket := <-wv.broadcaster.Register:
			// 새로운 클라이언트를 activeClients에 등록합니다.
			location := config.LocalIP + ":" + strconv.Itoa(config.WebSocketPort)
			err := cache.SetUserLocation(newSocket.UserID, location)
			if err != nil {
				slog.Error("register error", "error", err)
				close(newSocket.Send)
				wv.activeClients.delete(newSocket.UserID)
			} else {
				slog.Info("registered", "userID", newSocket.UserID, "location", location)
				wv.activeClients[newSocket.UserID] = newSocket
			}
		case closeSocket := <-wv.broadcaster.Unregister:
			// 연결이 끊긴 클라이언트를 activeClients에서 제거합니다.
			err := cache.DeleteUserLocation(closeSocket.UserID)
			if err != nil {
				slog.Error("unregister error", "error", err)
			}
			slog.Info("unregistered", "userID", closeSocket.UserID)
			if _, ok := wv.activeClients[closeSocket.UserID]; ok {
				close(closeSocket.Send)
				wv.activeClients.delete(closeSocket.UserID)
			}
		}
	}
}

func (m clientMap) contains(key int64) bool {
	_, found := m[key]
	return found
}

func (m clientMap) delete(key int64) {
	_, found := m[key]
	if found {
		delete(m, key)
	}
}
