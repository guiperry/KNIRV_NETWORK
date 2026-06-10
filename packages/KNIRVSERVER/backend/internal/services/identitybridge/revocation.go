package identitybridge

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type RevocationEntry struct {
	IdentityID string    `json:"identity_id"`
	NodeID     string    `json:"node_id"`
	Reason     string    `json:"reason"`
	RevokedAt  time.Time `json:"revoked_at"`
	RevokedBy  string    `json:"revoked_by"`
	ChainHash  string    `json:"chain_hash"`
}

type RevocationList struct {
	mu      sync.RWMutex
	entries []*RevocationEntry
	index   map[string]int
	tip     string
}

func NewRevocationList() *RevocationList {
	return &RevocationList{
		entries: make([]*RevocationEntry, 0),
		index:   make(map[string]int),
	}
}

func (rl *RevocationList) Revoke(identityID, nodeID, reason, revokedBy string) *RevocationEntry {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry := &RevocationEntry{
		IdentityID: identityID,
		NodeID:     nodeID,
		Reason:     reason,
		RevokedAt:  time.Now().UTC(),
		RevokedBy:  revokedBy,
	}

	chainInput := rl.tip + entry.IdentityID + entry.NodeID + entry.Reason + entry.RevokedBy + entry.RevokedAt.String()
	hash := sha256.Sum256([]byte(chainInput))
	entry.ChainHash = hex.EncodeToString(hash[:])
	rl.tip = entry.ChainHash

	rl.index[identityID] = len(rl.entries)
	rl.entries = append(rl.entries, entry)

	return entry
}

func (rl *RevocationList) IsRevoked(identityID string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	_, ok := rl.index[identityID]
	return ok
}

func (rl *RevocationList) VerifyChain() error {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if len(rl.entries) == 0 {
		return nil
	}

	var prevHash string
	for i, entry := range rl.entries {
		chainInput := prevHash + entry.IdentityID + entry.NodeID + entry.Reason + entry.RevokedBy + entry.RevokedAt.String()
		hash := sha256.Sum256([]byte(chainInput))
		computedHash := hex.EncodeToString(hash[:])

		if computedHash != entry.ChainHash {
			return fmt.Errorf("chain integrity violated at entry %d: expected hash %s, got %s", i, entry.ChainHash, computedHash)
		}
		prevHash = entry.ChainHash
	}
	return nil
}

func (rl *RevocationList) Export() []*RevocationEntry {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	result := make([]*RevocationEntry, len(rl.entries))
	copy(result, rl.entries)
	return result
}

func (rl *RevocationList) Len() int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return len(rl.entries)
}

func (rl *RevocationList) TipHash() string {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.tip
}
