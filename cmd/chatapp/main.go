package main

import (
	"context"
	"net/http"

	"github.com/FrostJ143/WebSocket_Study/internal/ws"
)

func main() {
	hub := ws.NewHub(context.Background())
	handler := ws.NewHandler(hub)
	serveMux := http.NewServeMux()

	serveMux.HandleFunc("POST /ws/createRoom", handler.CreateRoom)
	serveMux.HandleFunc("GET /ws/joinRoom/{roomID}", handler.JoinRoom)
	serveMux.HandleFunc("POST /ws/login", handler.Login)

	// Server TLS
	http.ListenAndServeTLS(":8080", "server.crt", "server.key", serveMux)
}
