package oracle

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/ledger"
	"github.com/knirvcorp/knirvoracle/internal/oracle/token"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// signBondHoldForTest signs a bond-hold on the Oracle's chain id with an arbitrary
// key, mirroring the CLI's controller-signed hold.
func signBondHoldForTest(t *testing.T, key *crypto.KeyPair, chainID string, hold knirvsigning.BondHold) knirvsigning.SignedMessage {
	t.Helper()
	payload, _ := json.Marshal(hold)
	signed, err := knirvsigning.SignMessage(ethcrypto.FromECDSA(key.PrivateKey), knirvsigning.MessageEnvelope{
		SchemaVersion: knirvsigning.MessageSchemaVersion,
		Domain:        knirvsigning.RegistryDomain,
		Purpose:       knirvsigning.PurposeBondHold,
		ChainID:       chainID,
		Nonce:         "bond-hold:" + hold.HoldID,
		IssuedAtUnix:  time.Now().Unix(),
		ExpiresAtUnix: int64(hold.ExpiresAtUnix),
		Payload:       payload,
	})
	if err != nil {
		t.Fatalf("sign hold: %v", err)
	}
	return signed
}

// newBondOracle builds a real Oracle (production init path) with the SDK core
// available so token operations work.
func newBondOracle(t *testing.T) (*Oracle, *crypto.KeyPair) {
	t.Helper()
	t.Setenv("KNIRV_SDK_DATA_DIR", "/tmp/knirvseed")
	dir := t.TempDir()
	o, err := NewOracle(newStartupTestConfig(t, dir), zap.NewNop())
	if err != nil {
		t.Fatalf("new oracle: %v", err)
	}
	opKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	return o, opKey
}

// P2: authority registration that carries a valid pending hold consumes it
// atomically — the escrow moves to the bond owner and a retried registration
// never double-sends.
func TestOracleRegisterChainConsumesBondHold(t *testing.T) {
	o, opKey := newBondOracle(t)
	tok := o.GetNRNToken()
	rootRegID := tok.RootRegistryID()
	if rootRegID == "" {
		t.Fatal("root registry id not set at init")
	}
	operator := opKey.Address
	// Mint enough for a stake (>= minStake of 1000 NRN) and the bond hold.
	if _, err := tok.Mint(operator, big.NewInt(2_000_000_000)); err != nil {
		t.Fatalf("mint: %v", err)
	}
	if _, err := o.economicsEngine.GetStakingManager().Stake(operator, big.NewInt(1_000_000_000)); err != nil {
		t.Fatalf("stake: %v", err)
	}

	holdID := "bond:" + operator.String()
	expiry := time.Now().Add(time.Hour)
	hold := knirvsigning.BondHold{
		HoldID:         holdID,
		Operator:       operator.String(),
		Amount:         "10000",
		RootRegistryID: rootRegID,
		ExpiresAtUnix:  uint64(expiry.Unix()),
	}
	if err := tok.Hold(holdID, hold, signBondHoldForTest(t, opKey, tok.ChainID(), hold), time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}

	reg, regKey := newTestChain(t, "knirvchain-bond-1")
	reg.Bond = 10_000
	reg.BondOwner = operator.String()
	reg.HoldID = holdID
	if err := types.SignRegistration(&reg, regKey); err != nil {
		t.Fatal(err)
	}
	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Registration RESERVES the hold: the 10k escrow is still locked, the operator
	// liquid balance is the staked remainder minus the held amount (1B - 10k). The
	// funds move to the boot node address only via the root-owner-signed Release.
	if got := tok.GetBalance(operator).Int64(); got != 1_000_000_000-10_000 {
		t.Fatalf("operator balance = %d, want 999990000 (escrow still held)", got)
	}
	// A second registration (retry after a response loss) must not change anything.
	_ = o.RegisterChain(&reg)
	if got := tok.GetBalance(operator).Int64(); got != 1_000_000_000-10_000 {
		t.Fatalf("operator balance after retry = %d, want 999990000 (no double send)", got)
	}
	// The hold is consumed (reserved); a redirect to a different chain is rejected,
	// proving the hold->chain binding cannot be redirected.
	other, _ := crypto.GenerateKeyPair()
	if err := tok.ConsumeHold(holdID, "other-chain", other.Address); err == nil {
		t.Fatal("expected already-consumed hold consumed by a different chain to be rejected")
	}
}

// P2: registration that names a hold id with no pending hold is rejected and the
// chain is left unregistered.
func TestOracleRegisterChainMissingHoldRejected(t *testing.T) {
	o, _ := newBondOracle(t)
	reg, regKey := newTestChain(t, "knirvchain-bond-missing")
	reg.Bond = 10_000
	reg.HoldID = "bond:does-not-exist"
	if err := types.SignRegistration(&reg, regKey); err != nil {
		t.Fatal(err)
	}
	if err := o.RegisterChain(&reg); err == nil {
		t.Fatal("expected registration with a missing hold to be rejected")
	}
	if _, ok := o.GetChainRegistration(reg.ChainID); ok {
		t.Fatal("chain was registered despite a missing hold")
	}
}

// P3: the production Oracle sets its root registry id at init from the owner key,
// and a hold against that derived id works immediately (no empty-id panic).
func TestOracleSetsRootRegistryIDAtInit(t *testing.T) {
	o, opKey := newBondOracle(t)
	tok := o.GetNRNToken()
	expected := token.DeriveRootRegistryID(tok.OwnerPublicKeyBytes())
	if got := tok.RootRegistryID(); got != expected {
		t.Fatalf("root registry id = %q, want %q", got, expected)
	}
	operator := opKey.Address
	if _, err := tok.Mint(operator, big.NewInt(100_000)); err != nil {
		t.Fatalf("mint: %v", err)
	}
	holdID := "bond:init"
	hold := knirvsigning.BondHold{
		HoldID:         holdID,
		Operator:       operator.String(),
		Amount:         "1000",
		RootRegistryID: tok.RootRegistryID(),
		ExpiresAtUnix:  uint64(time.Now().Add(time.Hour).Unix()),
	}
	if err := tok.Hold(holdID, hold, signBondHoldForTest(t, opKey, tok.ChainID(), hold), time.Now()); err != nil {
		t.Fatalf("hold against derived root id: %v", err)
	}
}

// P1#2: if consuming the enrolment hold fails, the chain is never registered and
// the pending hold stays recoverable (escrow not moved).
func TestOracleRegisterChainConsumeFailureLeavesPendingHold(t *testing.T) {
	o, opKey := newBondOracle(t)
	tok := o.GetNRNToken()
	operator := opKey.Address
	if _, err := tok.Mint(operator, big.NewInt(2_000_000_000)); err != nil {
		t.Fatalf("mint: %v", err)
	}
	if _, err := o.economicsEngine.GetStakingManager().Stake(operator, big.NewInt(1_000_000_000)); err != nil {
		t.Fatalf("stake: %v", err)
	}
	rootRegID := tok.RootRegistryID()
	holdID := "bond:" + operator.String()
	expiry := time.Now().Add(time.Hour)
	hold := knirvsigning.BondHold{HoldID: holdID, Operator: operator.String(), Amount: "10000", RootRegistryID: rootRegID, ExpiresAtUnix: uint64(expiry.Unix())}
	if err := tok.Hold(holdID, hold, signBondHoldForTest(t, opKey, tok.ChainID(), hold), time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	// Fail all durable ledger writes so the consume cannot commit.
	tok.SetRecorderBatch(func(entries []ledger.BatchEntry) error { return fmt.Errorf("inject ledger failure") })

	reg, regKey := newTestChain(t, "knirvchain-bond-fail")
	reg.Bond = 10_000
	reg.BondOwner = operator.String()
	reg.HoldID = holdID
	if err := types.SignRegistration(&reg, regKey); err != nil {
		t.Fatal(err)
	}
	if err := o.RegisterChain(&reg); err == nil {
		t.Fatal("expected registration to fail when hold consume fails")
	}
	if _, ok := o.GetChainRegistration(reg.ChainID); ok {
		t.Fatal("chain was registered despite a failed hold consume")
	}
	// Escrow untouched: operator holds the staked remainder only (1B - 10k held).
	if got := tok.GetBalance(operator).Int64(); got != 999_990_000 {
		t.Fatalf("operator balance = %d, want 999990000 (hold not consumed)", got)
	}
	// Restore the real recorder: the pending hold is recoverable and can still be consumed.
	tok.SetRecorderBatch(o.recordMutationBatch)
	if err := tok.ConsumeHold(holdID, reg.ChainID, operator); err != nil {
		t.Fatalf("recoverable hold could not be consumed: %v", err)
	}
}

// P1#2: a response-loss retry of an already-applied registration succeeds
// idempotently (no error) and never double-sends the escrow.
func TestOracleRegisterChainIdempotentRetry(t *testing.T) {
	o, opKey := newBondOracle(t)
	tok := o.GetNRNToken()
	operator := opKey.Address
	if _, err := tok.Mint(operator, big.NewInt(2_000_000_000)); err != nil {
		t.Fatalf("mint: %v", err)
	}
	if _, err := o.economicsEngine.GetStakingManager().Stake(operator, big.NewInt(1_000_000_000)); err != nil {
		t.Fatalf("stake: %v", err)
	}
	rootRegID := tok.RootRegistryID()
	holdID := "bond:" + operator.String()
	expiry := time.Now().Add(time.Hour)
	hold := knirvsigning.BondHold{HoldID: holdID, Operator: operator.String(), Amount: "10000", RootRegistryID: rootRegID, ExpiresAtUnix: uint64(expiry.Unix())}
	if err := tok.Hold(holdID, hold, signBondHoldForTest(t, opKey, tok.ChainID(), hold), time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	reg, regKey := newTestChain(t, "knirvchain-bond-retry")
	reg.Bond = 10_000
	reg.BondOwner = operator.String()
	reg.HoldID = holdID
	if err := types.SignRegistration(&reg, regKey); err != nil {
		t.Fatal(err)
	}
	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("first register: %v", err)
	}
	before := tok.GetBalance(operator).Int64()
	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("idempotent retry should succeed: %v", err)
	}
	if got := tok.GetBalance(operator).Int64(); got != before {
		t.Fatalf("retry changed balance: before %d after %d (double send?)", before, got)
	}
}

// P1#2: after a hold is consumed, a registration that redirects the same hold to a
// different owner is rejected (the chain is already registered and the hold spent).
func TestOracleRegisterChainRedirectAfterConsumeRejected(t *testing.T) {
	o, opKey := newBondOracle(t)
	tok := o.GetNRNToken()
	operator := opKey.Address
	if _, err := tok.Mint(operator, big.NewInt(2_000_000_000)); err != nil {
		t.Fatalf("mint: %v", err)
	}
	if _, err := o.economicsEngine.GetStakingManager().Stake(operator, big.NewInt(1_000_000_000)); err != nil {
		t.Fatalf("stake: %v", err)
	}
	rootRegID := tok.RootRegistryID()
	holdID := "bond:" + operator.String()
	expiry := time.Now().Add(time.Hour)
	hold := knirvsigning.BondHold{HoldID: holdID, Operator: operator.String(), Amount: "10000", RootRegistryID: rootRegID, ExpiresAtUnix: uint64(expiry.Unix())}
	if err := tok.Hold(holdID, hold, signBondHoldForTest(t, opKey, tok.ChainID(), hold), time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	reg, regKey := newTestChain(t, "knirvchain-bond-redirect")
	reg.Bond = 10_000
	reg.BondOwner = operator.String()
	reg.HoldID = holdID
	if err := types.SignRegistration(&reg, regKey); err != nil {
		t.Fatal(err)
	}
	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("register: %v", err)
	}
	// Redirect: same chain, different bond owner (different recipient for the hold).
	other, _ := crypto.GenerateKeyPair()
	reg2, regKey2 := newTestChain(t, "knirvchain-bond-redirect")
	reg2.Bond = 10_000
	reg2.BondOwner = other.Address.String()
	reg2.HoldID = holdID
	if err := types.SignRegistration(&reg2, regKey2); err != nil {
		t.Fatal(err)
	}
	if err := o.RegisterChain(&reg2); err == nil {
		t.Fatal("expected redirect-after-consume registration to be rejected")
	}
}
