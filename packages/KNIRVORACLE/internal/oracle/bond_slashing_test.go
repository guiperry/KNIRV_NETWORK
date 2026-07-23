package oracle

import (
	"math/big"
	"strings"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/economics"
	"github.com/knirvcorp/knirvoracle/internal/oracle/token"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

func TestWindowMissSlashesBondAndQueuesConsensusEvidence(t *testing.T) {
	o := newPhase4Oracle(t)
	nrn, err := token.NewNRN("NRN", "NRN", big.NewInt(10_000_000_000), big.NewInt(20_000_000_000), strings.Repeat("02", 32))
	if err != nil {
		t.Fatal(err)
	}
	o.economicsEngine = economics.NewEconomicsEngine(nrn, zap.NewNop())
	owner := nrn.Owner()
	if _, err := o.economicsEngine.GetStakingManager().Stake(owner, big.NewInt(2_000_000_000)); err != nil {
		t.Fatalf("stake: %v", err)
	}
	key, err := ethcrypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	reg := &types.ChainRegistration{
		ChainID: "bonded-chain", Authors: []types.RegisteredAuthor{{Address: types.OracleAddress(&key.PublicKey), PubKey: ethcrypto.CompressPubkey(&key.PublicKey), Weight: 1}},
		QuorumNumer: 2, QuorumDenom: 3, Bond: 1_000_000_000, BondOwner: owner.String(), ProofWindow: 1,
	}
	if err := types.SignRegistration(reg, key); err != nil {
		t.Fatal(err)
	}
	if err := o.RegisterChain(reg); err != nil {
		t.Fatalf("register bonded chain: %v", err)
	}
	rec := projectTestRollupAs(t, o, "bonded-chain", reg.Authors[0].Address, reg.Authors[0].PubKey, key)
	rec.FinalByHeight = 0
	o.sweepOnce()

	reg, ok := o.GetChainRegistration("bonded-chain")
	if !ok {
		t.Fatal("registration missing")
	}
	if reg.BondRemaining != 900_000_000 || reg.BondSlashed != 100_000_000 {
		t.Fatalf("unexpected bond view: remaining=%d slashed=%d", reg.BondRemaining, reg.BondSlashed)
	}
	if evidence := o.consensusEngine.PendingEvidence(); len(evidence) != 1 {
		t.Fatalf("pending consensus evidence = %d, want 1", len(evidence))
	}
	if burned := o.economicsEngine.GetBurnTracker().GetBurnsByReason(economics.BurnReasonSlashing); burned.Uint64() != 100_000_000 {
		t.Fatalf("slashed burn = %s", burned)
	}
}
