package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"

	"lilly/internal/cache"
	"lilly/internal/config"
	"lilly/internal/protocol"
	msg "lilly/proto/message"
	relay "lilly/proto/relay"
)

func HandleCreateMessage(payload json.RawMessage) {
	var reqCreateMsg protocol.CreateMessage
	err := json.Unmarshal(payload, &reqCreateMsg)
	if err != nil {
		slog.Error("Json Error", "error", err)
		return
	}

	if reqCreateMsg.Text == "" {
		slog.Error("HandleCreateMessage Error: Text is empty")
		return
	}
	// 받은 메시지를 출력합니다.
	slog.Info("[ReqCreateMessage]", "channelId", reqCreateMsg.ChannelId, "text", reqCreateMsg.Text)

	// 메시지 생성 요청
	resp, err := createMessage(reqCreateMsg)
	if err != nil {
		slog.Error("Failed to create message", "error", err)
		return
	}
	slog.Info("Created Message", "resp", resp)

	/*
		1. joined_users 중 현재 local에 있는 유저에게 릴레이
		2. local 에 릴레이 실패 유저 + joined_users 를 보고 현재 local 에 없는 유저 리스트 가져오기
		3. (2)의 결과 redis로 조회
		4. redis 조회시 나오는 ip로 relay 요청
		5. 그래도 보내지 못한 유저에게 push 요청
	*/
	userInfos, err := cache.GetUserLocations(resp.JoinedUsers)
	if err != nil {
		slog.Error("GetUserInfo err", "error", err)
	}
	targetRelayMap := make(map[string][]int64)
	notFoundUsers := make([]int64, 0)
	for userId, location := range userInfos {
		if location == "" {
			notFoundUsers = append(notFoundUsers, userId)
			continue
		}
		if location != config.LocalIP+":"+config.GetString("websocket.port") {
			targetRelayMap[location] = append(targetRelayMap[location], userId)
		}
	}

	msg := &relay.Message{
		Id:        resp.Message.Id,
		ChannelId: reqCreateMsg.ChannelId,
		Text:      reqCreateMsg.Text,
		SenderId:  reqCreateMsg.SenderId,
		Animal:    "cat",
	}

	for targetLocation, users := range targetRelayMap {
		targetIP := targetLocation[:len(targetLocation)-len(config.GetString("websocket.port"))-1]
		// 메시지 릴레이
		if len(users) == 0 {
			continue
		}
		relayMsg := &relay.RequestRelayMessage{
			Message:     msg,
			JoinedUsers: users,
		}
		_, err2 := relayMessage(relayMsg, targetIP)
		if err2 != nil {
			slog.Error("Failed to relay message", "error", err2)
			return
		}
		slog.Info("Relayed Message", "relayMsg", relayMsg)
	}

	if len(notFoundUsers) > 0 {
		slog.Info("PushMessage to", "notFoundUsers", notFoundUsers)
		_, err := pushMessage(resp.Message, notFoundUsers)
		if err != nil {
			slog.Error("Failed to push message", "error", err)
		}
	}

	jsonData, err := createJsonData("message", resp.Message)
	if err != nil {
		slog.Error("Failed to marshal Json", "error", err)
		return
	}

	broadcastEvent := protocol.BroadcastEvent{
		Event:       "message",
		Payload:     jsonData,
		JoinedUsers: resp.JoinedUsers,
	}

	broadcast <- broadcastEvent

	if reqCreateMsg.ChannelType == "bot" {
		_, err := createBotMessage(reqCreateMsg)
		if err != nil {
			slog.Error("Failed to create bot message", "error", err)
			return
		}
	}
}

func HandleReadMessage(payload json.RawMessage) {
	var reqReadMsg protocol.ReadMessage
	jsonErr := json.Unmarshal(payload, &reqReadMsg)
	if jsonErr != nil {
		slog.Error("Json Error", "error", jsonErr)
		return
	}

	// 받은 메시지를 출력합니다.
	slog.Info("[ReqReadMessage]", "channelId", reqReadMsg.ChannelId, "messageId", reqReadMsg.MessageId)
	resp, err := readMessage(reqReadMsg)
	if err != nil {
		slog.Error("Failed to read message", "error", err)
		return
	}
	slog.Info("Readed Message", "resp", resp)

	// TODO: read_message relay
	// read_message 릴레이하면, 클라에서 안읽은 유저수 카운트 처리할수있나?
}

func createMessage(reqCreateMsg protocol.CreateMessage) (*msg.ResponseCreateMessage, error) {
	createMsg := &msg.RequestCreateMessage{
		SenderId:    reqCreateMsg.SenderId,
		ChannelId:   reqCreateMsg.ChannelId,
		ChannelType: reqCreateMsg.ChannelType,
		Text:        reqCreateMsg.Text,
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

func relayMessage(relayMsg *relay.RequestRelayMessage, targetIP string) (*relay.ResponseRelayMessage, error) {
	client := GetRelayClient(targetIP)
	resp, err := client.RelayMessage(context.Background(), relayMsg)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func readMessage(reqReadMsg protocol.ReadMessage) (*msg.ResponseReadMessage, error) {
	readMsg := &msg.RequestReadMessage{
		UserId:    reqReadMsg.UserId,
		ChannelId: reqReadMsg.ChannelId,
		MessageId: reqReadMsg.MessageId,
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

func pushMessage(req *msg.Message, receivers []int64) (*msg.ResponsePushMessage, error) {
	pushMsg := &msg.RequestPushMessage{
		SenderId:        req.SenderId,
		Message:         req,
		ReceiverUserIds: receivers,
	}

	randIdx := rand.Intn(10)
	mutexes[randIdx].Lock()
	resp, err := messageClient[randIdx].PushMessage(context.Background(), pushMsg)
	mutexes[randIdx].Unlock()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func createBotMessage(reqCreateMsg protocol.CreateMessage) (*msg.ResponseBotMessage, error) {
	createBotMsg := &msg.RequestBotMessage{
		SenderId:    reqCreateMsg.SenderId,
		ChannelId:   reqCreateMsg.ChannelId,
		ChannelType: reqCreateMsg.ChannelType,
		Text:        reqCreateMsg.Text,
	}

	randIdx := rand.Intn(10)
	mutexes[randIdx].Lock()
	resp, err := messageClient[randIdx].CreateBotMessage(context.Background(), createBotMsg)
	mutexes[randIdx].Unlock()
	if err != nil {
		return nil, err
	}
	return resp, nil
}
