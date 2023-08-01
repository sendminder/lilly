package protocol

type CreateMessage struct {
	ConversationId int64  `json:"conv_id"`
	Text           string `json:"text"`
}

type BroadcastMessage struct {
	Id             int64   `json:"id"`
	ConversationId int64   `json:"conv_id"`
	Text           string  `json:"text"`
	JoinedUsers    []int64 `json:"joined_users"`
}
