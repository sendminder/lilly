package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"lilly/internal/cache"
	"lilly/internal/config"
	"lilly/internal/protocol"
	"lilly/proto/relay"
)

const (
	ROLE_BEAR   = "bear"
	ROLE_RABBIT = "rabbit"
)

type MatchHandler interface {
	handleRegisterRole(payload json.RawMessage)
}

func (wv *webSocketServer) handleRegisterRole(payload json.RawMessage) {
	var reqRegisterRole protocol.RegisterRole
	jsonErr := json.Unmarshal(payload, &reqRegisterRole)
	if jsonErr != nil {
		slog.Error("Json Error", "error", jsonErr)
		return
	}

	slog.Info("[ReqRegisterRole]", "userId", reqRegisterRole.UserId, "roleType", reqRegisterRole.RoleType)
	err := wv.registerRole(reqRegisterRole)
	if err != nil {
		slog.Error("Failed to read message", "error", err)
		return
	}
}

func (wv *webSocketServer) registerRole(reqRegisterRole protocol.RegisterRole) error {
	// TODO
	// 1. check redis rabbit or bear
	// 2. if exist create channel
	// 2-a. 가져온 유저 세션 확인,
	// 2-b. 나에게 붙어있으면 바로 created 전달, 다른파드면 relay 이벤트 요청
	// 2-c. 요청자에게 created 이벤트 전달,
	// 3. if not exist add to ready queue
	// 3-a. ready 이벤트 전달

	var targetRole = ""
	myRole := reqRegisterRole.RoleType
	myUserId := reqRegisterRole.UserId

	switch myRole {
	case ROLE_BEAR:
		targetRole = ROLE_RABBIT
	case ROLE_RABBIT:
		targetRole = ROLE_RABBIT
	default:
		return errors.New("not found role")
	}

	targetUserId, err := cache.DequeueReadyUser(targetRole)
	if err != nil || targetUserId == 0 {
		wv.readyToUser(myRole, myUserId)
		return nil
	}

	// send created event to userId, myUserId
	// targetUserId
	targetLocation, err := cache.GetUserLocation(targetUserId)
	if err != nil || targetLocation == "" {
		wv.readyToUser(myRole, myUserId)
		return nil
	}

	// create channel
	joinedUsers := []int64{myUserId, targetUserId}
	res, err := wv.createChannel("BareRabbit", myUserId, joinedUsers)
	if err != nil {
		slog.Error("[Error] cant create channel", "myUserId", myUserId)
		return err
	}

	// relay created event
	if targetLocation == config.LocalIP {
		// relay local
		payload := map[string]interface{}{
			"channel_id": res.ChannelId,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			// 오류 처리를 수행하거나 로깅합니다.
			return err
		}
		wv.relayEvent("created_channel", joinedUsers, payloadBytes)
		return nil
	}

	wv.relayCreatedChannel(res.ChannelId, joinedUsers, targetLocation, "50051")

	return nil
}

func (wv *webSocketServer) readyToUser(myRole string, myUserId int64) {
	cache.EnqueueReadyUser(myRole, myUserId)
	// send event "ready"
	wv.relayEvent("ready", []int64{myUserId}, []byte{})
	slog.Info("not found ready user", "userId", myUserId)
}

func (wv *webSocketServer) relayCreatedChannel(channelId int64, joinedUsers []int64, targetIP string, targetPort string) (*relay.ResponseRelayCreatedChannel, error) {
	payload := &relay.CreatedChannelPayload{
		ChannelId: channelId,
	}
	reqCreatedChannel := &relay.RequestRelayCreatedChannel{
		Event:       "created_channel",
		Payload:     payload,
		JoinedUsers: joinedUsers,
	}

	client := wv.relayClient.GetRelayClient(targetIP, targetPort)
	resp, err := client.RelayCreatedChannel(context.Background(), reqCreatedChannel)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (wv *webSocketServer) relayEvent(eventName string, relayLocalUsers []int64, payload []byte) {
	broadcastEvent := protocol.BroadcastEvent{
		Event:       eventName,
		Payload:     payload,
		JoinedUsers: relayLocalUsers,
	}

	wv.broadcaster.Broadcast <- broadcastEvent
}
