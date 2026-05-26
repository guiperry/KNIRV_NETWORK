package knirvarena

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

type ManagerConfig struct {
	SocketPath   string
	ExtractPath  string
	StartTimeout time.Duration
	StopTimeout  time.Duration
}

type Manager struct {
	config      *ManagerConfig
	logger      *zap.Logger
	server      *http.Server
	listener    net.Listener
	extractPath string
	mu          sync.RWMutex
	running     bool
}

type HealthStatus struct {
	Status      string    `json:"status"`
	Running     bool      `json:"running"`
	SocketPath  string    `json:"socket_path"`
	ExtractPath string    `json:"extract_path"`
	Timestamp   time.Time `json:"timestamp"`
}

func getAppDataDir() string {
	if explicit := os.Getenv("KNIRV_APP_DATA_DIR"); explicit != "" {
		return explicit
	}
	if err := os.MkdirAll("/var/lib/knirvserver", 0755); err == nil {
		return "/var/lib/knirvserver"
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "knirvserver")
	}
	return filepath.Join(os.TempDir(), "knirvserver", "data")
}

func DefaultConfig() *ManagerConfig {
	appDataDir := getAppDataDir()
	return &ManagerConfig{
		SocketPath:   filepath.Join(appDataDir, "sockets", "arena.sock"),
		ExtractPath:  filepath.Join(appDataDir, "knirvarena"),
		StartTimeout: 10 * time.Second,
		StopTimeout:  10 * time.Second,
	}
}

func NewManager(cfg *ManagerConfig, logger *zap.Logger) *Manager {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &Manager{
		config: cfg,
		logger: logger,
	}
}

func (m *Manager) Extract() (string, error) {
	extractPath, err := ExtractEmbeddedApp(m.config.ExtractPath)
	if err != nil {
		return "", fmt.Errorf("failed to extract arena bundle: %w", err)
	}
	m.extractPath = extractPath
	m.logger.Info("KNIRVARENA bundle extracted", zap.String("path", extractPath))
	return extractPath, nil
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		m.logger.Info("KNIRVARENA already running")
		return nil
	}

	if _, err := m.Extract(); err != nil {
		return err
	}

	socketDir := filepath.Dir(m.config.SocketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	if err := os.Remove(m.config.SocketPath); err != nil && !os.IsNotExist(err) {
		m.logger.Warn("Failed to remove stale socket", zap.String("path", m.config.SocketPath), zap.Error(err))
	}

	listener, err := net.Listen("unix", m.config.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on Unix socket %s: %w", m.config.SocketPath, err)
	}
	if err := os.Chmod(m.config.SocketPath, 0777); err != nil {
		listener.Close()
		return fmt.Errorf("failed to chmod socket: %w", err)
	}
	m.listener = listener

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"knirvarena","timestamp":"%s"}`, time.Now().UTC().Format(time.RFC3339))
	})

	mux.HandleFunc("/ws", m.handleWebSocket)

	fileServer := http.FileServer(http.Dir(m.extractPath))
	mux.Handle("/", fileServer)

	m.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		m.logger.Info("KNIRVARENA server started on Unix socket", zap.String("socket", m.config.SocketPath))
		if err := m.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			m.logger.Error("KNIRVARENA server error", zap.Error(err))
		}
	}()

	m.running = true

	go func() {
		if !m.waitForHealth(m.config.StartTimeout) {
			m.logger.Warn("KNIRVARENA did not become healthy within timeout")
			return
		}
		m.logger.Info("KNIRVARENA health check passed")
	}()

	return nil
}

func (m *Manager) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") != "websocket" {
		http.Error(w, "not a websocket request", http.StatusBadRequest)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	m.logger.Info("KNIRVARENA WebSocket connection established",
		zap.String("remote", conn.RemoteAddr().String()))

	bufrw.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	bufrw.WriteString("Upgrade: websocket\r\n")
	bufrw.WriteString("Connection: Upgrade\r\n")
	bufrw.WriteString("Sec-WebSocket-Accept: " + computeAcceptKey(r.Header.Get("Sec-WebSocket-Key")) + "\r\n")
	bufrw.WriteString("\r\n")
	bufrw.Flush()

	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		conn.Write(buf[:n])
	}
}

func computeAcceptKey(key string) string {
	if key == "" {
		return ""
	}
	const magicGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.New()
	h.Write([]byte(key + magicGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (m *Manager) waitForHealth(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if m.IsHealthy() {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running || m.server == nil {
		m.logger.Info("KNIRVARENA not running")
		return nil
	}

	m.logger.Info("Stopping KNIRVARENA server")

	ctx, cancel := context.WithTimeout(context.Background(), m.config.StopTimeout)
	defer cancel()

	if err := m.server.Shutdown(ctx); err != nil {
		m.logger.Warn("KNIRVARENA shutdown error", zap.Error(err))
	}

	if m.listener != nil {
		m.listener.Close()
	}

	os.Remove(m.config.SocketPath)

	m.running = false
	m.logger.Info("KNIRVARENA stopped")
	return nil
}

func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func (m *Manager) IsHealthy() bool {
	if !m.IsRunning() {
		return false
	}
	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", m.config.SocketPath)
			},
		},
	}
	resp, err := client.Get("http://localhost/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}

func (m *Manager) GetHealthStatus() *HealthStatus {
	status := &HealthStatus{
		Status:      "stopped",
		Running:     false,
		SocketPath:  m.config.SocketPath,
		ExtractPath: m.extractPath,
		Timestamp:   time.Now(),
	}
	if !m.IsRunning() {
		return status
	}
	status.Running = true
	if m.IsHealthy() {
		status.Status = "healthy"
	} else {
		status.Status = "unhealthy"
	}
	return status
}

func (m *Manager) GetSocketPath() string {
	return m.config.SocketPath
}

func (m *Manager) GetExtractPath() string {
	return m.extractPath
}

func (m *Manager) GetConfig() *ManagerConfig {
	return m.config
}
