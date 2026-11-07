package main

import (
	"testing"

	"go.uber.org/zap"
)

// replicationFileSigner must fall back to the legacy/key-file path (returning
// nil so the already-unlocked OwnerPrivateKey is used) when neither OpenBao nor
// the key-file provider is configured. This guards the 2a.5 default: no Bao
// socket in the environment means no Bao-backed signer.
func TestReplicationFileSignerFallsBackWithoutBaoOrKeyFile(t *testing.T) {
	t.Setenv("VALIDATION_CHAIN_BAO_SOCKET", "")
	t.Setenv("VALIDATION_CHAIN_BAO_TOKEN", "")
	t.Setenv("ORACLE_KEY_PASSWORD", "")

	content := &rootKeyContent{RootPrivateKeyHex: "0x00"}
	if signer := replicationFileSigner(content, zap.NewNop()); signer != nil {
		t.Fatalf("expected nil signer when neither OpenBao nor key-file provider is configured, got %v", signer)
	}
}
