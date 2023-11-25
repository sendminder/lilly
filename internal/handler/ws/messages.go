package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	rc "lilly/client/relay"
	"lilly/internal/cache"
	"lilly/internal/config"
	"lilly/internal/protocol"
	"lilly/internal/util"
	msg "lilly/proto/message"
	"lilly/proto/relay"
)

type MessageHandler interface {
	handleCreateMessage(payload json.RawMessage)
	handleReadMessage(payload json.RawMessage)
}

func (wv *webSocketServer) handleCreateMessage(payload json.RawMessage) {
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
	resp, err := wv.createMessage(reqCreateMsg)
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
		if len(users) == 0 {
			continue
		}
		relayMsg := &relay.RequestRelayMessage{
			Message:     msg,
			JoinedUsers: users,
		}
		go func() {
			// NOTE: relay 실패시 커넥트가 안된 상태라면 이미 내려간 릴레이서버로 간주하고 redis 정보를 지운다.
			relayMessage := relayMsg
			ip := targetIP
			_, relayError := wv.relayMessage(relayMessage, ip, config.GetString("relay.port"))
			if errors.Is(relayError, rc.ErrNotReady) || errors.Is(relayError, context.DeadlineExceeded) {
				slog.Error("Failed to connect relay server", "error", relayError)
				for _, userId := range relayMessage.JoinedUsers {
					cacheError := cache.DeleteUserLocation(userId)
					if cacheError != nil {
						slog.Error("GetUserInfo err", "error", cacheError)
					}
				}
				return
			} else if relayError != nil {
				slog.Error("Failed to relay message", "error", relayError)
				return
			}
			slog.Info("Relayed Message", "relayMsg", relayMsg)
		}()
	}

	if len(notFoundUsers) > 0 {
		slog.Info("PushMessage to", "notFoundUsers", notFoundUsers)
		_, err := wv.pushMessage(resp.Message, notFoundUsers)
		if err != nil {
			slog.Error("Failed to push message", "error", err)
		}
	}

	jsonData, err := util.CreateJsonData("message", resp.Message)
	if err != nil {
		slog.Error("Failed to marshal Json", "error", err)
		return
	}

	broadcastEvent := protocol.BroadcastEvent{
		Event:       "message",
		Payload:     jsonData,
		JoinedUsers: resp.JoinedUsers,
	}

	wv.broadcaster.Broadcast <- broadcastEvent

	if reqCreateMsg.ChannelType == "bot" {
		_, err := wv.createBotMessage(reqCreateMsg)
		if err != nil {
			slog.Error("Failed to create bot message", "error", err)
			return
		}
	}
}

func (wv *webSocketServer) handleReadMessage(payload json.RawMessage) {
	var reqReadMsg protocol.ReadMessage
	jsonErr := json.Unmarshal(payload, &reqReadMsg)
	if jsonErr != nil {
		slog.Error("Json Error", "error", jsonErr)
		return
	}

	// 받은 메시지를 출력합니다.
	slog.Info("[ReqReadMessage]", "channelId", reqReadMsg.ChannelId, "messageId", reqReadMsg.MessageId)
	resp, err := wv.readMessage(reqReadMsg)
	if err != nil {
		slog.Error("Failed to read message", "error", err)
		return
	}
	slog.Info("Readed Message", "resp", resp)

	// TODO: read_message relay
	// read_message 릴레이하면, 클라에서 안읽은 유저수 카운트 처리할수있나?
}

func (wv *webSocketServer) createMessage(reqCreateMsg protocol.CreateMessage) (*msg.ResponseCreateMessage, error) {
	createMsg := &msg.RequestCreateMessage{
		SenderId:    reqCreateMsg.SenderId,
		ChannelId:   reqCreateMsg.ChannelId,
		ChannelType: reqCreateMsg.ChannelType,
		Text:        reqCreateMsg.Text,
	}

	resp, err := wv.messageClient.GetMessageClient().CreateMessage(context.Background(), createMsg)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (wv *webSocketServer) relayMessage(relayMsg *relay.RequestRelayMessage, targetIP string, targetPort string) (*relay.ResponseRelayMessage, error) {
	rc, err := wv.relayClient.GetRelayClient(targetIP, targetPort)
	if err != nil {
		return nil, err
	}
	resp, err := rc.RelayMessage(context.Background(), relayMsg)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (wv *webSocketServer) readMessage(reqReadMsg protocol.ReadMessage) (*msg.ResponseReadMessage, error) {
	readMsg := &msg.RequestReadMessage{
		UserId:    reqReadMsg.UserId,
		ChannelId: reqReadMsg.ChannelId,
		MessageId: reqReadMsg.MessageId,
	}

	resp, err := wv.messageClient.GetMessageClient().ReadMessage(context.Background(), readMsg)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (wv *webSocketServer) pushMessage(req *msg.Message, receivers []int64) (*msg.ResponsePushMessage, error) {
	pushMsg := &msg.RequestPushMessage{
		SenderId:        req.SenderId,
		Message:         req,
		ReceiverUserIds: receivers,
	}

	resp, err := wv.messageClient.GetMessageClient().PushMessage(context.Background(), pushMsg)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (wv *webSocketServer) createBotMessage(reqCreateMsg protocol.CreateMessage) (*msg.ResponseBotMessage, error) {
	createBotMsg := &msg.RequestBotMessage{
		SenderId:    reqCreateMsg.SenderId,
		ChannelId:   reqCreateMsg.ChannelId,
		ChannelType: reqCreateMsg.ChannelType,
		Text:        reqCreateMsg.Text,
	}

	resp, err := wv.messageClient.GetMessageClient().CreateBotMessage(context.Background(), createBotMsg)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
