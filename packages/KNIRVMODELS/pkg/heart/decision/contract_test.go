package decision

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// The SDK client (what backend_server and KNIRVINFERENCER use) against the real
// routes and the real wasm: the wire contract holds end to end, not just against
// the SDK's own fixtures.
func TestSDKClientAgainstRealRoutesAndWasm(t *testing.T) {
	const v = "v1-deadbeef"
	svc := NewService(publish(t, v))
	defer svc.Close(context.Background())
	mux := http.NewServeMux()
	Routes(svc, mux)
	RoutesAt(svc, mux, sdk.HeartDecisionPrefix)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := sdk.NewClient(srv.URL, nil)
	ctx := context.Background()

	m, err := c.DecisionManifest(ctx)
	if err != nil || m.Version != v {
		t.Fatalf("manifest: %+v %v", m, err)
	}
	width, err := m.InputWidth()
	if err != nil || width <= 0 {
		t.Fatalf("width %d: %v", width, err)
	}

	// The golden model has a 32-token vocabulary, so it cannot take byte-v1 text.
	toks := make([]uint32, width)
	for i := range toks {
		toks[i] = uint32(3 + i%20)
	}
	r, err := c.Decide(ctx, toks)
	if err != nil {
		t.Fatal(err)
	}
	if r.ModelVersion != v || len(r.Decisions) == 0 {
		t.Fatalf("reasoning did not decode into the SDK type: %+v", r)
	}
	for _, d := range r.Decisions {
		if d.Name == "" || d.Kind == "" {
			t.Fatalf("decision lost fields: %+v", d)
		}
	}

	// byte-v1 text is outside the golden model's vocabulary, so the daemon refuses it:
	// the advice path must turn that into an abstention, never an error.
	a := c.Advise(ctx, sdk.AdviseContext{CurrentPhase: "develop"})
	if !a.Abstain || a.AbstainReason == "" {
		t.Fatalf("expected an abstention, got %+v", a)
	}

	if _, err := c.Matrix(ctx, v, "h.0.attn.q.weight"); err != nil {
		t.Fatalf("matrix: %v", err)
	}
}

// A v1 model has no choice-set heads. Both the native Rust core and the wasm the
// SDK client calls must reject a set request with the same error, so every runtime
// agrees on the contract (PendingImplementations 4.7). This is the cross-runtime
// golden: SDK client -> real route -> real wasm must echo Rust's "no choice-set heads".
func TestSetRequestRejectedOnV1Model(t *testing.T) {
	const v = "v1-deadbeef"
	svc := NewService(publish(t, v))
	defer svc.Close(context.Background())
	mux := http.NewServeMux()
	RoutesAt(svc, mux, sdk.HeartDecisionPrefix)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := sdk.NewClient(srv.URL, nil)
	ctx := context.Background()

	questions, _ := json.Marshal([]map[string]any{
		{"name": "route", "state": "develop", "choices": []map[string]string{{"id": "a", "text": "route-a"}, {"id": "b", "text": "route-b"}}},
	})
	r, err := c.DecideSets(ctx, questions)
	if err == nil {
		t.Fatalf("expected a v1 model to reject set requests, got %+v", r)
	}
	if !strings.Contains(err.Error(), "no choice-set heads") {
		t.Fatalf("error must match the Rust contract 'no choice-set heads', got %q", err.Error())
	}
}
