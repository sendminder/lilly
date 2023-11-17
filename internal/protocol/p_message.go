package protocol

import "encoding/json"

type CreateMessage struct {
	SenderId         int64  `json:"sender_id"`
	ConversationId   int64  `json:"conversation_id"`
	ConversationType string `json:"conversation_type"`
	Text             string `json:"text"`
}

type ReadMessage struct {
	UserId         int64 `json:"user_id"`
	ConversationId int64 `json:"conversation_id"`
	MessageId      int64 `json:"message_id"`
}

type DecryptConversation struct {
	ConversationId int64 `json:"conversation_id"`
}

type FinishConversation struct {
	ConversationId int64 `json:"conversation_id"`
}

type BroadcastEvent struct {
	Event       string          `json:"event"`
	Payload     json.RawMessage `json:"payload"`
	JoinedUsers []int64         `json:"joined_users"`
}
