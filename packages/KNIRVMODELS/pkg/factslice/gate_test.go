package factslice

import (
	"math/rand"
	"testing"

	"knirvmodels/pkg/evidence"
)

func one() []evidence.Hit { return []evidence.Hit{{SpanID: "s1", Tier: evidence.TierReference}} }

// full is a claim that satisfies every condition of every level.
func full() ClaimResult {
	return ClaimResult{Verdict: Supported, Calibrated: 0.95, Evidence: one(), Cited: []string{"s1"},
		Agreement: true, ReferenceTier: true, Corroborated: true, Witnessed: true, Proof: true, Certificate: true, Anchored: true}
}

func TestGateTable(t *testing.T) {
	th := DefaultThresholds()
	mod := func(f func(*ClaimResult)) ClaimResult { r := full(); f(&r); return r }
	cases := []struct {
		name string
		r    ClaimResult
		want GateLevel
	}{
		{"everything holds", full(), GateCommitted},
		{"no proof", mod(func(r *ClaimResult) { r.Proof = false }), GateAccepted},
		{"no certificate", mod(func(r *ClaimResult) { r.Certificate = false }), GateAccepted},
		{"not anchored", mod(func(r *ClaimResult) { r.Anchored = false }), GateAccepted},
		{"a=0.88 cannot commit", mod(func(r *ClaimResult) { r.Calibrated = 0.88 }), GateAccepted},
		{"a=0.85 exactly accepts", mod(func(r *ClaimResult) { r.Calibrated = 0.85; r.Proof = false }), GateAccepted},
		{"a just under accepted", mod(func(r *ClaimResult) { r.Calibrated = 0.8499 }), GateAttested},
		{"not corroborated", mod(func(r *ClaimResult) { r.Corroborated = false }), GateAttested},
		{"judges disagreed on order", mod(func(r *ClaimResult) { r.Agreement = false }), GateAttested},
		{"not reference tier", mod(func(r *ClaimResult) { r.ReferenceTier = false }), GateAttested},
		{"cumulative: sealed but unwitnessed stops at seen", mod(func(r *ClaimResult) { r.Witnessed = false }), GateSeen},
		{"a=0.70 witnessed attests", mod(func(r *ClaimResult) { r.Calibrated = 0.70; r.Agreement, r.ReferenceTier = false, false }), GateAttested},
		{"a=0.65 witnessed is only seen", mod(func(r *ClaimResult) { r.Calibrated = 0.65 }), GateSeen},
		{"a=0.50 seen", mod(func(r *ClaimResult) { r.Calibrated = 0.50; r.Witnessed = false }), GateSeen},
		{"a=0.49 uncommitted", mod(func(r *ClaimResult) { r.Calibrated = 0.49 }), GateUncommitted},
		{"a=0.30 uncommitted", mod(func(r *ClaimResult) { r.Calibrated = 0.30 }), GateUncommitted},
		{"supported but a=0.29 discouraged", mod(func(r *ClaimResult) { r.Calibrated = 0.29 }), GateDiscouraged},
		{"a=0.15 discouraged", mod(func(r *ClaimResult) { r.Calibrated = 0.15 }), GateDiscouraged},
		{"a<0.15 refused", mod(func(r *ClaimResult) { r.Calibrated = 0.14 }), GateRefused},
		{"NEI is unresolved", mod(func(r *ClaimResult) { r.Verdict = NEI }), GateUncommitted},
		{"refuted by non-reference leans false", mod(func(r *ClaimResult) { r.Verdict = Refuted; r.ReferenceTier = false; r.Calibrated = 0.2 }), GateDiscouraged},
		{"refuted by reference is refused", mod(func(r *ClaimResult) { r.Verdict = Refuted; r.Calibrated = 0.9 }), GateRefused},
		{"judge pair disagreed toward refuted", mod(func(r *ClaimResult) { r.Verdict = NEI; r.DisagreeToRefuted = true }), GateDiscouraged},
		{"no evidence is refused", mod(func(r *ClaimResult) { r.Evidence = nil }), GateRefused},
		{"degraded never exceeds uncommitted", mod(func(r *ClaimResult) { r.Degraded = true }), GateUncommitted},
		{"failed never exceeds uncommitted", mod(func(r *ClaimResult) { r.Failed = true }), GateUncommitted},
		{"pending never exceeds uncommitted", mod(func(r *ClaimResult) { r.Pending = true }), GateUncommitted},
		{"degraded with no evidence is not refused", mod(func(r *ClaimResult) { r.Degraded = true; r.Evidence = nil }), GateUncommitted},
	}
	for _, tc := range cases {
		if got := Gate(tc.r, th); got != tc.want {
			t.Errorf("%s: got %s, want %s", tc.name, got, tc.want)
		}
	}
}

// Properties that must hold for any input: degraded checks cap at 2; a level needs
// every condition of the levels below it; the function is pure.
func TestGateProperties(t *testing.T) {
	th := DefaultThresholds()
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 5000; i++ {
		r := ClaimResult{
			Verdict:    []Verdict{Supported, Refuted, NEI}[rng.Intn(3)],
			Calibrated: rng.Float64(),
			Agreement:  rng.Intn(2) == 0, ReferenceTier: rng.Intn(2) == 0, Corroborated: rng.Intn(2) == 0, Witnessed: rng.Intn(2) == 0,
			Proof: rng.Intn(2) == 0, Certificate: rng.Intn(2) == 0, Anchored: rng.Intn(2) == 0,
			Degraded: rng.Intn(5) == 0, Failed: rng.Intn(9) == 0, Pending: rng.Intn(9) == 0,
			DisagreeToRefuted: rng.Intn(6) == 0,
		}
		if rng.Intn(8) != 0 {
			r.Evidence = one()
		}
		g := Gate(r, th)
		if g != Gate(r, th) {
			t.Fatal("Gate is not deterministic")
		}
		if (r.Degraded || r.Failed || r.Pending) && g > GateUncommitted {
			t.Fatalf("degraded check reached %s: %+v", g, r)
		}
		if g >= GateSeen && (r.Verdict != Supported || r.Calibrated < th.Uncommitted) {
			t.Fatalf("level %s without a supported verdict and a>=0.5: %+v", g, r)
		}
		if g >= GateAttested && !r.Witnessed {
			t.Fatalf("attested without witnessed provenance: %+v", r)
		}
		if g >= GateAccepted && !(r.ReferenceTier && r.Agreement && r.Corroborated && r.Calibrated >= th.Accepted) {
			t.Fatalf("accepted without its conditions: %+v", r)
		}
		if g == GateCommitted && !(r.Proof && r.Certificate && r.Anchored && r.Calibrated >= th.Committed) {
			t.Fatalf("committed without proof, certificate, anchor and a>=0.90: %+v", r)
		}
	}
}

func TestThresholdsValidate(t *testing.T) {
	if err := DefaultThresholds().Validate(); err != nil {
		t.Fatal(err)
	}
	bad := DefaultThresholds()
	bad.Accepted = 0.6 // below attested
	if bad.Validate() == nil {
		t.Fatal("a non-increasing ladder was accepted")
	}
	bad = DefaultThresholds()
	bad.Committed = 1
	if bad.Validate() == nil {
		t.Fatal("a threshold of 1 was accepted")
	}
	bad = DefaultThresholds()
	bad.Seen = 0.55
	if bad.Validate() == nil {
		t.Fatal("seen and uncommitted must meet")
	}
}

func TestResponseGateIsTheWeakestClaimAndLegacyFieldsDerive(t *testing.T) {
	th := DefaultThresholds()
	mk := func(r ClaimResult) ClaimResult { r.Gate = Gate(r, th); return r }
	good, weak := mk(full()), mk(func() ClaimResult { r := full(); r.Calibrated = 0.6; r.Witnessed = false; r.Cited = nil; return r }())
	if good.Gate != GateCommitted || weak.Gate != GateSeen {
		t.Fatalf("setup: %s %s", good.Gate, weak.Gate)
	}
	res := Result{Claims: []ClaimResult{good, weak}, Unchecked: []Claim{{Text: "i think so"}}}
	// Cited spans: good cites its only span, weak cites none.
	finish(&res)
	if res.Gate != GateSeen || res.IsAccurate || res.Refused || res.Confidence != 0.6 {
		t.Fatalf("%+v", res)
	}
	if res.UncheckedFraction < 0.33 || res.UncheckedFraction > 0.34 {
		t.Fatalf("unchecked fraction %v", res.UncheckedFraction)
	}
	if len(res.Citations) != 1 || res.Citations[0] != 0 {
		t.Fatalf("citations %v", res.Citations)
	}

	acc := Result{Claims: []ClaimResult{mk(full())}}
	finish(&acc)
	if !acc.IsAccurate || acc.Refused || !acc.Chargeable {
		t.Fatalf("%+v", acc)
	}
	ref := Result{Claims: []ClaimResult{mk(func() ClaimResult { r := full(); r.Evidence = nil; return r }())}}
	finish(&ref)
	if !ref.Refused || ref.IsAccurate {
		t.Fatalf("%+v", ref)
	}
	deg := Result{Claims: []ClaimResult{mk(full()), mk(func() ClaimResult { r := full(); r.Degraded = true; return r }())}}
	finish(&deg)
	if deg.Chargeable || deg.Gate != GateUncommitted {
		t.Fatalf("a degraded check must not be charged: %+v", deg)
	}
	empty := Result{}
	finish(&empty)
	if empty.Gate != GateUncommitted || empty.Refused || empty.IsAccurate || empty.Chargeable {
		t.Fatalf("nothing checkable: %+v", empty)
	}
}

func TestCitationsAreOffsetsAcrossClaims(t *testing.T) {
	a := ClaimResult{Evidence: []evidence.Hit{{SpanID: "a1"}, {SpanID: "a2"}}, Cited: []string{"a2"}}
	b := ClaimResult{Evidence: []evidence.Hit{{SpanID: "b1"}, {SpanID: "b2"}, {SpanID: "b3"}}, Cited: []string{"b1", "b3"}}
	res := Result{Claims: []ClaimResult{a, b}}
	finish(&res)
	want := []int{1, 2, 4}
	if len(res.Citations) != 3 || res.Citations[0] != want[0] || res.Citations[1] != want[1] || res.Citations[2] != want[2] {
		t.Fatalf("citations %v, want %v", res.Citations, want)
	}
}
