package headless

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"knirvhasher/internal/analyzer"
	"knirvhasher/internal/cli/controller"
)

const (
	DefaultSocketPath = "/var/run/knirvhasher.sock"
	DefaultSocketPerm = 0660
)

type Server struct {
	controller         *controller.Controller
	mux                *http.ServeMux
	server             *http.Server
	socketPath         string
	socketPerm         os.FileMode
	ln                 net.Listener
	shutdownCh         chan struct{}
	mu                 sync.RWMutex
	goatMode           bool
	knirvserverManaged bool
}

func NewServer(socketPath string, socketPerm os.FileMode, goatModes ...bool) *Server {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}
	if socketPerm == 0 {
		socketPerm = DefaultSocketPerm
	}

	goatMode := false
	if len(goatModes) > 0 {
		goatMode = goatModes[0]
	}
	s := &Server{
		controller: controller.NewController(goatMode),
		mux:        http.NewServeMux(),
		socketPath: socketPath,
		socketPerm: socketPerm,
		shutdownCh: make(chan struct{}),
		goatMode:   goatMode,
	}

	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/api/v1/status", s.handleGetStatus)
	s.mux.HandleFunc("/api/v1/health", s.handleHealth)

	s.mux.HandleFunc("/api/v1/driver/start", s.handleStartDriver)
	s.mux.HandleFunc("/api/v1/driver/stop", s.handleStopDriver)
	s.mux.HandleFunc("/api/v1/driver/status", s.handleGetDriverStatus)

	s.mux.HandleFunc("/api/v1/pipeline/run", s.handleRunPipeline)
	s.mux.HandleFunc("/api/v1/pipeline/status", s.handleGetPipelineStatus)
	s.mux.HandleFunc("/api/v1/pipeline/logs", s.handlePipelineLogs)

	s.mux.HandleFunc("/api/v1/verify", s.handleVerify)

	s.mux.HandleFunc("/api/v1/asic/discover", s.handleDiscoverASIC)
	s.mux.HandleFunc("/api/v1/asic/probe", s.handleProbeASIC)
	s.mux.HandleFunc("/api/v1/asic/protocol", s.handleDetectProtocol)
	s.mux.HandleFunc("/api/v1/asic/provision", s.handleProvisionASIC)
	s.mux.HandleFunc("/api/v1/asic/troubleshoot", s.handleTroubleshootASIC)
	s.mux.HandleFunc("/api/v1/asic/configure", s.handleConfigureASIC)
	s.mux.HandleFunc("/api/v1/asic/test", s.handleTestASIC)

	s.mux.HandleFunc("/api/v1/rules", s.handleManageRules)

	s.mux.HandleFunc("/api/v1/chat", s.handleChat)

	s.mux.HandleFunc("/api/v1/shutdown", s.handleShutdown)
}

func (s *Server) Start() error {
	if err := s.setupSocket(); err != nil {
		return err
	}

	s.server = &http.Server{
		Handler: s.mux,
	}

	log.Printf("Headless server listening on unix://%s", s.socketPath)

	go func() {
		if err := s.server.Serve(s.ln); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	// In the standalone CLI, starting the driver remains an explicit user action.
	// KNIRVSERVER, however, owns the full service lifecycle.  When it provides an
	// ASIC address through root.key, start hasher-host automatically; hasher-host
	// then provisions and starts hasher-server on the ASIC.
	if s.knirvserverManaged {
		s.startConfiguredDriver()
	}

	return nil
}

func (s *Server) startConfiguredDriver() {
	deviceIP := strings.TrimSpace(os.Getenv("DEVICE_IP"))
	if deviceIP == "" {
		log.Printf("[ASIC] No DEVICE_IP configured; hasher-host auto-start skipped")
		return
	}

	log.Printf("[ASIC] DEVICE_IP=%s configured; starting hasher-host (driver) for hasher-server deployment", deviceIP)
	go func() {
		if err := s.controller.StartDriver(context.Background()); err != nil {
			log.Printf("[ASIC] Failed to start hasher-host for %s: %v", deviceIP, err)
			return
		}
		log.Printf("[ASIC] hasher-host started for %s; provisioning/health logs follow", deviceIP)
		for output := range s.controller.GetLogChan() {
			for _, line := range strings.Split(output, "\n") {
				if line = strings.TrimSpace(line); line != "" {
					log.Printf("[hasher-host] %s", line)
				}
			}
		}
	}()
}

func (s *Server) setupSocket() error {
	if err := os.RemoveAll(s.socketPath); err != nil {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	socketDir := filepath.Dir(s.socketPath)
	if socketDir != "." && socketDir != "/" {
		if err := os.MkdirAll(socketDir, 0755); err != nil {
			return fmt.Errorf("failed to create socket directory: %w", err)
		}
	}

	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket at %s: %w", s.socketPath, err)
	}

	if err := os.Chmod(s.socketPath, s.socketPerm); err != nil {
		ln.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	s.ln = ln
	return nil
}

func (s *Server) Stop() error {
	log.Println("Stopping headless server...")

	if s.ln != nil {
		s.ln.Close()
	}

	if err := os.RemoveAll(s.socketPath); err != nil {
		log.Printf("Warning: failed to remove socket: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	s.controller.Cleanup()

	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
		"socket": s.socketPath,
	})
}

func (s *Server) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := s.controller.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleStartDriver(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	err := s.controller.StartDriver(ctx)
	if err != nil {
		s.sendError(w, "Failed to start driver", err, http.StatusInternalServerError)
		return
	}

	s.sendSuccess(w, "Driver started successfully", nil)
}

func (s *Server) handleStopDriver(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	err := s.controller.StopDriver(ctx)
	if err != nil {
		s.sendError(w, "Failed to stop driver", err, http.StatusInternalServerError)
		return
	}

	s.sendSuccess(w, "Driver stopped successfully", nil)
}

func (s *Server) handleGetDriverStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := s.controller.GetDriverStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleRunPipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request body", err, http.StatusBadRequest)
		return
	}

	if req.Type == "" {
		req.Type = "goat"
	}

	s.runPipelineAsync(req.Type, "Pipeline")

	s.sendSuccess(w, "Pipeline started", map[string]string{"type": req.Type})
}

func (s *Server) handleGetPipelineStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := s.controller.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VerifyRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.sendError(w, "Invalid request body", err, http.StatusBadRequest)
			return
		}
	}

	mode := req.Mode
	if mode == "" {
		mode = r.URL.Query().Get("mode")
	}
	if mode == "" {
		mode = "semantic"
	}
	ctx := r.Context()

	switch mode {
	case "mathematical":
		err := s.controller.RunMathVerifier(ctx)
		if err != nil {
			s.sendError(w, "Mathematical verification failed", err, http.StatusInternalServerError)
			return
		}
		s.sendSuccess(w, "Mathematical verification started", map[string]string{"mode": mode})

	case "semantic":
		s.runPipelineAsync("goat", "Semantic verification")
		s.sendSuccess(w, "Semantic verification (pipeline) started", map[string]string{"mode": mode})

	default:
		s.sendError(w, "Invalid verification mode", fmt.Errorf("mode must be 'semantic' or 'mathematical'"), http.StatusBadRequest)
	}
}

func (s *Server) runPipelineAsync(pipelineType, label string) {
	go func() {
		// This work outlives the HTTP request. It also has a panic boundary:
		// pipeline defects must fail the run, never the long-lived headless
		// process (and therefore never KNIRVSERVER).
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("%s pipeline panic recovered: %v", label, recovered)
			}
		}()
		if err := s.controller.RunPipeline(context.Background(), pipelineType); err != nil {
			log.Printf("%s error: %v", label, err)
		}
	}()
}

func (s *Server) handleDiscoverASIC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	result, err := s.controller.DiscoverASIC(ctx)
	if err != nil {
		s.sendError(w, "Failed to discover ASIC devices", err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DiscoveryResponse{
		Devices:      s.devicesToMaps(result.Devices),
		SelectedIP:   result.SelectedIP,
		SelectedType: result.SelectedType,
		Output:       result.Output,
		Duration:     result.Duration,
	})
}

func (s *Server) handleProbeASIC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	err := s.controller.ProbeASIC(ctx)
	if err != nil {
		s.sendError(w, "Failed to probe ASIC", err, http.StatusInternalServerError)
		return
	}

	s.sendSuccess(w, "ASIC probe completed successfully", nil)
}

func (s *Server) handleDetectProtocol(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	err := s.controller.DetectProtocol(ctx)
	if err != nil {
		s.sendError(w, "Failed to detect protocol", err, http.StatusInternalServerError)
		return
	}

	s.sendSuccess(w, "Protocol detection completed successfully", nil)
}

func (s *Server) handleProvisionASIC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DeviceIP string `json:"device_ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request body", err, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	err := s.controller.ProvisionASIC(ctx, req.DeviceIP)
	if err != nil {
		s.sendError(w, "Failed to provision ASIC", err, http.StatusInternalServerError)
		return
	}

	s.sendSuccess(w, "ASIC provisioning completed successfully", nil)
}

func (s *Server) handleTroubleshootASIC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	err := s.controller.TroubleshootASIC(ctx)
	if err != nil {
		s.sendError(w, "Failed to troubleshoot ASIC", err, http.StatusInternalServerError)
		return
	}

	s.sendSuccess(w, "ASIC troubleshooting completed successfully", nil)
}

func (s *Server) handleTestASIC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	err := s.controller.TestASIC(ctx)
	if err != nil {
		s.sendError(w, "Failed to test ASIC", err, http.StatusInternalServerError)
		return
	}

	s.sendSuccess(w, "ASIC test completed successfully", nil)
}

func (s *Server) handleConfigureASIC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	output, err := s.controller.ConfigureASIC(ctx)
	if err != nil {
		s.sendError(w, "Failed to get ASIC configuration", err, http.StatusInternalServerError)
		return
	}

	s.sendSuccess(w, output, nil)
}

func (s *Server) handleManageRules(w http.ResponseWriter, r *http.Request) {
	var req RulesRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.sendError(w, "Invalid request body", err, http.StatusBadRequest)
			return
		}
	}

	ctx := r.Context()
	var args []string
	if req.Domain != "" {
		args = append(args, req.Domain)
	}
	if req.RuleType != "" {
		args = append(args, req.RuleType)
	}
	if req.Conclusion != "" {
		args = append(args, req.Conclusion)
	}

	output, err := s.controller.ManageRules(ctx, req.Action, args...)
	if err != nil {
		s.sendError(w, "Failed to manage rules", err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": output,
	})
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request body", err, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	response, err := s.controller.SendChat(ctx, req.Message)
	if err != nil {
		s.sendError(w, "Failed to send chat message", err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{
		Response:  response,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Println("Shutdown requested via API")

	go func() {
		time.Sleep(100 * time.Millisecond)
		close(s.shutdownCh)
	}()

	s.sendSuccess(w, "Shutdown initiated", nil)
}

func (s *Server) handlePipelineLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	logChan := s.controller.GetProgressChan()
	for {
		select {
		case msg, ok := <-logChan:
			if !ok {
				return
			}
			data, _ := json.Marshal(msg)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) sendError(w http.ResponseWriter, message string, err error, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   err.Error(),
		Code:    code,
		Message: message,
	})
}

func (s *Server) sendSuccess(w http.ResponseWriter, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GenericResponse{
		Message: message,
		Data:    data,
	})
}

func (s *Server) devicesToMaps(devices []analyzer.DeviceInfo) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(devices))
	for _, d := range devices {
		result = append(result, map[string]interface{}{
			"ip_address":  d.IPAddress,
			"device_type": d.DeviceType,
			"accessible":  d.Accessible,
			"protocol":    d.Protocol.String(),
		})
	}
	return result
}

func RunHeadless(socketPath string, socketPerm uint32, goatMode bool, knirvserverManaged ...bool) error {
	perm := os.FileMode(socketPerm)
	if socketPerm == 0 {
		perm = DefaultSocketPerm
	}

	server := NewServer(socketPath, perm, goatMode)
	if len(knirvserverManaged) > 0 {
		server.knirvserverManaged = knirvserverManaged[0]
	}

	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		log.Println("Received shutdown signal")
	case <-server.shutdownCh:
		log.Println("Received shutdown request via API")
	}

	return server.Stop()
}

func IsSocketReady(socketPath string) bool {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func WaitForSocketReady(socketPath string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if IsSocketReady(socketPath) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("socket not ready after %v", timeout)
}

func GetSocketConnections(socketPath string) (int, error) {
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		link, err := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
		if err != nil {
			continue
		}
		if link == socketPath {
			count++
		}
	}
	return count, nil
}

func ParseSocketPerm(permStr string) (os.FileMode, error) {
	if permStr == "" {
		return DefaultSocketPerm, nil
	}
	perm, err := strconv.ParseUint(permStr, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid socket permission: %w", err)
	}
	return os.FileMode(perm), nil
}

func extractLogLine(line string) string {
	line = strings.TrimSpace(line)
	if len(line) > 200 {
		return line[:197] + "..."
	}
	return line
}

func pipeOutput(reader *bufio.Reader, label string) (<-chan string, func()) {
	ch := make(chan string, 100)
	done := make(chan struct{})

	go func() {
		defer close(ch)
		for {
			select {
			case <-done:
				return
			default:
				line, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				select {
				case ch <- extractLogLine(line):
				default:
				}
			}
		}
	}()

	stop := func() { close(done) }
	return ch, stop
}
