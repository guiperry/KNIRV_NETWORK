package headless

import (
	"knirvmodels/pkg/evidence"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The daemon's other tests exercise the Gorgonite HEART; Qwen is the default
// backend in production, so they opt out and the Qwen wiring is tested below.
func TestMain(m *testing.M) {
	os.Setenv("KNIRV_HEART_BACKEND", "gpt")
	os.Exit(m.Run())
}

func TestLlamaSocketPath(t *testing.T) {
	t.Setenv("KNIRV_LLAMA_SOCKET", "/x/llama.sock")
	if got := llamaSocketPath(); got != "/x/llama.sock" {
		t.Fatal(got)
	}
	t.Setenv("KNIRV_LLAMA_SOCKET", "")
	t.Setenv("KNIRV_APP_DATA_DIR", t.TempDir())
	if got := llamaSocketPath(); filepath.Base(got) != "llama.sock" || filepath.Base(filepath.Dir(got)) != "sockets" {
		t.Fatalf("default must be <app data>/sockets/llama.sock, got %s", got)
	}
}

func get(s *Server, path string) (int, string) {
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Code, rec.Body.String()
}

// With Qwen as HEART's backend, health says so, and says it is unavailable (not
// "idle") while llama is not up, so the harness falls back to the external generator.
func TestHeartIsQwenByDefaultAndSaysWhenLlamaIsDown(t *testing.T) {
	t.Setenv("KNIRV_HEART_BACKEND", "")
	t.Setenv("KNIRV_APP_DATA_DIR", t.TempDir())
	t.Setenv("KNIRV_LLAMA_SOCKET", filepath.Join(t.TempDir(), "none.sock"))
	s := NewServer(t.TempDir()+"/h.sock", 0o660)
	code, body := get(s, "/models/heart/health")
	if code != 200 || !strings.Contains(body, "qwen-unavailable") {
		t.Fatalf("%d %s", code, body)
	}
	if code, body := get(s, "/models/manifest"); code != 200 || !strings.Contains(body, "heart.wasm.resolution") {
		t.Fatalf("manifest: %d %.200s", code, body)
	}
}

func TestHeartBackendCanBeKeptOnTheGPT(t *testing.T) {
	t.Setenv("KNIRV_APP_DATA_DIR", t.TempDir())
	s := NewServer(t.TempDir()+"/h.sock", 0o660) // TestMain sets KNIRV_HEART_BACKEND=gpt
	if _, body := get(s, "/models/heart/health"); strings.Contains(body, "qwen") {
		t.Fatalf("opt-out ignored: %s", body)
	}
}

func TestIndexSearchToolRunsAsTheCallersTenant(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KNIRV_EVIDENCE_DIR", dir)
	t.Setenv("KNIRV_APP_DATA_DIR", t.TempDir())
	ix, err := evidence.Open(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ix.Ingest(evidence.Batch{Tier: evidence.TierUser, Tenant: "acme", Docs: []evidence.Doc{{Text: "Acme secret: Project Heron launches 2027 with 55 engineers."}}}); err != nil {
		t.Fatal(err)
	}
	ix.Close()
	s := NewServer(t.TempDir()+"/h.sock", 0o660)
	body := []byte(`{"text":"Project Heron launches 2027","k":3}`)
	own, err := s.routeJSON(http.MethodPost, "/models/hasher/index/search", body, map[string]string{"X-KNIRV-Tenant": "acme"})
	if err != nil || !strings.Contains(string(own), "Project Heron") {
		t.Fatalf("%v %s", err, own)
	}
	other, err := s.routeJSON(http.MethodPost, "/models/hasher/index/search", body, map[string]string{"X-KNIRV-Tenant": "globex"})
	if err != nil || strings.Contains(string(other), "Project Heron") {
		t.Fatalf("leak: %v %s", err, other)
	}
}
