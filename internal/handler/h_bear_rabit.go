package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"lilly/internal/cache"
	"lilly/internal/config"
	protocol2 "lilly/internal/protocol"
	"lilly/proto/relay"
)

const (
	ROLE_BEAR   = "bear"
	ROLE_RABBIT = "rabbit"
)

func HandleRegisterRole(payload json.RawMessage) {
	var reqRegisterRole protocol2.RegisterRole
	jsonErr := json.Unmarshal(payload, &reqRegisterRole)
	if jsonErr != nil {
		slog.Error("Json Error", "error", jsonErr)
		return
	}

	slog.Info("[ReqRegisterRole]", "userId", reqRegisterRole.UserId, "roleType", reqRegisterRole.RoleType)
	err := registerRole(reqRegisterRole)
	if err != nil {
		slog.Error("Failed to read message", "error", err)
		return
	}
}

func registerRole(reqRegisterRole protocol2.RegisterRole) error {
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
		readyToUser(myRole, myUserId)
		return nil
	}

	// send created event to userId, myUserId
	// targetUserId
	targetLocation, err := cache.GetUserLocation(targetUserId)
	if err != nil || targetLocation == "" {
		readyToUser(myRole, myUserId)
		return nil
	}

	// create channel
	joinedUsers := []int64{myUserId, targetUserId}
	res, err := CreateChannel("BareRabbit", myUserId, joinedUsers)
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
		relayEvent("created_channel", joinedUsers, payloadBytes)
		return nil
	}

	relayCreatedChannel(res.ChannelId, joinedUsers, targetLocation)

	return nil
}

func readyToUser(myRole string, myUserId int64) {
	cache.EnqueueReadyUser(myRole, myUserId)
	// send event "ready"
	relayEvent("ready", []int64{myUserId}, []byte{})
	slog.Info("not found ready user", "userId", myUserId)
}

func relayCreatedChannel(channelId int64, joinedUsers []int64, targetIP string) (*relay.ResponseRelayCreatedChannel, error) {
	payload := &relay.CreatedChannelPayload{
		ChannelId: channelId,
	}
	reqCreatedChannel := &relay.RequestRelayCreatedChannel{
		Event:       "created_channel",
		Payload:     payload,
		JoinedUsers: joinedUsers,
	}

	client := GetRelayClient(targetIP)
	resp, err := client.RelayCreatedChannel(context.Background(), reqCreatedChannel)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func relayEvent(eventName string, relayLocalUsers []int64, payload []byte) {
	broadcastEvent := protocol2.BroadcastEvent{
		Event:       eventName,
		Payload:     payload,
		JoinedUsers: relayLocalUsers,
	}

	broadcast <- broadcastEvent
}
