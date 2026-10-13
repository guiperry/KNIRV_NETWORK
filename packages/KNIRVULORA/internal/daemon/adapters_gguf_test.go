package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"ulora/internal/config"
	"ulora/internal/connector"
	"ulora/internal/gguf"
	"ulora/internal/llamatarget"
)

const (
	tHidden = 8
	tHeads  = 4
	tLayers = 2
)

// writeTinyBase writes a GGUF with the metadata and attention weights of a small
// grouped-query llama (4 heads, kvHeads kv heads). It is not loadable by llama.cpp
// (the real-model test does that); it is exactly what ULoRA reads.
func writeTinyBase(t *testing.T, dir string, kvHeads int) string {
	t.Helper()
	meta := []gguf.Meta{
		{Key: "general.architecture", Value: "llama"}, {Key: "llama.embedding_length", Value: uint32(tHidden)},
		{Key: "llama.block_count", Value: uint32(tLayers)}, {Key: "llama.attention.head_count", Value: uint32(tHeads)},
		{Key: "llama.attention.head_count_kv", Value: uint32(kvHeads)}, {Key: "llama.feed_forward_length", Value: uint32(16)},
	}
	var ts []gguf.WriteTensor
	kv := kvHeads * (tHidden / tHeads)
	for l := 0; l < tLayers; l++ {
		for _, w := range []struct {
			n    string
			rows int
		}{{"attn_q", tHidden}, {"attn_k", kv}, {"attn_v", kv}, {"attn_output", tHidden}} {
			m := make([][]float32, w.rows)
			for i := range m {
				m[i] = make([]float32, tHidden)
				for j := range m[i] {
					m[i][j] = float32((i+1)*(j+2)%5) + float32(l)
				}
			}
			ts = append(ts, gguf.WriteTensor{Name: fmt.Sprintf("blk.%d.%s.weight", l, w.n), Rows: m})
		}
	}
	p := filepath.Join(dir, "base.gguf")
	if err := gguf.Write(p, meta, ts); err != nil {
		t.Fatal(err)
	}
	return p
}

func seedConnector(t *testing.T, cfg *config.Config, base string) *llamatarget.Target {
	t.Helper()
	tg, err := llamatarget.FromGGUF(base, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := connector.NewCache(filepath.Join(cfg.DataDir, "connectors")).Put(&connector.Connector{
		Family: tg.Spec.Family, ParamCount: tg.Spec.ParamCount, CanonicalDim: canonicalDim,
		PIn: mat(canonicalDim, tHidden), POut: mat(tHidden, canonicalDim),
	}); err != nil {
		t.Fatal(err)
	}
	return tg
}

func post(t *testing.T, cfg *config.Config, body any) (int, []byte) {
	t.Helper()
	raw, _ := json.Marshal(body)
	resp := rawSocketCall(t, cfg, http.MethodPost, "/ulora/v1/adapters/gguf", raw, cfg.AuthToken)
	defer resp.Body.Close()
	var b bytes.Buffer
	b.ReadFrom(resp.Body)
	return resp.StatusCode, b.Bytes()
}

func TestGGUFAdapterBindsQAndOAcrossEveryLayer(t *testing.T) {
	_, cfg, cleanup := startTestServer(t)
	defer cleanup()
	base := writeTinyBase(t, t.TempDir(), 2) // GQA: 2 kv heads of 4
	tg := seedConnector(t, cfg, base)
	if tg.Spec.AttentionMechanism != "gqa" || tg.KVDim != 4 || tg.QDim != 8 {
		t.Fatalf("target %+v", tg)
	}
	bundle := writeTestBundle(t, cfg.DataDir, "s1", []string{"self_attn.q_proj", "self_attn.o_proj"})
	wantHash := sha256File(t, bundle)

	code, body := post(t, cfg, GGUFAdapterRequest{BundleID: "s1", ContentHash: wantHash, BaseModelPath: base})
	if code != 200 {
		t.Fatalf("%d %s", code, body)
	}
	var resp GGUFAdapterResponse
	json.Unmarshal(body, &resp)
	if resp.Cached || resp.ContentHash != wantHash || resp.BaseSHA256 != tg.SHA256 || resp.Rank != 2 || resp.Alpha != 32 {
		t.Fatalf("%+v", resp)
	}
	info, f, err := gguf.ReadAdapterInfo(resp.AdapterPath)
	if err != nil || info.BaseSHA256 != tg.SHA256 || info.BundleHash != wantHash || info.Alpha != 32 || info.Architecture != "llama" {
		t.Fatalf("%+v %v", info, err)
	}
	// 2 modules x 2 layers x (A, B)
	if len(f.Tensors) != 8 {
		t.Fatalf("%d tensors", len(f.Tensors))
	}
	for l := 0; l < tLayers; l++ {
		for _, n := range []string{"attn_q", "attn_output"} {
			a, err := f.Matrix(fmt.Sprintf("blk.%d.%s.weight.lora_a", l, n))
			b, err2 := f.Matrix(fmt.Sprintf("blk.%d.%s.weight.lora_b", l, n))
			if err != nil || err2 != nil {
				t.Fatal(err, err2)
			}
			// llama.cpp: A is rank x in, B is out x rank
			if len(a) != 2 || len(a[0]) != tHidden || len(b) != tHidden || len(b[0]) != 2 {
				t.Fatalf("layer %d %s: A %dx%d B %dx%d", l, n, len(a), len(a[0]), len(b), len(b[0]))
			}
		}
	}
	// A = A_core(r x K) . P_in(K x d): every entry is a sum over K of two mat() values.
	a, _ := f.Matrix("blk.0.attn_q.weight.lora_a")
	var want float32
	for k := 0; k < canonicalDim; k++ {
		want += mat(rank, canonicalDim)[0][k] * mat(canonicalDim, tHidden)[k][0]
	}
	if d := a[0][0] - want; d > 1e-2 || d < -1e-2 {
		t.Fatalf("A[0][0]=%v want %v: the projection is not A_core . P_in", a[0][0], want)
	}

	// Same bundle + same base -> served from disk, not rebound.
	code, body = post(t, cfg, GGUFAdapterRequest{BundleID: "s1", BaseModelPath: base})
	json.Unmarshal(body, &resp)
	if code != 200 || !resp.Cached {
		t.Fatalf("second call %d cached=%v %s", code, resp.Cached, body)
	}
}

func TestGGUFAdapterRefusesKVAndNamesWhy(t *testing.T) {
	_, cfg, cleanup := startTestServer(t)
	defer cleanup()
	base := writeTinyBase(t, t.TempDir(), 2)
	seedConnector(t, cfg, base)
	writeTestBundle(t, cfg.DataDir, "kv", []string{"self_attn.q_proj", "self_attn.k_proj"})
	code, body := post(t, cfg, GGUFAdapterRequest{BundleID: "kv", BaseModelPath: base})
	if code != http.StatusUnprocessableEntity || !strings.Contains(string(body), "k_proj") || !strings.Contains(string(body), "kv(4)") {
		t.Fatalf("%d %s", code, body)
	}
	if _, err := os.Stat(filepath.Join(cfg.DataDir, "adapters")); err == nil {
		entries, _ := os.ReadDir(filepath.Join(cfg.DataDir, "adapters"))
		for _, e := range entries {
			if e.IsDir() {
				t.Fatalf("a refused bind left an adapter directory behind: %s", e.Name())
			}
		}
	}
}

func TestGGUFAdapterInputRefusals(t *testing.T) {
	_, cfg, cleanup := startTestServer(t)
	defer cleanup()
	base := writeTinyBase(t, t.TempDir(), 4)
	seedConnector(t, cfg, base)
	writeTestBundle(t, cfg.DataDir, "ok", []string{"self_attn.q_proj"})
	for name, c := range map[string]struct {
		req  GGUFAdapterRequest
		want int
	}{
		"no base":      {GGUFAdapterRequest{BundleID: "ok"}, 400},
		"missing base": {GGUFAdapterRequest{BundleID: "ok", BaseModelPath: "/nope.gguf"}, 400},
		"unknown":      {GGUFAdapterRequest{BundleID: "zzz", BaseModelPath: base}, 400},
		"stale hash":   {GGUFAdapterRequest{BundleID: "ok", BaseModelPath: base, ContentHash: strings.Repeat("0", 64)}, 409},
		"not a gguf":   {GGUFAdapterRequest{BundleID: "ok", BaseModelPath: filepath.Join(cfg.DataDir, "bundles", "ulora-ok.ulora")}, 422},
	} {
		if code, body := post(t, cfg, c.req); code != c.want {
			t.Errorf("%s: %d %s, want %d", name, code, body, c.want)
		}
	}
	// MHA target (kv == heads) with a source declared as GQA: Q/O only, so allowed.
	if code, body := post(t, cfg, GGUFAdapterRequest{BundleID: "ok", BaseModelPath: base}); code != 200 {
		t.Fatalf("mha target for a gqa-declared q/o core: %d %s", code, body)
	}
	// ...but with a K module the attention variants must match.
	writeTestBundle(t, cfg.DataDir, "kmod", []string{"self_attn.k_proj"})
	if code, _ := post(t, cfg, GGUFAdapterRequest{BundleID: "kmod", BaseModelPath: base}); code == 200 {
		t.Fatal("a K module must not bind")
	}
}

func TestGGUFAdapterRequiresAuth(t *testing.T) {
	_, cfg, cleanup := startTestServer(t)
	defer cleanup()
	resp := rawSocketCall(t, cfg, http.MethodPost, "/ulora/v1/adapters/gguf", []byte(`{}`), "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
		t.Fatalf("%d", resp.StatusCode)
	}
}

// Without a seeded connector the daemon derives one from the base model's own Q
// projection with the real python engine.
func TestGGUFAdapterDerivesConnectorFromTheModelsOwnWeights(t *testing.T) {
	if exec.Command("python3", "-c", "import numpy, safetensors").Run() != nil {
		t.Skip("python3 with numpy+safetensors is required for the connector engine")
	}
	srv, cfg, cleanup := startTestServer(t)
	defer cleanup()
	srv.SetEnginesDir(filepath.Join("..", "compiler", "engines"))
	base := writeTinyBase(t, t.TempDir(), 2)
	writeTestBundle(t, cfg.DataDir, "d1", []string{"self_attn.q_proj", "self_attn.o_proj"})
	code, body := post(t, cfg, GGUFAdapterRequest{BundleID: "d1", BaseModelPath: base})
	if code != 200 {
		t.Fatalf("%d %s", code, body)
	}
	var resp GGUFAdapterResponse
	json.Unmarshal(body, &resp)
	if !resp.ConnectorDerived || resp.ConnectorDigest == "" {
		t.Fatalf("%+v", resp)
	}
	tg, _ := llamatarget.FromGGUF(base, "")
	c, err := connector.NewCache(filepath.Join(cfg.DataDir, "connectors")).Get(tg.Spec.Family, tg.Spec.ParamCount, canonicalDim)
	if err != nil || c == nil || len(c.PIn) != canonicalDim || len(c.PIn[0]) != tHidden {
		t.Fatalf("connector not cached with the model's shape: %v %v", c, err)
	}
	// A different file (another finetune) must not reuse this model's connector.
	other := writeTinyBase(t, t.TempDir(), 2)
	o2, _ := llamatarget.FromGGUF(other, strings.Repeat("a", 64))
	if o2.Spec.Family == tg.Spec.Family {
		t.Fatal("the connector cache key does not depend on the file hash")
	}
}
