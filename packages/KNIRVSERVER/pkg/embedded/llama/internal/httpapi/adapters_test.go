package httpapi

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/guiperry/knirv/llama/internal/adapters"
)

var baseSHA = strings.Repeat("b", 64)

func adapterFile(t *testing.T, dir, name, base string) string {
	t.Helper()
	le := binary.LittleEndian
	var b []byte
	str := func(s string) { b = le.AppendUint64(b, uint64(len(s))); b = append(b, s...) }
	kv := [][2]string{{"general.type", "adapter"}, {"adapter.type", "lora"}, {"general.architecture", "llama"}, {"knirv.base_sha256", base}}
	b = append(b, "GGUF"...)
	b = le.AppendUint32(b, 3)
	b = le.AppendUint64(b, 0)
	b = le.AppendUint64(b, uint64(len(kv)))
	for _, p := range kv {
		str(p[0])
		b = le.AppendUint32(b, 8)
		str(p[1])
	}
	p := filepath.Join(dir, name+".gguf")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

type upstream struct {
	mu   sync.Mutex
	last map[string]any
	n    int
}

func (u *upstream) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		u.mu.Lock()
		u.last = map[string]any{}
		_ = json.Unmarshal(raw, &u.last)
		u.n++
		u.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
}

func setup(t *testing.T) (http.Handler, *adapters.Manager, *upstream, string) {
	t.Helper()
	up := &upstream{}
	srv := httptest.NewServer(up.handler())
	t.Cleanup(srv.Close)
	dir := t.TempDir()
	st, _ := adapters.OpenStore(dir)
	mgr := adapters.NewManager(st, adapters.Base{Model: "m", Path: "/m.gguf", SHA256: func() (string, error) { return baseSHA, nil }},
		func(context.Context, []adapters.Entry) error { return nil }, adapters.NewGate())
	mgr.DrainTimeout = 200 * time.Millisecond
	h, err := NewWithAdapters(strings.TrimPrefix(srv.URL, "http://"), "m", "", mgr)
	if err != nil {
		t.Fatal(err)
	}
	return h, mgr, up, dir
}

func call(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	return w
}

func TestRegisterOverHTTPThenSelectByName(t *testing.T) {
	h, _, up, dir := setup(t)
	for _, n := range []string{"alpha", "beta"} {
		w := call(h, "POST", "/v1/adapters", `{"name":"`+n+`","path":"`+adapterFile(t, dir, n, baseSHA)+`"}`)
		if w.Code != 201 {
			t.Fatalf("%s: %d %s", n, w.Code, w.Body)
		}
	}
	w := call(h, "POST", "/v1/chat/completions", `{"messages":[],"adapter":"beta","temperature":0}`)
	if w.Code != 200 || w.Header().Get(AdapterHeader) != "beta" {
		t.Fatalf("%d %v", w.Code, w.Header())
	}
	if _, has := up.last["adapter"]; has {
		t.Fatal("the adapter name leaked to llama-server")
	}
	lora, _ := json.Marshal(up.last["lora"])
	if string(lora) != `[{"id":0,"scale":0},{"id":1,"scale":1}]` {
		t.Fatalf("lora = %s", lora)
	}
	if up.last["temperature"] != float64(0) {
		t.Fatal("other fields must pass through untouched")
	}
}

func TestNoAdapterIsAnExplicitBaseRequest(t *testing.T) {
	h, _, up, dir := setup(t)
	call(h, "POST", "/v1/adapters", `{"name":"alpha","path":"`+adapterFile(t, dir, "alpha", baseSHA)+`"}`)
	w := call(h, "POST", "/v1/completions", `{"prompt":"x"}`)
	if w.Header().Get(AdapterHeader) != "base" {
		t.Fatalf("%v", w.Header())
	}
	lora, _ := json.Marshal(up.last["lora"])
	if string(lora) != `[{"id":0,"scale":0}]` {
		t.Fatalf("with an adapter loaded, a base request must say scale 0 explicitly, got %s", lora)
	}
}

func TestNoAdaptersLoadedForwardsUnchanged(t *testing.T) {
	h, _, up, _ := setup(t)
	call(h, "POST", "/v1/chat/completions", `{"messages":[]}`)
	if _, has := up.last["lora"]; has {
		t.Fatal("injected lora with nothing loaded")
	}
}

func TestUnknownAdapterIs404AndNeverReachesTheBaseModel(t *testing.T) {
	h, _, up, _ := setup(t)
	w := call(h, "POST", "/v1/chat/completions", `{"messages":[],"adapter":"ghost"}`)
	if w.Code != 404 || !strings.Contains(w.Body.String(), "ghost") || up.n != 0 {
		t.Fatalf("%d %s upstream calls=%d", w.Code, w.Body, up.n)
	}
}

func TestClientsCannotSendRawLora(t *testing.T) {
	h, _, up, _ := setup(t)
	if w := call(h, "POST", "/v1/chat/completions", `{"lora":[{"id":0,"scale":1}]}`); w.Code != 400 || up.n != 0 {
		t.Fatalf("%d", w.Code)
	}
	if w := call(h, "POST", "/v1/chat/completions", `{"adapter":7}`); w.Code != 400 {
		t.Fatalf("%d", w.Code)
	}
	if w := call(h, "POST", "/v1/chat/completions", `[1]`); w.Code != 400 {
		t.Fatalf("%d", w.Code)
	}
}

func TestRegisterErrorsMapToStatuses(t *testing.T) {
	h, _, _, dir := setup(t)
	for name, c := range map[string]struct {
		body string
		want int
	}{
		"other base": {`{"name":"x","path":"` + adapterFile(t, dir, "x", strings.Repeat("c", 64)) + `"}`, 409},
		"bad name":   {`{"name":"X Y","path":"` + adapterFile(t, dir, "y", baseSHA) + `"}`, 422},
		"bad body":   {`nope`, 400},
	} {
		if w := call(h, "POST", "/v1/adapters", c.body); w.Code != c.want {
			t.Errorf("%s: %d %s", name, w.Code, w.Body)
		}
	}
	if w := call(h, "DELETE", "/v1/adapters/ghost", ""); w.Code != 404 {
		t.Errorf("delete: %d", w.Code)
	}
}

func TestListShowsLoadedIdsAndBase(t *testing.T) {
	h, _, _, dir := setup(t)
	call(h, "POST", "/v1/adapters", `{"name":"alpha","path":"`+adapterFile(t, dir, "alpha", baseSHA)+`"}`)
	w := call(h, "GET", "/v1/adapters", "")
	var out struct {
		Base     map[string]any
		Adapters []adapterView
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || out.Base["sha256"] != baseSHA ||
		len(out.Adapters) != 1 || !out.Adapters[0].Loaded || out.Adapters[0].ID == nil || *out.Adapters[0].ID != 0 {
		t.Fatalf("%s %v", w.Body, err)
	}
	if w := call(h, "DELETE", "/v1/adapters/alpha", ""); w.Code != 204 {
		t.Fatalf("%d", w.Code)
	}
	w = call(h, "GET", "/v1/adapters", "")
	if !strings.Contains(w.Body.String(), `"adapters":[]`) {
		t.Fatalf("%s", w.Body)
	}
}

func TestRequestsDuringReloadWaitThenSucceed(t *testing.T) {
	h, mgr, _, dir := setup(t)
	call(h, "POST", "/v1/adapters", `{"name":"a","path":"`+adapterFile(t, dir, "a", baseSHA)+`"}`)
	resume, err := mgr.Gate().Pause(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan int, 1)
	go func() { done <- call(h, "POST", "/v1/chat/completions", `{"adapter":"a"}`).Code }()
	time.Sleep(60 * time.Millisecond)
	resume()
	if code := <-done; code != 200 {
		t.Fatalf("a request held during a reload got %d", code)
	}
}
