package websocket

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins - configure in production
	},
}

// HandleConnection upgrades HTTP connection to WebSocket
func HandleConnection(hub *Hub, w http.ResponseWriter, r *http.Request, userID string) error {
	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	// Create client
	client := NewClient(hub, conn, userID)

	// Register client
	hub.Register <- client

	// Start read/write pumps
	go client.WritePump()
	go client.ReadPump()

	return nil
}

// HandleConnectionWithConfig upgrades with custom upgrader config
func HandleConnectionWithConfig(hub *Hub, w http.ResponseWriter, r *http.Request, userID string, config *Config) error {
	customUpgrader := websocket.Upgrader{
		ReadBufferSize:  config.ReadBufferSize,
		WriteBufferSize: config.WriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := customUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	client := NewClient(hub, conn, userID)
	hub.Register <- client

	go client.WritePump()
	go client.ReadPump()

	return nil
}
