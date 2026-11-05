package models

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"knirvmodels/internal/ops"
	"knirvmodels/internal/policy"
	"knirvmodels/pkg/hands"
	"knirvmodels/pkg/hands/gpt"

	"github.com/guiperry/knirv-sdk-go/models/frame"
)

// newTestHands returns a HANDS whose build always yields an unready (untrained)
// engine, matching the default daemon wiring when no checkpoint is present. A
// tiny config keeps the random init instant (DefaultConfig is GPT-2-small and
// would take tens of seconds to initialize).
func newTestHands() *Hands {
	return NewHands(
		func() (*hands.Engine, error) {
			return hands.NewEngine(&gpt.Config{
				VocabSize: 16, EmbedDim: 8, NumHeads: 2, NumLayers: 1,
				ContextLen: 16, FFNHiddenDim: 16, DecayAlpha: 1.0,
			}, ""), nil
		},
		policy.NewGate(frame.Default(), nil), nil, ops.New(), frame.Default(),
	)
}

func TestHandsDisabledReportsNotReady(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "")
	st := newTestHands().Status()
	if st.Ready {
		t.Fatalf("disabled HANDS must report not ready")
	}
	if st.Backend != "disabled" {
		t.Fatalf("disabled HANDS backend = %q, want disabled", st.Backend)
	}
}

func TestHandsUntrainedReturns503(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	mux := http.NewServeMux()
	newTestHands().Mount(mux, "/models/hands")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// All trained-path routes must 503 before a checkpoint is loaded.
	for _, path := range []string{
		"/models/hands/generate",
		"/models/hands/plan",
		"/models/hands/act",
	} {
		resp, err := http.Post(srv.URL+path, "application/json",
			strings.NewReader(`{"prompt":"hello"}`))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("%s status = %d, want 503 (untrained)", path, resp.StatusCode)
		}
	}

	// /weights is also gated (GET).
	resp, err := http.Get(srv.URL + "/models/hands/weights")
	if err != nil {
		t.Fatalf("weights: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("/weights status = %d, want 503 (untrained)", resp.StatusCode)
	}
}

func TestHandsPlanActRequirePost(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	mux := http.NewServeMux()
	newTestHands().Mount(mux, "/models/hands")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, path := range []string{"/models/hands/plan", "/models/hands/act"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("%s status = %d, want 405 (GET not allowed)", path, resp.StatusCode)
		}
	}
}

func TestHandsModelsEndpoint(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	mux := http.NewServeMux()
	newTestHands().Mount(mux, "/models/hands")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/models/hands/models")
	if err != nil {
		t.Fatalf("get models: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/models status = %d, want 200", resp.StatusCode)
	}
}

func TestHandsHealthEndpoint(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	mux := http.NewServeMux()
	newTestHands().Mount(mux, "/models/hands")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/models/hands/health")
	if err != nil {
		t.Fatalf("get health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/health status = %d, want 200", resp.StatusCode)
	}
	var got map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if got["role"] != "server_tasks" {
		t.Fatalf("/health role = %v, want server_tasks", got["role"])
	}
}

func TestHandsHealthNotMountedWhenDisabled(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "")
	mux := http.NewServeMux()
	newTestHands().Mount(mux, "/models/hands")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/models/hands/health")
	if err != nil {
		t.Fatalf("get health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("/health when disabled status = %d, want 404", resp.StatusCode)
	}
}
