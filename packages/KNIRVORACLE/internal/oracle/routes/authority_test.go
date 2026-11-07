package routes

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/guiperry/knirv-sdk-go/authority"
	"github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle"
	"github.com/knirvcorp/knirvoracle/internal/oracle/consensus"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

var seedCoreOnce sync.Once

// Same seeding as the other authority tests: a locally built core is unpublished.
func useCore(t *testing.T) {
	t.Helper()
	seedCoreOnce.Do(func() {
		if os.Getenv("KNIRV_SDK_DATA_DIR") != "" {
			return
		}
		m, _ := filepath.Glob(filepath.Join("..", "..", "..", "..", "KNIRVSDK", "dist", "*", "knirv_core.wasm"))
		if len(m) == 0 {
			return
		}
		sort.Slice(m, func(i, j int) bool {
			a, _ := os.Stat(m[i])
			b, _ := os.Stat(m[j])
			return a.ModTime().After(b.ModTime())
		})
		dir, err := os.MkdirTemp("", "routes-authority-core-")
		if err != nil {
			return
		}
		data, err := os.ReadFile(m[0])
		v := filepath.Base(filepath.Dir(m[0]))
		if err != nil || os.MkdirAll(filepath.Join(dir, v), 0o700) != nil {
			return
		}
		if os.WriteFile(filepath.Join(dir, v, "knirv_core.wasm"), data, 0o600) == nil {
			os.Setenv("KNIRV_SDK_DATA_DIR", dir)
		}
	})
}

func key32(n byte) []byte { k := make([]byte, 32); k[31] = n; return k }

func toWire(r *authority.Record) wireAuthorityRecord {
	h := func(b []byte) string { return hex.EncodeToString(b) }
	c := r.SessionCert
	return wireAuthorityRecord{Tier: uint8(r.Tier), Domain: r.Domain, Scope: r.Scope, Operator: r.Operator, Epoch: r.Epoch,
		CanonicalActionHash: h(r.CanonicalActionHash[:]), PrevRecordHash: h(r.PrevRecordHash[:]), Header: h(r.Header[:]),
		Nonce: r.Nonce, Hash: h(r.Hash[:]), TargetBits: r.TargetBits, WitnessKind: r.WitnessKind, DeviceClaim: r.DeviceClaim,
		SessionCert: wireAuthorityCert{SessionPub: h(c.SessionPub), Operator: c.Operator, Epoch: c.Epoch, NotBefore: c.NotBefore,
			NotAfter: c.NotAfter, Scope: c.Scope, Entropy: c.Entropy, Signature: h(c.Signature)},
		Signature: h(r.Signature)}
}

type authFixture struct {
	oracle   *oracle.Oracle
	mux      *http.ServeMux
	operator string
	chainID  string
	record   func(scope string) *authority.Record
}

func newAuthFixture(t *testing.T) *authFixture {
	t.Helper()
	useCore(t)
	t.Setenv("KNIRV_VERIFIER_REGISTRATION_TOKEN", "tok")
	o := newTestOracleWithMode(t, oracle.ModeActive)
	// The fixture exercises HTTP routes, not the production minimum-bond
	// default; run it in observe mode so a zero-bond authority chain registers.
	if err := o.SetAuthorityMinBond(0); err != nil {
		t.Fatal(err)
	}
	longTerm, _ := signing.NewLocalSigner(key32(1))
	session, _ := signing.NewLocalSigner(key32(2))
	cert, err := authority.SignSessionCert(context.Background(), authority.SessionCert{SessionPub: session.PublicKey(),
		Operator: longTerm.Address(), Epoch: 7, NotBefore: 1_750_000_000, NotAfter: 1_750_086_400, Scope: longTerm.Address(), Entropy: "csprng"}, longTerm)
	if err != nil {
		t.Fatal(err)
	}
	key, _ := ethcrypto.ToECDSA(key32(1))
	f := &authFixture{oracle: o, operator: longTerm.Address(), chainID: "authority/" + longTerm.Address()}
	reg := types.ChainRegistration{ChainID: f.chainID, QuorumNumer: 1, QuorumDenom: 1, ProofWindow: 1800,
		Authors: []types.RegisteredAuthor{{Address: f.operator, PubKey: ethcrypto.CompressPubkey(&key.PublicKey), Weight: 1}}}
	if err := types.SignRegistration(&reg, key); err != nil {
		t.Fatal(err)
	}
	if err := o.RegisterChain(&reg); err != nil {
		t.Fatal(err)
	}
	f.record = func(scope string) *authority.Record {
		r := &authority.Record{Tier: authority.TierT1, Domain: "evidence.commit", Scope: scope, Operator: f.operator, Epoch: 7,
			CanonicalActionHash: sha256.Sum256([]byte("a")), PrevRecordHash: sha256.Sum256([]byte("g")),
			TargetBits: 0x207fffff, WitnessKind: authority.WitnessSoftware, SessionCert: cert}
		if err := r.Seal(context.Background(), session, 1_750_000_000); err != nil {
			t.Fatal(err)
		}
		return r
	}
	f.mux = http.NewServeMux()
	NewOracleRoutes(o, zap.NewNop()).RegisterRoutes(f.mux)
	return f
}

func (f *authFixture) post(t *testing.T, path, token string, body any) (int, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestAuthorityReportRouteGoodAndViolatingRecord(t *testing.T) {
	f := newAuthFixture(t)
	asOf := int64(1_750_000_100)

	code, out := f.post(t, "/oracle/v3/authority/report", "tok", map[string]any{
		"chain_id": f.chainID, "record": toWire(f.record(f.operator)), "as_of_unix": asOf})
	if code != 200 || out["verified"] != true || out["violation"] != nil {
		t.Fatalf("good record: %d %v", code, out)
	}

	code, out = f.post(t, "/oracle/v3/authority/report", "tok", map[string]any{
		"chain_id": f.chainID, "record": toWire(f.record("knirv1someoneelse")), "as_of_unix": asOf})
	v, _ := out["violation"].(map[string]any)
	if code != 200 || out["verified"] != false || v == nil || v["kind"] != "scope_violation" {
		t.Fatalf("scope violation: %d %v", code, out)
	}
	ev := f.oracle.GetConsensusEngine().PendingEvidence()
	if len(ev) != 1 {
		t.Fatalf("pending evidence = %d", len(ev))
	}
	if e, ok := ev[0].(consensus.AuthorityViolationEvidence); !ok || e.Operator != f.operator {
		t.Fatalf("evidence %#v", ev[0])
	}
}

func TestAuthorityReportRouteRejectsBadRequests(t *testing.T) {
	f := newAuthFixture(t)
	good := toWire(f.record(f.operator))
	path := "/oracle/v3/authority/report"

	if code, _ := f.post(t, path, "", map[string]any{"chain_id": f.chainID, "record": good}); code != 403 {
		t.Fatalf("no token: %d", code)
	}
	if code, _ := f.post(t, path, "wrong", map[string]any{"chain_id": f.chainID, "record": good}); code != 403 {
		t.Fatalf("wrong token: %d", code)
	}
	t.Setenv("KNIRV_VERIFIER_REGISTRATION_TOKEN", "")
	if code, _ := f.post(t, path, "", map[string]any{"chain_id": f.chainID, "record": good}); code != 403 {
		t.Fatalf("unconfigured token must refuse: %d", code)
	}
	t.Setenv("KNIRV_VERIFIER_REGISTRATION_TOKEN", "tok")

	bad := good
	bad.Hash = "zz"
	if code, _ := f.post(t, path, "tok", map[string]any{"chain_id": f.chainID, "record": bad}); code != 400 {
		t.Fatalf("bad hex: %d", code)
	}
	bad = good
	bad.Hash = hex.EncodeToString(make([]byte, 31))
	if code, _ := f.post(t, path, "tok", map[string]any{"chain_id": f.chainID, "record": bad}); code != 400 {
		t.Fatalf("short hash: %d", code)
	}
	if code, _ := f.post(t, path, "tok", map[string]any{"chain_id": "knirvchain", "record": good}); code != 400 {
		t.Fatalf("non-authority chain: %d", code)
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET: %d", rec.Code)
	}
	if len(f.oracle.AuthorityViolations(f.chainID)) != 0 {
		t.Fatal("a rejected request recorded a violation")
	}
}

func TestVerifierRegisterEnrolsAuthorityOperatorOnlyWhenAsked(t *testing.T) {
	f := newAuthFixture(t)
	// verifiers/register is an INTERNAL-ONLY route; it is served on the
	// internal unix socket via RegisterInternalRoutes, not on the public mux.
	internalMux := http.NewServeMux()
	NewOracleRoutes(f.oracle, zap.NewNop()).RegisterInternalRoutes(internalMux)
	pub := ethcrypto.CompressPubkey(&mustECDSA(t).PublicKey)
	addr, _ := signing.Address(pub, signing.DefaultAddressPrefix)
	path := "/oracle/v3/verifiers/register"

	postInternal := func(t *testing.T, token string, body any) (int, map[string]any) {
		t.Helper()
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		internalMux.ServeHTTP(rec, req)
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}

	// Unchanged behaviour for an ordinary verifier.
	if code, out := postInternal(t, "tok", map[string]any{"verifier_id": addr, "public_key": pub}); code != 200 {
		t.Fatalf("plain: %d %v", code, out)
	}
	if got := f.oracle.VerifierCount(); got != 1 {
		t.Fatalf("verifiers = %d", got)
	}
	// A bad authority_kind changes nothing.
	pub2 := ethcrypto.CompressPubkey(&mustECDSA(t).PublicKey)
	addr2, _ := signing.Address(pub2, signing.DefaultAddressPrefix)
	if code, _ := postInternal(t, "tok", map[string]any{"verifier_id": addr2, "public_key": pub2, "authority_kind": "king"}); code != 400 {
		t.Fatalf("bad kind: %d", code)
	}
	if got := f.oracle.VerifierCount(); got != 1 {
		t.Fatalf("a rejected request still registered a verifier (%d)", got)
	}
	if code, out := postInternal(t, "tok", map[string]any{"verifier_id": addr2, "public_key": pub2, "authority_kind": "root", "authority_scopes": []string{addr2, "knirv1other"}}); code != 200 {
		t.Fatalf("enrol: %d %v", code, out)
	}
}

func mustECDSA(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ethcrypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	return k
}
