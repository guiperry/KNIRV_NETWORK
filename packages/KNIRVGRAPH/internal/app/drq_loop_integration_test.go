package app

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"KNIRVGRAPH/internal/drq"
)

// validationContractSHA256 pins the shared validation contract fixture bytes.
// Keep this in sync with the backend_server copy at
// backend_server/internal/web/testdata/validation_contract.json (and the pin in its
// contract test). A mismatch means the two copies drifted: update BOTH and BOTH pins.
const validationContractSHA256 = "300fcea1962c1bc6b7e2c39104fafa23d7b8b511481ad92b722788d37357ff12"

// backendValidationServerDirEnv overrides where the KNIRV_CORP backend_server module
// lives. By default it is resolved from this package's location in the KNIRV
// workspace (KNIRV_NETWORK/packages/KNIRVGRAPH/internal/app -> KNIRV_CORP).
const backendValidationServerDirEnv = "KNIRV_BACKEND_SERVER_DIR"

// contractSecret is the HS256 secret shared by the real backend validation server
// process and the service tokens this test mints. It is a fixed test-only value
// that is never used outside this test process.
const contractSecret = "knirv-drq-contract-test-secret-32b!"

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

func contractStatus(t *testing.T, contract map[string]map[string]interface{}, key string) string {
	t.Helper()
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
		t.Fatalf("validation_contract.json sha256 %s != pinned %s; the fixture drifted", sum, validationContractSHA256)
	}
}

// buildBackendValidationServer compiles the REAL backend validation route server
// (backend_server/cmd/validation_contract_server) from its KNIRV_CORP module and
// returns the binary path. It fails the test if the backend module is absent.
func buildBackendValidationServer(t *testing.T) string {
	t.Helper()
	dir := os.Getenv(backendValidationServerDirEnv)
	if dir == "" {
		abs, err := filepath.Abs("../../../../../KNIRV_CORP/packages/server/backend_server")
		if err != nil {
			t.Fatal(err)
		}
		dir = abs
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		t.Fatalf("backend_server module not found at %s (set %s): %v", dir, backendValidationServerDirEnv, err)
	}
	out := filepath.Join(t.TempDir(), "validation_contract_server")
	cmd := exec.Command("go", "build", "-o", out, "./cmd/validation_contract_server")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=auto")
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build backend validation server: %v\n%s", err, combined)
	}
	return out
}

// startBackendValidationServer runs the real backend validation server as a separate
// process listening on a unix socket, and waits for its READY line. The socket
// directory is short so the path stays within the unix socket length limit.
func startBackendValidationServer(t *testing.T, bin string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "drqvs")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sock := filepath.Join(dir, "v.sock")

	cmd := exec.Command(bin, "-socket", sock)
	cmd.Env = append(os.Environ(), "KNIRV_CONTRACT_JWT_SECRET="+contractSecret)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	ready := make(chan error, 1)
	go func() {
		// The backend's package init may print diagnostics before the READY line;
		// consume lines until READY appears, and fail if the process exits first.
		reader := bufio.NewReader(stdout)
		for {
			line, err := reader.ReadString('\n')
			if strings.TrimSpace(line) == "READY" {
				ready <- nil
				return
			}
			if err != nil {
				ready <- fmt.Errorf("server exited before READY: %v (stderr: %s)", err, stderr.String())
				return
			}
		}
	}()
	select {
	case err := <-ready:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(60 * time.Second):
		t.Fatalf("backend validation server did not become ready (stderr: %s)", stderr.String())
	}
	return sock
}

// mintServiceToken signs an HS256 service JWT with the given secret, using exactly the
// claim shape the backend's ServiceClaims/ValidateServiceToken expect: scope, role
// "service", tenant, and the registered claims for issuer, audience and expiry.
func mintServiceToken(t *testing.T, secret string, scope []string, tenant string) string {
	t.Helper()
	now := time.Now()
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	payload, _ := json.Marshal(map[string]interface{}{
		"scope":  scope,
		"role":   "service",
		"tenant": tenant,
		"jti":    fmt.Sprintf("drq-%d", now.UnixNano()),
		"iss":    "knirv-server",
		"sub":    "drq-validation",
		"aud":    []string{"knirv-validation-service"},
		"iat":    now.Unix(),
		"nbf":    now.Unix(),
		"exp":    now.Add(time.Hour).Unix(),
	})
	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func writeTokenFile(t *testing.T, token string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "tok")
	if err := os.WriteFile(p, []byte(token), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// rawPostStatus issues a raw POST to the real server over the unix socket and returns
// the HTTP status, used to prove that routes outside the scope are refused.
func rawPostStatus(t *testing.T, sock, path, token string) int {
	t.Helper()
	client := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", sock)
		},
	}}
	req, err := http.NewRequest(http.MethodPost, "http://unix"+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

// TestValidationServiceRealBackendIntegration drives the production DRQ client
// (newValidationServiceClient / SubmitClusterForValidation / GetValidationProof)
// against the REAL backend validation routes, running in a separate process over a
// unix socket. It exercises the real RequireValidationAuth signature check, the real
// scope list, the real tenant/owner check and the real ValidationCore, and asserts
// the behaviour the backend actually produces, not a mock's.
func TestValidationServiceRealBackendIntegration(t *testing.T) {
	contract := loadValidationContract(t)
	bin := buildBackendValidationServer(t)
	sock := startBackendValidationServer(t, bin)

	tokA := mintServiceToken(t, contractSecret, []string{"POST /api/v1/validation/tasks", "GET /api/v1/validation/tasks/*/results"}, "tenant-A")
	tokB := mintServiceToken(t, contractSecret, []string{"POST /api/v1/validation/tasks", "GET /api/v1/validation/tasks/*/results"}, "tenant-B")

	t.Setenv(validationServiceURLEnv, "http+unix://"+sock)
	t.Setenv(validationServiceTokenFileEnv, writeTokenFile(t, tokA))
	client, ok := newValidationServiceClient()
	if !ok {
		t.Fatal("expected a client for the unix-socket endpoint")
	}

	// CREATE through the real route with a correctly signed scoped token.
	created, err := client.SubmitClusterForValidation(context.Background(), &drq.ErrorCluster{ClusterID: "c1", OwnerAgent: "drq-validation"})
	if err != nil {
		t.Fatalf("submit through real backend: %v", err)
	}
	if created.ID == "" {
		t.Fatal("create response missing id")
	}
	if created.Status != contractStatus(t, contract, "create_response") {
		t.Fatalf("create status %q does not match contract fixture %q", created.Status, contractStatus(t, contract, "create_response"))
	}

	// PROOF READ, owner tenant, before execution: the real backend reports the task as
	// pending with a null result, which the client maps to "no proof yet" (nil, nil).
	// A valid proof needs the execute route and an inference run, neither of which the
	// scoped service token may reach, so none is claimed here.
	proof, err := client.GetValidationProof(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("owner-tenant proof read: %v", err)
	}
	if proof != nil {
		t.Fatalf("pending task must have no proof yet, got %+v", proof)
	}

	// CROSS-TENANT PROOF READ: a correctly signed token for another tenant must be
	// refused by the real ownership check (403).
	clientB, ok := newValidationServiceClientWithToken(t, sock, tokB)
	if !ok {
		t.Fatal("expected a client for the tenant-B token")
	}
	if _, err := clientB.GetValidationProof(context.Background(), created.ID); err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("cross-tenant proof read must be refused with 403, got err=%v", err)
	}

	// UNSIGNED token: the real signature check must refuse it on create (401).
	clientUnsigned, ok := newValidationServiceClientWithToken(t, sock, fakeJWT(t, time.Now().Add(time.Hour)))
	if !ok {
		t.Fatal("expected a client for the unsigned token")
	}
	if _, err := clientUnsigned.SubmitClusterForValidation(context.Background(), &drq.ErrorCluster{ClusterID: "c2", OwnerAgent: "drq-validation"}); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("unsigned token must be refused with 401, got err=%v", err)
	}

	// WRONG-SECRET token (signed with another key): also refused (401).
	wrongKey := mintServiceToken(t, "a-different-secret-that-is-32-bytes!", []string{"POST /api/v1/validation/tasks"}, "tenant-A")
	clientWrong, ok := newValidationServiceClientWithToken(t, sock, wrongKey)
	if !ok {
		t.Fatal("expected a client for the wrong-key token")
	}
	if _, err := clientWrong.SubmitClusterForValidation(context.Background(), &drq.ErrorCluster{ClusterID: "c3", OwnerAgent: "drq-validation"}); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("wrong-key token must be refused with 401, got err=%v", err)
	}

	// EXECUTE is outside the scoped token's scope: the real route must refuse it.
	if code := rawPostStatus(t, sock, "/api/v1/validation/tasks/"+created.ID+"/execute", tokA); code != http.StatusUnauthorized {
		t.Fatalf("scoped service token reached execute: status %d, want 401", code)
	}
}

// newValidationServiceClientWithToken builds the production DRQ client for the given
// unix socket and token, using the same environment-driven constructor the launcher uses.
func newValidationServiceClientWithToken(t *testing.T, sock, token string) (drq.ValidationService, bool) {
	t.Helper()
	t.Setenv(validationServiceURLEnv, "http+unix://"+sock)
	t.Setenv(validationServiceTokenFileEnv, writeTokenFile(t, token))
	return newValidationServiceClient()
}
