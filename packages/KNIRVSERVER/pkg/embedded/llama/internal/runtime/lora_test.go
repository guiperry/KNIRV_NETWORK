package runtime

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newLlamaStub serves the adapter endpoints the way llama-server does: GET
// returns the loaded set, POST replaces scales and rejects a non-array body.
func newLlamaStub(t *testing.T, adapters []LoRAAdapterInfo) (address string, lastPost *[]LoRASelection) {
	t.Helper()

	posted := &[]LoRASelection{}
	state := adapters

	mux := http.NewServeMux()
	mux.HandleFunc("/lora-adapters", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(state)
		case http.MethodPost:
			raw, _ := io.ReadAll(r.Body)
			var selections []LoRASelection
			if err := json.Unmarshal(raw, &selections); err != nil {
				// Mirrors the real server's explicit array check.
				http.Error(w, `{"error":{"message":"Request body must be an array"}}`, http.StatusBadRequest)
				return
			}
			*posted = selections
			for i := range state {
				for _, sel := range selections {
					if state[i].ID == sel.ID {
						state[i].Scale = sel.Scale
					}
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := &httptest.Server{Listener: listener, Config: &http.Server{Handler: mux}}
	server.Start()
	t.Cleanup(server.Close)
	return listener.Addr().String(), posted
}

func TestGetLoRAAdapters(t *testing.T) {
	address, _ := newLlamaStub(t, []LoRAAdapterInfo{
		{ID: 0, Path: "/adapters/ulora-a.gguf", Scale: 1.0},
		{ID: 1, Path: "/adapters/ulora-b.gguf", Scale: 0.0},
	})

	adapters, err := GetLoRAAdapters(context.Background(), address, "")
	if err != nil {
		t.Fatalf("get adapters: %v", err)
	}
	if len(adapters) != 2 {
		t.Fatalf("got %d adapters, want 2", len(adapters))
	}
	if adapters[1].Scale != 0 {
		t.Fatalf("adapter 1 scale = %v, want 0", adapters[1].Scale)
	}
	if adapters[0].ID != 0 || adapters[0].Path != "/adapters/ulora-a.gguf" {
		t.Fatalf("adapter 0 = %+v", adapters[0])
	}
}

// Switching is the point: the POST must actually reach the server carrying the
// requested scales, and the change must be visible on a subsequent GET.
func TestSetLoRAAdaptersSwitchesScales(t *testing.T) {
	address, posted := newLlamaStub(t, []LoRAAdapterInfo{
		{ID: 0, Path: "/adapters/ulora-a.gguf", Scale: 0.0},
		{ID: 1, Path: "/adapters/ulora-b.gguf", Scale: 0.0},
	})

	// Select adapter 1 only, for this call.
	if err := SetLoRAAdapters(context.Background(), address, "", []LoRASelection{
		{ID: 0, Scale: 0},
		{ID: 1, Scale: 1.0},
	}); err != nil {
		t.Fatalf("set adapters: %v", err)
	}

	if len(*posted) != 2 || (*posted)[1].ID != 1 || (*posted)[1].Scale != 1.0 {
		t.Fatalf("server received %+v", *posted)
	}

	adapters, err := GetLoRAAdapters(context.Background(), address, "")
	if err != nil {
		t.Fatalf("get after set: %v", err)
	}
	if adapters[0].Scale != 0 || adapters[1].Scale != 1.0 {
		t.Fatalf("scales after switch = %v, %v; want 0, 1", adapters[0].Scale, adapters[1].Scale)
	}
}

// An empty selection is a caller mistake, not a no-op request to send.
func TestSetLoRAAdaptersRejectsEmptySelection(t *testing.T) {
	if err := SetLoRAAdapters(context.Background(), "127.0.0.1:1", "", nil); err == nil {
		t.Fatal("an empty selection must be refused")
	}
}

// A server too old for the endpoint must be reported as exactly that, rather
// than as a generic transport failure.
func TestLoRAEndpointsReportUnsupportedServer(t *testing.T) {
	mux := http.NewServeMux() // no routes registered -> 404
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := &httptest.Server{Listener: listener, Config: &http.Server{Handler: mux}}
	server.Start()
	defer server.Close()

	_, err = GetLoRAAdapters(context.Background(), listener.Addr().String(), "")
	if err == nil {
		t.Fatal("expected an error from a server without the endpoint")
	}
	if !strings.Contains(err.Error(), "runtime adapter switching is unavailable") {
		t.Fatalf("error should name the missing capability, got: %v", err)
	}
}

func TestLoRAInitWithoutApplyFlag(t *testing.T) {
	// Only meaningful with an adapter present.
	withAdapter := Args("/bin/llama-server", "/m.gguf", "127.0.0.1:8080", Options{
		LoRA:                 []LoRAAdapter{{Path: "/a.gguf"}},
		LoRAInitWithoutApply: true,
	})
	if !contains(withAdapter, "--lora-init-without-apply") {
		t.Fatalf("expected --lora-init-without-apply, got %v", withAdapter)
	}

	// With no adapters it would imply an intent the command line cannot express.
	withoutAdapter := Args("/bin/llama-server", "/m.gguf", "127.0.0.1:8080", Options{
		LoRAInitWithoutApply: true,
	})
	if contains(withoutAdapter, "--lora-init-without-apply") {
		t.Fatalf("flag must be omitted with no adapters, got %v", withoutAdapter)
	}

	// And off by default.
	def := Args("/bin/llama-server", "/m.gguf", "127.0.0.1:8080", Options{
		LoRA: []LoRAAdapter{{Path: "/a.gguf"}},
	})
	if contains(def, "--lora-init-without-apply") {
		t.Fatalf("flag must be opt-in, got %v", def)
	}
}
