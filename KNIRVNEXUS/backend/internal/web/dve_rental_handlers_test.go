package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend_server/internal/objects"
	"backend_server/internal/services/container"
	"backend_server/internal/services/dverental"
	"backend_server/internal/services/endpoints"
	"backend_server/internal/services/session"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/buntdb"
)

func TestNewDVERentalHandlers(t *testing.T) {
	dveRentalService := &dverental.DVERentalService{}
	containerOrchestrator := &container.ContainerOrchestrator{}
	db, _ := buntdb.Open(":memory:")
	sessionManager := session.NewSessionManager(db)
	endpointRegistry := &endpoints.EndpointRegistry{}

	handlers := NewDVERentalHandlers(dveRentalService, containerOrchestrator, sessionManager, endpointRegistry, db)

	assert.NotNil(t, handlers)
	assert.Equal(t, dveRentalService, handlers.dveRentalService)
	assert.Equal(t, containerOrchestrator, handlers.containerOrchestrator)
	assert.Equal(t, sessionManager, handlers.sessionManager)
	assert.Equal(t, endpointRegistry, handlers.endpointRegistry)
	assert.Equal(t, db, handlers.db)
}

func TestDownloadSSHPrivateKey_Success(t *testing.T) {
	// Setup mocks
	dveRentalService := &dverental.DVERentalService{}
	containerOrchestrator := &container.ContainerOrchestrator{}
	db, _ := buntdb.Open(":memory:")
	sessionManager := session.NewSessionManager(db)
	endpointRegistry := &endpoints.EndpointRegistry{}

	handlers := NewDVERentalHandlers(dveRentalService, containerOrchestrator, sessionManager, endpointRegistry, db)

	// Create a real SSH session
	privateKey := "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-private-key\n-----END OPENSSH PRIVATE KEY-----"
	sshSession, _ := handlers.sessionManager.CreateSSHSession("rental-123", "container-456", "testuser", privateKey)
	sessionID := sshSession.ID

	req := httptest.NewRequest("GET", "/api/sessions/ssh/"+sessionID+"/private-key", nil)
	req = mux.SetURLVars(req, map[string]string{"sessionId": sessionID})
	w := httptest.NewRecorder()

	handlers.DownloadSSHPrivateKey(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/octet-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "attachment; filename=\"ssh_private_key.pem\"", w.Header().Get("Content-Disposition"))
	assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
	assert.Equal(t, privateKey, w.Body.String())
}

func TestDownloadSSHPrivateKey_SessionNotFound(t *testing.T) {
	dveRentalService := &dverental.DVERentalService{}
	containerOrchestrator := &container.ContainerOrchestrator{}
	db, _ := buntdb.Open(":memory:")
	sessionManager := session.NewSessionManager(db)
	endpointRegistry := &endpoints.EndpointRegistry{}

	handlers := NewDVERentalHandlers(dveRentalService, containerOrchestrator, sessionManager, endpointRegistry, db)

	sessionID := "non-existent-session"

	req := httptest.NewRequest("GET", "/api/sessions/ssh/"+sessionID+"/private-key", nil)
	req = mux.SetURLVars(req, map[string]string{"sessionId": sessionID})
	w := httptest.NewRecorder()

	handlers.DownloadSSHPrivateKey(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response DVERentalResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "SSH session not found")
}

func TestDownloadSSHPrivateKey_SessionExpired(t *testing.T) {
	dveRentalService := &dverental.DVERentalService{}
	containerOrchestrator := &container.ContainerOrchestrator{}
	db, _ := buntdb.Open(":memory:")
	sessionManager := session.NewSessionManager(db)
	endpointRegistry := &endpoints.EndpointRegistry{}

	handlers := NewDVERentalHandlers(dveRentalService, containerOrchestrator, sessionManager, endpointRegistry, db)

	// Create an expired session
	sshSession, _ := handlers.sessionManager.CreateSSHSession("rental-123", "container-456", "testuser", "test-private-key")
	// Manually set expired time
	sshSession.ExpiresAt = time.Now().Add(-1 * time.Hour)
	sshSession.CreatedAt = time.Now().Add(-25 * time.Hour)
	sshSession.LastUsed = time.Now().Add(-2 * time.Hour)

	sessionID := sshSession.ID

	req := httptest.NewRequest("GET", "/api/sessions/ssh/"+sessionID+"/private-key", nil)
	req = mux.SetURLVars(req, map[string]string{"sessionId": sessionID})
	w := httptest.NewRecorder()

	handlers.DownloadSSHPrivateKey(w, req)

	assert.Equal(t, http.StatusGone, w.Code)

	var response DVERentalResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "SSH session has expired")
}

func TestDownloadSSHPrivateKey_MissingSessionID(t *testing.T) {
	dveRentalService := &dverental.DVERentalService{}
	containerOrchestrator := &container.ContainerOrchestrator{}
	db, _ := buntdb.Open(":memory:")
	sessionManager := session.NewSessionManager(db)
	endpointRegistry := &endpoints.EndpointRegistry{}

	handlers := NewDVERentalHandlers(dveRentalService, containerOrchestrator, sessionManager, endpointRegistry, db)

	req := httptest.NewRequest("GET", "/api/sessions/ssh//private-key", nil)
	w := httptest.NewRecorder()

	handlers.DownloadSSHPrivateKey(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response DVERentalResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Session ID is required")
}

func TestTerminateValidationSession_Success(t *testing.T) {
	db, _ := buntdb.Open(":memory:")
	dveRentalService, err := dverental.NewDVERentalService(db)
	require.NoError(t, err)
	containerOrchestrator := &container.ContainerOrchestrator{}
	sessionManager := session.NewSessionManager(db)
	endpointRegistry := &endpoints.EndpointRegistry{}

	handlers := NewDVERentalHandlers(dveRentalService, containerOrchestrator, sessionManager, endpointRegistry, db)

	rentalID := "rental-123"
	userID := "user-456"

	// Create a rental first
	rentalReq := objects.RentalRequest{
		UserID:        userID,
		PlanID:        "basic",
		Duration:      3600, // 1 hour
		PaymentTxHash: "test-tx",
	}
	rentalResp, err := dveRentalService.CreateRental(&rentalReq)
	require.NoError(t, err)
	require.True(t, rentalResp.Success)
	// Get the actual rental ID from the response
	rentalID = rentalResp.RentalID

	// Set container ID for the rental to pass validation
	err = db.Update(func(tx *buntdb.Tx) error {
		val, err := tx.Get("rental:" + rentalID)
		if err != nil {
			return err
		}
		var rental objects.DVERental
		if err := json.Unmarshal([]byte(val), &rental); err != nil {
			return err
		}
		rental.ContainerID = "test-container"
		data, _ := json.Marshal(rental)
		_, _, err = tx.Set("rental:"+rentalID, string(data), nil)
		return err
	})
	require.NoError(t, err)

	// Create a real validation session
	_, _ = handlers.sessionManager.CreateValidationSession(rentalID, "reasoning")

	req := httptest.NewRequest("DELETE", "/api/dve-rental/rentals/"+rentalID+"/validation-session?user_id="+userID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": rentalID})
	w := httptest.NewRecorder()

	handlers.TerminateValidationSession(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response DVERentalResponse
	unmarshalErr := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, unmarshalErr)
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "Validation session terminated successfully")
}

func TestTerminateValidationSession_RentalNotFound(t *testing.T) {
	db, _ := buntdb.Open(":memory:")
	dveRentalService, _ := dverental.NewDVERentalService(db)
	containerOrchestrator := &container.ContainerOrchestrator{}
	sessionManager := session.NewSessionManager(db)
	endpointRegistry := &endpoints.EndpointRegistry{}

	handlers := NewDVERentalHandlers(dveRentalService, containerOrchestrator, sessionManager, endpointRegistry, db)

	rentalID := "non-existent-rental"

	req := httptest.NewRequest("DELETE", "/api/dve-rental/rentals/"+rentalID+"/validation-session", nil)
	req = mux.SetURLVars(req, map[string]string{"id": rentalID})
	w := httptest.NewRecorder()

	handlers.TerminateValidationSession(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response DVERentalResponse
	unmarshalErr := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, unmarshalErr)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Rental not found or access denied")
}

func TestTerminateValidationSession_NoValidationSession(t *testing.T) {
	db, _ := buntdb.Open(":memory:")
	dveRentalService, _ := dverental.NewDVERentalService(db)
	containerOrchestrator := &container.ContainerOrchestrator{}
	sessionManager := session.NewSessionManager(db)
	endpointRegistry := &endpoints.EndpointRegistry{}

	handlers := NewDVERentalHandlers(dveRentalService, containerOrchestrator, sessionManager, endpointRegistry, db)

	rentalID := "rental-123"
	userID := "user-456"

	// Create a real SSH session but no validation session
	_, _ = handlers.sessionManager.CreateSSHSession(rentalID, "container-456", "testuser", "test-key")

	req := httptest.NewRequest("DELETE", "/api/dve-rental/rentals/"+rentalID+"/validation-session", nil)
	req = mux.SetURLVars(req, map[string]string{"id": rentalID})
	req.Header.Set("user_id", userID)
	w := httptest.NewRecorder()

	handlers.TerminateValidationSession(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response DVERentalResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Validation session not found for this rental")
}

func TestTerminateErrorResolutionSession_Success(t *testing.T) {
	db, _ := buntdb.Open(":memory:")
	dveRentalService, err := dverental.NewDVERentalService(db)
	require.NoError(t, err)
	containerOrchestrator := &container.ContainerOrchestrator{}
	sessionManager := session.NewSessionManager(db)
	endpointRegistry := &endpoints.EndpointRegistry{}

	handlers := NewDVERentalHandlers(dveRentalService, containerOrchestrator, sessionManager, endpointRegistry, db)

	rentalID := "rental-123"
	userID := "user-456"

	// Create a rental first
	rentalReq := objects.RentalRequest{
		UserID:        userID,
		PlanID:        "basic",
		Duration:      3600,
		PaymentTxHash: "test-tx",
	}
	rentalResp, err := dveRentalService.CreateRental(&rentalReq)
	require.NoError(t, err)
	require.True(t, rentalResp.Success)
	rentalID = rentalResp.RentalID

	// Set container ID for the rental to pass validation
	err = db.Update(func(tx *buntdb.Tx) error {
		val, err := tx.Get("rental:" + rentalID)
		if err != nil {
			return err
		}
		var rental objects.DVERental
		if err := json.Unmarshal([]byte(val), &rental); err != nil {
			return err
		}
		rental.ContainerID = "test-container"
		data, _ := json.Marshal(rental)
		_, _, err = tx.Set("rental:"+rentalID, string(data), nil)
		return err
	})
	require.NoError(t, err)

	// Create a real error resolution session
	supportedTypes := []string{"timeout", "network"}
	_, _ = handlers.sessionManager.CreateErrorResolutionSession(rentalID, supportedTypes)

	req := httptest.NewRequest("DELETE", "/api/dve-rental/rentals/"+rentalID+"/error-resolution-session?user_id="+userID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": rentalID})
	w := httptest.NewRecorder()

	handlers.TerminateErrorResolutionSession(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response DVERentalResponse
	unmarshalErr := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, unmarshalErr)
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "Error resolution session terminated successfully")
}

func TestTerminateErrorResolutionSession_RentalNotFound(t *testing.T) {
	dveRentalService := &dverental.DVERentalService{}
	containerOrchestrator := &container.ContainerOrchestrator{}
	db, _ := buntdb.Open(":memory:")
	sessionManager := session.NewSessionManager(db)
	endpointRegistry := &endpoints.EndpointRegistry{}

	handlers := NewDVERentalHandlers(dveRentalService, containerOrchestrator, sessionManager, endpointRegistry, db)

	rentalID := "non-existent-rental"

	req := httptest.NewRequest("DELETE", "/api/dve-rental/rentals/"+rentalID+"/error-resolution-session", nil)
	req = mux.SetURLVars(req, map[string]string{"id": rentalID})
	w := httptest.NewRecorder()

	handlers.TerminateErrorResolutionSession(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response DVERentalResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Rental not found or access denied")
}

func TestTerminateErrorResolutionSession_NoErrorResolutionSession(t *testing.T) {
	db, _ := buntdb.Open(":memory:")
	dveRentalService, _ := dverental.NewDVERentalService(db)
	containerOrchestrator := &container.ContainerOrchestrator{}
	sessionManager := session.NewSessionManager(db)
	endpointRegistry := &endpoints.EndpointRegistry{}

	handlers := NewDVERentalHandlers(dveRentalService, containerOrchestrator, sessionManager, endpointRegistry, db)

	rentalID := "rental-123"
	userID := "user-456"

	// Create a rental first
	rentalReq := objects.RentalRequest{
		UserID:        userID,
		PlanID:        "basic",
		Duration:      3600,
		PaymentTxHash: "test-tx",
	}
	rentalResp, err := dveRentalService.CreateRental(&rentalReq)
	require.NoError(t, err)
	require.True(t, rentalResp.Success)
	rentalID = rentalResp.RentalID

	// Set container ID for the rental to pass validation
	err = db.Update(func(tx *buntdb.Tx) error {
		val, err := tx.Get("rental:" + rentalID)
		if err != nil {
			return err
		}
		var rental objects.DVERental
		if err := json.Unmarshal([]byte(val), &rental); err != nil {
			return err
		}
		rental.ContainerID = "test-container"
		data, _ := json.Marshal(rental)
		_, _, err = tx.Set("rental:"+rentalID, string(data), nil)
		return err
	})
	require.NoError(t, err)

	// Create a real SSH session but no error resolution session
	_, _ = handlers.sessionManager.CreateSSHSession(rentalID, "container-456", "testuser", "test-key")

	req := httptest.NewRequest("DELETE", "/api/dve-rental/rentals/"+rentalID+"/error-resolution-session?user_id="+userID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": rentalID})
	w := httptest.NewRecorder()

	handlers.TerminateErrorResolutionSession(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response DVERentalResponse
	unmarshalErr := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, unmarshalErr)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Error resolution session not found for this rental")
}

func TestTerminateValidationSession_MissingRentalID(t *testing.T) {
	dveRentalService := &dverental.DVERentalService{}
	containerOrchestrator := &container.ContainerOrchestrator{}
	db, _ := buntdb.Open(":memory:")
	sessionManager := session.NewSessionManager(db)
	endpointRegistry := &endpoints.EndpointRegistry{}

	handlers := NewDVERentalHandlers(dveRentalService, containerOrchestrator, sessionManager, endpointRegistry, db)

	req := httptest.NewRequest("DELETE", "/api/dve-rental/rentals//validation-session", nil)
	w := httptest.NewRecorder()

	handlers.TerminateValidationSession(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response DVERentalResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Rental ID is required")
}

func TestTerminateErrorResolutionSession_MissingRentalID(t *testing.T) {
	dveRentalService := &dverental.DVERentalService{}
	containerOrchestrator := &container.ContainerOrchestrator{}
	db, _ := buntdb.Open(":memory:")
	sessionManager := session.NewSessionManager(db)
	endpointRegistry := &endpoints.EndpointRegistry{}

	handlers := NewDVERentalHandlers(dveRentalService, containerOrchestrator, sessionManager, endpointRegistry, db)

	req := httptest.NewRequest("DELETE", "/api/dve-rental/rentals//error-resolution-session", nil)
	w := httptest.NewRecorder()

	handlers.TerminateErrorResolutionSession(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response DVERentalResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Rental ID is required")
}

func TestDVERentalResponse_JSON(t *testing.T) {
	response := DVERentalResponse{
		Success:   true,
		Data:      "test data",
		Message:   "Test message",
		Error:     "",
		Timestamp: "2023-01-01T00:00:00Z",
	}

	data, err := json.Marshal(response)
	require.NoError(t, err)

	var decoded DVERentalResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, response.Success, decoded.Success)
	assert.Equal(t, response.Data, decoded.Data)
	assert.Equal(t, response.Message, decoded.Message)
	assert.Equal(t, response.Error, decoded.Error)
	assert.Equal(t, response.Timestamp, decoded.Timestamp)
}
