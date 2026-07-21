package checkpoint

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"KNIRVCHAIN/internal/blockchain"
	"KNIRVCHAIN/internal/database"
	"github.com/ethereum/go-ethereum/crypto"
)

type runtimeFixture struct {
	chainID string
	blocks  []*blockchain.Block
	authors map[string]bool
}

func (f *runtimeFixture) GetChainID() string { return f.chainID }
func (f *runtimeFixture) TipHeight() uint64  { return f.blocks[len(f.blocks)-1].BlockNumber }
func (f *runtimeFixture) AccumRootAt(height uint64) ([32]byte, error) {
	for _, b := range f.blocks {
		if b.BlockNumber == height {
			return b.Header.AccumRoot, nil
		}
	}
	return [32]byte{}, context.Canceled
}
func (f *runtimeFixture) NetworkAuthors() map[string]bool { return f.authors }
func (f *runtimeFixture) BlocksRange(start, end uint64) ([]*blockchain.Block, error) {
	out := []*blockchain.Block{}
	for _, b := range f.blocks {
		if b.BlockNumber >= start && b.BlockNumber <= end {
			out = append(out, b.DeepCopy())
		}
	}
	return out, nil
}

func TestRuntimePostsCheckpointThenUsesLocalVerifier(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	address := OracleAddress(&key.PublicKey)
	blocks := []*blockchain.Block{{BlockNumber: 1, Timestamp: 1, ProposerAddress: address}, {BlockNumber: 2, Timestamp: 2, ProposerAddress: address}}
	var accum [32]byte
	for _, block := range blocks {
		block.BlockHash = block.Hash()
		blockchain.PopulateBlockMerkleRoots(block, accum)
		accum = block.Header.AccumRoot
	}
	source := &runtimeFixture{chainID: "runtime-chain", blocks: blocks, authors: map[string]bool{address: true}}

	var checkpointPosts atomic.Int32
	oracleTransport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body []byte
		switch r.URL.Path {
		case "/oracle/v3/registry/register":
			body, _ = json.Marshal(map[string]interface{}{"registered": true})
		case "/oracle/v3/checkpoints":
			checkpointPosts.Add(1)
			body, _ = json.Marshal(map[string]interface{}{"mmr_position": 0, "leaf_hash": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"})
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewBufferString(`{"error":"not found"}`))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body))}, nil
	})
	var verifierPosts atomic.Int32
	verifierTransport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		verifierPosts.Add(1)
		body, _ := json.Marshal(map[string]interface{}{"approved": true})
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body))}, nil
	})

	db, err := database.NewLevelDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	builder := NewBuilder(source, db, 1, 1, address)
	poster := NewPoster(db, "http://oracle", "")
	poster.oracleUnixClient = nil
	poster.httpClient = &http.Client{Transport: oracleTransport}
	poster.verifierClient = &http.Client{Transport: verifierTransport}
	poster.verifierSocket = "/internal-test/verifier.sock"
	runtime, err := NewRuntime(source, builder, poster, map[string]*ecdsa.PrivateKey{address: key})
	if err != nil {
		t.Fatal(err)
	}
	runtime.OnBlockCommitted(2)
	deadline := time.Now().Add(3 * time.Second)
	for verifierPosts.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if checkpointPosts.Load() != 1 {
		t.Fatalf("checkpoint posts = %d", checkpointPosts.Load())
	}
	if verifierPosts.Load() != 1 {
		t.Fatalf("local verifier posts = %d", verifierPosts.Load())
	}
	if status := runtime.Status(); status.LastSubmitStatus != SubmitSubmitted || status.LastEndHeight != 1 {
		t.Fatalf("unexpected status: %+v", status)
	}
}
