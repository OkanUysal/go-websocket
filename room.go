package websocket

import (
	"errors"
	"log"
	"time"
)

// CreateRoom creates a new room
func (h *Hub) CreateRoom(config *RoomConfig) string {
	roomID := generateRoomID()

	room := &Room{
		ID:         roomID,
		Name:       config.Name,
		Clients:    make(map[string]*Client),
		MaxClients: config.MaxClients,
		IsPrivate:  config.IsPrivate,
		Password:   config.Password,
		CreatedAt:  time.Now(),
		Metadata:   config.Metadata,
	}

	h.roomsMu.Lock()
	h.rooms[roomID] = room
	h.roomsMu.Unlock()

	log.Printf("Room created: %s (%s)", roomID, config.Name)

	// Cache room metadata if cache is available
	if h.cache != nil {
		// h.cache.Set("ws:room:"+roomID, room, 24*time.Hour)
		// h.cache.Increment("ws:stats:total_rooms", 1)
	}

	return roomID
}

// JoinRoom adds a client to a room
func (h *Hub) JoinRoom(userID, roomID string) error {
	h.roomsMu.Lock()
	room, exists := h.rooms[roomID]
	h.roomsMu.Unlock()

	if !exists {
		return errors.New("room not found")
	}

	// Check if room is full
	if room.IsFull() {
		return errors.New("room is full")
	}

	h.clientsMu.RLock()
	client, clientExists := h.clients[userID]
	h.clientsMu.RUnlock()

	if !clientExists {
		return errors.New("client not connected")
	}

	// Add client to room
	room.mu.Lock()
	room.Clients[userID] = client
	room.mu.Unlock()

	// Add room to client's room list
	client.Rooms[roomID] = true

	log.Printf("User %s joined room %s", userID, roomID)

	// Notify other room members
	h.BroadcastToRoom(roomID, Message{
		Type: "user_joined",
		Data: map[string]interface{}{
			"user_id": userID,
			"room_id": roomID,
		},
	})

	return nil
}

// LeaveRoom removes a client from a room
func (h *Hub) LeaveRoom(userID, roomID string) error {
	h.roomsMu.RLock()
	room, exists := h.rooms[roomID]
	h.roomsMu.RUnlock()

	if !exists {
		return errors.New("room not found")
	}

	// Remove client from room
	room.mu.Lock()
	delete(room.Clients, userID)
	clientCount := len(room.Clients)
	room.mu.Unlock()

	// Remove room from client's list
	h.clientsMu.RLock()
	if client, ok := h.clients[userID]; ok {
		delete(client.Rooms, roomID)
	}
	h.clientsMu.RUnlock()

	log.Printf("User %s left room %s", userID, roomID)

	// If room is empty, close it
	if clientCount == 0 {
		h.CloseRoom(roomID)
		return nil
	}

	// Notify remaining members
	h.BroadcastToRoom(roomID, Message{
		Type: "user_left",
		Data: map[string]interface{}{
			"user_id": userID,
			"room_id": roomID,
		},
	})

	return nil
}

// LeaveAllRooms removes a client from all rooms
func (h *Hub) LeaveAllRooms(userID string) {
	h.clientsMu.RLock()
	client, exists := h.clients[userID]
	h.clientsMu.RUnlock()

	if !exists {
		return
	}

	// Get list of rooms to leave
	roomsToLeave := make([]string, 0, len(client.Rooms))
	for roomID := range client.Rooms {
		roomsToLeave = append(roomsToLeave, roomID)
	}

	// Leave each room
	for _, roomID := range roomsToLeave {
		h.LeaveRoom(userID, roomID)
	}
}

// CloseRoom closes a room and removes all clients
func (h *Hub) CloseRoom(roomID string) {
	h.roomsMu.Lock()
	room, exists := h.rooms[roomID]
	if !exists {
		h.roomsMu.Unlock()
		return
	}

	// Notify all clients in the room
	room.mu.RLock()
	for userID := range room.Clients {
		if client := h.GetClient(userID); client != nil {
			client.SendMessage(Message{
				Type: "room_closed",
				Data: map[string]interface{}{
					"room_id": roomID,
				},
			})
			delete(client.Rooms, roomID)
		}
	}
	room.mu.RUnlock()

	// Delete room
	delete(h.rooms, roomID)
	h.roomsMu.Unlock()

	log.Printf("Room closed: %s", roomID)

	// Remove from cache if available
	if h.cache != nil {
		// h.cache.Delete("ws:room:" + roomID)
		// h.cache.Increment("ws:stats:total_rooms", -1)
	}
}

// BroadcastToRoom sends a message to all clients in a room
func (h *Hub) BroadcastToRoom(roomID string, msg Message) {
	h.roomsMu.RLock()
	room, exists := h.rooms[roomID]
	h.roomsMu.RUnlock()

	if !exists {
		return
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	for _, client := range room.Clients {
		client.SendMessage(msg)
	}

	// If cache available, publish to other servers (distributed mode)
	if h.cache != nil {
		// h.cache.Publish("ws:room:"+roomID, msg)
	}
}

// GetRoom returns a room by ID
func (h *Hub) GetRoom(roomID string) *Room {
	h.roomsMu.RLock()
	defer h.roomsMu.RUnlock()
	return h.rooms[roomID]
}

// RoomExists checks if a room exists
func (h *Hub) RoomExists(roomID string) bool {
	h.roomsMu.RLock()
	defer h.roomsMu.RUnlock()
	_, exists := h.rooms[roomID]
	return exists
}

// GetRoomClientCount returns the number of clients in a room
func (h *Hub) GetRoomClientCount(roomID string) int {
	h.roomsMu.RLock()
	room, exists := h.rooms[roomID]
	h.roomsMu.RUnlock()

	if !exists {
		return 0
	}

	return room.GetClientCount()
}

// GetRoomClients returns the list of user IDs in a room
func (h *Hub) GetRoomClients(roomID string) []string {
	h.roomsMu.RLock()
	room, exists := h.rooms[roomID]
	h.roomsMu.RUnlock()

	if !exists {
		return []string{}
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	users := make([]string, 0, len(room.Clients))
	for userID := range room.Clients {
		users = append(users, userID)
	}
	return users
}

// ListRooms returns all public rooms
func (h *Hub) ListRooms() []*RoomInfo {
	h.roomsMu.RLock()
	defer h.roomsMu.RUnlock()

	rooms := make([]*RoomInfo, 0)
	for _, room := range h.rooms {
		if !room.IsPrivate {
			rooms = append(rooms, room.ToInfo())
		}
	}
	return rooms
}

// GetUserRooms returns all rooms a user is in
func (h *Hub) GetUserRooms(userID string) []string {
	h.clientsMu.RLock()
	client, exists := h.clients[userID]
	h.clientsMu.RUnlock()

	if !exists {
		return []string{}
	}

	rooms := make([]string, 0, len(client.Rooms))
	for roomID := range client.Rooms {
		rooms = append(rooms, roomID)
	}
	return rooms
}

// KickFromRoom forcefully removes a user from a room
func (h *Hub) KickFromRoom(userID, roomID, reason string) error {
	// Send kick notification
	if err := h.SendToUser(userID, Message{
		Type: "kicked",
		Data: map[string]interface{}{
			"room_id": roomID,
			"reason":  reason,
		},
	}); err == nil {
		// Give client time to receive the message
		time.Sleep(100 * time.Millisecond)
	}

	// Remove from room
	return h.LeaveRoom(userID, roomID)
}

// IsRoomFull checks if a room has reached max capacity
func (h *Hub) IsRoomFull(roomID string) bool {
	h.roomsMu.RLock()
	room, exists := h.rooms[roomID]
	h.roomsMu.RUnlock()

	if !exists {
		return false
	}

	return room.IsFull()
}
