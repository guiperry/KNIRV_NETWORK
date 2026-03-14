package session

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/sessions"
)

const (
	SessionName      = "knirv_session"
	ControllerURLKey = "controller_url"
)

// Manager handles session management
type Manager struct {
	store *sessions.CookieStore
	data  map[string]*SessionData
	mu    sync.RWMutex
}

// SessionData holds session-specific data
type SessionData struct {
	ControllerURL string
}

// NewManager creates a new session manager
func NewManager(secret string) *Manager {
	return &Manager{
		store: sessions.NewCookieStore([]byte(secret)),
		data:  make(map[string]*SessionData),
	}
}

// GetOrCreate gets or creates a session
func (m *Manager) GetOrCreate(r *http.Request, w http.ResponseWriter) (*sessions.Session, error) {
	session, err := m.store.Get(r, SessionName)
	if err != nil {
		// Create new session if get fails
		session, _ = m.store.New(r, SessionName)
	}

	// Ensure session has an ID
	if session.ID == "" {
		session.ID = generateSessionID()
	}

	// Initialize session data if not exists
	m.mu.Lock()
	if _, exists := m.data[session.ID]; !exists {
		m.data[session.ID] = &SessionData{}
	}
	m.mu.Unlock()

	return session, nil
}

// GetControllerURL retrieves the controller URL from session
func (m *Manager) GetControllerURL(sessionID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if data, exists := m.data[sessionID]; exists {
		return data.ControllerURL, data.ControllerURL != ""
	}

	return "", false
}

// SetControllerURL sets the controller URL in session
func (m *Manager) SetControllerURL(sessionID, controllerURL string) error {
	// Validate URL
	u, err := url.Parse(controllerURL)
	if err != nil {
		return err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return &URLValidationError{Message: "controller URL must use http or https"}
	}

	// Normalize URL (remove trailing slash)
	u.Path = "/"
	normalizedURL := u.String()
	if normalizedURL[len(normalizedURL)-1] == '/' {
		normalizedURL = normalizedURL[:len(normalizedURL)-1]
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.data[sessionID]; !exists {
		m.data[sessionID] = &SessionData{}
	}

	m.data[sessionID].ControllerURL = normalizedURL
	return nil
}

// ClearControllerURL clears the controller URL from session
func (m *Manager) ClearControllerURL(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if data, exists := m.data[sessionID]; exists {
		data.ControllerURL = ""
	}
}

// generateSessionID generates a random session ID
func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// URLValidationError represents a URL validation error
type URLValidationError struct {
	Message string
}

func (e *URLValidationError) Error() string {
	return e.Message
}
