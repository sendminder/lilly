package handler

import (
	"context"
	"encoding/json"
	"fmt"
	msg "lilly/proto/message"
	relay "lilly/proto/relay"
	"lilly/protocol"
	"log"
	"math/rand"
)

func HandleCreateMessage(payload json.RawMessage) {
	var reqCreateMsg protocol.CreateMessage
	err := json.Unmarshal(payload, &reqCreateMsg)
	if err != nil {
		log.Println("Json Error:", err)
		return
	}

	// 받은 메시지를 출력합니다.
	log.Printf("[ReqCreateMessage] convId: %d, text: %s\n", reqCreateMsg.ConversationId, reqCreateMsg.Text)

	// 메시지 생성 요청
	resp, err := createMessage(reqCreateMsg)
	if err != nil {
		log.Printf("Failed to create message: %v", err)
		return
	}
	log.Printf("Created Message: %v\n", resp)

	// 메시지 릴레이
	resp2, err2 := relayMessage(reqCreateMsg, resp)
	if err2 != nil {
		log.Printf("Failed to relay message: %v", err2)
		return
	}
	log.Printf("Relayed Message: %v\n", resp2)

	jsonData, err := createJsonData("message", resp.Message)
	if err != nil {
		fmt.Println("Failed to marshal JSON:", err)
		return
	}

	broadcastEvent := protocol.BroadcastEvent{
		Event:       "message",
		Payload:     jsonData,
		JoinedUsers: resp.JoinedUsers,
	}

	broadcast <- broadcastEvent
}

func HandleReadMessage(payload json.RawMessage) {
	var reqReadMsg protocol.ReadMessage
	jsonErr := json.Unmarshal(payload, &reqReadMsg)
	if jsonErr != nil {
		log.Println("Json Error: ", jsonErr)
		return
	}

	// 받은 메시지를 출력합니다.
	log.Printf("[ReqReadMessage] convId: %d, messageId: %d\n", reqReadMsg.ConversationId, reqReadMsg.MessageId)
	resp, err := readMessage(reqReadMsg)
	if err != nil {
		log.Printf("Failed to read message: %v", err)
		return
	}
	log.Printf("Readed Message: %v\n", resp)

	// TODO: read_message relay
	// read_message 릴레이하면, 클라에서 안읽은 유저수 카운트 처리할수있나?
}

func createMessage(reqCreateMsg protocol.CreateMessage) (*msg.ResponseCreateMessage, error) {
	createMsg := &msg.RequestCreateMessage{
		SenderId:       reqCreateMsg.SenderId,
		ConversationId: reqCreateMsg.ConversationId,
		Text:           reqCreateMsg.Text,
	}

	randIdx := rand.Intn(10)
	mutexes[randIdx].Lock()
	resp, err := messageClient[randIdx].CreateMessage(context.Background(), createMsg)
	mutexes[randIdx].Unlock()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func relayMessage(reqCreateMsg protocol.CreateMessage, resp *msg.ResponseCreateMessage) (*relay.ResponseRelayMessage, error) {
	relayMsg := &relay.RequestRelayMessage{
		Id:             resp.Message.Id,
		ConversationId: reqCreateMsg.ConversationId,
		Text:           reqCreateMsg.Text,
		JoinedUsers:    resp.JoinedUsers,
	}

	randIdx := rand.Intn(10)
	mutexes[randIdx].Lock()
	resp2, err := RelayClient[randIdx].RelayMessage(context.Background(), relayMsg)
	mutexes[randIdx].Unlock()
	if err != nil {
		return nil, err
	}
	return resp2, nil
}

func createJsonMessage(message *msg.Message) json.RawMessage {
	messageJson, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return nil
	}
	return json.RawMessage(messageJson)
}

func readMessage(reqReadMsg protocol.ReadMessage) (*msg.ResponseReadMessage, error) {
	readMsg := &msg.RequestReadMessage{
		UserId:         reqReadMsg.UserId,
		ConversationId: reqReadMsg.ConversationId,
		MessageId:      reqReadMsg.MessageId,
	}

	randIdx := rand.Intn(10)
	mutexes[randIdx].Lock()
	resp, err := messageClient[randIdx].ReadMessage(context.Background(), readMsg)
	mutexes[randIdx].Unlock()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// key, value로 마샬링
func createJsonData(key string, value interface{}) ([]byte, error) {
	data := map[string]interface{}{
		key: value,
	}

	// JSON 마샬링
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}
