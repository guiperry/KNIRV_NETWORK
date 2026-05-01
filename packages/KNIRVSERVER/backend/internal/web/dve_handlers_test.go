package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend_server/internal/objects"
	"backend_server/internal/services/dvemanager"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDVEHandlers(t *testing.T) {
	dveManager := &dvemanager.DVEManager{}
	handlers := NewDVEHandlers(dveManager, nil)

	assert.NotNil(t, handlers)
	assert.Equal(t, dveManager, handlers.dveManager)
}

func TestGetCurrentTimestamp(t *testing.T) {
	timestamp := getCurrentTimestamp()
	assert.NotEmpty(t, timestamp)
	// Should be RFC3339 format
	assert.Contains(t, timestamp, "T")
	// Can end with Z or timezone offset
	assert.True(t, strings.HasSuffix(timestamp, "Z") || strings.Contains(timestamp, "-") || strings.Contains(timestamp, "+"))
}

func TestDVEHandlers_PostDVENodes_InvalidJSON(t *testing.T) {
	dveManager := &dvemanager.DVEManager{}
	handlers := NewDVEHandlers(dveManager, nil)

	req := httptest.NewRequest("POST", "/api/dve-nodes", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handlers.PostDVENodes(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response DVENodeResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Invalid request body")
}

func TestDVEHandlers_PostDVENodes_MissingName(t *testing.T) {
	dveManager := &dvemanager.DVEManager{}
	handlers := NewDVEHandlers(dveManager, nil)

	reqBody := objects.RegisterNodeRequest{
		TEEType: "SGX",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/dve-nodes", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.PostDVENodes(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response DVENodeResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Node name is required")
}

func TestDVEHandlers_PostDVENodes_MissingTEEType(t *testing.T) {
	dveManager := &dvemanager.DVEManager{}
	handlers := NewDVEHandlers(dveManager, nil)

	reqBody := objects.RegisterNodeRequest{
		Name: "test-node",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/dve-nodes", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.PostDVENodes(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response DVENodeResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "TEE type is required")
}

func TestDVEHandlers_GetDVENode_MissingID(t *testing.T) {
	dveManager := &dvemanager.DVEManager{}
	handlers := NewDVEHandlers(dveManager, nil)

	req := httptest.NewRequest("GET", "/api/dve-nodes/", nil)
	w := httptest.NewRecorder()

	handlers.GetDVENode(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response DVENodeResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Node ID is required")
}

func TestDVEHandlers_UpdateDVENode_MissingID(t *testing.T) {
	dveManager := &dvemanager.DVEManager{}
	handlers := NewDVEHandlers(dveManager, nil)

	req := httptest.NewRequest("PUT", "/api/dve-nodes/", bytes.NewReader([]byte("{}")))
	w := httptest.NewRecorder()

	handlers.UpdateDVENode(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response DVENodeResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Node ID is required")
}

func TestDVEHandlers_UpdateDVENode_InvalidJSON(t *testing.T) {
	dveManager := &dvemanager.DVEManager{}
	handlers := NewDVEHandlers(dveManager, nil)

	req := httptest.NewRequest("PUT", "/api/dve-nodes/test-id", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	// Set up mux vars
	req = mux.SetURLVars(req, map[string]string{"id": "test-id"})

	handlers.UpdateDVENode(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response DVENodeResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Invalid request body")
}

func TestDVEHandlers_DeleteDVENode_MissingID(t *testing.T) {
	dveManager := &dvemanager.DVEManager{}
	handlers := NewDVEHandlers(dveManager, nil)

	req := httptest.NewRequest("DELETE", "/api/dve-nodes/", nil)
	w := httptest.NewRecorder()

	handlers.DeleteDVENode(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response DVENodeResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Node ID is required")
}

func TestDVEHandlers_GetDVENodeEndpoints_MissingID(t *testing.T) {
	dveManager := &dvemanager.DVEManager{}
	handlers := NewDVEHandlers(dveManager, nil)

	req := httptest.NewRequest("GET", "/api/dve-nodes//endpoints", nil)
	w := httptest.NewRecorder()

	handlers.GetDVENodeEndpoints(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response DVENodeResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Node ID is required")
}

func TestDVEHandlers_GetDVENodeSSHEndpoint_MissingID(t *testing.T) {
	dveManager := &dvemanager.DVEManager{}
	handlers := NewDVEHandlers(dveManager, nil)

	req := httptest.NewRequest("GET", "/api/dve-nodes//ssh-endpoint", nil)
	w := httptest.NewRecorder()

	handlers.GetDVENodeSSHEndpoint(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response DVENodeResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Node ID is required")
}

func TestDVEHandlers_GetDVENodeValidationEndpoint_MissingID(t *testing.T) {
	dveManager := &dvemanager.DVEManager{}
	handlers := NewDVEHandlers(dveManager, nil)

	req := httptest.NewRequest("GET", "/api/dve-nodes//validation-endpoint", nil)
	w := httptest.NewRecorder()

	handlers.GetDVENodeValidationEndpoint(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response DVENodeResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Node ID is required")
}

func TestDVEHandlers_GetDVENodeErrorResolutionEndpoint_MissingID(t *testing.T) {
	dveManager := &dvemanager.DVEManager{}
	handlers := NewDVEHandlers(dveManager, nil)

	req := httptest.NewRequest("GET", "/api/dve-nodes//error-resolution-endpoint", nil)
	w := httptest.NewRecorder()

	handlers.GetDVENodeErrorResolutionEndpoint(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response DVENodeResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Node ID is required")
}

func TestDVEHandlers_RegisterRoutes(t *testing.T) {
	dveManager := &dvemanager.DVEManager{}
	handlers := NewDVEHandlers(dveManager, nil)

	router := mux.NewRouter()
	authMiddleware := &middleware.AuthMiddleware{}

	// Should not panic
	assert.NotPanics(t, func() {
		handlers.RegisterRoutes(router, authMiddleware)
	})

	// Test with nil auth middleware
	assert.NotPanics(t, func() {
		handlers.RegisterRoutes(router, nil)
	})
}

// Test DVENodeResponse JSON marshaling
func TestDVENodeResponse_JSON(t *testing.T) {
	response := DVENodeResponse{
		Success:   true,
		Data:      "test data",
		Total:     10,
		Message:   "Test message",
		Error:     "",
		Timestamp: "2023-01-01T00:00:00Z",
	}

	data, err := json.Marshal(response)
	require.NoError(t, err)

	var decoded DVENodeResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, response.Success, decoded.Success)
	assert.Equal(t, response.Data, decoded.Data)
	assert.Equal(t, response.Total, decoded.Total)
	assert.Equal(t, response.Message, decoded.Message)
	assert.Equal(t, response.Error, decoded.Error)
	assert.Equal(t, response.Timestamp, decoded.Timestamp)
}
