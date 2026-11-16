package launcher

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
)

// captureLog redirects the standard logger into a buffer for the duration of the test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return &buf
}

// TestInvalidSecretsBackendWarnsAndFallsBackToFile: an invalid KNIRV_SECRETS_BACKEND keeps
// the file fallback (OpenBao is not started, no error is returned) and logs a warning that
// names the variable without echoing the value. Uses only process env and a captured logger;
// startOpenBao returns before any data-directory or key/secret path is touched.
func TestInvalidSecretsBackendWarnsAndFallsBackToFile(t *testing.T) {
	const bogus = "not-a-real-backend-FAKE-VALUE"
	t.Setenv(secretsBackendEnv, bogus)
	buf := captureLog(t)

	app := appWithOpenBao(true) // config asks for OpenBao; the invalid value must still block it
	if err := app.startOpenBao(context.Background()); err != nil {
		t.Fatalf("startOpenBao returned error %v; invalid backend must fall back to file, not fail", err)
	}
	if app.bao != nil {
		t.Fatal("OpenBao manager was created for an invalid backend value")
	}

	out := buf.String()
	if !strings.Contains(out, secretsBackendEnv) {
		t.Fatalf("warning does not name %s: %q", secretsBackendEnv, out)
	}
	if !strings.Contains(out, "invalid value") || !strings.Contains(out, "file backend") {
		t.Fatalf("warning does not state invalid value and file fallback: %q", out)
	}
	if strings.Contains(out, bogus) {
		t.Fatalf("warning leaked the configured value: %q", out)
	}
}

// TestValidFileSecretsBackendDoesNotWarnInvalid: a valid "file" value keeps its existing
// informational line and does not produce the invalid-value warning.
func TestValidFileSecretsBackendDoesNotWarnInvalid(t *testing.T) {
	t.Setenv(secretsBackendEnv, "file")
	buf := captureLog(t)

	if err := appWithOpenBao(true).startOpenBao(context.Background()); err != nil {
		t.Fatalf("startOpenBao: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "invalid value") {
		t.Fatalf("valid value produced invalid-value warning: %q", out)
	}
	if !strings.Contains(out, "file backend in use") {
		t.Fatalf("expected existing file-mode line, got %q", out)
	}
}
