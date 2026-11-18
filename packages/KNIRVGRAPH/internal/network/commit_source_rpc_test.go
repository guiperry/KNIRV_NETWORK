package network

import (
	"KNIRVGRAPH/internal/locator"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

const (
	srcDigestA = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	srcDigestB = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	srcCommit  = "0123456789abcdef0123456789abcdef01234567"
)

// sourceRouter serves the commit route and the public listing.
func sourceRouter(t *testing.T) (http.Handler, *locator.LocatorSystem) {
	t.Helper()
	sys := locator.NewLocatorSystem("test-peer", nil)
	rpc := &RPCServer{logger: zap.NewNop(), nrvSystem: sys}
	r := mux.NewRouter()
	handleLocatorRoute(r, zap.NewNop(), "/errors/commit", rpc.createErrorCommit, "POST")
	handleLocatorRoute(r, zap.NewNop(), "/errors", rpc.getAllErrors, "GET")
	return r, sys
}

// signedCommit builds a commit envelope whose ErrorRoot matches its content.
// source, when non-nil, is added as source_ref. ErrorRoot does not cover it.
func signedCommit(t *testing.T, source interface{}) map[string]interface{} {
	t.Helper()
	errCtx := map[string]interface{}{"exit_code": float64(1)}
	root, err := errorNodeCommitRoot(errorNodeCommitClaim{ErrorType: "crash", Description: "segfault", Context: errCtx, Severity: 1})
	if err != nil {
		t.Fatal(err)
	}
	commit := map[string]interface{}{
		"schema_version": ErrorNodeCommitSchema, "error_type": "crash", "description": "segfault",
		"project_id": "proj-42", "session_id": "sess_7",
		"context": errCtx, "severity": 1, "error_root": root,
		"signer_id": "cli", "signing_key_id": "cli", "signature": root,
	}
	if source != nil {
		commit["source_ref"] = source
	}
	return commit
}

func validSource() map[string]interface{} {
	return map[string]interface{}{
		"project_id": "proj-42", "session_id": "sess_7", "commit_sha": srcCommit,
		"ref": "main", "path": "internal/app/app.go", "line_start": 10, "line_end": 14,
		"diff_digest": srcDigestA, "workspace_base_hash": srcDigestB, "workspace_final_hash": srcDigestA,
	}
}

// A commit with a source stores it in full, the reply and the public listing
// show only the digests, and the private pointers never leave the server.
func TestCommitSourceRefStoredPrivatelyAndPublicListingDigestsOnly(t *testing.T) {
	h, sys := sourceRouter(t)
	if rec := doJSON(t, h, "POST", "/graph/errors/commit", "", signedCommit(t, validSource())); rec.Code != http.StatusOK {
		t.Fatalf("commit: %d %s", rec.Code, rec.Body.String())
	}
	nodes := sys.GetAllErrorNodes()
	if len(nodes) != 1 || nodes[0].Source == nil || nodes[0].Source.ProjectId != "proj-42" || nodes[0].Source.Path != "internal/app/app.go" {
		t.Fatalf("the owner-side store must keep the full source: %+v", nodes)
	}

	list := httptest.NewRecorder()
	h.ServeHTTP(list, httptest.NewRequest("GET", "/graph/errors", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list: %d", list.Code)
	}
	body := list.Body.String()
	for _, private := range []string{"proj-42", "sess_7", "internal/app/app.go", srcCommit, "\"main\""} {
		if strings.Contains(body, private) {
			t.Fatalf("public listing leaks %q: %s", private, body)
		}
	}
	if !strings.Contains(body, srcDigestA) || !strings.Contains(body, srcDigestB) {
		t.Fatalf("public listing must keep the digests: %s", body)
	}
}

// Rows and commits without a source keep working: additive field.
func TestCommitWithoutSourceRefStillCreates(t *testing.T) {
	h, sys := sourceRouter(t)
	if rec := doJSON(t, h, "POST", "/graph/errors/commit", "", signedCommit(t, nil)); rec.Code != http.StatusOK {
		t.Fatalf("commit without source: %d %s", rec.Code, rec.Body.String())
	}
	if nodes := sys.GetAllErrorNodes(); len(nodes) != 1 || nodes[0].Source != nil {
		t.Fatalf("a sourceless commit must create a sourceless node: %+v", nodes)
	}
}

// Unknown keys (where code or diffs would be smuggled) and free text are
// refused with 400, nothing is stored, and the reply does not echo the value.
func TestCommitRejectsCodeAndUnknownSourceFields(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"diff key": func() map[string]interface{} {
			s := validSource()
			s["diff"] = "diff --git a/x b/x\n+os.Exit(1)"
			return s
		}(),
		"code in path": func() map[string]interface{} {
			s := validSource()
			s["path"] = "x.go\nfunc leak() {}"
			return s
		}(),
		"bad digest": func() map[string]interface{} {
			s := validSource()
			s["diff_digest"] = "func leak() {}"
			return s
		}(),
		"wrong type": func() map[string]interface{} {
			s := validSource()
			s["line_start"] = "ten"
			return s
		}(),
	}
	for name, src := range cases {
		h, sys := sourceRouter(t)
		rec := doJSON(t, h, "POST", "/graph/errors/commit", "", signedCommit(t, src))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d %s", name, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "func leak") || strings.Contains(rec.Body.String(), "os.Exit") {
			t.Errorf("%s: reply echoes the rejected content: %s", name, rec.Body.String())
		}
		if n := len(sys.GetAllErrorNodes()); n != 0 {
			t.Errorf("%s: a refused commit must store nothing, got %d", name, n)
		}
	}
}
