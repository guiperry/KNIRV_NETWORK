package secrets

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/database"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

type SecretType string

const (
	SecretTypeAPIKey      SecretType = "api_key"
	SecretTypePassword    SecretType = "password"
	SecretTypeToken       SecretType = "token"
	SecretTypeCertificate SecretType = "certificate"
	SecretTypePrivateKey  SecretType = "private_key"
)

type Secret struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        SecretType             `json:"type"`
	OwnerID     string                 `json:"owner_id"`
	Value       string                 `json:"-"` // Not serialized
	Encrypted   string                 `json:"encrypted"`
	KeyID       string                 `json:"key_id"`
	SessionOnly bool                   `json:"session_only"`
	SessionID   string                 `json:"session_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	LastUsed    *time.Time             `json:"last_used,omitempty"`
	UseCount    int                    `json:"use_count"`
}

type SecretSession struct {
	ID           string     `json:"id"`
	NodeID       string     `json:"node_id"`
	OwnerID      string     `json:"owner_id"`
	ChainSession string     `json:"chain_session"`
	Status       string     `json:"status"`
	Secrets      []string   `json:"secrets"` // Secret IDs
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    time.Time  `json:"expires_at"`
	LastActivity *time.Time `json:"last_activity,omitempty"`
	Permissions  []string   `json:"permissions"`
}

type SecretRetrievalRequest struct {
	SessionID  string   `json:"session_id"`
	SecretKeys []string `json:"secret_keys"`
}

type SecretRetrievalResponse struct {
	SessionID string            `json:"session_id"`
	Secrets   map[string]string `json:"secrets"`
	Errors    map[string]string `json:"errors,omitempty"`
}

type SecretManager struct {
	db            *database.BuntDBManager
	secrets       map[string]*Secret
	sessions      map[string]*SecretSession
	encryptionKey []byte
	mu            sync.RWMutex
	defaultTTL    time.Duration
}

func NewSecretManager(db *database.BuntDBManager) *SecretManager {
	return &SecretManager{
		db:         db,
		secrets:    make(map[string]*Secret),
		sessions:   make(map[string]*SecretSession),
		defaultTTL: 24 * time.Hour,
	}
}

func (sm *SecretManager) SetEncryptionKey(key []byte) {
	sm.encryptionKey = key
}

func (sm *SecretManager) CreateSecret(name string, secretType SecretType, ownerID string, value string, sessionOnly bool, metadata map[string]interface{}) (*Secret, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	secret := &Secret{
		ID:          fmt.Sprintf("sec_%s", uuid.New().String()[:12]),
		Name:        name,
		Type:        secretType,
		OwnerID:     ownerID,
		Value:       value,
		SessionOnly: sessionOnly,
		Metadata:    metadata,
		CreatedAt:   time.Now(),
		UseCount:    0,
	}

	if sm.encryptionKey != nil {
		encrypted, err := sm.encrypt(value, sm.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt secret: %w", err)
		}
		secret.Encrypted = encrypted
		secret.KeyID = "local-key"
	}

	sm.secrets[secret.ID] = secret

	if err := sm.saveSecret(secret); err != nil {
		log.Printf("Warning: Failed to save secret: %v", err)
	}

	log.Printf("Created secret %s (type: %s, owner: %s)", secret.ID, secretType, ownerID)
	return secret, nil
}

func (sm *SecretManager) CreateSession(nodeID, ownerID, chainSession string, permissions []string) (*SecretSession, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := &SecretSession{
		ID:           fmt.Sprintf("ss_%s", uuid.New().String()[:12]),
		NodeID:       nodeID,
		OwnerID:      ownerID,
		ChainSession: chainSession,
		Status:       "active",
		Secrets:      []string{},
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(sm.defaultTTL),
		Permissions:  permissions,
	}

	sm.sessions[session.ID] = session

	if err := sm.saveSession(session); err != nil {
		log.Printf("Warning: Failed to save session: %v", err)
	}

	log.Printf("Created secret session %s (node: %s, owner: %s)", session.ID, nodeID, ownerID)
	return session, nil
}

func (sm *SecretManager) AddSecretToSession(sessionID, secretID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	secret, ok := sm.secrets[secretID]
	if !ok {
		return fmt.Errorf("secret not found: %s", secretID)
	}

	if secret.OwnerID != session.OwnerID && session.OwnerID != "system" {
		return fmt.Errorf("unauthorized: secret owner mismatch")
	}

	session.Secrets = append(session.Secrets, secretID)
	return sm.saveSession(session)
}

func (sm *SecretManager) RetrieveSecrets(sessionID string, secretKeys []string) (*SecretRetrievalResponse, error) {
	sm.mu.Lock()
	session, ok := sm.sessions[sessionID]
	sm.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	if session.Status != "active" {
		return nil, fmt.Errorf("session not active: %s", session.Status)
	}

	if time.Now().After(session.ExpiresAt) {
		sm.mu.Lock()
		session.Status = "expired"
		sm.mu.Unlock()
		return nil, fmt.Errorf("session expired")
	}

	response := &SecretRetrievalResponse{
		SessionID: sessionID,
		Secrets:   make(map[string]string),
		Errors:    make(map[string]string),
	}

	secretMap := make(map[string]bool)
	for _, id := range session.Secrets {
		secretMap[id] = true
	}

	for _, key := range secretKeys {
		secret, ok := sm.getSecretByName(key)
		if !ok {
			if !secretMap[key] {
				response.Errors[key] = "secret not in session"
				continue
			}
		}

		if secret == nil {
			sm.mu.RLock()
			secret, ok = sm.secrets[key]
			sm.mu.RUnlock()
		}

		if !ok {
			response.Errors[key] = "secret not found"
			continue
		}

		if secret.SessionOnly && secret.SessionID != "" && secret.SessionID != sessionID {
			response.Errors[key] = "secret is session-bound to different session"
			continue
		}

		value := secret.Value
		if secret.Encrypted != "" && sm.encryptionKey != nil {
			decrypted, err := sm.decrypt(secret.Encrypted, sm.encryptionKey)
			if err != nil {
				response.Errors[key] = fmt.Sprintf("decryption failed: %v", err)
				continue
			}
			value = string(decrypted)
		}

		response.Secrets[key] = value

		sm.mu.Lock()
		secret.UseCount++
		now := time.Now()
		secret.LastUsed = &now
		sm.mu.Unlock()
	}

	sm.mu.Lock()
	now := time.Now()
	session.LastActivity = &now
	sm.mu.Unlock()

	return response, nil
}

func (sm *SecretManager) RotateSession(sessionID string) (*SecretSession, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	session.ExpiresAt = time.Now().Add(sm.defaultTTL)
	session.LastActivity = new(time.Time)
	*session.LastActivity = time.Now()

	if err := sm.saveSession(session); err != nil {
		return nil, fmt.Errorf("failed to rotate session: %w", err)
	}

	log.Printf("Rotated secret session %s, new expiry: %s", sessionID, session.ExpiresAt.Format(time.RFC3339))
	return session, nil
}

func (sm *SecretManager) RevokeSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Status = "revoked"

	if err := sm.saveSession(session); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	log.Printf("Revoked secret session %s", sessionID)
	return nil
}

func (sm *SecretManager) DeleteSecret(secretID, ownerID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	secret, ok := sm.secrets[secretID]
	if !ok {
		return fmt.Errorf("secret not found: %s", secretID)
	}

	if secret.OwnerID != ownerID {
		return fmt.Errorf("unauthorized: not the owner of this secret")
	}

	delete(sm.secrets, secretID)

	return sm.db.Transaction(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(fmt.Sprintf("secret:%s", secretID))
		return err
	})
}

func (sm *SecretManager) GetSession(sessionID string) (*SecretSession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	session, ok := sm.sessions[sessionID]
	return session, ok
}

func (sm *SecretManager) ListSecrets(ownerID string) []*Secret {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var secrets []*Secret
	for _, secret := range sm.secrets {
		if ownerID == "" || secret.OwnerID == ownerID {
			secrets = append(secrets, secret)
		}
	}
	return secrets
}

func (sm *SecretManager) ListSessions(ownerID string) []*SecretSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var sessions []*SecretSession
	for _, session := range sm.sessions {
		if ownerID == "" || session.OwnerID == ownerID {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

func (sm *SecretManager) getSecretByName(name string) (*Secret, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, secret := range sm.secrets {
		if secret.Name == name {
			return secret, true
		}
	}
	return nil, false
}

func (sm *SecretManager) encrypt(plaintext string, key []byte) (string, error) {
	encrypted := make([]byte, len(plaintext))
	for i, b := range []byte(plaintext) {
		encrypted[i] = b ^ key[i%len(key)]
	}
	return fmt.Sprintf("%x", encrypted), nil
}

func (sm *SecretManager) decrypt(ciphertext string, key []byte) ([]byte, error) {
	var encrypted []byte
	_, err := fmt.Sscanf(ciphertext, "%x", &encrypted)
	if err != nil {
		encrypted = []byte(ciphertext)
	}
	decrypted := make([]byte, len(encrypted))
	for i, b := range encrypted {
		decrypted[i] = b ^ key[i%len(key)]
	}
	return decrypted, nil
}

func (sm *SecretManager) saveSecret(secret *Secret) error {
	if sm.db == nil {
		return nil
	}

	data, err := json.Marshal(secret)
	if err != nil {
		return err
	}

	return sm.db.Transaction(func(tx *buntdb.Tx) error {
		_, _, err = tx.Set(fmt.Sprintf("secret:%s", secret.ID), string(data), nil)
		return err
	})
}

func (sm *SecretManager) saveSession(session *SecretSession) error {
	if sm.db == nil {
		return nil
	}

	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return sm.db.Transaction(func(tx *buntdb.Tx) error {
		_, _, err = tx.Set(fmt.Sprintf("secret:session:%s", session.ID), string(data), nil)
		return err
	})
}

func (sm *SecretManager) LoadSecrets() error {
	if sm.db == nil {
		return nil
	}

	return sm.db.GetObjectsByPrefix("secret:", func(key string, value []byte) bool {
		var secret Secret
		if err := json.Unmarshal(value, &secret); err != nil {
			return true
		}
		sm.mu.Lock()
		sm.secrets[secret.ID] = &secret
		sm.mu.Unlock()
		return true
	})
}

func (sm *SecretManager) LoadSessions() error {
	if sm.db == nil {
		return nil
	}

	return sm.db.GetObjectsByPrefix("secret:session:", func(key string, value []byte) bool {
		var session SecretSession
		if err := json.Unmarshal(value, &session); err != nil {
			return true
		}
		sm.mu.Lock()
		sm.sessions[session.ID] = &session
		sm.mu.Unlock()
		return true
	})
}

func (sm *SecretManager) CleanupExpiredSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for id, session := range sm.sessions {
		if session.Status == "active" && now.After(session.ExpiresAt) {
			session.Status = "expired"
			log.Printf("Cleaned up expired secret session: %s", id)
		}
	}
}
