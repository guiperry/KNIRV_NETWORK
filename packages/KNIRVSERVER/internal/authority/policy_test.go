package authority

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/guiperry/knirv-sdk-go/authority"
)

func TestDefaultPolicyIsValidAndConservative(t *testing.T) {
	p := DefaultPolicy()
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		tier sdk.Tier
		kind string
		want bool
	}{
		{sdk.TierT1, KindRoot, true}, {sdk.TierT1, KindBoot, true},
		{sdk.TierT2, KindRoot, true}, {sdk.TierT2, KindBoot, false},
		{sdk.TierT3, KindRoot, false}, {sdk.TierT3, KindBoot, false}, // T3 disabled (D6)
	} {
		if got, why := p.Allows(tc.tier, tc.kind); got != tc.want {
			t.Errorf("Allows(%s,%s) = %v (%s), want %v", tc.tier, tc.kind, got, why, tc.want)
		}
	}
	for _, tier := range []sdk.Tier{sdk.TierT1, sdk.TierT2, sdk.TierT3} {
		if p.TargetBits(tier) != sdk.TierTargetBits(tier) {
			t.Errorf("%s target differs from the spec", tier)
		}
	}
}

func TestPolicyValidateRefusesWhatVerifyWouldReject(t *testing.T) {
	mutate := map[string]func(*Policy){
		"wrong target":        func(p *Policy) { r := p.Tiers["T2"]; r.TargetBits = 0x207fffff; p.Tiers["T2"] = r },
		"boot signs T2":       func(p *Policy) { r := p.Tiers["T2"]; r.Signers = []string{KindRoot, KindBoot}; p.Tiers["T2"] = r },
		"boot signs T3":       func(p *Policy) { r := p.Tiers["T3"]; r.Signers = []string{KindBoot}; p.Tiers["T3"] = r },
		"unknown signer":      func(p *Policy) { r := p.Tiers["T1"]; r.Signers = []string{"client"}; p.Tiers["T1"] = r },
		"T0 row":              func(p *Policy) { p.Tiers["T0"] = TierPolicy{} },
		"missing tier":        func(p *Policy) { delete(p.Tiers, "T3") },
		"zero epoch":          func(p *Policy) { p.Epoch = 0 },
		"ttl too short":       func(p *Policy) { p.CertTTLSeconds = 10 },
		"ttl too long":        func(p *Policy) { p.CertTTLSeconds = 8 * 24 * 3600 },
		"unsupported version": func(p *Policy) { p.Version = 2 },
	}
	for name, f := range mutate {
		t.Run(name, func(t *testing.T) {
			p := DefaultPolicy()
			f(&p)
			if err := p.Validate(); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestLoadPolicy(t *testing.T) {
	dir := t.TempDir()
	if p, err := LoadPolicy(filepath.Join(dir, "policy.json")); err != nil || p.Hash() != DefaultPolicy().Hash() {
		t.Fatalf("missing file should give the default policy, got %v", err)
	}
	write := func(body string) string {
		path := filepath.Join(dir, "p.json")
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	if _, err := LoadPolicy(write(`{"version":1,"surprise":true}`)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field must be refused, got %v", err)
	}
	if _, err := LoadPolicy(write(`not json`)); err == nil {
		t.Fatal("garbage must be refused, not replaced by the default")
	}
}

func TestPolicyHashTracksEveryField(t *testing.T) {
	base := DefaultPolicy().Hash()
	p := DefaultPolicy()
	r := p.Tiers["T3"]
	r.Enabled = true
	p.Tiers["T3"] = r
	if p.Hash() == base {
		t.Fatal("enabling a tier did not change the hash")
	}
	q := DefaultPolicy()
	q.Epoch = 2
	if q.Hash() == base {
		t.Fatal("changing the epoch did not change the hash")
	}
}
