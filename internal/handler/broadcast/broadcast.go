package broadcast

import (
	"lilly/internal/handler/socket"
	"lilly/internal/protocol"
)

type Broadcaster struct {
	Broadcast  chan protocol.BroadcastEvent
	Register   chan *socket.Socket
	Unregister chan *socket.Socket
}

func NewBroadcaster() Broadcaster {
	broadcast := make(chan protocol.BroadcastEvent)
	register := make(chan *socket.Socket)
	unregister := make(chan *socket.Socket)
	return Broadcaster{
		Broadcast:  broadcast,
		Register:   register,
		Unregister: unregister,
	}
}
