package authority

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"

	sdk "github.com/guiperry/knirv-sdk-go/authority"
)

// Node kinds, derived from which key file the provider finds (root.key or boot.key).
const (
	KindRoot = "root"
	KindBoot = "boot"
)

// TierPolicy is one tier's row of policy.json. Tier policy is data
// (AUTHORITY_SPEC.md section 3); changing it is a T3 action, and 2b.1 has no T3 flow, so
// the service only READS the file at start. An edit takes effect on restart and
// the policy hash in /authority/status shows which policy is live.
type TierPolicy struct {
	Enabled bool `json:"enabled"`
	// TargetBits must equal the spec's fixed target for the tier. Verify demands
	// the exact value (section 3), so a different number could only produce records that
	// do not verify. It is data so the T3 policy-change flow can carry it later.
	TargetBits uint32   `json:"target_bits"`
	Signers    []string `json:"signers"`
}

// Policy is the whole policy.json.
type Policy struct {
	Version int `json:"version"`
	// Epoch is the session epoch stamped into every record and certificate.
	// Until the session-key ceremony (Task 2c) the long-term key signs directly,
	// so the epoch never advances.
	Epoch uint64 `json:"epoch"`
	// CertTTLSeconds is the validity window of the self-issued session
	// certificate. Records can only be verified inside it (section 5.2, open D3).
	CertTTLSeconds int64                 `json:"cert_ttl_seconds"`
	Tiers          map[string]TierPolicy `json:"tiers"`
	// BreakGlassTiers lists the tiers that may be signed directly by the long-term
	// key while no session key is available (Task 2c.1 step 9). Empty by default:
	// with no live session the service refuses. Only "T1" is permitted, because
	// the spec makes T2 and T3 fail closed at certificate expiry (section 4.1).
	// PROVISIONAL: "T0/T1 follow policy" (plan step 10) leaves the default open;
	// the conservative reading is that nothing signs without a session unless the
	// operator opts in here.
	BreakGlassTiers []string `json:"break_glass_tiers,omitempty"`
}

// DefaultPolicy is used when no policy.json exists.
//
// Conservative readings, marked here because the owner has not decided them:
//   - T3 is disabled. It needs the highest difficulty plus multi-root
//     co-attestation, which is not on the wire (spec section 10, D6).
//   - A boot node never signs T2 or T3 (same reason, D6). Validate refuses a
//     policy that says otherwise, so an edited file cannot widen this.
func DefaultPolicy() Policy {
	return Policy{
		Version:        1,
		Epoch:          1,
		CertTTLSeconds: 24 * 60 * 60,
		Tiers: map[string]TierPolicy{
			"T1": {Enabled: true, TargetBits: sdk.TierTargetBits(sdk.TierT1), Signers: []string{KindRoot, KindBoot}},
			"T2": {Enabled: true, TargetBits: sdk.TierTargetBits(sdk.TierT2), Signers: []string{KindRoot}},
			"T3": {Enabled: false, TargetBits: sdk.TierTargetBits(sdk.TierT3), Signers: []string{KindRoot}},
		},
	}
}

func tierName(t sdk.Tier) string { return t.String() }

// Validate refuses a policy the verifier would not accept records from.
func (p Policy) Validate() error {
	if p.Version != 1 {
		return fmt.Errorf("policy version %d is not supported", p.Version)
	}
	if p.Epoch == 0 {
		return errors.New("policy epoch must be greater than zero")
	}
	if p.CertTTLSeconds < 300 || p.CertTTLSeconds > 7*24*60*60 {
		return fmt.Errorf("cert_ttl_seconds %d is outside 300..604800", p.CertTTLSeconds)
	}
	for _, name := range p.BreakGlassTiers {
		if name != "T1" {
			return fmt.Errorf("break_glass_tiers may list only T1 (got %q): T2 and T3 fail closed without a session", name)
		}
	}
	for name := range p.Tiers {
		if name != "T1" && name != "T2" && name != "T3" {
			return fmt.Errorf("unknown tier %q (T0 carries no record)", name)
		}
	}
	for _, tier := range []sdk.Tier{sdk.TierT1, sdk.TierT2, sdk.TierT3} {
		row, ok := p.Tiers[tierName(tier)]
		if !ok {
			return fmt.Errorf("tier %s is missing", tier)
		}
		if want := sdk.TierTargetBits(tier); row.TargetBits != want {
			return fmt.Errorf("tier %s target_bits %#x differs from the spec's %#x; Verify would reject every record", tier, row.TargetBits, want)
		}
		for _, who := range row.Signers {
			if who != KindRoot && who != KindBoot {
				return fmt.Errorf("tier %s: unknown signer kind %q", tier, who)
			}
			if tier != sdk.TierT1 && who == KindBoot {
				return fmt.Errorf("tier %s: a boot node may not sign it without root co-attestation (spec section 10, D6)", tier)
			}
		}
	}
	return nil
}

// Allows reports whether a node of kind may issue records at tier.
func (p Policy) Allows(tier sdk.Tier, kind string) (bool, string) {
	row, ok := p.Tiers[tierName(tier)]
	if !ok || !row.Enabled {
		return false, fmt.Sprintf("tier %s is disabled by policy", tier)
	}
	if slices.Contains(row.Signers, kind) {
		return true, ""
	}
	return false, fmt.Sprintf("a %s node may not sign %s", kind, tier)
}

// BreakGlassAllowed reports whether tier may be signed by the long-term key alone.
func (p Policy) BreakGlassAllowed(tier sdk.Tier) bool {
	return slices.Contains(p.BreakGlassTiers, tier.String())
}

// TargetBits is the policy target for tier (validated equal to the spec's).
func (p Policy) TargetBits(tier sdk.Tier) uint32 { return p.Tiers[tierName(tier)].TargetBits }

// Hash is SHA-256 over the policy's canonical form, shown in /authority/status.
func (p Policy) Hash() string {
	names := make([]string, 0, len(p.Tiers))
	for name := range p.Tiers {
		names = append(names, name)
	}
	sort.Strings(names)
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "v%d|e%d|ttl%d", p.Version, p.Epoch, p.CertTTLSeconds)
	for _, name := range names {
		row := p.Tiers[name]
		signers := append([]string(nil), row.Signers...)
		sort.Strings(signers)
		fmt.Fprintf(&buf, "|%s:%t:%d:%v", name, row.Enabled, row.TargetBits, signers)
	}
	if len(p.BreakGlassTiers) > 0 { // appended only when set, so existing policy hashes do not change
		bg := append([]string(nil), p.BreakGlassTiers...)
		sort.Strings(bg)
		fmt.Fprintf(&buf, "|bg%v", bg)
	}
	sum := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(sum[:])
}

// LoadPolicy reads path. A missing file means DefaultPolicy. A file that exists
// but is invalid is an error: the caller must not silently fall back, because a
// broken policy is a configuration problem the operator has to see.
func LoadPolicy(path string) (Policy, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultPolicy(), nil
	}
	if err != nil {
		return Policy{}, fmt.Errorf("read policy: %w", err)
	}
	var p Policy
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return Policy{}, fmt.Errorf("decode policy %s: %w", path, err)
	}
	if err := p.Validate(); err != nil {
		return Policy{}, fmt.Errorf("policy %s: %w", path, err)
	}
	return p, nil
}
