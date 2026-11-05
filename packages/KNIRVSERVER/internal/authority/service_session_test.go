package authority

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/authority"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

// spySigner counts every signature the wrapped long-term key produces.
type spySigner struct {
	knirvsigning.RemoteSigner
	signatures atomic.Int64
}

func (s *spySigner) SignMessage(ctx context.Context, e knirvsigning.MessageEnvelope) (knirvsigning.SignedMessage, error) {
	s.signatures.Add(1)
	return s.RemoteSigner.SignMessage(ctx, e)
}

func (s *spySigner) SignDirect(ctx context.Context, r knirvsigning.SignRequest) (knirvsigning.SignedTransaction, error) {
	s.signatures.Add(1)
	return s.RemoteSigner.SignDirect(ctx, r)
}

type sessionHarness struct {
	*harness
	spy      *spySigner
	sessions *SessionManager
	ent      *toggleEntropy
}

func newSessionHarness(t *testing.T, kind string, seed byte, policy func(*Policy)) *sessionHarness {
	t.Helper()
	dir := t.TempDir()
	return buildSessionHarness(t, dir, kind, seed, policy)
}

func buildSessionHarness(t *testing.T, dir, kind string, seed byte, policy func(*Policy)) *sessionHarness {
	t.Helper()
	st, _, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	spy := &spySigner{RemoteSigner: testSigner(t, seed)}
	clk, logs, ent := newClock(), &logBuf{}, &toggleEntropy{}
	pol := DefaultPolicy()
	if policy != nil {
		policy(&pol)
	}
	scope := "*"
	if kind == KindBoot {
		scope = spy.Address()
	}
	sm, err := NewSessionManager(SessionConfig{
		LongTerm: spy, Entropy: ent, Scope: scope, TTL: time.Duration(pol.CertTTLSeconds) * time.Second,
		EpochFloor: pol.Epoch, Dir: dir, Now: clk.Now, Logf: logs.Logf,
	})
	if err != nil {
		t.Fatal(err)
	}
	svc, err := New(Config{Kind: kind, Signer: spy, Policy: pol, Store: st, Sessions: sm, Now: clk.Now, Logf: logs.Logf})
	if err != nil {
		t.Fatal(err)
	}
	return &sessionHarness{harness: &harness{svc: svc, store: st, signer: spy, clock: clk, logs: logs, dir: dir, kind: kind}, spy: spy, sessions: sm, ent: ent}
}

func TestRoutineSigningNeverUsesTheLongTermKey(t *testing.T) {
	h := newSessionHarness(t, KindRoot, 1, nil)
	if _, err := h.sessions.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := h.spy.signatures.Load(); got != 1 {
		t.Fatalf("the long-term key should have signed exactly one certificate, signed %d", got)
	}
	var recs []*sdk.Record
	for i := 0; i < 4; i++ {
		recs = append(recs, mustAuthorize(t, h.harness, act(1, "evidence.commit", "", `{"i":`+strings.Repeat("1", i+1)+`}`)).Record)
	}
	recs = append(recs, mustAuthorize(t, h.harness, act(2, "badge.revoke", "knirv1other", `{}`)).Record)
	if got := h.spy.signatures.Load(); got != 1 {
		t.Fatalf("routine records used the long-term key: %d signatures", got)
	}
	cur := h.sessions.Current(h.clock.Now())
	for i, r := range recs {
		if bytes.Equal(r.SessionCert.SessionPub, h.spy.PublicKey()) || !bytes.Equal(r.SessionCert.SessionPub, cur.Cert.SessionPub) {
			t.Fatalf("record %d is not under the session key", i)
		}
		if r.Epoch != cur.Epoch || r.SessionCert.Epoch != cur.Epoch {
			t.Fatalf("record %d epoch %d, session epoch %d", i, r.Epoch, cur.Epoch)
		}
		if err := sdk.Verify(r, registryFor(h.spy, KindRoot), h.clock.Now()); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}
	if st := h.svc.Status(); st.BreakGlass != 0 || st.Session == nil || st.Session.State != "active" || st.Epoch != cur.Epoch {
		t.Fatalf("%+v", st)
	}
}

func TestAuthorizeRunsTheFirstCeremonyAndLaterRotations(t *testing.T) {
	h := newSessionHarness(t, KindBoot, 2, nil)
	own := h.signer.Address()
	a := mustAuthorize(t, h.harness, act(1, "evidence.commit", own, `{"n":1}`)) // no explicit Ensure: Authorize starts the session
	if a.Record.Epoch != 1 || h.spy.signatures.Load() != 1 {
		t.Fatalf("epoch %d, long-term signatures %d", a.Record.Epoch, h.spy.signatures.Load())
	}
	h.clock.Advance(19 * time.Hour) // inside the last quarter of the 24 h epoch
	b := mustAuthorize(t, h.harness, act(1, "evidence.commit", own, `{"n":2}`))
	if b.Record.Epoch != 2 || bytes.Equal(a.Record.SessionCert.SessionPub, b.Record.SessionCert.SessionPub) {
		t.Fatalf("no rotation: epoch %d", b.Record.Epoch)
	}
	if h.spy.signatures.Load() != 2 {
		t.Fatalf("the long-term key must sign once per epoch, signed %d", h.spy.signatures.Load())
	}
	if b.Record.PrevRecordHash != a.Record.Hash {
		t.Fatal("the chain must continue across a key rotation")
	}
	// The first record still verifies as of its own time under the first key.
	if err := sdk.Verify(a.Record, h.registryWithRetired(), time.Unix(a.Stored.IssuedAt, 0)); err != nil {
		t.Fatalf("old record, as_of its issue time: %v", err)
	}
}

func (h *sessionHarness) registryWithRetired() sdk.Registry {
	reg := registryFor(h.spy, h.kind)
	op := reg.Operators[h.spy.Address()]
	op.RetiredSessions = h.sessions.Retired()
	reg.Operators[h.spy.Address()] = op
	return reg
}

func TestFailClosedWithoutASession(t *testing.T) {
	h := newSessionHarness(t, KindRoot, 1, nil)
	h.ent.set(true)
	for _, a := range []Action{act(2, "badge.revoke", "knirv1other", `{}`), act(1, "evidence.commit", "", `{}`)} {
		_, err := h.svc.Authorize(context.Background(), a)
		var refusal *Refusal
		if !errors.As(err, &refusal) || refusal.Code != RefusedSession || !errors.Is(err, sdk.ErrExpiredSession) {
			t.Fatalf("tier %d: want a session refusal, got %v", a.Tier, err)
		}
	}
	if h.store.Total() != 0 || h.spy.signatures.Load() != 0 {
		t.Fatal("something was signed or stored without a session")
	}
	if st := h.svc.Status(); st.Refusals[RefusedSession] != 2 || st.Session.State != "none" || st.Session.LastError == "" {
		t.Fatalf("%+v", st)
	}
	h.ent.set(false) // entropy recovers: the very next request heals the service
	if r := mustAuthorize(t, h.harness, act(2, "badge.revoke", "knirv1other", `{}`)); !r.Stored.Verified {
		t.Fatalf("%+v", r.Stored)
	}
}

func TestExpiredSessionStopsT2AndT3ButKeepsEarlierRecords(t *testing.T) {
	h := newSessionHarness(t, KindRoot, 1, nil)
	before := mustAuthorize(t, h.harness, act(2, "badge.revoke", "knirv1other", `{"n":1}`))
	h.ent.set(true)
	h.clock.Advance(25 * time.Hour) // past not_after of the only session
	for _, tier := range []uint8{1, 2} {
		if _, err := h.svc.Authorize(context.Background(), act(tier, "evidence.commit", "", `{}`)); !errors.Is(err, sdk.ErrExpiredSession) {
			t.Fatalf("tier %d after expiry: %v", tier, err)
		}
	}
	if sr, ok := h.store.Get(before.Stored.Hash()); !ok || !sr.Verified {
		t.Fatal("a record issued before expiry was lost")
	}
}

func TestBreakGlassIsOptInFlaggedLoggedAndNeverForT2(t *testing.T) {
	h := newSessionHarness(t, KindRoot, 1, func(p *Policy) { p.BreakGlassTiers = []string{"T1"} })
	h.ent.set(true)

	bg := mustAuthorize(t, h.harness, act(1, "evidence.commit", "", `{"n":1}`))
	if !bg.Stored.BreakGlass || !bytes.Equal(bg.Record.SessionCert.SessionPub, h.spy.PublicKey()) {
		t.Fatalf("expected a break-glass record signed by the long-term key: %+v", bg.Stored)
	}
	if !bg.Stored.Verified {
		t.Fatalf("a break-glass record must still verify: %+v", bg.Stored)
	}
	if err := sdk.Verify(bg.Record, registryFor(h.spy, KindRoot), h.clock.Now()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.logs.String(), "BREAK-GLASS") {
		t.Fatalf("break-glass must be logged: %q", h.logs.String())
	}
	if st := h.svc.Status(); st.BreakGlass != 1 {
		t.Fatalf("break-glass counter %d", st.BreakGlass)
	}
	if sr, _ := h.store.Get(bg.Stored.Hash()); !sr.BreakGlass {
		t.Fatal("the stored record lost its break-glass flag")
	}
	// T2 never falls through to the long-term key.
	if _, err := h.svc.Authorize(context.Background(), act(2, "badge.revoke", "knirv1other", `{}`)); !errors.Is(err, sdk.ErrExpiredSession) {
		t.Fatalf("T2 must fail closed: %v", err)
	}
	// Once a session exists again, records are normal and unflagged.
	h.ent.set(false)
	normal := mustAuthorize(t, h.harness, act(1, "evidence.commit", "", `{"n":2}`))
	if normal.Stored.BreakGlass || bytes.Equal(normal.Record.SessionCert.SessionPub, h.spy.PublicKey()) {
		t.Fatalf("a normal record was treated as break-glass: %+v", normal.Stored)
	}
	if st := h.svc.Status(); st.BreakGlass != 1 {
		t.Fatalf("break-glass counter %d", st.BreakGlass)
	}
}

func TestBreakGlassPolicyIsT1Only(t *testing.T) {
	for _, bad := range [][]string{{"T2"}, {"T3"}, {"T1", "T2"}, {"all"}} {
		p := DefaultPolicy()
		p.BreakGlassTiers = bad
		if err := p.Validate(); err == nil {
			t.Errorf("%v was accepted", bad)
		}
	}
	p := DefaultPolicy()
	base := p.Hash()
	p.BreakGlassTiers = []string{"T1"}
	if err := p.Validate(); err != nil || !p.BreakGlassAllowed(sdk.TierT1) || p.BreakGlassAllowed(sdk.TierT2) || p.Hash() == base {
		t.Fatal("T1 break-glass should validate, apply to T1 only, and change the policy hash")
	}
	if DefaultPolicy().Hash() != base {
		t.Fatal("adding the field must not change the hash of a policy that does not use it")
	}
}

func TestSessionRestartContinuesTheChainUnderANewKey(t *testing.T) {
	dir := t.TempDir()
	h1 := buildSessionHarness(t, dir, KindRoot, 1, nil)
	a := mustAuthorize(t, h1.harness, act(1, "evidence.commit", "", `{"n":1}`))
	h1.store.Close()

	h2 := buildSessionHarness(t, dir, KindRoot, 1, nil)
	b := mustAuthorize(t, h2.harness, act(1, "evidence.commit", "", `{"n":2}`))
	if b.Record.Epoch != a.Record.Epoch+1 || bytes.Equal(a.Record.SessionCert.SessionPub, b.Record.SessionCert.SessionPub) {
		t.Fatalf("a restart must start a new epoch with a new key: %d -> %d", a.Record.Epoch, b.Record.Epoch)
	}
	if b.Record.PrevRecordHash != a.Record.Hash || !b.Stored.Verified {
		t.Fatal("the chain did not continue across the restart")
	}
	if _, ok := h2.sessions.Retired()[hex.EncodeToString(a.Record.SessionCert.SessionPub)]; !ok {
		t.Fatal("the previous process's key is not recorded as retired")
	}
}

// Spec section 10 D1: retirement is recorded and published to the verifier
// registry, but a record carries no authenticated issue time, so a verifier can
// still accept a record that the retired key signs inside a certificate that
// started earlier. This pins the limitation so a fix is a deliberate change.
func TestRetiredKeyStillVerifiesInsideItsCertificateKnownLimitationD1(t *testing.T) {
	h := newSessionHarness(t, KindRoot, 1, nil)
	old, err := h.sessions.Ensure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.sessions.Rotate(context.Background(), "suspected theft"); err != nil {
		t.Fatal(err)
	}
	h.clock.Advance(time.Hour)
	// An attacker holding the old key signs a fresh record after the rotation.
	forged := &sdk.Record{
		Tier: sdk.TierT1, Domain: "evidence.commit", Scope: h.spy.Address(), Operator: h.spy.Address(), Epoch: old.Epoch,
		TargetBits: sdk.TierTargetBits(sdk.TierT1), WitnessKind: sdk.WitnessSoftware, SessionCert: old.Cert,
	}
	if err := forged.Seal(context.Background(), old.Signer, uint32(h.clock.Now().Unix())); err != nil {
		t.Fatal(err)
	}
	reg := h.registryWithRetired()
	if reg.Operators[h.spy.Address()].RetiredSessions[hex.EncodeToString(old.Cert.SessionPub)] == 0 {
		t.Fatal("the retirement is not in the registry")
	}
	if err := sdk.Verify(forged, reg, h.clock.Now()); err != nil {
		t.Skipf("the verifier now rejects this (D1 may have been fixed): %v", err)
	}
}

func TestHTTPRotateSessionEndpoint(t *testing.T) {
	h := newSessionHarness(t, KindRoot, 1, nil)
	mustAuthorize(t, h.harness, act(1, "evidence.commit", "", `{}`))
	_, c := startServer(t, h.svc, nil)

	code, st := c.do(t, "POST", "/authority/session/rotate", `{"reason":"laptop stolen"}`)
	if code != http.StatusOK || st["state"] != "active" || st["epoch"] != float64(2) || st["retired"] != float64(1) {
		t.Fatalf("rotate: %d %v", code, st)
	}
	if code, _ := c.do(t, "POST", "/authority/session/rotate", ``); code != http.StatusOK { // reason is optional
		t.Fatalf("rotate without a body: %d", code)
	}
	for name, tc := range map[string]struct {
		method, body string
		want         int
	}{
		"wrong method":  {"GET", ``, 405},
		"unknown field": {"POST", `{"why":"x"}`, 400},
		"long reason":   {"POST", `{"reason":"` + strings.Repeat("x", 201) + `"}`, 400},
	} {
		if code, _ := c.do(t, tc.method, "/authority/session/rotate", tc.body); code != tc.want {
			t.Errorf("%s: got %d, want %d", name, code, tc.want)
		}
	}
	if code, _ := (apiClient{http: c.http}).do(t, "POST", "/authority/session/rotate", `{}`); code != http.StatusUnauthorized {
		t.Fatalf("rotation without the token: %d", code)
	}
	// A service with no session manager says so instead of pretending.
	plain := newHarness(t, KindRoot, 3)
	_, c2 := startServer(t, plain.svc, nil)
	if code, _ := c2.do(t, "POST", "/authority/session/rotate", `{}`); code != http.StatusServiceUnavailable {
		t.Fatalf("no session manager: %d", code)
	}
}
