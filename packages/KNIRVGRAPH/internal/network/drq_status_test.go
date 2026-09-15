package network

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"KNIRVGRAPH/internal/drq"
	"KNIRVGRAPH/internal/indexing"
	"KNIRVGRAPH/internal/query"

	"go.uber.org/zap"
)

// stubApp is a minimal AppInterface whose only interesting behaviour is the DRQ
// status it reports.
type stubApp struct {
	status drq.Status
}

func (s *stubApp) IsNetworkPaused() bool                            { return false }
func (s *stubApp) IsProcessingEnabled() bool                        { return false }
func (s *stubApp) GetIndexManager() *indexing.IndexManager          { return nil }
func (s *stubApp) GetQueryProcessor() *query.QueryProcessor         { return nil }
func (s *stubApp) SubsystemHealth(context.Context) map[string]error { return nil }
func (s *stubApp) DRQStatus() drq.Status                            { return s.status }

// The DRQ loop is served over the API, so a running node's mining state is
// inspectable rather than only visible in logs.
func TestGetDRQStatusServesLoopState(t *testing.T) {
	app := &stubApp{status: drq.Status{
		Enabled:        true,
		Ticks:          7,
		Clusters:       2,
		TrainerBackend: "none",
		ClustersDetail: []drq.ClusterView{
			{ClusterID: "cluster-1", Status: "active", Members: 3, Ready: true},
		},
		LastErrors: map[string]string{"cluster-1": "no LoRA training backend"},
	}}

	rpc := &RPCServer{app: app, logger: zap.NewNop()}
	rec := httptest.NewRecorder()
	rpc.getDRQStatus(rec, httptest.NewRequest(http.MethodGet, "/drq/status", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q", got)
	}

	var decoded drq.Status
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if !decoded.Enabled || decoded.Ticks != 7 || decoded.Clusters != 2 {
		t.Fatalf("unexpected status payload: %+v", decoded)
	}
	if len(decoded.ClustersDetail) != 1 || decoded.ClustersDetail[0].ClusterID != "cluster-1" {
		t.Fatalf("cluster detail missing: %+v", decoded.ClustersDetail)
	}
	if decoded.LastErrors["cluster-1"] == "" {
		t.Fatal("expected the blocking reason to be reported")
	}
}

// With no app reference the endpoint must report disabled, not panic.
func TestGetDRQStatusWithoutApp(t *testing.T) {
	rpc := &RPCServer{logger: zap.NewNop()}
	rec := httptest.NewRecorder()
	rpc.getDRQStatus(rec, httptest.NewRequest(http.MethodGet, "/drq/status", nil))

	var decoded drq.Status
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if decoded.Enabled {
		t.Fatal("expected disabled with no app reference")
	}
	if decoded.DisabledReason == "" {
		t.Fatal("expected a disabled reason")
	}
}
