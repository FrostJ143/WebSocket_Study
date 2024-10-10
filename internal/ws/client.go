package ws

import (
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// In WebSocket connections, the ping/pong mechanism is used to ensure that the connection between the client and server is still alive and responsive.
// This is crucial because WebSocket connections can be long-lived, and either side (client or server) may become unresponsive without closing the connection properly.

// The ping is sent from one side (typically the server) to check if the other side (the client) is still active.
// The client must respond with a pong message within a certain time limit (often called the pong wait time or pong timeout).
var (
	pongWait = 10 * time.Second
	// pingInterval must faster than pongWait so that in one pongWait window, ping can be sent multiple time
	// this gonna prevents network latency and unstable connection make ping message does not reach client
	pingInterval = (pongWait * 9) / 10
)

type Client struct {
	Conn  *websocket.Conn
	Hub   *Hub
	Rooms map[string]*Room
	// Msg used to safely write concurrently to websocket.Conn
	Msg      chan *Message
	ID       string `json:"id"`
	Username string `json:"username"`
}

type Message struct {
	Timestamp time.Time `json:"time_stamp"`
	Content   string    `json:"content"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	ID        string    `json:"id"`
}

func (cl *Client) readMessage() {
	defer func() {
		cl.cleanUpConn()
	}()

	// After duration, if no read, close the webosocket
	err := cl.Conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		fmt.Println(err)
	}

	// SetReadLimit sets maximum bytes of messages that client can send to server, if >= limit, the connection is closed
	cl.Conn.SetReadLimit(512)

	cl.Conn.SetPongHandler(func(pongMsg string) error {
		fmt.Println("pong")
		err := cl.Conn.SetReadDeadline(time.Now().Add(pongWait))

		return err
	})

	for {
		var msg Message
		err := cl.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("error reading message: %v", err)
			}
			break
		}

		cl.Hub.processMessage(&msg)
	}
}

func (cl *Client) writeMessage() {
	defer func() {
		cl.cleanUpConn()
	}()

	ticker := time.NewTicker(pingInterval)
	fmt.Println("ping")
	if err := cl.Conn.WriteMessage(websocket.PingMessage, []byte("")); err != nil {
		fmt.Println(err)
	}

	for {
		select {
		case msg, ok := <-cl.Msg:
			if !ok {
				if err := cl.Conn.WriteMessage(websocket.CloseMessage, nil); err != nil {
					fmt.Println("connection closed: ", err)
				}

				return
			}

			if err := cl.Conn.WriteJSON(msg); err != nil {
				fmt.Println("failed to send message: ", err)
			}
		case <-ticker.C:
			fmt.Println("ping")
			if err := cl.Conn.WriteMessage(websocket.PingMessage, []byte("")); err != nil {
				fmt.Println(err)
			}
		}
	}
}

func (cl *Client) cleanUpConn() {
	// Step 1: Close the WebSocket connection to release the network resources
	err := cl.Conn.Close()
	if err != nil {
		log.Println("Error closing WebSocket:", err)
	}

	// Step 2: Remove the client from all rooms it is part of
	for _, room := range cl.Rooms {
		room.Mutex.Lock()
		delete(room.Clients, cl.ID)
		room.Mutex.Unlock()
	}

	// Step 3: Close the send channel to stop any message sending routines
	close(cl.Msg)

	// Step 4: Clear references to prevent memory leaks
	cl.Rooms = nil
	cl.Conn = nil
}
