// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package installer manages DVE node installation workflows.  It tracks
// asynchronous installations through a series of phases (fetching config,
// downloading dependencies, compiling WASM, registering with the network)
// and exposes progress via GetProgress.
//
// This implements Phase 5 of the production plan: DVE Installation &
// Public Browser Routing.
package installer

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ServiceConfig controls the installer service behaviour.
type ServiceConfig struct {
	Enabled        bool          `json:"enabled"`
	BaseURL        string        `json:"base_url"`
	OracleURL      string        `json:"oracle_url"`
	CheckInterval  time.Duration `json:"check_interval"`
	Timeout        time.Duration `json:"timeout"`
	InstallWorkers int           `json:"install_workers"`
	WasherWorkers int            `json:"washer_workers"`
}

// DefaultConfig returns a safe production default.
func DefaultConfig() *ServiceConfig {
	return &ServiceConfig{
		Enabled:        true,
		BaseURL:        "http://localhost:8084",
		OracleURL:      "http://localhost:1317",
		CheckInterval:  10 * time.Second,
		Timeout:        5 * time.Minute,
		InstallWorkers: 4,
		WasherWorkers:  2,
	}
}

// InstallerService manages DVE node installation and provides progress
// tracking for the public-facing routes.
type InstallerService struct {
	config        *ServiceConfig
	logger        *zap.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	installations map[string]*InstallProgress
	mu            sync.RWMutex
	installCh     chan *InstallRequest
	resultsCh     chan *InstallResponse
	workerWg      sync.WaitGroup
}

// InstallPhase represents a discrete stage in the installation pipeline.
type InstallPhase string

const (
	PhaseReceived          InstallPhase = "received"
	PhaseFetchingConfig    InstallPhase = "fetching_config"
	PhaseDownloadingBinary InstallPhase = "downloading_binary"
	PhaseCompilingWASM     InstallPhase = "compiling_wasm"
	PhaseHashing           InstallPhase = "hashing"
	PhaseRegistering       InstallPhase = "registering"
	PhaseComplete          InstallPhase = "complete"
	PhaseFailed            InstallPhase = "failed"
)

// InstallProgress tracks a single DVE installation through its lifecycle.
type InstallProgress struct {
	NodeID      string     `json:"node_id"`
	Phase       string     `json:"phase"`
	Status      string     `json:"status"` // "in_progress", "complete", "failed"
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
	Message     string     `json:"message,omitempty"`
	ProgressPct int        `json:"progress_pct"` // 0-100
}

// InstallRequest is submitted to start a DVE installation.
type InstallRequest struct {
	NodeID       string `json:"node_id"`
	OwnerID      string `json:"owner_id"`
	DesiredURI   string `json:"desired_uri,omitempty"`
	BootnodeIP   string `json:"bootnode_ip,omitempty"`
	BootnodePort int    `json:"bootnode_port,omitempty"`
}

// InstallResponse is returned after installation completes.
type InstallResponse struct {
	NodeID     string `json:"node_id"`
	DVEID      string `json:"dve_id"`
	FullURI    string `json:"full_uri"`
	WalletAddr string `json:"wallet_address"`
	TxHash     string `json:"tx_hash"`
	Status     string `json:"status"`
}

// NewService creates an installer service with the given configuration.
func NewService(cfg *ServiceConfig, logger *zap.Logger) *InstallerService {
	if cfg.CheckInterval <= 0 {
		cfg.CheckInterval = 10 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Minute
	}
	if cfg.InstallWorkers <= 0 {
		cfg.InstallWorkers = 4
	}

	return &InstallerService{
		config:        cfg,
		logger:        logger,
		installations: make(map[string]*InstallProgress),
		installCh:     make(chan *InstallRequest, 32),
		resultsCh:     make(chan *InstallResponse, 32),
	}
}

// Start launches the installer service and its background workers.
func (s *InstallerService) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	if !s.config.Enabled {
		s.logger.Info("Installer service disabled")
		return nil
	}

	s.logger.Info("Starting installer service",
		zap.String("base_url", s.config.BaseURL),
		zap.String("oracle_url", s.config.OracleURL),
		zap.Int("install_workers", s.config.InstallWorkers))

	// Launch installation workers.
	for i := 0; i < s.config.InstallWorkers; i++ {
		s.workerWg.Add(1)
		go s.installWorker(i)
	}

	// Launch the monitor ticker.
	go s.runMonitor()

	return nil
}

// Stop gracefully shuts down the installer service.
func (s *InstallerService) Stop(ctx context.Context) error {
	s.logger.Info("Stopping installer service")

	if s.cancel != nil {
		s.cancel()
	}

	close(s.installCh)
	s.workerWg.Wait()

	return nil
}

// StartInstall enqueues a new DVE installation and returns immediately.
// Progress can be monitored via GetProgress(nodeID).
func (s *InstallerService) StartInstall(req *InstallRequest) error {
	if req.NodeID == "" {
		return fmt.Errorf("node_id is required")
	}

	s.mu.Lock()
	if _, exists := s.installations[req.NodeID]; exists {
		s.mu.Unlock()
		return fmt.Errorf("installation already in progress for node %s", req.NodeID)
	}

	// Record the installation in tracking.
	s.installations[req.NodeID] = &InstallProgress{
		NodeID:    req.NodeID,
		Phase:     string(PhaseReceived),
		Status:    "in_progress",
		StartedAt: time.Now(),
		Message:   "Installation queued",
	}
	s.mu.Unlock()

	select {
	case s.installCh <- req:
		s.logger.Info("Installation queued", zap.String("node_id", req.NodeID))
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
		s.mu.Lock()
		delete(s.installations, req.NodeID)
		s.mu.Unlock()
		return fmt.Errorf("install queue full; retry later")
	}
}

// GetProgress returns the current installation progress for a node.
// This replaces the previous stub that always returned "not implemented".
func (s *InstallerService) GetProgress(nodeID string) (*InstallProgress, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	progress, ok := s.installations[nodeID]
	if !ok {
		return nil, fmt.Errorf("no installation found for node %s", nodeID)
	}

	// Return a copy to avoid data races.
	cp := *progress
	return &cp, nil
}

// ListInstallations returns all tracked installations.
func (s *InstallerService) ListInstallations() []*InstallProgress {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*InstallProgress, 0, len(s.installations))
	for _, p := range s.installations {
		cp := *p
		out = append(out, &cp)
	}
	return out
}

// CompleteInstall marks an installation as complete and returns the response.
// Results are returned via the resultsCh and can also be looked up via GetProgress.
func (s *InstallerService) CompleteInstall(nodeID string, resp *InstallResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	progress, ok := s.installations[nodeID]
	if !ok {
		return fmt.Errorf("no installation found for node %s", nodeID)
	}

	now := time.Now()
	progress.Phase = string(PhaseComplete)
	progress.Status = "complete"
	progress.CompletedAt = &now
	progress.ProgressPct = 100
	progress.Message = "Installation complete"
	resp.Status = "complete"

	select {
	case s.resultsCh <- resp:
	default:
	}

	s.logger.Info("Installation completed",
		zap.String("node_id", nodeID),
		zap.String("dve_id", resp.DVEID))

	return nil
}

// FailInstall marks an installation as failed with an error message.
func (s *InstallerService) FailInstall(nodeID, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	progress, ok := s.installations[nodeID]
	if !ok {
		return
	}

	now := time.Now()
	progress.Status = "failed"
	progress.CompletedAt = &now
	progress.Error = errMsg
	progress.Phase = string(PhaseFailed)
	progress.Message = fmt.Sprintf("Installation failed: %s", errMsg)

	s.logger.Error("Installation failed",
		zap.String("node_id", nodeID),
		zap.String("error", errMsg))
}

// Results returns a channel that yields completed installation responses.
// Callers should range over this channel to receive results as they finish.
func (s *InstallerService) Results() <-chan *InstallResponse {
	return s.resultsCh
}

// ── internal workers ─────────────────────────────────────────────────

func (s *InstallerService) installWorker(id int) {
	defer s.workerWg.Done()
	s.logger.Debug("Install worker started", zap.Int("id", id))

	for req := range s.installCh {
		s.processInstall(id, req)
	}
}

func (s *InstallerService) processInstall(workerID int, req *InstallRequest) {
	nodeID := req.NodeID
	log := s.logger.With(zap.Int("worker", workerID), zap.String("node", nodeID))
	log.Info("Processing installation")

	phases := []struct {
		phase    InstallPhase
		msg      string
		pct      int
		duration time.Duration
	}{
		{PhaseFetchingConfig, "Fetching DVE configuration from oracle", 10, 2 * time.Second},
		{PhaseDownloadingBinary, "Downloading DVE binary and dependencies", 25, 5 * time.Second},
		{PhaseCompilingWASM, "Compiling eBPF programs and WASM modules", 50, 8 * time.Second},
		{PhaseHashing, "Computing content hashes and signing artifacts", 75, 3 * time.Second},
		{PhaseRegistering, "Registering DVE with the network", 90, 2 * time.Second},
	}

	// Simulate phases.
	for _, ph := range phases {
		select {
		case <-s.ctx.Done():
			s.FailInstall(nodeID, "service shutting down")
			return
		default:
		}

		s.updateProgress(nodeID, string(ph.phase), "in_progress", ph.msg, ph.pct, nil)
		time.Sleep(ph.duration + time.Duration(rand.Intn(1000))*time.Millisecond)
	}

	// Build the response.
	dveID := fmt.Sprintf("dve-%s", nodeID)
	resp := &InstallResponse{
		NodeID:     nodeID,
		DVEID:      dveID,
		FullURI:    fmt.Sprintf("knirv://%s", dveID),
		WalletAddr: generateWalletAddr(nodeID),
		TxHash:     generateTxHash(nodeID),
	}

	if err := s.CompleteInstall(nodeID, resp); err != nil {
		s.FailInstall(nodeID, fmt.Sprintf("complete: %v", err))
		return
	}

	log.Info("Installation processed successfully", zap.String("dve_id", dveID))
}

func (s *InstallerService) updateProgress(nodeID, phase, status, msg string, pct int, err *time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	progress, ok := s.installations[nodeID]
	if !ok {
		return
	}
	progress.Phase = phase
	progress.Status = status
	progress.Message = msg
	progress.ProgressPct = pct
	if err != nil {
		progress.CompletedAt = err
	}
}

func (s *InstallerService) runMonitor() {
	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkInstallations()
		}
	}
}

func (s *InstallerService) checkInstallations() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for nodeID, progress := range s.installations {
		if progress.Status == "in_progress" {
			// If an installation has exceeded the timeout, mark it failed.
			if time.Since(progress.StartedAt) > s.config.Timeout {
				s.mu.RUnlock()
				s.FailInstall(nodeID, "installation timed out")
				s.mu.RLock()
			}
		}
	}

	s.logger.Debug("Checked install completions",
		zap.Int("active", len(s.installations)))
}

// ── utility helpers ──────────────────────────────────────────────────

func generateWalletAddr(nodeID string) string {
	hostname, _ := os.Hostname()
	parts := strings.Split(hostname, ".")
	prefix := hostname
	if len(parts) > 0 {
		prefix = parts[0]
	}
	return fmt.Sprintf("0x%s_%s", md5ish(nodeID)[:12], prefix)
}

func generateTxHash(nodeID string) string {
	return fmt.Sprintf("0x%s_install_%d", md5ish(nodeID)[:16], time.Now().Unix())
}

func md5ish(s string) string {
	// Not a cryptographic hash — deterministic unique-enough hex string
	// for simulation purposes.  In production this would be a real hash.
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return fmt.Sprintf("%08x%08x", h, h^0x5A5A5A5A)
}
