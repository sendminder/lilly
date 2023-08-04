package handler

import (
	"context"
	"encoding/json"
	msg "lilly/proto/message"
	"lilly/protocol"
	"log"
	"math/rand"
)

func HandleDecryptConversation(payload json.RawMessage) {
	var reqDecryptConv protocol.DecryptConversation
	err := json.Unmarshal(payload, &reqDecryptConv)
	if err != nil {
		log.Println("Json Error:", err)
		return
	}
	// 받은 메시지를 출력
	log.Printf("[ReqDecryptConversation] convId: %d\n", reqDecryptConv.ConversationId)
	resp, err := decryptConversation(reqDecryptConv)
	if err != nil {
		log.Printf("Failed to decrypt conversation: %v", err)
		return
	}
	log.Printf("Decrypted Conversation: %v\n", resp)

	// TODO: decrypted_conversation relay
	// decrypted_conversation 받으면, 클라에서 복호화 하기
}

func decryptConversation(reqDecryptConv protocol.DecryptConversation) (*msg.ResponseDecryptConversation, error) {
	decryptConv := &msg.RequestDecryptConversation{
		ConversationId: reqDecryptConv.ConversationId,
	}

	randIdx := rand.Intn(10)
	mutexes[randIdx].Lock()
	resp, err := messageClient[randIdx].DecryptConversation(context.Background(), decryptConv)
	mutexes[randIdx].Unlock()
	if err != nil {
		return nil, err
	}
	return resp, nil
}
