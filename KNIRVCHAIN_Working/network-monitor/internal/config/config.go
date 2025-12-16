package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	ActiveNetwork string             `yaml:"active_network"`
	Networks      map[string]Network `yaml:"networks"`
	Monitoring    MonitoringConfig   `yaml:"monitoring"`
	Logging       LoggingConfig      `yaml:"logging"`
	Alerting      AlertingConfig     `yaml:"alerting"`
	GUI           GUIConfig          `yaml:"gui"`
}

// Network represents a KNIRV network configuration
type Network struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Services    []Service `yaml:"services"`
	Endpoints   Endpoints `yaml:"endpoints"`
}

// Service represents a monitored service
type Service struct {
	Name            string            `yaml:"name"`
	URL             string            `yaml:"url"`
	HealthEndpoint  string            `yaml:"health_endpoint"`
	MetricsEndpoint string            `yaml:"metrics_endpoint,omitempty"`
	LogPath         string            `yaml:"log_path,omitempty"`
	CheckInterval   time.Duration     `yaml:"check_interval"`
	Timeout         time.Duration     `yaml:"timeout"`
	Tags            map[string]string `yaml:"tags,omitempty"`
	Critical        bool              `yaml:"critical"`
}

// Endpoints represents network-specific endpoints
type Endpoints struct {
	Prometheus string `yaml:"prometheus"`
	Grafana    string `yaml:"grafana"`
	Kibana     string `yaml:"kibana"`
	AlertManager string `yaml:"alertmanager"`
}

// MonitoringConfig represents monitoring system configuration
type MonitoringConfig struct {
	Enabled           bool          `yaml:"enabled"`
	CheckInterval     time.Duration `yaml:"check_interval"`
	MetricsRetention  time.Duration `yaml:"metrics_retention"`
	PrometheusConfig  PrometheusConfig `yaml:"prometheus"`
	GrafanaConfig     GrafanaConfig    `yaml:"grafana"`
}

// PrometheusConfig represents Prometheus configuration
type PrometheusConfig struct {
	URL             string        `yaml:"url"`
	ScrapeInterval  time.Duration `yaml:"scrape_interval"`
	EvaluationInterval time.Duration `yaml:"evaluation_interval"`
	RetentionTime   string        `yaml:"retention_time"`
}

// GrafanaConfig represents Grafana configuration
type GrafanaConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	OrgID    int    `yaml:"org_id"`
}

// LoggingConfig represents logging system configuration
type LoggingConfig struct {
	Enabled       bool              `yaml:"enabled"`
	Level         string            `yaml:"level"`
	ElasticConfig ElasticConfig     `yaml:"elasticsearch"`
	LogPaths      map[string]string `yaml:"log_paths"`
	Retention     time.Duration     `yaml:"retention"`
}

// ElasticConfig represents Elasticsearch configuration
type ElasticConfig struct {
	URLs     []string `yaml:"urls"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	Index    string   `yaml:"index"`
}

// AlertingConfig represents alerting system configuration
type AlertingConfig struct {
	Enabled   bool                   `yaml:"enabled"`
	Rules     []AlertRule            `yaml:"rules"`
	Channels  map[string]AlertChannel `yaml:"channels"`
	Escalation EscalationConfig       `yaml:"escalation"`
}

// AlertRule represents an alerting rule
type AlertRule struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Condition   string            `yaml:"condition"`
	Severity    string            `yaml:"severity"`
	Duration    time.Duration     `yaml:"duration"`
	Channels    []string          `yaml:"channels"`
	Tags        map[string]string `yaml:"tags,omitempty"`
	Enabled     bool              `yaml:"enabled"`
}

// AlertChannel represents an alert notification channel
type AlertChannel struct {
	Type     string                 `yaml:"type"`
	Config   map[string]interface{} `yaml:"config"`
	Enabled  bool                   `yaml:"enabled"`
}

// EscalationConfig represents alert escalation configuration
type EscalationConfig struct {
	Enabled   bool          `yaml:"enabled"`
	Levels    []EscalationLevel `yaml:"levels"`
	MaxLevel  int           `yaml:"max_level"`
}

// EscalationLevel represents an escalation level
type EscalationLevel struct {
	Level    int           `yaml:"level"`
	Duration time.Duration `yaml:"duration"`
	Channels []string      `yaml:"channels"`
}

// GUIConfig represents GUI application configuration
type GUIConfig struct {
	Theme           string        `yaml:"theme"`
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	WindowSize      WindowSize    `yaml:"window_size"`
	Charts          ChartsConfig  `yaml:"charts"`
}

// WindowSize represents GUI window dimensions
type WindowSize struct {
	Width  int `yaml:"width"`
	Height int `yaml:"height"`
}

// ChartsConfig represents chart display configuration
type ChartsConfig struct {
	TimeRange     time.Duration `yaml:"time_range"`
	UpdateInterval time.Duration `yaml:"update_interval"`
	MaxDataPoints int           `yaml:"max_data_points"`
}

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	// Set default configuration
	cfg := &Config{
		ActiveNetwork: "production",
		Networks: map[string]Network{
			"production": {
				Name:        "KNIRV Production Network",
				Description: "Main KNIRV production network",
				Services: []Service{
					{
						Name:           "ipfs",
						URL:            "http://localhost:5001",
						HealthEndpoint: "/api/v0/version",
						CheckInterval:  30 * time.Second,
						Timeout:        10 * time.Second,
						Critical:       true,
					},
					{
						Name:           "KNIRVCHAIN",
						URL:            "http://localhost:8083",
						HealthEndpoint: "/health",
						CheckInterval:  30 * time.Second,
						Timeout:        10 * time.Second,
						Critical:       true,
					},
					{
						Name:           "knirvchain",
						URL:            "http://localhost:8080",
						HealthEndpoint: "/health",
						CheckInterval:  30 * time.Second,
						Timeout:        10 * time.Second,
						Critical:       true,
					},
					{
						Name:           "knirvgraph",
						URL:            "http://localhost:8081",
						HealthEndpoint: "/height",
						CheckInterval:  30 * time.Second,
						Timeout:        10 * time.Second,
						Critical:       true,
					},
					{
						Name:           "knirvnexus",
						URL:            "http://localhost:8082",
						HealthEndpoint: "/health",
						CheckInterval:  30 * time.Second,
						Timeout:        10 * time.Second,
						Critical:       true,
					},
					{
						Name:           "knirvrouter",
						URL:            "http://localhost:5001",
						HealthEndpoint: "/status",
						CheckInterval:  30 * time.Second,
						Timeout:        10 * time.Second,
						Critical:       true,
					},
				},
				Endpoints: Endpoints{
					Prometheus:   "http://localhost:9090",
					Grafana:      "http://localhost:3000",
					Kibana:       "http://localhost:5601",
					AlertManager: "http://localhost:9093",
				},
			},
			"testnet": {
				Name:        "KNIRV Testnet",
				Description: "KNIRV development and testing network",
				Services: []Service{
					{
						Name:           "ipfs",
						URL:            "http://localhost:5001",
						HealthEndpoint: "/api/v0/version",
						CheckInterval:  30 * time.Second,
						Timeout:        10 * time.Second,
						Critical:       true,
					},
					{
						Name:           "KNIRVCHAIN",
						URL:            "http://localhost:1317",
						HealthEndpoint: "/health",
						CheckInterval:  30 * time.Second,
						Timeout:        10 * time.Second,
						Critical:       true,
					},
					// Add other testnet services...
				},
				Endpoints: Endpoints{
					Prometheus:   "http://localhost:9091",
					Grafana:      "http://localhost:3001",
					Kibana:       "http://localhost:5602",
					AlertManager: "http://localhost:9094",
				},
			},
		},
		Monitoring: MonitoringConfig{
			Enabled:       true,
			CheckInterval: 30 * time.Second,
			MetricsRetention: 30 * 24 * time.Hour, // 30 days
		},
		Logging: LoggingConfig{
			Enabled: true,
			Level:   "info",
			Retention: 7 * 24 * time.Hour, // 7 days
		},
		Alerting: AlertingConfig{
			Enabled: true,
		},
		GUI: GUIConfig{
			Theme:           "dark",
			RefreshInterval: 5 * time.Second,
			WindowSize: WindowSize{
				Width:  1200,
				Height: 700,
			},
			Charts: ChartsConfig{
				TimeRange:     1 * time.Hour,
				UpdateInterval: 5 * time.Second,
				MaxDataPoints: 100,
			},
		},
	}

	// Load from file if it exists
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	return cfg, nil
}

// Save saves the configuration to a YAML file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetActiveNetwork returns the currently active network configuration
func (c *Config) GetActiveNetwork() (*Network, error) {
	network, exists := c.Networks[c.ActiveNetwork]
	if !exists {
		return nil, fmt.Errorf("network '%s' not found", c.ActiveNetwork)
	}
	return &network, nil
}
