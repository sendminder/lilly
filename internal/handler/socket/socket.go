package socket

import (
	"log/slog"

	"github.com/gorilla/websocket"
)

type Socket struct {
	Conn   *websocket.Conn
	Send   chan []byte
	UserId int64
}

func (c Socket) WritePump() {
	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				// 채널이 닫힌 경우
				slog.Error("Error acquired")
				err := c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				if err != nil {
					return
				}
				return
			}

			// 클라이언트로 메시지를 보냅니다.
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				slog.Error("Error writing message", "error", err)
				return
			}
		}
	}
}
