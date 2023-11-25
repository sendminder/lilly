package protocol

import "encoding/json"

type CreateMessage struct {
	SenderId    int64  `json:"sender_id"`
	ChannelId   int64  `json:"channel_id"`
	ChannelType string `json:"channel_type"`
	Text        string `json:"text"`
}

type ReadMessage struct {
	UserId    int64 `json:"user_id"`
	ChannelId int64 `json:"channel_id"`
	MessageId int64 `json:"message_id"`
}

type DecryptChannel struct {
	ChannelId int64 `json:"channel_id"`
}

type FinishChannel struct {
	ChannelId int64 `json:"channel_id"`
}

type BroadcastEvent struct {
	Event       string          `json:"event"`
	Payload     json.RawMessage `json:"payload"`
	JoinedUsers []int64         `json:"joined_users"`
}
