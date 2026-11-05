package authority

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/authority"
)

func mustAuthorize(t *testing.T, h *harness, a Action) *Result {
	t.Helper()
	res, err := h.svc.Authorize(context.Background(), a)
	if err != nil {
		t.Fatalf("Authorize(%+v): %v", a, err)
	}
	return res
}

func verifyIndependently(t *testing.T, h *harness, rec *sdk.Record, at time.Time) error {
	t.Helper()
	return sdk.Verify(rec, registryFor(h.signer, h.kind), at)
}

func TestRootNodeProducesVerifyingChainedRecords(t *testing.T) {
	h := newHarness(t, KindRoot, 1)
	own := h.signer.Address()
	foreign := "knirv1otheroperator"

	r1 := mustAuthorize(t, h, act(1, "evidence.commit", "", `{"event":"e1","n":1}`))
	r2 := mustAuthorize(t, h, act(2, "badge.assign", foreign, `{"user":"u7"}`)) // a foreign scope needs T2+ (spec section 3)
	r3 := mustAuthorize(t, h, act(2, "badge.revoke", foreign, `{"user":"u7"}`))

	for i, r := range []*Result{r1, r2, r3} {
		if !r.Stored.Verified || r.Stored.ErrorCode != "" {
			t.Fatalf("record %d did not verify at issue: %+v", i+1, r.Stored)
		}
		if err := verifyIndependently(t, h, r.Record, h.clock.Now()); err != nil {
			t.Fatalf("record %d fails independent verification: %v", i+1, err)
		}
	}
	// Chain: genesis -> r1 -> r2 -> r3, one chain per operator regardless of scope.
	if r1.Record.PrevRecordHash != ([32]byte{}) || r2.Record.PrevRecordHash != r1.Record.Hash || r3.Record.PrevRecordHash != r2.Record.Hash {
		t.Fatal("records do not form a hash chain")
	}
	if r1.Record.Scope != own || r2.Record.Scope != foreign {
		t.Fatalf("scopes %q %q", r1.Record.Scope, r2.Record.Scope)
	}
	// The header commits to the action digest, which is the spec's formula
	// (implemented independently in the test) over the record's own fields.
	for i, r := range []*Result{r1, r2, r3} {
		rec := r.Record
		want := specDigest(uint8(rec.Tier), rec.Domain, rec.Scope, rec.Operator, rec.Epoch, rec.CanonicalActionHash, rec.PrevRecordHash)
		if rec.Header.PrevHash() != want {
			t.Fatalf("record %d: header prevhash does not equal the spec digest", i+1)
		}
		if rec.Header.Merkle() != rec.PrevRecordHash {
			t.Fatalf("record %d: header merkle is not prev_record_hash", i+1)
		}
		if rec.TargetBits != sdk.TierTargetBits(rec.Tier) || rec.WitnessKind != sdk.WitnessSoftware {
			t.Fatalf("record %d: target %#x kind %q", i+1, rec.TargetBits, rec.WitnessKind)
		}
	}
	st := h.svc.Status()
	if st.Records != 3 || st.Issued != 3 || st.Verified != 3 || len(st.VerifyFailures) != 0 || st.Mode != ModeObserve || st.State != "ok" {
		t.Fatalf("status: %+v", st)
	}
	if st.ChainID != "authority/"+own || st.Kind != KindRoot {
		t.Fatalf("status identity: %+v", st)
	}
}

// Plan step 3 says Root may sign any scope, but T1 is own-address only in the spec
// and in Verify. The service refuses instead of issuing a record that cannot verify.
func TestRootT1ForForeignScopeIsRefusedNotIssuedUnverifiable(t *testing.T) {
	h := newHarness(t, KindRoot, 1)
	_, err := h.svc.Authorize(context.Background(), act(1, "evidence.commit", "knirv1otheroperator", `{}`))
	var refusal *Refusal
	if !errors.As(err, &refusal) || refusal.Code != RefusedScope || !errors.Is(err, sdk.ErrScope) {
		t.Fatalf("want a scope refusal, got %v", err)
	}
	if h.store.Total() != 0 {
		t.Fatal("a refusal stored a record")
	}
	if _, err := h.svc.Authorize(context.Background(), act(2, "badge.revoke", "knirv1otheroperator", `{}`)); err != nil {
		t.Fatalf("root T2 for a foreign scope should work: %v", err)
	}
}

func TestBootNodeSignsOwnScopeAndRefusesTheRest(t *testing.T) {
	h := newHarness(t, KindBoot, 2)
	own := h.signer.Address()

	ok := mustAuthorize(t, h, act(1, "evidence.commit", own, `{"e":1}`))
	if !ok.Stored.Verified {
		t.Fatalf("boot node's own-scope record did not verify: %+v", ok.Stored)
	}
	if err := verifyIndependently(t, h, ok.Record, h.clock.Now()); err != nil {
		t.Fatal(err)
	}
	if empty := mustAuthorize(t, h, act(1, "evidence.commit", "", `{"e":2}`)); empty.Record.Scope != own {
		t.Fatalf("an empty scope should default to the node's own address, got %q", empty.Record.Scope)
	}

	refusals := []struct {
		name string
		a    Action
		code string
		is   error
	}{
		{"foreign scope", act(1, "evidence.commit", "knirv1someoneelse", `{}`), RefusedScope, sdk.ErrScope},
		{"T2 without root co-attestation", act(2, "badge.revoke", own, `{}`), RefusedTier, sdk.ErrTier},
		{"T3 is disabled and root-only", act(3, "policy.change", own, `{}`), RefusedTier, sdk.ErrTier},
		{"T0 carries no record", act(0, "read.thing", own, `{}`), RefusedTier, sdk.ErrTier},
		{"wildcard scope", act(1, "evidence.commit", "*", `{}`), RefusedScope, sdk.ErrScope},
		{"bad domain", act(1, "Not A Domain", own, `{}`), RefusedAction, sdk.ErrInvalid},
		{"non-canonical action", act(1, "evidence.commit", own, `{"x":1.5}`), RefusedAction, sdk.ErrInvalid},
		{"empty action", act(1, "evidence.commit", own, ``), RefusedAction, sdk.ErrInvalid},
	}
	for _, tc := range refusals {
		t.Run(tc.name, func(t *testing.T) {
			before := h.store.Count(own)
			_, err := h.svc.Authorize(context.Background(), tc.a)
			var refusal *Refusal
			if !errors.As(err, &refusal) || refusal.Code != tc.code {
				t.Fatalf("want a %s refusal, got %v", tc.code, err)
			}
			if !errors.Is(err, tc.is) {
				t.Fatalf("refusal does not wrap %v: %v", tc.is, err)
			}
			if h.store.Count(own) != before {
				t.Fatal("a refusal stored a record")
			}
		})
	}
	if got := h.svc.Status().Refusals; got[RefusedScope] != 2 || got[RefusedTier] != 3 || got[RefusedAction] != 3 {
		t.Fatalf("refusal counters: %v", got)
	}
}

func TestRootCannotUseDisabledT3(t *testing.T) {
	h := newHarness(t, KindRoot, 1)
	_, err := h.svc.Authorize(context.Background(), act(3, "policy.change", "", `{}`))
	var refusal *Refusal
	if !errors.As(err, &refusal) || refusal.Code != RefusedTier || !strings.Contains(refusal.Reason, "disabled") {
		t.Fatalf("T3 must be refused as disabled by default, got %v", err)
	}
}

func TestChainSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	h1 := buildHarness(t, dir, KindRoot, 1)
	a := mustAuthorize(t, h1, act(1, "evidence.commit", "", `{"n":1}`))
	b := mustAuthorize(t, h1, act(1, "evidence.commit", "", `{"n":2}`))
	h1.store.Close()

	h2 := buildHarness(t, dir, KindRoot, 1)
	if h2.svc.Status().Records != 2 {
		t.Fatalf("records after restart: %d", h2.svc.Status().Records)
	}
	c := mustAuthorize(t, h2, act(1, "evidence.commit", "", `{"n":3}`))
	if c.Record.PrevRecordHash != b.Record.Hash || b.Record.PrevRecordHash != a.Record.Hash {
		t.Fatal("the chain did not continue across the restart")
	}
	if !c.Stored.Verified {
		t.Fatalf("post-restart record did not verify: %+v", c.Stored)
	}
	if sr, ok := h2.store.Get(a.Stored.Hash()); !ok || sr.Record.Hash != a.Stored.Hash() {
		t.Fatal("an old record is not retrievable after restart")
	}
}

func TestObserveModeNeverRejects(t *testing.T) {
	// A witnesser that returns a consistent but unmined nonce: the record fails its
	// own witness check. Observe mode keeps and stores it, counts the failure, and
	// the chain continues. Nothing is rejected.
	bad := scriptedWitnesser{mine: func(_ context.Context, header [80]byte, bits uint32) (uint32, [32]byte, error) {
		h := sdk.Header(header)
		return 0, h.Witness(), nil
	}}
	h := newHarness(t, KindRoot, 1, func(c *Config) { c.Witnesser = bad })

	var failed *Result
	for i := 0; i < 6 && failed == nil; i++ { // T2 accepts ~1 in 65k hashes; retry in the odd lucky case
		res := mustAuthorize(t, h, act(2, "badge.revoke", "", `{"i":`+string(rune('0'+i))+`}`))
		if !res.Stored.Verified {
			failed = res
		}
	}
	if failed == nil {
		t.Fatal("could not produce a record that fails verification")
	}
	if failed.Stored.ErrorCode != "ErrWitness" || failed.Stored.Mode != ModeObserve {
		t.Fatalf("stored verdict: %+v", failed.Stored)
	}
	st := h.svc.Status()
	if st.VerifyFailures["ErrWitness"] == 0 || st.State != "ok" {
		t.Fatalf("status after a failed self-verification: %+v", st)
	}
	if !strings.Contains(h.logs.String(), "observe mode keeps it") {
		t.Fatalf("the failure was not logged: %q", h.logs.String())
	}
	next := mustAuthorize(t, h, act(1, "evidence.commit", "", `{}`))
	if next.Record.PrevRecordHash != failed.Record.Hash && h.store.Count(h.signer.Address()) < 2 {
		t.Fatal("the chain stopped at a record that failed verification")
	}
}

func TestEnforceIsRequestedButNotImplemented(t *testing.T) {
	for _, mode := range []string{ModeEnforce, "ENFORCE"} {
		h := newHarness(t, KindRoot, 1, func(c *Config) { c.Mode = mode })
		st := h.svc.Status()
		if st.Mode != ModeObserve || !st.EnforceRequested {
			t.Fatalf("mode %q: %+v", mode, st)
		}
		if !strings.Contains(h.logs.String(), "running in observe mode") {
			t.Fatalf("no warning logged for %q: %q", mode, h.logs.String())
		}
	}
	h := newHarness(t, KindRoot, 1, func(c *Config) { c.Mode = "strict" })
	if h.svc.Status().Mode != ModeObserve || h.svc.Status().EnforceRequested || !strings.Contains(h.logs.String(), "not recognised") {
		t.Fatal("an unknown mode must fall back to observe with a warning")
	}
}

func TestSessionCertificateIsRenewedAndOldRecordsStayVerifiableAtTheirTime(t *testing.T) {
	h := newHarness(t, KindRoot, 1)
	ttl := time.Duration(h.svc.cfg.Policy.CertTTLSeconds) * time.Second

	first := mustAuthorize(t, h, act(1, "evidence.commit", "", `{"n":1}`))
	same := mustAuthorize(t, h, act(1, "evidence.commit", "", `{"n":2}`))
	if first.Record.SessionCert.NotAfter != same.Record.SessionCert.NotAfter {
		t.Fatal("the certificate was renewed too early")
	}
	h.clock.Advance(ttl*3/4 + time.Minute) // into the last quarter of the window
	renewed := mustAuthorize(t, h, act(1, "evidence.commit", "", `{"n":3}`))
	if renewed.Record.SessionCert.NotAfter <= first.Record.SessionCert.NotAfter {
		t.Fatal("the certificate was not renewed")
	}
	h.clock.Advance(ttl * 2) // the first certificate has long expired
	late := mustAuthorize(t, h, act(1, "evidence.commit", "", `{"n":4}`))
	if !late.Stored.Verified {
		t.Fatalf("a record after expiry of the old certificate must still verify: %+v", late.Stored)
	}
	// Spec section 5.2 / open D3: a record verifies only inside its own certificate
	// window. Re-verifying the first record now, with the wrong time, fails; with
	// its issue time (the as_of the store keeps) it passes.
	if err := verifyIndependently(t, h, first.Record, h.clock.Now()); !errors.Is(err, sdk.ErrExpiredSession) {
		t.Fatalf("re-verification at the current time: want ErrExpiredSession, got %v", err)
	}
	if err := verifyIndependently(t, h, first.Record, time.Unix(first.Stored.IssuedAt, 0)); err != nil {
		t.Fatalf("re-verification as of issue time: %v", err)
	}
}

func TestInconsistentWitnessIsAnErrorNotARecord(t *testing.T) {
	liar := scriptedWitnesser{mine: func(context.Context, [80]byte, uint32) (uint32, [32]byte, error) {
		return 5, [32]byte{1}, nil // hash that does not belong to nonce 5
	}}
	h := newHarness(t, KindRoot, 1, func(c *Config) { c.Witnesser = liar })
	if _, err := h.svc.Authorize(context.Background(), act(1, "evidence.commit", "", `{}`)); err == nil {
		t.Fatal("a witnesser that returns a mismatched hash must not produce a record")
	}
	if h.store.Total() != 0 {
		t.Fatal("a record was stored")
	}
}

func TestNtimeExtendsWhenTheNonceSpaceIsExhausted(t *testing.T) {
	calls := 0
	w := scriptedWitnesser{mine: func(ctx context.Context, header [80]byte, bits uint32) (uint32, [32]byte, error) {
		calls++
		if calls <= 2 {
			return 0, [32]byte{}, ErrNonceExhausted
		}
		return SoftwareWitnesser{}.Mine(ctx, header, bits)
	}}
	h := newHarness(t, KindRoot, 1, func(c *Config) { c.Witnesser = w })
	res := mustAuthorize(t, h, act(1, "evidence.commit", "", `{}`))
	if want := uint32(h.clock.Now().Unix()) + 2; res.Record.Header.NTime() != want {
		t.Fatalf("ntime %d, want %d (extended twice)", res.Record.Header.NTime(), want)
	}
	if !res.Stored.Verified {
		t.Fatalf("record with an extended ntime did not verify: %+v", res.Stored)
	}
	// Persistent exhaustion gives up with an error rather than looping forever.
	never := scriptedWitnesser{mine: func(context.Context, [80]byte, uint32) (uint32, [32]byte, error) {
		return 0, [32]byte{}, ErrNonceExhausted
	}}
	h2 := newHarness(t, KindRoot, 1, func(c *Config) { c.Witnesser = never })
	if _, err := h2.svc.Authorize(context.Background(), act(1, "evidence.commit", "", `{}`)); !errors.Is(err, ErrNonceExhausted) {
		t.Fatalf("want ErrNonceExhausted, got %v", err)
	}
}

func TestConcurrentAuthorizationNeverForksTheChain(t *testing.T) {
	h := newHarness(t, KindRoot, 1)
	const n = 12
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := h.svc.Authorize(context.Background(), act(1, "evidence.commit", "", `{"i":`+strings.Repeat("1", i+1)+`}`))
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := h.store.Count(h.signer.Address()); got != n {
		t.Fatalf("stored %d of %d records", got, n)
	}
	// Re-open to run the store's own chain-integrity check over everything written.
	h.store.Close()
	st, _, err := OpenStore(h.dir)
	if err != nil {
		t.Fatalf("the stored chain is not a single linear chain: %v", err)
	}
	st.Close()
}

func TestDegradedServiceRefusesAndReports(t *testing.T) {
	svc, err := New(Config{DegradedReason: "no root.key or boot.key on this node", Mode: ModeObserve})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authorize(context.Background(), act(1, "evidence.commit", "", `{}`)); !errors.Is(err, ErrDegraded) {
		t.Fatalf("want ErrDegraded, got %v", err)
	}
	if st := svc.Status(); st.State != "degraded" || !strings.Contains(st.Reason, "no root.key") || st.Mode != ModeObserve {
		t.Fatalf("status: %+v", st)
	}
}

func TestNewRefusesIncompleteConfig(t *testing.T) {
	if _, err := New(Config{Kind: KindRoot}); err == nil {
		t.Error("a service without a signer or store was built")
	}
	h := newHarness(t, KindRoot, 1)
	if _, err := New(Config{Kind: "client", Signer: h.signer, Store: h.store, Policy: DefaultPolicy()}); err == nil {
		t.Error("an unknown node kind was accepted")
	}
	bad := DefaultPolicy()
	bad.Epoch = 0
	if _, err := New(Config{Kind: KindRoot, Signer: h.signer, Store: h.store, Policy: bad}); err == nil {
		t.Error("an invalid policy was accepted")
	}
}
