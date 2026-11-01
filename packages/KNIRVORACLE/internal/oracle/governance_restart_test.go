package oracle

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/governance"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

func testOracleConfig(t *testing.T) (*OracleConfig, string) {
	t.Helper()
	ownerKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("owner key: %v", err)
	}
	cfg := DefaultOracleConfig()
	cfg.DataDir = t.TempDir()
	cfg.DBBackend = "memory"
	cfg.OwnerPrivateKey = ownerKey.PrivateKeyHex()
	return cfg, cfg.DataDir
}

// Governance must survive a restart too: proposals, votes, validators and the
// network parameters are one snapshot record per changed state.
func TestGovernanceSurvivesRestart(t *testing.T) {
	cfg, _ := testOracleConfig(t)
	validatorAddr, err := types.AddressFromString("knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e")
	if err != nil {
		t.Fatalf("validator address: %v", err)
	}

	first, err := NewOracle(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("first start: %v", err)
	}
	minStake := first.GetGovernanceSystem().GetNetworkParameters().MinValidatorStake
	if _, err := first.GetGovernanceSystem().RegisterValidator(&governance.ValidatorRegistration{
		Address:        validatorAddr,
		PubKey:         []byte{0x02, 0x03},
		StakedAmount:   new(big.Int).Add(minStake, big.NewInt(1)),
		CommissionRate: 0.1,
	}); err != nil {
		t.Fatalf("register validator: %v", err)
	}
	if got := len(first.GetGovernanceSystem().ListValidators()); got != 1 {
		t.Fatalf("validators before restart = %d, want 1", got)
	}
	wantRoot := first.stateRoot
	first.ledger.Close()

	second, err := NewOracle(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("restart: %v", err)
	}
	defer second.ledger.Close()

	validators := second.GetGovernanceSystem().ListValidators()
	if len(validators) != 1 {
		t.Fatalf("validators after restart = %d, want 1", len(validators))
	}
	if validators[0].Address != validatorAddr {
		t.Fatalf("validator address after restart = %s, want %s", validators[0].Address, validatorAddr)
	}
	if second.stateRoot != wantRoot {
		t.Fatalf("state root after restart = %s, want %s", second.stateRoot, wantRoot)
	}
}

// Every committed ledger record must become an audit-MMR leaf, and the leaf log
// must be on disk, so a checkpoint commits to the state that produced it.
func TestLedgerRecordsBecomeAuditLeaves(t *testing.T) {
	cfg, dir := testOracleConfig(t)
	instance, err := NewOracle(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer instance.ledger.Close()

	recipient, err := types.AddressFromString("knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e")
	if err != nil {
		t.Fatalf("recipient: %v", err)
	}

	instance.checkpoint.mu.RLock()
	before := instance.checkpoint.mmr.Size()
	instance.checkpoint.mu.RUnlock()

	if _, err := instance.GetNRNToken().Mint(recipient, big.NewInt(7)); err != nil {
		t.Fatalf("mint: %v", err)
	}

	instance.checkpoint.mu.RLock()
	after := instance.checkpoint.mmr.Size()
	instance.checkpoint.mu.RUnlock()

	// A fresh Oracle starts with an empty audit MMR; the mint adds one leaf.
	if before != 0 {
		t.Fatalf("audit MMR had %d leaves at startup, want 0", before)
	}
	if after-before != 1 {
		t.Fatalf("audit MMR grew by %d leaves on mint, want 1", after-before)
	}
	if _, err := os.Stat(filepath.Join(dir, "mmr_leaf_log.json")); err != nil {
		t.Fatalf("leaf log not persisted: %v", err)
	}
}
