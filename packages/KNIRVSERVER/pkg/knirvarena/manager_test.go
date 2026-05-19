package knirvarena

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestLogger() *zap.Logger {
	return zap.NewNop()
}

func tempSocketPath(t *testing.T) string {
	return filepath.Join(t.TempDir(), "arena.sock")
}

func TestNewManager_DefaultConfig(t *testing.T) {
	m := NewManager(nil, newTestLogger())
	assert.NotNil(t, m)
	assert.NotNil(t, m.config)
	assert.Equal(t, 10*time.Second, m.config.StartTimeout)
	assert.Equal(t, 10*time.Second, m.config.StopTimeout)
	assert.Contains(t, m.config.SocketPath, "arena.sock")
	assert.False(t, m.IsRunning())
}

func TestNewManager_CustomConfig(t *testing.T) {
	socketPath := tempSocketPath(t)
	cfg := &ManagerConfig{
		SocketPath:   socketPath,
		ExtractPath:  t.TempDir(),
		StartTimeout: 5 * time.Second,
		StopTimeout:  5 * time.Second,
	}
	m := NewManager(cfg, newTestLogger())
	assert.Equal(t, socketPath, m.config.SocketPath)
	assert.Equal(t, 5*time.Second, m.config.StartTimeout)
}

func TestManager_StartStop(t *testing.T) {
	socketPath := tempSocketPath(t)
	extractPath := t.TempDir()

	m := NewManager(&ManagerConfig{
		SocketPath:   socketPath,
		ExtractPath:  extractPath,
		StartTimeout: 5 * time.Second,
		StopTimeout:  5 * time.Second,
	}, newTestLogger())

	err := m.Start()
	require.NoError(t, err)
	assert.True(t, m.IsRunning())

	// Socket file should exist
	_, err = os.Stat(socketPath)
	assert.NoError(t, err, "socket file should exist")

	// Server should be healthy
	assert.True(t, m.IsHealthy())

	// Second start should be idempotent
	err = m.Start()
	assert.NoError(t, err)
	assert.True(t, m.IsRunning())

	err = m.Stop()
	assert.NoError(t, err)
	assert.False(t, m.IsRunning())

	// Socket file should be cleaned up
	_, err = os.Stat(socketPath)
	assert.True(t, os.IsNotExist(err), "socket file should be removed on stop")
}

func TestManager_Stop_Idempotent(t *testing.T) {
	m := NewManager(nil, newTestLogger())
	err := m.Stop()
	assert.NoError(t, err, "stopping non-running manager should be idempotent")
}

func TestManager_HealthEndpoint(t *testing.T) {
	socketPath := tempSocketPath(t)

	m := NewManager(&ManagerConfig{
		SocketPath:   socketPath,
		ExtractPath:  t.TempDir(),
		StartTimeout: 5 * time.Second,
		StopTimeout:  5 * time.Second,
	}, newTestLogger())

	require.NoError(t, m.Start())
	defer m.Stop()

	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	resp, err := client.Get("http://localhost/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestManager_GetHealthStatus(t *testing.T) {
	socketPath := tempSocketPath(t)

	m := NewManager(&ManagerConfig{
		SocketPath:   socketPath,
		ExtractPath:  t.TempDir(),
		StartTimeout: 5 * time.Second,
		StopTimeout:  5 * time.Second,
	}, newTestLogger())

	// Before start
	status := m.GetHealthStatus()
	assert.Equal(t, "stopped", status.Status)
	assert.False(t, status.Running)

	require.NoError(t, m.Start())
	defer m.Stop()

	// After start
	status = m.GetHealthStatus()
	assert.Equal(t, "healthy", status.Status)
	assert.True(t, status.Running)
	assert.Equal(t, socketPath, status.SocketPath)
}

func TestManager_GetSocketPath(t *testing.T) {
	socketPath := tempSocketPath(t)
	m := NewManager(&ManagerConfig{SocketPath: socketPath}, newTestLogger())
	assert.Equal(t, socketPath, m.GetSocketPath())
}

func TestManager_GetConfig(t *testing.T) {
	cfg := &ManagerConfig{SocketPath: "/tmp/test.sock"}
	m := NewManager(cfg, newTestLogger())
	assert.Equal(t, cfg, m.GetConfig())
}

func TestManager_WebSocketUpgrade(t *testing.T) {
	socketPath := tempSocketPath(t)

	m := NewManager(&ManagerConfig{
		SocketPath:   socketPath,
		ExtractPath:  t.TempDir(),
		StartTimeout: 5 * time.Second,
		StopTimeout:  5 * time.Second,
	}, newTestLogger())

	require.NoError(t, m.Start())
	defer m.Stop()

	conn, err := net.Dial("unix", socketPath)
	require.NoError(t, err)
	defer conn.Close()

	key := "dGhlIHNhbXBsZSBub25jZQ=="
	req := "GET /ws HTTP/1.1\r\nHost: localhost\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: " + key + "\r\nSec-WebSocket-Version: 13\r\n\r\n"
	_, err = conn.Write([]byte(req))
	require.NoError(t, err)

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	require.NoError(t, err)

	resp := string(buf[:n])
	assert.Contains(t, resp, "101 Switching Protocols")
	assert.Contains(t, resp, "Sec-WebSocket-Accept")
}
