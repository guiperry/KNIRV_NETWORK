package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"knirv-server/internal/monitor/aggregator"
	"knirv-server/internal/monitor/probes"
)

type Server struct {
	config    *ServerConfig
	startTime time.Time
	metrics   *aggregator.ProcessMetrics
	registry  *aggregator.Registry
	probes    *probes.ProbeManager
	mu        sync.Mutex
	http      *http.Server
	listener  net.Listener
}

func NewServer(cfg *ServerConfig) *Server {
	srv := &Server{
		config:    cfg,
		startTime: time.Now(),
		metrics:   aggregator.NewProcessMetrics(),
		registry:  aggregator.NewRegistry(),
		probes:    probes.NewProbeManager(),
	}

	if cfg.KNIRVBaseURL != "" {
		srv.probes.Register(probes.NewKNIRVBaseProbe(cfg.KNIRVBaseURL))
	}
	if cfg.KNIRVChainURL != "" {
		srv.probes.Register(probes.NewKNIRVChainProbe(cfg.KNIRVChainURL))
	}
	if cfg.KNIRVGraphURL != "" {
		srv.probes.Register(probes.NewKNIRVGraphProbe(cfg.KNIRVGraphURL))
	}
	if cfg.KNIRVOracleURL != "" {
		srv.probes.Register(probes.NewKNIRVOracleProbe(cfg.KNIRVOracleURL))
	}
	if strings.TrimSpace(cfg.RegistryURL) != "" {
		srv.registry.RootStatus.Set(0)
	}
	// KNIRVGATEWAY is intentionally NOT registered as a generic Prometheus
	// probe here: /api/network-monitor/routes returns JSON, not Prometheus
	// exposition text, so the generic line-oriented scraper in probes.go
	// would silently yield nothing useful. handleGatewayRoutes/handleGatewayHealth
	// below fetch and decode it directly instead.

	return srv
}

func (s *Server) Probes() *probes.ProbeManager {
	return s.probes
}

func (s *Server) Registry() *aggregator.Registry {
	return s.registry
}

func (s *Server) HasBackendSocket() bool { return s.config.BackendSocketPath != "" }

// Start binds the monitor's private Unix socket and serves it until Shutdown
// is called. The listener is deliberately owned by KNIRVSERVER rather than a
// separate monitor subprocess, so its lifecycle is tied to the main server.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", s.handleHealthz)
	// /health is an alias of /healthz for the main server's monitor proxy and
	// for consistency with the other embedded services.
	mux.HandleFunc("/health", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.Handle("/metrics", promhttp.HandlerFor(s.registry.Registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/api/v1/monitor/root-failover", s.handleRootFailover)
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/actuarial/metrics", s.handleActuarialMetrics)
	mux.HandleFunc("/api/v1/knirvbase/metrics", s.handleKNIRVBaseMetrics)
	mux.HandleFunc("/api/v1/knirvbase/health", s.handleKNIRVBaseHealth)
	mux.HandleFunc("/api/v1/knirvchain/metrics", s.handleKNIRVChainMetrics)
	mux.HandleFunc("/api/v1/knirvchain/health", s.handleKNIRVChainHealth)
	mux.HandleFunc("/api/v1/network/interfaces", s.handleNetworkInterfaces)
	mux.HandleFunc("/api/v1/network/stats", s.handleNetworkStats)
	mux.HandleFunc("/api/v1/knirvgraph/scalability", s.handleKNIRVGraphScalability)
	mux.HandleFunc("/api/v1/knirvgraph/embeddings", s.handleKNIRVGraphEmbeddings)
	mux.HandleFunc("/api/v1/knirvgraph/health", s.handleKNIRVGraphHealth)
	mux.HandleFunc("/api/v1/knirvoracle/economics", s.handleKNIRVOracleEconomics)
	mux.HandleFunc("/api/v1/knirvoracle/health", s.handleKNIRVOracleHealth)
	mux.HandleFunc("/api/v1/gateway/routes", s.handleGatewayRoutes)
	mux.HandleFunc("/api/v1/gateway/health", s.handleGatewayHealth)
	mux.HandleFunc("/api/v1/onboarding/applications", s.handleOnboardingApplications)
	// Subtree pattern (trailing "/") — coexists with the exact match above in
	// stdlib ServeMux, matches e.g. "/api/v1/onboarding/applications/{id}/review".
	mux.HandleFunc("/api/v1/onboarding/applications/", s.handleOnboardingApplicationAction)
	mux.HandleFunc("/api/v1/onboarding/users", s.handleOnboardingUsers)
	mux.HandleFunc("/api/v1/onboarding/users/", s.handleOnboardingUserAction)

	if strings.TrimSpace(s.config.SocketPath) == "" {
		return fmt.Errorf("monitor socket path is required")
	}
	if err := os.MkdirAll(filepath.Dir(s.config.SocketPath), 0750); err != nil {
		return fmt.Errorf("create monitor socket directory: %w", err)
	}
	// A prior crashed monitor can leave its filesystem entry behind. Removing
	// this exact configured path is safe; the listener below is the owner.
	if err := os.Remove(s.config.SocketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale monitor socket: %w", err)
	}
	listener, err := net.Listen("unix", s.config.SocketPath)
	if err != nil {
		return fmt.Errorf("listen on monitor socket: %w", err)
	}
	if err := os.Chmod(s.config.SocketPath, 0660); err != nil {
		listener.Close()
		return fmt.Errorf("set monitor socket permissions: %w", err)
	}
	s.mu.Lock()
	if s.http != nil {
		s.mu.Unlock()
		listener.Close()
		return fmt.Errorf("monitor server already started")
	}
	s.listener = listener
	s.http = &http.Server{Handler: s.loggingMiddleware(mux)}
	httpServer := s.http
	s.mu.Unlock()

	go func() {
		if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			println("KNIRVMONITOR server failed:", err.Error())
		}
	}()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.Shutdown(shutdownCtx)
	}()
	if strings.TrimSpace(s.config.RegistryURL) != "" {
		go s.pollRegistryRoot(ctx)
	}
	return nil
}

func (s *Server) pollRegistryRoot(ctx context.Context) {
	interval := s.config.ScrapeInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	client := &http.Client{Timeout: s.config.RequestTimeout}
	if client.Timeout <= 0 {
		client.Timeout = 5 * time.Second
	}
	poll := func() {
		url := strings.TrimRight(s.config.RegistryURL, "/")
		if !strings.Contains(url, "/state/") {
			url += "/state/knirvoracle-1"
		}
		resp, err := client.Get(url)
		if err != nil {
			s.registry.RootStatus.Set(0)
			s.registry.ScrapeErrors.Inc()
			return
		}
		defer resp.Body.Close()
		var state struct {
			Phase         string `json:"phase"`
			CurrentRootID string `json:"currentRootID"`
		}
		if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&state) != nil || strings.TrimSpace(state.CurrentRootID) == "" {
			s.registry.RootStatus.Set(0)
			return
		}
		switch state.Phase {
		case "NORMAL", "CONFIRMED", "RECLAIMED":
			s.registry.RootStatus.Set(1)
		default:
			s.registry.RootStatus.Set(0)
		}
	}
	poll()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			poll()
		}
	}
}

// Shutdown stops the private listener and removes only its configured socket.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	httpServer := s.http
	listener := s.listener
	s.http = nil
	s.listener = nil
	s.mu.Unlock()

	if httpServer == nil {
		return nil
	}
	err := httpServer.Shutdown(ctx)
	if listener != nil {
		_ = listener.Close()
	}
	if removeErr := os.Remove(s.config.SocketPath); removeErr != nil && !os.IsNotExist(removeErr) && err == nil {
		err = removeErr
	}
	return err
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if os.Getenv("NETWORK_MONITOR_LOG_LEVEL") != "silent" {
			println(r.Method, r.URL.Path, time.Since(start).String())
		}
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ReadyResponse{
		Ready:     true,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	procMetrics := s.metrics.Collect()

	data := map[string]interface{}{
		"name":       "KNIRV Network Monitor",
		"uptime":     time.Since(s.startTime).Seconds(),
		"go_version": runtime.Version(),
		"goroutines": runtime.NumGoroutine(),
		"memory": map[string]interface{}{
			"heap_alloc_bytes":  mem.HeapAlloc,
			"heap_sys_bytes":    mem.HeapSys,
			"stack_inuse_bytes": mem.StackInuse,
		},
		"process": procMetrics,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(StatusResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// handleRootFailover provides the dashboard with a single server-proxied view
// of registry state and the current consensus round. The browser never talks
// to the registry directly, preserving deployment topology and future auth.
func (s *Server) handleRootFailover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	registryURL := strings.TrimRight(strings.TrimSpace(s.config.RegistryURL), "/")
	if registryURL == "" {
		http.Error(w, "registry URL is not configured", http.StatusServiceUnavailable)
		return
	}
	client := &http.Client{Timeout: s.config.RequestTimeout}
	if client.Timeout <= 0 {
		client.Timeout = 5 * time.Second
	}
	stateURL := registryURL
	if !strings.Contains(stateURL, "/state/") {
		stateURL += "/state/knirvoracle-1"
	}
	resp, err := client.Get(stateURL)
	if err != nil {
		http.Error(w, "registry state unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "registry state unavailable", http.StatusBadGateway)
		return
	}
	var state struct {
		ChainID        string `json:"chainID"`
		Phase          string `json:"phase"`
		CurrentRootID  string `json:"currentRootID"`
		Since          int64  `json:"since"`
		Round          uint64 `json:"round"`
		PreviousRootID string `json:"previousRootID"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&state); err != nil {
		http.Error(w, "invalid registry state", http.StatusBadGateway)
		return
	}
	consensusURL := strings.TrimSuffix(stateURL, "/state/knirvoracle-1") + "/consensus/knirvoracle-1/" + fmt.Sprint(state.Round)
	var consensus any
	if consensusResp, err := client.Get(consensusURL); err == nil {
		defer consensusResp.Body.Close()
		if consensusResp.StatusCode == http.StatusOK {
			_ = json.NewDecoder(io.LimitReader(consensusResp.Body, 1<<20)).Decode(&consensus)
		}
	}
	rootStatus := 0
	if state.CurrentRootID != "" && (state.Phase == "NORMAL" || state.Phase == "CONFIRMED" || state.Phase == "RECLAIMED") {
		rootStatus = 1
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"chainID": state.ChainID, "phase": state.Phase, "currentRootID": state.CurrentRootID, "previousRootID": state.PreviousRootID, "since": state.Since, "round": state.Round, "rootStatus": rootStatus, "consensus": consensus})
}

// handleOnboardingApplications and handleOnboardingUsers proxy to
// KNIRVGATEWAY's own native /api/admin/* endpoints (internal/rootkey +
// internal/d1 in KNIRVGATEWAY query Cloudflare D1 directly) — GatewayURL is
// the correct target; the paths below just need to match what KNIRVGATEWAY
// actually registers.
func (s *Server) handleOnboardingApplications(w http.ResponseWriter, r *http.Request) {
	s.proxyToOnboarding(w, r, "/api/admin/operators")
}

func (s *Server) handleOnboardingUsers(w http.ResponseWriter, r *http.Request) {
	s.proxyToOnboarding(w, r, "/api/admin/users")
}

// handleOnboardingApplicationAction forwards e.g.
// "/api/v1/onboarding/applications/{id}/review" to
// "/api/admin/operators/{id}/review" on KNIRVGATEWAY, preserving whatever
// suffix follows the fixed prefix.
func (s *Server) handleOnboardingApplicationAction(w http.ResponseWriter, r *http.Request) {
	suffix := strings.TrimPrefix(r.URL.Path, "/api/v1/onboarding/applications/")
	s.proxyToOnboarding(w, r, "/api/admin/operators/"+suffix)
}

// handleOnboardingUserAction forwards e.g.
// "/api/v1/onboarding/users/{id}/role" or ".../status" to
// "/api/admin/users/{id}/role" or ".../status" on KNIRVGATEWAY.
func (s *Server) handleOnboardingUserAction(w http.ResponseWriter, r *http.Request) {
	suffix := strings.TrimPrefix(r.URL.Path, "/api/v1/onboarding/users/")
	s.proxyToOnboarding(w, r, "/api/admin/users/"+suffix)
}

func (s *Server) proxyToOnboarding(w http.ResponseWriter, r *http.Request, path string) {
	onboardingURL := s.config.GatewayURL
	if onboardingURL == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(MetricsResponse{
			Success:   false,
			Data:      map[string]interface{}{"error": "onboarding proxy not configured"},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	proxyURL := onboardingURL + path
	if r.URL.RawQuery != "" {
		proxyURL += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, proxyURL, r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(MetricsResponse{
			Success:   false,
			Data:      map[string]interface{}{"error": err.Error()},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Set("Host", "")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(MetricsResponse{
			Success:   false,
			Data:      map[string]interface{}{"error": err.Error()},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// fetchGatewayRoutes calls KNIRVGATEWAY's native /api/network-monitor/routes
// handler directly (JSON, not Prometheus exposition format) and reshapes its
// statically-registered route table into GatewayRoute entries. Status is
// reported as "configured" rather than "up" because this list is reified
// from KNIRVGATEWAY's mux registration code, not per-route health-probed —
// don't claim liveness data this endpoint doesn't actually have.
func (s *Server) fetchGatewayRoutes() ([]GatewayRoute, error) {
	if strings.TrimSpace(s.config.GatewayURL) == "" {
		return nil, fmt.Errorf("gateway URL not configured")
	}

	reqURL := strings.TrimRight(s.config.GatewayURL, "/") + "/api/network-monitor/routes"
	client := &http.Client{Timeout: s.requestTimeout()}
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build gateway routes request: %w", err)
	}
	// The gateway performs its own admin authorization.  This in-process
	// monitor uses a short-lived service JWT, signed with the same secret the
	// gateway validates, instead of leaving the route inventory public.
	secret := strings.TrimSpace(os.Getenv("KNIRV_JWT_SECRET"))
	if secret == "" {
		return nil, fmt.Errorf("KNIRV_JWT_SECRET is required to fetch gateway routes")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"role": "admin", "sub": "knirvmonitor", "exp": time.Now().Add(time.Minute).Unix(),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return nil, fmt.Errorf("sign gateway monitor token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+signed)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", reqURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gateway routes endpoint returned status %d", resp.StatusCode)
	}

	var envelope struct {
		Success bool `json:"success"`
		Data    []struct {
			Name       string `json:"name"`
			PathPrefix string `json:"pathPrefix"`
			Target     string `json:"target"`
			Protocol   string `json:"protocol"`
			Status     string `json:"status"`
			LatencyMs  int64  `json:"latencyMs"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode gateway routes response: %w", err)
	}

	routes := make([]GatewayRoute, 0, len(envelope.Data))
	for _, r := range envelope.Data {
		routes = append(routes, GatewayRoute{
			Name:       r.Name,
			PathPrefix: r.PathPrefix,
			Target:     r.Target,
			Protocol:   r.Protocol,
			Status:     r.Status,
			LatencyMs:  r.LatencyMs,
		})
	}
	return routes, nil
}

func (s *Server) requestTimeout() time.Duration {
	if s.config.RequestTimeout > 0 {
		return s.config.RequestTimeout
	}
	return 5 * time.Second
}

func (s *Server) handleGatewayRoutes(w http.ResponseWriter, r *http.Request) {
	routes, err := s.fetchGatewayRoutes()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(MetricsResponse{
			Success:   false,
			Data:      map[string]interface{}{"error": err.Error()},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	json.NewEncoder(w).Encode(MetricsResponse{
		Success:   true,
		Data:      map[string]interface{}{"routes": routes, "scraped_at": time.Now().UTC().Format(time.RFC3339)},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleGatewayHealth(w http.ResponseWriter, r *http.Request) {
	status := "unreachable"
	if strings.TrimSpace(s.config.GatewayURL) != "" {
		client := &http.Client{Timeout: 3 * time.Second}
		if resp, err := client.Get(strings.TrimRight(s.config.GatewayURL, "/") + "/health"); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				status = "ok"
			}
		}
	}

	health := GatewayHealth{
		URL:       s.config.GatewayURL,
		Status:    status,
		LastCheck: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MetricsResponse{
		Success:   true,
		Data:      map[string]interface{}{"health": health},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleKNIRVBaseMetrics(w http.ResponseWriter, r *http.Request) {
	result, ok := s.probes.GetResult("knirvbase")
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(MetricsResponse{
			Success:   false,
			Data:      map[string]interface{}{"error": "probe not registered"},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	metrics := make([]KnirvbaseMetric, 0, len(result.Metrics))
	for _, m := range result.Metrics {
		metrics = append(metrics, KnirvbaseMetric{
			Name:   m.Name,
			Help:   "",
			Type:   "",
			Value:  m.Value,
			Labels: m.Labels,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MetricsResponse{
		Success:   true,
		Data:      map[string]interface{}{"metrics": metrics, "scraped_at": result.Scraped},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleKNIRVBaseHealth(w http.ResponseWriter, r *http.Request) {
	result, ok := s.probes.GetResult("knirvbase")
	status := "ok"
	if !ok {
		status = "unreachable"
	}

	health := KnirvbaseHealth{
		URL:       s.config.KNIRVBaseURL,
		Status:    status,
		LastCheck: time.Now(),
	}

	if ok && result != nil {
		if m, ok2 := result.Metrics["knirvbase_blocks_committed_total"]; ok2 {
			health.BlocksCommitted = int64(m.Value)
		}
		if hits, ok2 := result.Metrics["knirvbase_cache_hits_total"]; ok2 {
			if misses, ok3 := result.Metrics["knirvbase_cache_misses_total"]; ok3 {
				total := hits.Value + misses.Value
				if total > 0 {
					health.CacheHitRatio = hits.Value / total
				}
			}
		}
		if m, ok2 := result.Metrics["knirvbase_active_connections"]; ok2 {
			health.ActiveConnections = int64(m.Value)
		}
		if m, ok2 := result.Metrics["knirvbase_errors_total"]; ok2 {
			health.ErrorRate = m.Value
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MetricsResponse{
		Success:   true,
		Data:      map[string]interface{}{"health": health},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleKNIRVChainMetrics(w http.ResponseWriter, r *http.Request) {
	result, ok := s.probes.GetResult("knirvchain")
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(MetricsResponse{
			Success:   false,
			Data:      map[string]interface{}{"error": "probe not registered"},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	metrics := make([]KnirvchainMetric, 0, len(result.Metrics))
	for _, m := range result.Metrics {
		metrics = append(metrics, KnirvchainMetric{
			Name:   m.Name,
			Help:   "",
			Type:   "",
			Value:  m.Value,
			Labels: m.Labels,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MetricsResponse{
		Success:   true,
		Data:      map[string]interface{}{"metrics": metrics, "scraped_at": result.Scraped},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleKNIRVChainHealth(w http.ResponseWriter, r *http.Request) {
	result, ok := s.probes.GetResult("knirvchain")
	status := "ok"
	if !ok {
		status = "unreachable"
	}

	health := KnirvchainHealth{
		URL:       s.config.KNIRVChainURL,
		Status:    status,
		LastCheck: time.Now(),
	}

	if ok && result != nil {
		if m, ok2 := result.Metrics["knirvchain_blocks_committed_total"]; ok2 {
			_ = m
		}
		if m, ok2 := result.Metrics["knirvchain_active_connections"]; ok2 {
			_ = m
		}
		if m, ok2 := result.Metrics["knirvchain_errors_total"]; ok2 {
			_ = m
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MetricsResponse{
		Success:   true,
		Data:      map[string]interface{}{"health": health},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleNetworkInterfaces(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MetricsResponse{
		Success:   true,
		Data:      map[string]interface{}{"interfaces": []interface{}{}, "message": "NetworkMetrics implementation pending — interfaces are type-only in KNIRVCHAIN"},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// handleNetworkStats forwards KNIRVBASE's authoritative network snapshot.
// It intentionally does not manufacture zero-valued counters: an unavailable
// KNIRVBASE must be visible to operators as an unavailable dependency.
func (s *Server) handleNetworkStats(w http.ResponseWriter, r *http.Request) {
	baseURL := strings.TrimRight(strings.TrimSpace(s.config.KNIRVBaseURL), "/")
	if baseURL == "" {
		writeMonitorError(w, http.StatusServiceUnavailable, "KNIRVBASE network stats endpoint is not configured")
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, baseURL+"/network/stats", nil)
	if err != nil {
		writeMonitorError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp, err := (&http.Client{Timeout: s.requestTimeout()}).Do(req)
	if err != nil {
		writeMonitorError(w, http.StatusBadGateway, fmt.Sprintf("fetch KNIRVBASE network stats: %v", err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		writeMonitorError(w, http.StatusBadGateway, fmt.Sprintf("KNIRVBASE network stats returned status %d", resp.StatusCode))
		return
	}
	var stats map[string]interface{}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&stats); err != nil {
		writeMonitorError(w, http.StatusBadGateway, fmt.Sprintf("decode KNIRVBASE network stats: %v", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(MetricsResponse{Success: true, Data: map[string]interface{}{"stats": stats}, Timestamp: time.Now().UTC().Format(time.RFC3339)})
}

func (s *Server) handleKNIRVGraphScalability(w http.ResponseWriter, r *http.Request) {
	result, ok := s.probes.GetResult("knirvgraph")
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(MetricsResponse{
			Success:   false,
			Data:      map[string]interface{}{"error": "probe not registered"},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	metrics := make([]KnirvgraphMetric, 0, len(result.Metrics))
	for _, m := range result.Metrics {
		metrics = append(metrics, KnirvgraphMetric{
			Name:   m.Name,
			Help:   "",
			Type:   "",
			Value:  m.Value,
			Labels: m.Labels,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MetricsResponse{
		Success:   true,
		Data:      map[string]interface{}{"metrics": metrics, "scraped_at": result.Scraped},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleKNIRVGraphEmbeddings(w http.ResponseWriter, r *http.Request) {
	result, ok := s.probes.GetResult("knirvgraph")
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(MetricsResponse{
			Success:   false,
			Data:      map[string]interface{}{"error": "probe not registered"},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	metrics := make([]KnirvgraphMetric, 0)
	for name, m := range result.Metrics {
		if len(name) >= 18 && name[:18] == "knirvgraph_rag_" {
			metrics = append(metrics, KnirvgraphMetric{
				Name:   m.Name,
				Help:   "",
				Type:   "",
				Value:  m.Value,
				Labels: m.Labels,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MetricsResponse{
		Success:   true,
		Data:      map[string]interface{}{"metrics": metrics, "scraped_at": result.Scraped},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleKNIRVGraphHealth(w http.ResponseWriter, r *http.Request) {
	_, ok := s.probes.GetResult("knirvgraph")
	status := "ok"
	if !ok {
		status = "unreachable"
	}

	health := KnirvgraphHealth{
		URL:       s.config.KNIRVGraphURL,
		Status:    status,
		LastCheck: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MetricsResponse{
		Success:   true,
		Data:      map[string]interface{}{"health": health},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleKNIRVOracleEconomics(w http.ResponseWriter, r *http.Request) {
	result, ok := s.probes.GetResult("knirvoracle")
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(MetricsResponse{
			Success:   false,
			Data:      map[string]interface{}{"error": "probe not registered"},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MetricsResponse{
		Success:   true,
		Data:      map[string]interface{}{"metrics": result.Metrics, "scraped_at": result.Scraped},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleKNIRVOracleHealth(w http.ResponseWriter, r *http.Request) {
	_, ok := s.probes.GetResult("knirvoracle")
	status := "ok"
	if !ok {
		status = "unreachable"
	}

	health := KNIRVOracleHealth{
		URL:       s.config.KNIRVOracleURL,
		Status:    status,
		LastCheck: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MetricsResponse{
		Success:   true,
		Data:      map[string]interface{}{"health": health},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
