package nrv

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSubmitErrorCommit_HitsExpectedPathAndBody(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody ErrorNodeCommit

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	commit := &ErrorNodeCommit{
		SchemaVersion: SchemaVersion,
		ErrorType:     "security_vulnerability:sql_injection",
		Description:   "possible SQL injection",
		Context:       map[string]interface{}{"file_path": "main.go"},
		Severity:      9,
		ErrorRoot:     "sha256:abc123",
		SignerID:      "deadbeef",
		SigningKeyID:  "ed25519:deadbeef",
		Signature:     "cafebabe",
	}

	if err := client.SubmitErrorCommit(context.Background(), commit); err != nil {
		t.Fatalf("SubmitErrorCommit returned error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/graph/nrv/errors/commit" {
		t.Errorf("path = %q, want /api/graph/nrv/errors/commit", gotPath)
	}
	if gotBody.SchemaVersion != SchemaVersion {
		t.Errorf("body schema_version = %q, want %q", gotBody.SchemaVersion, SchemaVersion)
	}
	if gotBody.ErrorRoot != commit.ErrorRoot {
		t.Errorf("body error_root = %q, want %q", gotBody.ErrorRoot, commit.ErrorRoot)
	}
	if gotBody.SignerID != commit.SignerID {
		t.Errorf("body signer_id = %q, want %q", gotBody.SignerID, commit.SignerID)
	}
}

func TestSubmitErrorCommit_NonOKStatusIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "error root does not match committed content", http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.SubmitErrorCommit(context.Background(), &ErrorNodeCommit{})
	if err == nil {
		t.Fatal("expected an error for a non-2xx response")
	}
}
