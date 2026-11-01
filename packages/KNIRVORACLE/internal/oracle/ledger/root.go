package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// StateRoot hashes the whole state. encoding/json sorts map keys, so the bytes
// are canonical: the same state always produces the same root, in any process,
// regardless of map iteration order.
func StateRoot(s *State) (string, error) {
	if s == nil {
		return "", fmt.Errorf("ledger: nil state")
	}
	body, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("ledger: encode state: %w", err)
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}
