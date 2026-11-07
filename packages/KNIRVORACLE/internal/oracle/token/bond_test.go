package token

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/ledger"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

const testRootRegistryID = "root-reg-1"

// bondTestEnv wires an NRN token with a distinct owner (root owner, signs
// releases) and operator (signs holds), plus a recorder that captures the
// durable ledger records so restart can be simulated.
type bondTestEnv struct {
	nrn              *NRN
	ownerKP          *crypto.KeyPair
	operatorKP       *crypto.KeyPair
	rootRegistryID   string
	recorded         []ledger.Record
}

func newBondTestEnv(t *testing.T) *bondTestEnv {
	t.Helper()
	ownerKP, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	operatorKP, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	ownerHex := hex.EncodeToString(ethcrypto.FromECDSA(ownerKP.PrivateKey))
	nrn, err := NewNRN("NRN", "NRN", big.NewInt(1_000_000_000), big.NewInt(2_000_000_000), ownerHex)
	if err != nil {
		t.Fatal(err)
	}
	nrn.SetRootRegistryID(testRootRegistryID)
	env := &bondTestEnv{nrn: nrn, ownerKP: ownerKP, operatorKP: operatorKP, rootRegistryID: testRootRegistryID}
	nrn.SetRecorder(func(kind string, payload any) error {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		env.recorded = append(env.recorded, ledger.Record{
			Seq:     uint64(len(env.recorded) + 1),
			Kind:    kind,
			Payload: raw,
		})
		return nil
	})
	// Fund the operator so holds have balance.
	nrn.balances[operatorKP.Address] = big.NewInt(100_000)
	return env
}

// freshAddr returns an arbitrary valid address distinct from the env's keys.
func freshAddr(t *testing.T) types.Address {
	t.Helper()
	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	return kp.Address
}

// signBondHold builds the canonical BondHold and signs it on the Oracle chain id.
func (e *bondTestEnv) signBondHold(t *testing.T, holdID string, amount int64, expiry time.Time) (knirvsigning.BondHold, knirvsigning.SignedMessage) {
	t.Helper()
	hold := knirvsigning.BondHold{
		HoldID:         holdID,
		Operator:       e.operatorKP.Address.String(),
		Amount:         big.NewInt(amount).String(),
		RootRegistryID: e.rootRegistryID,
		ExpiresAtUnix:  uint64(expiry.Unix()),
	}
	payload, _ := json.Marshal(hold)
	signed, err := knirvsigning.SignMessage(ethcrypto.FromECDSA(e.operatorKP.PrivateKey), knirvsigning.MessageEnvelope{
		SchemaVersion: knirvsigning.MessageSchemaVersion,
		Domain:        knirvsigning.RegistryDomain,
		Purpose:       knirvsigning.PurposeBondHold,
		ChainID:       e.nrn.ChainID(),
		Nonce:         "bond-hold:" + holdID,
		IssuedAtUnix:  time.Now().Unix(),
		ExpiresAtUnix: expiry.Unix(),
		Payload:       payload,
	})
	if err != nil {
		t.Fatalf("sign hold: %v", err)
	}
	return hold, signed
}

// signBondRelease builds the canonical BondRelease signed by the root owner.
func (e *bondTestEnv) signBondRelease(t *testing.T, holdID, bootChainID string, bootAddr types.Address) (knirvsigning.BondRelease, knirvsigning.SignedMessage) {
	t.Helper()
	rel := knirvsigning.BondRelease{
		HoldID:          holdID,
		BootNodeChainID: bootChainID,
		BootNodeAddress: bootAddr.String(),
	}
	payload, _ := json.Marshal(rel)
	signed, err := knirvsigning.SignMessage(ethcrypto.FromECDSA(e.ownerKP.PrivateKey), knirvsigning.MessageEnvelope{
		SchemaVersion: knirvsigning.MessageSchemaVersion,
		Domain:        knirvsigning.RegistryDomain,
		Purpose:       knirvsigning.PurposeBondRelease,
		ChainID:       e.nrn.ChainID(),
		Nonce:         "bond-release:" + holdID,
		IssuedAtUnix:  time.Now().Unix(),
		ExpiresAtUnix: time.Now().Add(time.Hour).Unix(),
		Payload:       payload,
	})
	if err != nil {
		t.Fatalf("sign release: %v", err)
	}
	return rel, signed
}

// signBondReleaseAs signs a release with an arbitrary key (for the unauthorized case).
func (e *bondTestEnv) signBondReleaseAs(t *testing.T, kp *crypto.KeyPair, holdID, bootChainID string, bootAddr types.Address) knirvsigning.SignedMessage {
	t.Helper()
	rel := knirvsigning.BondRelease{HoldID: holdID, BootNodeChainID: bootChainID, BootNodeAddress: bootAddr.String()}
	payload, _ := json.Marshal(rel)
	signed, err := knirvsigning.SignMessage(ethcrypto.FromECDSA(kp.PrivateKey), knirvsigning.MessageEnvelope{
		SchemaVersion: knirvsigning.MessageSchemaVersion,
		Domain:        knirvsigning.RegistryDomain,
		Purpose:       knirvsigning.PurposeBondRelease,
		ChainID:       e.nrn.ChainID(),
		Nonce:         "bond-release:" + holdID,
		IssuedAtUnix:  time.Now().Unix(),
		ExpiresAtUnix: time.Now().Add(time.Hour).Unix(),
		Payload:       payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func TestBondHoldThenRelease(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-1"
	expiry := time.Now().Add(time.Hour)
	amount := int64(10_000)
	hold, signed := env.signBondHold(t, holdID, amount, expiry)

	if err := env.nrn.Hold(holdID, hold, signed, time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	escrow := bondEscrowAddress(holdID)
	if got := env.nrn.GetBalance(escrow).Int64(); got != amount {
		t.Fatalf("escrow balance = %d, want %d", got, amount)
	}
	if got := env.nrn.GetBalance(env.operatorKP.Address).Int64(); got != 90_000 {
		t.Fatalf("operator balance = %d, want 90000", got)
	}

	bootAddr := freshAddr(t)
	rel, relSigned := env.signBondRelease(t, holdID, "boot-chain-9", bootAddr)
	if err := env.nrn.Release(holdID, rel, relSigned, time.Now()); err != nil {
		t.Fatalf("release: %v", err)
	}
	if got := env.nrn.GetBalance(escrow).Int64(); got != 0 {
		t.Fatalf("escrow balance after release = %d, want 0", got)
	}
	if got := env.nrn.GetBalance(bootAddr).Int64(); got != amount {
		t.Fatalf("boot node balance = %d, want %d", got, amount)
	}
}

func TestBondHoldIdempotent(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-idem"
	hold, signed := env.signBondHold(t, holdID, 5_000, time.Now().Add(time.Hour))
	if err := env.nrn.Hold(holdID, hold, signed, time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	if err := env.nrn.Hold(holdID, hold, signed, time.Now()); err != nil {
		t.Fatalf("replay hold: %v", err)
	}
	if got := env.nrn.GetBalance(bondEscrowAddress(holdID)).Int64(); got != 5_000 {
		t.Fatalf("escrow balance = %d, want 5000", got)
	}
	if got := env.nrn.GetBalance(env.operatorKP.Address).Int64(); got != 95_000 {
		t.Fatalf("operator balance = %d, want 95000", got)
	}
}

func TestBondReleaseIdempotentSameBoot(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-relidem"
	hold, signed := env.signBondHold(t, holdID, 5_000, time.Now().Add(time.Hour))
	_ = env.nrn.Hold(holdID, hold, signed, time.Now())
	bootAddr := freshAddr(t)
	rel, relSigned := env.signBondRelease(t, holdID, "boot-chain-9", bootAddr)
	if err := env.nrn.Release(holdID, rel, relSigned, time.Now()); err != nil {
		t.Fatalf("release: %v", err)
	}
	if err := env.nrn.Release(holdID, rel, relSigned, time.Now()); err != nil {
		t.Fatalf("replay release: %v", err)
	}
	if got := env.nrn.GetBalance(bootAddr).Int64(); got != 5_000 {
		t.Fatalf("boot node balance = %d, want 5000", got)
	}
}

func TestBondReleaseUnauthorized(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-unauth"
	hold, signed := env.signBondHold(t, holdID, 5_000, time.Now().Add(time.Hour))
	_ = env.nrn.Hold(holdID, hold, signed, time.Now())
	bootAddr := freshAddr(t)
	// Signed by the operator, not the root owner.
	rel := knirvsigning.BondRelease{HoldID: holdID, BootNodeChainID: "boot-chain-9", BootNodeAddress: bootAddr.String()}
	bogus := env.signBondReleaseAs(t, env.operatorKP, holdID, "boot-chain-9", bootAddr)
	_ = rel
	if err := env.nrn.Release(holdID, rel, bogus, time.Now()); err == nil {
		t.Fatal("expected release by non-owner to fail")
	}
}

func TestBondReleaseMismatchedBootAfterRelease(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-mismatch"
	hold, signed := env.signBondHold(t, holdID, 5_000, time.Now().Add(time.Hour))
	_ = env.nrn.Hold(holdID, hold, signed, time.Now())
	bootA := freshAddr(t)
	relA, relSignedA := env.signBondRelease(t, holdID, "boot-chain-9", bootA)
	if err := env.nrn.Release(holdID, relA, relSignedA, time.Now()); err != nil {
		t.Fatalf("first release: %v", err)
	}
	bootB := freshAddr(t)
	relB, relSignedB := env.signBondRelease(t, holdID, "boot-chain-9", bootB)
	if err := env.nrn.Release(holdID, relB, relSignedB, time.Now()); err == nil {
		t.Fatal("expected mismatched-boot release to fail")
	}
	if got := env.nrn.GetBalance(bootB).Int64(); got != 0 {
		t.Fatalf("bootB wrongly credited: %d", got)
	}
}

func TestBondRefundBeforeExpiryFails(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-refund"
	hold, signed := env.signBondHold(t, holdID, 5_000, time.Now().Add(time.Hour))
	_ = env.nrn.Hold(holdID, hold, signed, time.Now())
	if err := env.nrn.Refund(holdID, time.Now()); err == nil {
		t.Fatal("expected refund before expiry to fail")
	}
}

func TestBondRefundAfterExpiry(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-refund2"
	expiry := time.Now().Add(time.Hour) // valid (future) at sign time
	hold, signed := env.signBondHold(t, holdID, 5_000, expiry)
	_ = env.nrn.Hold(holdID, hold, signed, time.Now())
	if err := env.nrn.Refund(holdID, time.Now().Add(2*time.Hour)); err != nil {
		t.Fatalf("refund: %v", err)
	}
	if got := env.nrn.GetBalance(bondEscrowAddress(holdID)).Int64(); got != 0 {
		t.Fatalf("escrow balance = %d, want 0", got)
	}
	if got := env.nrn.GetBalance(env.operatorKP.Address).Int64(); got != 100_000 {
		t.Fatalf("operator balance = %d, want 100000", got)
	}
}

func TestBondHoldWrongRootRegistryID(t *testing.T) {
	env := newBondTestEnv(t)
	hold := knirvsigning.BondHold{
		HoldID: "hold-wrong", Operator: env.operatorKP.Address.String(),
		Amount: "5000", RootRegistryID: "root-reg-OTHER", ExpiresAtUnix: uint64(time.Now().Add(time.Hour).Unix()),
	}
	payload, _ := json.Marshal(hold)
	signed, err := knirvsigning.SignMessage(ethcrypto.FromECDSA(env.operatorKP.PrivateKey), knirvsigning.MessageEnvelope{
		SchemaVersion: knirvsigning.MessageSchemaVersion,
		Domain:        knirvsigning.RegistryDomain,
		Purpose:       knirvsigning.PurposeBondHold,
		ChainID:       env.nrn.ChainID(),
		Nonce:         "bond-hold:hold-wrong",
		IssuedAtUnix:  time.Now().Unix(),
		ExpiresAtUnix: int64(hold.ExpiresAtUnix),
		Payload:       payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := env.nrn.Hold("hold-wrong", hold, signed, time.Now()); err == nil {
		t.Fatal("expected hold with wrong root registry id to fail")
	}
}

func TestBondHoldOperatorMismatch(t *testing.T) {
	env := newBondTestEnv(t)
	// Operator in the payload says the owner, but the signature is from operator.
	hold := knirvsigning.BondHold{
		HoldID: "hold-opmismatch", Operator: env.ownerKP.Address.String(),
		Amount: "5000", RootRegistryID: env.rootRegistryID, ExpiresAtUnix: uint64(time.Now().Add(time.Hour).Unix()),
	}
	payload, _ := json.Marshal(hold)
	signed, err := knirvsigning.SignMessage(ethcrypto.FromECDSA(env.operatorKP.PrivateKey), knirvsigning.MessageEnvelope{
		SchemaVersion: knirvsigning.MessageSchemaVersion,
		Domain:        knirvsigning.RegistryDomain,
		Purpose:       knirvsigning.PurposeBondHold,
		ChainID:       env.nrn.ChainID(),
		Nonce:         "bond-hold:hold-opmismatch",
		IssuedAtUnix:  time.Now().Unix(),
		ExpiresAtUnix: int64(hold.ExpiresAtUnix),
		Payload:       payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := env.nrn.Hold("hold-opmismatch", hold, signed, time.Now()); err == nil {
		t.Fatal("expected hold where signer != named operator to fail")
	}
}

func TestBondRestartBetweenHoldAndRelease(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-restart"
	amount := int64(7_000)
	hold, signed := env.signBondHold(t, holdID, amount, time.Now().Add(time.Hour))
	_ = env.nrn.Hold(holdID, hold, signed, time.Now())
	// Simulate a restart: prepend a mint of the operator's funds (the live log
	// would carry genesis/mint records) then replay the durable records.
	opMint, _ := json.Marshal(ledger.TokenMint{To: env.operatorKP.Address.String(), Amount: "100000"})
	records := append([]ledger.Record{
		{Seq: 0, Kind: ledger.KindTokenMint, Payload: opMint},
	}, env.recorded...)
	state, err := ledger.Replay(records)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if state.Bonds[holdID].Status != ledger.BondStatusPending {
		t.Fatalf("replayed bond status = %q, want pending", state.Bonds[holdID].Status)
	}
	if state.Balances[bondEscrowAddress(holdID).String()] != "7000" {
		t.Fatalf("replayed escrow balance = %q, want 7000", state.Balances[bondEscrowAddress(holdID).String()])
	}
	state.Owner = env.nrn.Owner().String()
	state.ChainID = env.nrn.ChainID()
	ownerHex := hex.EncodeToString(ethcrypto.FromECDSA(env.ownerKP.PrivateKey))
	restarted, err := NewNRN("NRN", "NRN", big.NewInt(1_000_000_000), big.NewInt(2_000_000_000), ownerHex)
	if err != nil {
		t.Fatal(err)
	}
	restarted.SetRootRegistryID(env.rootRegistryID)
	balances := map[string]*big.Int{}
	for addr, raw := range state.Balances {
		b, ok := new(big.Int).SetString(raw, 10)
		if !ok {
			t.Fatalf("bad balance %q", raw)
		}
		balances[addr] = b
	}
	if err := restarted.Hydrate(state.ChainID, state.Owner, big.NewInt(1_000_000_000), balances, state.Nonces, state.Registered, state.Bonds); err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	if got := restarted.GetBalance(bondEscrowAddress(holdID)).Int64(); got != amount {
		t.Fatalf("restarted escrow balance = %d, want %d", got, amount)
	}
	bootAddr := freshAddr(t)
	rel, relSigned := env.signBondRelease(t, holdID, "boot-chain-9", bootAddr)
	if err := restarted.Release(holdID, rel, relSigned, time.Now()); err != nil {
		t.Fatalf("restarted release: %v", err)
	}
	if got := restarted.GetBalance(bootAddr).Int64(); got != amount {
		t.Fatalf("restarted boot node balance = %d, want %d", got, amount)
	}
}

// --- Audit txchain_p45 P1/P2: atomicity and hold-consumption tests ---

func (e *bondTestEnv) setFailingBatchRecorder() {
	e.nrn.SetRecorder(nil)
	e.nrn.SetRecorderBatch(func(entries []ledger.BatchEntry) error {
		return fmt.Errorf("simulated batch recorder failure")
	})
}

// P1: a recorder failure during Hold must not mutate any balance.
func TestBondHoldRecorderFailureNoMutation(t *testing.T) {
	env := newBondTestEnv(t)
	env.setFailingBatchRecorder()
	holdID := "hold-rec-fail"
	hold, signed := env.signBondHold(t, holdID, 5_000, time.Now().Add(time.Hour))
	if err := env.nrn.Hold(holdID, hold, signed, time.Now()); err == nil {
		t.Fatal("expected hold to fail when the batch recorder fails")
	}
	if got := env.nrn.GetBalance(bondEscrowAddress(holdID)).Int64(); got != 0 {
		t.Fatalf("escrow balance = %d, want 0 (no mutation on recorder failure)", got)
	}
	if got := env.nrn.GetBalance(env.operatorKP.Address).Int64(); got != 100_000 {
		t.Fatalf("operator balance = %d, want 100000 (unchanged)", got)
	}
}

// P1: an underfunded hold must not reach the durable log at all.
func TestBondHoldInsufficientFundsNoDurableWrite(t *testing.T) {
	env := newBondTestEnv(t)
	env.nrn.balances[env.operatorKP.Address] = big.NewInt(0)
	var batchWrites int
	env.nrn.SetRecorder(nil)
	env.nrn.SetRecorderBatch(func(entries []ledger.BatchEntry) error {
		batchWrites++
		return nil
	})
	holdID := "hold-insuff"
	hold, signed := env.signBondHold(t, holdID, 5_000, time.Now().Add(time.Hour))
	if err := env.nrn.Hold(holdID, hold, signed, time.Now()); err == nil {
		t.Fatal("expected underfunded hold to fail")
	}
	if batchWrites != 0 {
		t.Fatalf("underfunded hold wrote %d batches; want 0 (validated before any durable write)", batchWrites)
	}
}

// P1: a recorder failure during Release must leave the escrow untouched.
func TestBondReleaseRecorderFailureNoMutation(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-rel-fail"
	hold, signed := env.signBondHold(t, holdID, 5_000, time.Now().Add(time.Hour))
	if err := env.nrn.Hold(holdID, hold, signed, time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	env.setFailingBatchRecorder()
	bootAddr := freshAddr(t)
	rel, relSigned := env.signBondRelease(t, holdID, "boot-chain-9", bootAddr)
	if err := env.nrn.Release(holdID, rel, relSigned, time.Now()); err == nil {
		t.Fatal("expected release to fail when the batch recorder fails")
	}
	if got := env.nrn.GetBalance(bondEscrowAddress(holdID)).Int64(); got != 5_000 {
		t.Fatalf("escrow balance = %d, want 5000 (release failed, escrow intact)", got)
	}
	if got := env.nrn.GetBalance(bootAddr).Int64(); got != 0 {
		t.Fatalf("boot node wrongly credited: %d", got)
	}
}

// P2: consuming a pending hold releases its escrow to the named address.
// P2: consuming a hold RESERVES it — the escrow stays put and the hold is bound
// to the chain (status consumed). The funds are moved later by the root-owner
// signed Release, not by ConsumeHold.
func TestConsumeHoldReservesEscrow(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-consume"
	hold, signed := env.signBondHold(t, holdID, 5_000, time.Now().Add(time.Hour))
	if err := env.nrn.Hold(holdID, hold, signed, time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	recipient := freshAddr(t)
	if err := env.nrn.ConsumeHold(holdID, "chain-1", recipient); err != nil {
		t.Fatalf("consume: %v", err)
	}
	// Escrow is reserved, not moved: the funds remain in escrow.
	if got := env.nrn.GetBalance(bondEscrowAddress(holdID)).Int64(); got != 5_000 {
		t.Fatalf("escrow balance = %d, want 5000 (reserved, not moved)", got)
	}
	if got := env.nrn.GetBalance(recipient).Int64(); got != 0 {
		t.Fatalf("recipient balance = %d, want 0 (not yet released)", got)
	}
	if st := env.nrn.bondHolds[holdID].Status; st != bondStatusConsumed {
		t.Fatalf("hold status = %q, want %q", st, bondStatusConsumed)
	}
}

// P2: consuming an unknown hold is rejected outright.
func TestConsumeHoldUnknownRejected(t *testing.T) {
	env := newBondTestEnv(t)
	if err := env.nrn.ConsumeHold("ghost", "chain-1", freshAddr(t)); err == nil {
		t.Fatal("expected consume of unknown hold to fail")
	}
}

// P2: a retried consume to the same address never double-sends the escrow.
func TestConsumeHoldIdempotentNoDoubleSend(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-consumed-idem"
	hold, signed := env.signBondHold(t, holdID, 5_000, time.Now().Add(time.Hour))
	if err := env.nrn.Hold(holdID, hold, signed, time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	recipient := freshAddr(t)
	if err := env.nrn.ConsumeHold(holdID, "chain-1", recipient); err != nil {
		t.Fatalf("consume: %v", err)
	}
	if err := env.nrn.ConsumeHold(holdID, "chain-1", recipient); err != nil {
		t.Fatalf("retry consume: %v", err)
	}
	// ConsumeHold only reserves; no funds move, so the recipient stays at zero.
	if got := env.nrn.GetBalance(recipient).Int64(); got != 0 {
		t.Fatalf("recipient balance = %d, want 0 (no send on consume)", got)
	}
	if st := env.nrn.bondHolds[holdID].Status; st != bondStatusConsumed {
		t.Fatalf("hold status = %q, want %q", st, bondStatusConsumed)
	}
}

// P2: an already-consumed hold cannot be redirected to a different address.
func TestConsumeHoldDifferentRecipientRejected(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-consumed-diff"
	hold, signed := env.signBondHold(t, holdID, 5_000, time.Now().Add(time.Hour))
	if err := env.nrn.Hold(holdID, hold, signed, time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	a := freshAddr(t)
	b := freshAddr(t)
	if err := env.nrn.ConsumeHold(holdID, "chain-1", a); err != nil {
		t.Fatalf("consume: %v", err)
	}
	if err := env.nrn.ConsumeHold(holdID, "chain-1", b); err == nil {
		t.Fatal("expected redirected consume to a different address to fail")
	}
	if got := env.nrn.GetBalance(b).Int64(); got != 0 {
		t.Fatalf("second recipient wrongly credited: %d", got)
	}
}

// P2: a crash after a successful hold+consume replays to a reserved (consumed)
// hold with the escrow still intact — never to live funds sitting in escrow with
// no binding, and never to a released hold. The root Release is what later moves
// the escrow.
func TestBondRestartHoldConsume(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-restart-consume"
	amount := int64(7_000)
	hold, signed := env.signBondHold(t, holdID, amount, time.Now().Add(time.Hour))
	_ = env.nrn.Hold(holdID, hold, signed, time.Now())
	recipient := freshAddr(t)
	if err := env.nrn.ConsumeHold(holdID, "chain-1", recipient); err != nil {
		t.Fatalf("consume: %v", err)
	}
	opMint, _ := json.Marshal(ledger.TokenMint{To: env.operatorKP.Address.String(), Amount: "100000"})
	records := append([]ledger.Record{
		{Seq: 0, Kind: ledger.KindTokenMint, Payload: opMint},
	}, env.recorded...)
	state, err := ledger.Replay(records)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if state.Bonds[holdID].Status != ledger.BondStatusConsumed {
		t.Fatalf("replayed bond status = %q, want consumed", state.Bonds[holdID].Status)
	}
	// Escrow remains reserved after replay; nothing has been released yet.
	if state.Balances[bondEscrowAddress(holdID).String()] != fmt.Sprintf("%d", amount) {
		t.Fatalf("replayed escrow balance = %q, want %d", state.Balances[bondEscrowAddress(holdID).String()], amount)
	}
	// The recipient is never credited (ConsumeHold only reserves, and Replay omits
	// zero-balance entries).
	if v := state.Balances[recipient.String()]; v != "" && v != "0" {
		t.Fatalf("replayed recipient balance = %q, want 0", v)
	}
}

// TestConsumeHoldIdempotentSameChain proves the round-4 binding: once a hold is
// consumed by chain "chain-X", a retry (same chain id, same destination) is a no-op
// and a different chain "chain-Y" is rejected — one enrolment hold cannot authorise
// two chains.
func TestConsumeHoldIdempotentSameChain(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-idem"
	hold, signed := env.signBondHold(t, holdID, 5_000, time.Now().Add(time.Hour))
	if err := env.nrn.Hold(holdID, hold, signed, time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	recipient := freshAddr(t)
	if err := env.nrn.ConsumeHold(holdID, "chain-X", recipient); err != nil {
		t.Fatalf("consume: %v", err)
	}
	// Same chain + same destination: idempotent no-op (no double send).
	if err := env.nrn.ConsumeHold(holdID, "chain-X", recipient); err != nil {
		t.Fatalf("retry consume: %v", err)
	}
	// ConsumeHold only reserves, so the recipient is never credited here.
	if got := env.nrn.GetBalance(recipient).Int64(); got != 0 {
		t.Fatalf("recipient = %d, want 0 (no send on consume)", got)
	}
	if st := env.nrn.bondHolds[holdID].Status; st != bondStatusConsumed {
		t.Fatalf("hold status = %q, want %q", st, bondStatusConsumed)
	}
}

// TestConsumeHoldRejectsDifferentChain proves a second, distinct chain cannot spend
// the same hold after it has been bound to the first chain.
func TestConsumeHoldRejectsDifferentChain(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-diff"
	hold, signed := env.signBondHold(t, holdID, 5_000, time.Now().Add(time.Hour))
	if err := env.nrn.Hold(holdID, hold, signed, time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	recipient := freshAddr(t)
	if err := env.nrn.ConsumeHold(holdID, "chain-X", recipient); err != nil {
		t.Fatalf("consume: %v", err)
	}
	// A different chain id must be rejected even with the same destination.
	if err := env.nrn.ConsumeHold(holdID, "chain-Y", recipient); !errors.Is(err, ErrHoldAlreadyConsumed) {
		t.Fatalf("expected ErrHoldAlreadyConsumed, got %v", err)
	}
	// ConsumeHold only reserves, so the recipient is never credited here.
	if got := env.nrn.GetBalance(recipient).Int64(); got != 0 {
		t.Fatalf("recipient = %d, want 0 (no second spend)", got)
	}
}

// P2: the full D14 sequence — hold, reserve via ConsumeHold, then move the
// escrow to the boot node address via the root-owner-signed Release. ConsumeHold
// only reserves; Release is what transfers.
func TestConsumeThenReleaseMovesEscrow(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-consume-release"
	amount := int64(7_000)
	hold, signed := env.signBondHold(t, holdID, amount, time.Now().Add(time.Hour))
	if err := env.nrn.Hold(holdID, hold, signed, time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	boot := freshAddr(t)
	if err := env.nrn.ConsumeHold(holdID, "chain-X", boot); err != nil {
		t.Fatalf("consume: %v", err)
	}
	// After reserve the escrow is intact.
	if got := env.nrn.GetBalance(bondEscrowAddress(holdID)).Int64(); got != amount {
		t.Fatalf("escrow after consume = %d, want %d", got, amount)
	}
	// Root-owner-signed release moves the escrow to the boot node address.
	rel, relSigned := env.signBondRelease(t, holdID, "chain-X", boot)
	if err := env.nrn.Release(holdID, rel, relSigned, time.Now()); err != nil {
		t.Fatalf("release: %v", err)
	}
	if got := env.nrn.GetBalance(bondEscrowAddress(holdID)).Int64(); got != 0 {
		t.Fatalf("escrow after release = %d, want 0", got)
	}
	if got := env.nrn.GetBalance(boot).Int64(); got != amount {
		t.Fatalf("boot node balance = %d, want %d", got, amount)
	}
	// A second release to a different address is rejected.
	other := freshAddr(t)
	rel2 := knirvsigning.BondRelease{HoldID: holdID, BootNodeChainID: "chain-X", BootNodeAddress: other.String()}
	if err := env.nrn.Release(holdID, rel2, env.signBondReleaseAs(t, env.ownerKP, holdID, "chain-X", other), time.Now()); err == nil {
		t.Fatal("expected release to a different address to be rejected")
	}
}

// TestReleaseRejectsWrongChainIDAgainstConsumedHold enforces the chain-id binding
// added for the D14 audit: once a hold is consumed (reserved) to a chain, a
// root-owner-signed release naming a DIFFERENT boot-node chain id must be rejected
// and leave the hold/escrow untouched, so a forgery cannot redirect the escrow.
func TestReleaseRejectsWrongChainIDAgainstConsumedHold(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-chain-binding"
	amount := int64(10_000)
	hold, signed := env.signBondHold(t, holdID, amount, time.Now().Add(time.Hour))
	if err := env.nrn.Hold(holdID, hold, signed, time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	boot := freshAddr(t)
	// Reserve (consume) the hold on chain-X.
	if err := env.nrn.ConsumeHold(holdID, "chain-X", boot); err != nil {
		t.Fatalf("consume: %v", err)
	}
	// A validly signed release carrying a different boot-node chain id is
	// rejected even though the signature and owner are correct.
	wrong, wrongSigned := env.signBondRelease(t, holdID, "chain-Y", boot)
	if err := env.nrn.Release(holdID, wrong, wrongSigned, time.Now()); err == nil {
		t.Fatal("expected release with wrong boot-node chain id against a consumed hold to be rejected")
	}
	// The hold/escrow are unchanged: still consumed, escrow intact.
	h, err := env.nrn.GetHold(holdID)
	if err != nil {
		t.Fatalf("get hold: %v", err)
	}
	if h.Status != ledger.BondStatusConsumed {
		t.Fatalf("hold status = %q, want consumed", h.Status)
	}
	if got := env.nrn.GetBalance(bondEscrowAddress(holdID)).Int64(); got != amount {
		t.Fatalf("escrow after rejected release = %d, want %d (unchanged)", got, amount)
	}
	// The correct chain id releases normally.
	rel, relSigned := env.signBondRelease(t, holdID, "chain-X", boot)
	if err := env.nrn.Release(holdID, rel, relSigned, time.Now()); err != nil {
		t.Fatalf("release with correct chain id: %v", err)
	}
	if got := env.nrn.GetBalance(bondEscrowAddress(holdID)).Int64(); got != 0 {
		t.Fatalf("escrow after release = %d, want 0", got)
	}
	if got := env.nrn.GetBalance(boot).Int64(); got != amount {
		t.Fatalf("boot node balance = %d, want %d", got, amount)
	}
}

// TestConsumeHoldReplayBinding proves the hold->chain binding survives a restart:
// after the hold/consume batch is replayed into a fresh token, the hold is still
// bound to the first chain, a second chain is still rejected, and the same-chain
// retry still succeeds idempotently.
func TestConsumeHoldReplayBinding(t *testing.T) {
	env := newBondTestEnv(t)
	holdID := "hold-replay"
	amount := int64(7_000)
	hold, signed := env.signBondHold(t, holdID, amount, time.Now().Add(time.Hour))
	if err := env.nrn.Hold(holdID, hold, signed, time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	recipient := freshAddr(t)
	if err := env.nrn.ConsumeHold(holdID, "chain-X", recipient); err != nil {
		t.Fatalf("consume: %v", err)
	}
	// Replay the durable log into a fresh state. The operator's initial funding is
	// applied directly (not recorded) by the env, so prepend it like a real mint.
	opMint, _ := json.Marshal(ledger.TokenMint{To: env.operatorKP.Address.String(), Amount: "100000"})
	records := append([]ledger.Record{
		{Seq: 0, Kind: ledger.KindTokenMint, Payload: opMint},
	}, env.recorded...)
	state, err := ledger.Replay(records)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if state.Bonds[holdID].ConsumedChainID != "chain-X" {
		t.Fatalf("replayed hold bound to %q, want chain-X", state.Bonds[holdID].ConsumedChainID)
	}
	// Hydrate a brand-new token from the replayed state.
	ownerHex := hex.EncodeToString(ethcrypto.FromECDSA(env.ownerKP.PrivateKey))
	fresh, err := NewNRN("NRN", "NRN", big.NewInt(1_000_000_000), big.NewInt(2_000_000_000), ownerHex)
	if err != nil {
		t.Fatalf("new nrn: %v", err)
	}
	fresh.SetRootRegistryID(testRootRegistryID)
	balances := map[string]*big.Int{}
	for addr, s := range state.Balances {
		balances[addr] = new(big.Int)
		balances[addr].SetString(s, 10)
	}
	if err := fresh.Hydrate(env.nrn.ChainID(), env.ownerKP.Address.String(), env.nrn.TotalSupply(), balances, state.Nonces, state.Registered, state.Bonds); err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	// The replayed token still rejects a different chain and still accepts the
	// idempotent same-chain retry.
	if err := fresh.ConsumeHold(holdID, "chain-Y", recipient); !errors.Is(err, ErrHoldAlreadyConsumed) {
		t.Fatalf("expected ErrHoldAlreadyConsumed after replay, got %v", err)
	}
	if err := fresh.ConsumeHold(holdID, "chain-X", recipient); err != nil {
		t.Fatalf("idempotent replay retry: %v", err)
	}
}

// signBondHoldWith signs a bond-hold with an arbitrary operator key (for concurrent
// holds by distinct operators).
func (e *bondTestEnv) signBondHoldWith(t *testing.T, kp *crypto.KeyPair, holdID string, amount int64, expiry time.Time) (knirvsigning.BondHold, knirvsigning.SignedMessage) {
	t.Helper()
	hold := knirvsigning.BondHold{
		HoldID:         holdID,
		Operator:       kp.Address.String(),
		Amount:         big.NewInt(amount).String(),
		RootRegistryID: e.rootRegistryID,
		ExpiresAtUnix:  uint64(expiry.Unix()),
	}
	payload, _ := json.Marshal(hold)
	signed, err := knirvsigning.SignMessage(ethcrypto.FromECDSA(kp.PrivateKey), knirvsigning.MessageEnvelope{
		SchemaVersion: knirvsigning.MessageSchemaVersion,
		Domain:        knirvsigning.RegistryDomain,
		Purpose:       knirvsigning.PurposeBondHold,
		ChainID:       e.nrn.ChainID(),
		Nonce:         "bond-hold:" + holdID,
		IssuedAtUnix:  time.Now().Unix(),
		ExpiresAtUnix: expiry.Unix(),
		Payload:       payload,
	})
	if err != nil {
		t.Fatalf("sign hold: %v", err)
	}
	return hold, signed
}

// signBondReleaseWith signs a bond-release with the root owner key.
func (e *bondTestEnv) signBondReleaseWith(t *testing.T, holdID string, bootAddr types.Address) (knirvsigning.BondRelease, knirvsigning.SignedMessage) {
	t.Helper()
	rel := knirvsigning.BondRelease{HoldID: holdID, BootNodeChainID: "boot", BootNodeAddress: bootAddr.String()}
	payload, _ := json.Marshal(rel)
	signed, err := knirvsigning.SignMessage(ethcrypto.FromECDSA(e.ownerKP.PrivateKey), knirvsigning.MessageEnvelope{
		SchemaVersion: knirvsigning.MessageSchemaVersion,
		Domain:        knirvsigning.RegistryDomain,
		Purpose:       knirvsigning.PurposeBondRelease,
		ChainID:       e.nrn.ChainID(),
		Nonce:         "bond-release:" + holdID,
		IssuedAtUnix:  time.Now().Unix(),
		ExpiresAtUnix: time.Now().Add(time.Hour).Unix(),
		Payload:       payload,
	})
	if err != nil {
		t.Fatalf("sign release: %v", err)
	}
	return rel, signed
}

// P1#1: concurrent Hold/Release/Refund must be safe under -race. The token
// serializes all ledger mutations behind a single lock, so a heavy mix of
// operations on distinct holds must never race or corrupt state.
func TestBondConcurrentHoldReleaseRefund(t *testing.T) {
	env := newBondTestEnv(t)
	// Stateless recorder so concurrent recordBatch calls don't race on shared state.
	env.nrn.SetRecorderBatch(func(entries []ledger.BatchEntry) error { return nil })

	const n = 32
	type actor struct {
		holdID  string
		op      *crypto.KeyPair
		expiry  time.Time
	}
	// Sequential setup: distinct funded operators (no concurrent map writes).
	live := make([]actor, n)
	for i := 0; i < n; i++ {
		kp, err := crypto.GenerateKeyPair()
		if err != nil {
			t.Fatal(err)
		}
		live[i] = actor{holdID: fmt.Sprintf("hold-%d", i), op: kp, expiry: time.Now().Add(time.Hour)}
		env.nrn.balances[kp.Address] = big.NewInt(100_000)
	}

	var wg sync.WaitGroup
	// Concurrent Hold.
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(a actor) {
			defer wg.Done()
			hold, sig := env.signBondHoldWith(t, a.op, a.holdID, 100, a.expiry)
			if err := env.nrn.Hold(a.holdID, hold, sig, time.Now()); err != nil {
				t.Errorf("hold %s: %v", a.holdID, err)
			}
		}(live[i])
	}
	wg.Wait()

	// Concurrent Release (owner-signed) to distinct boot nodes.
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(a actor) {
			defer wg.Done()
			boot := freshAddr(t)
			rel, sig := env.signBondReleaseWith(t, a.holdID, boot)
			if err := env.nrn.Release(a.holdID, rel, sig, time.Now()); err != nil {
				t.Errorf("release %s: %v", a.holdID, err)
			}
		}(live[i])
	}
	wg.Wait()

	// Concurrent Hold + Refund on a second batch of holds we then refund after
	// their (valid, future-at-sign-time) expiry.
	expired := make([]actor, n)
	for i := 0; i < n; i++ {
		kp, err := crypto.GenerateKeyPair()
		if err != nil {
			t.Fatal(err)
		}
		expired[i] = actor{holdID: fmt.Sprintf("exp-%d", i), op: kp, expiry: time.Now().Add(time.Hour)}
		env.nrn.balances[kp.Address] = big.NewInt(100_000)
	}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(a actor) {
			defer wg.Done()
			hold, sig := env.signBondHoldWith(t, a.op, a.holdID, 100, a.expiry)
			_ = env.nrn.Hold(a.holdID, hold, sig, time.Now())
			if err := env.nrn.Refund(a.holdID, time.Now().Add(2*time.Hour)); err != nil {
				t.Errorf("refund %s: %v", a.holdID, err)
			}
		}(expired[i])
	}
	wg.Wait()
}
