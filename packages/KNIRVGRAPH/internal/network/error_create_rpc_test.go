package network

import (
	"KNIRVGRAPH/internal/nrv"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func errorCreateRouter(t *testing.T) (http.Handler, *nrv.NRVSystem) {
	t.Helper()
	sys := nrv.NewNRVSystem("test-peer", nil)
	rpc := &RPCServer{logger: zap.NewNop(), nrvSystem: sys}
	r := mux.NewRouter()
	r.HandleFunc("/nrv/errors", rpc.createError).Methods("POST")
	r.HandleFunc("/nrv/errors/commit", rpc.createErrorCommit).Methods("POST")
	return r, sys
}

func storedContext(t *testing.T, sys *nrv.NRVSystem, errorType string) map[string]interface{} {
	t.Helper()
	for _, node := range sys.GetAllErrorNodes() {
		if node.ErrorType == errorType {
			return nrv.ErrorContextMap(node)
		}
	}
	t.Fatalf("no stored error node of type %s", errorType)
	return nil
}

func TestCreateErrorRequiresInternalTokenAndStripsResolverClaims(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "secret")
	h, sys := errorCreateRouter(t)
	body := map[string]interface{}{
		"error_type": "leak", "description": "fd leak", "severity": 2,
		"context": map[string]interface{}{"resolved_by": "attacker", "agent_id": "attacker", "file": "main.go"},
	}

	if rec := doJSON(t, h, "POST", "/nrv/errors", "", body); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous error creation must be refused, got %d", rec.Code)
	}
	if rec := doJSON(t, h, "POST", "/nrv/errors", "wrong", body); rec.Code != http.StatusUnauthorized {
		t.Fatalf("a wrong token must be refused, got %d", rec.Code)
	}
	if n := len(sys.GetAllErrorNodes()); n != 0 {
		t.Fatalf("refused requests must not create nodes, have %d", n)
	}

	if rec := doJSON(t, h, "POST", "/nrv/errors", "secret", body); rec.Code != http.StatusOK {
		t.Fatalf("internal creation: %d %s", rec.Code, rec.Body.String())
	}
	ctx := storedContext(t, sys, "leak")
	for _, k := range nrv.ResolverContextKeys {
		if _, ok := ctx[k]; ok {
			t.Fatalf("resolver claim %q must be stripped: %v", k, ctx)
		}
	}
	if ctx["file"] != "main.go" {
		t.Fatalf("ordinary context must survive: %v", ctx)
	}
}

func TestCreateErrorCommitStripsResolverClaims(t *testing.T) {
	h, sys := errorCreateRouter(t)
	errCtx := map[string]interface{}{"owner_agent": "attacker", "resolver": "attacker", "exit_code": float64(1)}
	root, err := errorNodeCommitRoot(errorNodeCommitClaim{ErrorType: "crash", Description: "segfault", Context: errCtx, Severity: 1})
	if err != nil {
		t.Fatal(err)
	}
	commit := map[string]interface{}{
		"schema_version": ErrorNodeCommitSchema, "error_type": "crash", "description": "segfault",
		"context": errCtx, "severity": 1, "error_root": root,
		"signer_id": "cli", "signing_key_id": "cli", "signature": root,
	}
	if rec := doJSON(t, h, "POST", "/nrv/errors/commit", "", commit); rec.Code != http.StatusOK {
		t.Fatalf("commit: %d %s", rec.Code, rec.Body.String())
	}
	ctx := storedContext(t, sys, "crash")
	for _, k := range nrv.ResolverContextKeys {
		if _, ok := ctx[k]; ok {
			t.Fatalf("resolver claim %q must be stripped: %v", k, ctx)
		}
	}
	if ctx["exit_code"] != float64(1) {
		t.Fatalf("ordinary context must survive: %v", ctx)
	}
}
