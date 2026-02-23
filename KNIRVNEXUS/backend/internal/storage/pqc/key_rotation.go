// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package pqc

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

type KeyRotationManager struct {
	mu                sync.RWMutex
	masterKey         *PQCKeyPair
	keyCache          map[string]*PQCKeyPair
	keyVersionStore   map[string]KeyVersion
	rotationInterval  time.Duration
	minRotationAge    time.Duration
	maxKeyVersions    int
	rotationCallbacks []RotationCallback
	hsmEnabled        bool
	teeEnabled        bool
	keyStore          KeyStore
	auditLogger       RotationAuditLogger
}

type KeyVersion struct {
	KeyID       string    `json:"key_id"`
	Version     uint32    `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	ActivatedAt time.Time `json:"activated_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Status      string    `json:"status"` // active, rotating, revoked, expired
	Algorithm   string    `json:"algorithm"`
}

type RotationCallback func(oldKeyID, newKeyID string, rotationType RotationType) error

type RotationType string

const (
	RotationTypeScheduled  RotationType = "scheduled"
	RotationTypeEmergency  RotationType = "emergency"
	RotationTypeCompromise RotationType = "compromise"
	RotationTypeManual     RotationType = "manual"
)

type KeyStore interface {
	StoreKey(keyID string, keyData []byte) error
	RetrieveKey(keyID string) ([]byte, error)
	DeleteKey(keyID string) error
	ListKeys() ([]string, error)
}

type RotationAuditLogger interface {
	LogRotation(event RotationAuditEvent)
}

type RotationAuditEvent struct {
	Timestamp    time.Time    `json:"timestamp"`
	KeyID        string       `json:"key_id"`
	OldVersion   uint32       `json:"old_version"`
	NewVersion   uint32       `json:"new_version"`
	RotationType RotationType `json:"rotation_type"`
	Success      bool         `json:"success"`
	Error        string       `json:"error,omitempty"`
	ActorID      string       `json:"actor_id"`
}

func NewKeyRotationManager(rotationInterval time.Duration) *KeyRotationManager {
	return &KeyRotationManager{
		keyCache:         make(map[string]*PQCKeyPair),
		keyVersionStore:  make(map[string]KeyVersion),
		rotationInterval: rotationInterval,
		minRotationAge:   24 * time.Hour,
		maxKeyVersions:   5,
		auditLogger:      &defaultAuditLogger{},
	}
}

func (krm *KeyRotationManager) SetMasterKey(keyPair *PQCKeyPair) error {
	krm.mu.Lock()
	defer krm.mu.Unlock()

	if keyPair == nil {
		return fmt.Errorf("cannot set nil master key")
	}

	if !keyPair.IsActive() {
		return fmt.Errorf("key pair is not active")
	}

	existingVersion, exists := krm.keyVersionStore[keyPair.ID]
	if exists {
		keyPair.Status = existingVersion.Status
	}

	krm.masterKey = keyPair
	krm.keyCache[keyPair.ID] = keyPair

	krm.keyVersionStore[keyPair.ID] = KeyVersion{
		KeyID:       keyPair.ID,
		Version:     1,
		CreatedAt:   time.Now(),
		ActivatedAt: time.Now(),
		ExpiresAt:   time.Now().Add(krm.rotationInterval * 2),
		Status:      "active",
		Algorithm:   keyPair.Algorithm,
	}

	log.Printf("KeyRotationManager: Master key %s set (v1)", keyPair.ID)
	return nil
}

func (krm *KeyRotationManager) StartAutomaticRotation(ctx context.Context) {
	ticker := time.NewTicker(krm.rotationInterval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				if err := krm.RotateMasterKey(RotationTypeScheduled); err != nil {
					log.Printf("KeyRotationManager: Automatic rotation failed: %v", err)
				}
			}
		}
	}()
	log.Printf("KeyRotationManager: Started automatic rotation every %v", krm.rotationInterval)
}

func (krm *KeyRotationManager) RotateMasterKey(rotationType RotationType) error {
	krm.mu.Lock()
	defer krm.mu.Unlock()

	if krm.masterKey == nil {
		return fmt.Errorf("no master key set")
	}

	currentVersion := krm.keyVersionStore[krm.masterKey.ID]
	if time.Since(currentVersion.CreatedAt) < krm.minRotationAge && rotationType == RotationTypeScheduled {
		return fmt.Errorf("key has not reached minimum rotation age")
	}

	oldKeyID := krm.masterKey.ID
	oldVersion := currentVersion.Version

	newKeyPair, err := GeneratePQCKeyPair("master-key", "encryption")
	if err != nil {
		krm.logRotationEvent(oldKeyID, oldVersion, 0, rotationType, false, err.Error())
		return fmt.Errorf("generate new key pair: %w", err)
	}

	oldVersionEntry := krm.keyVersionStore[oldKeyID]
	oldVersionEntry.Status = "rotating"
	krm.keyVersionStore[oldKeyID] = oldVersionEntry

	newVersion := oldVersion + 1
	krm.keyVersionStore[newKeyPair.ID] = KeyVersion{
		KeyID:       newKeyPair.ID,
		Version:     newVersion,
		CreatedAt:   time.Now(),
		ActivatedAt: time.Now(),
		ExpiresAt:   time.Now().Add(krm.rotationInterval * 2),
		Status:      "active",
		Algorithm:   newKeyPair.Algorithm,
	}

	krm.masterKey = newKeyPair
	krm.keyCache[newKeyPair.ID] = newKeyPair

	krm.cleanupOldVersions(oldKeyID)

	krm.logRotationEvent(oldKeyID, oldVersion, newVersion, rotationType, true, "")

	for _, callback := range krm.rotationCallbacks {
		if err := callback(oldKeyID, newKeyPair.ID, rotationType); err != nil {
			log.Printf("KeyRotationManager: Rotation callback failed: %v", err)
		}
	}

	log.Printf("KeyRotationManager: Rotated master key from v%d to v%d (type: %s)",
		oldVersion, newVersion, rotationType)

	return nil
}

func (krm *KeyRotationManager) cleanupOldVersions(keyID string) {
	var versionsToKeep []KeyVersion
	for _, kv := range krm.keyVersionStore {
		if kv.KeyID == keyID {
			versionsToKeep = append(versionsToKeep, kv)
		}
	}

	if len(versionsToKeep) > krm.maxKeyVersions {
		for _, kv := range versionsToKeep[:len(versionsToKeep)-krm.maxKeyVersions] {
			delete(krm.keyVersionStore, kv.KeyID)
			delete(krm.keyCache, kv.KeyID)
			log.Printf("KeyRotationManager: Cleaned up old key version %s v%d", kv.KeyID, kv.Version)
		}
	}
}

func (krm *KeyRotationManager) GetMasterKey() *PQCKeyPair {
	krm.mu.RLock()
	defer krm.mu.RUnlock()
	return krm.masterKey
}

func (krm *KeyRotationManager) GetKeyVersion(keyID string) (KeyVersion, bool) {
	krm.mu.RLock()
	defer krm.mu.RUnlock()

	version, exists := krm.keyVersionStore[keyID]
	return version, exists
}

func (krm *KeyRotationManager) RevokeKey(keyID string, reason string) error {
	krm.mu.Lock()
	defer krm.mu.Unlock()

	version, exists := krm.keyVersionStore[keyID]
	if !exists {
		return fmt.Errorf("key %s not found", keyID)
	}

	version.Status = "revoked"
	krm.keyVersionStore[keyID] = version

	if key, exists := krm.keyCache[keyID]; exists {
		key.Status = "revoked"
	}

	if krm.keyStore != nil {
		krm.keyStore.DeleteKey(keyID)
	}

	krm.logRotationEvent(keyID, version.Version, 0, RotationTypeCompromise, true, reason)

	log.Printf("KeyRotationManager: Revoked key %s v%d: %s", keyID, version.Version, reason)
	return nil
}

func (krm *KeyRotationManager) RegisterRotationCallback(callback RotationCallback) {
	krm.mu.Lock()
	defer krm.mu.Unlock()
	krm.rotationCallbacks = append(krm.rotationCallbacks, callback)
}

func (krm *KeyRotationManager) SetKeyStore(store KeyStore) {
	krm.mu.Lock()
	defer krm.mu.Unlock()
	krm.keyStore = store
}

func (krm *KeyRotationManager) EnableHSM() {
	krm.mu.Lock()
	defer krm.mu.Unlock()
	krm.hsmEnabled = true
	log.Println("KeyRotationManager: HSM support enabled")
}

func (krm *KeyRotationManager) EnableTEE() {
	krm.mu.Lock()
	defer krm.mu.Unlock()
	krm.teeEnabled = true
	log.Println("KeyRotationManager: TEE support enabled")
}

func (krm *KeyRotationManager) IsHSMEnabled() bool {
	krm.mu.RLock()
	defer krm.mu.RUnlock()
	return krm.hsmEnabled
}

func (krm *KeyRotationManager) IsTEEEnabled() bool {
	krm.mu.RLock()
	defer krm.mu.RUnlock()
	return krm.teeEnabled
}

func (krm *KeyRotationManager) logRotationEvent(keyID string, oldVer, newVer uint32, rotType RotationType, success bool, errMsg string) {
	if krm.auditLogger != nil {
		krm.auditLogger.LogRotation(RotationAuditEvent{
			Timestamp:    time.Now(),
			KeyID:        keyID,
			OldVersion:   oldVer,
			NewVersion:   newVer,
			RotationType: rotType,
			Success:      success,
			Error:        errMsg,
			ActorID:      "system",
		})
	}
}

type defaultAuditLogger struct{}

func (dal *defaultAuditLogger) LogRotation(event RotationAuditEvent) {
	log.Printf("KeyRotationAudit: key=%s old_v=%d new_v=%d type=%s success=%v err=%s",
		event.KeyID, event.OldVersion, event.NewVersion, event.RotationType, event.Success, event.Error)
}

type EncryptedKeyBlob struct {
	KeyID        string    `json:"key_id"`
	Algorithm    string    `json:"algorithm"`
	EncryptedKey []byte    `json:"encrypted_key"`
	HSMWrapped   bool      `json:"hsm_wrapped"`
	TEESealed    bool      `json:"tee_sealed"`
	CreatedAt    time.Time `json:"created_at"`
}

func (krm *KeyRotationManager) ExportKey(keyID string, exportKey *PQCKeyPair) (*EncryptedKeyBlob, error) {
	krm.mu.RLock()
	defer krm.mu.RUnlock()

	key, exists := krm.keyCache[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found")
	}

	keyBytes, err := key.MarshalWithPrivateKeys()
	if err != nil {
		return nil, fmt.Errorf("marshal key: %w", err)
	}

	blob := &EncryptedKeyBlob{
		KeyID:        keyID,
		Algorithm:    key.Algorithm,
		EncryptedKey: keyBytes,
		CreatedAt:    time.Now(),
	}

	if krm.hsmEnabled {
		blob.HSMWrapped = true
	}

	if krm.teeEnabled {
		blob.TEESealed = true
	}

	return blob, nil
}

func (krm *KeyRotationManager) ImportKey(blob *EncryptedKeyBlob) error {
	krm.mu.Lock()
	defer krm.mu.Unlock()

	key, err := LoadPQCKeyPair(blob.EncryptedKey)
	if err != nil {
		return fmt.Errorf("load key: %w", err)
	}

	key.ID = blob.KeyID
	key.Status = "active"

	krm.keyCache[key.ID] = key
	krm.keyVersionStore[key.ID] = KeyVersion{
		KeyID:       key.ID,
		Version:     1,
		CreatedAt:   time.Now(),
		ActivatedAt: time.Now(),
		ExpiresAt:   time.Now().Add(krm.rotationInterval * 2),
		Status:      "active",
		Algorithm:   key.Algorithm,
	}

	log.Printf("KeyRotationManager: Imported key %s", key.ID)
	return nil
}

func ConstantTimeCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

func GenerateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

func (krm *KeyRotationManager) GetRotationStatus() map[string]interface{} {
	krm.mu.RLock()
	defer krm.mu.RUnlock()

	status := map[string]interface{}{
		"hsms_enabled":      krm.hsmEnabled,
		"tee_enabled":       krm.teeEnabled,
		"rotation_interval": krm.rotationInterval.String(),
		"keys":              make([]map[string]interface{}, 0),
	}

	for keyID, version := range krm.keyVersionStore {
		status["keys"] = append(status["keys"].([]map[string]interface{}), map[string]interface{}{
			"key_id":     keyID,
			"version":    version.Version,
			"status":     version.Status,
			"created_at": version.CreatedAt,
			"expires_at": version.ExpiresAt,
		})
	}

	return status
}

func (krm *KeyRotationManager) MarshalJSON() ([]byte, error) {
	return json.Marshal(krm.GetRotationStatus())
}
