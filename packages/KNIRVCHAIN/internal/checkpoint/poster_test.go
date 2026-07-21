package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

func TestPosterSubmitsCheckpoint(t *testing.T) {
	var got Checkpoint
	p := NewPoster(nil, "http://oracle", "")
	p.oracleUnixClient = nil
	p.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/oracle/v3/checkpoints" {
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewBufferString(`{"error":"not found"}`))}, nil
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			return nil, err
		}
		body, _ := json.Marshal(map[string]interface{}{
			"status":       "admitted",
			"mmr_position": 0,
			"leaf_hash":    "abc",
		})
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body))}, nil
	})}
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
