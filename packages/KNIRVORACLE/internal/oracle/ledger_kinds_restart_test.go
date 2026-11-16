package oracle

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/governance"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// Per-kind restart coverage for the ledger. TestLedgerSurvivesRestart covers
// mint only. Each test below mutates one more kind, restarts the Oracle from the
// same data dir, and requires the same balances, nonces, supply, wallets,
// proposals, votes and state root.

// oracleSnapshot is what a restart must reproduce.
type oracleSnapshot struct {
	root    string
	supply  string
	balance map[string]string
}

func snapshotOf(o *Oracle, addrs ...types.Address) oracleSnapshot {
	s := oracleSnapshot{root: o.stateRoot, supply: o.GetNRNToken().TotalSupply().String(),
		balance: map[string]string{}}
	for _, a := range addrs {
		s.balance[a.String()] = o.GetNRNToken().GetBalance(a).String()
	}
	return s
}

func assertSnapshotEqual(t *testing.T, before, after oracleSnapshot) {
	t.Helper()
	if before.root != after.root {
		t.Fatalf("state root changed across restart: %s -> %s", before.root, after.root)
	}
	if before.supply != after.supply {
		t.Fatalf("supply changed across restart: %s -> %s", before.supply, after.supply)
	}
	for k, v := range before.balance {
		if after.balance[k] != v {
			t.Fatalf("balance of %s changed across restart: %s -> %s", k, v, after.balance[k])
		}
	}
}

// startOracle opens an Oracle on cfg's data dir and returns it with its owner
// address. The caller closes the ledger.
func startOracle(t *testing.T, cfg *OracleConfig) *Oracle {
	t.Helper()
	o, err := NewOracle(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	return o
}

func TestLedgerTransferBurnWalletSurviveRestart(t *testing.T) {
	cfg, _ := testOracleConfig(t)
	alice, err := types.AddressFromString("knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e")
	if err != nil {
		t.Fatalf("alice: %v", err)
	}
	ownerAddr, err := crypto.AddressFromPrivateKey(cfg.OwnerPrivateKey)
	if err != nil {
		t.Fatalf("owner address: %v", err)
	}

	first := startOracle(t, cfg)
	token := first.GetNRNToken()
	if _, err := token.Mint(alice, big.NewInt(100)); err != nil {
		t.Fatalf("mint: %v", err)
	}
	if !token.RegisterAddress(alice) {
		t.Fatal("wallet register returned false for a new address")
	}
	if err := token.TransferBetween(ownerAddr, alice, big.NewInt(25)); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if err := token.BurnFrom(alice, big.NewInt(10), "restart test"); err != nil {
		t.Fatalf("burn: %v", err)
	}
	before := snapshotOf(first, alice, ownerAddr)
	if before.balance[alice.String()] != "115" {
		t.Fatalf("alice before restart = %s, want 115 (100+25-10)", before.balance[alice.String()])
	}
	first.ledger.Close()

	second := startOracle(t, cfg)
	defer second.ledger.Close()
	if second.ledgerDegraded {
		t.Fatal("restart with matching config must not be degraded")
	}
	assertSnapshotEqual(t, before, snapshotOf(second, alice, ownerAddr))
	// RegisterAddress returns false for an address that is already registered.
	if second.GetNRNToken().RegisterAddress(alice) {
		t.Fatal("wallet registration lost across restart")
	}
}

func TestLedgerProposalSurvivesRestart(t *testing.T) {
	cfg, _ := testOracleConfig(t)
	proposer, err := types.AddressFromString("knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e")
	if err != nil {
		t.Fatalf("proposer: %v", err)
	}

	first := startOracle(t, cfg)
	gs := first.GetGovernanceSystem()
	deposit := new(big.Int).Set(gs.GetNetworkParameters().ProposalDeposit)
	proposal, err := gs.CreateProposal(&governance.ProposalRequest{
		Type:        governance.ProposalTypeTextProposal,
		Title:       "restart proposal",
		Description: "must survive a restart",
		Proposer:    proposer,
		Content:     map[string]interface{}{"text": "hello"},
	}, deposit)
	if err != nil {
		t.Fatalf("create proposal: %v", err)
	}
	wantID, wantStatus := proposal.ID, proposal.Status
	wantRoot := first.stateRoot
	first.ledger.Close()

	second := startOracle(t, cfg)
	defer second.ledger.Close()
	if second.stateRoot != wantRoot {
		t.Fatalf("state root after restart = %s, want %s", second.stateRoot, wantRoot)
	}
	got, err := second.GetGovernanceSystem().GetProposal(wantID)
	if err != nil {
		t.Fatalf("proposal lost across restart: %v", err)
	}
	if got.Title != "restart proposal" || got.Status != wantStatus {
		t.Fatalf("proposal after restart = %q/%v, want %q/%v", got.Title, got.Status, "restart proposal", wantStatus)
	}
}

// A vote is cast by an active validator with a signed governance-vote message.
// Voting opens an hour after submission, so the test moves the voting window of
// the proposal it just created to the present through the returned pointer. The
// vote is then made durable and must survive a restart.
func TestLedgerVoteSurvivesRestart(t *testing.T) {
	cfg, _ := testOracleConfig(t)
	voterKey, err := ethcrypto.GenerateKey()
	if err != nil {
		t.Fatalf("voter key: %v", err)
	}
	voterHex := hex.EncodeToString(ethcrypto.FromECDSA(voterKey))
	voter, err := crypto.AddressFromPrivateKey(voterHex)
	if err != nil {
		t.Fatalf("voter address: %v", err)
	}

	first := startOracle(t, cfg)
	gs := first.GetGovernanceSystem()
	minStake := gs.GetNetworkParameters().MinValidatorStake
	if _, err := gs.RegisterValidator(&governance.ValidatorRegistration{
		Address:        voter,
		PubKey:         ethcrypto.CompressPubkey(&voterKey.PublicKey),
		StakedAmount:   new(big.Int).Add(minStake, big.NewInt(1)),
		CommissionRate: 0.1,
	}); err != nil {
		t.Fatalf("register validator: %v", err)
	}
	proposal, err := gs.CreateProposal(&governance.ProposalRequest{
		Type:        governance.ProposalTypeTextProposal,
		Title:       "vote proposal",
		Description: "voted on before restart",
		Proposer:    voter,
		Content:     map[string]interface{}{"text": "vote"},
	}, new(big.Int).Set(gs.GetNetworkParameters().ProposalDeposit))
	if err != nil {
		t.Fatalf("create proposal: %v", err)
	}
	// Open the voting window now. GetProposal returns the stored pointer.
	proposal.VotingStart = time.Now().Add(-time.Minute)
	proposal.VotingEnd = time.Now().Add(time.Hour)

	voteData := "vote:" + proposal.ID + ":" + voter.String() + ":" + governance.VoteOptionYes.String()
	signed, err := knirvsigning.SignMessage(ethcrypto.FromECDSA(voterKey), knirvsigning.MessageEnvelope{
		Domain:        "knirv.oracle",
		Purpose:       "governance-vote",
		ChainID:       cfg.ChainID,
		Nonce:         "vote:" + proposal.ID,
		IssuedAtUnix:  time.Now().Unix(),
		ExpiresAtUnix: time.Now().Add(time.Hour).Unix(),
		Payload:       []byte(voteData),
	})
	if err != nil {
		t.Fatalf("sign vote: %v", err)
	}
	if _, err := gs.CastVote(&governance.VoteRequest{ProposalID: proposal.ID, Voter: voter, Option: governance.VoteOptionYes}, signed); err != nil {
		t.Fatalf("cast vote: %v", err)
	}
	wantRoot := first.stateRoot
	first.ledger.Close()

	second := startOracle(t, cfg)
	defer second.ledger.Close()
	if second.stateRoot != wantRoot {
		t.Fatalf("state root after restart = %s, want %s", second.stateRoot, wantRoot)
	}
	vote, err := second.GetGovernanceSystem().GetVote(proposal.ID, voter)
	if err != nil {
		t.Fatalf("vote lost across restart: %v", err)
	}
	if vote.Option != governance.VoteOptionYes {
		t.Fatalf("vote option after restart = %v, want yes", vote.Option)
	}
}
