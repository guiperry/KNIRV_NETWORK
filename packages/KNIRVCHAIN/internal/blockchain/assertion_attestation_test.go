package blockchain

import "testing"

func TestAssertionAttestationUsesTransactionMerkleLeaf(t *testing.T) {
	a := AssertionAttestation{SchemaVersion: 2, AssertionKey: "assertion-v2:test", CommitmentTarget: 7, ContextHash: 9, BestSeed: "aabb"}
	tx, err := NewAssertionAttestationTransaction("node", a, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := AssertionMerkleCommitment(tx)
	if err != nil {
		t.Fatal(err)
	}
	root := TxMerkleRoot([]*Transaction{tx})
	if leaf != root {
		t.Fatalf("single transaction root must be its assertion Merkle leaf")
	}
}

func TestAssertionAttestationRejectsMalformedSeed(t *testing.T) {
	if err := ValidateAssertionAttestation(AssertionAttestation{SchemaVersion: 2, AssertionKey: "k", BestSeed: "not hex"}); err == nil {
		t.Fatal("expected malformed seed rejection")
	}
}
