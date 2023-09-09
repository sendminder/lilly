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

func HandleFinishConversation(payload json.RawMessage) {
	var reqFinishConv protocol.FinishConversation
	err := json.Unmarshal(payload, &reqFinishConv)
	if err != nil {
		log.Println("Json Error:", err)
		return
	}

	log.Printf("[ReqFinishConversation] convId: %d\n", reqFinishConv.ConversationId)
	resp, err := finishConversation(reqFinishConv)
	if err != nil {
		log.Printf("Failed to finish conversation: %v", err)
		return
	}
	log.Printf("Finished Conversation: %v\n", resp)

	// TODO: finish_conversation relay
	// finish_conversation 받으면, 클라에서 deactivated 처리
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

func finishConversation(reqFinishConv protocol.FinishConversation) (*msg.ResponseFinishConversation, error) {
	finishConv := &msg.RequestFinishConversation{
		ConversationId: reqFinishConv.ConversationId,
	}

	randIdx := rand.Intn(10)
	mutexes[randIdx].Lock()
	resp, err := messageClient[randIdx].FinishConversation(context.Background(), finishConv)
	mutexes[randIdx].Unlock()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func CreateConversation(name string, userId int64, joinedUsers []int64) (*msg.ResponseCreateConversation, error) {
	createConv := &msg.RequestCreateConversation{
		Name:        name,
		HostUserId:  userId,
		JoinedUsers: joinedUsers,
	}

	randIdx := rand.Intn(10)
	mutexes[randIdx].Lock()
	resp, err := messageClient[randIdx].CreateConversation(context.Background(), createConv)
	mutexes[randIdx].Unlock()
	if err != nil {
		return nil, err
	}
	return resp, nil
}
