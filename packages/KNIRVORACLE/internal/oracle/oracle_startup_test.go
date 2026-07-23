package oracle

import (
	"os"
	"path/filepath"
	"testing"

	oraclecrypto "github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

func newStartupTestConfig(t *testing.T, dataDir string) *OracleConfig {
	t.Helper()
	ownerKey, err := oraclecrypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate owner key: %v", err)
	}
	cfg := DefaultOracleConfig()
	cfg.OwnerPrivateKey = ownerKey.PrivateKeyHex()
	cfg.DataDir = dataDir
	cfg.DBBackend = "memory"
	return cfg
}

// TestNewOracleIgnoresStaleIndependentAuditFile is a regression test for the
// "checkpoint MMR and persisted AppHash MMR diverged; refusing unsafe
// recovery" startup failure. That check compared checkpoint_store.go's
// mmr_leaf_log.json (the real, single source of truth) against a second,
// independently-persisted audit_mmr_leaf_log.json that ABCIApplication used
// to keep. The two were written by different call sites and could fall out
// of sync on an unclean shutdown — trivially reproducible by killing the
// process between the two writes, which is exactly what repeated forced
// restarts during testing do. With ABCIApplication no longer keeping that
// second file at all, NewOracle must start cleanly regardless of what
// leftover cruft (stale, mismatched, or just present from an old binary)
// still sits in DataDir under that filename.
func TestNewOracleIgnoresStaleIndependentAuditFile(t *testing.T) {
	dir := t.TempDir()

	// First lifecycle: submit a real checkpoint so mmr_leaf_log.json holds
	// genuine, non-empty state.
	o1, err := NewOracle(newStartupTestConfig(t, dir), zap.NewNop())
	if err != nil {
		t.Fatalf("new oracle (first lifecycle): %v", err)
	}
	reg, key := newTestChain(t, "knirvchain-startup")
	if err := types.SignRegistration(&reg, key); err != nil {
		t.Fatal(err)
	}
	if err := o1.RegisterChain(&reg); err != nil {
		t.Fatalf("register: %v", err)
	}
	cp := &types.Checkpoint{
		SchemaVersion: "knirv.checkpoint.v1",
		ChainID:       reg.ChainID,
		StartHeight:   1,
		EndHeight:     64,
		Proposer:      reg.Authors[0].Address,
	}
	if err := types.SignCheckpoint(cp, key); err != nil {
		t.Fatalf("sign checkpoint: %v", err)
	}
	if _, err := o1.SubmitCheckpoint(cp); err != nil {
		t.Fatalf("admit checkpoint: %v", err)
	}
	root1 := o1.MMRRoot()

	// Simulate exactly the failure mode this bug produced: an
	// audit_mmr_leaf_log.json left behind (by an old binary, or a crash mid
	// dual-write) that does NOT match mmr_leaf_log.json's actual content.
	// Previously this alone was enough to permanently refuse startup.
	stale := []byte(`["deadbeef00000000000000000000000000000000000000000000000000000000"]`)
	if err := os.WriteFile(filepath.Join(dir, "audit_mmr_leaf_log.json"), stale, 0644); err != nil {
		t.Fatalf("write stale audit file: %v", err)
	}

	// Second lifecycle: restart against the same DataDir. Must succeed, and
	// must recover the checkpoint MMR exactly — the stale file is simply
	// never read.
	o2, err := NewOracle(newStartupTestConfig(t, dir), zap.NewNop())
	if err != nil {
		t.Fatalf("new oracle (second lifecycle) unexpectedly failed — a stale/mismatched leftover file should never block startup: %v", err)
	}
	if got := o2.MMRRoot(); got != root1 {
		t.Fatalf("recovered checkpoint MMR %x != original %x", got, root1)
	}
	if size := o2.CheckpointCount(); size != 1 {
		t.Fatalf("expected 1 recovered leaf, got %d", size)
	}
}

// TestNewOracleStartsFromEmptyDataDir is the base case: a first-ever startup
// (no persisted state at all) must succeed and start from an empty MMR.
func TestNewOracleStartsFromEmptyDataDir(t *testing.T) {
	dir := t.TempDir()
	o, err := NewOracle(newStartupTestConfig(t, dir), zap.NewNop())
	if err != nil {
		t.Fatalf("new oracle: %v", err)
	}
	if size := o.CheckpointCount(); size != 0 {
		t.Fatalf("expected empty MMR on first startup, got size %d", size)
	}
}
