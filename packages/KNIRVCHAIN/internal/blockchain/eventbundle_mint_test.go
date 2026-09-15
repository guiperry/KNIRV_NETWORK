package blockchain

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The event-bundle mint route is KNIRVCHAIN's real capability/skill minting
// entry point: KNIRVGRAPH's DRQ client posts skill nodes to it over the chain's
// Unix socket (see KNIRVGRAPH/internal/drq/knirvchain_client.go,
// mintSkillEventBundle). These tests pin the contract that makes it callable
// and safe, so "the chain has a minting entry point" is a verified claim rather
// than an assertion.
func TestEventBundleMintRouteContract(t *testing.T) {
	newServer := func() *BlockchainServer { return &BlockchainServer{testMode: true} }

	t.Run("rejects non-POST", func(t *testing.T) {
		rec := httptest.NewRecorder()
		newServer().handleEventBundleMint(rec, httptest.NewRequest(http.MethodGet, "/api/v1/event-bundles/mint", nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", rec.Code)
		}
	})

	t.Run("fails closed when no internal token is configured", func(t *testing.T) {
		// An unset token must not mean "open to everyone".
		t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/event-bundles/mint", strings.NewReader(`{}`))
		req.Header.Set("X-KNIRV-Internal-Token", "anything")
		newServer().handleEventBundleMint(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503 when minting is unconfigured", rec.Code)
		}
	})

	t.Run("rejects a wrong internal token", func(t *testing.T) {
		t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "expected-secret")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/event-bundles/mint", strings.NewReader(`{}`))
		req.Header.Set("X-KNIRV-Internal-Token", "wrong-secret")
		newServer().handleEventBundleMint(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401 for a wrong token", rec.Code)
		}
	})

	t.Run("rejects a malformed bundle with the correct token", func(t *testing.T) {
		// Proves the handler proceeds past auth and validates its input, i.e.
		// the route is genuinely reachable by an authorised minter.
		t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "expected-secret")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/event-bundles/mint", strings.NewReader(`{"not":"a bundle"}`))
		req.Header.Set("X-KNIRV-Internal-Token", "expected-secret")
		newServer().handleEventBundleMint(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 for an invalid bundle body", rec.Code)
		}
	})
}
