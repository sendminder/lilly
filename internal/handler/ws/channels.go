package ws

import (
	"encoding/json"
	"log/slog"

	"lilly/internal/protocol"
	msg "lilly/proto/message"
)

type ChannelHandler interface {
	handleDecryptChannel(payload json.RawMessage)
	handleFinishChannel(payload json.RawMessage)
	createChannel(name string, userID int64, joinedUsers []int64) (*msg.ResponseCreateChannel, error)
}

func (wv *webSocketServer) handleDecryptChannel(payload json.RawMessage) {
	var reqDecryptChannel protocol.DecryptChannel
	err := json.Unmarshal(payload, &reqDecryptChannel)
	if err != nil {
		slog.Error("Json Error", "error", err)
		return
	}
	// 받은 메시지를 출력
	slog.Info("[ReqDecryptChannel]", "channelID", reqDecryptChannel.ChannelID)
	resp, err := wv.decryptChannel(reqDecryptChannel)
	if err != nil {
		slog.Error("Failed to decrypt channel", "error", err)
		return
	}
	slog.Info("Decrypted Channel", "resp", resp)

	// TODO: decrypted_channel relay
	// decrypted_channel 받으면, 클라에서 복호화 하기
}

func (wv *webSocketServer) handleFinishChannel(payload json.RawMessage) {
	var reqFinishChannel protocol.FinishChannel
	err := json.Unmarshal(payload, &reqFinishChannel)
	if err != nil {
		slog.Error("Json Error", "error", err)
		return
	}

	slog.Info("[ReqFinishChannel]", "channelID", reqFinishChannel.ChannelID)
	resp, err := wv.finishChannel(reqFinishChannel)
	if err != nil {
		slog.Error("Failed to finish channel", "error", err)
		return
	}
	slog.Info("Finished Channel", "resp", resp)

	// TODO: finish_channel relay
	// finish_channel 받으면, 클라에서 deactivated 처리
}

func (wv *webSocketServer) createChannel(name string, userID int64, joinedUsers []int64) (*msg.ResponseCreateChannel, error) {
	createChannel := &msg.RequestCreateChannel{
		Name:        name,
		HostUserId:  userID,
		JoinedUsers: joinedUsers,
	}

	resp, err := wv.messageClient.GetMessageClient().CreateChannel(wv.ctx, createChannel)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (wv *webSocketServer) decryptChannel(reqDecryptChannel protocol.DecryptChannel) (*msg.ResponseDecryptChannel, error) {
	decryptChannel := &msg.RequestDecryptChannel{
		ChannelId: reqDecryptChannel.ChannelID,
	}

	resp, err := wv.messageClient.GetMessageClient().DecryptChannel(wv.ctx, decryptChannel)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (wv *webSocketServer) finishChannel(reqFinishChannel protocol.FinishChannel) (*msg.ResponseFinishChannel, error) {
	finishChannel := &msg.RequestFinishChannel{
		ChannelId: reqFinishChannel.ChannelID,
	}

	resp, err := wv.messageClient.GetMessageClient().FinishChannel(wv.ctx, finishChannel)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
