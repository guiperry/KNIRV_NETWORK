package inferencer

import (
	"compress/gzip"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	models "github.com/guiperry/knirv-sdk-go/models"
)

// TestHarnessAgainstTheRealDaemon runs the harness and the status builder against the
// real KNIRVMODELS daemon binary on a real unix socket. Set KNIRV_MODELS_BIN to the
// binary (plain or gzipped, e.g. KNIRVSERVER/pkg/knirvmodels/bin/knirvmodels); skipped
// otherwise.
func TestHarnessAgainstTheRealDaemon(t *testing.T) {
	src := os.Getenv("KNIRV_MODELS_BIN")
	if src == "" {
		t.Skip("KNIRV_MODELS_BIN not set")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "knirvmodels")
	in, err := os.Open(src)
	if err != nil {
		t.Skipf("daemon binary not available: %v", err)
	}
	defer in.Close()
	var r io.Reader = in
	if zr, zerr := gzip.NewReader(in); zerr == nil {
		r = zr
	} else if _, err := in.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	out, err := os.OpenFile(bin, os.O_CREATE|os.O_WRONLY, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(out, r); err != nil {
		t.Fatal(err)
	}
	out.Close()

	sock := filepath.Join(dir, "m.sock")
	cmd := exec.Command(bin, "--headless", "--socket-path", sock)
	cmd.Env = append(os.Environ(), "KNIRV_APP_DATA_DIR="+filepath.Join(dir, "data"))
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
	for i := 0; i < 100; i++ {
		if _, err := os.Stat(sock); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	client := models.NewClient("http://models", models.NewUnixHTTPClient(sock, 30*time.Second))
	head, heart, hasher := DaemonModels(client)
	h := NewHarness([]HarnessModel{head, heart, hasher}, WithChooser(head, 0.6))
	ctx := context.Background()

	// HASHER really answers, through the harness.
	res, tr, err := h.Do(ctx, Task{Kind: KindValidate, Input: models.HasherValidateRequest{Prediction: 1, Domain: "smoke"}})
	if err != nil || tr.Chosen != "hasher" {
		t.Fatalf("validate: %v %+v", err, tr)
	}
	if v, ok := res.Value.(*models.HasherValidateResult); !ok || !v.Valid {
		t.Fatalf("%+v", res.Value)
	}
	// HEART is idle: never built by asking, and the harness reports why it is not used.
	if _, tr, err := h.Do(ctx, Task{Kind: KindGenerate, Input: "hi"}); err == nil || tr.Candidates[0].Excluded == "" {
		t.Fatalf("an idle HEART must be excluded with a reason and (here, with no other model) the call fails: %v %+v", err, tr.Candidates)
	}

	s := BuildPlatformStatus(ctx, client, nil, h, 10)
	if s.DaemonError != "" || len(s.Daemon) != 3 {
		t.Fatalf("registry: %+v", s)
	}
	states := map[string]string{}
	for _, l := range s.Roles() {
		states[l.Model] = l.State
	}
	if states["hasher"] != "ready" || states["heart"] != "idle" {
		t.Fatalf("role states from the real daemon: %v", states)
	}
	if states["head"] == "ready" {
		t.Fatalf("a fresh data dir has no published HEAD: %v", states)
	}
	if len(s.Routing) != 2 || s.Routing[0].Kind != "generate" || s.Routing[1].Chosen != "hasher" {
		t.Fatalf("routing history: %+v", s.Routing)
	}
}
