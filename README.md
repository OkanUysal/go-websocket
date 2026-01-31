# Go WebSocket

A powerful, production-ready WebSocket library for Go with room management, broadcasting, and optional cache support. Built on top of [gorilla/websocket](https://github.com/gorilla/websocket).

## Features

- **Hub Pattern**: Centralized connection management
- **Room Management**: Create, join, leave, and close rooms with ease
- **User-based Messaging**: Send messages to specific users
- **Broadcasting**: Broadcast to all users or specific rooms
- **Middleware Hooks**: OnConnect, OnDisconnect, OnMessage callbacks
- **Ping/Pong**: Built-in connection health checks
- **Optional Cache**: Integrate with [go-cache](https://github.com/OkanUysal/go-cache) for distributed scenarios
- **Type-safe Messages**: Structured message handling
- **Production Ready**: Graceful shutdown, rate limiting ready

## Installation

```bash
go get github.com/OkanUysal/go-websocket
```

## Quick Start

### 1. Create Hub and Start Server

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/OkanUysal/go-websocket"
)

func main() {
    // Create WebSocket hub
    hub := websocket.NewHub(nil) // nil = use default config
    go hub.Run() // Start hub in background
    
    // Setup Gin
    r := gin.Default()
    
    // WebSocket endpoint
    r.GET("/ws", func(c *gin.Context) {
        userID := c.Query("user_id") // or get from JWT token
        
        // Upgrade to WebSocket
        err := websocket.HandleConnection(hub, c.Writer, c.Request, userID)
        if err != nil {
            log.Printf("WebSocket error: %v", err)
        }
    })
    
    r.Run(":8080")
}
```

### 2. Client Connection (JavaScript)

```javascript
const ws = new WebSocket('ws://localhost:8080/ws?user_id=user123');

ws.onopen = () => {
    console.log('Connected to WebSocket server');
};

ws.onmessage = (event) => {
    const message = JSON.parse(event.data);
    console.log('Received:', message);
    
    switch(message.type) {
        case 'user_joined':
            console.log('User joined room:', message.data);
            break;
        case 'message':
            console.log('New message:', message.data);
            break;
    }
};

// Send message
function sendMessage(type, data) {
    ws.send(JSON.stringify({ type, data }));
}

sendMessage('chat_message', { text: 'Hello!' });
```

## Usage Examples

### Broadcasting

```go
// Broadcast to all connected users
hub.BroadcastToAll(websocket.Message{
    Type: "announcement",
    Data: map[string]interface{}{
        "text": "System maintenance in 5 minutes",
    },
})

// Send to specific user
hub.SendToUser("user123", websocket.Message{
    Type: "notification",
    Data: map[string]interface{}{
        "title": "New message",
        "body":  "You have a new message",
    },
})
```

### Room Management

```go
// Create a room
roomID := hub.CreateRoom(&websocket.RoomConfig{
    Name:       "Game Room #1",
    MaxClients: 10,           // 0 = unlimited
    IsPrivate:  false,
    Metadata: map[string]interface{}{
        "game_type": "ranked",
        "map":       "desert",
    },
})

// Join room
err := hub.JoinRoom("user123", roomID)
if err != nil {
    log.Printf("Failed to join room: %v", err)
}

// Broadcast to room
hub.BroadcastToRoom(roomID, websocket.Message{
    Type: "room_message",
    Data: map[string]interface{}{
        "user":    "user123",
        "message": "Hello room!",
    },
})

// Leave room
hub.LeaveRoom("user123", roomID)

// Close room (removes all clients)
hub.CloseRoom(roomID)
```

### Room Queries

```go
// Check if room exists
if hub.RoomExists(roomID) {
    // ...
}

// Check if room is full
if hub.IsRoomFull(roomID) {
    log.Println("Room is full!")
}

// Get room client count
count := hub.GetRoomClientCount(roomID)

// Get list of users in room
users := hub.GetRoomClients(roomID)

// Get all rooms a user is in
userRooms := hub.GetUserRooms("user123")

// List all public rooms
publicRooms := hub.ListRooms()
for _, room := range publicRooms {
    log.Printf("Room: %s (%d/%d users)", 
        room.Name, room.ClientCount, room.MaxClients)
}
```

### Middleware Hooks

```go
hub := websocket.NewHub(nil)

// OnConnect callback
hub.SetOnConnect(func(client *websocket.Client) {
    log.Printf("User connected: %s", client.UserID)
    
    // Send welcome message
    client.SendMessage(websocket.Message{
        Type: "welcome",
        Data: map[string]interface{}{
            "message": "Welcome to the server!",
        },
    })
})

// OnDisconnect callback
hub.SetOnDisconnect(func(client *websocket.Client) {
    log.Printf("User disconnected: %s", client.UserID)
    
    // Cleanup, save state, etc.
})

// OnMessage callback
hub.SetOnMessage(func(client *websocket.Client, msg websocket.Message) {
    log.Printf("Message from %s: %s", client.UserID, msg.Type)
    
    // Custom message routing
    switch msg.Type {
    case "chat_message":
        // Handle chat message
        roomID := msg.Data["room_id"].(string)
        hub.BroadcastToRoom(roomID, msg)
        
    case "private_message":
        // Handle private message
        targetUser := msg.Data["to"].(string)
        hub.SendToUser(targetUser, msg)
    }
})
```

### Custom Configuration

```go
config := &websocket.Config{
    ReadBufferSize:  2048,
    WriteBufferSize: 2048,
    PingInterval:    30 * time.Second,
    PongWait:        60 * time.Second,
    WriteWait:       10 * time.Second,
    MaxMessageSize:  1024 * 1024, // 1MB
}

hub := websocket.NewHub(config)
```

### With Cache (Optional)

For distributed scenarios with multiple servers:

```go
import (
    "github.com/OkanUysal/go-cache"
    "github.com/OkanUysal/go-websocket"
)

// Setup Redis cache
cache := cache.NewCache(&cache.Config{
    Type: cache.RedisCache,
    Redis: cache.RedisConfig{
        Host: "localhost:6379",
    },
})

// Pass cache to WebSocket hub
hub := websocket.NewHub(&websocket.Config{
    Cache: cache,
})

// Now room metadata, online counts, etc. can be shared across servers
```

Or use in-memory cache:

```go
cache := cache.NewCache(&cache.Config{
    Type: cache.MemoryCache,
})

hub := websocket.NewHub(&websocket.Config{
    Cache: cache,
})
```

### Complete Example - Game Match

```go
// Create match room
matchID := "match_12345"
roomID := hub.CreateRoom(&websocket.RoomConfig{
    Name:       "Ranked Match",
    MaxClients: 2, // 1v1
    Metadata: map[string]interface{}{
        "match_id": matchID,
        "type":     "ranked",
    },
})

// Players join
hub.JoinRoom("player1", roomID)
hub.JoinRoom("player2", roomID)

// Game event
hub.BroadcastToRoom(roomID, websocket.Message{
    Type: "game_event",
    Data: map[string]interface{}{
        "event":  "goal",
        "player": "player1",
        "score":  map[string]int{"player1": 1, "player2": 0},
    },
})

// Match ends, close room
hub.CloseRoom(roomID)
```

### Live Leaderboard Example

```go
// Create global leaderboard room
leaderboardRoom := hub.CreateRoom(&websocket.RoomConfig{
    Name:       "Global Leaderboard",
    MaxClients: 0, // unlimited
})

// Users join leaderboard
hub.JoinRoom(userID, leaderboardRoom)

// When score updates
hub.BroadcastToRoom(leaderboardRoom, websocket.Message{
    Type: "leaderboard_update",
    Data: map[string]interface{}{
        "rankings": updatedRankings,
    },
})
```

### Kicking Users

```go
// Kick user from room with reason
err := hub.KickFromRoom("user123", roomID, "Spam detected")
if err != nil {
    log.Printf("Failed to kick user: %v", err)
}

// Remove user from all rooms
hub.LeaveAllRooms("user123")
```

## API Reference

### Hub Methods

#### Connection Management
- `NewHub(config *Config) *Hub` - Create new hub
- `Run()` - Start hub main loop (call in goroutine)
- `GetOnlineCount() int` - Get total connected users
- `GetOnlineUsers() []string` - Get list of connected user IDs
- `GetClient(userID string) *Client` - Get client by user ID

#### Broadcasting
- `BroadcastToAll(msg Message)` - Send to all connected users
- `BroadcastToRoom(roomID string, msg Message)` - Send to room members
- `SendToUser(userID string, msg Message) error` - Send to specific user

#### Room Management
- `CreateRoom(config *RoomConfig) string` - Create and return room ID
- `JoinRoom(userID, roomID string) error` - Add user to room
- `LeaveRoom(userID, roomID string) error` - Remove user from room
- `LeaveAllRooms(userID string)` - Remove user from all rooms
- `CloseRoom(roomID string)` - Close room and remove all users
- `KickFromRoom(userID, roomID, reason string) error` - Kick user

#### Room Queries
- `GetRoom(roomID string) *Room` - Get room by ID
- `RoomExists(roomID string) bool` - Check if room exists
- `IsRoomFull(roomID string) bool` - Check if room is at capacity
- `GetRoomClientCount(roomID string) int` - Get user count in room
- `GetRoomClients(roomID string) []string` - Get users in room
- `GetUserRooms(userID string) []string` - Get rooms user is in
- `ListRooms() []*RoomInfo` - Get all public rooms

#### Middleware
- `SetOnConnect(fn func(*Client))` - Set connect callback
- `SetOnDisconnect(fn func(*Client))` - Set disconnect callback
- `SetOnMessage(fn func(*Client, Message))` - Set message callback

### Handler

- `HandleConnection(hub *Hub, w http.ResponseWriter, r *http.Request, userID string) error` - Upgrade HTTP to WebSocket

## Types

### Message
```go
type Message struct {
    Type string                 `json:"type"`
    Data map[string]interface{} `json:"data"`
}
```

### RoomConfig
```go
type RoomConfig struct {
    Name       string
    MaxClients int                    // 0 = unlimited
    IsPrivate  bool
    Password   string
    Metadata   map[string]interface{}
}
```

### Config
```go
type Config struct {
    ReadBufferSize  int
    WriteBufferSize int
    PingInterval    time.Duration
    PongWait        time.Duration
    WriteWait       time.Duration
    MaxMessageSize  int64
    Cache           interface{} // *cache.Cache
}
```

## Default Configuration

- **ReadBufferSize**: 1024 bytes
- **WriteBufferSize**: 1024 bytes
- **PingInterval**: 30 seconds
- **PongWait**: 60 seconds
- **WriteWait**: 10 seconds
- **MaxMessageSize**: 512 KB
- **Cache**: nil (disabled)

## Use Cases

- **Real-time Chat**: Group chats, private messages
- **Live Gaming**: Match updates, leaderboards
- **Collaborative Tools**: Real-time editing, whiteboarding
- **Live Dashboards**: Analytics, monitoring
- **Notifications**: Push notifications, alerts
- **IoT**: Sensor data streaming

## Requirements

- Go 1.21 or higher
- [gorilla/websocket](https://github.com/gorilla/websocket) v1.5.0+

## Optional Dependencies

- [go-cache](https://github.com/OkanUysal/go-cache) - For distributed mode

## License

MIT License

## Author

Okan Uysal - [GitHub](https://github.com/OkanUysal)
