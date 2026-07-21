package checkpoint

import (
	"context"
	"fmt"
	"testing"

	"KNIRVCHAIN/internal/blockchain"
)

type failingProver struct{}

func (failingProver) Prove([]*blockchain.Block) (*TransitionProof, error) {
	return nil, fmt.Errorf("offline")
}

func TestAsyncFallbackProver(t *testing.T) {
	batch := []*blockchain.Block{proofBlock(1, "author", 1)}
	fallback := &FallbackProver{Primary: failingProver{}, Fallback: &HashchainProver{ChainID: "chain"}}
	async, err := NewAsyncProver(fallback, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer async.Close()
	result, err := async.Submit(context.Background(), batch)
	if err != nil {
		t.Fatal(err)
	}
	got := <-result
	if got.Err != nil {
		t.Fatal(got.Err)
	}
	if got.Proof == nil || got.Proof.ProofSystem != "hashchain-v0" {
		t.Fatal("expected degraded hashchain-v0 proof")
	}
}
