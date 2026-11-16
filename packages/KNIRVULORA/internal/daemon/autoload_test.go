package daemon

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ulora/internal/config"
	"ulora/internal/gguf"
)

// newAutoloadServer prepares a data dir, lets the test write autoload.json and
// adapter files into it, and only then starts the server (auto-load runs in
// NewServer). Logs are captured so refusals can be asserted.
func newAutoloadServer(t *testing.T, setup func(cfg *config.Config)) (*Server, *config.Config, *bytes.Buffer) {
	t.Helper()
	cfg := newTestConfig(t)
	if err := os.MkdirAll(filepath.Join(cfg.DataDir, "adapters"), 0o755); err != nil {
		t.Fatal(err)
	}
	setup(cfg)

	var logBuf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&logBuf)
	t.Cleanup(func() { log.SetOutput(prev) })

	srv, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve() }()
	t.Cleanup(func() {
		srv.Shutdown()
		<-errCh
	})
	return srv, cfg, &logBuf
}

// writeAdapter writes a real GGUF LoRA file recording baseSHA as its base.
func writeAdapter(t *testing.T, path, baseSHA, skill string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	layers := []gguf.LayerLoRA{{Weight: "blk.0.attn_q.weight", A: mat(2, tHidden), B: mat(tHidden, 2)}}
	if err := gguf.WriteLoRA(path, gguf.AdapterInfo{
		Architecture: "llama", Alpha: 32, BaseSHA256: baseSHA, BundleHash: strings.Repeat("c", 64),
		Skill: skill, Modules: "attn_q",
	}, layers); err != nil {
		t.Fatalf("write adapter: %v", err)
	}
}

func writeDescriptor(t *testing.T, cfg *config.Config, entries []AutoloadEntry) {
	t.Helper()
	raw, err := json.Marshal(AutoloadFile{Version: autoloadVersion, Adapters: entries})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(AutoloadPath(cfg.DataDir), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func listAutoloaded(t *testing.T, cfg *config.Config) []LoadedAdapter {
	t.Helper()
	resp := rawSocketCall(t, cfg, http.MethodGet, "/ulora/v1/adapters/autoloaded", nil, cfg.AuthToken)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("autoloaded listing returned %d", resp.StatusCode)
	}
	var out struct {
		Adapters []LoadedAdapter `json:"adapters"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out.Adapters
}

// An adapter whose recorded base matches the configured base model is loaded at
// start, and the loaded record reports the verified digest.
func TestAutoloadLoadsAdapterWhoseBaseMatches(t *testing.T) {
	var base string
	_, cfg, logs := newAutoloadServer(t, func(cfg *config.Config) {
		base = writeTinyBase(t, t.TempDir(), 2)
		baseSHA := sha256File(t, base)
		writeAdapter(t, filepath.Join(cfg.DataDir, "adapters", "ok.gguf"), baseSHA, "skill-ok")
		writeDescriptor(t, cfg, []AutoloadEntry{{AdapterPath: "ok.gguf", BaseModelPath: base}})
	})

	got := listAutoloaded(t, cfg)
	if len(got) != 1 {
		t.Fatalf("loaded %d adapters, want 1; logs: %s", len(got), logs.String())
	}
	if got[0].Skill != "skill-ok" || got[0].BaseSHA256 != sha256File(t, base) {
		t.Fatalf("loaded record %+v", got[0])
	}
	if !strings.Contains(logs.String(), "loaded adapter") {
		t.Fatalf("no load log line: %s", logs.String())
	}
}

// An adapter merged into a different base is refused and logged; it is not
// listed as loaded. A sibling entry that does match still loads.
func TestAutoloadRefusesBaseMismatchAndKeepsGoodEntries(t *testing.T) {
	var goodBase, otherBase string
	_, cfg, logs := newAutoloadServer(t, func(cfg *config.Config) {
		goodBase = writeTinyBase(t, t.TempDir(), 2)
		otherBase = writeTinyBase(t, t.TempDir(), 1) // different weights, different digest
		if sha256File(t, goodBase) == sha256File(t, otherBase) {
			t.Fatal("test bases must differ")
		}
		goodSHA := sha256File(t, goodBase)
		// Adapter targets goodBase but the descriptor points at otherBase.
		writeAdapter(t, filepath.Join(cfg.DataDir, "adapters", "wrong.gguf"), goodSHA, "skill-wrong")
		writeAdapter(t, filepath.Join(cfg.DataDir, "adapters", "right.gguf"), goodSHA, "skill-right")
		writeDescriptor(t, cfg, []AutoloadEntry{
			{AdapterPath: "wrong.gguf", BaseModelPath: otherBase},
			{AdapterPath: "right.gguf", BaseModelPath: goodBase},
		})
	})

	got := listAutoloaded(t, cfg)
	if len(got) != 1 || got[0].Skill != "skill-right" {
		t.Fatalf("loaded %+v, want only skill-right", got)
	}
	logText := logs.String()
	if !strings.Contains(logText, "refused adapter") || !strings.Contains(logText, "base model mismatch") {
		t.Fatalf("mismatch was not logged as a refusal: %s", logText)
	}
	// Digests are logged (not secret), but no key material is ever present here;
	// the refusal must name the adapter it refused.
	if !strings.Contains(logText, "wrong.gguf") {
		t.Fatalf("refusal does not name the adapter: %s", logText)
	}
}

// An adapter that records no base digest cannot be verified and is refused.
func TestAutoloadRefusesAdapterWithoutRecordedBase(t *testing.T) {
	var base string
	_, cfg, logs := newAutoloadServer(t, func(cfg *config.Config) {
		base = writeTinyBase(t, t.TempDir(), 2)
		writeAdapter(t, filepath.Join(cfg.DataDir, "adapters", "nobase.gguf"), "", "skill-nobase")
		writeDescriptor(t, cfg, []AutoloadEntry{{AdapterPath: "nobase.gguf", BaseModelPath: base}})
	})
	if got := listAutoloaded(t, cfg); len(got) != 0 {
		t.Fatalf("adapter with no recorded base was loaded: %+v", got)
	}
	if !strings.Contains(logs.String(), "records no base_sha256") {
		t.Fatalf("missing refusal reason: %s", logs.String())
	}
}

// With no descriptor file, start is normal and nothing is loaded or logged.
func TestAutoloadAbsentDescriptorIsNotAnError(t *testing.T) {
	_, cfg, logs := newAutoloadServer(t, func(cfg *config.Config) {})
	if got := listAutoloaded(t, cfg); len(got) != 0 {
		t.Fatalf("loaded %+v with no descriptor", got)
	}
	if strings.Contains(logs.String(), "autoload") {
		t.Fatalf("unexpected autoload log with no descriptor: %s", logs.String())
	}
}

// A descriptor of an unknown version loads nothing rather than guessing.
func TestAutoloadRejectsUnknownDescriptorVersion(t *testing.T) {
	_, cfg, logs := newAutoloadServer(t, func(cfg *config.Config) {
		base := writeTinyBase(t, t.TempDir(), 2)
		writeAdapter(t, filepath.Join(cfg.DataDir, "adapters", "v.gguf"), sha256File(t, base), "s")
		raw, _ := json.Marshal(AutoloadFile{Version: 99, Adapters: []AutoloadEntry{{AdapterPath: "v.gguf", BaseModelPath: base}}})
		if err := os.WriteFile(AutoloadPath(cfg.DataDir), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	})
	if got := listAutoloaded(t, cfg); len(got) != 0 {
		t.Fatalf("loaded %+v from unknown version", got)
	}
	if !strings.Contains(logs.String(), "unsupported") {
		t.Fatalf("missing version log: %s", logs.String())
	}
}
