package api

import "time"

type StatusResponse struct {
	Success   bool                   `json:"success"`
	Data      map[string]interface{} `json:"data"`
	Timestamp string                 `json:"timestamp"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type ReadyResponse struct {
	Ready     bool   `json:"ready"`
	Timestamp string `json:"timestamp"`
}

type MetricsResponse struct {
	Success   bool                   `json:"success"`
	Data      map[string]interface{} `json:"data"`
	Timestamp string                 `json:"timestamp"`
}

type ServiceStatus struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	LastCheck   string `json:"lastCheck"`
	ResponseTime int64 `json:"responseTime"`
}

type ProcessMetrics struct {
	CPUUsage    float64 `json:"cpu_usage_percent"`
	MemoryUsage float64 `json:"memory_usage_percent"`
	MemoryBytes uint64  `json:"memory_bytes"`
	DiskTotal   uint64  `json:"disk_total_bytes"`
	DiskUsed    uint64  `json:"disk_used_bytes"`
	DiskAvail   uint64  `json:"disk_available_bytes"`
	Goroutines  int     `json:"goroutines"`
	Uptime      float64 `json:"uptime_seconds"`
}

type ServerConfig struct {
	Port            string
	PrometheusURL   string
	GrafanaURL      string
	ScrapeInterval  time.Duration
	RequestTimeout  time.Duration
	KNIRVBaseURL    string
	KNIRVChainURL   string
	KNIRVGraphURL   string
	KNIRVOracleURL  string
	GatewayURL      string
}

type KnirvbaseMetric struct {
	Name   string            `json:"name"`
	Help   string            `json:"help"`
	Type   string            `json:"type"`
	Value  float64           `json:"value"`
	Labels map[string]string `json:"labels,omitempty"`
}

type KnirvbaseHealth struct {
	URL               string    `json:"url"`
	Status            string    `json:"status"`
	BlocksCommitted   int64     `json:"blocks_committed"`
	ErrorRate         float64   `json:"error_rate"`
	CacheHitRatio     float64   `json:"cache_hit_ratio"`
	ActiveConnections int64     `json:"active_connections"`
	LastCheck         time.Time `json:"last_check"`
}

type KnirvchainMetric struct {
	Name   string            `json:"name"`
	Help   string            `json:"help"`
	Type   string            `json:"type"`
	Value  float64           `json:"value"`
	Labels map[string]string `json:"labels,omitempty"`
}

type KnirvchainHealth struct {
	URL        string    `json:"url"`
	Status     string    `json:"status"`
	LastCheck  time.Time `json:"last_check"`
}

type GatewayRoute struct {
	Name        string `json:"name"`
	PathPrefix  string `json:"pathPrefix"`
	Target      string `json:"target"`
	Protocol    string `json:"protocol"`
	Status      string `json:"status"`
	LatencyMs   int64  `json:"latencyMs"`
}

type OnboardingApplication struct {
	ID           string `json:"id"`
	LegalName    string `json:"legalName"`
	KYCStatus    string `json:"kycStatus"`
	Status       string `json:"status"`
	SubmittedAt  string `json:"submittedAt"`
}

type OnboardingUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Status   string `json:"status"`
	Plan     string `json:"plan"`
}

type KnirvgraphMetric struct {
	Name   string            `json:"name"`
	Help   string            `json:"help"`
	Type   string            `json:"type"`
	Value  float64           `json:"value"`
	Labels map[string]string `json:"labels,omitempty"`
}

type KnirvgraphHealth struct {
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	LastCheck time.Time `json:"last_check"`
}

type KNIRVOracleEconomics struct {
	TotalSupply    int64   `json:"total_supply"`
	TotalStaked    int64   `json:"total_staked"`
	TotalBurned    int64   `json:"total_burned"`
	APY            float64 `json:"apy"`
	FeeVolume      int64   `json:"fee_volume"`
	RewardsIssued  int64   `json:"rewards_issued"`
}

type KNIRVOracleHealth struct {
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	LastCheck time.Time `json:"last_check"`
}

type GatewayHealth struct {
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	LastCheck time.Time `json:"last_check"`
}
