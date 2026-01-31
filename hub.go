package websocket

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"sync"
)

// Hub manages WebSocket connections and rooms
type Hub struct {
	config *Config

	// Client management
	clients   map[string]*Client
	clientsMu sync.RWMutex

	// Room management
	rooms   map[string]*Room
	roomsMu sync.RWMutex

	// Channels
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan Message

	// Optional cache (go-cache)
	cache interface{}

	// Middleware hooks
	onConnect    func(*Client)
	onDisconnect func(*Client)
	onMessage    func(*Client, Message)
}

// NewHub creates a new WebSocket hub
func NewHub(config *Config) *Hub {
	if config == nil {
		config = DefaultConfig()
	}

	return &Hub{
		config:     config,
		clients:    make(map[string]*Client),
		rooms:      make(map[string]*Room),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan Message),
		cache:      config.Cache,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.Unregister:
			h.unregisterClient(client)

		case message := <-h.Broadcast:
			h.broadcastMessage(message)
		}
	}
}

// registerClient registers a new client
func (h *Hub) registerClient(client *Client) {
	h.clientsMu.Lock()
	h.clients[client.UserID] = client
	h.clientsMu.Unlock()

	log.Printf("Client connected: %s", client.UserID)

	// Call onConnect hook
	if h.onConnect != nil {
		h.onConnect(client)
	}

	// Update cache if available
	if h.cache != nil {
		// Increment online count (assuming go-cache interface)
		// h.cache.Increment("ws:stats:online_users", 1)
	}
}

// unregisterClient removes a client and cleans up
func (h *Hub) unregisterClient(client *Client) {
	h.clientsMu.Lock()
	if _, ok := h.clients[client.UserID]; ok {
		delete(h.clients, client.UserID)
		close(client.Send)
	}
	h.clientsMu.Unlock()

	// Remove from all rooms
	h.LeaveAllRooms(client.UserID)

	log.Printf("Client disconnected: %s", client.UserID)

	// Call onDisconnect hook
	if h.onDisconnect != nil {
		h.onDisconnect(client)
	}

	// Update cache if available
	if h.cache != nil {
		// Decrement online count
		// h.cache.Increment("ws:stats:online_users", -1)
	}
}

// broadcastMessage sends message to all connected clients
func (h *Hub) broadcastMessage(message Message) {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	for _, client := range h.clients {
		client.SendMessage(message)
	}
}

// BroadcastToAll broadcasts a message to all connected clients
func (h *Hub) BroadcastToAll(msg Message) {
	h.Broadcast <- msg
}

// SendToUser sends a message to a specific user
func (h *Hub) SendToUser(userID string, msg Message) error {
	h.clientsMu.RLock()
	client, ok := h.clients[userID]
	h.clientsMu.RUnlock()

	if !ok {
		return errors.New("user not connected")
	}

	client.SendMessage(msg)
	return nil
}

// GetClient returns a client by user ID
func (h *Hub) GetClient(userID string) *Client {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	return h.clients[userID]
}

// GetOnlineCount returns the number of connected clients
func (h *Hub) GetOnlineCount() int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	return len(h.clients)
}

// GetOnlineUsers returns list of connected user IDs
func (h *Hub) GetOnlineUsers() []string {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	users := make([]string, 0, len(h.clients))
	for userID := range h.clients {
		users = append(users, userID)
	}
	return users
}

// HandleMessage processes incoming messages
func (h *Hub) HandleMessage(client *Client, msg Message) {
	// Call onMessage hook
	if h.onMessage != nil {
		h.onMessage(client, msg)
	}

	// Default message handling can be added here
	log.Printf("Message from %s: type=%s", client.UserID, msg.Type)
}

// SetOnConnect sets the onConnect hook
func (h *Hub) SetOnConnect(fn func(*Client)) {
	h.onConnect = fn
}

// SetOnDisconnect sets the onDisconnect hook
func (h *Hub) SetOnDisconnect(fn func(*Client)) {
	h.onDisconnect = fn
}

// SetOnMessage sets the onMessage hook
func (h *Hub) SetOnMessage(fn func(*Client, Message)) {
	h.onMessage = fn
}

// generateID generates a random ID
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Helper function to generate room IDs
func generateRoomID() string {
	return "room_" + generateID()[:12]
}
