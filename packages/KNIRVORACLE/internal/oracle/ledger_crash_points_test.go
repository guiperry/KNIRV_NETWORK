package oracle

import (
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"github.com/knirvcorp/knirvoracle/internal/testhooks"
)

// Crash points on the Oracle's live mutation path: validate, append and fsync the
// record, commit its audit leaf, apply to memory. A crash after the fsync and
// before the leaf or the apply must be repaired by replay on restart (the log is
// authoritative), and the leaf gap must be reported as Degraded, not hidden.

var errCrashInjected = errors.New("injected crash")

// A record durable in the log but never acknowledged (the process died before
// the leaf and the apply) replays on restart, and the missing leaf is reported.
func TestCrashAfterLogFsyncReplaysRecordAndReportsMissingLeaf(t *testing.T) {
	cfg, _ := testOracleConfig(t)
	recipient, err := types.AddressFromString("knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e")
	if err != nil {
		t.Fatalf("recipient: %v", err)
	}

	first := startOracle(t, cfg)
	genesisSupply := first.GetNRNToken().TotalSupply().String()
	restoreFault := testhooks.SetLedgerFault(first.ledger.Path(), func(stage string) error {
		if stage == "after_sync" {
			return errCrashInjected
		}
		return nil
	})
	_, mintErr := first.GetNRNToken().Mint(recipient, big.NewInt(42))
	restoreFault() // the restart below must not inherit the injected crash
	if mintErr == nil {
		t.Fatal("mint succeeded through an injected crash")
	}
	if got := first.GetNRNToken().GetBalance(recipient).String(); got != "0" {
		t.Fatalf("in-memory balance after failed mint = %s, want 0", got)
	}
	first.ledger.Close()

	second := startOracle(t, cfg)
	defer second.ledger.Close()
	if got := second.GetNRNToken().GetBalance(recipient).String(); got != "42" {
		t.Fatalf("balance after restart = %s, want 42 (the fsynced record is durable)", got)
	}
	wantSupply := new(big.Int)
	wantSupply.SetString(genesisSupply, 10)
	wantSupply.Add(wantSupply, big.NewInt(42))
	if got := second.GetNRNToken().TotalSupply().String(); got != wantSupply.String() {
		t.Fatalf("supply after restart = %s, want %s", got, wantSupply)
	}
	if !second.ledgerDegraded {
		t.Fatal("a record with no audit leaf must start the Oracle Degraded")
	}
}

// Removing the last audit leaf from the persisted leaf log (a crash between the
// log append and the leaf append) must be detected at startup.
func TestMissingAuditLeafOnDiskStartsDegraded(t *testing.T) {
	cfg, dir := testOracleConfig(t)
	recipient, err := types.AddressFromString("knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e")
	if err != nil {
		t.Fatalf("recipient: %v", err)
	}

	first := startOracle(t, cfg)
	if _, err := first.GetNRNToken().Mint(recipient, big.NewInt(42)); err != nil {
		t.Fatalf("mint: %v", err)
	}
	wantRoot := first.stateRoot
	first.ledger.Close()

	leafPath := filepath.Join(dir, "mmr_leaf_log.json")
	raw, err := os.ReadFile(leafPath)
	if err != nil {
		t.Fatalf("read leaf log: %v", err)
	}
	var leaves []json.RawMessage
	if err := json.Unmarshal(raw, &leaves); err != nil {
		t.Fatalf("decode leaf log: %v", err)
	}
	if len(leaves) == 0 {
		t.Fatal("leaf log is empty; the mint should have committed a leaf")
	}
	trimmed, err := json.Marshal(leaves[:len(leaves)-1])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(leafPath, trimmed, 0644); err != nil {
		t.Fatal(err)
	}

	second := startOracle(t, cfg)
	defer second.ledger.Close()
	if !second.ledgerDegraded {
		t.Fatal("a logged record with no audit leaf must start Degraded")
	}
	if second.stateRoot != wantRoot {
		t.Fatalf("state root changed by a missing leaf: %s != %s", second.stateRoot, wantRoot)
	}
	if got := second.GetNRNToken().GetBalance(recipient).String(); got != "42" {
		t.Fatalf("balance = %s, want 42 (the ledger wins)", got)
	}
}
