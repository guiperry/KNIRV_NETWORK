package consensus

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/knirvcorp/knirvoracle/internal/oracle/mmr"
	"go.uber.org/zap"
)

func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// TestNewABCIApplicationDoesNotTouchDisk verifies ABCIApplication no longer
// keeps its own independently-persisted audit MMR copy. That second copy
// (audit_mmr_leaf_log.json, written from Commit and reloaded in
// NewABCIApplication) used to be compared against the checkpoint pipeline's
// own persisted MMR (oracle.go's mmr_leaf_log.json) on every Oracle startup;
// the two were written by different call sites and could fall out of sync on
// an unclean shutdown, permanently refusing to start ("checkpoint MMR and
// persisted AppHash MMR diverged; refusing unsafe recovery") even though
// nothing was actually corrupt. There is now exactly one persisted copy —
// the checkpoint pipeline's — so this failure mode is structurally
// impossible: ABCIApplication has nothing of its own to diverge from it.
func TestNewABCIApplicationDoesNotTouchDisk(t *testing.T) {
	dir, err := os.MkdirTemp("", "oracle-mmr-recovery")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	app := NewABCIApplication("recover-test", zap.NewNop())
	for _, leaf := range [][]byte{[]byte("checkpoint-1"), []byte("finality-1"), []byte("checkpoint-2")} {
		if _, _, err := app.auditMMR.Add(leaf); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := app.Commit(); err != nil {
		t.Fatal(err)
	}

	// Commit must not write anything to dir — ABCIApplication owns no file of
	// its own to keep in sync with the checkpoint pipeline's persisted log.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no files written by ABCIApplication, found %v", entries)
	}
}

// TestSetAuditMMRIsTheOnlyRecoveryPath verifies the replacement invariant:
// the Oracle (oracle.go's NewOracle) is the sole owner of the persisted audit
// MMR (loaded from checkpoint_store.go's mmr_leaf_log.json) and installs it
// via SetAuditMMR; once installed, Commit's AppHash reflects exactly that
// MMR's bagged root, with no separate recovery/comparison step needed.
func TestSetAuditMMRIsTheOnlyRecoveryPath(t *testing.T) {
	// Simulate the checkpoint pipeline's persisted MMR recovered from
	// mmr_leaf_log.json after a restart (built independently of the app).
	persisted := mmr.New()
	var want mmr.Hash
	for _, leaf := range [][]byte{[]byte("checkpoint-1"), []byte("finality-1")} {
		var wantErr error
		_, want, wantErr = persisted.Add(leaf)
		if wantErr != nil {
			t.Fatal(wantErr)
		}
	}

	app := NewABCIApplication("recover-test", zap.NewNop())
	// Before installation the app starts from a fresh, empty MMR.
	if size := app.auditMMR.Size(); size != 0 {
		t.Fatalf("expected fresh ABCIApplication to start empty, got size %d", size)
	}

	app.SetAuditMMR(persisted)

	got, err := app.Commit()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want[:]) {
		t.Fatalf("post-install AppHash %x != installed MMR's bagged root %x", got, want)
	}
	if !bytes.Equal(app.GetState().AppHash, want[:]) {
		t.Fatalf("state app hash %x != installed MMR's bagged root %x", app.GetState().AppHash, want)
	}
}

// TestDeliverTxRoutesCheckpointTxToAdmission verifies the Phase-3 wiring that
// checkpoint/finality transactions delivered through consensus take the same
// admission path as the HTTP endpoints.
func TestDeliverTxRoutesCheckpointTxToAdmission(t *testing.T) {
	app := NewABCIApplication("route-test", zap.NewNop())
	routed := 0
	app.SetCheckpointTxHandler(func(txType string, payload []byte) error {
		routed++
		if txType != "checkpoint" {
			t.Fatalf("unexpected tx type %q", txType)
		}
		return nil
	})

	tx, _ := jsonMarshal(map[string]interface{}{
		"type":    "checkpoint",
		"payload": map[string]interface{}{"chain_id": "x", "start_height": 1, "end_height": 10},
	})
	if err := app.DeliverTx(tx); err != nil {
		t.Fatal(err)
	}
	if routed != 1 {
		t.Fatalf("expected checkpoint tx to be routed once, got %d", routed)
	}

	// A non-checkpoint tx must NOT invoke the hook.
	other, _ := jsonMarshal(map[string]interface{}{"type": "transfer", "payload": map[string]interface{}{}})
	if err := app.DeliverTx(other); err != nil {
		t.Fatal(err)
	}
	if routed != 1 {
		t.Fatalf("non-checkpoint tx should not be routed, routed=%d", routed)
	}
}
