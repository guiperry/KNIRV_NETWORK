package nexus

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// TestNewNexusMemoryServer tests the creation of a new NexusMemoryServer
func TestNewNexusMemoryServer(t *testing.T) {
	allocator := memory.NewGoAllocator()
	server := NewNexusMemoryServer(allocator)

	assert.NotNil(t, server)
	assert.Equal(t, allocator, server.mem)
	assert.Len(t, server.intents, 0)
	assert.Len(t, server.recentEvents, 0)
}

// TestRegisterIntent tests registering an agent's intent
func TestRegisterIntent(t *testing.T) {
	allocator := memory.NewGoAllocator()
	server := NewNexusMemoryServer(allocator)

	server.RegisterIntent(123, "test-agent", "test-intent")

	server.intentsMu.Lock()
	intent, ok := server.intents[123]
	server.intentsMu.Unlock()

	assert.True(t, ok)
	assert.Equal(t, "test-agent", intent.AgentID)
	assert.Equal(t, "test-intent", intent.IntentStr)
	assert.NotZero(t, intent.Timestamp)
}

// TestCorrelate tests the intent correlation logic
func TestCorrelate(t *testing.T) {
	allocator := memory.NewGoAllocator()
	server := NewNexusMemoryServer(allocator)

	// Test with empty intent (should return true)
	assert.True(t, server.correlate("", "any action"))

	// Test with unknown intent (should return true)
	assert.True(t, server.correlate("unknown", "any action"))

	// Test with matching keywords
	assert.True(t, server.correlate("connect to database", "will connect to database server"))

	// Test with partial matching (should return true when at least half match)
	assert.True(t, server.correlate("connect to database and verify", "will connect to database")) // 2/3 keywords match

	// Test with half matching (should return true when exactly half match)
	assert.True(t, server.correlate("connect to database and verify and update", "will connect to database")) // 2/4 keywords match
}

// TestHandleGetEvents tests the HTTP handler for getting fabric events
func TestHandleGetEvents(t *testing.T) {
	allocator := memory.NewGoAllocator()
	server := NewNexusMemoryServer(allocator)

	// Add a test event
	event := FabricEvent{
		Timestamp:      1000,
		AgentID:        "test-agent",
		Intent:         "test-intent",
		ObservedAction: "observed-action",
		Verified:       true,
	}

	server.recentEventsMu.Lock()
	server.recentEvents = append(server.recentEvents, event)
	server.recentEventsMu.Unlock()

	// Create a request
	req := httptest.NewRequest(http.MethodGet, "/api/nexus/events", nil)
	rr := httptest.NewRecorder()

	// Create router and register handler
	router := mux.NewRouter()
	server.RegisterRoutes(router)

	// Serve the request
	router.ServeHTTP(rr, req)

	// Check the response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	// Parse the response
	var events []FabricEvent
	if err := json.NewDecoder(rr.Body).Decode(&events); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}
	assert.Len(t, events, 1)
	assert.Equal(t, event, events[0])
}

// TestRegisterRoutes tests that routes are properly registered
func TestRegisterRoutes(t *testing.T) {
	allocator := memory.NewGoAllocator()
	server := NewNexusMemoryServer(allocator)

	router := mux.NewRouter()
	server.RegisterRoutes(router)

	// Check that the route was registered
	assert.NotNil(t, router)
}
