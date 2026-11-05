package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Authority chains (AUTHORITY_SPEC section 2): the chain `authority/<operator>`
// is the commitment to one operator's authority records. This file holds the
// registry-side state that goes with it. It is local to this Oracle node and is
// not part of the replicated registry snapshot (see AUTHORITY_SPEC section 10, D14).

// AuthorityChainPrefix is the chain-id prefix reserved for authority chains.
const AuthorityChainPrefix = "authority/"

// DefaultAuthorityMinBond is the bond an `authority/*` chain must post on its
// first registration when the Oracle has not been configured otherwise. The
// owner set this to 10,000 NRN to replace the provisional observe-mode 0.
// An explicit 0 still disables the requirement (observe mode).
const DefaultAuthorityMinBond uint64 = 10000

// AuthorityOperator returns the operator address an authority chain id names.
func AuthorityOperator(chainID string) (string, bool) {
	if !strings.HasPrefix(chainID, AuthorityChainPrefix) {
		return "", false
	}
	op := strings.TrimPrefix(chainID, AuthorityChainPrefix)
	return op, op != "" && !strings.Contains(op, "/")
}

// OperatorEntry is the Oracle's record of one authority operator: the long-term
// public key its address is bound to, its kind and its granted scopes.
type OperatorEntry struct {
	Address         string           `json:"address"`
	Kind            string           `json:"kind"` // "root" | "boot"
	LongTermPub     []byte           `json:"long_term_pub"`
	Scopes          []string         `json:"scopes"`
	RetiredSessions map[string]int64 `json:"retired_sessions,omitempty"`
}

// Violation kinds.
const (
	ViolationScope        = "scope_violation"
	ViolationEquivocation = "equivocation"
)

// Violation is a proven, attributable authority misuse.
type Violation struct {
	Key        string    `json:"key"`
	Kind       string    `json:"kind"`
	Operator   string    `json:"operator"`
	ChainID    string    `json:"chain_id"`
	RecordHash string    `json:"record_hash"`
	OtherHash  string    `json:"other_record_hash,omitempty"`
	Reason     string    `json:"reason"`
	Amount     uint64    `json:"amount"`
	OccurredAt time.Time `json:"occurred_at"`
}

type authorityState struct {
	mu         sync.Mutex
	path       string
	minBond    uint64
	operators  map[string]*OperatorEntry
	seen       map[string]string // operator|prev_record_hash -> record hash
	violations []Violation
}

type authorityFile struct {
	MinBond    uint64                    `json:"min_bond"`
	Operators  map[string]*OperatorEntry `json:"operators"`
	Seen       map[string]string         `json:"seen"`
	Violations []Violation               `json:"violations"`
}

func newAuthorityState(path string) *authorityState {
	return &authorityState{path: path, minBond: DefaultAuthorityMinBond, operators: map[string]*OperatorEntry{}, seen: map[string]string{}}
}

func loadAuthorityState(dir string) (*authorityState, error) {
	if dir == "" {
		return newAuthorityState(""), nil
	}
	s := newAuthorityState(filepath.Join(dir, "authority_state.json"))
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("read authority state: %w", err)
	}
	var f authorityFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("decode authority state: %w", err)
	}
	s.minBond = f.MinBond
	if f.Operators != nil {
		s.operators = f.Operators
	}
	if f.Seen != nil {
		s.seen = f.Seen
	}
	s.violations = f.Violations
	return s, nil
}

// persistLocked writes the state atomically; the caller holds s.mu.
func (s *authorityState) persistLocked() error {
	if s.path == "" {
		return nil
	}
	payload, err := json.MarshalIndent(authorityFile{MinBond: s.minBond, Operators: s.operators, Seen: s.seen, Violations: s.violations}, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// SetMinAuthorityBond sets the bond an `authority/*` chain must post when it is
// first registered. The default is DefaultAuthorityMinBond (10,000 NRN); an
// explicit 0 restores observe mode (no requirement).
func (r *Registry) SetMinAuthorityBond(min uint64) error {
	r.auth.mu.Lock()
	defer r.auth.mu.Unlock()
	r.auth.minBond = min
	return r.auth.persistLocked()
}

// MinAuthorityBond returns the current requirement.
func (r *Registry) MinAuthorityBond() uint64 {
	r.auth.mu.Lock()
	defer r.auth.mu.Unlock()
	return r.auth.minBond
}

// checkAuthorityBond is called by Register for a chain that is not yet known.
func (r *Registry) checkAuthorityBond(chainID string, bond uint64) error {
	if _, ok := AuthorityOperator(chainID); !ok {
		return nil
	}
	if min := r.MinAuthorityBond(); bond < min {
		return fmt.Errorf("authority chain %s must post a bond of at least %d (got %d)", chainID, min, bond)
	}
	return nil
}

// Operator returns the stored entry for an operator address.
func (r *Registry) Operator(address string) (OperatorEntry, bool) {
	r.auth.mu.Lock()
	defer r.auth.mu.Unlock()
	e, ok := r.auth.operators[address]
	if !ok {
		return OperatorEntry{}, false
	}
	cp := *e
	cp.LongTermPub = append([]byte(nil), e.LongTermPub...)
	cp.Scopes = append([]string(nil), e.Scopes...)
	return cp, true
}

// SetOperator stores or replaces an operator entry. Callers decide who may do
// this; there is deliberately no network route for it (D12, D14).
func (r *Registry) SetOperator(e OperatorEntry) error {
	if e.Address == "" || len(e.LongTermPub) != 33 {
		return fmt.Errorf("operator entry needs an address and a 33-byte long-term key")
	}
	if e.Kind != "root" && e.Kind != "boot" {
		return fmt.Errorf("operator kind must be root or boot")
	}
	r.auth.mu.Lock()
	defer r.auth.mu.Unlock()
	cp := e
	cp.LongTermPub = append([]byte(nil), e.LongTermPub...)
	cp.Scopes = append([]string(nil), e.Scopes...)
	r.auth.operators[e.Address] = &cp
	return r.auth.persistLocked()
}

// ObserveAuthorityRecord remembers that `operator` chained `recordHash` onto
// `prev`. If a different record already claimed the same predecessor it returns
// that record's hash and true: the operator signed a fork (equivocation).
func (r *Registry) ObserveAuthorityRecord(operator, prev, recordHash string) (string, bool, error) {
	r.auth.mu.Lock()
	defer r.auth.mu.Unlock()
	key := operator + "|" + prev
	if have, ok := r.auth.seen[key]; ok {
		return have, have != recordHash, nil
	}
	r.auth.seen[key] = recordHash
	return "", false, r.auth.persistLocked()
}

// AddViolation records a violation once (by Key). It reports whether it was new.
func (r *Registry) AddViolation(v Violation) (bool, error) {
	r.auth.mu.Lock()
	defer r.auth.mu.Unlock()
	for _, have := range r.auth.violations {
		if have.Key == v.Key {
			return false, nil
		}
	}
	r.auth.violations = append(r.auth.violations, v)
	return true, r.auth.persistLocked()
}

// Violations lists the violations recorded against a chain.
func (r *Registry) Violations(chainID string) []Violation {
	r.auth.mu.Lock()
	defer r.auth.mu.Unlock()
	var out []Violation
	for _, v := range r.auth.violations {
		if v.ChainID == chainID {
			out = append(out, v)
		}
	}
	return out
}
