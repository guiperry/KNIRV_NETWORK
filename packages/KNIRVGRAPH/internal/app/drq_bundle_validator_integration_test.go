package app

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"KNIRVGRAPH/internal/drq"
)

// TestDVEBundleValidatorAgainstRealBackend drives the production bundle gate
// (drq.DVEBundleValidator) against the REAL backend validation routes, in a
// separate process over a unix socket. It proves the submit path
// (POST /api/v1/validation/tasks) and the verdict path
// (GET /api/v1/validation/tasks/{id}/results) are the ones the backend serves.
//
// The real backend has no executor for uLoRA bundles, so no attributable verdict
// can exist. The honest expectation is therefore: the task is accepted and
// created, the results route answers "pending" with a null result, and the gate
// refuses the bundle when its budget expires. A fabricated pass is not possible.
func TestDVEBundleValidatorAgainstRealBackend(t *testing.T) {
	bin := buildBackendValidationServer(t)
	sock := startBackendValidationServer(t, bin)

	token := mintServiceToken(t, contractSecret,
		[]string{"POST /api/v1/validation/tasks", "GET /api/v1/validation/tasks/*/results"}, "tenant-A")

	v := &drq.DVEBundleValidator{
		BaseURL: "http://unix",
		Token:   token,
		Client: &http.Client{Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", sock)
			},
		}},
		PollInterval: 20 * time.Millisecond,
		PollTimeout:  300 * time.Millisecond,
	}

	// The gate refuses a bundle whose fit kind cannot be established (PI_Q40),
	// so the fixture declares real gradient training. Otherwise the test would
	// stop at the gate and never reach the verdict route it exists to prove.
	manifest := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(manifest, []byte(`{"training":{"fit_kind":"gradient_sft"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	err := v.ValidateBundle(context.Background(),
		&drq.ErrorCluster{ClusterID: "c-bundle", OwnerAgent: "drq-validation"},
		&drq.ULoRACompileResult{BundleID: "b1", ContentHash: "abc123", ManifestPath: manifest},
		[]drq.ULoRADatasetRecord{{Context: "x"}})
	if err == nil {
		t.Fatal("a bundle with no executable validator must not pass the gate")
	}
	// Reaching this message means submit returned a task id and the results route
	// answered 200 with no result yet. A 404/405 from the wrong route would fail
	// earlier with a different message.
	if !strings.Contains(err.Error(), "did not produce a verdict") {
		t.Fatalf("expected a pending-then-timeout refusal from the real routes, got: %v", err)
	}
	if strings.Contains(err.Error(), "validation service returned 404") ||
		strings.Contains(err.Error(), "validation service refused") {
		t.Fatalf("validator is still hitting a route the backend does not serve: %v", err)
	}
}
