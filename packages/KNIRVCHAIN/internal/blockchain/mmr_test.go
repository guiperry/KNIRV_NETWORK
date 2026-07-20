package blockchain

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

type mmrGoldenFile struct {
	Version   uint64          `json:"version"`
	Algorithm string          `json:"algorithm"`
	Bagging   string          `json:"bagging"`
	EmptyRoot string          `json:"empty_root"`
	TreeSize  uint64          `json:"tree_size"`
	Root      string          `json:"root"`
	Leaves    []mmrGoldenLeaf `json:"leaves"`
}

type mmrGoldenLeaf struct {
	Index    uint64             `json:"index"`
	DataUTF8 string             `json:"data_utf8"`
	LeafHash string             `json:"leaf_hash"`
	Path     []mmrGoldenSibling `json:"path"`
}

type mmrGoldenSibling struct {
	Hash string `json:"hash"`
	Side string `json:"side"`
}

func TestMMRSharedGoldenVectors(t *testing.T) {
	raw, err := os.ReadFile("../../../KNIRVORACLE/internal/oracle/mmr/testdata/mmr_golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors mmrGoldenFile
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	if vectors.Version != 1 || vectors.Algorithm != "sha256" || vectors.Bagging != "left-fold" {
		t.Fatal("unexpected golden-vector convention")
	}
	if got := MMREmptyRoot(); got != parseMMRGoldenHash(t, vectors.EmptyRoot) {
		t.Fatalf("empty root = %s, want %s", got.Hex(), vectors.EmptyRoot)
	}
	root := parseMMRGoldenHash(t, vectors.Root)
	for _, leaf := range vectors.Leaves {
		leafHash := parseMMRGoldenHash(t, leaf.LeafHash)
		if got := MMRLeafHash([]byte(leaf.DataUTF8)); got != leafHash {
			t.Fatalf("leaf %d hash = %s, want %s", leaf.Index, got.Hex(), leaf.LeafHash)
		}
		proof := MMRProof{LeafIndex: leaf.Index, TreeSize: vectors.TreeSize}
		for _, item := range leaf.Path {
			side := MMRSideLeft
			switch item.Side {
			case "left":
			case "right":
				side = MMRSideRight
			default:
				t.Fatalf("leaf %d: unknown side %q", leaf.Index, item.Side)
			}
			proof.Path = append(proof.Path, MMRSibling{Hash: parseMMRGoldenHash(t, item.Hash), Side: side})
		}
		if !VerifyMMRProof(root, leafHash, proof, vectors.TreeSize) {
			t.Fatalf("golden proof failed for leaf %d", leaf.Index)
		}
	}
}

func TestMMRProofRejectsRelabelledIndex(t *testing.T) {
	leaf := MMRLeafHash([]byte("leaf"))
	proof := MMRProof{LeafIndex: 0, TreeSize: 1}
	if !VerifyMMRProof(leaf, leaf, proof, 1) {
		t.Fatal("single-leaf proof did not verify")
	}
	proof.LeafIndex = 1
	if VerifyMMRProof(leaf, leaf, proof, 1) {
		t.Fatal("out-of-range leaf index verified")
	}
}

func parseMMRGoldenHash(t *testing.T, value string) MMRHash {
	t.Helper()
	raw, err := hex.DecodeString(value)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := MMRHashFromBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	return hash
}
