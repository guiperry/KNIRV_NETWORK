package evidence

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/storage/pqc"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

type ChainClient interface {
	AnchorEvidencePack(req EvidenceAnchorRequest) (string, error)
	GetBlockHeight() (uint64, error)
}

type ChainTransaction struct {
	Type      string
	Data      []byte
	Signature []byte
	PublicKey string
}

type EvidenceAnchorRequest struct {
	EvidenceID   string                 `json:"evidence_id,omitempty"`
	EvidenceType string                 `json:"evidence_type,omitempty"`
	NodeID       string                 `json:"node_id,omitempty"`
	ValidationID string                 `json:"validation_id,omitempty"`
	Payload      map[string]interface{} `json:"payload,omitempty"`
	Signature    string                 `json:"signature,omitempty"`
	Algorithm    string                 `json:"algorithm,omitempty"`
	PublicKey    string                 `json:"public_key,omitempty"`
}

type EvidencePack struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	NodeID       string                 `json:"node_id"`
	ValidationID string                 `json:"validation_id"`
	Status       string                 `json:"status"`
	Evidence     map[string]interface{} `json:"evidence"`
	PQCSignature *PQCSignature          `json:"pqc_signature,omitempty"`
	BlockHeight  uint64                 `json:"block_height,omitempty"`
	ChainTxHash  string                 `json:"chain_tx_hash,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	AnchoredAt   *time.Time             `json:"anchored_at,omitempty"`
}

type PQCSignature struct {
	Algorithm   string    `json:"algorithm"`
	PublicKeyID string    `json:"public_key_id"`
	Signature   string    `json:"signature"`
	SignedAt    time.Time `json:"signed_at"`
}

type AnchoringService struct {
	db            *database.BuntDBManager
	pqcManager    *pqc.EncryptionManager
	chainClient   ChainClient
	mu            sync.RWMutex
	evidencePacks map[string]*EvidencePack
	keyID         string
}

func NewAnchoringService(db *database.BuntDBManager, pqcManager *pqc.EncryptionManager, keyID string) *AnchoringService {
	return &AnchoringService{
		db:            db,
		pqcManager:    pqcManager,
		evidencePacks: make(map[string]*EvidencePack),
		keyID:         keyID,
	}
}

func (s *AnchoringService) SetValidationChainClient(client ChainClient) {
	s.chainClient = client
}

func (s *AnchoringService) SetChainClient(client ChainClient) {
	s.SetValidationChainClient(client)
}

func (s *AnchoringService) CreateEvidencePack(packType, nodeID, validationID string, evidence map[string]interface{}) (*EvidencePack, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pack := &EvidencePack{
		ID:           fmt.Sprintf("ep_%s", uuid.New().String()[:12]),
		Type:         packType,
		NodeID:       nodeID,
		ValidationID: validationID,
		Status:       "created",
		Evidence:     evidence,
		CreatedAt:    time.Now(),
	}

	s.evidencePacks[pack.ID] = pack

	if err := s.saveEvidencePack(pack); err != nil {
		log.Printf("Warning: Failed to save evidence pack: %v", err)
	}

	log.Printf("Created evidence pack %s (type: %s, node: %s)", pack.ID, packType, nodeID)
	return pack, nil
}

func (s *AnchoringService) SignEvidencePack(packID string) (*EvidencePack, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pack, ok := s.evidencePacks[packID]
	if !ok {
		return nil, fmt.Errorf("evidence pack not found: %s", packID)
	}

	evidenceJSON, err := json.Marshal(pack.Evidence)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal evidence: %w", err)
	}

	signature, err := s.signEvidence(evidenceJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to sign evidence: %w", err)
	}

	pack.PQCSignature = signature
	pack.Status = "signed"

	if err := s.saveEvidencePack(pack); err != nil {
		return nil, fmt.Errorf("failed to save signed pack: %w", err)
	}

	log.Printf("Evidence pack %s signed with %s", pack.ID, signature.Algorithm)
	return pack, nil
}

func (s *AnchoringService) AnchorToChain(packID string) (*EvidencePack, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pack, ok := s.evidencePacks[packID]
	if !ok {
		return nil, fmt.Errorf("evidence pack not found: %s", packID)
	}

	if pack.PQCSignature == nil {
		if _, err := s.SignEvidencePack(packID); err != nil {
			return nil, fmt.Errorf("failed to sign pack before anchoring: %w", err)
		}
		pack = s.evidencePacks[packID]
	}

	if s.chainClient == nil {
		return nil, fmt.Errorf("chain client not configured")
	}

	anchorData := map[string]interface{}{
		"evidence_id":   pack.ID,
		"type":          pack.Type,
		"node_id":       pack.NodeID,
		"validation_id": pack.ValidationID,
		"signature":     pack.PQCSignature.Signature,
		"algorithm":     pack.PQCSignature.Algorithm,
		"timestamp":     time.Now().Unix(),
	}

	anchorJSON, err := json.Marshal(anchorData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal anchor data: %w", err)
	}

	txHash, err := s.chainClient.AnchorEvidencePack(EvidenceAnchorRequest{
		EvidenceID:   pack.ID,
		EvidenceType: pack.Type,
		NodeID:       pack.NodeID,
		ValidationID: pack.ValidationID,
		Payload: map[string]interface{}{
			"anchor": string(anchorJSON),
		},
		Signature: pack.PQCSignature.Signature,
		Algorithm: pack.PQCSignature.Algorithm,
		PublicKey: pack.PQCSignature.PublicKeyID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to anchor to chain: %w", err)
	}

	blockHeight, _ := s.chainClient.GetBlockHeight()

	now := time.Now()
	pack.ChainTxHash = txHash
	pack.BlockHeight = blockHeight
	pack.Status = "anchored"
	pack.AnchoredAt = &now
	pack.CompletedAt = &now

	if err := s.saveEvidencePack(pack); err != nil {
		return nil, fmt.Errorf("failed to save anchored pack: %w", err)
	}

	log.Printf("Evidence pack %s anchored to KNIRVCHAIN (tx: %s, block: %d)", pack.ID, txHash[:16], blockHeight)
	return pack, nil
}

func (s *AnchoringService) GetEvidencePack(packID string) (*EvidencePack, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pack, ok := s.evidencePacks[packID]
	return pack, ok
}

func (s *AnchoringService) ListEvidencePacks(nodeID string, limit int) []*EvidencePack {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var packs []*EvidencePack
	for _, pack := range s.evidencePacks {
		if nodeID != "" && pack.NodeID != nodeID {
			continue
		}
		packs = append(packs, pack)
	}

	if limit > 0 && len(packs) > limit {
		return packs[len(packs)-limit:]
	}
	return packs
}

func (s *AnchoringService) VerifyEvidencePack(packID string) (bool, error) {
	s.mu.RLock()
	pack, ok := s.evidencePacks[packID]
	s.mu.RUnlock()

	if !ok {
		return false, fmt.Errorf("evidence pack not found: %s", packID)
	}

	if pack.PQCSignature == nil {
		return false, fmt.Errorf("evidence pack not signed")
	}

	evidenceJSON, err := json.Marshal(pack.Evidence)
	if err != nil {
		return false, fmt.Errorf("failed to marshal evidence: %w", err)
	}

	valid, err := s.verifySignature(evidenceJSON, pack.PQCSignature)
	if err != nil {
		return false, err
	}

	if !valid {
		return false, nil
	}

	if pack.ChainTxHash != "" {
		return true, nil
	}

	return true, nil
}

func (s *AnchoringService) GetAnchoredEvidenceCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, pack := range s.evidencePacks {
		if pack.Status == "anchored" {
			count++
		}
	}
	return count
}

func (s *AnchoringService) GetStatistics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.evidencePacks)
	signed := 0
	anchored := 0
	pending := 0

	for _, pack := range s.evidencePacks {
		switch pack.Status {
		case "created":
			pending++
		case "signed":
			signed++
		case "anchored":
			anchored++
		}
	}

	return map[string]interface{}{
		"total_packs":     total,
		"signed_packs":    signed,
		"anchored_packs":  anchored,
		"pending_packs":   pending,
		"chain_connected": s.chainClient != nil,
	}
}

func (s *AnchoringService) signEvidence(data []byte) (*PQCSignature, error) {
	masterKey := s.pqcManager.GetMasterKey()
	if masterKey == nil {
		return nil, fmt.Errorf("master key not available")
	}

	signature, err := masterKey.Sign(data)
	if err != nil {
		return nil, fmt.Errorf("failed to sign with Dilithium: %w", err)
	}

	return &PQCSignature{
		Algorithm:   "Dilithium-3",
		PublicKeyID: masterKey.ID,
		Signature:   fmt.Sprintf("%x", signature),
		SignedAt:    time.Now(),
	}, nil
}

func (s *AnchoringService) verifySignature(data []byte, sig *PQCSignature) (bool, error) {
	keyPair, err := s.pqcManager.GetKey(sig.PublicKeyID)
	if err != nil {
		return false, fmt.Errorf("failed to get signing key: %w", err)
	}

	var signatureBytes []byte
	_, err = fmt.Sscanf(sig.Signature, "%x", &signatureBytes)
	if err != nil {
		signatureBytes = []byte(sig.Signature)
	}

	return keyPair.Verify(data, signatureBytes), nil
}

func (s *AnchoringService) saveEvidencePack(pack *EvidencePack) error {
	if s.db == nil {
		return nil
	}

	data, err := json.Marshal(pack)
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *buntdb.Tx) error {
		_, _, err = tx.Set(fmt.Sprintf("evidence:pack:%s", pack.ID), string(data), nil)
		return err
	})
}

func (s *AnchoringService) LoadEvidencePacks() error {
	if s.db == nil {
		return nil
	}

	return s.db.GetObjectsByPrefix("evidence:pack:", func(key string, value []byte) bool {
		var pack EvidencePack
		if err := json.Unmarshal(value, &pack); err != nil {
			return true
		}
		s.mu.Lock()
		s.evidencePacks[pack.ID] = &pack
		s.mu.Unlock()
		return true
	})
}
