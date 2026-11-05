// Package authority builds, signs, stores and anchors authority records
// (AUTHORITY_SPEC.md, Task 2b.1). It runs in OBSERVE mode only: every record is
// self-verified and the verdict is logged, counted and stored, but nothing is ever
// rejected because of it. Refusing to ISSUE a record (a boot node asked for a
// foreign scope) is not rejecting an action; the action itself is unaffected.
//
// The record format, digest, header, mining and verification live once, in the
// Rust core, reached through the SDK's authority package (repository rule 8). This
// package adds the node-side service around them.
//
// Where the owner has not decided something (spec section 10, D1 to D9), the
// code takes the conservative reading and marks it with the decision id.
package authority

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"sync"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/authority"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

// Modes. Only observe is implemented; enforce is accepted as a request and
// downgraded (Task 2d implements it).
const (
	ModeObserve = "observe"
	ModeEnforce = "enforce"
)

// Refusal codes: why a record was not issued.
const (
	RefusedTier     = "tier"
	RefusedScope    = "scope"
	RefusedAction   = "action"
	RefusedDegraded = "degraded"
	// RefusedSession means there is no live session key and break-glass is not
	// permitted for the tier (Task 2c.1 step 10, fail closed).
	RefusedSession = "session"
)

// Refusal is returned when the service will not issue a record. It carries the
// matching SDK sentinel (ErrTier, ErrScope, ErrInvalid) for errors.Is.
type Refusal struct {
	Code   string
	Reason string
	Err    error
}

func (r *Refusal) Error() string { return fmt.Sprintf("authority: refused (%s): %s", r.Code, r.Reason) }
func (r *Refusal) Unwrap() error { return r.Err }

// ErrDegraded means the service has no usable signing identity or store.
var ErrDegraded = errors.New("authority: service is degraded")

var (
	domainPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
	scopePattern  = regexp.MustCompile(`^[a-z0-9]{1,90}$`)
)

// Action is a request for a record.
type Action struct {
	Tier   uint8  `json:"tier"`
	Domain string `json:"domain"`
	// Scope is the operator address that owns the target; empty means the
	// node's own address.
	Scope string `json:"scope"`
	// Body is the action itself; its canonical JSON hash is
	// canonical_action_hash (canonical.go).
	Body []byte `json:"-"`
}

// Config assembles a Service.
type Config struct {
	Mode   string
	Kind   string // KindRoot or KindBoot
	Signer knirvsigning.RemoteSigner
	Policy Policy
	Store  *Store
	// Sessions, when set, supplies the ephemeral session key that signs routine
	// records (Task 2c.1). When nil the long-term key signs directly through a
	// self-issued certificate, the Task 2b.1 behaviour, which exists for tests and
	// tools only; Assemble always sets it.
	Sessions  *SessionManager
	Witnesser Witnesser
	Now       func() time.Time
	Logf      func(format string, args ...any)

	// DegradedReason, when set, builds a degraded service: it answers /status
	// and refuses to issue, so callers can tell "absent" from "broken".
	DegradedReason string
}

// Result is an issued record.
type Result struct {
	Record *sdk.Record
	Stored StoredRecord
}

// Service issues records for one operator.
type Service struct {
	cfg  Config
	mode string
	// enforceRequested is true when the operator asked for enforce. Reported in
	// /status so the downgrade is visible.
	enforceRequested bool
	degraded         string

	issue sync.Mutex      // serialises head read, mining, signing and append
	cert  sdk.SessionCert // self-issued certificate, used only without Sessions or for break-glass

	cmu        sync.Mutex
	issued     uint64
	verified   uint64
	breakGlass uint64
	failures   map[string]uint64
	refusals   map[string]uint64
}

// New builds a Service. A nil Witnesser means the software witnesser.
func New(cfg Config) (*Service, error) {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Logf == nil {
		cfg.Logf = func(string, ...any) {}
	}
	s := &Service{cfg: cfg, mode: ModeObserve, failures: map[string]uint64{}, refusals: map[string]uint64{}}
	s.noteMode(cfg.Mode)
	if cfg.DegradedReason != "" {
		s.degraded = cfg.DegradedReason
		return s, nil
	}
	if cfg.Signer == nil || cfg.Store == nil {
		return nil, errors.New("authority: a signer and a store are required")
	}
	if cfg.Kind != KindRoot && cfg.Kind != KindBoot {
		return nil, fmt.Errorf("authority: unknown node kind %q", cfg.Kind)
	}
	if err := cfg.Policy.Validate(); err != nil {
		return nil, err
	}
	if s.cfg.Witnesser == nil {
		s.cfg.Witnesser = SoftwareWitnesser{}
	}
	return s, nil
}

// noteMode applies the mode flag. OBSERVE-ONLY: "enforce" is recorded as
// requested but the service keeps observing, because enforcement is Task 2d and
// a half-built enforce mode that rejects things is worse than none.
func (s *Service) noteMode(requested string) {
	switch trimmedLower(requested) {
	case "", ModeObserve:
	case ModeEnforce:
		s.enforceRequested = true
		s.cfg.Logf("WARN authority.mode=enforce requested, but enforcement is not implemented in 2b.1; running in observe mode (nothing is rejected)")
	default:
		s.cfg.Logf("WARN authority.mode=%q is not recognised; running in observe mode", requested)
	}
}

// Operator returns the node's long-term address, or "" when degraded.
func (s *Service) Operator() string {
	if s.cfg.Signer == nil {
		return ""
	}
	return s.cfg.Signer.Address()
}

// Store exposes the record store (nil when degraded).
func (s *Service) Store() *Store { return s.cfg.Store }

// Signer exposes the signing identity (nil when degraded).
func (s *Service) Signer() knirvsigning.RemoteSigner { return s.cfg.Signer }

func (s *Service) refuse(code, reason string, err error) error {
	s.cmu.Lock()
	s.refusals[code]++
	s.cmu.Unlock()
	s.cfg.Logf("authority: refused to issue (%s): %s", code, reason)
	return &Refusal{Code: code, Reason: reason, Err: err}
}

// Authorize issues one verifying record for the action.
func (s *Service) Authorize(ctx context.Context, a Action) (*Result, error) {
	if s.degraded != "" {
		return nil, fmt.Errorf("%w: %s", ErrDegraded, s.degraded)
	}
	tier := sdk.Tier(a.Tier)
	if tier == sdk.TierT0 || !tier.Valid() {
		return nil, s.refuse(RefusedTier, fmt.Sprintf("tier %d carries no record (T0 has none; valid tiers are 1 to 3)", a.Tier), sdk.ErrTier)
	}
	if !domainPattern.MatchString(a.Domain) {
		return nil, s.refuse(RefusedAction, "domain must match [a-z0-9][a-z0-9._-]{0,127}", sdk.ErrInvalid)
	}
	own := s.cfg.Signer.Address()
	scope := a.Scope
	if scope == "" {
		scope = own
	}
	// A record scope is an address, never the certificate wildcard (conservative
	// reading: "*" is meaningful only in a certificate or registration).
	if !scopePattern.MatchString(scope) {
		return nil, s.refuse(RefusedScope, "scope must be an address of lowercase letters and digits, not a wildcard", sdk.ErrScope)
	}
	if ok, why := s.cfg.Policy.Allows(tier, s.cfg.Kind); !ok {
		return nil, s.refuse(RefusedTier, why, sdk.ErrTier)
	}
	// Scope rules. Boot signs only its own address. For Root, plan step 3 says
	// "any scope", but spec section 3 and Verify (check 13) require T1 to be scoped to
	// the operator's OWN address even for a root, so a Root T1 record for a foreign
	// scope would fail its own verification. Conservative reading, marked: refuse
	// it here (use T2 or higher for a foreign scope) rather than issue a record that
	// cannot verify. Plan and spec disagree; the spec and the verifier win.
	if s.cfg.Kind == KindBoot && scope != own {
		return nil, s.refuse(RefusedScope, fmt.Sprintf("a boot node may only sign scope %s (its own address)", own), sdk.ErrScope)
	}
	if tier == sdk.TierT1 && scope != own {
		return nil, s.refuse(RefusedScope, fmt.Sprintf("T1 is scoped to the operator's own address %s even for a root node; use T2 for a foreign scope", own), sdk.ErrScope)
	}
	actionHash, err := ActionHash(a.Body)
	if err != nil {
		return nil, s.refuse(RefusedAction, err.Error(), sdk.ErrInvalid)
	}

	s.issue.Lock()
	defer s.issue.Unlock()

	now := s.cfg.Now()
	id, err := s.signingIdentity(ctx, tier, now)
	if err != nil {
		return nil, err
	}
	cert := id.cert
	prev := s.cfg.Store.Head(own)
	rec := &sdk.Record{
		Tier: tier, Domain: a.Domain, Scope: scope, Operator: own, Epoch: id.epoch,
		CanonicalActionHash: actionHash, PrevRecordHash: prev,
		TargetBits: s.cfg.Policy.TargetBits(tier), WitnessKind: s.cfg.Witnesser.Kind(),
		SessionCert: cert,
	}
	digest := rec.Digest()

	ntime := uint32(now.Unix())
	var header sdk.Header
	var nonce uint32
	var hash [32]byte
	const ntimeRetries = 4
	for attempt := 0; ; attempt++ {
		header = sdk.NewHeader(digest, prev, ntime, rec.TargetBits, 0, sdk.HeaderVersion, tier)
		nonce, hash, err = s.cfg.Witnesser.Mine(ctx, [80]byte(header), rec.TargetBits)
		if err == nil {
			break
		}
		if errors.Is(err, ErrNonceExhausted) && attempt < ntimeRetries {
			ntime++ // the nonce space is per header; a later ntime is a fresh space
			continue
		}
		return nil, fmt.Errorf("authority: mining: %w", err)
	}
	header = sdk.NewHeader(digest, prev, ntime, rec.TargetBits, nonce, sdk.HeaderVersion, tier)
	if header.Witness() != hash {
		return nil, errors.New("authority: witnesser returned a hash that does not match its nonce")
	}
	rec.Header, rec.Nonce, rec.Hash = header, nonce, hash

	// Routine records are signed by the ephemeral session key; the long-term key
	// only certified it. A break-glass record is the one exception (logged below).
	if err := rec.Sign(ctx, id.signer); err != nil {
		return nil, fmt.Errorf("authority: sign: %w", err)
	}
	if id.breakGlass {
		s.cfg.Logf("WARN authority: BREAK-GLASS record %x (%s, tier %s) was signed by the long-term key because no session key is available", hash[:6], a.Domain, tier)
	}

	code := ""
	if err := sdk.Verify(rec, s.registry(), now); err != nil {
		code = FailureCode(err)
		s.cfg.Logf("WARN authority: record %x failed its own verification (%s): %v; observe mode keeps it", hash[:6], code, err)
	}
	stored := StoredRecord{Record: ToJSON(rec), IssuedAt: now.Unix(), Verified: code == "", ErrorCode: code, Mode: s.mode, BreakGlass: id.breakGlass}
	if err := s.cfg.Store.Append(stored); err != nil {
		return nil, fmt.Errorf("authority: store: %w", err)
	}
	s.cmu.Lock()
	s.issued++
	if id.breakGlass {
		s.breakGlass++
	}
	if code == "" {
		s.verified++
	} else {
		s.failures[code]++
	}
	s.cmu.Unlock()
	return &Result{Record: rec, Stored: stored}, nil
}

// signing is whoever signs a record and the certificate that vouches for them.
type signing struct {
	signer     knirvsigning.RemoteSigner
	cert       sdk.SessionCert
	epoch      uint64
	breakGlass bool
}

// signingIdentity picks the signer for a record of the given tier. Callers hold
// s.issue.
//
//  1. A live ephemeral session key, the normal path.
//  2. Otherwise, only if policy lists the tier in break_glass_tiers (T1 only), the
//     long-term key signs directly through a self-issued certificate and the
//     record is flagged break-glass.
//  3. Otherwise the request is refused (fail closed). T2 and T3 can never reach
//     step 2.
func (s *Service) signingIdentity(ctx context.Context, tier sdk.Tier, now time.Time) (signing, error) {
	if s.cfg.Sessions == nil {
		cert, err := s.selfCert(ctx, now, s.cfg.Policy.Epoch)
		if err != nil {
			return signing{}, fmt.Errorf("authority: session certificate: %w", err)
		}
		return signing{signer: s.cfg.Signer, cert: cert, epoch: s.cfg.Policy.Epoch}, nil
	}
	sess, err := s.cfg.Sessions.Ensure(ctx)
	if err == nil && sess != nil {
		return signing{signer: sess.Signer, cert: sess.Cert, epoch: sess.Epoch}, nil
	}
	if s.cfg.Policy.BreakGlassAllowed(tier) {
		epoch := max(s.cfg.Sessions.LastEpoch(), s.cfg.Policy.Epoch)
		cert, cerr := s.selfCert(ctx, now, epoch)
		if cerr != nil {
			return signing{}, fmt.Errorf("authority: break-glass certificate: %w", cerr)
		}
		return signing{signer: s.cfg.Signer, cert: cert, epoch: epoch, breakGlass: true}, nil
	}
	reason := "no live session key and the ceremony is failing"
	if err != nil {
		reason = fmt.Sprintf("no live session key: %v", err)
	}
	return signing{}, s.refuse(RefusedSession, reason+"; "+tier.String()+" is not permitted to use break-glass", sdk.ErrExpiredSession)
}

// selfCert returns a valid certificate in which the long-term key delegates to
// itself, renewing it when less than a quarter of its window remains or the epoch
// changed. It is used without a SessionManager and for break-glass only.
//
// Entropy is "csprng", the honest value when no ceremony ran. Whether csprng should
// count as break-glass is open (spec section 10, D4); this service flags a record
// as break-glass only when the long-term key itself signed it.
func (s *Service) selfCert(ctx context.Context, now time.Time, epoch uint64) (sdk.SessionCert, error) {
	ttl := s.cfg.Policy.CertTTLSeconds
	if s.cert.NotAfter != 0 && s.cert.Epoch == epoch && now.Unix() < s.cert.NotAfter-ttl/4 {
		return s.cert, nil
	}
	scope := s.cfg.Signer.Address()
	if s.cfg.Kind == KindRoot {
		scope = "*"
	}
	cert := sdk.SessionCert{
		SessionPub: s.cfg.Signer.PublicKey(),
		Operator:   s.cfg.Signer.Address(),
		Epoch:      epoch,
		NotBefore:  now.Unix() - 60, // tolerate a verifier clock slightly behind ours
		NotAfter:   now.Unix() + ttl,
		Scope:      scope,
		Entropy:    "csprng",
	}
	signed, err := sdk.SignSessionCert(ctx, cert, s.cfg.Signer)
	if err != nil {
		return sdk.SessionCert{}, err
	}
	s.cert = signed
	return signed, nil
}

// registry is the verifier's view used for self-verification: this one operator.
func (s *Service) registry() sdk.Registry {
	addr := s.cfg.Signer.Address()
	kind, scopes := sdk.OperatorBoot, []string{addr}
	if s.cfg.Kind == KindRoot {
		kind, scopes = sdk.OperatorRoot, []string{"*"}
	}
	return sdk.Registry{Operators: map[string]sdk.Operator{
		addr: {Address: addr, Kind: kind, LongTermPub: s.cfg.Signer.PublicKey(), Scopes: scopes, RetiredSessions: registryRetired(s.cfg.Sessions)},
	}}
}

// FailureCode names a verification error with the spec's failure code.
func FailureCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, sdk.ErrUnknownOperator):
		return "ErrUnknownOperator"
	case errors.Is(err, sdk.ErrExpiredSession):
		return "ErrExpiredSession"
	case errors.Is(err, sdk.ErrSessionSignature):
		return "ErrSessionSignature"
	case errors.Is(err, sdk.ErrRecordSignature):
		return "ErrRecordSignature"
	case errors.Is(err, sdk.ErrHeader):
		return "ErrHeader"
	case errors.Is(err, sdk.ErrWitness):
		return "ErrWitness"
	case errors.Is(err, sdk.ErrTarget):
		return "ErrTarget"
	case errors.Is(err, sdk.ErrScope):
		return "ErrScope"
	case errors.Is(err, sdk.ErrTier):
		return "ErrTier"
	case errors.Is(err, sdk.ErrInvalid):
		return "ErrInvalid"
	default:
		return "ErrOther"
	}
}

// Status is the body of GET /authority/status.
type Status struct {
	Mode             string            `json:"mode"`
	EnforceRequested bool              `json:"enforce_requested"`
	State            string            `json:"state"` // "ok" or "degraded"
	Reason           string            `json:"reason,omitempty"`
	Kind             string            `json:"kind,omitempty"`
	Operator         string            `json:"operator,omitempty"`
	ChainID          string            `json:"chain_id,omitempty"`
	Records          uint64            `json:"records"`
	Head             string            `json:"head,omitempty"`
	PolicyHash       string            `json:"policy_hash,omitempty"`
	Epoch            uint64            `json:"epoch,omitempty"`
	Issued           uint64            `json:"issued"`
	Verified         uint64            `json:"verified"`
	VerifyFailures   map[string]uint64 `json:"verify_failures"`
	Refusals         map[string]uint64 `json:"refusals"`
	Anchor           *AnchorStatus     `json:"anchor,omitempty"`
	Session          *SessionStatus    `json:"session,omitempty"`
	BreakGlass       uint64            `json:"break_glass"`
	RegistryEntry    *RegistryEntry    `json:"registry_entry,omitempty"`
}

// RegistryEntry is the operator's entry in the verifier registry, in the wire shape
// the Rust core accepts. A consumer that must verify one of this node's records
// from the record alone (KNIRVCHAIN, Task 2d.1) takes it from here.
//
// PROVISIONAL (spec section 10, D12): until the Oracle registry is the source of
// truth (Task 2d.2), this entry is self-asserted by the issuing node. It proves the
// record is internally consistent and that the operator address matches the key, not
// that the operator is authorized.
type RegistryEntry struct {
	Address         string           `json:"address"`
	Kind            string           `json:"kind"`
	LongTermPub     string           `json:"long_term_pub"`
	Scopes          []string         `json:"scopes"`
	RetiredSessions map[string]int64 `json:"retired_sessions,omitempty"`
}

// RegistryEntry returns this node's verifier-registry entry, or nil when degraded.
func (s *Service) RegistryEntry() *RegistryEntry {
	if s.degraded != "" || s.cfg.Signer == nil {
		return nil
	}
	addr := s.cfg.Signer.Address()
	scopes := []string{addr}
	if s.cfg.Kind == KindRoot {
		scopes = []string{"*"}
	}
	return &RegistryEntry{
		Address: addr, Kind: s.cfg.Kind, LongTermPub: hex.EncodeToString(s.cfg.Signer.PublicKey()),
		Scopes: scopes, RetiredSessions: registryRetired(s.cfg.Sessions),
	}
}

// ChainID is the Oracle chain this operator's records are anchored on.
func ChainID(operator string) string { return "authority/" + operator }

// Status reports the service's state and counters.
func (s *Service) Status() Status {
	s.cmu.Lock()
	defer s.cmu.Unlock()
	st := Status{
		Mode: s.mode, EnforceRequested: s.enforceRequested, State: "ok", Kind: s.cfg.Kind,
		Issued: s.issued, Verified: s.verified, BreakGlass: s.breakGlass,
		VerifyFailures: copyCounts(s.failures), Refusals: copyCounts(s.refusals),
	}
	if s.degraded != "" {
		st.State, st.Reason = "degraded", s.degraded
		return st
	}
	own := s.cfg.Signer.Address()
	head := s.cfg.Store.Head(own)
	st.Operator, st.ChainID = own, ChainID(own)
	st.Records = s.cfg.Store.Count(own)
	if st.Records > 0 {
		st.Head = fmt.Sprintf("%x", head)
	}
	st.PolicyHash, st.Epoch = s.cfg.Policy.Hash(), s.cfg.Policy.Epoch
	st.RegistryEntry = s.RegistryEntry()
	if s.cfg.Sessions != nil {
		ss := s.cfg.Sessions.Status(s.cfg.Now())
		st.Session = &ss
		if ss.Epoch != 0 {
			st.Epoch = ss.Epoch
		}
	}
	return st
}

func copyCounts(in map[string]uint64) map[string]uint64 {
	out := make(map[string]uint64, len(in))
	maps.Copy(out, in)
	return out
}

// Sessions exposes the session manager (nil without one).
func (s *Service) Sessions() *SessionManager { return s.cfg.Sessions }

// Run keeps the session key fresh until ctx ends. It is a no-op without a
// SessionManager.
func (s *Service) Run(ctx context.Context) {
	if s.cfg.Sessions != nil && s.degraded == "" {
		s.cfg.Sessions.Run(ctx, time.Minute)
	}
}
