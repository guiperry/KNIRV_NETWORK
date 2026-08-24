package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

func (s *Server) handleActuarialMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := s.RefreshActuarialMetrics(r.Context())
	if err != nil {
		writeMonitorError(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(MetricsResponse{Success: true, Data: map[string]interface{}{"metrics": metrics}, Timestamp: time.Now().UTC().Format(time.RFC3339)})
}

// RefreshActuarialMetrics is invoked by the monitor scrape loop and by the
// read endpoint. It reaches backend_server only through its Unix socket.
func (s *Server) RefreshActuarialMetrics(ctx context.Context) (*ActuarialMetrics, error) {
	if s.config.BackendSocketPath == "" {
		return nil, fmt.Errorf("backend Unix socket is not configured")
	}
	client := &http.Client{Timeout: s.config.RequestTimeout}
	if client.Timeout <= 0 {
		client.Timeout = 5 * time.Second
	}
	client.Transport = &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", s.config.BackendSocketPath)
	}}
	url := "http://knirvserver/api/v1/actuarial/metrics"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create actuarial metrics request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch actuarial metrics: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("actuarial metrics upstream returned %s", response.Status)
	}
	var upstream struct {
		Metrics ActuarialMetrics `json:"metrics"`
	}
	if err := json.NewDecoder(response.Body).Decode(&upstream); err != nil {
		return nil, fmt.Errorf("decode actuarial metrics response: %w", err)
	}
	s.registry.RegisterRemoteGauge("network_monitor_actuarial_enabled", "Whether the actuarial syndicate is enabled").Set(float64(upstream.Metrics.Enabled))
	s.registry.RegisterRemoteGauge("network_monitor_actuarial_paused", "Whether actuarial deposits and payout dispatch are paused").Set(float64(upstream.Metrics.Paused))
	s.registry.RegisterRemoteGauge("network_monitor_actuarial_liquid_balance", "Aggregate liquid balance across syndicate pools").Set(float64(upstream.Metrics.LiquidBalance))
	s.registry.RegisterRemoteGauge("network_monitor_actuarial_reserved_balance", "Aggregate reserved balance across syndicate pools").Set(float64(upstream.Metrics.ReservedBalance))
	s.registry.RegisterRemoteGauge("network_monitor_actuarial_settlements_pending", "Number of non-final actuarial settlements").Set(float64(upstream.Metrics.SettlementsPending))
	s.registry.RegisterRemoteGauge("network_monitor_actuarial_settlements_failed", "Number of failed actuarial settlements").Set(float64(upstream.Metrics.SettlementsFailed))
	return &upstream.Metrics, nil
}

func writeMonitorError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(MetricsResponse{Success: false, Data: map[string]interface{}{"error": message}, Timestamp: time.Now().UTC().Format(time.RFC3339)})
}
