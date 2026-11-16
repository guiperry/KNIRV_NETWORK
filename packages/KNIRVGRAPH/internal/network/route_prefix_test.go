package network

import (
	"KNIRVGRAPH/internal/locator"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// newLocatorServer builds the real router NewRPCServerWithNRV registers, so the
// test covers the production route table, not a hand-built copy of it.
func newLocatorServer(t *testing.T, logger *zap.Logger) (http.Handler, *locator.LocatorSystem) {
	t.Helper()
	sys := locator.NewLocatorSystem("test-peer", nil)
	rpc := NewRPCServerWithNRV(nil, sys, logger, 0)
	return rpc.server.Handler, sys
}

func serve(t *testing.T, h http.Handler, method, path string, body interface{}) (int, string) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

// normalizeList sorts a JSON array of objects by "id" so two reads of the same
// store compare equal even though the store iterates in map order.
func normalizeList(body string) string {
	var items []map[string]interface{}
	if err := json.Unmarshal([]byte(body), &items); err != nil {
		return body
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i]["id"] == nil || (items[j]["id"] != nil && items[i]["id"].(string) < items[j]["id"].(string))
	})
	out, _ := json.Marshal(items)
	return string(out)
}

func signedCommitBody(t *testing.T, errorType string) map[string]interface{} {
	t.Helper()
	errCtx := map[string]interface{}{"exit_code": float64(1)}
	root, err := errorNodeCommitRoot(errorNodeCommitClaim{ErrorType: errorType, Description: "boom", Context: errCtx, Severity: 1})
	if err != nil {
		t.Fatal(err)
	}
	return map[string]interface{}{
		"schema_version": ErrorNodeCommitSchema, "error_type": errorType, "description": "boom",
		"context": errCtx, "severity": 1, "error_root": root,
		"signer_id": "cli", "signing_key_id": "cli", "signature": root,
	}
}

// TestLocatorRoutesCanonicalAndLegacyAgree checks that every locator route answers
// the same under /graph (canonical) and /nrv (deprecated alias).
func TestLocatorRoutesCanonicalAndLegacyAgree(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "secret")
	h, sys := newLocatorServer(t, zap.NewNop())

	if code, body := serve(t, h, "POST", "/graph/errors/commit", signedCommitBody(t, "canonical-crash")); code != http.StatusOK {
		t.Fatalf("POST /graph/errors/commit: %d %s", code, body)
	}
	if code, body := serve(t, h, "POST", "/nrv/errors/commit", signedCommitBody(t, "legacy-crash")); code != http.StatusOK {
		t.Fatalf("POST /nrv/errors/commit: %d %s", code, body)
	}
	if n := len(sys.GetAllErrorNodes()); n != 2 {
		t.Fatalf("both prefixes must create nodes, have %d", n)
	}

	reads := []string{
		"/errors", "/skills", "/vectors", "/vectors/resolve/abc",
		"/skills/for-error/crash", "/contexts", "/ideas", "/errors/missing/tests",
	}
	for _, p := range reads {
		canonCode, canonBody := serve(t, h, "GET", "/graph"+p, nil)
		legacyCode, legacyBody := serve(t, h, "GET", "/nrv"+p, nil)
		if canonCode != legacyCode || normalizeList(canonBody) != normalizeList(legacyBody) {
			t.Fatalf("GET %s: /graph gave %d %q, /nrv gave %d %q", p, canonCode, canonBody, legacyCode, legacyBody)
		}
		// mux's own not-found body means no route matched at all.
		if canonBody == "404 page not found\n" {
			t.Fatalf("GET /graph%s is not routed", p)
		}
	}
}

// TestLegacyNRVAliasLogsDeprecationOnce checks the alias warns once per route and
// the canonical prefix never warns.
func TestLegacyNRVAliasLogsDeprecationOnce(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	h, _ := newLocatorServer(t, zap.New(core))

	serve(t, h, "GET", "/graph/skills", nil)
	serve(t, h, "GET", "/graph/skills", nil)
	if n := logs.FilterMessageSnippet("deprecated route alias").Len(); n != 0 {
		t.Fatalf("canonical prefix must not warn, got %d", n)
	}

	serve(t, h, "GET", "/nrv/skills", nil)
	serve(t, h, "GET", "/nrv/skills", nil)
	if n := logs.FilterMessageSnippet("deprecated route alias").Len(); n != 1 {
		t.Fatalf("legacy alias must warn exactly once per route, got %d", n)
	}
}
