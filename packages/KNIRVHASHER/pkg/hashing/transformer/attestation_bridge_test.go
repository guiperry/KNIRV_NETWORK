package transformer

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestAttestationBridge_GroundsExactSpanAndQueuesMiss(t *testing.T) {
	dir := t.TempDir()
	embedding := []float32{-1, -0.5, 0.25, 1}
	context := []int32{11, 22, 33}
	span := []int32{44, 55}
	empty := NewEmptyAttestationBridge(dir, []int{0, 1, 2, 3}, 2)
	bucket, err := empty.Bucket(embedding)
	if err != nil {
		t.Fatalf("Bucket: %v", err)
	}
	seed := make([]byte, 32)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	entry := fmt.Sprintf(`{"schema_version":2,"source_file":"test","assertion_key":"%s","context_tokens":[11,22,33],"assertion_span":[44,55],"commitment_target":7,"context_hash":9,"asic_slots":[%d,%d,%d,%d,0,0,0,0,0,0,0,0],"best_seed":"%s","seed_bytes":32}`,
		assertionIdentity(context, span), bucket[0], bucket[1], bucket[2], bucket[3], base64.StdEncoding.EncodeToString(seed))
	if err := os.WriteFile(filepath.Join(dir, seedWritesFile), []byte(entry+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	bridge, err := NewAttestationBridge(dir, []int{0, 1, 2, 3}, 2)
	if err != nil {
		t.Fatalf("NewAttestationBridge: %v", err)
	}
	hit, err := bridge.Ground(AttestationCandidate{ContextTokens: context, AssertionSpan: span, ContextEmbedding: embedding})
	if err != nil {
		t.Fatalf("Ground hit: %v", err)
	}
	if !hit.Hit || hit.LowConfidence || hit.Proof == nil || string(hit.Proof.BestSeed) != string(seed) {
		t.Fatalf("expected proof-backed hit, got %#v", hit)
	}

	missCandidate := AttestationCandidate{ContextTokens: context, AssertionSpan: []int32{99}, ContextEmbedding: embedding}
	miss, err := bridge.Ground(missCandidate)
	if err != nil {
		t.Fatalf("Ground miss: %v", err)
	}
	if miss.Hit || !miss.LowConfidence || !miss.Queued {
		t.Fatalf("expected queued low-confidence miss, got %#v", miss)
	}
	queued := <-bridge.Pending()
	if got := assertionIdentity(queued.ContextTokens, queued.AssertionSpan); got != assertionIdentity(missCandidate.ContextTokens, missCandidate.AssertionSpan) {
		t.Fatalf("queued assertion = %s", got)
	}
}

func TestAttestationBridge_RejectsInvalidLedgerIdentity(t *testing.T) {
	dir := t.TempDir()
	entry := `{"schema_version":2,"assertion_key":"assertion-v2:wrong","context_tokens":[1],"assertion_span":[2],"asic_slots":[0,0,0,0,0,0,0,0,0,0,0,0],"best_seed":"AQ=="}`
	if err := os.WriteFile(filepath.Join(dir, seedWritesFile), []byte(entry+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	bridge, err := NewAttestationBridge(dir, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	result, err := bridge.Ground(AttestationCandidate{ContextTokens: []int32{1}, AssertionSpan: []int32{2}, ContextEmbedding: []float32{0, 0, 0, 0}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Hit {
		t.Fatal("invalid ledger identity must not produce a hit")
	}
}
