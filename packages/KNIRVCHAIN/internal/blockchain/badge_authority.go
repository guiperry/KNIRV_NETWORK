package blockchain

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	knirv "github.com/guiperry/knirv-sdk-go"
	"github.com/guiperry/knirv-sdk-go/signing"
)

// Authority records on badge credentials (Task 2d.1).
//
// A mint or revocation may carry the authority record that authorised it
// (KNIRV_NETWORK/docs/AUTHORITY_SPEC.md). Every node re-verifies it, from the
// transaction alone, when the block is accepted, and when the request is first seen.
//
// OBSERVE MODE ONLY. Verification outcomes are logged and counted. Nothing in this
// file can reject a request or a block, return an error to a caller, or panic: a
// verifier that is broken, missing its core, or fed garbage costs a counter increment.
// Enforcement is Task 2d.3, after a soak.
//
// What "verifies" means today (PROVISIONAL, spec section 10, D12): the record is
// internally consistent against the operator entry CARRIED WITH IT. Specifically:
//   - the record is bound to this exact action (hash, domain, tier, and a
//     canonical_action_hash recomputed here from the NFT or revocation fields);
//   - the carried registry entry's long-term key hashes to the record's operator
//     address, so the entry cannot claim another operator's identity;
//   - the Rust core accepts the certificate chain, the session signature, the
//     witness and the tier/scope rules, as of the block time.
//
// What it does NOT prove: that the operator is authorised on the network. That needs
// the Oracle registry as the source of the entry (Task 2d.2). The outcomes say so by
// being counted under the surface name only; there is no "authorised" outcome.
const (
	badgeAuthorityMint   = "badge.mint"
	badgeAuthorityRevoke = "badge.revoke"

	// Expected tiers and domains. PROVISIONAL (D13): mint = T1, revoke = T2.
	badgeAuthorityMintTier   = 1
	badgeAuthorityRevokeTier = 2

	// maxBadgeAuthorityRecordBytes bounds what a transaction may carry. A real
	// record is about 1.5 KB.
	maxBadgeAuthorityRecordBytes = 64 << 10
)

// BadgeAuthorityAttachment is the record and the operator entry needed to verify it.
type BadgeAuthorityAttachment struct {
	Record   json.RawMessage         `json:"record"`
	Registry *BadgeAuthorityOperator `json:"registry,omitempty"`
}

// UnmarshalJSON decodes the attachment leniently. The endpoints that carry it reject
// unknown fields everywhere else, which is right for the credential itself, but an
// attachment is observe-mode evidence: a field this version does not know must not
// turn into the one thing observe mode promises never to do, refuse the request.
func (a *BadgeAuthorityAttachment) UnmarshalJSON(data []byte) error {
	type lenient BadgeAuthorityAttachment
	var l lenient
	if err := json.Unmarshal(data, &l); err != nil {
		return err
	}
	*a = BadgeAuthorityAttachment(l)
	return nil
}

// BadgeAuthorityOperator is the operator's verifier-registry entry in the core's
// wire shape.
type BadgeAuthorityOperator struct {
	Address         string           `json:"address"`
	Kind            string           `json:"kind"`
	LongTermPub     string           `json:"long_term_pub"`
	Scopes          []string         `json:"scopes"`
	RetiredSessions map[string]int64 `json:"retired_sessions,omitempty"`
}

// sanitizeBadgeAuthority makes an incoming attachment safe to store in a block. It
// never fails the request: anything unusable is dropped and reported as the verdict.
func sanitizeBadgeAuthority(hash string, att *BadgeAuthorityAttachment) (string, *BadgeAuthorityAttachment, string) {
	if hash == "" && att == nil {
		return "", nil, ""
	}
	if hash != "" && !sha256Hex.MatchString(hash) {
		return "", nil, "malformed_hash"
	}
	if att != nil && len(att.Record) > maxBadgeAuthorityRecordBytes {
		return hash, nil, "attachment_too_large"
	}
	if att != nil && len(att.Record) == 0 {
		att = nil
	}
	return hash, att, ""
}

// badgeAuthorityClock is the time a request is verified against when it is first
// accepted. A variable only so tests can pin it; a record is valid for its session
// certificate's window (about a day), so wall-clock time matters (spec section 5.2,
// open D3). Block validation uses the block's own time instead.
var badgeAuthorityClock = time.Now

// ---- counters ----

type badgeAuthorityStats struct {
	mu     sync.Mutex
	counts map[string]uint64
}

var badgeAuthorityCounters = &badgeAuthorityStats{counts: map[string]uint64{}}

// BadgeAuthorityCounters returns a copy of the observe-mode outcome counters, keyed
// "<surface>/<outcome>" (for example "badge.mint/ok", "badge.mint/ErrScope",
// "badge.revoke/record_missing").
func BadgeAuthorityCounters() map[string]uint64 {
	badgeAuthorityCounters.mu.Lock()
	defer badgeAuthorityCounters.mu.Unlock()
	out := make(map[string]uint64, len(badgeAuthorityCounters.counts))
	for k, v := range badgeAuthorityCounters.counts {
		out[k] = v
	}
	return out
}

func countBadgeAuthority(surface, outcome string) {
	badgeAuthorityCounters.mu.Lock()
	badgeAuthorityCounters.counts[surface+"/"+outcome]++
	badgeAuthorityCounters.mu.Unlock()
}

// ---- canonical action ----

// badgeMintAction and badgeRevokeAction re-derive the action object the record must
// be bound to. They mirror backend_server's MintAction/RevokeAction exactly; the
// shared vector testvectors/authority/badge_actions.json fixes all three.
func badgeMintAction(nft BadgeCredentialNFT) map[string]string {
	return map[string]string{
		"action":           badgeAuthorityMint,
		"credential_id":    nft.CredentialID,
		"badge_id":         nft.BadgeID,
		"holder_agent_id":  nft.HolderAgentID,
		"kind":             nft.Kind,
		"suite_hash":       nft.SuiteHash,
		"certificate_hash": nft.CertificateHash,
	}
}

func badgeRevokeAction(rev badgeCredentialRevocation) map[string]string {
	return map[string]string{
		"action":        badgeAuthorityRevoke,
		"credential_id": rev.CredentialID,
		"token_id":      rev.TokenID,
		"reason":        rev.Reason,
	}
}

// canonicalBadgeAction is the conservative canonical JSON of an all-string object:
// sorted ASCII keys, no whitespace, RFC 8785 string escapes. It is the same profile
// as KNIRVSERVER's internal/authority/canonical.go restricted to string values,
// which is all a badge action contains (spec section 10, D9).
func canonicalBadgeAction(action map[string]string) []byte {
	keys := make([]string, 0, len(action))
	for k := range action {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		writeCanonicalString(&b, k)
		b.WriteByte(':')
		writeCanonicalString(&b, action[k])
	}
	b.WriteByte('}')
	return b.Bytes()
}

func writeCanonicalString(b *bytes.Buffer, s string) {
	const hexDigits = "0123456789abcdef"
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				b.WriteString(`\u00`)
				b.WriteByte(hexDigits[r>>4])
				b.WriteByte(hexDigits[r&0xf])
			} else {
				b.WriteString(string(r))
			}
		}
	}
	b.WriteByte('"')
}

// ---- verification ----

// badgeAuthorityCore verifies a record in the Rust core. It is a variable so tests
// can simulate a core that is missing or broken.
var badgeAuthorityCore = verifyInCore

var (
	coreOnce   sync.Once
	coreClient *knirv.Client
)

type coreVerdict struct {
	Valid   bool   `json:"valid"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// verifyInCore hands the record's own bytes to the core's authority.verify_record.
// Passing the raw JSON (not a re-encoded copy) is what makes this verification
// "from the block alone": what was committed is what gets checked.
func verifyInCore(record json.RawMessage, op BadgeAuthorityOperator, asOf time.Time) (coreVerdict, error) {
	coreOnce.Do(func() { coreClient = knirv.NewClient() })
	request, err := json.Marshal(map[string]any{
		"record":   record,
		"registry": map[string]any{"operators": map[string]BadgeAuthorityOperator{op.Address: op}},
		"now_unix": asOf.Unix(),
	})
	if err != nil {
		return coreVerdict{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := coreClient.Execute(ctx, "authority.verify_record", request)
	if err != nil {
		// A core error here is either "the module could not be loaded" or a
		// malformed-input error from the core itself; the caller counts both.
		return coreVerdict{}, err
	}
	var v coreVerdict
	if err := json.Unmarshal(out, &v); err != nil {
		return coreVerdict{}, err
	}
	return v, nil
}

type badgeAuthorityExpectation struct {
	surface string
	tier    uint8
	action  map[string]string
}

// observeBadgeAuthority verifies one attachment and records the outcome. It returns
// the outcome label and ALWAYS returns: it recovers from any panic, including the
// SDK's own panics on a missing core.
func observeBadgeAuthority(want badgeAuthorityExpectation, hash string, att *BadgeAuthorityAttachment, asOf time.Time) (outcome string) {
	defer func() {
		if r := recover(); r != nil {
			outcome = "verifier_panic"
			log.Printf("[badge-authority] observe: %s verifier panicked and was contained: %v", want.surface, r)
		}
		countBadgeAuthority(want.surface, outcome)
		if outcome != "ok" && outcome != "no_record" {
			log.Printf("[badge-authority] observe: %s record %q: %s (observe mode, nothing is rejected)", want.surface, hash, outcome)
		}
	}()
	return evaluateBadgeAuthority(want, hash, att, asOf)
}

func evaluateBadgeAuthority(want badgeAuthorityExpectation, hash string, att *BadgeAuthorityAttachment, asOf time.Time) string {
	if hash == "" && att == nil {
		return "no_record"
	}
	if att == nil || len(att.Record) == 0 {
		return "record_missing"
	}
	var rec struct {
		Tier                uint8  `json:"tier"`
		Domain              string `json:"domain"`
		Operator            string `json:"operator"`
		Hash                string `json:"hash"`
		CanonicalActionHash string `json:"canonical_action_hash"`
	}
	if err := json.Unmarshal(att.Record, &rec); err != nil {
		return "record_malformed"
	}
	if hash != "" && rec.Hash != hash {
		return "hash_mismatch"
	}
	if rec.Domain != want.surface {
		return "binding_domain"
	}
	if rec.Tier != want.tier {
		return "binding_tier"
	}
	sum := sha256.Sum256(canonicalBadgeAction(want.action))
	if rec.CanonicalActionHash != hex.EncodeToString(sum[:]) {
		return "binding_action"
	}
	op := att.Registry
	if op == nil || op.Address == "" || op.LongTermPub == "" {
		return "registry_missing"
	}
	pub, err := hex.DecodeString(op.LongTermPub)
	if err != nil {
		return "registry_malformed"
	}
	address, err := signing.Address(pub, signing.DefaultAddressPrefix)
	if err != nil {
		return "registry_malformed"
	}
	if address != rec.Operator || address != op.Address {
		return "registry_operator_mismatch"
	}
	if op.Kind != "root" && op.Kind != "boot" {
		return "registry_malformed"
	}
	verdict, err := badgeAuthorityCore(att.Record, *op, asOf)
	if err != nil {
		return "verifier_error"
	}
	if !verdict.Valid {
		if verdict.Code == "" {
			return "ErrOther"
		}
		return strings.TrimSpace(verdict.Code)
	}
	return "ok"
}

func observeBadgeMint(nft BadgeCredentialNFT, asOf time.Time) string {
	return observeBadgeAuthority(badgeAuthorityExpectation{surface: badgeAuthorityMint, tier: badgeAuthorityMintTier, action: badgeMintAction(nft)},
		nft.AuthorityRecordHash, nft.Authority, asOf)
}

func observeBadgeRevoke(rev badgeCredentialRevocation, asOf time.Time) string {
	return observeBadgeAuthority(badgeAuthorityExpectation{surface: badgeAuthorityRevoke, tier: badgeAuthorityRevokeTier, action: badgeRevokeAction(rev)},
		rev.AuthorityRecordHash, rev.Authority, asOf)
}

// handleBadgeAuthorityStats serves GET /api/v1/badge-credentials/authority/stats: the
// observe-mode outcome counters, behind the same internal token as minting.
func (bcs *BlockchainServer) handleBadgeAuthorityStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireInternalToken(w, r, "badge authority statistics") {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"mode": "observe", "counters": BadgeAuthorityCounters()})
}
