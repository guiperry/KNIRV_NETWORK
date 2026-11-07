package authority

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"knirv-server/pkg/embedded/validationchain/checkpoint"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

// OraclePoster is the part of checkpoint.Poster the anchorer needs. The real
// Poster satisfies it; tests use a fake.
type OraclePoster interface {
	RegisterChain(ctx context.Context, reg *checkpoint.ChainRegistration) (map[string]any, error)
	PostCheckpoint(ctx context.Context, cp *checkpoint.Checkpoint) (map[string]any, error)
}

// DefaultAuthorityBond is the authority/* chain bond (owner decision D14,
// 2026-11-04): exactly 10,000 NRN from the operator's controller vault. A
// missing bond defaults to this; any other explicit value is rejected so a
// misconfigured node never registers with the wrong bond.
const DefaultAuthorityBond uint64 = 10000

// AnchorConfig configures the Oracle side of an operator's authority chain.
type AnchorConfig struct {
	Poster OraclePoster
	Signer knirvsigning.RemoteSigner
	Store  *Store
	// Bond and BondOwner follow the operator-set pattern (D-U1): the operator
	// chooses them in configuration. Owner decision D14 pins the authority bond
	// to exactly DefaultAuthorityBond (10,000 NRN): a zero (unset) bond defaults
	// to it, and any other explicit value is rejected at construction.
	Bond      uint64
	BondOwner string

	// The long-term key is always the chain's author. Owner decision D10 (spec
	// section 10): Oracle-author key rotation is deprecated until a good use case
	// exists, so session keys sign routine records only and never become the
	// Oracle chain's author.
	Now  func() time.Time
	Logf func(format string, args ...any)
}

// AnchorStatus is the anchorer's part of /authority/status.
type AnchorStatus struct {
	Registered bool   `json:"registered"`
	LastEnd    uint64 `json:"last_end"`
	LastError  string `json:"last_error,omitempty"`
	// Author is the address registered as the chain's author at the Oracle. The
	// long-term key is always the author (D10: Oracle-author rotation deprecated).
	Author string `json:"author,omitempty"`
}

// Anchor registers the chain `authority/<operator address>` with the Oracle and
// commits the head of the operator's record chain to it. Each record's header
// commits to the previous record's hash, so the newest record hash is already a
// hash-chained accumulator over the whole history. The checkpoint root is that
// hash, the same construction the Validation Chain runtime uses for its block tip
// (AUTHORITY_SPEC.md section 1).
//
// Anchoring never blocks issuing: failures are logged, kept in /status, and
// retried on the next tick.
type Anchor struct {
	cfg AnchorConfig

	mu      sync.Mutex
	state   AnchorState
	loaded  bool
	lastErr string
}

// NewAnchor builds an Anchor. Registration and checkpoint continuity survive
// restarts through the store's anchor state.
func NewAnchor(cfg AnchorConfig) (*Anchor, error) {
	if cfg.Poster == nil || cfg.Signer == nil || cfg.Store == nil {
		return nil, errors.New("authority: anchor needs a poster, a signer and a store")
	}
	if cfg.Logf == nil {
		cfg.Logf = func(string, ...any) {}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	// Owner decision D14: the authority/* bond is exactly DefaultAuthorityBond
	// (10,000 NRN). An unset bond defaults to it; any other explicit value is
	// rejected so a misconfigured node never registers with the wrong bond. Apply
	// this before capturing cfg into the Anchor so the defaulted value is stored.
	switch {
	case cfg.Bond == 0:
		cfg.Logf("authority: bond unset; defaulting to D14 bond %d NRN for chain %s", DefaultAuthorityBond, ChainID(cfg.Signer.Address()))
		cfg.Bond = DefaultAuthorityBond
	case cfg.Bond != DefaultAuthorityBond:
		return nil, fmt.Errorf("authority: chain %s bond must be exactly %d NRN (owner decision D14); got %d", ChainID(cfg.Signer.Address()), DefaultAuthorityBond, cfg.Bond)
	}
	a := &Anchor{cfg: cfg}
	st, err := cfg.Store.LoadAnchor()
	if err != nil {
		return nil, fmt.Errorf("load anchor state: %w", err)
	}
	a.state, a.loaded = st, true
	return a, nil
}

// Status reports the anchorer's state.
func (a *Anchor) Status() AnchorStatus {
	a.mu.Lock()
	defer a.mu.Unlock()
	return AnchorStatus{Registered: a.state.Registered, LastEnd: a.state.LastEnd, LastError: a.lastErr, Author: a.registeredAuthorLocked()}
}

// registeredAuthorLocked is the address the Oracle has as the chain's author.
// The long-term key is always the author (D10: Oracle-author rotation
// deprecated), so this is simply the operator's signer address.
func (a *Anchor) registeredAuthorLocked() string {
	return a.cfg.Signer.Address()
}

// checkpointSignerLocked returns the key that speaks for the chain at the Oracle.
// The long-term key is always the author (D10: Oracle-author rotation is
// deprecated), so it is simply the operator's signer.
func (a *Anchor) checkpointSignerLocked() (knirvsigning.RemoteSigner, error) {
	return a.cfg.Signer, nil
}

// Registration builds the signed registration for the operator's chain: one
// author (the operator's current signer, weight 1), quorum 1/1.
func (a *Anchor) Registration() (*checkpoint.ChainRegistration, error) {
	signer := a.cfg.Signer
	reg := &checkpoint.ChainRegistration{
		ChainID: ChainID(signer.Address()),
		Authors: []checkpoint.RegisteredAuthor{
			{Address: signer.Address(), PubKey: signer.PublicKey(), Weight: 1},
		},
		QuorumNumer: 1,
		QuorumDenom: 1,
		Bond:        a.cfg.Bond,
		BondOwner:   a.cfg.BondOwner,
		ProofWindow: 0,
	}
	if err := checkpoint.SignRegistration(reg, signer); err != nil {
		return nil, fmt.Errorf("sign registration: %w", err)
	}
	return reg, nil
}

// EnsureRegistered registers the chain once. "Already registered" counts as done.
func (a *Anchor) EnsureRegistered(ctx context.Context) error {
	a.mu.Lock()
	done := a.state.Registered
	a.mu.Unlock()
	if done {
		return nil
	}
	reg, err := a.Registration()
	if err != nil {
		return err
	}
	if _, err := a.cfg.Poster.RegisterChain(ctx, reg); err != nil && !strings.Contains(err.Error(), "already registered") {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Registered = true
	return a.cfg.Store.SaveAnchor(a.state)
}

// Tick posts a checkpoint if the chain has grown since the last one. The saved
// state only advances after the Oracle accepts, so a failed post is retried with
// the same range.
func (a *Anchor) Tick(ctx context.Context) error {
	err := a.tick(ctx)
	a.mu.Lock()
	if err != nil {
		a.lastErr = err.Error()
	} else {
		a.lastErr = ""
	}
	a.mu.Unlock()
	return err
}

func (a *Anchor) tick(ctx context.Context) error {
	if err := a.EnsureRegistered(ctx); err != nil {
		return fmt.Errorf("register chain: %w", err)
	}
	// Post what is pending with the current (long-term) author.
	if err := a.postPending(ctx); err != nil {
		return err
	}
	return nil
}

func (a *Anchor) postPending(ctx context.Context) error {
	own := a.cfg.Signer.Address()
	height := a.cfg.Store.Count(own)

	a.mu.Lock()
	st := a.state
	signer, err := a.checkpointSignerLocked()
	a.mu.Unlock()
	if err != nil {
		return err
	}
	if height <= st.LastEnd {
		return nil
	}
	cp := &checkpoint.Checkpoint{
		SchemaVersion: checkpoint.SchemaVersion,
		ChainID:       ChainID(own),
		StartHeight:   st.LastEnd + 1,
		EndHeight:     height,
		Root:          a.cfg.Store.Head(own),
		Proposer:      signer.Address(), // must be a registered author at the Oracle
	}
	if st.PrevCheckpoint != "" {
		raw, err := hex.DecodeString(st.PrevCheckpoint)
		if err != nil || len(raw) != 32 {
			return fmt.Errorf("saved anchor state has a malformed prev_checkpoint %q", st.PrevCheckpoint)
		}
		copy(cp.PrevCheckHash[:], raw)
	}
	if err := checkpoint.SignCheckpoint(cp, signer); err != nil {
		return fmt.Errorf("sign checkpoint: %w", err)
	}
	if _, err := a.cfg.Poster.PostCheckpoint(ctx, cp); err != nil {
		return err
	}
	digest := cp.Digest()
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.LastEnd = cp.EndHeight
	a.state.PrevCheckpoint = hex.EncodeToString(digest[:])
	if err := a.cfg.Store.SaveAnchor(a.state); err != nil {
		return fmt.Errorf("save anchor state: %w", err)
	}
	a.cfg.Logf("authority: checkpoint posted for %s, heights %d-%d", cp.ChainID, cp.StartHeight, cp.EndHeight)
	return nil
}

// Oracle-author rotation (the session key becoming the chain's author) was
// removed: owner decision D10 (spec section 10) deprecated it until a good use
// case exists. The long-term key is always the chain's author; session keys sign
// routine records only and never become the Oracle chain's author.

// Run ticks every interval until ctx ends. Errors are logged, never fatal.
func (a *Anchor) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if err := a.Tick(ctx); err != nil {
		a.cfg.Logf("WARN authority: anchoring failed (will retry): %v", err)
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := a.Tick(ctx); err != nil {
				a.cfg.Logf("WARN authority: anchoring failed (will retry): %v", err)
			}
		}
	}
}
