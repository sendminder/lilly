package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"

	"lilly/internal/protocol"
	msg "lilly/proto/message"
)

func HandleDecryptChannel(payload json.RawMessage) {
	var reqDecryptChannel protocol.DecryptChannel
	err := json.Unmarshal(payload, &reqDecryptChannel)
	if err != nil {
		slog.Error("Json Error", "error", err)
		return
	}
	// 받은 메시지를 출력
	slog.Info("[ReqDecryptChannel]", "channelId", reqDecryptChannel.ChannelId)
	resp, err := decryptChannel(reqDecryptChannel)
	if err != nil {
		slog.Error("Failed to decrypt channel", "error", err)
		return
	}
	slog.Info("Decrypted Channel", "resp", resp)

	// TODO: decrypted_channel relay
	// decrypted_channel 받으면, 클라에서 복호화 하기
}

func HandleFinishChannel(payload json.RawMessage) {
	var reqFinishChannel protocol.FinishChannel
	err := json.Unmarshal(payload, &reqFinishChannel)
	if err != nil {
		slog.Error("Json Error", "error", err)
		return
	}

	slog.Info("[ReqFinishChannel]", "channelId", reqFinishChannel.ChannelId)
	resp, err := finishChannel(reqFinishChannel)
	if err != nil {
		slog.Error("Failed to finish channel", "error", err)
		return
	}
	slog.Info("Finished Channel", "resp", resp)

	// TODO: finish_channel relay
	// finish_channel 받으면, 클라에서 deactivated 처리
}

func decryptChannel(reqDecryptChannel protocol.DecryptChannel) (*msg.ResponseDecryptChannel, error) {
	decryptChannel := &msg.RequestDecryptChannel{
		ChannelId: reqDecryptChannel.ChannelId,
	}

	randIdx := rand.Intn(10)
	mutexes[randIdx].Lock()
	resp, err := messageClient[randIdx].DecryptChannel(context.Background(), decryptChannel)
	mutexes[randIdx].Unlock()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func finishChannel(reqFinishChannel protocol.FinishChannel) (*msg.ResponseFinishChannel, error) {
	finishChannel := &msg.RequestFinishChannel{
		ChannelId: reqFinishChannel.ChannelId,
	}

	randIdx := rand.Intn(10)
	mutexes[randIdx].Lock()
	resp, err := messageClient[randIdx].FinishChannel(context.Background(), finishChannel)
	mutexes[randIdx].Unlock()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func CreateChannel(name string, userId int64, joinedUsers []int64) (*msg.ResponseCreateChannel, error) {
	createChannel := &msg.RequestCreateChannel{
		Name:        name,
		HostUserId:  userId,
		JoinedUsers: joinedUsers,
	}

	randIdx := rand.Intn(10)
	mutexes[randIdx].Lock()
	resp, err := messageClient[randIdx].CreateChannel(context.Background(), createChannel)
	mutexes[randIdx].Unlock()
	if err != nil {
		return nil, err
	}
	return resp, nil
}