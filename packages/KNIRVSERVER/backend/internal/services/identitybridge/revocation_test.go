package identitybridge

import (
	"testing"
)

func TestNewRevocationList(t *testing.T) {
	rl := NewRevocationList()
	if rl == nil {
		t.Fatal("expected non-nil RevocationList")
	}
	if rl.Len() != 0 {
		t.Errorf("expected empty list, got %d entries", rl.Len())
	}
	if rl.TipHash() != "" {
		t.Errorf("expected empty tip, got %s", rl.TipHash())
	}
}

func TestRevokeSingleEntry(t *testing.T) {
	rl := NewRevocationList()
	entry := rl.Revoke("identity-1", "node-1", "compromised", "admin")

	if entry.IdentityID != "identity-1" {
		t.Errorf("expected identity-1, got %s", entry.IdentityID)
	}
	if entry.NodeID != "node-1" {
		t.Errorf("expected node-1, got %s", entry.NodeID)
	}
	if entry.Reason != "compromised" {
		t.Errorf("expected compromised, got %s", entry.Reason)
	}
	if entry.RevokedBy != "admin" {
		t.Errorf("expected admin, got %s", entry.RevokedBy)
	}
	if entry.ChainHash == "" {
		t.Fatal("expected non-empty chain hash")
	}

	if !rl.IsRevoked("identity-1") {
		t.Error("expected identity-1 to be revoked")
	}
	if rl.Len() != 1 {
		t.Errorf("expected 1 entry, got %d", rl.Len())
	}
}

func TestRevokeMultipleEntries(t *testing.T) {
	rl := NewRevocationList()
	e1 := rl.Revoke("id1", "node1", "reason1", "admin")
	e2 := rl.Revoke("id2", "node2", "reason2", "admin")

	if e1.ChainHash == "" || e2.ChainHash == "" {
		t.Fatal("expected non-empty chain hashes")
	}
	if e1.ChainHash == e2.ChainHash {
		t.Error("chain hashes should differ")
	}

	if rl.TipHash() != e2.ChainHash {
		t.Errorf("expected tip %s, got %s", e2.ChainHash, rl.TipHash())
	}
}

func TestIsRevokedNotFound(t *testing.T) {
	rl := NewRevocationList()
	if rl.IsRevoked("nonexistent") {
		t.Error("expected false for nonexistent identity")
	}
}

func TestVerifyChainEmpty(t *testing.T) {
	rl := NewRevocationList()
	if err := rl.VerifyChain(); err != nil {
		t.Errorf("expected nil error for empty chain, got %v", err)
	}
}

func TestVerifyChainValid(t *testing.T) {
	rl := NewRevocationList()
	rl.Revoke("id1", "node1", "reason1", "admin")
	rl.Revoke("id2", "node2", "reason2", "admin")
	rl.Revoke("id3", "node3", "reason3", "admin")

	if err := rl.VerifyChain(); err != nil {
		t.Errorf("expected valid chain, got error: %v", err)
	}
}

func TestVerifyChainTampered(t *testing.T) {
	rl := NewRevocationList()
	rl.Revoke("id1", "node1", "reason1", "admin")
	rl.Revoke("id2", "node2", "reason2", "admin")

	entries := rl.Export()
	entries[0].Reason = "tampered"

	if err := rl.VerifyChain(); err == nil {
		t.Error("expected chain verification to fail after tampering")
	}
}

func TestExport(t *testing.T) {
	rl := NewRevocationList()
	rl.Revoke("id1", "node1", "reason1", "admin")
	rl.Revoke("id2", "node2", "reason2", "admin")

	entries := rl.Export()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].IdentityID != "id1" || entries[1].IdentityID != "id2" {
		t.Error("exported entries out of order or incorrect")
	}
}

func TestRevokeAfterTamper(t *testing.T) {
	rl := NewRevocationList()
	rl.Revoke("id1", "node1", "reason1", "admin")

	entries := rl.Export()
	entries[0].Reason = "tampered"

	rl.Revoke("id2", "node2", "reason2", "admin")

	if err := rl.VerifyChain(); err == nil {
		t.Error("expected chain verification to fail after tampering detected")
	}
}
