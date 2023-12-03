package protocol

import "encoding/json"

type CreateMessage struct {
	SenderID    int64  `json:"sender_id"`
	ChannelID   int64  `json:"channel_id"`
	ChannelType string `json:"channel_type"`
	Text        string `json:"text"`
}

type ReadMessage struct {
	UserID    int64 `json:"user_id"`
	ChannelID int64 `json:"channel_id"`
	MessageID int64 `json:"message_id"`
}

type DecryptChannel struct {
	ChannelID int64 `json:"channel_id"`
}

type FinishChannel struct {
	ChannelID int64 `json:"channel_id"`
}

type BroadcastEvent struct {
	Event       string          `json:"event"`
	Payload     json.RawMessage `json:"payload"`
	JoinedUsers []int64         `json:"joined_users"`
}
