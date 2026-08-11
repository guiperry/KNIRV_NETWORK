package d1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_Query_Select(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody queryRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"errors": [],
			"messages": [],
			"result": [
				{
					"results": [
						{"id": "op-1", "legal_name": "Acme Node Co"},
						{"id": "op-2", "legal_name": "Beta Nodes LLC"}
					],
					"success": true,
					"meta": {"changes": 0, "last_row_id": 0}
				}
			]
		}`))
	}))
	defer srv.Close()

	c := &Client{
		AccountID:  "acct-123",
		DatabaseID: "db-456",
		APIToken:   "tok-789",
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	}

	result, err := c.Query(context.Background(), "SELECT id, legal_name FROM operator_applications WHERE kyc_status = ?", "pending")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	wantPath := "/client/v4/accounts/acct-123/d1/database/db-456/query"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotAuth != "Bearer tok-789" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer tok-789")
	}
	if gotBody.SQL != "SELECT id, legal_name FROM operator_applications WHERE kyc_status = ?" {
		t.Errorf("SQL = %q", gotBody.SQL)
	}
	if len(gotBody.Params) != 1 || gotBody.Params[0] != "pending" {
		t.Errorf("Params = %v, want [\"pending\"]", gotBody.Params)
	}

	if len(result.Rows) != 2 {
		t.Fatalf("Rows = %d, want 2", len(result.Rows))
	}
	if result.Rows[0]["legal_name"] != "Acme Node Co" {
		t.Errorf("Rows[0][legal_name] = %v", result.Rows[0]["legal_name"])
	}
}

func TestClient_Query_Update(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": true,
			"errors": [],
			"result": [
				{"results": [], "success": true, "meta": {"changes": 1, "last_row_id": 0}}
			]
		}`))
	}))
	defer srv.Close()

	c := &Client{AccountID: "a", DatabaseID: "d", APIToken: "t", BaseURL: srv.URL, HTTPClient: srv.Client()}
	result, err := c.Query(context.Background(), "UPDATE accounts SET role = ? WHERE id = ?", "admin", "user-1")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if result.Changes != 1 {
		t.Errorf("Changes = %d, want 1", result.Changes)
	}
}

func TestClient_Query_CloudflareError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"success": false,
			"errors": [{"code": 7403, "message": "Authentication error"}],
			"result": []
		}`))
	}))
	defer srv.Close()

	c := &Client{AccountID: "a", DatabaseID: "d", APIToken: "bad-token", BaseURL: srv.URL, HTTPClient: srv.Client()}
	_, err := c.Query(context.Background(), "SELECT 1")
	if err == nil {
		t.Fatal("expected an error for a Cloudflare-level failure, got nil")
	}
	if !strings.Contains(err.Error(), "Authentication error") {
		t.Errorf("error = %v, want it to mention the Cloudflare error message", err)
	}
}

func TestClient_Query_MissingCredentials(t *testing.T) {
	c := &Client{DatabaseID: "d", APIToken: "t"}
	if _, err := c.Query(context.Background(), "SELECT 1"); err == nil {
		t.Fatal("expected an error when AccountID is empty, got nil")
	}
}
