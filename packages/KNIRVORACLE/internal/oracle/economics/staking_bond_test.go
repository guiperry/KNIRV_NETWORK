package economics

import (
	"math/big"
	"strings"
	"testing"

	"github.com/knirvcorp/knirvoracle/internal/oracle/token"
	"go.uber.org/zap"
)

func TestChainBondLocksAndSlashesStakedNRN(t *testing.T) {
	nrn, err := token.NewNRN("NRN", "NRN", big.NewInt(10_000_000_000), big.NewInt(20_000_000_000), strings.Repeat("01", 32))
	if err != nil {
		t.Fatal(err)
	}
	engine := NewEconomicsEngine(nrn, zap.NewNop())
	owner := nrn.Owner()
	if _, err := engine.GetStakingManager().Stake(owner, big.NewInt(2_000_000_000)); err != nil {
		t.Fatalf("stake: %v", err)
	}
	if _, err := engine.LockChainBond("chain-1", owner, big.NewInt(1_000_000_000)); err != nil {
		t.Fatalf("lock bond: %v", err)
	}
	if _, err := engine.LockChainBond("chain-2", owner, big.NewInt(1_500_000_000)); err == nil {
		t.Fatal("same stake must not be overcommitted")
	}
	if err := engine.GetStakingManager().Unstake(owner, big.NewInt(1_500_000_000)); err == nil {
		t.Fatal("locked stake must not be unbonded")
	}
	if err := engine.GetStakingManager().Unstake(owner, big.NewInt(1_000_000_000)); err != nil {
		t.Fatalf("unbonded portion should remain withdrawable: %v", err)
	}
	bond, actual, err := engine.SlashChainBond("chain-1", "window-miss", big.NewInt(100_000_000))
	if err != nil {
		t.Fatalf("slash bond: %v", err)
	}
	if actual.Int64() != 100_000_000 || bond.Remaining.Int64() != 900_000_000 {
		t.Fatalf("unexpected slash result: actual=%s remaining=%s", actual, bond.Remaining)
	}
	if burned := engine.GetBurnTracker().GetBurnsByReason(BurnReasonSlashing); burned.Cmp(actual) != 0 {
		t.Fatalf("slashing burn = %s, want %s", burned, actual)
	}
}
