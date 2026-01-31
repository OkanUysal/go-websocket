package websocket

import (
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewHub(t *testing.T) {
	t.Run("with nil config", func(t *testing.T) {
		hub := NewHub(nil)
		if hub == nil {
			t.Fatal("Expected hub to be created")
		}
		if hub.config == nil {
			t.Fatal("Expected default config to be used")
		}
		if hub.config.PingInterval != 30*time.Second {
			t.Errorf("Expected default ping interval 30s, got %v", hub.config.PingInterval)
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &Config{
			PingInterval:    15 * time.Second,
			ReadBufferSize:  2048,
			WriteBufferSize: 2048,
		}
		hub := NewHub(config)
		if hub.config.PingInterval != 15*time.Second {
			t.Errorf("Expected ping interval 15s, got %v", hub.config.PingInterval)
		}
		if hub.config.ReadBufferSize != 2048 {
			t.Errorf("Expected read buffer 2048, got %d", hub.config.ReadBufferSize)
		}
	})
}

func TestRoomManagement(t *testing.T) {
	hub := NewHub(nil)

	t.Run("create room", func(t *testing.T) {
		roomID := hub.CreateRoom(&RoomConfig{
			Name:       "Test Room",
			MaxClients: 10,
			IsPrivate:  false,
		})

		if roomID == "" {
			t.Fatal("Expected room ID to be generated")
		}

		if !hub.RoomExists(roomID) {
			t.Error("Expected room to exist")
		}

		room := hub.GetRoom(roomID)
		if room == nil {
			t.Fatal("Expected room to be retrievable")
		}
		if room.Name != "Test Room" {
			t.Errorf("Expected room name 'Test Room', got '%s'", room.Name)
		}
		if room.MaxClients != 10 {
			t.Errorf("Expected max clients 10, got %d", room.MaxClients)
		}
	})

	t.Run("room client count", func(t *testing.T) {
		roomID := hub.CreateRoom(&RoomConfig{
			Name:       "Test Room 2",
			MaxClients: 5,
		})

		count := hub.GetRoomClientCount(roomID)
		if count != 0 {
			t.Errorf("Expected 0 clients, got %d", count)
		}
	})

	t.Run("list rooms", func(t *testing.T) {
		// Create public room
		hub.CreateRoom(&RoomConfig{
			Name:      "Public Room",
			IsPrivate: false,
		})

		// Create private room
		hub.CreateRoom(&RoomConfig{
			Name:      "Private Room",
			IsPrivate: true,
		})

		rooms := hub.ListRooms()
		// Should only list public rooms
		for _, room := range rooms {
			if room.IsPrivate {
				t.Error("ListRooms should not return private rooms")
			}
		}
	})

	t.Run("room is full check", func(t *testing.T) {
		roomID := hub.CreateRoom(&RoomConfig{
			Name:       "Limited Room",
			MaxClients: 2,
		})

		if hub.IsRoomFull(roomID) {
			t.Error("Expected room to not be full initially")
		}

		room := hub.GetRoom(roomID)
		if room.IsFull() {
			t.Error("Expected room.IsFull() to return false")
		}

		// Room with MaxClients = 0 should never be full
		unlimitedRoomID := hub.CreateRoom(&RoomConfig{
			Name:       "Unlimited Room",
			MaxClients: 0,
		})
		if hub.IsRoomFull(unlimitedRoomID) {
			t.Error("Expected unlimited room to never be full")
		}
	})

	t.Run("close room", func(t *testing.T) {
		roomID := hub.CreateRoom(&RoomConfig{
			Name: "Temporary Room",
		})

		if !hub.RoomExists(roomID) {
			t.Fatal("Expected room to exist before closing")
		}

		hub.CloseRoom(roomID)

		if hub.RoomExists(roomID) {
			t.Error("Expected room to not exist after closing")
		}
	})
}

func TestRoomInfo(t *testing.T) {
	hub := NewHub(nil)

	roomID := hub.CreateRoom(&RoomConfig{
		Name:       "Info Test Room",
		MaxClients: 5,
		IsPrivate:  false,
		Metadata: map[string]interface{}{
			"game_type": "ranked",
			"map":       "desert",
		},
	})

	room := hub.GetRoom(roomID)
	info := room.ToInfo()

	if info.Name != "Info Test Room" {
		t.Errorf("Expected name 'Info Test Room', got '%s'", info.Name)
	}
	if info.MaxClients != 5 {
		t.Errorf("Expected max clients 5, got %d", info.MaxClients)
	}
	if info.IsPrivate {
		t.Error("Expected IsPrivate to be false")
	}
	if info.Metadata["game_type"] != "ranked" {
		t.Error("Expected metadata to be preserved")
	}
}

func TestMessageType(t *testing.T) {
	msg := Message{
		Type: "test",
		Data: map[string]interface{}{
			"key": "value",
		},
	}

	if msg.Type != "test" {
		t.Errorf("Expected type 'test', got '%s'", msg.Type)
	}
	if msg.Data["key"] != "value" {
		t.Error("Expected data to be preserved")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.ReadBufferSize != 1024 {
		t.Errorf("Expected read buffer 1024, got %d", config.ReadBufferSize)
	}
	if config.WriteBufferSize != 1024 {
		t.Errorf("Expected write buffer 1024, got %d", config.WriteBufferSize)
	}
	if config.PingInterval != 30*time.Second {
		t.Errorf("Expected ping interval 30s, got %v", config.PingInterval)
	}
	if config.PongWait != 60*time.Second {
		t.Errorf("Expected pong wait 60s, got %v", config.PongWait)
	}
	if config.MaxMessageSize != 512*1024 {
		t.Errorf("Expected max message size 512KB, got %d", config.MaxMessageSize)
	}
	if config.Cache != nil {
		t.Error("Expected cache to be nil by default")
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Error("Expected ID to be generated")
	}
	if id1 == id2 {
		t.Error("Expected IDs to be unique")
	}
	if len(id1) != 32 {
		t.Errorf("Expected ID length 32, got %d", len(id1))
	}
}

func TestGenerateRoomID(t *testing.T) {
	roomID := generateRoomID()

	if roomID == "" {
		t.Error("Expected room ID to be generated")
	}
	if len(roomID) < 5 {
		t.Error("Expected room ID to have 'room_' prefix")
	}
	if roomID[:5] != "room_" {
		t.Errorf("Expected room ID to start with 'room_', got '%s'", roomID)
	}
}

// MockConn for testing (basic implementation)
type MockConn struct {
	*websocket.Conn
}

func TestOnlineCount(t *testing.T) {
	hub := NewHub(nil)

	count := hub.GetOnlineCount()
	if count != 0 {
		t.Errorf("Expected 0 online users, got %d", count)
	}
}

func TestGetOnlineUsers(t *testing.T) {
	hub := NewHub(nil)

	users := hub.GetOnlineUsers()
	if len(users) != 0 {
		t.Errorf("Expected 0 users, got %d", len(users))
	}
}
