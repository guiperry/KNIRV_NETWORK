package session

import (
	"fmt"
	"time"

	"backend_server/internal/objects"
)

// MockSessionManager is a mock implementation of SessionManager for testing
type MockSessionManager struct {
	sessions map[string]interface{}
}

func NewMockSessionManager() *MockSessionManager {
	return &MockSessionManager{
		sessions: make(map[string]interface{}),
	}
}

// Additional methods to match the actual SessionManager interface
func (m *MockSessionManager) CreateSSHSession(rentalID, containerID, username, privateKey string) (*objects.SSHSession, error) {
	sessionID := fmt.Sprintf("ssh-session-%s", rentalID)
	session := &objects.SSHSession{
		ID:            sessionID,
		RentalID:      rentalID,
		ContainerID:   containerID,
		Username:      username,
		PrivateKey:    privateKey,
		PrivateKeyURL: fmt.Sprintf("/api/sessions/ssh/%s/private-key", sessionID),
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		CreatedAt:     time.Now(),
		LastUsed:      time.Now(),
	}

	m.sessions[sessionID] = session
	return session, nil
}

func (m *MockSessionManager) CreateValidationSession(rentalID string, validationType string) (*objects.ValidationSession, error) {
	sessionID := fmt.Sprintf("val-session-%s", rentalID)
	session := &objects.ValidationSession{
		ID:             sessionID,
		RentalID:       rentalID,
		SessionToken:   fmt.Sprintf("token-%s", sessionID),
		EndpointURL:    "http://10.0.1.42:23145",
		Port:           23145,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		CreatedAt:      time.Now(),
		ValidationType: validationType,
	}

	m.sessions[sessionID] = session
	return session, nil
}

func (m *MockSessionManager) CreateErrorResolutionSession(rentalID string, supportedTypes []string) (*objects.ErrorResolutionSession, error) {
	sessionID := fmt.Sprintf("err-session-%s", rentalID)
	session := &objects.ErrorResolutionSession{
		ID:             sessionID,
		RentalID:       rentalID,
		SessionToken:   fmt.Sprintf("token-%s", sessionID),
		EndpointURL:    "http://10.0.1.42:24145",
		Port:           24145,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		CreatedAt:      time.Now(),
		SupportedTypes: supportedTypes,
	}

	m.sessions[sessionID] = session
	return session, nil
}
