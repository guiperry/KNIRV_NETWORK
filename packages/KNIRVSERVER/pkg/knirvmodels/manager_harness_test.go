package knirvmodels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"testing"
)

// serveHarnessStatus answers /harness/status on a Unix socket like the inference
// instance does.
func serveHarnessStatus(t *testing.T, body string) string {
	t.Helper()
	sock := filepath.Join(t.TempDir(), "llama.sock")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/harness/status" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, body)
	})}
	go srv.Serve(l)
	t.Cleanup(func() { srv.Close() })
	return sock
}

// GetStatus carries the inference status document's blocks through untouched, so a
// block KNIRVMODELS adds later (the embedding server's phase and digest today) shows
// in the models status without SERVER knowing its name.
func TestGetStatusCarriesHarnessBlocksFromTheDaemon(t *testing.T) {
	sock := serveHarnessStatus(t, `{"primary_note":"","embedder":{"ready":true,"endpoint":"http://127.0.0.1:11434","digest":"0a109f","phase":"ready"}}`)
	inf := NewInference(InferenceConfig{SocketPath: sock})
	m := &Manager{}
	m.SetHarnessStatusSource(inf.HarnessStatus)

	st := m.GetStatus()
	raw, ok := st.Harness["embedder"]
	if !ok {
		t.Fatalf("block missing: %v", st.Harness)
	}
	var blk struct {
		Ready    bool
		Endpoint string
		Digest   string
	}
	if err := json.Unmarshal(raw, &blk); err != nil || !blk.Ready || blk.Endpoint != "http://127.0.0.1:11434" || blk.Digest != "0a109f" {
		t.Fatalf("block %s (%v)", raw, err)
	}
	// And it survives the JSON the status endpoint serves.
	out, _ := json.Marshal(st)
	var back struct {
		Harness map[string]any `json:"harness"`
	}
	if err := json.Unmarshal(out, &back); err != nil || back.Harness["embedder"].(map[string]any)["ready"] != true {
		t.Fatalf("serialized status %s", out)
	}
}

func TestGetStatusWithoutOrWithFailingSourceStillAnswers(t *testing.T) {
	m := &Manager{}
	if st := m.GetStatus(); st == nil || st.Harness != nil {
		t.Fatalf("no source: %+v", st)
	}
	m.SetHarnessStatusSource(func(context.Context) (map[string]json.RawMessage, error) { return nil, errors.New("down") })
	if st := m.GetStatus(); st == nil || st.Harness != nil || st.Status != "stopped" {
		t.Fatalf("failing source: %+v", st)
	}
	// An unreachable socket is the same: best effort.
	m.SetHarnessStatusSource(NewInference(InferenceConfig{SocketPath: filepath.Join(t.TempDir(), "none.sock")}).HarnessStatus)
	if st := m.GetStatus(); st == nil || st.Harness != nil {
		t.Fatalf("dead socket: %+v", st)
	}
}
