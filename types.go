package websocket

import (
	"sync"
	"time"
)

// Message represents a WebSocket message
type Message struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// Room represents a chat room or channel
type Room struct {
	ID         string
	Name       string
	Clients    map[string]*Client
	MaxClients int
	IsPrivate  bool
	Password   string
	CreatedAt  time.Time
	CreatedBy  string
	Metadata   map[string]interface{}
	mu         sync.RWMutex
}

// RoomConfig contains configuration for creating a room
type RoomConfig struct {
	Name       string
	MaxClients int                    // 0 = unlimited
	IsPrivate  bool                   // false = public
	Password   string                 // for private rooms
	Metadata   map[string]interface{} // custom data
}

// RoomInfo represents public room information
type RoomInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	ClientCount int                    `json:"client_count"`
	MaxClients  int                    `json:"max_clients"`
	IsPrivate   bool                   `json:"is_private"`
	CreatedAt   time.Time              `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Config contains WebSocket server configuration
type Config struct {
	// WebSocket settings
	ReadBufferSize  int
	WriteBufferSize int
	PingInterval    time.Duration
	PongWait        time.Duration
	WriteWait       time.Duration
	MaxMessageSize  int64

	// Optional cache for distributed mode (from go-cache)
	Cache interface{} // *cache.Cache - interface to avoid hard dependency
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		PingInterval:    30 * time.Second,
		PongWait:        60 * time.Second,
		WriteWait:       10 * time.Second,
		MaxMessageSize:  512 * 1024, // 512KB
		Cache:           nil,
	}
}

// GetClientCount returns the number of clients in a room
func (r *Room) GetClientCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Clients)
}

// IsFull returns true if room has reached max capacity
func (r *Room) IsFull() bool {
	if r.MaxClients == 0 {
		return false // unlimited
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Clients) >= r.MaxClients
}

// ToInfo converts Room to RoomInfo (public data)
func (r *Room) ToInfo() *RoomInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return &RoomInfo{
		ID:          r.ID,
		Name:        r.Name,
		ClientCount: len(r.Clients),
		MaxClients:  r.MaxClients,
		IsPrivate:   r.IsPrivate,
		CreatedAt:   r.CreatedAt,
		Metadata:    r.Metadata,
	}
}
