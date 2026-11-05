package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"KNIRVCHAIN/internal/utils"

	"github.com/gorilla/mux"
	"github.com/guiperry/knirv-sdk-go/signing"
)

// ---- shared vectors ----

func vectorFile(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "KNIRVSDK", "testvectors", "authority", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type actionVector struct {
	Name                string            `json:"name"`
	Domain              string            `json:"domain"`
	Tier                uint8             `json:"tier"`
	Action              map[string]string `json:"action"`
	Canonical           string            `json:"canonical"`
	CanonicalActionHash string            `json:"canonical_action_hash"`
}

func loadActionVectors(t *testing.T) []actionVector {
	var doc struct {
		Cases []actionVector `json:"cases"`
	}
	if err := json.Unmarshal(vectorFile(t, "badge_actions.json"), &doc); err != nil || len(doc.Cases) == 0 {
		t.Fatalf("badge_actions.json: %v", err)
	}
	return doc.Cases
}

type recordFixture struct {
	AsOf          int64                  `json:"as_of"`
	RegistryEntry BadgeAuthorityOperator `json:"registry_entry"`
	Cases         []struct {
		Name   string          `json:"name"`
		Hash   string          `json:"hash"`
		Record json.RawMessage `json:"record"`
	} `json:"cases"`
}

func loadRecordFixture(t *testing.T) recordFixture {
	var fx recordFixture
	if err := json.Unmarshal(vectorFile(t, "badge_records.json"), &fx); err != nil || len(fx.Cases) < 3 {
		t.Fatalf("badge_records.json: %v", err)
	}
	return fx
}

// nftFromVector builds the NFT exactly as the chain would from a mint request that
// carries the vector's action, with the given authority attachment.
func nftFromVector(v actionVector, hash string, att *BadgeAuthorityAttachment) BadgeCredentialNFT {
	return BadgeCredentialNFT{
		SchemaVersion: badgeCredentialMintSchema, CredentialID: v.Action["credential_id"], BadgeID: v.Action["badge_id"], BadgeName: "Leak Fixer",
		BadgeType: "skill", HolderAgentID: v.Action["holder_agent_id"], Kind: v.Action["kind"], ExecutionMode: "agent",
		SuiteHash: v.Action["suite_hash"], CertificateHash: v.Action["certificate_hash"], IssuedAt: time.Unix(1_750_000_000, 0).UTC(),
		AuthorityRecordHash: hash, Authority: att,
	}
}

func mintTx(t *testing.T, nft BadgeCredentialNFT) *Transaction {
	t.Helper()
	token, err := badgeCredentialTokenID(nft)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(nft)
	tx := NewTransaction(utils.BLOCKCHAIN_ADDRESS, token, 0, data)
	tx.Type = txTypeBadgeCredentialMint
	return tx
}

func attachmentFor(fx recordFixture, i int) (string, *BadgeAuthorityAttachment) {
	entry := fx.RegistryEntry
	return fx.Cases[i].Hash, &BadgeAuthorityAttachment{Record: fx.Cases[i].Record, Registry: &entry}
}

// outcomeDelta runs fn and returns the counters it added.
func outcomeDelta(fn func()) map[string]uint64 {
	before := BadgeAuthorityCounters()
	fn()
	delta := map[string]uint64{}
	for k, v := range BadgeAuthorityCounters() {
		if v != before[k] {
			delta[k] = v - before[k]
		}
	}
	return delta
}

func validateBlock(t *testing.T, tx *Transaction, at time.Time) (error, map[string]uint64) {
	t.Helper()
	var err error
	delta := outcomeDelta(func() {
		err = (&BlockchainStruct{}).validateBadgeCredentialsInBlock(&Block{Timestamp: at.Unix(), Transactions: []*Transaction{tx}})
	})
	return err, delta
}

// ---- tests ----

func TestChainDerivationOfTheActionMatchesTheSharedVectors(t *testing.T) {
	for _, v := range loadActionVectors(t) {
		t.Run(v.Name, func(t *testing.T) {
			var action map[string]string
			switch v.Domain {
			case "badge.mint":
				action = badgeMintAction(nftFromVector(v, "", nil))
			case "badge.revoke":
				action = badgeRevokeAction(badgeCredentialRevocation{CredentialID: v.Action["credential_id"], TokenID: v.Action["token_id"], Reason: v.Action["reason"]})
			default:
				t.Fatalf("unknown domain %q", v.Domain)
			}
			if len(action) != len(v.Action) {
				t.Fatalf("derived %v, vector %v", action, v.Action)
			}
			for k, want := range v.Action {
				if action[k] != want {
					t.Fatalf("%s = %q, vector %q", k, action[k], want)
				}
			}
			if got := string(canonicalBadgeAction(action)); got != v.Canonical {
				t.Fatalf("canonical\n got %s\nwant %s", got, v.Canonical)
			}
			sum := sha256.Sum256(canonicalBadgeAction(action))
			if hex.EncodeToString(sum[:]) != v.CanonicalActionHash {
				t.Fatal("canonical action hash differs from the vector")
			}
		})
	}
}

// The decisive check of the task: a record the real authority service issued is
// re-verified by a chain that has never seen it, from the transaction in a block.
func TestARealRecordIsReverifiedFromTheBlockAlone(t *testing.T) {
	fx, vectors := loadRecordFixture(t), loadActionVectors(t)
	hash, att := attachmentFor(fx, 0)
	tx := mintTx(t, nftFromVector(vectors[0], hash, att))
	asOf := time.Unix(fx.AsOf, 0)

	// A brand-new chain value: no database, no pool, no prior blocks, nothing but the tx.
	err, delta := validateBlock(t, tx, asOf.Add(5*time.Second))
	if err != nil {
		t.Fatalf("a block carrying a record must validate: %v", err)
	}
	if delta["badge.mint/ok"] != 1 || len(delta) != 1 {
		t.Fatalf("expected exactly one badge.mint/ok, got %v", delta)
	}

	// The second mint vector (a purchased credential, empty suite hash) verifies too.
	hash2, att2 := attachmentFor(fx, 1)
	if _, delta := validateBlock(t, mintTx(t, nftFromVector(vectors[1], hash2, att2)), asOf); delta["badge.mint/ok"] != 1 {
		t.Fatalf("purchased credential: %v", delta)
	}

	// The revocation vector verifies at the revocation level (its token id is a
	// vector constant, so this is checked against the derived revocation directly).
	hash3, att3 := attachmentFor(fx, 2)
	rev := badgeCredentialRevocation{CredentialID: vectors[2].Action["credential_id"], TokenID: vectors[2].Action["token_id"], Reason: vectors[2].Action["reason"], AuthorityRecordHash: hash3, Authority: att3}
	if got := observeBadgeRevoke(rev, asOf); got != "ok" {
		t.Fatalf("revocation record: %s", got)
	}
}

func TestEveryKindOfBadRecordIsCountedAndNothingIsRejected(t *testing.T) {
	fx, vectors := loadRecordFixture(t), loadActionVectors(t)
	asOf := time.Unix(fx.AsOf, 0).Add(time.Minute)
	goodHash, good := attachmentFor(fx, 0)
	mint := vectors[0]

	tamper := func(mutate func(map[string]any)) json.RawMessage {
		var rec map[string]any
		if err := json.Unmarshal(good.Record, &rec); err != nil {
			t.Fatal(err)
		}
		mutate(rec)
		raw, _ := json.Marshal(rec)
		return raw
	}
	other, _ := signing.NewLocalSigner(append(make([]byte, 31), 99))
	clone := func(op BadgeAuthorityOperator) *BadgeAuthorityOperator { return &op }

	cases := []struct {
		name    string
		nft     func() BadgeCredentialNFT
		at      time.Time
		outcome string
	}{
		{"no record at all", func() BadgeCredentialNFT { return nftFromVector(mint, "", nil) }, asOf, "no_record"},
		{"hash without the record", func() BadgeCredentialNFT { return nftFromVector(mint, goodHash, nil) }, asOf, "record_missing"},
		{"record bytes that are not JSON", func() BadgeCredentialNFT {
			return nftFromVector(mint, goodHash, &BadgeAuthorityAttachment{Record: json.RawMessage(`"not an object"`), Registry: good.Registry})
		}, asOf, "record_malformed"},
		{"hash differs from the record", func() BadgeCredentialNFT { return nftFromVector(mint, strings.Repeat("0", 64), good) }, asOf, "hash_mismatch"},
		{"record belongs to a different action", func() BadgeCredentialNFT {
			n := nftFromVector(mint, goodHash, good)
			n.CertificateHash = strings.Repeat("e", 64) // the credential changed, the record did not
			return n
		}, asOf, "binding_action"},
		{"a revocation record presented as a mint", func() BadgeCredentialNFT {
			h, a := attachmentFor(fx, 2)
			return nftFromVector(mint, h, a)
		}, asOf, "binding_domain"},
		{"registry entry missing", func() BadgeCredentialNFT {
			return nftFromVector(mint, goodHash, &BadgeAuthorityAttachment{Record: good.Record})
		}, asOf, "registry_missing"},
		{"registry entry for a different key", func() BadgeCredentialNFT {
			op := *good.Registry
			op.LongTermPub = hex.EncodeToString(other.PublicKey())
			return nftFromVector(mint, goodHash, &BadgeAuthorityAttachment{Record: good.Record, Registry: clone(op)})
		}, asOf, "registry_operator_mismatch"},
		{"registry kind is not root or boot", func() BadgeCredentialNFT {
			op := *good.Registry
			op.Kind = "client"
			return nftFromVector(mint, goodHash, &BadgeAuthorityAttachment{Record: good.Record, Registry: clone(op)})
		}, asOf, "registry_malformed"},
		{"the certificate window has passed", func() BadgeCredentialNFT { return nftFromVector(mint, goodHash, good) }, asOf.Add(48 * time.Hour), "ErrExpiredSession"},
		{"the record's signature was altered", func() BadgeCredentialNFT {
			raw := tamper(func(m map[string]any) {
				sig := m["signature"].(string)
				flipped := "0"
				if sig[0] == '0' {
					flipped = "1"
				}
				m["signature"] = flipped + sig[1:]
			})
			return nftFromVector(mint, goodHash, &BadgeAuthorityAttachment{Record: raw, Registry: good.Registry})
		}, asOf, "ErrRecordSignature"},
		{"the witness was altered", func() BadgeCredentialNFT {
			raw := tamper(func(m map[string]any) { m["nonce"] = m["nonce"].(float64) + 1 })
			return nftFromVector(mint, goodHash, &BadgeAuthorityAttachment{Record: raw, Registry: good.Registry})
		}, asOf, "ErrHeader"},
		{"the scope was altered after signing", func() BadgeCredentialNFT {
			raw := tamper(func(m map[string]any) { m["epoch"] = m["epoch"].(float64) + 1 })
			return nftFromVector(mint, goodHash, &BadgeAuthorityAttachment{Record: raw, Registry: good.Registry})
		}, asOf, "ErrSessionSignature"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err, delta := validateBlock(t, mintTx(t, tc.nft()), tc.at)
			if err != nil {
				t.Fatalf("observe mode must never reject a block over an authority record: %v", err)
			}
			if delta["badge.mint/"+tc.outcome] != 1 || len(delta) != 1 {
				t.Fatalf("want exactly badge.mint/%s, got %v", tc.outcome, delta)
			}
		})
	}
}

func TestABrokenOrMissingVerifierCostsACounterNotABlock(t *testing.T) {
	fx, vectors := loadRecordFixture(t), loadActionVectors(t)
	hash, att := attachmentFor(fx, 0)
	tx := mintTx(t, nftFromVector(vectors[0], hash, att))
	asOf := time.Unix(fx.AsOf, 0)
	orig := badgeAuthorityCore
	defer func() { badgeAuthorityCore = orig }()

	badgeAuthorityCore = func(json.RawMessage, BadgeAuthorityOperator, time.Time) (coreVerdict, error) {
		return coreVerdict{}, errors.New("cannot load the SDK core: no network")
	}
	if err, delta := validateBlock(t, tx, asOf); err != nil || delta["badge.mint/verifier_error"] != 1 {
		t.Fatalf("a missing core: %v %v", err, delta)
	}
	badgeAuthorityCore = func(json.RawMessage, BadgeAuthorityOperator, time.Time) (coreVerdict, error) {
		panic("the SDK panicked")
	}
	if err, delta := validateBlock(t, tx, asOf); err != nil || delta["badge.mint/verifier_panic"] != 1 {
		t.Fatalf("a panicking verifier must be contained: %v %v", err, delta)
	}
	badgeAuthorityCore = func(json.RawMessage, BadgeAuthorityOperator, time.Time) (coreVerdict, error) {
		return coreVerdict{Valid: false}, nil // invalid with no code
	}
	if err, delta := validateBlock(t, tx, asOf); err != nil || delta["badge.mint/ErrOther"] != 1 {
		t.Fatalf("%v %v", err, delta)
	}
}

func authorityRouter(bcs *BlockchainServer) http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/badge-credentials/mint", bcs.handleBadgeCredentialMint)
	r.HandleFunc("/api/v1/badge-credentials/revoke", bcs.handleBadgeCredentialRevoke)
	r.HandleFunc("/api/v1/badge-credentials/authority/stats", bcs.handleBadgeAuthorityStats)
	r.HandleFunc("/api/v1/badge-credentials/{credential_id}", bcs.handleBadgeCredentialGet).Methods(http.MethodGet)
	return r
}

func mintRequestFromVector(v actionVector, hash string, att *BadgeAuthorityAttachment) map[string]any {
	req := map[string]any{
		"schema_version": badgeCredentialMintSchema, "credential_id": v.Action["credential_id"], "badge_id": v.Action["badge_id"],
		"badge_name": "Leak Fixer", "holder_agent_id": v.Action["holder_agent_id"], "kind": v.Action["kind"], "execution_mode": "agent",
		"suite_hash": v.Action["suite_hash"], "certificate_hash": v.Action["certificate_hash"], "issued_at": time.Unix(1_750_000_000, 0).UTC(),
	}
	if hash != "" {
		req["authority_record_hash"] = hash
	}
	if att != nil {
		req["authority"] = att
	}
	return req
}

func TestMintEndpointVerifiesAtAcceptanceAndStoresTheHash(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "secret")
	fx, vectors := loadRecordFixture(t), loadActionVectors(t)
	badgeAuthorityClock = func() time.Time { return time.Unix(fx.AsOf, 0).Add(10 * time.Second) }
	defer func() { badgeAuthorityClock = time.Now }()

	bcs := newBadgeCredentialServer(t)
	h := authorityRouter(bcs)
	hash, att := attachmentFor(fx, 0)

	var rec *httptest.ResponseRecorder
	delta := outcomeDelta(func() {
		rec = post(t, h, "/api/v1/badge-credentials/mint", "secret", mintRequestFromVector(vectors[0], hash, att))
	})
	if rec.Code != http.StatusAccepted && rec.Code != http.StatusOK {
		t.Fatalf("mint: %d %s", rec.Code, rec.Body)
	}
	if delta["badge.mint/ok"] != 1 {
		t.Fatalf("expected an ok verdict at acceptance: %v", delta)
	}
	var receipt badgeCredentialReceipt
	json.Unmarshal(rec.Body.Bytes(), &receipt)
	if receipt.AuthorityRecordHash != hash || receipt.AuthorityVerdict != "ok" {
		t.Fatalf("receipt: %+v", receipt)
	}
	// The transaction in the pool carries the full record, so a peer can re-verify it.
	var nft BadgeCredentialNFT
	if err := json.Unmarshal(bcs.BlockchainPtr.TransactionPool[0].Data, &nft); err != nil || nft.AuthorityRecordHash != hash || nft.Authority == nil {
		t.Fatalf("the pooled transaction lacks the record: %+v %v", nft, err)
	}
	// Marshalling compacts a RawMessage, so compare the JSON, not the whitespace.
	var want, got bytes.Buffer
	if json.Compact(&want, att.Record) != nil || json.Compact(&got, nft.Authority.Record) != nil || want.String() != got.String() {
		t.Fatal("the attached record changed on its way into the transaction")
	}
	// And the token id commits to it: altering the attached record changes the token.
	altered := nft
	altered.Authority = &BadgeAuthorityAttachment{Record: json.RawMessage(`{"hash":"x"}`), Registry: nft.Authority.Registry}
	a, _ := badgeCredentialTokenID(nft)
	b, _ := badgeCredentialTokenID(altered)
	if a == b {
		t.Fatal("the token id does not commit to the attached record")
	}
}

func TestMintEndpointNeverRefusesBecauseOfTheAttachment(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "secret")
	fx, vectors := loadRecordFixture(t), loadActionVectors(t)
	hash, att := attachmentFor(fx, 0)
	base := func(cert string) actionVector {
		v := vectors[0]
		v.Action = map[string]string{"credential_id": "cred-x", "badge_id": "badge-9", "holder_agent_id": "dve-node-42", "kind": "earned", "suite_hash": vectors[0].Action["suite_hash"], "certificate_hash": cert}
		return v
	}
	cases := map[string]struct {
		hash    string
		att     *BadgeAuthorityAttachment
		verdict string
		stored  bool // whether authority data ends up in the NFT
	}{
		"malformed hash":              {"not-a-hash", att, "malformed_hash", false},
		"oversized record":            {hash, &BadgeAuthorityAttachment{Record: json.RawMessage(`"` + strings.Repeat("x", maxBadgeAuthorityRecordBytes+1) + `"`), Registry: att.Registry}, "attachment_too_large", false}, // dropped, hash kept
		"record that is not a record": {hash, &BadgeAuthorityAttachment{Record: json.RawMessage(`[1,2,3]`), Registry: att.Registry}, "record_malformed", true},
		"nothing":                     {"", nil, "no_record", false},
	}
	n := 0
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			bcs := newBadgeCredentialServer(t)
			n++
			cert := strings.Repeat(string(rune('a'+n)), 64)
			rec := post(t, authorityRouter(bcs), "/api/v1/badge-credentials/mint", "secret", mintRequestFromVector(base(cert), tc.hash, tc.att))
			if rec.Code != http.StatusAccepted && rec.Code != http.StatusOK {
				t.Fatalf("an attachment problem must not refuse the mint: %d %s", rec.Code, rec.Body)
			}
			var receipt badgeCredentialReceipt
			json.Unmarshal(rec.Body.Bytes(), &receipt)
			if receipt.AuthorityVerdict != tc.verdict {
				t.Fatalf("verdict %q, want %q", receipt.AuthorityVerdict, tc.verdict)
			}
			var nft BadgeCredentialNFT
			json.Unmarshal(bcs.BlockchainPtr.TransactionPool[0].Data, &nft)
			if tc.stored != (nft.Authority != nil) {
				t.Fatalf("attachment stored = %v, want %v", nft.Authority != nil, tc.stored)
			}
		})
	}
}

func TestUnknownFieldsInsideTheAttachmentDoNotRefuseTheMint(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "secret")
	vectors := loadActionVectors(t)
	req := mintRequestFromVector(vectors[0], strings.Repeat("a", 64), nil)
	raw, _ := json.Marshal(req)
	body := strings.Replace(string(raw), "}", `,"authority":{"record":{"hash":"x"},"registry":null,"from_the_future":true}}`, 1)
	httpReq := httptest.NewRequest(http.MethodPost, "/api/v1/badge-credentials/mint", strings.NewReader(body))
	httpReq.Header.Set("X-KNIRV-Internal-Token", "secret")
	rec := httptest.NewRecorder()
	authorityRouter(newBadgeCredentialServer(t)).ServeHTTP(rec, httpReq)
	if rec.Code != http.StatusAccepted && rec.Code != http.StatusOK {
		t.Fatalf("a newer attachment shape must not refuse the mint: %d %s", rec.Code, rec.Body)
	}
}

func TestCredentialsWithoutARecordAreByteForByteWhatTheyWereBefore(t *testing.T) {
	nft := BadgeCredentialNFT{SchemaVersion: badgeCredentialMintSchema, CredentialID: "c", BadgeID: "b", BadgeName: "n", BadgeType: "skill",
		HolderAgentID: "h", Kind: "earned", ExecutionMode: "agent", CertificateHash: hexA, IssuedAt: time.Unix(1_750_000_000, 0).UTC()}
	raw, _ := json.Marshal(nft)
	if strings.Contains(string(raw), "authority") {
		t.Fatalf("an NFT without a record grew authority fields, which would change its token id: %s", raw)
	}
	want := `{"schema_version":"knirv.badge-credential.v1","credential_id":"c","badge_id":"b","badge_name":"n","badge_type":"skill","holder_agent_id":"h","kind":"earned","execution_mode":"agent","certificate_hash":"` + hexA + `","issued_at":"2025-06-15T15:06:40Z"}`
	if string(raw) != want {
		t.Fatalf("serialisation changed:\n got %s\nwant %s", raw, want)
	}
	// Old chains understand it, new chains read it: decode with the strict decoder.
	if _, ok := decodeBadgeCredentialMint(raw); !ok {
		t.Fatal("the strict decoder rejects a plain credential")
	}
}

func TestStatsRouteIsTokenGated(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "secret")
	h := authorityRouter(newBadgeCredentialServer(t))
	get := func(token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/badge-credentials/authority/stats", nil)
		if token != "" {
			req.Header.Set("X-KNIRV-Internal-Token", token)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	if get("").Code != http.StatusUnauthorized || get("wrong").Code != http.StatusUnauthorized {
		t.Fatal("the stats must require the internal token")
	}
	rec := get("secret")
	var body struct {
		Mode     string            `json:"mode"`
		Counters map[string]uint64 `json:"counters"`
	}
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &body) != nil || body.Mode != "observe" || body.Counters == nil {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
}
