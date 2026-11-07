package routes

import (
	"bytes"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/ledger"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

func newTestOracleWithMode(t *testing.T, mode string) *oracle.Oracle {
	t.Helper()
	ownerKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate owner key: %v", err)
	}
	cfg := oracle.DefaultOracleConfig()
	cfg.Mode = mode
	cfg.OwnerPrivateKey = ownerKey.PrivateKeyHex()
	cfg.DataDir = t.TempDir()
	cfg.DBBackend = "memory"
	instance, err := oracle.NewOracle(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("new oracle: %v", err)
	}
	return instance
}

func samplePath(pattern string) string {
	if strings.HasSuffix(pattern, "/") {
		return pattern + "sample"
	}
	return pattern
}

func TestEveryRouteIsClassified(t *testing.T) {
	r := NewOracleRoutes(newTestOracleWithMode(t, oracle.ModeActive), zap.NewNop())
	seen := map[string]bool{}
	for _, route := range r.routeTable() {
		if route.class != classRead && route.class != classReplication && route.class != classMutating {
			t.Errorf("%s has no class", route.pattern)
		}
		if seen[route.pattern] {
			t.Errorf("%s is registered twice", route.pattern)
		}
		seen[route.pattern] = true
		if route.handler == nil {
			t.Errorf("%s has no handler", route.pattern)
		}
	}
	if len(seen) < 60 {
		t.Fatalf("route table has %d entries; expected the full Oracle surface", len(seen))
	}
}

// The table's standby flags and the deny-by-default allowlist must agree, or a
// standby would serve a route the policy says it should refuse (or vice versa).
func TestStandbyFlagsMatchAllowlist(t *testing.T) {
	r := NewOracleRoutes(newTestOracleWithMode(t, oracle.ModeStandby), zap.NewNop())
	for _, route := range r.routeTable() {
		method := http.MethodGet
		if route.class == classMutating {
			// A mutating route always arrives as a write; GET on some of those
			// paths is answered 405 by the handler and cannot change state.
			method = http.MethodPost
		}
		got := StandbyServes(method, samplePath(route.pattern))
		if got != route.standby {
			t.Errorf("%s (%s): StandbyServes=%v but table says standby=%v", route.pattern, method, got, route.standby)
		}
	}
}

func TestStandbyRefusesEveryMutatingRoute(t *testing.T) {
	r := NewOracleRoutes(newTestOracleWithMode(t, oracle.ModeStandby), zap.NewNop())
	mux := http.NewServeMux()
	r.RegisterRoutes(mux)

	internal := map[string]bool{}
	for _, p := range r.InternalRoutePatterns() {
		internal[p] = true
	}
	for _, route := range r.routeTable() {
		if route.class != classMutating {
			continue
		}
		if internal[route.pattern] {
			// Internal-only routes are not in the public table; they are
			// refused with 403 by the public-side guard, not 503 standby.
			continue
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, route.pattern, strings.NewReader("{}")))
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s: status = %d, want 503 in standby", route.pattern, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "oracle standby") {
			t.Errorf("%s: body = %q, want the standby refusal", route.pattern, rec.Body.String())
		}
	}
}

func TestStandbyServesItsLedger(t *testing.T) {
	r := NewOracleRoutes(newTestOracleWithMode(t, oracle.ModeStandby), zap.NewNop())
	mux := http.NewServeMux()
	r.RegisterRoutes(mux)

	for _, path := range []string{
		"/oracle/v3/health",
		"/oracle/v3/status",
		"/oracle/v3/consensus/status",
		"/oracle/v3/mmr/root",
		"/oracle/v3/checkpoints/validation-chain",
		"/oracle/v3/registry/chain-1",
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code == http.StatusServiceUnavailable {
			t.Errorf("%s: standby refused a route it must serve: %s", path, rec.Body.String())
		}
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/oracle/v3/status", nil))
	var status map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status["mode"] != oracle.ModeStandby {
		t.Errorf("status mode = %v, want standby", status["mode"])
	}
	for _, key := range []string{"head_seq", "lag_seq", "lag_seconds", "diverged"} {
		if _, ok := status[key]; !ok {
			t.Errorf("status is missing %q", key)
		}
	}
}

func TestActiveModeDoesNotRefuse(t *testing.T) {
	r := NewOracleRoutes(newTestOracleWithMode(t, oracle.ModeActive), zap.NewNop())
	mux := http.NewServeMux()
	r.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/oracle/v3/token/mint", strings.NewReader("{}")))
	if rec.Code == http.StatusServiceUnavailable {
		t.Fatalf("active Oracle refused a route: %s", rec.Body.String())
	}
}

// A path nobody classified must be refused by the outer guard, not served and
// not silently 404'd: this is what makes the policy deny-by-default for routes
// added after this change.
func TestGuardRefusesUnclassifiedPaths(t *testing.T) {
	mux := http.NewServeMux()
	r := NewOracleRoutes(newTestOracleWithMode(t, oracle.ModeStandby), zap.NewNop())
	r.RegisterRoutes(mux)
	mux.HandleFunc("/oracle/v3/some/future/route", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	guarded := Guard(oracle.ModeStandby, mux)
	rec := httptest.NewRecorder()
	guarded.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/oracle/v3/some/future/route", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unclassified path status = %d, want 503", rec.Code)
	}

	allowed := httptest.NewRecorder()
	guarded.ServeHTTP(allowed, httptest.NewRequest(http.MethodGet, "/oracle/v3/mmr/root", nil))
	if allowed.Code != http.StatusOK {
		t.Fatalf("guard refused an allowlisted path: status %d", allowed.Code)
	}
}

// Defense-in-depth for D10: the rotation route must refuse an authority-chain
// author rotation at the HTTP boundary, before any registry write is attempted.
func TestHandleRegistryRotateRejectsAuthorityChain_D10(t *testing.T) {
	r := NewOracleRoutes(newTestOracleWithMode(t, oracle.ModeActive), zap.NewNop())
	mux := http.NewServeMux()
	r.RegisterRoutes(mux)

	body := `{"chain_id":"authority/operator1","authors":[{"address":"0xabc","pubkey":"","weight":1}],"quorum_numer":1,"quorum_denom":1}`
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/oracle/v3/registry/rotate", strings.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "authority chain author rotation is deprecated (D10)") {
		t.Fatalf("body = %q, want the D10 rejection", rec.Body.String())
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// TestVerifiersRegisterNotOnPublicListenerButInternal enforces DEC-1 rule 2: the
// v3 verifiers/register enrollment route must not be a public route. It is served
// ONLY on the internal unix socket (RegisterInternalRoutes) and is refused with
// 403 on the public listener that KNIRVGATEWAY proxies.
func TestVerifiersRegisterNotOnPublicListenerButInternal(t *testing.T) {
	o := newTestOracleWithMode(t, oracle.ModeActive)
	t.Setenv("KNIRV_VERIFIER_REGISTRATION_TOKEN", "tok")

	pub := ethcrypto.CompressPubkey(&mustECDSA(t).PublicKey)
	addr, err := knirvsigning.Address(pub, knirvsigning.DefaultAddressPrefix)
	if err != nil {
		t.Fatalf("address from pub: %v", err)
	}
	body := mustJSON(t, map[string]any{"verifier_id": addr, "public_key": pub})

	// Public listener (the mux KNIRVGATEWAY proxies).
	publicMux := http.NewServeMux()
	NewOracleRoutes(o, zap.NewNop()).RegisterRoutes(publicMux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oracle/v3/verifiers/register", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	publicMux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("public listener status = %d, want 403 (route must be blocked publicly)", rec.Code)
	}
	// The public listener must not have registered the verifier.
	if o.VerifierCount() != 0 {
		t.Fatalf("verifier registered through the public listener: %d", o.VerifierCount())
	}

	// Internal socket (RegisterInternalRoutes).
	internalMux := http.NewServeMux()
	NewOracleRoutes(o, zap.NewNop()).RegisterInternalRoutes(internalMux)

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/oracle/v3/verifiers/register", bytes.NewReader(body))
	req2.Header.Set("Authorization", "Bearer tok")
	internalMux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("internal listener status = %d, want 200: %s", rec2.Code, rec2.Body.String())
	}
	if o.VerifierCount() != 1 {
		t.Fatalf("verifier not registered through internal socket: %d", o.VerifierCount())
	}
}

// bondedAuthorityRegistrationWithHold builds a signed bonded authority/* registration
// backed by a valid D14 enrolment hold owned by the bond owner, mirroring what the
// KNIRV_CORP CLI posts for D14 enrolment. It is the fixture for the real-route tests.
func bondedAuthorityRegistrationWithHold(t *testing.T, o *oracle.Oracle, kp *crypto.KeyPair) (types.ChainRegistration, string) {
	t.Helper()
	tok := o.GetNRNToken()
	addr := kp.Address.String()
	opAddr, err := types.AddressFromString(addr)
	if err != nil {
		t.Fatalf("address: %v", err)
	}
	if _, err := tok.Mint(opAddr, big.NewInt(2_000_000_000)); err != nil {
		t.Fatalf("mint: %v", err)
	}
	if _, err := o.GetEconomicsEngine().GetStakingManager().Stake(opAddr, big.NewInt(1_000_000_000)); err != nil {
		t.Fatalf("stake: %v", err)
	}
	holdID := "bond:" + addr
	expiry := time.Now().Add(time.Hour)
	hold := knirvsigning.BondHold{
		HoldID:         holdID,
		Operator:       addr,
		Amount:         "10000",
		RootRegistryID: tok.RootRegistryID(),
		ExpiresAtUnix:  uint64(expiry.Unix()),
	}
	signed, err := knirvsigning.SignMessage(ethcrypto.FromECDSA(kp.PrivateKey), knirvsigning.MessageEnvelope{
		SchemaVersion: knirvsigning.MessageSchemaVersion,
		Domain:        knirvsigning.RegistryDomain,
		Purpose:       knirvsigning.PurposeBondHold,
		ChainID:       tok.ChainID(),
		Nonce:         "bond-hold:" + holdID,
		IssuedAtUnix:  time.Now().Unix(),
		ExpiresAtUnix: int64(hold.ExpiresAtUnix),
		Payload:       mustJSON(t, hold),
	})
	if err != nil {
		t.Fatalf("sign hold: %v", err)
	}
	if err := tok.Hold(holdID, hold, signed, time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	cpk := ethcrypto.CompressPubkey(kp.PublicKey)
	reg := types.ChainRegistration{
		ChainID:     "authority/" + addr,
		Authors:     []types.RegisteredAuthor{{Address: addr, PubKey: cpk, Weight: 1}},
		QuorumNumer: 1,
		QuorumDenom: 1,
		LastHeight:  0,
		Bond:        10000,
		BondOwner:   addr,
		ProofWindow: 256,
		HoldID:      holdID,
	}
	if err := types.SignRegistration(&reg, kp.PrivateKey); err != nil {
		t.Fatalf("sign registration: %v", err)
	}
	return reg, holdID
}

// TestRouteRegisterAuthorityOmittedHoldRejectedHTTP is the missing real-route test
// (round 4): a signed bonded authority registration with hold_id omitted, delivered as
// a JSON body to /oracle/v3/registry/register, must return 4xx and leave the registry
// unmutated.
func TestRouteRegisterAuthorityOmittedHoldRejectedHTTP(t *testing.T) {
	o := newTestOracleWithMode(t, oracle.ModeActive)
	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	addr := kp.Address.String()
	cpk := ethcrypto.CompressPubkey(kp.PublicKey)
	reg := types.ChainRegistration{
		ChainID:     "authority/" + addr,
		Authors:     []types.RegisteredAuthor{{Address: addr, PubKey: cpk, Weight: 1}},
		QuorumNumer: 1,
		QuorumDenom: 1,
		Bond:        10000,
		BondOwner:   addr,
		ProofWindow: 256,
	}
	if err := types.SignRegistration(&reg, kp.PrivateKey); err != nil {
		t.Fatal(err)
	}
	body := bytes.NewReader(mustJSON(t, reg))
	req := httptest.NewRequest(http.MethodPost, "/oracle/v3/registry/register", body)
	req.Header.Set("Content-Type", "application/json")
	r := NewOracleRoutes(o, zap.NewNop())
	rec := httptest.NewRecorder()
	r.handleRegistryRegister(rec, req)
	if rec.Code < 400 {
		t.Fatalf("expected 4xx for omitted hold, got %d", rec.Code)
	}
	if _, ok := o.GetChainRegistration(reg.ChainID); ok {
		t.Fatal("chain registered despite omitted hold id at the route")
	}
}

// TestRouteRegisterAuthorityValidHoldAcceptedHTTP is the matching positive route test:
// a signed bonded authority registration carrying a valid D14 enrolment hold is
// accepted (200) and registered through the real handler, and the hold is consumed.
func TestRouteRegisterAuthorityValidHoldAcceptedHTTP(t *testing.T) {
	o := newTestOracleWithMode(t, oracle.ModeActive)
	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	reg, holdID := bondedAuthorityRegistrationWithHold(t, o, kp)
	body := bytes.NewReader(mustJSON(t, reg))
	req := httptest.NewRequest(http.MethodPost, "/oracle/v3/registry/register", body)
	req.Header.Set("Content-Type", "application/json")
	r := NewOracleRoutes(o, zap.NewNop())
	rec := httptest.NewRecorder()
	r.handleRegistryRegister(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid hold, got %d: %s", rec.Code, rec.Body.String())
	}
	got, ok := o.GetChainRegistration(reg.ChainID)
	if !ok {
		t.Fatal("chain not registered through the route")
	}
	if got.Bond != 10000 {
		t.Fatalf("registered bond = %d, want 10000", got.Bond)
	}
	h, herr := o.GetNRNToken().GetHold(holdID)
	if herr != nil {
		t.Fatalf("get hold: %v", herr)
	}
	if h.Status != ledger.BondStatusReleased {
		t.Fatalf("hold %q status = %q, want released (consumed through route)", holdID, h.Status)
	}
}
