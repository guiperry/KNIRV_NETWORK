package consensus

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/knirvcorp/knirvoracle/internal/oracle/mmr"
	"go.uber.org/zap"
)

func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// TestCommitPersistsMMRAndRecoversAfterRestart verifies the Phase-3 invariant
// that the Oracle AppHash survives a restart: the audit MMR is persisted to disk
// on Commit and reconstructed identically on the next NewABCIApplication, so a
// freshly-loaded AppHash equals the independently-bagged pre-shutdown root.
func TestCommitPersistsMMRAndRecoversAfterRestart(t *testing.T) {
	dir, err := os.MkdirTemp("", "oracle-mmr-recovery")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// First lifecycle: append leaves and commit (persisting the audit MMR).
	app1 := NewABCIApplication("recover-test", dir, zap.NewNop())
	for _, leaf := range [][]byte{[]byte("checkpoint-1"), []byte("finality-1"), []byte("checkpoint-2")} {
		if _, _, err := app1.auditMMR.Add(leaf); err != nil {
			t.Fatal(err)
		}
	}
	want, err := app1.Commit()
	if err != nil {
		t.Fatal(err)
	}
	// Sanity: AppHash equals the bagged root of the audit MMR.
	if got := app1.auditMMR.BagRoot(); !bytes.Equal(want, got[:]) {
		t.Fatalf("app hash %x != bagged root %x", want, got)
	}

	// Capture the independent bagged root from the leaf log.
	leaves := app1.auditMMR.Leaves()
	indep := mmr.FromLeaves(leaves).BagRoot()
	if !bytes.Equal(want, indep[:]) {
		t.Fatalf("pre-shutdown app hash %x != independently bagged root %x", want, indep)
	}

	// Second lifecycle: simulate a restart by reloading from the same DataDir.
	app2 := NewABCIApplication("recover-test", dir, zap.NewNop())
	recovered := app2.auditMMR.BagRoot()
	if !bytes.Equal(recovered[:], want) {
		t.Fatalf("recovered root %x != pre-shutdown root %x", recovered, want)
	}
	// A post-reload Commit must recompute the exact same AppHash as the
	// pre-shutdown commit — the persisted MMR reconstructs identically.
	recommit, err := app2.Commit()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(recommit, want) {
		t.Fatalf("recovered AppHash %x != pre-shutdown root %x", recommit, want)
	}
	if !bytes.Equal(app2.GetState().AppHash, want) {
		t.Fatalf("recovered state app hash %x != pre-shutdown root %x", app2.GetState().AppHash, want)
	}

	// The audit MMR leaf log file must exist and be non-empty.
	if fi, err := os.Stat(filepath.Join(dir, "audit_mmr_leaf_log.json")); err != nil || fi.Size() == 0 {
		t.Fatalf("audit MMR leaf log not persisted: err=%v size=%v", err, fi.Size())
	}
}

// TestDeliverTxRoutesCheckpointTxToAdmission verifies the Phase-3 wiring that
// checkpoint/finality transactions delivered through consensus take the same
// admission path as the HTTP endpoints.
func TestDeliverTxRoutesCheckpointTxToAdmission(t *testing.T) {
	app := NewABCIApplication("route-test", "", zap.NewNop())
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
