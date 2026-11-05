// Package monitor embeds KNIRVMONITOR in KNIRVSERVER and owns its private
// Unix-socket API and background probe loop.
package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"knirv-server/internal/monitor/api"
)

// Config is the monitor configuration consumed by KNIRVSERVER.
type Config = api.ServerConfig

// Service is the monitor lifecycle owned by the main KNIRVSERVER process.
type Service struct {
	server *api.Server
	cancel context.CancelFunc
	oracle *http.Client
}

func Start(parent context.Context, cfg *Config) (*Service, error) {
	if cfg == nil {
		return nil, fmt.Errorf("monitor config is required")
	}
	if cfg.ScrapeInterval <= 0 {
		cfg.ScrapeInterval = 15 * time.Second
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 5 * time.Second
	}
	ctx, cancel := context.WithCancel(parent)
	server := api.NewServer(cfg)
	if err := server.Start(ctx); err != nil {
		cancel()
		return nil, err
	}
	service := &Service{server: server, cancel: cancel}
	if cfg.OracleSocketPath != "" {
		socket := cfg.OracleSocketPath
		service.oracle = &http.Client{Timeout: cfg.RequestTimeout, Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		}}}
	}
	go service.runProbeLoop(ctx, cfg.ScrapeInterval)
	return service, nil
}

func (s *Service) Shutdown(ctx context.Context) error {
	s.cancel()
	return s.server.Shutdown(ctx)
}

func (s *Service) runProbeLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if s.oracle != nil {
				s.refreshReplica(ctx)
			}
			if s.server.HasBackendSocket() {
				if _, err := s.server.RefreshActuarialMetrics(context.Background()); err != nil {
					s.server.Registry().ScrapeErrors.Inc()
				}
			}
			for probeName, result := range s.server.Probes().ScrapeAll() {
				for _, metric := range result.Metrics {
					gauge := s.server.Registry().RegisterRemoteGauge(
						fmt.Sprintf("network_monitor_probe_%s_%s", probeName, sanitizeName(metric.Name)), metric.Name)
					gauge.Set(metric.Value)
				}
			}
		}
	}
}

func (s *Service) refreshReplica(ctx context.Context) {
	available := s.server.Registry().RegisterRemoteGauge("oracle_replica_available", "Whether the latest Oracle replication scrape succeeded")
	available.Set(0)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix/oracle/v3/status", nil)
	if err != nil {
		s.server.Registry().ScrapeErrors.Inc()
		return
	}
	resp, err := s.oracle.Do(req)
	if err != nil {
		s.server.Registry().ScrapeErrors.Inc()
		return
	}
	defer resp.Body.Close()
	var status struct {
		Mode       string   `json:"mode"`
		HeadSeq    *uint64  `json:"head_seq"`
		LagSeq     *uint64  `json:"lag_seq"`
		LagSeconds *int64   `json:"lag_seconds"`
		Diverged   *bool    `json:"diverged"`
		Paused     *bool    `json:"replication_paused"`
		Unverified []string `json:"unverified_classes"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&status) != nil || (status.Mode != "active" && status.Mode != "standby" && status.Mode != "off") || status.HeadSeq == nil || status.LagSeq == nil || status.LagSeconds == nil || status.Diverged == nil || status.Paused == nil {
		s.server.Registry().ScrapeErrors.Inc()
		return
	}
	available.Set(1)
	values := map[string]float64{"oracle_replica_head_seq": float64(*status.HeadSeq), "oracle_replica_lag_seq": float64(*status.LagSeq), "oracle_replica_lag_seconds": float64(*status.LagSeconds), "oracle_replica_diverged": 0, "oracle_replica_paused": 0, "oracle_replica_unverified_classes": float64(len(status.Unverified)), "oracle_mode": 0}
	if status.Mode == "active" {
		values["oracle_mode"] = 1
	}
	if status.Mode == "standby" {
		values["oracle_mode"] = 2
	}
	if *status.Diverged {
		values["oracle_replica_diverged"] = 1
	}
	if *status.Paused {
		values["oracle_replica_paused"] = 1
	}
	for name, value := range values {
		s.server.Registry().RegisterRemoteGauge(name, "Local Oracle replication status").Set(value)
	}
}

func sanitizeName(name string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			return r
		}
		return '_'
	}, name)
}
