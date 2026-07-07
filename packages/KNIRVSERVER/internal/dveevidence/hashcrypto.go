package dveevidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

type chainedEvent struct {
	Index     int             `json:"index"`
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	PrevHash  string          `json:"prev_hash,omitempty"`
}

func hashEvent(ev Event) string {
	raw, _ := json.Marshal(chainedEvent{
		Index:     ev.Index,
		Timestamp: ev.Timestamp,
		Type:      ev.Type,
		Payload:   ev.Payload,
		PrevHash:  ev.PrevHash,
	})
	return sha256Hex(raw)
}

func BuildEventLog(events []Event) ([]Event, string, error) {
	out := make([]Event, 0, len(events))
	prev := ""
	for i, ev := range events {
		ev.Index = i
		if strings.TrimSpace(ev.Timestamp) == "" {
			return nil, "", fmt.Errorf("event %d missing timestamp", i)
		}
		ev.PrevHash = prev
		ev.Hash = hashEvent(ev)
		out = append(out, ev)
		prev = ev.Hash
	}
	return out, prev, nil
}

func VerifyEventLog(events []Event, expectedRoot string) error {
	if len(events) == 0 {
		if expectedRoot == "" {
			return nil
		}
		if expectedRoot == sha256Hex(nil) {
			return nil
		}
		return fmt.Errorf("event log root mismatch: got empty log, want %s", expectedRoot)
	}
	prev := ""
	for i, ev := range events {
		if ev.Index != i {
			return fmt.Errorf("event %d has unexpected index %d", i, ev.Index)
		}
		if strings.TrimSpace(ev.Timestamp) == "" {
			return fmt.Errorf("event %d missing timestamp", i)
		}
		if ev.PrevHash != "" && ev.PrevHash != prev {
			return fmt.Errorf("event %d has prev_hash %q, want %q", i, ev.PrevHash, prev)
		}
		computed := hashEvent(Event{
			Index:     i,
			Timestamp: ev.Timestamp,
			Type:      ev.Type,
			Payload:   ev.Payload,
			PrevHash:  prev,
		})
		if ev.Hash != "" && ev.Hash != computed {
			return fmt.Errorf("event %d hash mismatch", i)
		}
		prev = computed
	}
	if prev != expectedRoot {
		return fmt.Errorf("event log root mismatch: got %s, want %s", prev, expectedRoot)
	}
	return nil
}

func normalizeArtifactHash(hash string) string {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return hash
	}
	return hash
}

func hashPair(left, right string) string {
	return sha256Hex([]byte(left + "|" + right))
}
