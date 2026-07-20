package mmr

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

type goldenVectorFile struct {
	Version    uint64             `json:"version"`
	Algorithm  string             `json:"algorithm"`
	LeafHash   string             `json:"leaf_hash"`
	ParentHash string             `json:"parent_hash"`
	Bagging    string             `json:"bagging"`
	EmptyRoot  string             `json:"empty_root"`
	TreeSize   uint64             `json:"tree_size"`
	Root       string             `json:"root"`
	Leaves     []goldenVectorLeaf `json:"leaves"`
}

type goldenVectorLeaf struct {
	Index    uint64                `json:"index"`
	DataUTF8 string                `json:"data_utf8"`
	LeafHash string                `json:"leaf_hash"`
	Path     []goldenVectorSibling `json:"path"`
}

type goldenVectorSibling struct {
	Hash string `json:"hash"`
	Side string `json:"side"`
}

func TestSharedGoldenVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/mmr_golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors goldenVectorFile
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	if vectors.Version != 1 || vectors.Algorithm != "sha256" || vectors.Bagging != "left-fold" {
		t.Fatal("unexpected golden-vector convention")
	}
	if got := EmptyRoot(); got != goldenHash(t, vectors.EmptyRoot) {
		t.Fatalf("empty root = %s, want %s", got.Hex(), vectors.EmptyRoot)
	}
	root := goldenHash(t, vectors.Root)
	if uint64(len(vectors.Leaves)) != vectors.TreeSize {
		t.Fatalf("leaf count = %d, tree size = %d", len(vectors.Leaves), vectors.TreeSize)
	}
	for _, leaf := range vectors.Leaves {
		if got := LeafHash([]byte(leaf.DataUTF8)); got != goldenHash(t, leaf.LeafHash) {
			t.Fatalf("leaf %d hash = %s, want %s", leaf.Index, got.Hex(), leaf.LeafHash)
		}
		proof := Proof{LeafIndex: leaf.Index, TreeSize: vectors.TreeSize}
		for _, item := range leaf.Path {
			side := SideLeft
			switch item.Side {
			case "left":
			case "right":
				side = SideRight
			default:
				t.Fatalf("leaf %d: unknown side %q", leaf.Index, item.Side)
			}
			proof.Path = append(proof.Path, Sibling{Hash: goldenHash(t, item.Hash), Side: side})
		}
		if !VerifyProof(root, goldenHash(t, leaf.LeafHash), proof, vectors.TreeSize) {
			t.Fatalf("golden proof failed for leaf %d", leaf.Index)
		}
	}
}

func goldenHash(t *testing.T, value string) Hash {
	t.Helper()
	raw, err := hex.DecodeString(value)
	if err != nil {
		t.Fatal(err)
	}
	h, err := HashFromBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	return h
}
