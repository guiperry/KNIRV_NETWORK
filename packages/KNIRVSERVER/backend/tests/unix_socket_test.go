package tests

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUnixSocketListening(t *testing.T) {
	tests := []struct {
		name        string
		socketPath  string
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid socket path in /tmp",
			socketPath: "/tmp/test_socket_listen.sock",
			wantErr:    false,
		},
		{
			name:        "socket in non-existent directory",
			socketPath:  "/nonexistent/path/test_socket.sock",
			wantErr:     true,
			errContains: "no such file or directory",
		},
		{
			name:       "empty socket path - uses TCP fallback",
			socketPath: "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var listener net.Listener
			var err error

			if tt.socketPath != "" {
				os.RemoveAll(tt.socketPath)
				defer os.RemoveAll(tt.socketPath)

				dir := filepath.Dir(tt.socketPath)
				if err := os.MkdirAll(dir, 0755); err != nil && !os.IsExist(err) {
					if !tt.wantErr {
						t.Errorf("unexpected error creating directory: %v", err)
					}
					return
				}

				listener, err = net.Listen("unix", tt.socketPath)
			} else {
				listener, err = net.Listen("tcp", "localhost:0")
			}

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
					if listener != nil {
						listener.Close()
					}
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if listener != nil {
				listener.Close()
			}
		})
	}
}

func TestUnixSocketHTTPCommunication(t *testing.T) {
	socketPath := "/tmp/test_http_socket_comm.sock"
	os.RemoveAll(socketPath)
	defer os.RemoveAll(socketPath)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	})

	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		t.Fatalf("failed to create socket directory: %v", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to listen on socket: %v", err)
	}
	defer listener.Close()

	server := &http.Server{Handler: handler}
	go server.Serve(listener)

	time.Sleep(50 * time.Millisecond)

	if err := os.Chmod(socketPath, 0666); err != nil {
		t.Fatalf("failed to set socket permissions: %v", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	resp, err := client.Get("http://unix/health")
	if err != nil {
		t.Fatalf("failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected status 200")

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, `{"status": "ok"}`, string(body), "response body mismatch")
}

func TestUnixSocketPermissions(t *testing.T) {
	socketPath := "/tmp/test_perms_socket.sock"
	os.RemoveAll(socketPath)
	defer os.RemoveAll(socketPath)

	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		t.Fatalf("failed to create socket directory: %v", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to listen on socket: %v", err)
	}

	// Set permissions while socket is alive
	if err := os.Chmod(socketPath, 0666); err != nil {
		listener.Close()
		t.Fatalf("failed to set socket permissions: %v", err)
	}
	listener.Close()

	// Note: On Linux, the socket file is removed when the listener closes.
	// This test verifies the permissions can be set while socket is active.
	// The actual permissions would be applied via the listener.
}

func TestConcurrentSocketAccess(t *testing.T) {
	socketPath := "/tmp/test_concurrent_sock.sock"
	os.RemoveAll(socketPath)
	defer os.RemoveAll(socketPath)

	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		t.Fatalf("failed to create socket directory: %v", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to listen on socket: %v", err)
	}

	var wg sync.WaitGroup
	numClients := 10
	errorChan := make(chan error, numClients)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{Handler: handler}
	go server.Serve(listener)
	defer server.Close()

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			client := &http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
						return net.Dial("unix", socketPath)
					},
				},
			}

			resp, err := client.Get("http://unix/health")
			if err != nil {
				errorChan <- err
				return
			}
			resp.Body.Close()
		}()
	}

	wg.Wait()

	close(errorChan)
	for err := range errorChan {
		t.Errorf("client error: %v", err)
	}
}

func TestSocketPathFromEnv(t *testing.T) {
	tests := []struct {
		name      string
		socketEnv string
		wantPath  string
	}{
		{
			name:      "env variable set",
			socketEnv: "/var/run/test.sock",
			wantPath:  "/var/run/test.sock",
		},
		{
			name:      "env variable empty",
			socketEnv: "",
			wantPath:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.socketEnv != "" {
				os.Setenv("SOCKET_PATH", tt.socketEnv)
				defer os.Unsetenv("SOCKET_PATH")
			} else {
				os.Unsetenv("SOCKET_PATH")
			}

			socketPath := os.Getenv("SOCKET_PATH")
			assert.Equal(t, tt.wantPath, socketPath)
		})
	}
}

func TestTCPFallback(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to listen on TCP: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	assert.NotZero(t, addr.Port, "expected a non-zero port")

	t.Logf("TCP server listening on port %d", addr.Port)
}

func TestSocketDirectoryCreation(t *testing.T) {
	tests := []struct {
		name      string
		socketDir string
		wantErr   bool
	}{
		{
			name:      "valid directory",
			socketDir: "/tmp/socket_test_dir",
			wantErr:   false,
		},
		{
			name:      "nested directory",
			socketDir: "/tmp/nested/socket/dir",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.RemoveAll(tt.socketDir)
			defer os.RemoveAll(tt.socketDir)

			err := os.MkdirAll(tt.socketDir, 0755)
			if tt.wantErr && err != nil {
				return
			}
			assert.NoError(t, err)

			info, err := os.Stat(tt.socketDir)
			assert.NoError(t, err)
			assert.True(t, info.IsDir())
		})
	}
}

func TestSocketCleanup(t *testing.T) {
	socketPath := "/tmp/test_cleanup.sock"

	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		t.Fatalf("failed to create socket directory: %v", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to listen on socket: %v", err)
	}
	listener.Close()

	_, err = os.Stat(socketPath)
	assert.Error(t, err, "socket file should be cleaned up after listener close")
}

func TestSocketVsTCPPortAvailability(t *testing.T) {
	socketPath := "/tmp/test_avail.sock"
	os.RemoveAll(socketPath)
	defer os.RemoveAll(socketPath)

	os.MkdirAll(filepath.Dir(socketPath), 0755)

	listener1, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to listen on socket: %v", err)
	}
	defer listener1.Close()

	_, err = net.Listen("unix", socketPath)
	assert.Error(t, err, "should not be able to bind to same socket twice")

	listener1.Close()

	listener2, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to re-listen on socket after cleanup: %v", err)
	}
	listener2.Close()
}

func BenchmarkUnixSocketHTTP(b *testing.B) {
	socketPath := "/tmp/bench_socket.sock"
	os.RemoveAll(socketPath)
	defer os.RemoveAll(socketPath)

	os.MkdirAll(filepath.Dir(socketPath), 0755)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		b.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	os.Chmod(socketPath, 0666)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{Handler: handler}
	go server.Serve(listener)

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get("http://unix/")
		if err != nil {
			b.Fatalf("request failed: %v", err)
		}
		resp.Body.Close()
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
