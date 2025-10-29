package sse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSSEManager(t *testing.T) {
	manager := NewSSEManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.clients)
	assert.NotNil(t, manager.rooms)
}

func TestSSEManager_AddClient(t *testing.T) {
	manager := NewSSEManager()

	client := &SSEClient{
		ID:      "test-client",
		UserID:  "user123",
		Channel: make(chan SSEMessage, 10),
	}

	manager.AddClient(client)

	// Verify client was added
	assert.Equal(t, 1, manager.GetClientCount())
}

func TestSSEManager_RemoveClient(t *testing.T) {
	manager := NewSSEManager()

	client := &SSEClient{
		ID:      "test-client",
		UserID:  "user123",
		Channel: make(chan SSEMessage, 10),
		Rooms:   []string{"room1", "room2"},
	}

	manager.AddClient(client)
	assert.Equal(t, 1, manager.GetClientCount())

	// Add client to rooms
	manager.JoinRoom("test-client", "room1")
	manager.JoinRoom("test-client", "room2")

	manager.RemoveClient("test-client")
	assert.Equal(t, 0, manager.GetClientCount())
	assert.Equal(t, 0, manager.GetRoomCount()) // Rooms should be empty
}

func TestSSEManager_SendToClient(t *testing.T) {
	manager := NewSSEManager()

	client := &SSEClient{
		ID:      "test-client",
		UserID:  "user123",
		Channel: make(chan SSEMessage, 10),
	}

	manager.AddClient(client)

	message := SSEMessage{
		Event: "test",
		Data:  "test data",
		ID:    "msg-123",
	}

	err := manager.SendToClient("test-client", message)
	assert.NoError(t, err)

	// Verify message was received
	select {
	case received := <-client.Channel:
		assert.Equal(t, message.Event, received.Event)
		assert.Equal(t, message.Data, received.Data)
		assert.Equal(t, message.ID, received.ID)
	case <-time.After(time.Second):
		t.Fatal("Message not received")
	}
}

func TestSSEManager_SendToClient_NotFound(t *testing.T) {
	manager := NewSSEManager()

	message := SSEMessage{
		Event: "test",
		Data:  "test data",
	}

	err := manager.SendToClient("nonexistent", message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client nonexistent not found")
}

func TestSSEManager_BroadcastToRoom(t *testing.T) {
	manager := NewSSEManager()

	// Create two clients
	client1 := &SSEClient{
		ID:      "client1",
		UserID:  "user1",
		Channel: make(chan SSEMessage, 10),
	}
	client2 := &SSEClient{
		ID:      "client2",
		UserID:  "user2",
		Channel: make(chan SSEMessage, 10),
	}

	manager.AddClient(client1)
	manager.AddClient(client2)

	// Join room
	manager.JoinRoom("client1", "test-room")
	manager.JoinRoom("client2", "test-room")

	message := SSEMessage{
		Event: "broadcast",
		Data:  "room message",
	}

	manager.BroadcastToRoom("test-room", message)

	// Verify both clients received the message
	for _, client := range []*SSEClient{client1, client2} {
		select {
		case received := <-client.Channel:
			assert.Equal(t, message.Event, received.Event)
			assert.Equal(t, message.Data, received.Data)
		case <-time.After(time.Second):
			t.Fatalf("Client %s did not receive message", client.ID)
		}
	}
}

func TestSSEManager_BroadcastToAll(t *testing.T) {
	manager := NewSSEManager()

	// Create two clients
	client1 := &SSEClient{
		ID:      "client1",
		UserID:  "user1",
		Channel: make(chan SSEMessage, 10),
	}
	client2 := &SSEClient{
		ID:      "client2",
		UserID:  "user2",
		Channel: make(chan SSEMessage, 10),
	}

	manager.AddClient(client1)
	manager.AddClient(client2)

	message := SSEMessage{
		Event: "broadcast",
		Data:  "all message",
	}

	manager.BroadcastToAll(message)

	// Verify both clients received the message
	for _, client := range []*SSEClient{client1, client2} {
		select {
		case received := <-client.Channel:
			assert.Equal(t, message.Event, received.Event)
			assert.Equal(t, message.Data, received.Data)
		case <-time.After(time.Second):
			t.Fatalf("Client %s did not receive message", client.ID)
		}
	}
}

func TestSSEManager_JoinRoom(t *testing.T) {
	manager := NewSSEManager()

	client := &SSEClient{
		ID:      "test-client",
		UserID:  "user123",
		Channel: make(chan SSEMessage, 10),
		Rooms:   make([]string, 0),
	}

	manager.AddClient(client)

	// Join room
	err := manager.JoinRoom("test-client", "test-room")
	assert.NoError(t, err)

	// Verify client is in room
	clientsInRoom := manager.GetClientsInRoom("test-room")
	assert.Contains(t, clientsInRoom, "test-client")
	assert.Equal(t, 1, manager.GetRoomCount())
}

func TestSSEManager_JoinRoom_ClientNotFound(t *testing.T) {
	manager := NewSSEManager()

	err := manager.JoinRoom("nonexistent", "test-room")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client nonexistent not found")
}

func TestSSEManager_LeaveRoom(t *testing.T) {
	manager := NewSSEManager()

	client := &SSEClient{
		ID:      "test-client",
		UserID:  "user123",
		Channel: make(chan SSEMessage, 10),
		Rooms:   make([]string, 0),
	}

	manager.AddClient(client)

	// Join and then leave room
	manager.JoinRoom("test-client", "test-room")
	assert.Equal(t, 1, manager.GetRoomCount())

	err := manager.LeaveRoom("test-client", "test-room")
	assert.NoError(t, err)
	assert.Equal(t, 0, manager.GetRoomCount())
}

func TestSSEManager_LeaveRoom_ClientNotFound(t *testing.T) {
	manager := NewSSEManager()

	err := manager.LeaveRoom("nonexistent", "test-room")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client nonexistent not found")
}

func TestSSEManager_GetClientCount(t *testing.T) {
	manager := NewSSEManager()

	assert.Equal(t, 0, manager.GetClientCount())

	manager.AddClient(&SSEClient{ID: "client1"})
	assert.Equal(t, 1, manager.GetClientCount())

	manager.AddClient(&SSEClient{ID: "client2"})
	assert.Equal(t, 2, manager.GetClientCount())
}

func TestSSEManager_GetRoomCount(t *testing.T) {
	manager := NewSSEManager()

	assert.Equal(t, 0, manager.GetRoomCount())

	client := &SSEClient{ID: "client1"}
	manager.AddClient(client)
	manager.JoinRoom("client1", "room1")

	assert.Equal(t, 1, manager.GetRoomCount())
}

func TestSSEManager_GetClientsInRoom(t *testing.T) {
	manager := NewSSEManager()

	client1 := &SSEClient{ID: "client1"}
	client2 := &SSEClient{ID: "client2"}
	manager.AddClient(client1)
	manager.AddClient(client2)

	manager.JoinRoom("client1", "test-room")
	manager.JoinRoom("client2", "test-room")

	clients := manager.GetClientsInRoom("test-room")
	assert.Len(t, clients, 2)
	assert.Contains(t, clients, "client1")
	assert.Contains(t, clients, "client2")

	// Test non-existent room
	clients = manager.GetClientsInRoom("nonexistent")
	assert.Len(t, clients, 0)
}

func TestSSEManager_CleanupStaleClients(t *testing.T) {
	manager := NewSSEManager()

	// Create a stale client (last ping 10 minutes ago)
	staleClient := &SSEClient{
		ID:       "stale-client",
		UserID:   "user123",
		Channel:  make(chan SSEMessage, 10),
		LastPing: time.Now().Add(-10 * time.Minute),
		Rooms:    []string{"room1"},
	}

	// Create a fresh client
	freshClient := &SSEClient{
		ID:       "fresh-client",
		UserID:   "user456",
		Channel:  make(chan SSEMessage, 10),
		LastPing: time.Now(),
	}

	manager.AddClient(staleClient)
	manager.AddClient(freshClient)
	manager.JoinRoom("stale-client", "room1")

	assert.Equal(t, 2, manager.GetClientCount())
	assert.Equal(t, 1, manager.GetRoomCount())

	// Cleanup stale clients
	manager.CleanupStaleClients()

	assert.Equal(t, 1, manager.GetClientCount()) // Only fresh client should remain
	assert.Equal(t, 0, manager.GetRoomCount())   // Room should be empty
}

func TestSSEManager_Close(t *testing.T) {
	manager := NewSSEManager()

	client1 := &SSEClient{
		ID:      "client1",
		Channel: make(chan SSEMessage, 10),
	}
	client2 := &SSEClient{
		ID:      "client2",
		Channel: make(chan SSEMessage, 10),
	}

	manager.AddClient(client1)
	manager.AddClient(client2)
	manager.JoinRoom("client1", "room1")

	assert.Equal(t, 2, manager.GetClientCount())
	assert.Equal(t, 1, manager.GetRoomCount())

	manager.Close()

	assert.Equal(t, 0, manager.GetClientCount())
	assert.Equal(t, 0, manager.GetRoomCount())
}

func TestSSEManager_BroadcastUpdate(t *testing.T) {
	manager := NewSSEManager()

	client := &SSEClient{
		ID:      "test-client",
		UserID:  "user123",
		Channel: make(chan SSEMessage, 10),
	}

	manager.AddClient(client)
	manager.JoinRoom("test-client", "test-room")

	update := map[string]string{"status": "updated", "value": "123"}

	manager.BroadcastUpdate("test-room", update)

	// Verify message was received
	select {
	case received := <-client.Channel:
		assert.Equal(t, "update", received.Event)
		assert.Equal(t, update, received.Data)
		assert.NotEmpty(t, received.ID)
	case <-time.After(time.Second):
		t.Fatal("Update message not received")
	}
}

func TestSSEManager_CreateSSEHandler(t *testing.T) {
	manager := NewSSEManager()

	handler := manager.CreateSSEHandler("user123")
	assert.NotNil(t, handler)

	// We can't easily test the full SSE handler without complex HTTP setup
	// But we can verify the handler function is created
	assert.NotNil(t, handler)
}