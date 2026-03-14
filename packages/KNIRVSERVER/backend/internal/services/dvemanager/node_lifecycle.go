package dvemanager

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"backend_server/internal/objects"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

// RegisterDVENode registers a new DVE node derived from a DVECreationRequest.
//
// The method creates a DVECreation record and an initial DVESession, persists
// both to the BuntDB database using the same key prefixes as
// dvecreation.DVECreationService ("dve_creation:<id>" and "dve_session:<id>"),
// and also registers the corresponding DVENode with the node tracker so the
// node appears in dvemanager queries immediately.
//
// This method is intentionally self-contained: it only uses dm.db and
// dm.nodeTracker so it does not add new dependencies to DVEManager.
func (dm *DVEManager) RegisterDVENode(req *objects.DVECreationRequest) (*objects.DVECreation, error) {
	if req == nil {
		return nil, fmt.Errorf("DVECreationRequest must not be nil")
	}
	if req.OwnerID == "" || req.OwnerAddress == "" {
		return nil, fmt.Errorf("owner ID and address are required")
	}
	if req.StakeAmount <= 0 {
		return nil, fmt.Errorf("stake amount must be positive")
	}

	dm.mu.Lock()
	defer dm.mu.Unlock()

	dveNodeID := fmt.Sprintf("dve-%s", uuid.New().String()[:12])
	creationID := uuid.New().String()
	sessionKeyID := uuid.New().String()

	now := time.Now()

	creation := &objects.DVECreation{
		ID:             creationID,
		Name:           req.Name,
		OwnerID:        req.OwnerID,
		OwnerAddress:   req.OwnerAddress,
		DVENodeID:      dveNodeID,
		StakeAmount:    req.StakeAmount,
		Status:         "active",
		TEEType:        req.TEEType,
		TEEAttestation: req.TEEAttestation,
		SessionKeyID:   sessionKeyID,
		Capabilities:   req.Capabilities,
		ResourceLimits: req.ResourceLimits,
		RegisteredAt:   now,
		ActivatedAt:    &now,
		LastHeartbeat:  now,
		UpdatedAt:      now,
		Persistent:     req.Persistent,
		// Default grace period: 7 days
		GracePeriod: 7 * 24 * 60 * 60,
	}

	// Persist the creation record.
	if err := dm.saveCreation(creation); err != nil {
		return nil, fmt.Errorf("failed to persist DVE creation: %w", err)
	}

	// Create an initial active session.
	session := &objects.DVESession{
		ID:            uuid.New().String(),
		DVECreationID: creation.ID,
		DVENodeID:     dveNodeID,
		OwnerID:       creation.OwnerID,
		SessionKey:    generateLifecycleSessionKey(),
		SessionKeyID:  sessionKeyID,
		TEEBinding:    req.TEEAttestation,
		Status:        "active",
		CreatedAt:     now,
		ExpiresAt:     now.Add(24 * time.Hour),
		LastValidated: now,
	}

	if err := dm.saveSession(session); err != nil {
		// Non-fatal: creation is already persisted; log and continue.
		log.Printf("[DVEManager/lifecycle] Warning: failed to persist initial session for node %s: %v",
			dveNodeID, err)
	}

	// Register the underlying DVENode so it appears in manager queries.
	node := &objects.DVENode{
		ID:              dveNodeID,
		Name:            req.Name,
		Status:          "online",
		TEEType:         req.TEEType,
		StakeAmount:     req.StakeAmount,
		ReputationScore: 100,
		Capabilities:    req.Capabilities,
		LastHeartbeat:   now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := dm.storeNode(node); err != nil {
		log.Printf("[DVEManager/lifecycle] Warning: failed to store DVENode %s: %v", dveNodeID, err)
	}
	dm.nodeTracker.AddNode(node)

	log.Printf("[DVEManager/lifecycle] Registered DVE node %s (creation %s) for owner %s",
		dveNodeID, creation.ID, req.OwnerID)

	return creation, nil
}

// Heartbeat records a keep-alive for the DVE node identified by nodeID.
//
// It updates LastHeartbeat on the persisted DVECreation record (looked up by
// DVENodeID), refreshes the in-memory node tracker, and updates the
// LastValidated timestamp on any active DVESession for that node.
func (dm *DVEManager) Heartbeat(nodeID string) error {
	if nodeID == "" {
		return fmt.Errorf("nodeID must not be empty")
	}

	now := time.Now()

	// Update the in-memory node tracker heartbeat.
	dm.nodeTracker.UpdateHeartbeat(nodeID)

	// Update the DVECreation record stored under this nodeID.
	var foundCreation *objects.DVECreation
	err := dm.db.GetObjectsByPrefix("dve_creation:", func(key string, value []byte) bool {
		var c objects.DVECreation
		if jsonErr := json.Unmarshal(value, &c); jsonErr != nil {
			return true // continue
		}
		if c.DVENodeID == nodeID {
			foundCreation = &c
			return false // stop iterating
		}
		return true
	})
	if err != nil {
		return fmt.Errorf("error scanning creations for node %s: %w", nodeID, err)
	}

	if foundCreation != nil {
		foundCreation.LastHeartbeat = now
		foundCreation.UpdatedAt = now
		if saveErr := dm.saveCreation(foundCreation); saveErr != nil {
			log.Printf("[DVEManager/lifecycle] Warning: failed to update heartbeat for creation %s: %v",
				foundCreation.ID, saveErr)
		}

		// Also refresh the matching session's LastValidated.
		dm.refreshSessionHeartbeat(foundCreation.ID, now)
	}

	log.Printf("[DVEManager/lifecycle] Heartbeat recorded for node %s", nodeID)
	return nil
}

// GetNodeSession returns the most recently created active DVESession for the
// given nodeID.  It scans the database for sessions whose DVENodeID matches
// and Status == "active", returning the one with the latest CreatedAt.
//
// Returns an error wrapping ErrSessionNotFound if no active session exists.
func (dm *DVEManager) GetNodeSession(nodeID string) (*objects.DVESession, error) {
	if nodeID == "" {
		return nil, fmt.Errorf("nodeID must not be empty")
	}

	var best *objects.DVESession

	err := dm.db.GetObjectsByPrefix("dve_session:", func(key string, value []byte) bool {
		var s objects.DVESession
		if jsonErr := json.Unmarshal(value, &s); jsonErr != nil {
			return true
		}
		if s.DVENodeID != nodeID || s.Status != "active" {
			return true
		}
		if best == nil || s.CreatedAt.After(best.CreatedAt) {
			copy := s
			best = &copy
		}
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("error scanning sessions for node %s: %w", nodeID, err)
	}

	if best == nil {
		return nil, fmt.Errorf("no active session found for node %s", nodeID)
	}

	return best, nil
}

// -----------------------------------------------------------------------
// Internal helpers (not exported; only used within the dvemanager package)
// -----------------------------------------------------------------------

// saveCreation persists a DVECreation to BuntDB using the standard
// "dve_creation:<id>" key that dvecreation.DVECreationService also uses.
func (dm *DVEManager) saveCreation(c *objects.DVECreation) error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal DVECreation: %w", err)
	}
	return dm.db.Transaction(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set("dve_creation:"+c.ID, string(data), nil)
		return err
	})
}

// saveSession persists a DVESession to BuntDB using the standard
// "dve_session:<id>" key that dvecreation.DVECreationService also uses.
func (dm *DVEManager) saveSession(s *objects.DVESession) error {
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal DVESession: %w", err)
	}
	return dm.db.Transaction(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set("dve_session:"+s.ID, string(data), nil)
		return err
	})
}

// refreshSessionHeartbeat updates LastValidated on all active sessions
// belonging to creationID.
func (dm *DVEManager) refreshSessionHeartbeat(creationID string, now time.Time) {
	_ = dm.db.GetObjectsByPrefix("dve_session:", func(key string, value []byte) bool {
		var s objects.DVESession
		if jsonErr := json.Unmarshal(value, &s); jsonErr != nil {
			return true
		}
		if s.DVECreationID != creationID || s.Status != "active" {
			return true
		}
		s.LastValidated = now
		if saveErr := dm.saveSession(&s); saveErr != nil {
			log.Printf("[DVEManager/lifecycle] Warning: failed to refresh session %s: %v", s.ID, saveErr)
		}
		return true
	})
}

// generateLifecycleSessionKey returns 32 random bytes to use as a session key.
func generateLifecycleSessionKey() []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		// Fall back to a deterministic placeholder if crypto/rand fails.
		for i := range key {
			key[i] = byte(i)
		}
	}
	return key
}
