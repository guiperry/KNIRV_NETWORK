package checkpoint

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestPosterSubmitsCheckpoint(t *testing.T) {
	var got Checkpoint
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oracle/v3/checkpoints" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "admitted",
			"mmr_position": 0,
			"leaf_hash":    "abc",
		})
	}))
	defer srv.Close()

	p := NewPoster(nil, srv.URL, "")
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	cp := &Checkpoint{
		SchemaVersion: SchemaVersion,
		ChainID:       "knirvchain-1",
		StartHeight:   1,
		EndHeight:     64,
		Proposer:      OracleAddress(&key.PublicKey),
	}
	if err := SignCheckpoint(cp, key); err != nil {
		t.Fatalf("sign: %v", err)
	}
	out, err := p.PostCheckpoint(context.Background(), cp)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if out["status"] != "admitted" {
		t.Fatalf("unexpected response: %v", out)
	}
	// The server must have received the exact checkpoint we posted.
	if got.ChainID != cp.ChainID || got.EndHeight != cp.EndHeight {
		t.Fatalf("oracle received mismatched checkpoint: %+v", got)
	}
	if got.Signatures == nil {
		t.Fatal("posted checkpoint should carry signatures")
	}
}
