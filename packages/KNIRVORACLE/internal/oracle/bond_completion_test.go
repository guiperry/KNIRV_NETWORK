package oracle

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/ledger"
	"github.com/knirvcorp/knirvoracle/internal/oracle/registry"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// bondEscrowAddr mirrors the token package's deterministic escrow derivation so
// the e2e test can assert where the held funds sit.
func bondEscrowAddr(holdID string) types.Address {
	sum := sha256.Sum256([]byte("knirv.oracle.bond.hold:" + holdID))
	addr, _ := types.AddressFromBytes(sum[:20])
	return addr
}

// fundOperator mints and stakes enough NRN for a D14 authority enrolment.
func fundOperator(t *testing.T, o *Oracle, opKey *crypto.KeyPair) types.Address {
	t.Helper()
	tok := o.GetNRNToken()
	operator := opKey.Address
	if _, err := tok.Mint(operator, big.NewInt(2_000_000_000)); err != nil {
		t.Fatalf("mint: %v", err)
	}
	if _, err := o.economicsEngine.GetStakingManager().Stake(operator, big.NewInt(1_000_000_000)); err != nil {
		t.Fatalf("stake: %v", err)
	}
	return operator
}

// d14Enrolment builds a controller-signed bond hold and a complete authority
// registration that consumes it, exactly as the production CLI flow does.
func d14Enrolment(t *testing.T, o *Oracle, opKey *crypto.KeyPair, chainSuffix string) (types.Address, types.ChainRegistration, string) {
	t.Helper()
	tok := o.GetNRNToken()
	operator := opKey.Address
	rootRegID := tok.RootRegistryID()
	if rootRegID == "" {
		t.Fatal("root registry id not set at init")
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
	reg, regKey := newTestChain(t, registry.AuthorityChainPrefix+chainSuffix)
	reg.Bond = 10_000
	reg.BondOwner = operator.String()
	reg.HoldID = holdID
	if err := types.SignRegistration(&reg, regKey); err != nil {
		t.Fatal(err)
	}
	return operator, reg, holdID
}

// P1/D14 main flow: hold -> register reserves the hold -> the root-owner-signed
// Release (server-side supervisor) moves the escrow to the boot node address, so
// balances end correct with no partial registration.
func TestD14CompletionMainFlow(t *testing.T) {
	o, opKey := newBondOracle(t)
	operator := fundOperator(t, o, opKey)
	_, reg, holdID := d14Enrolment(t, o, opKey, "knirv1main")

	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("register: %v", err)
	}
	tok := o.GetNRNToken()
	// After registration the hold is RESERVED: escrow intact, operator liquid is
	// the staked remainder minus the held amount.
	if got := tok.GetBalance(operator).Int64(); got != 1_000_000_000-10_000 {
		t.Fatalf("operator after register = %d, want 999990000 (escrow still held)", got)
	}
	if got := tok.GetBalance(bondEscrowAddr(holdID)).Int64(); got != 10_000 {
		t.Fatalf("escrow after register = %d, want 10000 (reserved)", got)
	}
	h, err := tok.GetHold(holdID)
	if err != nil {
		t.Fatalf("get hold: %v", err)
	}
	if h.Status != ledger.BondStatusConsumed {
		t.Fatalf("hold status = %q, want consumed", h.Status)
	}

	// Server-side completion: the root owner releases the reserved hold.
	if err := o.CompleteBondReleases(context.Background()); err != nil {
		t.Fatalf("complete bond releases: %v", err)
	}

	// The escrow moved to the boot node address (the operator, per the
	// enrolment), so the operator is made whole again.
	if got := tok.GetBalance(operator).Int64(); got != 1_000_000_000 {
		t.Fatalf("operator after release = %d, want 1000000000 (escrow released)", got)
	}
	if got := tok.GetBalance(bondEscrowAddr(holdID)).Int64(); got != 0 {
		t.Fatalf("escrow after release = %d, want 0", got)
	}
	h, err = tok.GetHold(holdID)
	if err != nil {
		t.Fatalf("get hold: %v", err)
	}
	if h.Status != ledger.BondStatusReleased {
		t.Fatalf("hold status = %q, want released", h.Status)
	}
}

// P1/D14 idempotency: a registration retried after a lost response never
// double-sends the escrow, and the completion supervisor is a no-op once released.
func TestD14CompletionIdempotentRetry(t *testing.T) {
	o, opKey := newBondOracle(t)
	operator := fundOperator(t, o, opKey)
	_, reg, _ := d14Enrolment(t, o, opKey, "knirv1retry")

	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("register: %v", err)
	}
	// Response lost: the CLI retries the same registration.
	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("retry register: %v", err)
	}
	tok := o.GetNRNToken()
	if got := tok.GetBalance(operator).Int64(); got != 1_000_000_000-10_000 {
		t.Fatalf("operator after retries = %d, want 999990000 (no double send)", got)
	}
	if err := o.CompleteBondReleases(context.Background()); err != nil {
		t.Fatalf("complete: %v", err)
	}
	// Running completion again is a no-op.
	if err := o.CompleteBondReleases(context.Background()); err != nil {
		t.Fatalf("complete again: %v", err)
	}
	if got := tok.GetBalance(operator).Int64(); got != 1_000_000_000 {
		t.Fatalf("operator after release = %d, want 1000000000 (no double send)", got)
	}
}

// P1/D14 one hold cannot authorise two chains: a second registration that names
// the same hold is rejected even after the first consumed it.
func TestD14CompletionSecondChainRejected(t *testing.T) {
	o, opKey := newBondOracle(t)
	operator := fundOperator(t, o, opKey)
	_, reg, holdID := d14Enrolment(t, o, opKey, "knirv1first")

	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("first register: %v", err)
	}
	// A second, distinct authority chain tries to consume the same hold.
	reg2, reg2Key := newTestChain(t, registry.AuthorityChainPrefix+"knirv1second")
	reg2.Bond = 10_000
	reg2.BondOwner = operator.String()
	reg2.HoldID = holdID
	if err := types.SignRegistration(&reg2, reg2Key); err != nil {
		t.Fatal(err)
	}
	if err := o.RegisterChain(&reg2); err == nil {
		t.Fatal("expected second registration on the same hold to be rejected")
	}
	if _, ok := o.GetChainRegistration(reg2.ChainID); ok {
		t.Fatal("second chain was registered despite the shared hold")
	}
}

// P1/D14 the escrow can only be released to the registered boot node address: a
// release naming a different address, submitted BEFORE any valid release, is
// rejected and leaves the hold + escrow untouched.
func TestD14CompletionReleaseToWrongAddressRejected(t *testing.T) {
	o, opKey := newBondOracle(t)
	operator := fundOperator(t, o, opKey)
	_, reg, holdID := d14Enrolment(t, o, opKey, "knirv1wrong")
	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("register: %v", err)
	}
	tok := o.GetNRNToken()

	// Submit a release naming a DIFFERENT address before any valid release. It
	// must be rejected and must leave the hold/escrow unchanged.
	other, _ := crypto.GenerateKeyPair()
	wrong := knirvsigning.BondRelease{HoldID: holdID, BootNodeChainID: reg.ChainID, BootNodeAddress: other.Address.String()}
	wrongSigned, err := signReleaseWithOwner(t, o, wrong)
	if err != nil {
		t.Fatalf("sign wrong release: %v", err)
	}
	if err := tok.Release(holdID, wrong, wrongSigned, time.Now()); err == nil {
		t.Fatal("expected release to a different address to be rejected")
	}
	h, err := tok.GetHold(holdID)
	if err != nil {
		t.Fatalf("get hold: %v", err)
	}
	if h.Status != ledger.BondStatusConsumed {
		t.Fatalf("hold status = %q, want consumed (unchanged)", h.Status)
	}
	if got := tok.GetBalance(bondEscrowAddr(holdID)).Int64(); got != 10_000 {
		t.Fatalf("escrow after wrong release = %d, want 10000 (unchanged)", got)
	}
	if got := tok.GetBalance(operator).Int64(); got != 1_000_000_000-10_000 {
		t.Fatalf("operator after wrong release = %d, want 999990000 (unchanged)", got)
	}

	// The legitimate supervisor release still succeeds afterwards.
	if err := o.CompleteBondReleases(context.Background()); err != nil {
		t.Fatalf("complete: %v", err)
	}
	h, err = tok.GetHold(holdID)
	if err != nil {
		t.Fatalf("get hold after complete: %v", err)
	}
	if h.Status != ledger.BondStatusReleased {
		t.Fatalf("hold status after complete = %q, want released", h.Status)
	}
}

// P1/D14 restart safety: a hold consumed by registration immediately before a
// crash survives the ledger replay, and once the Oracle comes back up the
// root-owner Release moves the escrow exactly once (the supervisor's job after a
// restart).
func TestD14CompletionRestartBetweenConsumeAndRelease(t *testing.T) {
	dir := t.TempDir()
	o, opKey := newBondOracleInDir(t, dir)
	operator := fundOperator(t, o, opKey)
	_, reg, holdID := d14Enrolment(t, o, opKey, "knirv1restart")

	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("register: %v", err)
	}
	tok := o.GetNRNToken()
	// The hold is reserved (escrow intact) but NOT yet released.
	if got := tok.GetBalance(bondEscrowAddr(holdID)).Int64(); got != 10_000 {
		t.Fatalf("escrow before restart = %d, want 10000", got)
	}

	// Simulate a crash + restart from the same data dir: close the first Oracle's
	// ledger, then bring a fresh Oracle up against the same dir. The restarted
	// Oracle must use the SAME root-owner key (as a real restart from root.key
	// would) so it can authorise the post-restart release.
	if err := o.ledger.Close(); err != nil {
		t.Fatalf("close first ledger: %v", err)
	}
	o2, _ := newBondOracleInDirWithOwner(t, dir, o.config.OwnerPrivateKey)
	tok2 := o2.GetNRNToken()
	// The consumed hold + escrow survived the replay.
	h2, err := tok2.GetHold(holdID)
	if err != nil {
		t.Fatalf("get hold after restart: %v", err)
	}
	if h2.Status != ledger.BondStatusConsumed {
		t.Fatalf("hold status after restart = %q, want consumed", h2.Status)
	}
	if got := tok2.GetBalance(bondEscrowAddr(holdID)).Int64(); got != 10_000 {
		t.Fatalf("escrow after restart = %d, want 10000 (survives replay)", got)
	}
	// The post-restart supervisor (the only release trigger) completes the
	// reserved hold exactly once, moving the escrow to the boot node. No manual
	// token.Release is used — this proves the restart-safety guarantee.
	if err := o2.CompleteBondReleases(context.Background()); err != nil {
		t.Fatalf("complete bond releases after restart: %v", err)
	}
	if got := tok2.GetBalance(bondEscrowAddr(holdID)).Int64(); got != 0 {
		t.Fatalf("escrow after restart release = %d, want 0", got)
	}
	if got := tok2.GetBalance(operator).Int64(); got != 1_000_000_000 {
		t.Fatalf("operator after restart release = %d, want 1000000000", got)
	}
}

// P1/D14 the safety net: an enrolment hold whose registration never completes is
// refunded once it expires, so escrow is never stranded ("no partial registration").
func TestD14CompletionRefundAfterExpiry(t *testing.T) {
	o, opKey := newBondOracle(t)
	operator := fundOperator(t, o, opKey)
	tok := o.GetNRNToken()

	// A hold with a short (but valid) expiry, deliberately never registered.
	holdID := "bond:" + operator.String() + ":abandoned"
	expiry := time.Now().Add(5 * time.Second)
	hold := knirvsigning.BondHold{
		HoldID:         holdID,
		Operator:       operator.String(),
		Amount:         "10000",
		RootRegistryID: tok.RootRegistryID(),
		ExpiresAtUnix:  uint64(expiry.Unix()),
	}
	if err := tok.Hold(holdID, hold, signBondHoldForTest(t, opKey, tok.ChainID(), hold), time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	before := tok.GetBalance(operator).Int64() // 1B - 10k
	if before != 1_000_000_000-10_000 {
		t.Fatalf("operator before expiry = %d, want 999990000", before)
	}

	// Wait comfortably past the expiry (accounting for second truncation), then
	// run the refund sweep.
	time.Sleep(6200 * time.Millisecond)
	if err := o.SweepExpiredBondRefunds(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	// The escrow is returned to the operator.
	if got := tok.GetBalance(operator).Int64(); got != 1_000_000_000 {
		t.Fatalf("operator after refund = %d, want 1000000000 (escrow returned)", got)
	}
	if got := tok.GetBalance(bondEscrowAddr(holdID)).Int64(); got != 0 {
		t.Fatalf("escrow after refund = %d, want 0", got)
	}
}

// P1/D14 the bond supervisor runs on the Oracle's lifecycle context (audit fix):
// Stop() cancels it and ends the background loop, and the supervisor goroutine
// actually exits (no leaked goroutine after shutdown).
func TestBondSupervisorStopsWithOracleLifecycle(t *testing.T) {
	o, _ := newBondOracle(t)
	o.startBondSupervisor(o.ctx)

	// Stop must cancel the lifecycle context the supervisor selects on.
	if err := o.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	select {
	case <-o.ctx.Done():
		// The supervisor's `select { case <-ctx.Done(): return }` now returns
		// and the loop ends.
	case <-time.After(2 * time.Second):
		t.Fatal("oracle lifecycle context was not cancelled by Stop(); supervisor loop would leak")
	}
	// The supervisor goroutine must actually terminate, not merely observe the
	// cancelled context. Wait on the done-channel seam with a timeout.
	select {
	case <-o.supervisorDone:
		// goroutine exited cleanly
	case <-time.After(2 * time.Second):
		t.Fatal("bond supervisor goroutine did not exit after Stop(); leaked goroutine")
	}
}

// signReleaseWithOwner signs a BondRelease with the Oracle's root owner key.
func signReleaseWithOwner(t *testing.T, o *Oracle, rel knirvsigning.BondRelease) (knirvsigning.SignedMessage, error) {
	t.Helper()
	payload, _ := json.Marshal(rel)
	return o.replication.signer.SignMessage(context.Background(), knirvsigning.MessageEnvelope{
		SchemaVersion: knirvsigning.MessageSchemaVersion,
		Domain:        knirvsigning.RegistryDomain,
		Purpose:       knirvsigning.PurposeBondRelease,
		ChainID:       o.GetNRNToken().ChainID(),
		Nonce:         "bond-release:" + rel.HoldID,
		IssuedAtUnix:  time.Now().Unix(),
		ExpiresAtUnix: time.Now().Add(time.Hour).Unix(),
		Payload:       payload,
	})
}

// newBondOracleInDir builds a real Oracle rooted at dir (so it can be restarted
// from the same data dir), with the SDK core available.
func newBondOracleInDir(t *testing.T, dir string) (*Oracle, *crypto.KeyPair) {
	t.Helper()
	return newBondOracleInDirWithOwner(t, dir, "")
}

// newBondOracleInDirWithOwner is like newBondOracleInDir but pins the root-owner
// key, so a restarted Oracle reuses the same owner the ledger was sealed with.
func newBondOracleInDirWithOwner(t *testing.T, dir, ownerHex string) (*Oracle, *crypto.KeyPair) {
	t.Helper()
	t.Setenv("KNIRV_SDK_DATA_DIR", "/tmp/knirvseed")
	cfg := newStartupTestConfig(t, dir)
	if ownerHex != "" {
		cfg.OwnerPrivateKey = ownerHex
	}
	o, err := NewOracle(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("new oracle: %v", err)
	}
	opKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	return o, opKey
}
