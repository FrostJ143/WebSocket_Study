package ws

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

type Hub struct {
	DB    *sql.DB
	Rooms map[string]*Room
	OTPs  RetentionMap

	Register   chan *RegisterRequest
	Unregister chan *RegisterRequest
	Broadcast  chan *Message
}

type RegisterRequest struct {
	Client *Client
	Room   *Room
}

func NewHub(ctx context.Context) *Hub {
	return &Hub{
		Rooms: make(map[string]*Room),
		OTPs:  NewRetentionMap(ctx, 20*time.Second),

		// Register used to safely write concurrently to map (Room.Clients and Clients.Rooms)
		Register: make(chan *RegisterRequest),
		// Unregister used to safely write concurrently to map (Room.Clients and Clients.Rooms)
		Unregister: make(chan *RegisterRequest),
		Broadcast:  make(chan *Message),
	}
}

func (h *Hub) Run() {
	go func() {
		for {
			select {
			case req := <-h.Register:
				if r, ok := h.Rooms[req.Room.ID]; ok {
					if cl, ok := r.Clients[req.Client.ID]; !ok {
						r.Clients[req.Client.ID] = req.Client
						cl.Rooms[r.ID] = r
					}
				}
			case req := <-h.Unregister:
				if r, ok := h.Rooms[req.Room.ID]; ok {
					if cl, ok := r.Clients[req.Client.ID]; ok {
						// Broadcast a message saying that the client has left the room
						if len(r.Clients) >= 2 {
							h.Broadcast <- &Message{
								Content: "user has left the chat",
								RoomID:  r.ID,
								UserID:  cl.ID,
							}
						}

						delete(cl.Rooms, r.ID)
						delete(r.Clients, cl.ID)
					}
				}
			case msg := <-h.Broadcast:
				if r, ok := h.Rooms[msg.RoomID]; ok {
					r.Broadcast(msg)
				}
			}
		}
	}()
}

func (h *Hub) storeMessageInDB(msg *Message) error {
	// SQL to insert message into the database
	query := "INSERT INTO messages (room_id, user_id, content) VALUES ($1, $2, $3)"
	_, err := h.DB.Exec(query, msg.RoomID, msg.UserID, msg.Content)
	return err
}

func (h *Hub) processMessage(msg *Message) {
	// Step 1: Store the message in the database
	err := h.storeMessageInDB(msg)
	if err != nil {
		fmt.Println("Error storing message: ", err)
		return
	}

	// Step 2: Get the room to broadcast the message to active clients
	room, ok := h.Rooms[msg.RoomID]
	if !ok {
		fmt.Println("Room not found: ", err)
		return
	}

	// Step 3: Broadcast the message to all active clients in the room
	room.Broadcast(msg)
}

// handleNewClient queries all rooms that client is a part of and add that to room's Client map
// to notify that this client is now active so that server can Broadcast message to this client
func (h *Hub) handleNewClient(conn *websocket.Conn, userID string) {
	// Create new client object
	client := &Client{
		ID:    userID,
		Conn:  conn,
		Rooms: make(map[string]*Room),
		Msg:   make(chan *Message),
		Hub:   h,
	}

	// Step 1: Query database to get all the rooms the user is part of
	rows, err := h.DB.Query("SELECT room_id FROM room_memberships WHERE user_id = $1", userID)
	if err != nil {
		// Handle error
	}

	// Step 2: For each room, add the client to the room's Clients map
	for rows.Next() {
		var roomID string
		err = rows.Scan(&roomID)
		if err != nil {
			// Handle error
		}

		// Get or create the room
		room, ok := h.Rooms[roomID]
		if !ok {
			// Room doesn't exist, create a new one
			room = &Room{
				Clients: make(map[string]*Client),
				ID:      roomID,
			}
			h.Rooms[roomID] = room
		}

		// Add the client to the room's Clients map
		room.Clients[client.ID] = client
		// Add the room to the client's Rooms map
		client.Rooms[roomID] = room

	}

	// Step 3: Load past messages
	for _, room := range client.Rooms {
		msg := GetMessages(h, room.ID, 50)
		fmt.Println(msg)
		// Send messages to client
	}

	// Step 4: Start listening and writing messages
	go client.readMessage()
	go client.writeMessage()
}
