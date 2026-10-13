package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ulora/internal/gguf"
)

// TestWriteAdaptersForLlamaE2E is the ULoRA half of scripts/gguf-spike/e2e.sh. It
// binds a real bundle to a real llama-loadable GGUF (the tiny GQA model written by
// tinyllama.py) and writes the adapter files the llama half then loads. It only
// runs when the script sets ULORA_E2E_BASE / ULORA_E2E_OUT.
func TestWriteAdaptersForLlamaE2E(t *testing.T) {
	base, out := os.Getenv("ULORA_E2E_BASE"), os.Getenv("ULORA_E2E_OUT")
	if base == "" || out == "" {
		t.Skip("run via scripts/gguf-spike/e2e.sh")
	}
	srv, cfg, cleanup := startTestServer(t)
	defer cleanup()
	srv.SetEnginesDir(filepath.Join("..", "compiler", "engines"))
	// Large core so the adapter visibly changes a random tiny model.
	writeScaledTestBundle(t, cfg.DataDir, "e2e", []string{"self_attn.q_proj", "self_attn.o_proj"}, 60)

	code, body := post(t, cfg, GGUFAdapterRequest{BundleID: "e2e", BaseModelPath: base})
	if code != 200 {
		t.Fatalf("%d %s", code, body)
	}
	var resp GGUFAdapterResponse
	json.Unmarshal(body, &resp)
	if !resp.ConnectorDerived {
		t.Fatal("expected the connector to be derived from the model's own weights")
	}

	// The daemon's data dir is a temp dir that dies with the test; keep a copy.
	kept := filepath.Join(out, "adapter.gguf")
	raw, err := os.ReadFile(resp.AdapterPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(kept, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	// A second adapter bound to some OTHER base model, to prove the llama side refuses it.
	info, _, err := gguf.ReadAdapterInfo(resp.AdapterPath)
	if err != nil {
		t.Fatal(err)
	}
	f, _ := gguf.Open(resp.AdapterPath)
	a, _ := f.Matrix("blk.0.attn_q.weight.lora_a")
	b, _ := f.Matrix("blk.0.attn_q.weight.lora_b")
	info.BaseSHA256 = "f" + info.BaseSHA256[1:]
	wrong := filepath.Join(out, "wrong-base.gguf")
	if err := gguf.WriteLoRA(wrong, info, []gguf.LayerLoRA{{Weight: "blk.0.attn_q.weight", A: a, B: b}}); err != nil {
		t.Fatal(err)
	}
	manifest, _ := json.Marshal(map[string]any{"adapter": kept, "wrong_base": wrong, "base_sha256": resp.BaseSHA256})
	if err := os.WriteFile(filepath.Join(out, "e2e.json"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
}
