package consensus

import (
	"bytes"
	"testing"

	"github.com/knirvcorp/knirvoracle/internal/oracle/mmr"
	"go.uber.org/zap"
)

func TestCommitUsesMMRBagRootAsAppHash(t *testing.T) {
	app := NewABCIApplication("mmr-test", "", zap.NewNop())
	assertCommitRoot := func(height int, want mmr.Hash) {
		t.Helper()
		got, err := app.Commit()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want[:]) {
			t.Fatalf("height %d app hash = %x, want %x", height, got, want)
		}
		if !bytes.Equal(app.GetState().AppHash, want[:]) {
			t.Fatalf("height %d state app hash does not match MMR root", height)
		}
	}

	assertCommitRoot(1, mmr.EmptyRoot())
	_, want, err := app.auditMMR.Add([]byte("checkpoint-1"))
	if err != nil {
		t.Fatal(err)
	}
	assertCommitRoot(2, want)
	_, want, err = app.auditMMR.Add([]byte("finality-1"))
	if err != nil {
		t.Fatal(err)
	}
	assertCommitRoot(3, want)
}
