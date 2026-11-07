package token

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle/ledger"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

// ErrHoldAlreadyConsumed is returned by ConsumeHold when a hold has already been
// consumed by a *different* chain registration. It signals a double-spend attempt
// that the D14 escrow binding must reject: one enrolment hold authorises exactly
// one authority-chain registration, never two concurrent ones.
var ErrHoldAlreadyConsumed = errors.New("bond hold already consumed by a different chain")

// Bond escrow statuses (mirror ledger.BondStatus*).
const (
	bondStatusPending  = ledger.BondStatusPending
	bondStatusConsumed = ledger.BondStatusConsumed
	bondStatusReleased = ledger.BondStatusReleased
	bondStatusRefunded = ledger.BondStatusRefunded
)

// bondHold is the in-memory bond escrow state kept by the NRN token. It is
// reconstructed from the ledger on replay, so a restart observes the same holds
// as the live process.
type bondHold struct {
	HoldID          string
	Operator        types.Address
	Amount          *big.Int
	RootRegistryID  string
	ExpiresAtUnix   uint64
	EscrowAddr      types.Address
	Status          string
	BootNodeChainID string
	BootNodeAddress types.Address
	// ConsumedChainID binds the hold to the exact chain registration that consumed
	// it. It is set when the hold is released via ConsumeHold and stays empty for a
	// root-owner release/refund. Idempotent retries reuse the same chain id; a
	// second, different chain id is rejected so one escrow cannot back two chains.
	ConsumedChainID string
}

// bondEscrowAddress derives a deterministic escrow address from the hold id,
// mirroring the governance-deposit escrow derivation.
func bondEscrowAddress(holdID string) types.Address {
	sum := sha256.Sum256([]byte("knirv.oracle.bond.hold:" + holdID))
	addr, _ := types.AddressFromBytes(sum[:20])
	return addr
}

// Hold locks an operator's funds into a deterministic escrow address, keyed by
// holdID. It is idempotent: replaying the same hold while it is still pending is
// a no-op. The operator authorizes the hold with a bond-hold signature (domain
// knirv.registry, purpose bond-hold) bound to the trusted root registry id. The
// signature is verified against the canonical BondHold payload, so a tampered
// amount or operator is rejected.
//
// Atomicity: the funds are validated for sufficiency BEFORE any durable write, so
// an underfunded hold cannot poison the log. The transfer and the hold status are
// then committed as ONE validated ledger batch; the in-memory cache is mutated
// only after the batch is durable. A recorder failure or crash between the two
// records therefore leaves either nothing committed or a fully replayable pair —
// never live funds moved without a corresponding hold.
func (n *NRN) Hold(holdID string, hold knirvsigning.BondHold, signed knirvsigning.SignedMessage, now time.Time) error {
	if holdID == "" {
		return fmt.Errorf("bond hold: hold id is required")
	}
	if n.rootRegistryID == "" {
		return fmt.Errorf("bond hold: root registry id is not configured")
	}
	payload, err := json.Marshal(hold)
	if err != nil {
		return fmt.Errorf("bond hold: encode payload: %w", err)
	}
	// The envelope is verified on the Oracle's own chain id (the same chain id a
	// KNIRVCONTROLLER-signed message carries), with the canonical BondHold payload
	// and the bond-hold purpose. The root registry binding comes from the payload
	// (hold.RootRegistryID), and the operator authorization from the signer.
	if err := knirvsigning.VerifyMessagePayload(signed, knirvsigning.RegistryDomain, knirvsigning.PurposeBondHold, n.chainID, payload, now); err != nil {
		return fmt.Errorf("bond hold: %w", err)
	}
	if hold.HoldID != holdID {
		return fmt.Errorf("bond hold: hold id mismatch")
	}
	if hold.RootRegistryID != n.rootRegistryID {
		return fmt.Errorf("bond hold: root registry id mismatch")
	}
	operator, err := types.AddressFromString(hold.Operator)
	if err != nil {
		return fmt.Errorf("bond hold: operator address: %w", err)
	}
	if signed.Address != operator.String() {
		return fmt.Errorf("bond hold: signature does not authorize the named operator")
	}
	amount, ok := new(big.Int).SetString(hold.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		return fmt.Errorf("bond hold: invalid amount")
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	if existing, ok := n.bondHolds[holdID]; ok {
		if existing.Status != bondStatusPending {
			return fmt.Errorf("bond hold: hold %q already %s", holdID, existing.Status)
		}
		return nil
	}
	escrow := bondEscrowAddress(holdID)
	// Validate sufficiency before any durable write so an underfunded hold cannot
	// append a transfer that would later fail validation on replay.
	if err := n.checkMoveLocked(operator, escrow, amount); err != nil {
		return fmt.Errorf("bond hold: %w", err)
	}
	if err := n.recordBatch([]ledger.BatchEntry{
		{ledger.KindTokenTransfer, ledger.TokenTransfer{
			From: operator.String(), To: escrow.String(), Amount: amount.String(),
		}},
		{ledger.KindBondHold, ledger.BondHold{
			HoldID: holdID, Operator: operator.String(), Amount: amount.String(),
			RootRegistryID: hold.RootRegistryID, ExpiresAtUnix: hold.ExpiresAtUnix,
			EscrowAddr: escrow.String(), Status: bondStatusPending,
		}},
	}); err != nil {
		return fmt.Errorf("bond hold: record batch: %w", err)
	}
	// Mutate the cache only after the batch is durable.
	_ = n.moveLocked(operator, escrow, amount)
	n.bondHolds[holdID] = &bondHold{
		HoldID: holdID, Operator: operator, Amount: amount,
		RootRegistryID: hold.RootRegistryID, ExpiresAtUnix: hold.ExpiresAtUnix,
		EscrowAddr: escrow, Status: bondStatusPending,
	}
	return nil
}

// Release moves a held escrow to the boot node address named in a root-owner
// signed bond-release. It is idempotent: a replay that names the same boot node
// address is a no-op; one that names a different address is rejected. Only the
// root owner (the token owner) may release.
//
// Atomicity mirrors Hold: validate, commit the transfer + release as one batch,
// then mutate the cache.
func (n *NRN) Release(holdID string, release knirvsigning.BondRelease, signed knirvsigning.SignedMessage, now time.Time) error {
	if holdID == "" {
		return fmt.Errorf("bond release: hold id is required")
	}
	payload, err := json.Marshal(release)
	if err != nil {
		return fmt.Errorf("bond release: encode payload: %w", err)
	}
	if err := knirvsigning.VerifyMessagePayload(signed, knirvsigning.RegistryDomain, knirvsigning.PurposeBondRelease, n.chainID, payload, now); err != nil {
		return fmt.Errorf("bond release: %w", err)
	}
	if release.HoldID != holdID {
		return fmt.Errorf("bond release: hold id mismatch")
	}
	if signed.Address != n.owner.String() {
		return fmt.Errorf("bond release: not authorized by the root owner")
	}
	bootAddr, err := types.AddressFromString(release.BootNodeAddress)
	if err != nil {
		return fmt.Errorf("bond release: boot node address: %w", err)
	}
	if bootAddr.IsZero() {
		return fmt.Errorf("bond release: boot node address is required")
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	hold, ok := n.bondHolds[holdID]
	if !ok {
		return fmt.Errorf("bond release: unknown hold %q", holdID)
	}
	if hold.Status == bondStatusReleased {
		if hold.BootNodeAddress == bootAddr {
			return nil
		}
		return fmt.Errorf("bond release: hold %q already released to a different boot node", holdID)
	}
	if hold.Status == bondStatusRefunded {
		return fmt.Errorf("bond release: hold %q was already refunded", holdID)
	}
	// A consumed (reserved) hold is bound to a specific boot-node chain AND
	// address. A release must name both, so a replay or forgery cannot redirect
	// the escrow to a different chain or boot node. A pending hold has no such
	// binding yet, so only the consumed case is checked.
	if hold.Status == bondStatusConsumed {
		if release.BootNodeChainID != hold.ConsumedChainID {
			return fmt.Errorf("bond release: boot node chain id %q does not match the chain %q that consumed the hold", release.BootNodeChainID, hold.ConsumedChainID)
		}
		if bootAddr != hold.BootNodeAddress {
			return fmt.Errorf("bond release: boot node address %s does not match the address %s the hold was consumed for", bootAddr, hold.BootNodeAddress)
		}
	}
	if err := n.checkMoveLocked(hold.EscrowAddr, bootAddr, hold.Amount); err != nil {
		return fmt.Errorf("bond release: %w", err)
	}
	if err := n.recordBatch([]ledger.BatchEntry{
		{ledger.KindTokenTransfer, ledger.TokenTransfer{
			From: hold.EscrowAddr.String(), To: bootAddr.String(), Amount: hold.Amount.String(),
		}},
		{ledger.KindBondRelease, ledger.BondRelease{
			HoldID: holdID, BootNodeChainID: release.BootNodeChainID,
			BootNodeAddress: bootAddr.String(), Status: bondStatusReleased,
		}},
	}); err != nil {
		return fmt.Errorf("bond release: record batch: %w", err)
	}
	hold.Status = bondStatusReleased
	hold.BootNodeChainID = release.BootNodeChainID
	hold.BootNodeAddress = bootAddr
	_ = n.moveLocked(hold.EscrowAddr, bootAddr, hold.Amount)
	return nil
}

// Refund returns a held escrow to the operator, but only after the hold's
// expiry has passed. It is idempotent.
func (n *NRN) Refund(holdID string, now time.Time) error {
	if holdID == "" {
		return fmt.Errorf("bond refund: hold id is required")
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	hold, ok := n.bondHolds[holdID]
	if !ok {
		return fmt.Errorf("bond refund: unknown hold %q", holdID)
	}
	if hold.Status == bondStatusRefunded {
		return nil
	}
	if hold.Status == bondStatusReleased {
		return fmt.Errorf("bond refund: hold %q already released", holdID)
	}
	if hold.Status == bondStatusConsumed {
		return fmt.Errorf("bond refund: hold %q already consumed; release it instead of refunding", holdID)
	}
	if uint64(now.Unix()) <= hold.ExpiresAtUnix {
		return fmt.Errorf("bond refund: hold %q has not expired (expires %d)", holdID, hold.ExpiresAtUnix)
	}
	if err := n.checkMoveLocked(hold.EscrowAddr, hold.Operator, hold.Amount); err != nil {
		return fmt.Errorf("bond refund: %w", err)
	}
	if err := n.recordBatch([]ledger.BatchEntry{
		{ledger.KindTokenTransfer, ledger.TokenTransfer{
			From: hold.EscrowAddr.String(), To: hold.Operator.String(), Amount: hold.Amount.String(),
		}},
		{ledger.KindBondRelease, ledger.BondRelease{
			HoldID: holdID, Status: bondStatusRefunded,
		}},
	}); err != nil {
		return fmt.Errorf("bond refund: record batch: %w", err)
	}
	hold.Status = bondStatusRefunded
	_ = n.moveLocked(hold.EscrowAddr, hold.Operator, hold.Amount)
	return nil
}

// RefundExpiredHolds refunds every pending bond hold whose expiry has passed. It
// returns the number of holds refunded. It is used by the Oracle supervisor to
// sweep abandoned authority enrollments (registration never completed), so the
// operator's escrow is eventually returned rather than stranded.
func (n *NRN) RefundExpiredHolds(now time.Time) (int, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	refunded := 0
	for holdID, hold := range n.bondHolds {
		if hold.Status != bondStatusPending {
			continue
		}
		if hold.ExpiresAtUnix == 0 || uint64(now.Unix()) <= hold.ExpiresAtUnix {
			continue
		}
		if err := n.checkMoveLocked(hold.EscrowAddr, hold.Operator, hold.Amount); err != nil {
			return refunded, fmt.Errorf("bond refund sweep: hold %q: %w", holdID, err)
		}
		if err := n.recordBatch([]ledger.BatchEntry{
			{ledger.KindTokenTransfer, ledger.TokenTransfer{
				From: hold.EscrowAddr.String(), To: hold.Operator.String(), Amount: hold.Amount.String(),
			}},
			{ledger.KindBondRelease, ledger.BondRelease{
				HoldID: holdID, Status: bondStatusRefunded,
			}},
		}); err != nil {
			return refunded, fmt.Errorf("bond refund sweep: hold %q: record batch: %w", holdID, err)
		}
		hold.Status = bondStatusRefunded
		_ = n.moveLocked(hold.EscrowAddr, hold.Operator, hold.Amount)
		refunded++
	}
	return refunded, nil
}

// ConsumeHold releases a pending bond hold to the given address as part of an
// atomic caller operation (e.g. authority chain registration). It requires no
// external signature: the hold was authorized when it was created and the caller
// has already authenticated the surrounding operation.
//
// The hold is bound to the chain id that consumes it. Idempotent retry — a second
// ConsumeHold with the SAME chainID *and* destination for an already-released hold
// — is a no-op, so a registration retried after a lost response never double-sends
// the escrow. A CONSUME by a DIFFERENT chain id is rejected with
// ErrHoldAlreadyConsumed, so one D14 enrolment hold cannot authorise two concurrent
// authority-chain registrations (the round-4 TOCTOU). A hold that is unknown or
// already refunded is rejected.
//
// Atomicity: the status read, the consume-claim and the durable batch are all
// performed under a single lock, so the pending-check and the claim are one
// indivisible operation — two concurrent different-chain registrations cannot both
// pass the precheck and both consume.
func (n *NRN) ConsumeHold(holdID, chainID string, to types.Address) error {
	if holdID == "" {
		return fmt.Errorf("consume hold: hold id is required")
	}
	if chainID == "" {
		return fmt.Errorf("consume hold: chain id is required")
	}
	if to.IsZero() {
		return fmt.Errorf("consume hold: destination address is required")
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	hold, ok := n.bondHolds[holdID]
	if !ok {
		return fmt.Errorf("consume hold: unknown hold %q", holdID)
	}
	if hold.Status == bondStatusReleased {
		// Idempotent retry: same chain AND same destination as the original claim.
		if hold.ConsumedChainID == chainID && hold.BootNodeAddress == to {
			return nil
		}
		return fmt.Errorf("consume hold: %w (hold %q released to %s)", ErrHoldAlreadyConsumed, holdID, hold.ConsumedChainID)
	}
	if hold.Status == bondStatusConsumed {
		// Idempotent retry: same chain AND same destination as the original binding.
		if hold.ConsumedChainID == chainID && hold.BootNodeAddress == to {
			return nil
		}
		return fmt.Errorf("consume hold: %w (hold %q consumed by %s)", ErrHoldAlreadyConsumed, holdID, hold.ConsumedChainID)
	}
	if hold.Status == bondStatusRefunded {
		return fmt.Errorf("consume hold: hold %q already refunded", holdID)
	}
	// Pending: reserve the hold. The escrow stays put; we only bind the hold to
	// this chain (ConsumedChainID) and record the intended boot node address. The
	// funds are moved to the boot node address later, by the root-owner-signed
	// Release (see Release). The binding is the TOCTOU-safe single-use reservation
	// — redirecting the hold to another chain after binding is rejected, so
	// registration can never double-spend the escrow.
	if err := n.recordBatch([]ledger.BatchEntry{
		{ledger.KindBondRelease, ledger.BondRelease{
			HoldID: holdID, BootNodeAddress: to.String(), ConsumedChainID: chainID, Status: bondStatusConsumed,
		}},
	}); err != nil {
		return fmt.Errorf("consume hold: record batch: %w", err)
	}
	hold.Status = bondStatusConsumed
	hold.BootNodeAddress = to
	hold.ConsumedChainID = chainID
	return nil
}

// BondHoldView is a read-only projection of a bond hold, returned by GetHold. It
// carries the fields a caller needs to bind a hold to a registration (owner, amount,
// root registry id, status, and the chain id that consumed it) without exposing the
// token's internal escrow bookkeeping.
type BondHoldView struct {
	HoldID          string
	Operator        types.Address
	Amount          *big.Int
	RootRegistryID  string
	ExpiresAtUnix   uint64
	Status          string
	ConsumedChainID string
}

// GetHold returns a read-only view of a hold by id. An unknown hold is reported as
// an error so callers (e.g. chain registration) can reject the registration rather
// than treating a missing hold as a no-op.
func (n *NRN) GetHold(holdID string) (*BondHoldView, error) {
	if holdID == "" {
		return nil, fmt.Errorf("get hold: hold id is required")
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	h, ok := n.bondHolds[holdID]
	if !ok {
		return nil, fmt.Errorf("get hold: unknown hold %q", holdID)
	}
	return &BondHoldView{
		HoldID:          h.HoldID,
		Operator:        h.Operator,
		Amount:          new(big.Int).Set(h.Amount),
		RootRegistryID:  h.RootRegistryID,
		ExpiresAtUnix:   h.ExpiresAtUnix,
		Status:          h.Status,
		ConsumedChainID: h.ConsumedChainID,
	}, nil
}

// checkMoveLocked validates that `from` can cover `amount` without mutating any
// state. Callers hold n.mu. It is the pre-commit guard that keeps an underfunded
// operation from appending an invalid transfer to the durable log.
func (n *NRN) checkMoveLocked(from, to types.Address, amount *big.Int) error {
	fromBalance := n.balances[from]
	if fromBalance == nil || fromBalance.Cmp(amount) < 0 {
		return fmt.Errorf("%w: have %s, need %s", types.ErrInsufficientBalance, valueOrZero(fromBalance), amount)
	}
	return nil
}

// moveLocked transfers value between two addresses under the caller's lock. It
// fails without recording so a crash cannot split the move.
func (n *NRN) moveLocked(from, to types.Address, amount *big.Int) error {
	fromBalance := n.balances[from]
	if fromBalance == nil || fromBalance.Cmp(amount) < 0 {
		return fmt.Errorf("%w: have %s, need %s", types.ErrInsufficientBalance, valueOrZero(fromBalance), amount)
	}
	toBalance := n.balances[to]
	if toBalance == nil {
		toBalance = big.NewInt(0)
	}
	n.setBalance(from, new(big.Int).Sub(fromBalance, amount))
	n.setBalance(to, new(big.Int).Add(toBalance, amount))
	return nil
}
