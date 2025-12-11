package p2p

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SkillConfirmationStatus represents the status of a skill confirmation
type SkillConfirmationStatus int

const (
	SkillConfirmationPending SkillConfirmationStatus = iota
	SkillConfirmationConfirmed
	SkillConfirmationRejected
	SkillConfirmationExpired
)

func (scs SkillConfirmationStatus) String() string {
	switch scs {
	case SkillConfirmationPending:
		return "Pending"
	case SkillConfirmationConfirmed:
		return "Confirmed"
	case SkillConfirmationRejected:
		return "Rejected"
	case SkillConfirmationExpired:
		return "Expired"
	default:
		return "Unknown"
	}
}

// SkillConfirmation represents a skill confirmation request/response
type SkillConfirmation struct {
	SkillID       string                  `json:"skill_id"`
	NodeID        string                  `json:"node_id"`
	Status        SkillConfirmationStatus `json:"status"`
	Confirmations map[PeerID]bool         `json:"confirmations"`
	CreatedAt     time.Time               `json:"created_at"`
	ExpiresAt     time.Time               `json:"expires_at"`
	Metadata      map[string]interface{}  `json:"metadata"`
}

// SkillConfirmationProtocol manages skill confirmation via GossipSub
type SkillConfirmationProtocol struct {
	gossip                *GossipManager
	confirmations         map[string]*SkillConfirmation
	topicName             string
	expiryDuration        time.Duration
	requiredConfirmations int
	logger                *zap.Logger
	mu                    sync.RWMutex
}

// NewSkillConfirmationProtocol creates a new skill confirmation protocol
func NewSkillConfirmationProtocol(gossip *GossipManager, logger *zap.Logger) *SkillConfirmationProtocol {
	scp := &SkillConfirmationProtocol{
		gossip:                gossip,
		confirmations:         make(map[string]*SkillConfirmation),
		topicName:             "skill-confirmations",
		expiryDuration:        5 * time.Minute,
		requiredConfirmations: 3, // Require 3 peer confirmations
		logger:                logger,
	}

	// Subscribe to skill confirmation topic
	if gossip != nil {
		gossip.Subscribe(scp.topicName, scp.handleConfirmationMessage)
	}

	return scp
}

// RequestConfirmation requests confirmation for a skill
func (scp *SkillConfirmationProtocol) RequestConfirmation(skillID, nodeID string, metadata map[string]interface{}) (*SkillConfirmation, error) {
	scp.mu.Lock()
	defer scp.mu.Unlock()

	// Create confirmation
	confirmation := &SkillConfirmation{
		SkillID:       skillID,
		NodeID:        nodeID,
		Status:        SkillConfirmationPending,
		Confirmations: make(map[PeerID]bool),
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(scp.expiryDuration),
		Metadata:      metadata,
	}

	// Store confirmation
	confirmKey := fmt.Sprintf("%s:%s", skillID, nodeID)
	scp.confirmations[confirmKey] = confirmation

	// Broadcast request
	if scp.gossip != nil {
		data, err := json.Marshal(map[string]interface{}{
			"type":      "request",
			"skill_id":  skillID,
			"node_id":   nodeID,
			"metadata":  metadata,
			"timestamp": time.Now().Unix(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		if err := scp.gossip.Publish(scp.topicName, data); err != nil {
			return nil, fmt.Errorf("failed to publish request: %w", err)
		}
	}

	scp.logger.Info("Skill confirmation requested",
		zap.String("skill_id", skillID),
		zap.String("node_id", nodeID),
	)

	return confirmation, nil
}

// ConfirmSkill confirms a skill from a peer
func (scp *SkillConfirmationProtocol) ConfirmSkill(skillID, nodeID string, peerID PeerID) error {
	scp.mu.Lock()
	defer scp.mu.Unlock()

	confirmKey := fmt.Sprintf("%s:%s", skillID, nodeID)
	confirmation, exists := scp.confirmations[confirmKey]
	if !exists {
		return fmt.Errorf("confirmation not found")
	}

	if confirmation.Status != SkillConfirmationPending {
		return fmt.Errorf("confirmation no longer pending")
	}

	// Check expiry
	if time.Now().After(confirmation.ExpiresAt) {
		confirmation.Status = SkillConfirmationExpired
		return fmt.Errorf("confirmation expired")
	}

	// Add confirmation
	confirmation.Confirmations[peerID] = true

	// Check if we have enough confirmations
	if len(confirmation.Confirmations) >= scp.requiredConfirmations {
		confirmation.Status = SkillConfirmationConfirmed
		scp.logger.Info("Skill confirmed",
			zap.String("skill_id", skillID),
			zap.String("node_id", nodeID),
			zap.Int("confirmations", len(confirmation.Confirmations)),
		)
	}

	// Broadcast confirmation
	if scp.gossip != nil {
		data, err := json.Marshal(map[string]interface{}{
			"type":      "confirm",
			"skill_id":  skillID,
			"node_id":   nodeID,
			"peer_id":   string(peerID),
			"timestamp": time.Now().Unix(),
		})
		if err != nil {
			return fmt.Errorf("failed to marshal confirmation: %w", err)
		}

		if err := scp.gossip.Publish(scp.topicName, data); err != nil {
			return fmt.Errorf("failed to publish confirmation: %w", err)
		}
	}

	return nil
}

// RejectSkill rejects a skill confirmation
func (scp *SkillConfirmationProtocol) RejectSkill(skillID, nodeID string, peerID PeerID, reason string) error {
	scp.mu.Lock()
	defer scp.mu.Unlock()

	confirmKey := fmt.Sprintf("%s:%s", skillID, nodeID)
	confirmation, exists := scp.confirmations[confirmKey]
	if !exists {
		return fmt.Errorf("confirmation not found")
	}

	confirmation.Status = SkillConfirmationRejected
	confirmation.Metadata["rejection_reason"] = reason
	confirmation.Metadata["rejected_by"] = string(peerID)

	scp.logger.Warn("Skill rejected",
		zap.String("skill_id", skillID),
		zap.String("node_id", nodeID),
		zap.String("reason", reason),
	)

	return nil
}

// GetConfirmation retrieves a confirmation
func (scp *SkillConfirmationProtocol) GetConfirmation(skillID, nodeID string) (*SkillConfirmation, error) {
	scp.mu.RLock()
	defer scp.mu.RUnlock()

	confirmKey := fmt.Sprintf("%s:%s", skillID, nodeID)
	confirmation, exists := scp.confirmations[confirmKey]
	if !exists {
		return nil, fmt.Errorf("confirmation not found")
	}

	return confirmation, nil
}

// ListConfirmations lists all confirmations
func (scp *SkillConfirmationProtocol) ListConfirmations() []*SkillConfirmation {
	scp.mu.RLock()
	defer scp.mu.RUnlock()

	confirmations := make([]*SkillConfirmation, 0, len(scp.confirmations))
	for _, confirmation := range scp.confirmations {
		confirmations = append(confirmations, confirmation)
	}

	return confirmations
}

// CleanupExpired removes expired confirmations
func (scp *SkillConfirmationProtocol) CleanupExpired() int {
	scp.mu.Lock()
	defer scp.mu.Unlock()

	now := time.Now()
	removed := 0

	for key, confirmation := range scp.confirmations {
		if confirmation.Status == SkillConfirmationPending && now.After(confirmation.ExpiresAt) {
			confirmation.Status = SkillConfirmationExpired
			delete(scp.confirmations, key)
			removed++
		}
	}

	if removed > 0 {
		scp.logger.Info("Cleaned up expired confirmations",
			zap.Int("removed", removed),
		)
	}

	return removed
}

// handleConfirmationMessage handles incoming confirmation messages
func (scp *SkillConfirmationProtocol) handleConfirmationMessage(data []byte) error {
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		return fmt.Errorf("invalid message type")
	}

	switch msgType {
	case "request":
		// Handle skill confirmation request
		skillID, _ := msg["skill_id"].(string)
		nodeID, _ := msg["node_id"].(string)
		scp.logger.Debug("Received skill confirmation request",
			zap.String("skill_id", skillID),
			zap.String("node_id", nodeID),
		)
		// In production, would validate and auto-confirm if criteria met

	case "confirm":
		// Handle peer confirmation
		skillID, _ := msg["skill_id"].(string)
		nodeID, _ := msg["node_id"].(string)
		peerIDStr, _ := msg["peer_id"].(string)
		scp.logger.Debug("Received skill confirmation",
			zap.String("skill_id", skillID),
			zap.String("node_id", nodeID),
			zap.String("peer_id", peerIDStr),
		)
		// Update local confirmation state

	default:
		return fmt.Errorf("unknown message type: %s", msgType)
	}

	return nil
}

// GetStats returns skill confirmation statistics
func (scp *SkillConfirmationProtocol) GetStats() map[string]interface{} {
	scp.mu.RLock()
	defer scp.mu.RUnlock()

	statusCounts := make(map[string]int)
	for _, confirmation := range scp.confirmations {
		statusCounts[confirmation.Status.String()]++
	}

	return map[string]interface{}{
		"total_confirmations":    len(scp.confirmations),
		"status_counts":          statusCounts,
		"required_confirmations": scp.requiredConfirmations,
		"expiry_duration":        scp.expiryDuration.String(),
	}
}
