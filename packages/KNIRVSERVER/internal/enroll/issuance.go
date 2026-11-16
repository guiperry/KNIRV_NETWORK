package enroll

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// DefaultIssuanceTTL is how long an issued enrollment nonce stays redeemable when
// the caller does not ask for a specific lifetime.
const DefaultIssuanceTTL = 10 * time.Minute

// MaxIssuanceTTL caps the lifetime a caller may request for an issued nonce.
const MaxIssuanceTTL = time.Hour

// maxPendingIssuances bounds the in-memory set of outstanding challenges so a
// flood of issue requests cannot grow the process without limit.
const maxPendingIssuances = 1024

var (
	errIssuanceUnconfigured   = errors.New("challenge issuance is not configured")
	errNonceNotIssued         = errors.New("enrollment nonce was not issued by this server")
	errNonceExpired           = errors.New("issued enrollment nonce has expired")
	errNonceReused            = errors.New("issued enrollment nonce was already redeemed (possible replay)")
	errNonceRegistration      = errors.New("issued enrollment nonce belongs to a different registration")
	errTooManyPendingIssuance = errors.New("too many outstanding enrollment challenges")
)

// Issued is the server-issued challenge the approver hands to the registrant. The
// registrant's boot-node key must sign exactly this nonce, for this registration,
// chain and operator, before it expires.
type Issued struct {
	Nonce          string    `json:"nonce"`
	RegistrationID string    `json:"registration_id"`
	ChainID        string    `json:"chain_id"`
	ValidatorID    string    `json:"validator_id"`
	Operator       string    `json:"operator"`
	IssuedAt       time.Time `json:"issued_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type issuedRecord struct {
	registrationID string
	chainID        string
	validatorID    string
	expiresAt      time.Time
	redeemed       bool
}

// Issuer records server-issued, single-use enrollment nonces. A nonce is issued
// only for a registration the approver has already written to the approved
// registration store, so issuance is the "on approval" step of DEC-2: no nonce
// exists for an unapproved registrant, and a nonce is useless for any other
// registration or chain. Issuance state is in memory: a restart invalidates every
// outstanding challenge (fail closed), while redeemed nonces remain blocked by the
// durable NonceStore.
type Issuer struct {
	registrations RegistrationStore
	chainID       string
	ttl           time.Duration
	random        func([]byte) (int, error)

	mu      sync.Mutex
	pending map[string]*issuedRecord
}

// NewIssuer returns an Issuer bound to chainID. ttl <= 0 uses DefaultIssuanceTTL;
// values above MaxIssuanceTTL are clamped.
func NewIssuer(registrations RegistrationStore, chainID string, ttl time.Duration) *Issuer {
	if ttl <= 0 {
		ttl = DefaultIssuanceTTL
	}
	if ttl > MaxIssuanceTTL {
		ttl = MaxIssuanceTTL
	}
	return &Issuer{
		registrations: registrations,
		chainID:       strings.TrimSpace(chainID),
		ttl:           ttl,
		random:        rand.Read,
		pending:       map[string]*issuedRecord{},
	}
}

// Issue creates a fresh nonce for an approved registration. The registration must
// exist in the approved store, and its chain (if pinned) must match the issuer's.
func (i *Issuer) Issue(ctx context.Context, registrationID string, now time.Time, ttl time.Duration) (Issued, error) {
	if i == nil || i.registrations == nil {
		return Issued{}, errIssuanceUnconfigured
	}
	registrationID = strings.TrimSpace(registrationID)
	if registrationID == "" {
		return Issued{}, errors.New("registration_id is required")
	}
	reg, ok, err := i.registrations.Lookup(ctx, registrationID)
	if err != nil {
		return Issued{}, err
	}
	if !ok {
		return Issued{}, errUnknownRegistration
	}
	if i.chainID == "" {
		return Issued{}, errors.New("issuer chain is not configured")
	}
	if reg.ChainID != "" && reg.ChainID != i.chainID {
		return Issued{}, errWrongChain
	}
	if ttl <= 0 || ttl > MaxIssuanceTTL {
		ttl = i.ttl
	}
	if ttl > i.ttl {
		ttl = i.ttl
	}

	buf := make([]byte, 16)
	if _, err := i.random(buf); err != nil {
		return Issued{}, fmt.Errorf("generate nonce: %w", err)
	}
	nonce := "enroll:" + hex.EncodeToString(buf)

	i.mu.Lock()
	defer i.mu.Unlock()
	i.pruneLocked(now)
	if len(i.pending) >= maxPendingIssuances {
		return Issued{}, errTooManyPendingIssuance
	}
	expires := now.Add(ttl)
	i.pending[nonce] = &issuedRecord{
		registrationID: registrationID,
		chainID:        i.chainID,
		validatorID:    reg.ValidatorID,
		expiresAt:      expires,
	}
	return Issued{
		Nonce:          nonce,
		RegistrationID: registrationID,
		ChainID:        i.chainID,
		ValidatorID:    reg.ValidatorID,
		IssuedAt:       now,
		ExpiresAt:      expires,
	}, nil
}

// Check verifies that nonce was issued for registrationID on chainID, is not
// expired and has not been redeemed. It never changes issuance state, so a failed
// check leaves a valid nonce redeemable.
func (i *Issuer) Check(nonce, registrationID, chainID string, now time.Time) error {
	if i == nil {
		return errIssuanceUnconfigured
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	rec, ok := i.pending[nonce]
	if !ok {
		return errNonceNotIssued
	}
	if rec.redeemed {
		return errNonceReused
	}
	if !now.Before(rec.expiresAt) {
		return errNonceExpired
	}
	if rec.registrationID != registrationID {
		return errNonceRegistration
	}
	if rec.chainID != chainID {
		return errWrongChain
	}
	return nil
}

// Redeem marks an issued nonce as spent. The server calls it only after the
// durable NonceStore has consumed the nonce, so a redeemed nonce is also burned in
// the store and cannot be replayed after a restart.
func (i *Issuer) Redeem(nonce string) {
	if i == nil {
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if rec, ok := i.pending[nonce]; ok {
		rec.redeemed = true
	}
}

// Outstanding reports how many issued nonces are still tracked (tests, stats).
func (i *Issuer) Outstanding() int {
	i.mu.Lock()
	defer i.mu.Unlock()
	return len(i.pending)
}

// pruneLocked drops expired and redeemed records. Callers must hold i.mu.
func (i *Issuer) pruneLocked(now time.Time) {
	for nonce, rec := range i.pending {
		if rec.redeemed || !now.Before(rec.expiresAt) {
			delete(i.pending, nonce)
		}
	}
}
