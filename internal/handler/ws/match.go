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
	RoleBear   = "bear"
	RoleRabbit = "rabbit"
)

var (
	errNotFoundRole = errors.New("not found role")
	errNotFoundUser = errors.New("not found user")
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

	slog.Info("[ReqRegisterRole]", "userID", reqRegisterRole.UserID, "roleType", reqRegisterRole.RoleType)
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
	myUserID := reqRegisterRole.UserID

	switch myRole {
	case RoleBear:
		targetRole = RoleRabbit
	case RoleRabbit:
		targetRole = RoleBear
	default:
		return errNotFoundRole
	}

	targetUserID, err := cache.DequeueReadyUser(targetRole)
	if err != nil {
		wv.readyToUser(myRole, myUserID)
		return err
	}

	// send created event to userID, myUserID
	targetLocation, err := cache.GetUserLocation(targetUserID)
	if err != nil {
		wv.readyToUser(myRole, myUserID)
		return err
	}
	if targetLocation == "" || targetUserID == 0 {
		wv.readyToUser(myRole, myUserID)
		return errNotFoundUser
	}

	// create channel
	joinedUsers := []int64{myUserID, targetUserID}
	res, err := wv.createChannel("BareRabbit", myUserID, joinedUsers)
	if err != nil {
		slog.Error("[Error] cant create channel", "myUserID", myUserID)
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

	_, err = wv.relayCreatedChannel(res.ChannelId, joinedUsers, targetLocation, "50051")
	if err != nil {
		return err
	}

	return nil
}

func (wv *webSocketServer) readyToUser(myRole string, myUserID int64) {
	cache.EnqueueReadyUser(myRole, myUserID)
	// send event "ready"
	wv.relayEvent("ready", []int64{myUserID}, []byte{})
	slog.Info("not found ready user", "userID", myUserID)
}

func (wv *webSocketServer) relayCreatedChannel(channelID int64, joinedUsers []int64, targetIP string, targetPort string) (*relay.ResponseRelayCreatedChannel, error) {
	payload := &relay.CreatedChannelPayload{
		ChannelId: channelID,
	}
	reqCreatedChannel := &relay.RequestRelayCreatedChannel{
		Event:       "created_channel",
		Payload:     payload,
		JoinedUsers: joinedUsers,
	}

	client, err := wv.relayClient.GetRelayClient(targetIP, targetPort)
	if err != nil {
		return nil, err
	}
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
