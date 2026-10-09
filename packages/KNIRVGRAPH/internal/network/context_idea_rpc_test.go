package network

import (
	"KNIRVGRAPH/internal/nrv"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func contextIdeaRouter(t *testing.T, withNRV bool) http.Handler {
	t.Helper()
	rpc := &RPCServer{logger: zap.NewNop()}
	if withNRV {
		rpc.nrvSystem = nrv.NewNRVSystem("test-peer", nil)
	}
	r := mux.NewRouter()
	rpc.registerContextIdeaRoutes(r)
	return r
}

func TestContextNodeLifecycle(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "secret")
	h := contextIdeaRouter(t, true)
	body := map[string]interface{}{
		"context_type": "mcp_server", "description": "Weather MCP server",
		"schema": map[string]interface{}{"category": "integration"}, "submitted_by": "user-42",
	}
	if rec := doJSON(t, h, "POST", "/nrv/contexts", "", body); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous creation must be refused, got %d", rec.Code)
	}
	for _, bad := range []map[string]interface{}{
		{"context_type": "MCP Server Context", "description": "x"},
		{"context_type": "tool", "description": "  "},
	} {
		if rec := doJSON(t, h, "POST", "/nrv/contexts", "secret", bad); rec.Code != http.StatusBadRequest {
			t.Fatalf("invalid context %v must be rejected, got %d", bad, rec.Code)
		}
	}
	rec := doJSON(t, h, "POST", "/nrv/contexts", "secret", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created nrv.ContextNode
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if created.ID == "" || created.Schema["submitted_by"] != "user-42" || created.Status != "pending" {
		t.Fatalf("unexpected node: %+v", created)
	}
	if rec := doJSON(t, h, "GET", "/nrv/contexts/"+created.ID, "", nil); rec.Code != http.StatusOK {
		t.Fatalf("get: %d", rec.Code)
	}
	if rec := doJSON(t, h, "GET", "/nrv/contexts/missing", "", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("missing node: %d", rec.Code)
	}
	var all []nrv.ContextNode
	_ = json.Unmarshal(doJSON(t, h, "GET", "/nrv/contexts", "", nil).Body.Bytes(), &all)
	if len(all) != 1 {
		t.Fatalf("list: %d nodes", len(all))
	}
}

func TestIdeaNodeLifecycle(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "secret")
	h := contextIdeaRouter(t, true)
	if rec := doJSON(t, h, "POST", "/nrv/ideas", "secret", map[string]interface{}{"idea_type": "User Innovation Concept", "description": "x"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("free-text idea_type must be rejected, got %d", rec.Code)
	}
	rec := doJSON(t, h, "POST", "/nrv/ideas", "secret", map[string]interface{}{
		"idea_type": "innovation", "description": "Self-healing build cache", "feasibility_data": map[string]interface{}{"score": 0.7}, "submitted_by": "user-42",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created nrv.IdeaNode
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if created.FeasibilityData["submitted_by"] != "user-42" || created.FeasibilityData["score"] != 0.7 {
		t.Fatalf("unexpected node: %+v", created)
	}
	if rec := doJSON(t, h, "GET", "/nrv/ideas/"+created.ID, "", nil); rec.Code != http.StatusOK {
		t.Fatalf("get: %d", rec.Code)
	}
}

func TestContextIdeaRoutesWithoutNRVSystem(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "secret")
	h := contextIdeaRouter(t, false)
	if rec := doJSON(t, h, "POST", "/nrv/ideas", "secret", map[string]interface{}{"idea_type": "asset", "description": "x"}); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("without NRV the route must refuse rather than fake a node, got %d", rec.Code)
	}
}
