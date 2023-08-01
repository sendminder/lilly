package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"lilly/protocol"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"

	"lilly/proto/message"
	msg "lilly/proto/message"
	relay "lilly/proto/relay"

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
	broadcast     = make(chan protocol.BroadcastMessage)
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
	fmt.Println("Client connected.", newClient.userId)

	go newClient.writePump()

	for {
		// 클라이언트로부터 메시지를 읽습니다.
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err, messageType)
			break
		}

		var broadcastMsg protocol.BroadcastMessage
		var reqCreateMsg protocol.CreateMessage
		jsonErr := json.Unmarshal([]byte(message), &reqCreateMsg)
		if jsonErr != nil {
			log.Println("JSON Error: ", jsonErr)
			return
		}
		// 받은 메시지를 출력합니다.
		fmt.Printf("Received convId: %d, text: %s\n", reqCreateMsg.ConversationId, reqCreateMsg.Text)

		// -- message 생성
		// 0. 모카에게 create message 요청을 한다.
		// 1. msg의 conversation id를 가져온다.
		// 2. conv_id 기준으로 Redis에 조회해서 user ids를 가져온다.
		// 2-1. redis에 없다면 db에서 가져오는데 이거 가져오는건 gRPC로 모카에게 요청
		// 3. 가져온 user_ids를 각각 Redis에 조회해서 어떤 서버에 붙었는지 ip를 가져온다.
		// 4. 각 IP의 릴리 에게 gRPC로 메시지 릴레이 요청을 한다.
		// 5. user_ids중 레디스에 없는 유저는 펀치에게 push 요청을 한다.
		// 6. 메시지의 안읽음 카운트를 -1 한다.

		createMsg := &msg.RequestCreateMessage{
			SenderId:       newClient.userId,
			ConversationId: reqCreateMsg.ConversationId,
			Text:           reqCreateMsg.Text,
		}

		randIdx := rand.Intn(10)
		mutexes[randIdx].Lock()
		resp, err := messageClient[randIdx].CreateMessage(context.Background(), createMsg)
		mutexes[randIdx].Unlock()
		if err != nil {
			log.Fatalf("failed to relay message: %v", err)
		}
		fmt.Printf("Relayed Message : %v\n", resp)

		// -- join 요청
		// 1. 모카에게 join conversation 요청을 한다.
		// 2. 모카는 메시지 읽음처리를 한다. 메시지의 안읽음 카운트를 -1 한다.
		// 3. 메시지 읽음 이벤트를 보낸다.

		// -- read 요청
		// 1. 모카에게 read message 요청을 한다.
		// 2. 메시지의 안읽음 카운트를 -1 한다.
		// 3. 메시지 읽음 이벤트를 보낸다.

		relayMsg := &relay.RequestRelayMessage{
			Id:             resp.Message.Id,
			ConversationId: reqCreateMsg.ConversationId,
			Text:           reqCreateMsg.Text,
			JoinedUsers:    resp.JoinedUsers,
		}

		randIdx = rand.Intn(10)
		mutexes[randIdx].Lock()
		resp2, err2 := RelayClient[randIdx].RelayMessage(context.Background(), relayMsg)
		mutexes[randIdx].Unlock()
		if err2 != nil {
			log.Fatalf("failed to relay message: %v", err2)
		}
		fmt.Printf("Relayed Message : %v\n", resp2)

		broadcastMsg.ConversationId = reqCreateMsg.ConversationId
		broadcastMsg.Text = reqCreateMsg.Text
		broadcastMsg.JoinedUsers = relayMsg.JoinedUsers

		broadcast <- broadcastMsg
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
					select {
					case client.send <- []byte(msg.Text):
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
	fmt.Println("WebSocket server listening on :8080")

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
