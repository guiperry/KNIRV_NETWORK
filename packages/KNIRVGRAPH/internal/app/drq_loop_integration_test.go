package app

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"KNIRVGRAPH/internal/drq"
)

// decodeJWTClaims is a test-only helper that base64-decodes the JWT payload
// WITHOUT verifying the signature. This file is a client-side contract test
// against the shared fixture (validation_contract.json); the JWT is decoded only
// to drive a test-local ownership map, not to prove backend auth. The real-route
// proof lives in backend_server/internal/web/validation_handlers_contract_test.go.
func decodeJWTClaims(t *testing.T, token string) map[string]interface{} {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("bad token in test server: %q", token)
	}
	p, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("bad payload: %v", err)
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(p, &claims); err != nil {
		t.Fatalf("bad claims: %v", err)
	}
	return claims
}

// validationContractSHA256 pins the shared validation contract fixture bytes.
// Keep this in sync with backend_server/internal/web/testdata/validation_contract.json
// (and the pin in validation_handlers_contract_test.go). A mismatch means the two
// copies drifted: update BOTH fixtures and BOTH pins together.
const validationContractSHA256 = "300fcea1962c1bc6b7e2c39104fafa23d7b8b511481ad92b722788d37357ff12"

// loadValidationContract reads the shared JSON contract fixture (the single source
// of truth for the POST /api/v1/validation/tasks and GET .../{id}/results shapes)
// that this DRQ client-side contract test and the backend_server real-route
// contract test both assert against.
func loadValidationContract(t *testing.T) map[string]map[string]interface{} {
	t.Helper()
	raw, err := os.ReadFile("testdata/validation_contract.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

// contractStatus returns the string "status" field from a contract section.
func contractStatus(t *testing.T, contract map[string]map[string]interface{}, key string) string {
	v, ok := contract[key]["status"]
	if !ok {
		t.Fatalf("contract missing %s.status", key)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("contract %s.status is not a string", key)
	}
	return s
}

// TestNewValidationServiceClientRejectsTCP enforces the internal unix-only
// transport: a non-http+unix URL must fail closed even with a valid token.
func TestNewValidationServiceClientRejectsTCP(t *testing.T) {
	t.Setenv(validationServiceURLEnv, "http://127.0.0.1:1")
	tokenFile := filepath.Join(t.TempDir(), "tok")
	if err := os.WriteFile(tokenFile, []byte(fakeJWT(t, time.Now().Add(time.Hour))), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(validationServiceTokenFileEnv, tokenFile)

	if c, ok := newValidationServiceClient(); ok {
		t.Fatalf("expected failure for TCP URL, got client %v", c)
	}
}

// TestValidationContractFixtureSHA256 pins the shared validation contract fixture
// so the KNIRVGRAPH and backend_server copies cannot drift silently.
func TestValidationContractFixtureSHA256(t *testing.T) {
	raw, err := os.ReadFile("testdata/validation_contract.json")
	if err != nil {
		t.Fatal(err)
	}
	got := sha256.Sum256(raw)
	if sum := hex.EncodeToString(got[:]); sum != validationContractSHA256 {
		t.Fatalf("validation_contract.json sha256 %s != pinned %s; the fixture drifted — update KNIRVGRAPH/internal/app/testdata/validation_contract.json, backend_server/internal/web/testdata/validation_contract.json, and both SHA256 pins together", sum, validationContractSHA256)
	}
}

// TestValidationServiceUnixSocketIntegration is a client-side contract test: it
// exercises the DRQ client's unix-socket wiring (KNIRV_VALIDATION_SERVICE_URL),
// handoff token file (KNIRV_VALIDATION_SERVICE_TOKEN_FILE), create-response
// decoding, and owner+tenant-checked proof read against the shared fixture,
// using a local test server and test-local ownership state (JWT decoded without
// signature verification). It does not prove backend router/auth/middleware;
// the real-route proof is backend_server/internal/web/validation_handlers_contract_test.go.
func TestValidationServiceUnixSocketIntegration(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "validation.sock")
	if err := os.RemoveAll(sock); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}

	// Local test server for client-side contract/ownership checks (not a real
	// backend route/auth integration).
	tasks := map[string]struct {
		owner  string
		tenant string
	}{}
	contract := loadValidationContract(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/validation/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		claims := decodeJWTClaims(t, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		id := "task-" + time.Now().Format("150405.000")
		tasks[id] = struct {
			owner  string
			tenant string
		}{owner: claims["sub"].(string), tenant: claims["tenant"].(string)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		// Response must match the shared contract fixture exactly (the DRQ client
		// parses id/status from it).
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   contract["create_response"]["success"],
			"id":        id,
			"status":    contract["create_response"]["status"],
			"message":   contract["create_response"]["message"],
			"timestamp": contract["create_response"]["timestamp"],
		})
	})
	mux.HandleFunc("/api/v1/validation/tasks/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/validation/tasks/")
		id = strings.TrimSuffix(id, "/results")
		task, ok := tasks[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		claims := decodeJWTClaims(t, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		// Owner + tenant enforcement, mirroring the real handler.
		if claims["sub"].(string) != task.owner || claims["tenant"].(string) != task.tenant {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"task_id": id,
			"status":  contract["results_response"]["status"],
			"result":  contract["results_response"]["result"],
		})
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(listener)
	defer srv.Close()

	// Launcher-style propagation.
	t.Setenv(validationServiceURLEnv, "http+unix://"+sock)
	ownerTokenFile := filepath.Join(t.TempDir(), "tok")
	if err := os.WriteFile(ownerTokenFile, []byte(fakeJWT(t, time.Now().Add(time.Hour))), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(validationServiceTokenFileEnv, ownerTokenFile)

	client, ok := newValidationServiceClient()
	if !ok {
		t.Fatal("expected a client for the unix-socket endpoint")
	}

	created, err := client.SubmitClusterForValidation(context.Background(), &drq.ErrorCluster{ClusterID: "c1", OwnerAgent: "drq-validation"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if created.ID == "" {
		t.Fatal("create response missing id")
	}
	if created.Status != "pending" {
		t.Fatalf("create status = %q, want pending", created.Status)
	}
	// Assert the client parsed the shared contract shape exactly.
	if created.Status != contractStatus(t, contract, "create_response") {
		t.Fatalf("create status %q does not match contract fixture", created.Status)
	}

	proof, err := client.GetValidationProof(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get proof: %v", err)
	}
	if proof == nil || !proof.Valid {
		t.Fatalf("expected a valid proof, got %+v", proof)
	}

	// A different-owner caller must be rejected (owner+tenant check).
	otherFile := filepath.Join(t.TempDir(), "other")
	otherTok := fakeJWTWith(t, time.Now().Add(time.Hour), "other-owner", "other-tenant")
	if err := os.WriteFile(otherFile, []byte(otherTok), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(validationServiceTokenFileEnv, otherFile)
	client2, ok := newValidationServiceClient()
	if !ok {
		t.Fatal("expected a client for the other-owner token")
	}
	if _, err := client2.GetValidationProof(context.Background(), created.ID); err == nil {
		t.Fatal("expected owner/tenant mismatch to be rejected")
	}
}

// fakeJWTWith builds a valid-shaped JWT with explicit sub/tenant claims.
func fakeJWTWith(t *testing.T, exp time.Time, sub, tenant string) string {
	t.Helper()
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	payload, _ := json.Marshal(map[string]interface{}{
		"sub":    sub,
		"tenant": tenant,
		"exp":    exp.Unix(),
		"iss":    "knirv-server",
		"aud":    "knirv-validation-service",
	})
	return strings.Join([]string{
		base64.RawURLEncoding.EncodeToString(header),
		base64.RawURLEncoding.EncodeToString(payload),
		"signature",
	}, ".")
}
