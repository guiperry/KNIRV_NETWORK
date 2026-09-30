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

// A paid mint must reach the pool as a transaction the chain accepts. The
// handler burns the minter's NRN first, so a transaction the pool then
// rejected (it was once minter-addressed and unsigned) charged for nothing.
func TestEventBundleMintSubmitsAVerifiableProtocolTransaction(t *testing.T) {
	burns := 0
	txChain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/transfer" {
			burns++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer txChain.Close()
	t.Setenv("KNIRV_TRANSACTION_CHAIN_URL", txChain.URL)
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "secret")

	bcs := newBadgeCredentialServer(t)
	body := map[string]any{
		"schema_version": eventBundleMintSchema, "event_id": "evt-1", "session_id": "sess-1",
		"project_id": "proj-1", "event_kind": "decision", "minter_address": "knirv1minter",
		"skills": []map[string]string{{"id": "skill-1"}},
	}
	rec := post(t, http.HandlerFunc(bcs.handleEventBundleMint), "/api/v1/event-bundles/mint", "secret", body)
	if rec.Code/100 != 2 {
		t.Fatalf("mint: %d %s", rec.Code, rec.Body.String())
	}
	if burns != 1 {
		t.Fatalf("expected one NRN burn, got %d", burns)
	}
	pool := bcs.BlockchainPtr.TransactionPool
	if len(pool) != 1 || pool[0].Type != TransactionTypeEventBundleMint {
		t.Fatalf("expected one event bundle mint in the pool: %+v", pool)
	}
	if ok, err := pool[0].VerifySignature(); !ok || err != nil {
		t.Fatalf("mint transaction must pass verification: %v %v", ok, err)
	}
	if !pool[0].VerifyTxn() {
		t.Fatal("mint transaction must pass VerifyTxn")
	}
	bundle, ok := decodeEventBundleMint(pool[0].Data)
	if !ok || bundle.MinterAddress != "knirv1minter" {
		t.Fatalf("the bundle must keep its minter on record: %+v", bundle)
	}

	// Replaying the same event neither burns again nor resubmits.
	post(t, http.HandlerFunc(bcs.handleEventBundleMint), "/api/v1/event-bundles/mint", "secret", body)
	if burns != 1 || len(bcs.BlockchainPtr.TransactionPool) != 1 {
		t.Fatalf("replay must not charge or resubmit: burns=%d pool=%d", burns, len(bcs.BlockchainPtr.TransactionPool))
	}
}
