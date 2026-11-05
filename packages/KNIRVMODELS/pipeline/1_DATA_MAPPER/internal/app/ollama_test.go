package app

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func fastWaits(t *testing.T, wait, provision time.Duration) {
	t.Helper()
	ow, op, oi := ollamaWait, ollamaProvisionWait, ollamaPoll
	ollamaWait, ollamaProvisionWait, ollamaPoll = wait, provision, 10*time.Millisecond
	t.Cleanup(func() { ollamaWait, ollamaProvisionWait, ollamaPoll = ow, op, oi })
}

// fakeDaemon serves /harness/status on a Unix socket with the given ollama block.
func fakeDaemon(t *testing.T, block func() string) string {
	t.Helper()
	appData := t.TempDir()
	t.Setenv("KNIRV_APP_DATA_DIR", appData)
	sock := filepath.Join(appData, "sockets", "llama.sock")
	if err := os.MkdirAll(filepath.Dir(sock), 0o700); err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"primary_note":"","ollama":%s}`, block())
	})}
	go srv.Serve(l)
	t.Cleanup(func() { srv.Close() })
	return sock
}

func fakeOllama(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			fmt.Fprint(w, `{"models":[]}`)
			return
		}
		fmt.Fprint(w, "Ollama is running")
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestInferenceSocketPathUsesGatewayAppDataConvention(t *testing.T) {
	t.Setenv("KNIRV_APP_DATA_DIR", "/srv/knirv")
	if got, want := inferenceSocketPath(), "/srv/knirv/sockets/llama.sock"; got != want {
		t.Fatalf("inferenceSocketPath() = %q, want %q", got, want)
	}
}

func TestInferenceSocketPathFallsBackToServerDataDirectory(t *testing.T) {
	t.Setenv("KNIRV_APP_DATA_DIR", "")
	if got, want := inferenceSocketPath(), "/var/lib/knirvserver/sockets/llama.sock"; got != want {
		t.Fatalf("inferenceSocketPath() = %q, want %q", got, want)
	}
}

func TestCheckOrStartOllamaReturnsAtOnceWhenRunning(t *testing.T) {
	srv := fakeOllama(t)
	if err := CheckOrStartOllama(srv.URL, "nomic-embed-text"); err != nil {
		t.Fatal(err)
	}
}

// The mapper no longer starts Ollama: with nothing on the PATH and nothing running,
// it must fail with the daemon's account rather than spawn a process.
func TestCheckOrStartOllamaNeverStartsAProcess(t *testing.T) {
	fastWaits(t, 100*time.Millisecond, time.Second)
	marker := filepath.Join(t.TempDir(), "started")
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ollama"), []byte("#!/bin/sh\ntouch "+marker+"\n"), 0o755)
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	err := CheckOrStartOllama("http://127.0.0.1:1", "")
	if err == nil || !strings.Contains(err.Error(), "does not start it") {
		t.Fatalf("err = %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Fatal("the mapper started an ollama process")
	}
}

func TestCheckOrStartOllamaWaitsWhileDaemonProvisions(t *testing.T) {
	srv := fakeOllama(t)
	var calls int32
	fakeDaemon(t, func() string {
		atomic.AddInt32(&calls, 1)
		return `{"phase":"pulling","ready":false,"endpoint":"` + srv.URL + `"}`
	})
	fastWaits(t, 50*time.Millisecond, 5*time.Second)

	// Ollama comes up 400 ms from now, well past the plain wait but inside the
	// provisioning wait.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	go func() {
		time.Sleep(400 * time.Millisecond)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return
		}
		late := &http.Server{Handler: srv.Config.Handler}
		go late.Serve(ln)
		t.Cleanup(func() { late.Close() })
	}()
	if err := CheckOrStartOllama("http://"+addr, ""); err != nil {
		t.Fatalf("did not wait for the provisioning daemon: %v", err)
	}
	if atomic.LoadInt32(&calls) == 0 {
		t.Fatal("the daemon status was never consulted")
	}
}

func TestCheckOrStartOllamaReportsDaemonFailureAndEndpointMismatch(t *testing.T) {
	fastWaits(t, 80*time.Millisecond, time.Second)
	fakeDaemon(t, func() string {
		return `{"phase":"degraded","ready":false,"error":"an Ollama listener on port 11434 is bound to all interfaces"}`
	})
	err := CheckOrStartOllama("http://127.0.0.1:1", "")
	if err == nil || !strings.Contains(err.Error(), "degraded") || !strings.Contains(err.Error(), "all interfaces") {
		t.Fatalf("err = %v", err)
	}

	srv := fakeOllama(t)
	fakeDaemon(t, func() string { return `{"phase":"ready","ready":true,"endpoint":"` + srv.URL + `"}` })
	err = CheckOrStartOllama("http://127.0.0.1:1", "")
	if err == nil || !strings.Contains(err.Error(), "configured for") {
		t.Fatalf("err = %v", err)
	}
}
