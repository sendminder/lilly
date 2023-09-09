package handler

import (
	"context"
	"encoding/json"
	"errors"
	"lilly/cache"
	"lilly/config"
	"lilly/proto/relay"
	"lilly/protocol"
	"log"
)

const (
	ROLE_BEAR   = "bear"
	ROLE_RABBIT = "rabbit"
)

func HandleRegisterRole(payload json.RawMessage) {
	var reqRegisterRole protocol.RegisterRole
	jsonErr := json.Unmarshal(payload, &reqRegisterRole)
	if jsonErr != nil {
		log.Println("Json Error: ", jsonErr)
		return
	}

	log.Printf("[ReqRegisterRole] UserId: %d, RoleType: %s\n", reqRegisterRole.UserId, reqRegisterRole.RoleType)
	err := registerRole(reqRegisterRole)
	if err != nil {
		log.Printf("Failed to read message: %v", err)
		return
	}
}

func registerRole(reqRegisterRole protocol.RegisterRole) error {
	// TODO
	// 1. check redis rabbit or bear
	// 2. if exist create conversation
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

	// create conversation
	joinedUsers := []int64{myUserId, targetUserId}
	res, err := CreateConversation("BareRabbit", myUserId, joinedUsers)
	if err != nil {
		log.Println("[Error] cant create conversation myUserId: ", myUserId)
		return err
	}

	// relay created event
	if targetLocation == config.LocalIP {
		// relay local
		payload := map[string]interface{}{
			"conversation_id": res.ConversationId,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			// 오류 처리를 수행하거나 로깅합니다.
			return err
		}
		relayEvent("created_conversation", joinedUsers, payloadBytes)
		return nil
	}

	relayCreatedConversation(res.ConversationId, joinedUsers, targetLocation)

	return nil
}

func readyToUser(myRole string, myUserId int64) {
	cache.EnqueueReadyUser(myRole, myUserId)
	// send event "ready"
	relayEvent("ready", []int64{myUserId}, []byte{})
	log.Println("not found ready user, userId:", myUserId)
}

func relayCreatedConversation(convId int64, joinedUsers []int64, targetIP string) (*relay.ResponseRelayCreatedConversation, error) {
	payload := &relay.CreatedConversationPayload{
		ConversationId: convId,
	}
	reqCreatedConv := &relay.RequestRelayCreatedConversation{
		Event:       "created_conversation",
		Payload:     payload,
		JoinedUsers: joinedUsers,
	}

	client := GetRelayClient(targetIP)
	resp, err := client.RelayCreatedConversation(context.Background(), reqCreatedConv)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func relayEvent(eventName string, relayLocalUsers []int64, payload []byte) {
	broadcastEvent := protocol.BroadcastEvent{
		Event:       eventName,
		Payload:     payload,
		JoinedUsers: relayLocalUsers,
	}

	broadcast <- broadcastEvent
}
