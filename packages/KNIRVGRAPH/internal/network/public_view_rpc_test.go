package network

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const q83Code = "func leak() { secret := \"hunter2\" }"

// q83CommitContext is the context an errorscan-driven CLI commit carries.
func q83CommitContext() map[string]interface{} {
	return map[string]interface{}{
		"exit_code":    float64(1),
		"file_path":    "internal/app/app.go",
		"line_number":  float64(12),
		"code_context": q83Code,
		"project_path": "/home/user/private-repo",
		"note":         "read /home/user/private-repo/internal/app/app.go",
	}
}

// The commit reply and the public listing show neither code_context, project_path
// nor an absolute path; the stored record still has them for the owner.
func TestCommitReplyAndListingHideCodeContextAndProjectPath(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "")
	h, sys := sourceRouter(t)
	commit := signedCommitWithContext(t, q83CommitContext(), validSource())
	rec := doJSON(t, h, "POST", "/graph/errors/commit", "", commit)
	if rec.Code != http.StatusOK {
		t.Fatalf("commit: %d %s", rec.Code, rec.Body.String())
	}
	assertNoPrivateContext(t, "commit reply", rec.Body.String())

	list := httptest.NewRecorder()
	h.ServeHTTP(list, httptest.NewRequest("GET", "/graph/errors", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list: %d", list.Code)
	}
	assertNoPrivateContext(t, "public listing", list.Body.String())

	nodes := sys.GetAllErrorNodes()
	if len(nodes) != 1 {
		t.Fatalf("want 1 stored node, got %d", len(nodes))
	}
	stored := nodes[0].GetContext().AsMap()
	if stored["code_context"] != q83Code || stored["project_path"] != "/home/user/private-repo" {
		t.Fatalf("owner-side record must keep code_context and project_path: %v", stored)
	}
}

func assertNoPrivateContext(t *testing.T, where, body string) {
	t.Helper()
	for _, private := range []string{q83Code, "hunter2", "/home/user", "private-repo"} {
		if strings.Contains(body, private) {
			t.Errorf("%s leaks %q: %s", where, private, body)
		}
	}
	if !strings.Contains(body, "internal/app/app.go") {
		t.Errorf("%s must keep the safe context keys: %s", where, body)
	}
}

// signedCommitWithContext builds a commit envelope whose ErrorRoot covers ctx.
func signedCommitWithContext(t *testing.T, ctx map[string]interface{}, source interface{}) map[string]interface{} {
	t.Helper()
	root, err := errorNodeCommitRoot(errorNodeCommitClaim{ErrorType: "crash", Description: "segfault", Context: ctx, Severity: 1})
	if err != nil {
		t.Fatal(err)
	}
	commit := map[string]interface{}{
		"schema_version": ErrorNodeCommitSchema, "error_type": "crash", "description": "segfault",
		"project_id": "proj-42", "session_id": "sess_7",
		"context": ctx, "severity": 1, "error_root": root,
		"signer_id": "cli", "signing_key_id": "cli", "signature": root,
	}
	if source != nil {
		commit["source_ref"] = source
	}
	return commit
}

// The backend (KNIRV_INTERNAL_AUTH_TOKEN) reads the same listing with the full
// record, so Testlab's source excerpt and project_path still work. Without the
// token the listing stays cut down, and a wrong token gets the cut-down view.
func TestListingKeepsCodeContextForInternalCaller(t *testing.T) {
	const token = "test-internal-token"
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", token)
	h, _ := sourceRouter(t)
	if rec := doJSON(t, h, "POST", "/graph/errors/commit", "", signedCommitWithContext(t, q83CommitContext(), validSource())); rec.Code != http.StatusOK {
		t.Fatalf("commit: %d %s", rec.Code, rec.Body.String())
	}

	authed := doJSON(t, h, "GET", "/graph/errors", token, nil)
	if authed.Code != http.StatusOK {
		t.Fatalf("authorised list: %d", authed.Code)
	}
	// The body is JSON, so the code is matched in its encoded form.
	enc, err := json.Marshal(q83Code)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"code_context":` + string(enc), `"project_path":"/home/user/private-repo"`} {
		if !strings.Contains(authed.Body.String(), want) {
			t.Errorf("authorised listing lost %s: %s", want, authed.Body.String())
		}
	}

	for name, tok := range map[string]string{"anonymous": "", "wrong token": "guess"} {
		rec := doJSON(t, h, "GET", "/graph/errors", tok, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s list: %d", name, rec.Code)
		}
		assertNoPrivateContext(t, name+" listing", rec.Body.String())
	}
}
