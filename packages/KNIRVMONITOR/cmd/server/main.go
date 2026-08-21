package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/knirv/network-monitor/internal/api"
)

func main() {
	socketPath := flag.String("socket", "", "Unix socket path (required)")
	prometheusURL := flag.String("prometheus-url", "http://localhost:9090", "Prometheus URL")
	grafanaURL := flag.String("grafana-url", "http://localhost:3333", "Grafana URL")
	scrapeInterval := flag.Duration("scrape-interval", 15*time.Second, "Scrape interval for metrics collection")
	requestTimeout := flag.Duration("request-timeout", 5*time.Second, "HTTP request timeout for external probes")
	knirvbaseURL := flag.String("knirvbase-url", "", "KNIRVBASE metrics endpoint URL")
	knirvchainURL := flag.String("knirvchain-url", "", "KNIRVCHAIN metrics endpoint URL")
	knirvgraphURL := flag.String("knirvgraph-url", "", "KNIRVGRAPH metrics endpoint URL")
	knirvoracleURL := flag.String("knirvoracle-url", "", "KNIRVORACLE economics endpoint URL")
	gatewayURL := flag.String("gateway-url", "", "KNIRVGATEWAY routes endpoint URL")
	flag.Parse()

	cfg := &api.ServerConfig{
		SocketPath:        *socketPath,
		PrometheusURL:     *prometheusURL,
		GrafanaURL:        *grafanaURL,
		ScrapeInterval:    *scrapeInterval,
		RequestTimeout:    *requestTimeout,
		KNIRVBaseURL:      *knirvbaseURL,
		KNIRVChainURL:     *knirvchainURL,
		KNIRVGraphURL:     "",
		KNIRVOracleURL:    "",
		BackendSocketPath: os.Getenv("BACKEND_SOCKET_PATH"),
	}

	if *knirvbaseURL != "" {
		cfg.KNIRVBaseURL = *knirvbaseURL
	}
	if *knirvchainURL != "" {
		cfg.KNIRVChainURL = *knirvchainURL
	}
	if *knirvgraphURL != "" {
		cfg.KNIRVGraphURL = *knirvgraphURL
	}
	if *knirvoracleURL != "" {
		cfg.KNIRVOracleURL = *knirvoracleURL
	}
	if *gatewayURL != "" {
		cfg.GatewayURL = *gatewayURL
	}

	server := api.NewServer(cfg)

	go runProbeLoop(server, *scrapeInterval)

	fmt.Printf("network_monitor starting on unix://%s\n", *socketPath)
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "server failed: %v\n", err)
		os.Exit(1)
	}
}

func runProbeLoop(s *api.Server, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if s.HasBackendSocket() {
			if _, err := s.RefreshActuarialMetrics(context.Background()); err != nil {
				s.Registry().ScrapeErrors.Inc()
			}
		}
		results := s.Probes().ScrapeAll()
		for probeName, result := range results {
			for _, metric := range result.Metrics {
				promName := fmt.Sprintf("network_monitor_probe_%s_%s", probeName, sanitizeName(metric.Name))
				g := s.Registry().RegisterRemoteGauge(promName, metric.Name)
				g.Set(metric.Value)
			}
		}
	}
}

func sanitizeName(name string) string {
	result := make([]byte, 0, len(name))
	for _, c := range []byte(name) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			result = append(result, c)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}
