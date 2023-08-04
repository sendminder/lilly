package handler

import (
	"encoding/json"
	"lilly/protocol"
	"log"
	"net/http"
	"strconv"
	"sync"

	"lilly/proto/message"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
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
		log.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close()

	newClient := &client{
		conn: conn,
		send: make(chan []byte),
	}
	newClient.userId = getClientUserId(r)

	log.Println("connection:", newClient)
	register <- newClient
	defer func() {
		unregister <- newClient
	}()

	// 클라이언트와의 웹소켓 연결이 성공적으로 이루어졌을 때 로직을 작성합니다.
	log.Println("Client connected.", newClient.userId)

	go newClient.writePump()

	for {
		// 클라이언트로부터 메시지를 읽습니다.
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err, messageType)
			break
		}

		handleWebSocketMessage(conn, message)
	}
}

func handleWebSocketMessage(conn *websocket.Conn, message []byte) {
	var reqMessage struct {
		Event   string          `json:"event"`
		Payload json.RawMessage `json:"payload"`
	}

	jsonErr := json.Unmarshal(message, &reqMessage)
	if jsonErr != nil {
		log.Println("Json Error: ", jsonErr)
		return
	}

	switch reqMessage.Event {
	case "create_message":
		HandleCreateMessage(reqMessage.Payload)

	case "read_message":
		HandleReadMessage(reqMessage.Payload)

	default:
		log.Println("Unknown event:", reqMessage.Event)
	}
}

func (c *client) writePump() {
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// 채널이 닫힌 경우
				log.Println("Error acquired")
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 클라이언트로 메시지를 보냅니다.
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Println("Error writing message:", err)
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

					// BroadcastEvent의 event와 payload를 따로 분리하여 생성
					eventPayload := map[string]json.RawMessage{
						"event":   []byte(msg.Event),
						"payload": msg.Payload,
					}

					// eventPayload를 Json으로 인코딩
					payloadJson, err := json.Marshal(eventPayload)
					if err != nil {
						log.Printf("failed to marshal payload: %v", err)
						clientLock[joinedUserId%10].Unlock()
						continue
					}

					select {
					case client.send <- []byte(payloadJson):
					default:
						// 보내기 실패한 경우 클라이언트를 제거합니다.
						log.Println("broadcast Error acquired")
						close(client.send)
						activeClients.delete(joinedUserId)
					}
				}
				clientLock[joinedUserId%10].Unlock()
			}
		case client := <-register:
			// 새로운 클라이언트를 activeClients에 등록합니다.
			log.Println("registered")
			activeClients[client.userId] = client
		case client := <-unregister:
			// 연결이 끊긴 클라이언트를 activeClients에서 제거합니다.
			log.Println("unregistered")
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
		log.Println("error: ", err)
		return -1
	}
	return num
}

func StartWebSocketServer(wg *sync.WaitGroup) {
	defer wg.Done()
	http.HandleFunc("/", handleWebSocketConnection)

	// 웹소켓 핸들러를 등록하고 서버를 8080 포트에서 실행합니다.
	log.Println("WebSocket server listening on :8080")

	go handleMessages()

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Error starting WebSocket server:", err)
	}
}

func CreateMessageConnection(wg *sync.WaitGroup) {
	defer wg.Done()
	messageConn, err := grpc.Dial("localhost:3100", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	for i := 0; i < 10; i++ {
		messageClient[i] = message.NewMessageServiceClient(messageConn)
		log.Println("rest client connection", i)
	}
}
