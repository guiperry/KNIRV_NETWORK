package inferencer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

type fakeStack struct {
	llama, ulora *httptest.Server
	registered   atomic.Int32
	bound        atomic.Int32
	loaded       map[string]string // name -> bundle hash
	registerCode int
	bindCode     int
}

func newStack(t *testing.T) *fakeStack {
	f := &fakeStack{loaded: map[string]string{}, registerCode: 201, bindCode: 200}
	lm := http.NewServeMux()
	lm.HandleFunc("GET /v1/adapters", func(w http.ResponseWriter, _ *http.Request) {
		type a struct {
			Name       string `json:"name"`
			Loaded     bool   `json:"loaded"`
			BaseSHA256 string `json:"base_sha256"`
			BundleHash string `json:"bundle_hash"`
		}
		list := []a{}
		for n, h := range f.loaded {
			list = append(list, a{n, true, "base1", h})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"base": map[string]string{"path": "/m/base.gguf", "sha256": "base1"}, "adapters": list})
	})
	lm.HandleFunc("POST /v1/adapters", func(w http.ResponseWriter, r *http.Request) {
		f.registered.Add(1)
		var in struct{ Name, Path string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		if f.registerCode != 201 {
			w.WriteHeader(f.registerCode)
			_, _ = w.Write([]byte(`{"error":{"message":"adapter was bound to a different base model"}}`))
			return
		}
		f.loaded[in.Name] = "hash-" + in.Name
		w.WriteHeader(201)
	})
	f.llama = httptest.NewServer(lm)
	um := http.NewServeMux()
	um.HandleFunc("POST /ulora/v1/adapters/gguf", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(401)
			return
		}
		f.bound.Add(1)
		var in map[string]string
		_ = json.NewDecoder(r.Body).Decode(&in)
		if f.bindCode != 200 {
			http.Error(w, "module self_attn.k_proj is not supported", f.bindCode)
			return
		}
		if in["base_model_path"] != "/m/base.gguf" {
			http.Error(w, "wrong base", 400)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"adapter_path": "/a/x.gguf", "base_sha256": "base1", "content_hash": "abcdef0123456789ffff", "cached": f.bound.Load() > 1})
	})
	f.ulora = httptest.NewServer(um)
	t.Cleanup(func() { f.llama.Close(); f.ulora.Close() })
	return f
}

func (f *fakeStack) sa() *SkillAdapters {
	return &SkillAdapters{Llama: f.llama.Client(), LlamaBase: f.llama.URL, Ulora: f.ulora.Client(), UloraBase: f.ulora.URL, UloraToken: "tok"}
}

func TestEnsureBindsToTheRunningModelThenRegisters(t *testing.T) {
	f := newStack(t)
	got, err := f.sa().Ensure(context.Background(), "b1", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "skill-abcdef0123456789" || got.Reused || !got.BoundNow || got.BaseSHA256 != "base1" || f.registered.Load() != 1 {
		t.Fatalf("%+v registered=%d", got, f.registered.Load())
	}
}

func TestEnsureIsIdempotentAndDoesNotRestartLlamaAgain(t *testing.T) {
	f := newStack(t)
	sa := f.sa()
	if _, err := sa.Ensure(context.Background(), "b1", ""); err != nil {
		t.Fatal(err)
	}
	// llama now reports the adapter loaded under the same bundle hash
	f.loaded["skill-abcdef0123456789"] = "abcdef0123456789ffff"
	got, err := sa.Ensure(context.Background(), "b1", "")
	if err != nil || !got.Reused || f.registered.Load() != 1 {
		t.Fatalf("%+v %v registered=%d (a second Ensure must not restart llama)", got, err, f.registered.Load())
	}
}

func TestEnsureFailuresAreErrorsNamingTheStep(t *testing.T) {
	f := newStack(t)
	f.bindCode = 422
	if _, err := f.sa().Ensure(context.Background(), "b1", ""); err == nil || !strings.Contains(err.Error(), "ulora refused") || !strings.Contains(err.Error(), "k_proj") {
		t.Fatalf("%v", err)
	}
	if f.registered.Load() != 0 {
		t.Fatal("llama was asked to load an adapter that was never bound")
	}
	f = newStack(t)
	f.registerCode = 409
	if _, err := f.sa().Ensure(context.Background(), "b1", ""); err == nil || !strings.Contains(err.Error(), "llama refused") || !strings.Contains(err.Error(), "409") {
		t.Fatalf("%v", err)
	}
	f = newStack(t)
	sa := f.sa()
	sa.UloraToken = "wrong"
	if _, err := sa.Ensure(context.Background(), "b1", ""); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("%v", err)
	}
	f.llama.Close()
	if _, err := f.sa().Ensure(context.Background(), "b1", ""); err == nil || !strings.Contains(err.Error(), "llama adapters unavailable") {
		t.Fatalf("%v", err)
	}
}

func TestAdapterNameIsStableAndValid(t *testing.T) {
	n := AdapterName("ABCDEF0123456789ABCDEF")
	if n != "skill-abcdef0123456789" {
		t.Fatal(n)
	}
}

func TestLlamaProviderForwardsTheAdapter(t *testing.T) {
	p := NewLlamaProvider("", "m", nil).(*LlamaProvider)
	raw, err := p.PrepareRequest("hi", map[string]interface{}{"adapter": "skill-x"})
	if err != nil {
		t.Fatal(err)
	}
	var req map[string]any
	_ = json.Unmarshal(raw, &req)
	if req["adapter"] != "skill-x" {
		t.Fatalf("%s", raw)
	}
	raw, _ = p.PrepareRequest("hi", nil)
	req = nil
	_ = json.Unmarshal(raw, &req)
	if _, has := req["adapter"]; has {
		t.Fatalf("no adapter requested but one was sent: %s", raw)
	}
}
