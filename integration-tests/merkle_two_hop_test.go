package integration_tests

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/bits"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

type lightTx struct {
	From            string `json:"from"`
	To              string `json:"to"`
	Value           uint64 `json:"value"`
	Data            []byte `json:"data"`
	Status          string `json:"status"`
	Timestamp       int64  `json:"timestamp"`
	TransactionHash string `json:"transaction_hash"`
	PublicKey       string `json:"public_key"`
	Signature       []byte `json:"signature"`
	Fee             uint64 `json:"fee"`
	Type            string `json:"type"`
	Verified        bool   `json:"verified"`
}
type lightSibling struct {
	Hash   [32]byte `json:"hash"`
	IsLeft bool     `json:"is_left"`
}
type lightStep struct {
	Height    uint64   `json:"height"`
	TxRoot    [32]byte `json:"tx_root"`
	BlockHash []byte   `json:"block_hash"`
}
type lightTxProof struct {
	SchemaVersion  string         `json:"schema_version"`
	Transaction    *lightTx       `json:"transaction"`
	TxSiblings     []lightSibling `json:"tx_siblings"`
	BlockHeight    uint64         `json:"block_height"`
	PreAccumRoot   [32]byte       `json:"pre_accum_root"`
	BlockHash      []byte         `json:"block_hash"`
	BlockTxRoot    [32]byte       `json:"block_tx_root"`
	BlockAccumRoot [32]byte       `json:"block_accum_root"`
	Accumulator    []lightStep    `json:"accumulator"`
	TargetHeight   uint64         `json:"target_height"`
	AccumRoot      [32]byte       `json:"accum_root"`
}
type lightCheckpointRecord struct {
	Checkpoint struct {
		EndHeight uint64   `json:"end_height"`
		Root      [32]byte `json:"root"`
	} `json:"checkpoint"`
	MMRPosition uint64   `json:"mmr_position"`
	LeafHash    [32]byte `json:"leaf_hash"`
	Status      string   `json:"status"`
}
type lightMMRSibling struct {
	Hash [32]byte `json:"hash"`
	Side bool     `json:"side"`
}
type lightMMRProof struct {
	LeafIndex uint64            `json:"leaf_index"`
	TreeSize  uint64            `json:"tree_size"`
	Path      []lightMMRSibling `json:"path"`
}

// TestMerkleTwoHopAgainstTestnet is an environment-driven standalone light
// client. It imports no KNIRV package and independently verifies both hops.
// Set KNIRVCHAIN_URL, KNIRVORACLE_URL, KNIRV_TX_HASH, and KNIRV_CHECKPOINT_CHAIN_ID.
func TestMerkleTwoHopAgainstTestnet(t *testing.T) {
	chainURL, oracleURL := os.Getenv("KNIRVCHAIN_URL"), os.Getenv("KNIRVORACLE_URL")
	txHash, chainID := os.Getenv("KNIRV_TX_HASH"), os.Getenv("KNIRV_CHECKPOINT_CHAIN_ID")
	if chainURL == "" || oracleURL == "" || txHash == "" || chainID == "" {
		t.Skip("set KNIRVCHAIN_URL, KNIRVORACLE_URL, KNIRV_TX_HASH, and KNIRV_CHECKPOINT_CHAIN_ID")
	}
	client := &http.Client{Timeout: 20 * time.Second}
	var checkpointResponse struct {
		Checkpoints []lightCheckpointRecord `json:"checkpoints"`
	}
	getJSON(t, client, strings.TrimRight(oracleURL, "/")+"/oracle/v3/checkpoints/"+chainID, &checkpointResponse)
	sort.Slice(checkpointResponse.Checkpoints, func(i, j int) bool {
		return checkpointResponse.Checkpoints[i].Checkpoint.EndHeight > checkpointResponse.Checkpoints[j].Checkpoint.EndHeight
	})

	var txProof lightTxProof
	var checkpoint *lightCheckpointRecord
	for i := range checkpointResponse.Checkpoints {
		candidate := &checkpointResponse.Checkpoints[i]
		url := fmt.Sprintf("%s/proof/tx/%s?target_height=%d", strings.TrimRight(chainURL, "/"), txHash, candidate.Checkpoint.EndHeight)
		if fetchJSON(client, url, &txProof) == nil && txProof.AccumRoot == candidate.Checkpoint.Root {
			checkpoint = candidate
			break
		}
	}
	if checkpoint == nil {
		t.Fatal("no Oracle checkpoint covers the requested transaction")
	}
	if err := verifyLightTxProof(&txProof); err != nil {
		t.Fatalf("first hop: %v", err)
	}

	var mmrProof lightMMRProof
	getJSON(t, client, fmt.Sprintf("%s/oracle/v3/mmr/proof/%d", strings.TrimRight(oracleURL, "/"), checkpoint.MMRPosition), &mmrProof)
	var rootResponse struct {
		Root string `json:"root"`
		Size uint64 `json:"size"`
	}
	getJSON(t, client, strings.TrimRight(oracleURL, "/")+"/oracle/v3/mmr/root", &rootResponse)
	rootBytes, err := hex.DecodeString(strings.TrimPrefix(rootResponse.Root, "0x"))
	if err != nil || len(rootBytes) != 32 {
		t.Fatalf("invalid Oracle root")
	}
	var root [32]byte
	copy(root[:], rootBytes)
	if !verifyLightMMR(root, checkpoint.LeafHash, mmrProof, rootResponse.Size) {
		t.Fatal("second hop: invalid Oracle MMR inclusion proof")
	}
}

func getJSON(t *testing.T, client *http.Client, url string, out interface{}) {
	t.Helper()
	if err := fetchJSON(client, url, out); err != nil {
		t.Fatal(err)
	}
}
func fetchJSON(client *http.Client, url string, out interface{}) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s returned %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func verifyLightTxProof(p *lightTxProof) error {
	if p == nil || p.Transaction == nil || p.SchemaVersion != "knirv.tx-accum-proof.v1" {
		return fmt.Errorf("bad schema")
	}
	root := sha256.Sum256(append([]byte{0}, canonicalLightTx(p.Transaction)...))
	for _, sibling := range p.TxSiblings {
		if sibling.IsLeft {
			root = lightParent(sibling.Hash, root)
		} else {
			root = lightParent(root, sibling.Hash)
		}
	}
	if root != p.BlockTxRoot {
		return fmt.Errorf("transaction root mismatch")
	}
	accum := lightAccum(p.PreAccumRoot, p.BlockTxRoot, p.BlockHash)
	if accum != p.BlockAccumRoot {
		return fmt.Errorf("block accumulator mismatch")
	}
	height := p.BlockHeight
	for _, step := range p.Accumulator {
		if step.Height != height+1 {
			return fmt.Errorf("non-contiguous step")
		}
		accum = lightAccum(accum, step.TxRoot, step.BlockHash)
		height = step.Height
	}
	if height != p.TargetHeight || accum != p.AccumRoot {
		return fmt.Errorf("target root mismatch")
	}
	return nil
}
func lightParent(left, right [32]byte) [32]byte {
	data := append([]byte{1}, left[:]...)
	data = append(data, right[:]...)
	return sha256.Sum256(data)
}
func lightAccum(prev, txRoot [32]byte, blockHash []byte) [32]byte {
	data := append([]byte{1}, prev[:]...)
	data = append(data, txRoot[:]...)
	data = append(data, blockHash...)
	return sha256.Sum256(data)
}
func canonicalLightTx(tx *lightTx) []byte {
	var out []byte
	out = appendSized(out, []byte("knirv-tx-merkle-v1"))
	out = append(out, 1)
	out = appendSized(out, []byte(tx.From))
	out = appendSized(out, []byte(tx.To))
	out = appendU64(out, tx.Value)
	out = appendSized(out, tx.Data)
	out = appendSized(out, []byte(tx.Status))
	out = appendU64(out, uint64(tx.Timestamp))
	out = appendSized(out, []byte(tx.TransactionHash))
	out = appendSized(out, []byte(tx.PublicKey))
	out = appendSized(out, tx.Signature)
	out = appendU64(out, tx.Fee)
	out = appendSized(out, []byte(tx.Type))
	if tx.Verified {
		out = append(out, 1)
	} else {
		out = append(out, 0)
	}
	return out
}
func appendSized(out, value []byte) []byte {
	out = appendU64(out, uint64(len(value)))
	return append(out, value...)
}
func appendU64(out []byte, value uint64) []byte {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	return append(out, encoded[:]...)
}

func verifyLightMMR(root, leaf [32]byte, proof lightMMRProof, size uint64) bool {
	if proof.TreeSize != size || proof.LeafIndex >= size {
		return false
	}
	expectedSides, ok := lightMMRProofSides(size, proof.LeafIndex)
	if !ok || len(expectedSides) != len(proof.Path) {
		return false
	}
	value := leaf
	for i, sibling := range proof.Path {
		if sibling.Side != expectedSides[i] {
			return false
		}
		if sibling.Side {
			value = lightParent(value, sibling.Hash)
		} else {
			value = lightParent(sibling.Hash, value)
		}
	}
	return value == root
}

func lightMMRProofSides(treeSize, leafIndex uint64) ([]bool, bool) {
	if treeSize == 0 || leafIndex >= treeSize {
		return nil, false
	}
	peakIndex, peakCount := 0, bits.OnesCount64(treeSize)
	var peakHeight uint
	var relative uint64
	found := false
	for height := bits.Len64(treeSize); height > 0; height-- {
		h := uint(height - 1)
		if treeSize&(uint64(1)<<h) == 0 {
			continue
		}
		count := uint64(1) << h
		if leafIndex < count {
			peakHeight, relative, found = h, leafIndex, true
			break
		}
		leafIndex -= count
		peakIndex++
	}
	if !found {
		return nil, false
	}
	sides := make([]bool, 0, int(peakHeight)+peakCount)
	for level := uint(0); level < peakHeight; level++ {
		sides = append(sides, relative&(uint64(1)<<level) == 0)
	}
	if peakIndex > 0 {
		sides = append(sides, false)
	}
	for later := peakIndex + 1; later < peakCount; later++ {
		sides = append(sides, true)
	}
	return sides, true
}
