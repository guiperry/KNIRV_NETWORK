package oracle

import (
	"math/big"
	"testing"

	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// A restart must not re-mint or reset anything: the log is the source of truth
// (D-R9/D-R10), and the same records must replay to the same state root.
func TestLedgerSurvivesRestart(t *testing.T) {
	ownerKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("owner key: %v", err)
	}
	recipient, err := types.AddressFromString("knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e")
	if err != nil {
		t.Fatalf("recipient: %v", err)
	}

	cfg := DefaultOracleConfig()
	cfg.DataDir = t.TempDir()
	cfg.DBBackend = "memory"
	cfg.OwnerPrivateKey = ownerKey.PrivateKeyHex()

	first, err := NewOracle(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("first start: %v", err)
	}
	if _, err := first.GetNRNToken().Mint(recipient, big.NewInt(42)); err != nil {
		t.Fatalf("mint: %v", err)
	}
	wantRoot := first.stateRoot
	wantSupply := first.GetNRNToken().TotalSupply().String()
	if first.ledger.Len() != 2 {
		t.Fatalf("log has %d records after genesis+mint, want 2", first.ledger.Len())
	}
	if err := first.ledger.Close(); err != nil {
		t.Fatalf("close first log: %v", err)
	}

	second, err := NewOracle(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("restart: %v", err)
	}
	if got := second.GetNRNToken().GetBalance(recipient).String(); got != "42" {
		t.Fatalf("balance after restart = %s, want 42", got)
	}
	if got := second.GetNRNToken().TotalSupply().String(); got != wantSupply {
		t.Fatalf("supply after restart = %s, want %s", got, wantSupply)
	}
	if second.stateRoot != wantRoot {
		t.Fatalf("state root after restart = %s, want %s", second.stateRoot, wantRoot)
	}
	if second.ledgerDegraded {
		t.Fatal("restart with matching config must not be degraded")
	}
	second.ledger.Close()
}

// A configuration change to the initial supply must not rewrite a live ledger;
// the node reports Degraded and keeps the logged numbers.
func TestLedgerWinsOverChangedConfig(t *testing.T) {
	ownerKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("owner key: %v", err)
	}
	cfg := DefaultOracleConfig()
	cfg.DataDir = t.TempDir()
	cfg.DBBackend = "memory"
	cfg.OwnerPrivateKey = ownerKey.PrivateKeyHex()

	first, err := NewOracle(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("first start: %v", err)
	}
	loggedSupply := first.GetNRNToken().TotalSupply().String()
	first.ledger.Close()

	cfg.InitialSupply = big.NewInt(999)
	second, err := NewOracle(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("restart: %v", err)
	}
	defer second.ledger.Close()
	if !second.ledgerDegraded {
		t.Fatal("changed genesis config must mark the ledger Degraded")
	}
	if got := second.GetNRNToken().TotalSupply().String(); got != loggedSupply {
		t.Fatalf("supply = %s, want the logged %s", got, loggedSupply)
	}
}
