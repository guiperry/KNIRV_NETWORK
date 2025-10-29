package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/objects"
)

// SessionManager manages access sessions for DVE rentals
type SessionManager struct {
	sessions map[string]interface{} // sessionID -> session object
	mutex    sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]interface{}),
	}

	// Start cleanup routine
	go sm.cleanupExpiredSessions()

	return sm
}

// CreateSSHSession creates a new SSH session for a rental
func (sm *SessionManager) CreateSSHSession(rentalID, containerID, username string) (*objects.SSHSession, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sessionID := sm.generateSessionID()
	expiresAt := time.Now().Add(24 * time.Hour) // 24 hours

	session := &objects.SSHSession{
		ID:            sessionID,
		RentalID:      rentalID,
		ContainerID:   containerID,
		Username:      username,
		PublicKeyHash: "", // Will be set when keys are generated
		PrivateKeyURL: fmt.Sprintf("/api/sessions/ssh/%s/private-key", sessionID),
		ExpiresAt:     expiresAt,
		CreatedAt:     time.Now(),
		LastUsed:      time.Now(),
	}

	sm.sessions[sessionID] = session

	log.Printf("Created SSH session %s for rental %s", sessionID, rentalID)
	return session, nil
}

// GetSSHSession retrieves an SSH session by ID
func (sm *SessionManager) GetSSHSession(sessionID string) (*objects.SSHSession, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("SSH session not found: %s", sessionID)
	}

	sshSession, ok := session.(*objects.SSHSession)
	if !ok {
		return nil, fmt.Errorf("invalid session type for SSH session: %s", sessionID)
	}

	// Check if expired
	if time.Now().After(sshSession.ExpiresAt) {
		return nil, fmt.Errorf("SSH session expired: %s", sessionID)
	}

	// Update last used
	sshSession.LastUsed = time.Now()
	return sshSession, nil
}

// UpdateSSHSessionPublicKey updates the public key hash for an SSH session
func (sm *SessionManager) UpdateSSHSessionPublicKey(sessionID, publicKeyHash string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("SSH session not found: %s", sessionID)
	}

	sshSession, ok := session.(*objects.SSHSession)
	if !ok {
		return fmt.Errorf("invalid session type for SSH session: %s", sessionID)
	}

	sshSession.PublicKeyHash = publicKeyHash
	log.Printf("Updated public key hash for SSH session %s", sessionID)
	return nil
}

// TerminateSSHSession terminates an SSH session
func (sm *SessionManager) TerminateSSHSession(sessionID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("SSH session not found: %s", sessionID)
	}

	if _, ok := session.(*objects.SSHSession); !ok {
		return fmt.Errorf("invalid session type for SSH session: %s", sessionID)
	}

	delete(sm.sessions, sessionID)
	log.Printf("Terminated SSH session %s", sessionID)
	return nil
}

// CreateValidationSession creates a new reasoning validation session
func (sm *SessionManager) CreateValidationSession(rentalID string, validationType string) (*objects.ValidationSession, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sessionID := sm.generateSessionID()
	sessionToken := sm.generateSessionToken()
	expiresAt := time.Now().Add(24 * time.Hour)

	session := &objects.ValidationSession{
		ID:             sessionID,
		RentalID:       rentalID,
		SessionToken:   sessionToken,
		EndpointURL:    "", // Will be set by endpoint registry
		Port:           0,  // Will be set by endpoint registry
		ExpiresAt:      expiresAt,
		CreatedAt:      time.Now(),
		ValidationType: validationType,
	}

	sm.sessions[sessionID] = session

	log.Printf("Created validation session %s for rental %s", sessionID, rentalID)
	return session, nil
}

// GetValidationSession retrieves a validation session by ID
func (sm *SessionManager) GetValidationSession(sessionID string) (*objects.ValidationSession, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("validation session not found: %s", sessionID)
	}

	valSession, ok := session.(*objects.ValidationSession)
	if !ok {
		return nil, fmt.Errorf("invalid session type for validation session: %s", sessionID)
	}

	if time.Now().After(valSession.ExpiresAt) {
		return nil, fmt.Errorf("validation session expired: %s", sessionID)
	}

	return valSession, nil
}

// UpdateValidationEndpoint updates the endpoint information for a validation session
func (sm *SessionManager) UpdateValidationEndpoint(sessionID, endpointURL string, port int) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("validation session not found: %s", sessionID)
	}

	valSession, ok := session.(*objects.ValidationSession)
	if !ok {
		return fmt.Errorf("invalid session type for validation session: %s", sessionID)
	}

	valSession.EndpointURL = endpointURL
	valSession.Port = port
	log.Printf("Updated endpoint for validation session %s: %s:%d", sessionID, endpointURL, port)
	return nil
}

// TerminateValidationSession terminates a validation session
func (sm *SessionManager) TerminateValidationSession(sessionID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("validation session not found: %s", sessionID)
	}

	if _, ok := session.(*objects.ValidationSession); !ok {
		return fmt.Errorf("invalid session type for validation session: %s", sessionID)
	}

	delete(sm.sessions, sessionID)
	log.Printf("Terminated validation session %s", sessionID)
	return nil
}

// CreateErrorResolutionSession creates a new error resolution session
func (sm *SessionManager) CreateErrorResolutionSession(rentalID string, supportedTypes []string) (*objects.ErrorResolutionSession, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sessionID := sm.generateSessionID()
	sessionToken := sm.generateSessionToken()
	expiresAt := time.Now().Add(24 * time.Hour)

	session := &objects.ErrorResolutionSession{
		ID:             sessionID,
		RentalID:       rentalID,
		SessionToken:   sessionToken,
		EndpointURL:    "", // Will be set by endpoint registry
		Port:           0,  // Will be set by endpoint registry
		ExpiresAt:      expiresAt,
		CreatedAt:      time.Now(),
		SupportedTypes: supportedTypes,
	}

	sm.sessions[sessionID] = session

	log.Printf("Created error resolution session %s for rental %s", sessionID, rentalID)
	return session, nil
}

// GetErrorResolutionSession retrieves an error resolution session by ID
func (sm *SessionManager) GetErrorResolutionSession(sessionID string) (*objects.ErrorResolutionSession, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("error resolution session not found: %s", sessionID)
	}

	errSession, ok := session.(*objects.ErrorResolutionSession)
	if !ok {
		return nil, fmt.Errorf("invalid session type for error resolution session: %s", sessionID)
	}

	if time.Now().After(errSession.ExpiresAt) {
		return nil, fmt.Errorf("error resolution session expired: %s", sessionID)
	}

	return errSession, nil
}

// UpdateErrorResolutionEndpoint updates the endpoint information for an error resolution session
func (sm *SessionManager) UpdateErrorResolutionEndpoint(sessionID, endpointURL string, port int) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("error resolution session not found: %s", sessionID)
	}

	errSession, ok := session.(*objects.ErrorResolutionSession)
	if !ok {
		return fmt.Errorf("invalid session type for error resolution session: %s", sessionID)
	}

	errSession.EndpointURL = endpointURL
	errSession.Port = port
	log.Printf("Updated endpoint for error resolution session %s: %s:%d", sessionID, endpointURL, port)
	return nil
}

// TerminateErrorResolutionSession terminates an error resolution session
func (sm *SessionManager) TerminateErrorResolutionSession(sessionID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("error resolution session not found: %s", sessionID)
	}

	if _, ok := session.(*objects.ErrorResolutionSession); !ok {
		return fmt.Errorf("invalid session type for error resolution session: %s", sessionID)
	}

	delete(sm.sessions, sessionID)
	log.Printf("Terminated error resolution session %s", sessionID)
	return nil
}

// GetSessionsByRentalID returns all sessions for a rental
func (sm *SessionManager) GetSessionsByRentalID(rentalID string) ([]interface{}, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	sessions := make([]interface{}, 0)
	for _, session := range sm.sessions {
		switch s := session.(type) {
		case *objects.SSHSession:
			if s.RentalID == rentalID {
				sessions = append(sessions, s)
			}
		case *objects.ValidationSession:
			if s.RentalID == rentalID {
				sessions = append(sessions, s)
			}
		case *objects.ErrorResolutionSession:
			if s.RentalID == rentalID {
				sessions = append(sessions, s)
			}
		}
	}

	return sessions, nil
}

// TerminateAllSessionsForRental terminates all sessions for a rental
func (sm *SessionManager) TerminateAllSessionsForRental(rentalID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sessionIDsToDelete := make([]string, 0)
	for sessionID, session := range sm.sessions {
		switch s := session.(type) {
		case *objects.SSHSession:
			if s.RentalID == rentalID {
				sessionIDsToDelete = append(sessionIDsToDelete, sessionID)
			}
		case *objects.ValidationSession:
			if s.RentalID == rentalID {
				sessionIDsToDelete = append(sessionIDsToDelete, sessionID)
			}
		case *objects.ErrorResolutionSession:
			if s.RentalID == rentalID {
				sessionIDsToDelete = append(sessionIDsToDelete, sessionID)
			}
		}
	}

	for _, sessionID := range sessionIDsToDelete {
		delete(sm.sessions, sessionID)
		log.Printf("Terminated session %s for rental %s", sessionID, rentalID)
	}

	if len(sessionIDsToDelete) > 0 {
		log.Printf("Terminated %d sessions for rental %s", len(sessionIDsToDelete), rentalID)
	}

	return nil
}

// generateSessionID generates a unique session ID
func (sm *SessionManager) generateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateSessionToken generates a session token
func (sm *SessionManager) generateSessionToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// cleanupExpiredSessions periodically cleans up expired sessions
func (sm *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.mutex.Lock()
		now := time.Now()
		expiredSessions := make([]string, 0)

		for sessionID, session := range sm.sessions {
			var expiresAt time.Time
			switch s := session.(type) {
			case *objects.SSHSession:
				expiresAt = s.ExpiresAt
			case *objects.ValidationSession:
				expiresAt = s.ExpiresAt
			case *objects.ErrorResolutionSession:
				expiresAt = s.ExpiresAt
			}

			if now.After(expiresAt) {
				expiredSessions = append(expiredSessions, sessionID)
			}
		}

		for _, sessionID := range expiredSessions {
			delete(sm.sessions, sessionID)
		}

		sm.mutex.Unlock()

		if len(expiredSessions) > 0 {
			log.Printf("Cleaned up %d expired sessions", len(expiredSessions))
		}
	}
}
