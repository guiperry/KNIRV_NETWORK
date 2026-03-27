package session

import (
	"testing"
	"time"

	"backend_server/internal/objects"
)

func TestNewSessionManager(t *testing.T) {
	sm := NewSessionManager(nil)

	if sm == nil {
		t.Fatal("Session manager should not be nil")
	}

	if sm.sessions == nil {
		t.Error("Sessions map should be initialized")
	}

	if len(sm.sessions) != 0 {
		t.Error("Sessions map should be empty initially")
	}
}

func TestSessionManager_CreateSSHSession(t *testing.T) {
	sm := NewSessionManager(nil)
	rentalID := "rental_123"
	containerID := "container_456"
	username := "testuser"

	session, err := sm.CreateSSHSession(rentalID, containerID, username, "test-private-key")
	if err != nil {
		t.Fatalf("Failed to create SSH session: %v", err)
	}

	if session == nil {
		t.Fatal("SSH session should not be nil")
	}

	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}

	if session.CreationID != rentalID {
		t.Errorf("Expected creation ID %s, got %s", rentalID, session.CreationID)
	}

	if session.ContainerID != containerID {
		t.Errorf("Expected container ID %s, got %s", containerID, session.ContainerID)
	}

	if session.Username != username {
		t.Errorf("Expected username %s, got %s", username, session.Username)
	}

	if session.PublicKeyHash != "" {
		t.Error("Public key hash should be empty initially")
	}

	if !session.CreatedAt.Before(time.Now()) {
		t.Error("CreatedAt should be in the past")
	}

	if session.LastUsed.IsZero() {
		t.Error("LastUsed should be set")
	}

	if session.ExpiresAt.Before(time.Now()) {
		t.Error("ExpiresAt should be in the future")
	}
}

func TestSessionManager_GetSSHSession(t *testing.T) {
	sm := NewSessionManager(nil)
	rentalID := "rental_123"
	containerID := "container_456"
	username := "testuser"

	// Create session
	session, err := sm.CreateSSHSession(rentalID, containerID, username, "test-private-key")
	if err != nil {
		t.Fatalf("Failed to create SSH session: %v", err)
	}

	// Retrieve session
	retrievedSession, err := sm.GetSSHSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to get SSH session: %v", err)
	}

	if retrievedSession.ID != session.ID {
		t.Errorf("Expected session ID %s, got %s", session.ID, retrievedSession.ID)
	}

	if retrievedSession.CreationID != rentalID {
		t.Errorf("Expected creation ID %s, got %s", rentalID, retrievedSession.CreationID)
	}

	// Test non-existent session
	_, err = sm.GetSSHSession("non_existent_session")
	if err == nil {
		t.Error("Getting non-existent session should return error")
	}
}

func TestSessionManager_UpdateSSHSessionPublicKey(t *testing.T) {
	sm := NewSessionManager(nil)
	rentalID := "rental_123"
	containerID := "container_456"
	username := "testuser"

	// Create session
	session, err := sm.CreateSSHSession(rentalID, containerID, username, "test-private-key")
	if err != nil {
		t.Fatalf("Failed to create SSH session: %v", err)
	}

	publicKeyHash := "sha256:abc123def456"

	// Update public key
	err = sm.UpdateSSHSessionPublicKey(session.ID, publicKeyHash)
	if err != nil {
		t.Fatalf("Failed to update SSH session public key: %v", err)
	}

	// Verify update
	retrievedSession, err := sm.GetSSHSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to get SSH session: %v", err)
	}

	if retrievedSession.PublicKeyHash != publicKeyHash {
		t.Errorf("Expected public key hash %s, got %s", publicKeyHash, retrievedSession.PublicKeyHash)
	}

	// Test updating non-existent session
	err = sm.UpdateSSHSessionPublicKey("non_existent_session", publicKeyHash)
	if err == nil {
		t.Error("Updating non-existent session should return error")
	}
}

func TestSessionManager_TerminateSSHSession(t *testing.T) {
	sm := NewSessionManager(nil)
	rentalID := "rental_123"
	containerID := "container_456"
	username := "testuser"

	// Create session
	session, err := sm.CreateSSHSession(rentalID, containerID, username, "test-private-key")
	if err != nil {
		t.Fatalf("Failed to create SSH session: %v", err)
	}

	// Terminate session
	err = sm.TerminateSSHSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to terminate SSH session: %v", err)
	}

	// Verify session is gone
	_, err = sm.GetSSHSession(session.ID)
	if err == nil {
		t.Error("Getting terminated session should return error")
	}

	// Test terminating non-existent session
	err = sm.TerminateSSHSession("non_existent_session")
	if err == nil {
		t.Error("Terminating non-existent session should return error")
	}
}

func TestSessionManager_CreateValidationSession(t *testing.T) {
	sm := NewSessionManager(nil)
	rentalID := "rental_123"
	validationType := "reasoning"

	session, err := sm.CreateValidationSession(rentalID, validationType)
	if err != nil {
		t.Fatalf("Failed to create validation session: %v", err)
	}

	if session == nil {
		t.Fatal("Validation session should not be nil")
	}

	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}

	if session.CreationID != rentalID {
		t.Errorf("Expected creation ID %s, got %s", rentalID, session.CreationID)
	}

	if session.SessionToken == "" {
		t.Error("Session token should not be empty")
	}

	if session.ValidationType != validationType {
		t.Errorf("Expected validation type %s, got %s", validationType, session.ValidationType)
	}

	if session.ExpiresAt.Before(time.Now()) {
		t.Error("ExpiresAt should be in the future")
	}
}

func TestSessionManager_GetValidationSession(t *testing.T) {
	sm := NewSessionManager(nil)
	rentalID := "rental_123"
	validationType := "reasoning"

	// Create session
	session, err := sm.CreateValidationSession(rentalID, validationType)
	if err != nil {
		t.Fatalf("Failed to create validation session: %v", err)
	}

	// Retrieve session
	retrievedSession, err := sm.GetValidationSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to get validation session: %v", err)
	}

	if retrievedSession.ID != session.ID {
		t.Errorf("Expected session ID %s, got %s", session.ID, retrievedSession.ID)
	}

	// Test non-existent session
	_, err = sm.GetValidationSession("non_existent_session")
	if err == nil {
		t.Error("Getting non-existent session should return error")
	}
}

func TestSessionManager_UpdateValidationEndpoint(t *testing.T) {
	sm := NewSessionManager(nil)
	rentalID := "rental_123"
	validationType := "reasoning"

	// Create session
	session, err := sm.CreateValidationSession(rentalID, validationType)
	if err != nil {
		t.Fatalf("Failed to create validation session: %v", err)
	}

	endpointURL := "http://localhost:8080"
	port := 8080

	// Update endpoint
	err = sm.UpdateValidationEndpoint(session.ID, endpointURL, port)
	if err != nil {
		t.Fatalf("Failed to update validation endpoint: %v", err)
	}

	// Verify update
	retrievedSession, err := sm.GetValidationSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to get validation session: %v", err)
	}

	if retrievedSession.EndpointURL != endpointURL {
		t.Errorf("Expected endpoint URL %s, got %s", endpointURL, retrievedSession.EndpointURL)
	}

	if retrievedSession.Port != port {
		t.Errorf("Expected port %d, got %d", port, retrievedSession.Port)
	}

	// Test updating non-existent session
	err = sm.UpdateValidationEndpoint("non_existent_session", endpointURL, port)
	if err == nil {
		t.Error("Updating non-existent session should return error")
	}
}

func TestSessionManager_TerminateValidationSession(t *testing.T) {
	sm := NewSessionManager(nil)
	rentalID := "rental_123"
	validationType := "reasoning"

	// Create session
	session, err := sm.CreateValidationSession(rentalID, validationType)
	if err != nil {
		t.Fatalf("Failed to create validation session: %v", err)
	}

	// Terminate session
	err = sm.TerminateValidationSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to terminate validation session: %v", err)
	}

	// Verify session is gone
	_, err = sm.GetValidationSession(session.ID)
	if err == nil {
		t.Error("Getting terminated session should return error")
	}

	// Test terminating non-existent session
	err = sm.TerminateValidationSession("non_existent_session")
	if err == nil {
		t.Error("Terminating non-existent session should return error")
	}
}

func TestSessionManager_GetSessionsByRentalID(t *testing.T) {
	sm := NewSessionManager(nil)
	rentalID1 := "rental_123"
	rentalID2 := "rental_456"

	// Create sessions for rental 1
	sshSession, err := sm.CreateSSHSession(rentalID1, "container_1", "user1", "test-private-key")
	if err != nil {
		t.Fatalf("Failed to create SSH session: %v", err)
	}

	valSession, err := sm.CreateValidationSession(rentalID1, "reasoning")
	if err != nil {
		t.Fatalf("Failed to create validation session: %v", err)
	}

	// Create session for rental 2
	errSession, err := sm.CreateErrorResolutionSession(rentalID2, []string{"timeout"})
	if err != nil {
		t.Fatalf("Failed to create error resolution session: %v", err)
	}

	// Get sessions for rental 1
	sessions, err := sm.GetSessionsByRentalID(rentalID1)
	if err != nil {
		t.Fatalf("Failed to get sessions by rental ID: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}

	// Verify sessions contain the expected ones
	foundSSH := false
	foundVal := false
	for _, session := range sessions {
		switch s := session.(type) {
		case *objects.SSHSession:
			if s.ID == sshSession.ID {
				foundSSH = true
			}
		case *objects.ValidationSession:
			if s.ID == valSession.ID {
				foundVal = true
			}
		}
	}

	if !foundSSH {
		t.Error("SSH session not found in results")
	}

	if !foundVal {
		t.Error("Validation session not found in results")
	}

	// Get sessions for rental 2
	sessions2, err := sm.GetSessionsByRentalID(rentalID2)
	if err != nil {
		t.Fatalf("Failed to get sessions by rental ID: %v", err)
	}

	if len(sessions2) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions2))
	}

	// Verify it's the error resolution session
	if errSess, ok := sessions2[0].(*objects.ErrorResolutionSession); !ok || errSess.ID != errSession.ID {
		t.Error("Error resolution session not found in results")
	}
}

func TestSessionManager_TerminateAllSessionsForRental(t *testing.T) {
	sm := NewSessionManager(nil)
	rentalID := "rental_123"

	// Create multiple sessions for the rental
	sshSession, err := sm.CreateSSHSession(rentalID, "container_1", "user1", "test-private-key")
	if err != nil {
		t.Fatalf("Failed to create SSH session: %v", err)
	}

	valSession, err := sm.CreateValidationSession(rentalID, "reasoning")
	if err != nil {
		t.Fatalf("Failed to create validation session: %v", err)
	}

	errSession, err := sm.CreateErrorResolutionSession(rentalID, []string{"timeout"})
	if err != nil {
		t.Fatalf("Failed to create error resolution session: %v", err)
	}

	// Terminate all sessions for the rental
	err = sm.TerminateAllSessionsForRental(rentalID)
	if err != nil {
		t.Fatalf("Failed to terminate all sessions for rental: %v", err)
	}

	// Verify all sessions are gone
	_, err = sm.GetSSHSession(sshSession.ID)
	if err == nil {
		t.Error("SSH session should be terminated")
	}

	_, err = sm.GetValidationSession(valSession.ID)
	if err == nil {
		t.Error("Validation session should be terminated")
	}

	_, err = sm.GetErrorResolutionSession(errSession.ID)
	if err == nil {
		t.Error("Error resolution session should be terminated")
	}
}

func TestSessionManager_GenerateSessionID(t *testing.T) {
	sm := NewSessionManager(nil)

	id1 := sm.generateSessionID()
	id2 := sm.generateSessionID()

	if id1 == "" {
		t.Error("Session ID should not be empty")
	}

	if id2 == "" {
		t.Error("Session ID should not be empty")
	}

	if id1 == id2 {
		t.Error("Session IDs should be unique")
	}

	if len(id1) != 32 {
		t.Errorf("Expected session ID length 32, got %d", len(id1))
	}
}

func TestSessionManager_GenerateSessionToken(t *testing.T) {
	sm := NewSessionManager(nil)

	token1 := sm.generateSessionToken()
	token2 := sm.generateSessionToken()

	if token1 == "" {
		t.Error("Session token should not be empty")
	}

	if token2 == "" {
		t.Error("Session token should not be empty")
	}

	if token1 == token2 {
		t.Error("Session tokens should be unique")
	}

	if len(token1) != 64 {
		t.Errorf("Expected session token length 64, got %d", len(token1))
	}
}

func TestSessionManager_CleanupExpiredSessions(t *testing.T) {
	sm := NewSessionManager(nil)

	// Create a session that expires immediately
	rentalID := "rental_123"
	containerID := "container_456"
	username := "testuser"

	session, err := sm.CreateSSHSession(rentalID, containerID, username, "test-private-key")
	if err != nil {
		t.Fatalf("Failed to create SSH session: %v", err)
	}

	// Manually set expiration to past
	session.ExpiresAt = time.Now().Add(-time.Hour)

	// Since cleanup runs in a goroutine, we can't easily test it synchronously
	// This test mainly ensures the session creation and expiration logic works
	// The session should be considered expired now
	if _, err := sm.GetSSHSession(session.ID); err == nil {
		t.Error("Session should be considered expired")
	}
}

func TestSessionManager_ErrorResolutionSession(t *testing.T) {
	sm := NewSessionManager(nil)
	rentalID := "rental_123"
	supportedTypes := []string{"timeout", "network", "resource"}

	session, err := sm.CreateErrorResolutionSession(rentalID, supportedTypes)
	if err != nil {
		t.Fatalf("Failed to create error resolution session: %v", err)
	}

	if session == nil {
		t.Fatal("Error resolution session should not be nil")
	}

	if len(session.SupportedTypes) != len(supportedTypes) {
		t.Errorf("Expected %d supported types, got %d", len(supportedTypes), len(session.SupportedTypes))
	}

	for i, expectedType := range supportedTypes {
		if session.SupportedTypes[i] != expectedType {
			t.Errorf("Expected supported type %s, got %s", expectedType, session.SupportedTypes[i])
		}
	}

	// Test getting the session
	retrievedSession, err := sm.GetErrorResolutionSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to get error resolution session: %v", err)
	}

	if retrievedSession.ID != session.ID {
		t.Errorf("Expected session ID %s, got %s", session.ID, retrievedSession.ID)
	}

	// Test updating endpoint
	endpointURL := "http://localhost:9090"
	port := 9090
	err = sm.UpdateErrorResolutionEndpoint(session.ID, endpointURL, port)
	if err != nil {
		t.Fatalf("Failed to update error resolution endpoint: %v", err)
	}

	retrievedSession, err = sm.GetErrorResolutionSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to get error resolution session: %v", err)
	}

	if retrievedSession.EndpointURL != endpointURL {
		t.Errorf("Expected endpoint URL %s, got %s", endpointURL, retrievedSession.EndpointURL)
	}

	if retrievedSession.Port != port {
		t.Errorf("Expected port %d, got %d", port, retrievedSession.Port)
	}

	// Test terminating the session
	err = sm.TerminateErrorResolutionSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to terminate error resolution session: %v", err)
	}

	_, err = sm.GetErrorResolutionSession(session.ID)
	if err == nil {
		t.Error("Getting terminated session should return error")
	}
}
