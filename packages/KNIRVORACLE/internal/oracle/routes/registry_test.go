package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/knirvcorp/knirvoracle/internal/oracle"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
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

	for _, route := range r.routeTable() {
		if route.class != classMutating {
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
