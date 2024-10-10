package ws

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

type Handler struct {
	hub *Hub
}

func NewHandler(h *Hub) *Handler {
	return &Handler{
		hub: h,
	}
}

type CreateRoomReq struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *Handler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	// defer r.Body.Close()
	//
	// buf, err := io.ReadAll(r.Body)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	//
	// var req CreateRoomReq
	// err = json.Unmarshal(buf, &req)
	// if err != nil {
	// 	fmt.Println(err)
	// }

	var req CreateRoomReq
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Println(err)
	}

	h.hub.Rooms[req.ID] = &Room{
		ID:      req.ID,
		Name:    req.Name,
		Clients: make(map[string]*Client),
	}

	w.WriteHeader(http.StatusOK)
	buf, err := json.Marshal(req)
	if err != nil {
		fmt.Println(err)
	}
	w.Write(buf)
}

// upgrader upgrades the normal http request to websocket connection
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *Handler) JoinRoom(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	for key := range h.hub.OTPs {
		fmt.Println(key)
	}
	key := r.URL.Query().Get("otp")
	if key == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if !h.hub.OTPs.VerifyOTP(key) {
		fmt.Println("true")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Create websocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
	}

	// Parse path value
	// roomID := r.PathValue("roomID")
	// Parse all query params
	queryParams := r.URL.Query()
	userID := queryParams.Get("userID")
	userName := queryParams.Get("userName")

	client := &Client{
		ID:       userID,
		Username: userName,
		Hub:      h.hub,
		Rooms:    make(map[string]*Room),
		Msg:      make(chan *Message),
		Conn:     conn,
	}

	go client.readMessage()
	go client.writeMessage()

	// h.hub.handleNewClient(conn, userID)
}

type UserLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req UserLoginRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if req.Username == "sang" && req.Password == "123" {
		opt := h.hub.OTPs.NewOTP()

		data, err := json.Marshal(opt)
		if err != nil {
			fmt.Println(err)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
}
