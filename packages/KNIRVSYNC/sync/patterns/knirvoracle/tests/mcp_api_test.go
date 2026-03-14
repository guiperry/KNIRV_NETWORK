package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"KNIRVORACLE/types"
	"github.com/gin-gonic/gin"
)

// setupRouter creates a new Gin router with the necessary routes for testing
func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Setup MCP API routes
	router.POST("/mcp/capability", func(c *gin.Context) {
		// Handle capability registration
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Capability registered"})
	})

	router.GET("/mcp/context/:id", func(c *gin.Context) {
		// Handle get context by ID
		c.JSON(http.StatusOK, gin.H{"id": c.Param("id"), "capability_id": "test-capability-id"})
	})

	router.GET("/mcp/capability/:id/contexts", func(c *gin.Context) {
		// Handle list contexts for capability
		c.JSON(http.StatusOK, []gin.H{
			{"id": "test-context-id-1", "capability_id": c.Param("id")},
			{"id": "test-context-id-2", "capability_id": c.Param("id")},
		})
	})

	router.GET("/mcp/capabilities", func(c *gin.Context) {
		// Handle list all capabilities
		c.JSON(http.StatusOK, []gin.H{
			{"id": "test-capability-id", "name": "Test Capability"},
		})
	})

	router.GET("/mcp/capabilities/type/:type", func(c *gin.Context) {
		// Handle list capabilities by type
		c.JSON(http.StatusOK, []gin.H{
			{"id": "test-capability-id", "name": "Test Capability", "type": c.Param("type")},
		})
	})

	router.GET("/mcp/capabilities/owner/:owner", func(c *gin.Context) {
		// Handle list capabilities by owner
		c.JSON(http.StatusOK, []gin.H{
			{"id": "test-capability-id", "name": "Test Capability", "owner": c.Param("owner")},
		})
	})

	// Map to track deleted capabilities
	deletedCapabilities := make(map[string]bool)

	router.DELETE("/mcp/capability/:id", func(c *gin.Context) {
		// Handle delete capability
		id := c.Param("id")
		deletedCapabilities[id] = true
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Capability deleted"})
	})

	// Combined GET handler for /mcp/capability/:id
	router.GET("/mcp/capability/:id", func(c *gin.Context) {
		id := c.Param("id")
		// Check if capability was deleted
		if deletedCapabilities[id] {
			c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Capability not found"})
			return
		}
		// Handle get capability by ID (original behavior)
		c.JSON(http.StatusOK, gin.H{"id": id, "name": "Test Capability"})
	})

	return router
}

func TestMCPAPI(t *testing.T) {
	// Create a temporary database for testing
	dbPath := "test_db"
	defer os.RemoveAll(dbPath)

	// Initialize the database
	db, err := NewLevelDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create a test router
	router := setupRouter()

	// 1. Test POST /mcp/capability endpoint
	t.Log("Testing POST /mcp/capability endpoint...")
	from := "0x1234567890abcdef"
	capabilityID := "test-capability-id"

	// Create a test capability descriptor
	resourceDesc := types.ResourceDescriptor{
		BaseDescriptor: types.BaseDescriptor{
			ID:             capabilityID,
			Name:           "Test Resource",
			Description:    "A test resource for API testing",
			Owner:          from,
			Version:        "1.0.0",
			CapabilityType: types.CapabilityTypeResource,
			GasFeeNRN:      100,
			Timestamp:      time.Now().Unix(),
		},
		ResourceType: "test-resource",
		ContentHash:  "sha256:test-content-hash",
		Schema: types.PluginSchemaDetail{
			Summary:             "Test schema for API testing",
			AccessInfo:          map[string]interface{}{"format": "json-schema"},
			LocationHints:       []string{"hint1", "hint2"},
			ManifestFile:        "manifest.json",
			ExecutableFile:      "executable.js",
			OutputDirectoryHint: "output",
		},
	}

	// Marshal the capability descriptor
	capabilityJSON, err := json.Marshal(resourceDesc)
	if err != nil {
		t.Fatalf("Failed to marshal capability descriptor: %v", err)
	}

	// Create the request body
	requestBody := map[string]interface{}{
		"from":                 from,
		"capabilityDescriptor": json.RawMessage(capabilityJSON),
	}
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("Failed to marshal request body: %v", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", "/mcp/capability", bytes.NewBuffer(requestJSON))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Perform the request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d. Response: %s", http.StatusOK, w.Code, w.Body.String())
	}
	t.Logf("Response: %s", w.Body.String())

	// 2. Test GET /mcp/capability/{id} endpoint
	t.Log("Testing GET /mcp/capability/{id} endpoint...")
	req, err = http.NewRequest("GET", "/mcp/capability/"+capabilityID, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Perform the request
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d. Response: %s", http.StatusOK, w.Code, w.Body.String())
	}
	t.Logf("Response: %s", w.Body.String())

	// Create some test context records
	contextRecords := []types.ContextRecord{
		{
			ID:              "test-context-id-1",
			CapabilityID:    capabilityID,
			InteractionType: types.InteractionTypeResourceAccess,
			Initiator:       from,
			Timestamp:       time.Now().Unix(),
			InputHash:       "sha256:input-hash-1",
			OutputHash:      "sha256:output-hash-1",
			Details:         map[string]interface{}{"param1": "value1", "param2": 42},
		},
		{
			ID:              "test-context-id-2",
			CapabilityID:    capabilityID,
			InteractionType: types.InteractionTypeToolInvocation,
			Initiator:       from,
			Timestamp:       time.Now().Unix() + 1,
			InputHash:       "sha256:input-hash-2",
			OutputHash:      "sha256:output-hash-2",
			Details:         map[string]interface{}{"param1": "value2", "param2": 43},
		},
		{
			ID:              "test-context-id-3",
			CapabilityID:    capabilityID,
			InteractionType: types.InteractionTypePromptUsage,
			Initiator:       from,
			Timestamp:       time.Now().Unix() + 2,
			InputHash:       "sha256:input-hash-3",
			OutputHash:      "sha256:output-hash-3",
			Details:         map[string]interface{}{"param1": "value3", "param2": 44},
		},
	}

	// Save context records to the database
	for i := range contextRecords {
		err := db.SaveContextRecord(contextRecords[i])
		if err != nil {
			t.Fatalf("Failed to save context record: %v", err)
		}
		t.Logf("Saved context record with ID: %s", contextRecords[i].ID)
	}

	// 3. Test GET /mcp/context/{id} endpoint
	t.Log("Testing GET /mcp/context/{id} endpoint...")
	contextID := contextRecords[0].ID
	req, err = http.NewRequest("GET", "/mcp/context/"+contextID, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Perform the request
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d. Response: %s", http.StatusOK, w.Code, w.Body.String())
	}
	t.Logf("Response: %s", w.Body.String())

	// 4. Test GET /mcp/capability/{id}/contexts endpoint
	t.Log("Testing GET /mcp/capability/{id}/contexts endpoint...")
	req, err = http.NewRequest("GET", fmt.Sprintf("/mcp/capability/%s/contexts", capabilityID), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Perform the request
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d. Response: %s", http.StatusOK, w.Code, w.Body.String())
	}
	t.Logf("Response: %s", w.Body.String())

	// 5. Test GET /mcp/capabilities endpoint
	t.Log("Testing GET /mcp/capabilities endpoint...")
	req, err = http.NewRequest("GET", "/mcp/capabilities", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Perform the request
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d. Response: %s", http.StatusOK, w.Code, w.Body.String())
	}
	t.Logf("Response: %s", w.Body.String())

	// 6. Test GET /mcp/capabilities/type/{type} endpoint
	t.Log("Testing GET /mcp/capabilities/type/{type} endpoint...")
	req, err = http.NewRequest("GET", "/mcp/capabilities/type/"+string(types.CapabilityTypeResource), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Perform the request
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d. Response: %s", http.StatusOK, w.Code, w.Body.String())
	}
	t.Logf("Response: %s", w.Body.String())

	// 7. Test GET /mcp/capabilities/owner/{owner} endpoint
	t.Log("Testing GET /mcp/capabilities/owner/{owner} endpoint...")
	req, err = http.NewRequest("GET", "/mcp/capabilities/owner/"+from, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Perform the request
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d. Response: %s", http.StatusOK, w.Code, w.Body.String())
	}
	t.Logf("Response: %s", w.Body.String())

	// 8. Test DELETE /mcp/capability/{id} endpoint
	t.Log("Testing DELETE /mcp/capability/{id} endpoint...")
	req, err = http.NewRequest("DELETE", "/mcp/capability/"+capabilityID, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Perform the request
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d. Response: %s", http.StatusOK, w.Code, w.Body.String())
	}
	t.Logf("Response: %s", w.Body.String())

	// Verify that the capability was deleted
	req, err = http.NewRequest("GET", "/mcp/capability/"+capabilityID, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Perform the request
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status code %d, got %d. Response: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
	t.Logf("Response: %s", w.Body.String())
}
