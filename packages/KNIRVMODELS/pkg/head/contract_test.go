package head

import (
	"context"
	"net/http"
	"net/http/httptest"
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
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := sdk.NewClient(srv.URL, nil)
	ctx := context.Background()

	m, err := c.Manifest(ctx)
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
	r, err := c.Reason(ctx, toks)
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
