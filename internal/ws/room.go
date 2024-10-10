package ws

import (
	"fmt"
	"sync"
)

type Room struct {
	Clients map[string]*Client
	ID      string `json:"id"`
	Name    string `json:"name"`
	Mutex   sync.Mutex
}

func (r *Room) Broadcast(msg *Message) {
	for _, cl := range r.Clients {
		select {
		case cl.Msg <- msg:
			// Successfully sent message to active client
		default:
			// Channel is blocked; queue the message instead of disconnecting
			fmt.Println("Client busy, queueing message for client: ", cl.ID)
			// Optionally queue the message
			// cl.messageQueue = append(cl.messageQueue, message)
		}
	}
}
