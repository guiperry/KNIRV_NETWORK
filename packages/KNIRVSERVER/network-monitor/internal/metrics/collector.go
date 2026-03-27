package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/knirv/network-monitor/internal/config"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/sirupsen/logrus"
)

// Collector handles metrics collection and aggregation
type Collector struct {
	config     *config.Config
	logger     *logrus.Logger
	prometheus v1.API
	metrics    map[string]interface{}
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// MetricValue represents a single metric value
type MetricValue struct {
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SystemMetrics represents system-level metrics
type SystemMetrics struct {
	CPU        CPUMetrics        `json:"cpu"`
	Memory     MemoryMetrics     `json:"memory"`
	Disk       DiskMetrics       `json:"disk"`
	Network    NetworkMetrics    `json:"network"`
	Containers ContainerMetrics  `json:"containers"`
	IPFS       IPFSMetrics       `json:"ipfs"`
	Timestamp  time.Time         `json:"timestamp"`
}

// CPUMetrics represents CPU usage metrics
type CPUMetrics struct {
	UsagePercent float64            `json:"usage_percent"`
	LoadAverage  []float64          `json:"load_average"`
	CoreUsage    map[string]float64 `json:"core_usage"`
}

// MemoryMetrics represents memory usage metrics
type MemoryMetrics struct {
	TotalBytes     uint64  `json:"total_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
	SwapUsed       uint64  `json:"swap_used"`
	SwapTotal      uint64  `json:"swap_total"`
}

// DiskMetrics represents disk usage metrics
type DiskMetrics struct {
	TotalBytes     uint64  `json:"total_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
	IOReads        uint64  `json:"io_reads"`
	IOWrites       uint64  `json:"io_writes"`
}

// NetworkMetrics represents network usage metrics
type NetworkMetrics struct {
	BytesReceived    uint64 `json:"bytes_received"`
	BytesSent        uint64 `json:"bytes_sent"`
	PacketsReceived  uint64 `json:"packets_received"`
	PacketsSent      uint64 `json:"packets_sent"`
	ConnectionsTotal int    `json:"connections_total"`
}

// ContainerMetrics represents container-specific metrics
type ContainerMetrics struct {
	RunningContainers int                            `json:"running_containers"`
	TotalContainers   int                            `json:"total_containers"`
	ContainerDetails  map[string]ContainerDetail     `json:"container_details"`
}

// ContainerDetail represents individual container metrics
type ContainerDetail struct {
	Name         string  `json:"name"`
	Status       string  `json:"status"`
	CPUPercent   float64 `json:"cpu_percent"`
	MemoryUsage  uint64  `json:"memory_usage"`
	MemoryLimit  uint64  `json:"memory_limit"`
	NetworkRx    uint64  `json:"network_rx"`
	NetworkTx    uint64  `json:"network_tx"`
}

// IPFSMetrics represents IPFS-specific metrics
type IPFSMetrics struct {
	RepoSize        uint64 `json:"repo_size"`
	RepoSizeMax     uint64 `json:"repo_size_max"`
	StoragePercent  float64 `json:"storage_percent"`
	PeerCount       int    `json:"peer_count"`
	PinnedObjects   int    `json:"pinned_objects"`
	BlocksStored    int    `json:"blocks_stored"`
	GatewayRequests uint64 `json:"gateway_requests"`
	APIRequests     uint64 `json:"api_requests"`
}

// New creates a new metrics collector
func New(cfg *config.Config, logger *logrus.Logger) (*Collector, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	collector := &Collector{
		config:  cfg,
		logger:  logger,
		metrics: make(map[string]interface{}),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Initialize Prometheus client if configured
	if cfg.Monitoring.PrometheusConfig.URL != "" {
		client, err := api.NewClient(api.Config{
			Address: cfg.Monitoring.PrometheusConfig.URL,
		})
		if err != nil {
			logger.Warnf("Failed to initialize Prometheus client: %v", err)
		} else {
			collector.prometheus = v1.NewAPI(client)
		}
	}

	return collector, nil
}

// Start starts the metrics collection
func (c *Collector) Start(ctx context.Context) error {
	c.logger.Info("Starting metrics collection...")

	// Start periodic metrics collection
	go c.collectMetricsPeriodically(ctx)

	c.logger.Info("Metrics collection started")
	return nil
}

// Stop stops the metrics collection
func (c *Collector) Stop() {
	c.logger.Info("Stopping metrics collection...")
	c.cancel()
	c.logger.Info("Metrics collection stopped")
}

// GetCurrentMetrics returns the current metrics snapshot
func (c *Collector) GetCurrentMetrics() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Create a copy of the metrics
	result := make(map[string]interface{})
	for k, v := range c.metrics {
		result[k] = v
	}
	
	return result
}

// GetSystemMetrics returns current system metrics
func (c *Collector) GetSystemMetrics() (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		Timestamp: time.Now(),
	}

	// Collect CPU metrics
	if err := c.collectCPUMetrics(&metrics.CPU); err != nil {
		c.logger.Warnf("Failed to collect CPU metrics: %v", err)
	}

	// Collect memory metrics
	if err := c.collectMemoryMetrics(&metrics.Memory); err != nil {
		c.logger.Warnf("Failed to collect memory metrics: %v", err)
	}

	// Collect disk metrics
	if err := c.collectDiskMetrics(&metrics.Disk); err != nil {
		c.logger.Warnf("Failed to collect disk metrics: %v", err)
	}

	// Collect network metrics
	if err := c.collectNetworkMetrics(&metrics.Network); err != nil {
		c.logger.Warnf("Failed to collect network metrics: %v", err)
	}

	// Collect container metrics
	if err := c.collectContainerMetrics(&metrics.Containers); err != nil {
		c.logger.Warnf("Failed to collect container metrics: %v", err)
	}

	// Collect IPFS metrics
	if err := c.collectIPFSMetrics(&metrics.IPFS); err != nil {
		c.logger.Warnf("Failed to collect IPFS metrics: %v", err)
	}

	return metrics, nil
}

// collectMetricsPeriodically runs periodic metrics collection
func (c *Collector) collectMetricsPeriodically(ctx context.Context) {
	ticker := time.NewTicker(c.config.Monitoring.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.collectAndStoreMetrics()
		}
	}
}

// collectAndStoreMetrics collects and stores current metrics
func (c *Collector) collectAndStoreMetrics() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Collect system metrics
	systemMetrics, err := c.GetSystemMetrics()
	if err != nil {
		c.logger.Errorf("Failed to collect system metrics: %v", err)
		return
	}

	// Store metrics
	c.metrics["system"] = systemMetrics
	c.metrics["last_update"] = time.Now()

	c.logger.Debugf("Metrics collected and stored")
}

// collectCPUMetrics collects CPU usage metrics
func (c *Collector) collectCPUMetrics(cpu *CPUMetrics) error {
	if c.prometheus == nil {
		// Fallback to basic metrics collection
		cpu.UsagePercent = 45.0 // Placeholder
		cpu.LoadAverage = []float64{1.2, 1.1, 1.0}
		cpu.CoreUsage = map[string]float64{
			"cpu0": 40.0,
			"cpu1": 50.0,
		}
		return nil
	}

	// Query Prometheus for CPU metrics
	query := `100 - (avg by(instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`
	_, _, err := c.prometheus.Query(c.ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to query CPU metrics: %w", err)
	}

	// Parse result and populate CPU metrics
	// This is a simplified implementation
	cpu.UsagePercent = 45.0 // Would parse from result
	cpu.LoadAverage = []float64{1.2, 1.1, 1.0}
	cpu.CoreUsage = map[string]float64{
		"cpu0": 40.0,
		"cpu1": 50.0,
	}

	return nil
}

// collectMemoryMetrics collects memory usage metrics
func (c *Collector) collectMemoryMetrics(memory *MemoryMetrics) error {
	// Placeholder implementation
	memory.TotalBytes = 16 * 1024 * 1024 * 1024 // 16GB
	memory.UsedBytes = 10 * 1024 * 1024 * 1024  // 10GB
	memory.AvailableBytes = memory.TotalBytes - memory.UsedBytes
	memory.UsagePercent = float64(memory.UsedBytes) / float64(memory.TotalBytes) * 100
	memory.SwapUsed = 0
	memory.SwapTotal = 2 * 1024 * 1024 * 1024 // 2GB

	return nil
}

// collectDiskMetrics collects disk usage metrics
func (c *Collector) collectDiskMetrics(disk *DiskMetrics) error {
	// Placeholder implementation
	disk.TotalBytes = 500 * 1024 * 1024 * 1024 // 500GB
	disk.UsedBytes = 200 * 1024 * 1024 * 1024  // 200GB
	disk.AvailableBytes = disk.TotalBytes - disk.UsedBytes
	disk.UsagePercent = float64(disk.UsedBytes) / float64(disk.TotalBytes) * 100
	disk.IOReads = 1000
	disk.IOWrites = 500

	return nil
}

// collectNetworkMetrics collects network usage metrics
func (c *Collector) collectNetworkMetrics(network *NetworkMetrics) error {
	// Placeholder implementation
	network.BytesReceived = 1024 * 1024 * 100  // 100MB
	network.BytesSent = 1024 * 1024 * 50       // 50MB
	network.PacketsReceived = 10000
	network.PacketsSent = 5000
	network.ConnectionsTotal = 25

	return nil
}

// collectContainerMetrics collects container-specific metrics
func (c *Collector) collectContainerMetrics(containers *ContainerMetrics) error {
	// Placeholder implementation
	containers.RunningContainers = 8
	containers.TotalContainers = 10
	containers.ContainerDetails = map[string]ContainerDetail{
		"knirv-prometheus": {
			Name:        "knirv-prometheus",
			Status:      "running",
			CPUPercent:  15.5,
			MemoryUsage: 512 * 1024 * 1024, // 512MB
			MemoryLimit: 1024 * 1024 * 1024, // 1GB
			NetworkRx:   1024 * 100,         // 100KB
			NetworkTx:   1024 * 50,          // 50KB
		},
		"knirv-grafana": {
			Name:        "knirv-grafana",
			Status:      "running",
			CPUPercent:  8.2,
			MemoryUsage: 256 * 1024 * 1024, // 256MB
			MemoryLimit: 512 * 1024 * 1024, // 512MB
			NetworkRx:   1024 * 200,        // 200KB
			NetworkTx:   1024 * 150,        // 150KB
		},
	}

	return nil
}

// collectIPFSMetrics collects IPFS-specific metrics
func (c *Collector) collectIPFSMetrics(ipfs *IPFSMetrics) error {
	// Placeholder implementation - in real implementation, this would
	// query IPFS API endpoints for actual metrics
	ipfs.RepoSize = 15 * 1024 * 1024 * 1024      // 15GB
	ipfs.RepoSizeMax = 50 * 1024 * 1024 * 1024   // 50GB
	ipfs.StoragePercent = float64(ipfs.RepoSize) / float64(ipfs.RepoSizeMax) * 100
	ipfs.PeerCount = 42
	ipfs.PinnedObjects = 1250
	ipfs.BlocksStored = 50000
	ipfs.GatewayRequests = 1500
	ipfs.APIRequests = 3200

	return nil
}
