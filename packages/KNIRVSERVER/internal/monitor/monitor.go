// Package monitor embeds KNIRVMONITOR in KNIRVSERVER and owns its private
// Unix-socket API and background probe loop.
package monitor

import (
	"context"
	"fmt"
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

func sanitizeName(name string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			return r
		}
		return '_'
	}, name)
}
