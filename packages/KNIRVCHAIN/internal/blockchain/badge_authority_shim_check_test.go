package blockchain

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"KNIRVCHAIN/internal/utils"
)

// Test-only shim check for Task 2d.1 (observe-mode authority wiring, CHAIN side).
//
// The decisive behaviour: a badge mint carries a record hash that CHAIN re-verifies
// from the block alone. These checks run the block path (validateBadgeCredentialsInBlock)
// with the real SDK core, and with a test-only recording shim in front of it, so a
// regression that hands the verifier something other than the committed bytes, skips
// verification, or drops the core's verdict is caught here.
//
// Nothing here changes production code. The shim is a package variable swapped and
// restored by each test.

// blockedMintFixture returns the fixture mint as the block carries it: the transaction,
// its decoded NFT (exactly what validateBadgeCredentialsInBlock reads back from
// tx.Data), and a block time inside the record's validity window.
func blockedMintFixture(t *testing.T) (*Transaction, BadgeCredentialNFT, time.Time) {
	t.Helper()
	fx, vectors := loadRecordFixture(t), loadActionVectors(t)
	hash, att := attachmentFor(fx, 0)
	tx := mintTx(t, nftFromVector(vectors[0], hash, att))
	nft, ok := decodeBadgeCredentialMint(tx.Data)
	if !ok || nft.Authority == nil {
		t.Fatal("the block's mint does not decode to a credential with an authority record")
	}
	return tx, nft, time.Unix(fx.AsOf, 0).Add(5 * time.Second)
}

// sameCompactJSON reports whether two JSON documents are equal ignoring whitespace.
func sameCompactJSON(t *testing.T, a, b json.RawMessage) bool {
	t.Helper()
	var ca, cb bytes.Buffer
	if err := json.Compact(&ca, a); err != nil {
		t.Fatalf("first record is not JSON: %v", err)
	}
	if err := json.Compact(&cb, b); err != nil {
		t.Fatalf("second record is not JSON: %v", err)
	}
	return ca.String() == cb.String()
}

// The real core, reached through the block path, verifies the block's record. An
// "ok" here also proves the SDK core loaded: a missing core shows as verifier_error.
func TestShimCheckRealCoreVerifiesTheBlockRecord(t *testing.T) {
	tx, _, at := blockedMintFixture(t)

	err, delta := validateBlock(t, tx, at)
	if err != nil {
		t.Fatalf("a block carrying a valid record must validate: %v", err)
	}
	if len(delta) != 1 || delta["badge.mint/ok"] != 1 {
		t.Fatalf("the real core must verify the block's record as ok, got %v (verifier_error means the SDK core did not load)", delta)
	}
}

// The verifier is handed exactly the record bytes that were committed in the block,
// not a copy rebuilt from the request.
func TestShimCheckCoreReceivesTheCommittedRecordBytes(t *testing.T) {
	tx, nft, at := blockedMintFixture(t)

	orig := badgeAuthorityCore
	defer func() { badgeAuthorityCore = orig }()
	var seen []json.RawMessage
	badgeAuthorityCore = func(rec json.RawMessage, op BadgeAuthorityOperator, asOf time.Time) (coreVerdict, error) {
		seen = append(seen, append(json.RawMessage(nil), rec...))
		return orig(rec, op, asOf)
	}

	err, delta := validateBlock(t, tx, at)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(seen) != 1 {
		t.Fatalf("core consulted %d times, want exactly once", len(seen))
	}
	if !sameCompactJSON(t, seen[0], nft.Authority.Record) {
		t.Fatalf("the verifier was handed bytes other than the block's record:\n got %s\nwant %s", seen[0], nft.Authority.Record)
	}
	if len(delta) != 1 || delta["badge.mint/ok"] != 1 {
		t.Fatalf("want exactly badge.mint/ok, got %v", delta)
	}
}

// The request body that verifyInCore builds is checked, not only the record the block
// hands it: the record in the body must be the committed bytes, the registry must carry
// exactly the record's own operator, and now_unix must be the block time. The action
// name is checked too. The core's reply is canned, so this does not depend on it.
func TestShimCheckVerifyInCoreRequestBodyIsFromTheBlock(t *testing.T) {
	tx, nft, at := blockedMintFixture(t)
	if nft.Authority.Registry == nil {
		t.Fatal("fixture mint has no registry operator")
	}

	orig := coreExecute
	defer func() { coreExecute = orig }()
	var actions []string
	var bodies [][]byte
	coreExecute = func(_ context.Context, action string, payload []byte) ([]byte, error) {
		actions = append(actions, action)
		bodies = append(bodies, append([]byte(nil), payload...))
		return []byte(`{"valid":true}`), nil
	}

	err, delta := validateBlock(t, tx, at)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(bodies) != 1 {
		t.Fatalf("core executed %d times, want exactly once", len(bodies))
	}
	if actions[0] != "authority.verify_record" {
		t.Fatalf("core action %q, want authority.verify_record", actions[0])
	}

	var req struct {
		Record   json.RawMessage `json:"record"`
		Registry struct {
			Operators map[string]json.RawMessage `json:"operators"`
		} `json:"registry"`
		NowUnix int64 `json:"now_unix"`
	}
	if err := json.Unmarshal(bodies[0], &req); err != nil {
		t.Fatalf("request body is not JSON: %v", err)
	}
	if !sameCompactJSON(t, req.Record, nft.Authority.Record) {
		t.Fatalf("request record is not the committed record:\n got %s\nwant %s", req.Record, nft.Authority.Record)
	}
	if req.NowUnix != at.Unix() {
		t.Fatalf("now_unix %d, want the block time %d", req.NowUnix, at.Unix())
	}
	if len(req.Registry.Operators) != 1 {
		t.Fatalf("registry carries %d operators, want exactly the record's own", len(req.Registry.Operators))
	}
	wantOp, err := json.Marshal(nft.Authority.Registry)
	if err != nil {
		t.Fatal(err)
	}
	gotOp, ok := req.Registry.Operators[nft.Authority.Registry.Address]
	if !ok || !sameCompactJSON(t, gotOp, wantOp) {
		t.Fatalf("registry operator is not the record's own operator:\n got %s\nwant %s", gotOp, wantOp)
	}
	if len(delta) != 1 || delta["badge.mint/ok"] != 1 {
		t.Fatalf("want exactly badge.mint/ok, got %v", delta)
	}
}

// A record altered inside the block is re-verified from the block and counted, and
// the block still validates (observe mode). The token id commits to the record, so the
// altered transaction is re-addressed to the token its altered content derives.
func TestShimCheckAlteredRecordInBlockIsCountedNotRejected(t *testing.T) {
	_, nft, at := blockedMintFixture(t)

	var rec map[string]any
	if err := json.Unmarshal(nft.Authority.Record, &rec); err != nil {
		t.Fatal(err)
	}
	sig, ok := rec["signature"].(string)
	if !ok || sig == "" {
		t.Fatal("fixture record has no signature field")
	}
	flipped := "0"
	if sig[0] == '0' {
		flipped = "1"
	}
	rec["signature"] = flipped + sig[1:]
	raw, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	nft.Authority = &BadgeAuthorityAttachment{Record: raw, Registry: nft.Authority.Registry}
	tokenID, err := badgeCredentialTokenID(nft)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(nft)
	if err != nil {
		t.Fatal(err)
	}
	tx := NewTransaction(utils.BLOCKCHAIN_ADDRESS, tokenID, 0, data)
	tx.Type = txTypeBadgeCredentialMint

	err, delta := validateBlock(t, tx, at)
	if err != nil {
		t.Fatalf("observe mode must not reject a block over an altered record: %v", err)
	}
	if len(delta) != 1 || delta["badge.mint/ErrRecordSignature"] != 1 {
		t.Fatalf("want exactly badge.mint/ErrRecordSignature, got %v", delta)
	}
}

// A verdict code from the core is passed through to the counter under the surface name,
// and the block still validates.
func TestShimCheckCoreVerdictCodeIsCountedUnderTheSurface(t *testing.T) {
	tx, _, at := blockedMintFixture(t)

	orig := badgeAuthorityCore
	defer func() { badgeAuthorityCore = orig }()
	badgeAuthorityCore = func(json.RawMessage, BadgeAuthorityOperator, time.Time) (coreVerdict, error) {
		return coreVerdict{Valid: false, Code: "ErrScope"}, nil
	}

	err, delta := validateBlock(t, tx, at)
	if err != nil {
		t.Fatalf("observe mode must not reject a block over a core verdict: %v", err)
	}
	if len(delta) != 1 || delta["badge.mint/ErrScope"] != 1 {
		t.Fatalf("want exactly badge.mint/ErrScope, got %v", delta)
	}
}
