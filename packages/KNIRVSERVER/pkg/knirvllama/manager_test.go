package knirvllama

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
)

func TestSetEnvOverrideReplacesInheritedValue(t *testing.T) {
	env := []string{"LLAMA_ADDRESS=stale.example", "OTHER=value", "LLAMA_ADDRESS=older.example"}
	got := setEnvOverride(env, "LLAMA_ADDRESS", "127.0.0.1:8081")
	want := []string{"OTHER=value", "LLAMA_ADDRESS=127.0.0.1:8081"}
	if len(got) != len(want) {
		t.Fatalf("setEnvOverride() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("setEnvOverride()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestDefaultManagerConfig(t *testing.T) {
	cfg := DefaultManagerConfig()
	if cfg.ListenAddr == "" {
		t.Error("DefaultManagerConfig ListenAddr should not be empty")
	}
	if cfg.DataDir == "" {
		t.Error("DefaultManagerConfig DataDir should not be empty")
	}
	if cfg.LlamaAddress != DefaultLlamaAddress {
		t.Errorf("DefaultManagerConfig LlamaAddress = %q, want %q", cfg.LlamaAddress, DefaultLlamaAddress)
	}
	if cfg.StartTimeout == 0 {
		t.Error("DefaultManagerConfig StartTimeout should not be zero")
	}
	if cfg.StopTimeout == 0 {
		t.Error("DefaultManagerConfig StopTimeout should not be zero")
	}
}

func TestNewManagerDefaults(t *testing.T) {
	m := NewManager(nil, nil)
	if m == nil {
		t.Fatal("NewManager(nil, nil) returned nil")
	}
	if m.config == nil {
		t.Fatal("NewManager(nil, nil) config is nil")
	}
	if m.listenAddr == "" {
		t.Error("NewManager listenAddr should not be empty")
	}
	if m.logger == nil {
		t.Error("NewManager must provide a no-op logger when none is supplied")
	}
}

func TestManagerGetHealth(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("local TCP listeners unavailable in this environment: %v", err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("health request path = %q, want /health", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()

	m := NewManager(&ManagerConfig{ListenAddr: strings.TrimPrefix(server.URL, "http://")}, nil)
	// A started command is not needed to validate the wrapper's endpoint; a
	// non-nil command represents the launched child for this lightweight test.
	m.cmd = &exec.Cmd{}
	healthy, err := m.GetHealth()
	if err != nil || !healthy {
		t.Fatalf("GetHealth() = (%t, %v), want (true, nil)", healthy, err)
	}
}

func TestManagerGetStatus(t *testing.T) {
	m := NewManager(nil, nil)
	status := m.GetStatus()
	if status == nil {
		t.Fatal("GetStatus returned nil")
	}
	if status.Status != "stopped" {
		t.Errorf("expected status 'stopped', got '%s'", status.Status)
	}
	if status.Running {
		t.Error("expected Running to be false for new manager")
	}
	if status.ListenAddr != DefaultListenAddr {
		t.Errorf("expected ListenAddr '%s', got '%s'", DefaultListenAddr, status.ListenAddr)
	}
}
