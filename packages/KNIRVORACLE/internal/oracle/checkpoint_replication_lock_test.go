package oracle

import (
	"testing"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

func TestProjectedRollupPublishesReplicationHead(t *testing.T) {
	o, err := NewOracle(newStartupTestConfig(t, t.TempDir()), zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer o.cancel()
	if o.replication == nil || o.replication.signer == nil {
		t.Fatal("test requires an active replication signer")
	}
	before, ok := o.replication.CurrentHead()
	if !ok {
		t.Fatal("initial head missing")
	}
	result := make(chan error, 1)
	go func() {
		_, err := o.ProjectRollup(&types.RollupRecord{ID: "test", ChainID: "test-chain", StartHeight: 1, EndHeight: 1})
		result <- err
	}()
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("rollup admission deadlocked while producing a replication head")
	}
	after, ok := o.replication.CurrentHead()
	if !ok || after.Head.Seq <= before.Head.Seq || after.Head.MMRSize != before.Head.MMRSize+1 {
		t.Fatalf("head did not advance for the rollup: before=%+v after=%+v", before.Head, after.Head)
	}
}
