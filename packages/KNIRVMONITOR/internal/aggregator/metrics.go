package aggregator

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

type Registry struct {
	mu         sync.RWMutex
	Registry   *prometheus.Registry

	ProcessCPUSeconds    prometheus.Gauge
	ProcessMemoryBytes   prometheus.Gauge
	ProcessDiskTotal     prometheus.Gauge
	ProcessDiskUsed      prometheus.Gauge
	ProcessGoroutines    prometheus.Gauge
	ProcessUptimeSeconds prometheus.Gauge
	ScrapeErrors         prometheus.Counter

	remoteGauges   map[string]prometheus.Gauge
	remoteCounters map[string]prometheus.Counter
}

func NewRegistry() *Registry {
	reg := prometheus.NewRegistry()

	registry := &Registry{
		Registry:       reg,
		remoteGauges:   make(map[string]prometheus.Gauge),
		remoteCounters: make(map[string]prometheus.Counter),
	}

	registry.ProcessCPUSeconds = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "network_monitor_process_cpu_seconds_total",
		Help: "Total CPU time consumed by the network_monitor process in seconds",
	})

	registry.ProcessMemoryBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "network_monitor_process_memory_bytes",
		Help: "Current memory usage of the network_monitor process in bytes",
	})

	registry.ProcessDiskTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "network_monitor_process_disk_total_bytes",
		Help: "Total disk capacity available to the network_monitor process host in bytes",
	})

	registry.ProcessDiskUsed = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "network_monitor_process_disk_used_bytes",
		Help: "Disk space used on the network_monitor process host in bytes",
	})

	registry.ProcessGoroutines = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "network_monitor_process_goroutines",
		Help: "Number of goroutines currently running in the network_monitor process",
	})

	registry.ProcessUptimeSeconds = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "network_monitor_process_uptime_seconds",
		Help: "Uptime of the network_monitor process in seconds",
	})

	registry.ScrapeErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "network_monitor_scrape_errors_total",
		Help: "Total number of errors encountered during metrics collection",
	})

	reg.MustRegister(
		registry.ProcessCPUSeconds,
		registry.ProcessMemoryBytes,
		registry.ProcessDiskTotal,
		registry.ProcessDiskUsed,
		registry.ProcessGoroutines,
		registry.ProcessUptimeSeconds,
		registry.ScrapeErrors,
	)

	return registry
}

func (r *Registry) RegisterRemoteGauge(name, help string) prometheus.Gauge {
	r.mu.Lock()
	defer r.mu.Unlock()

	if g, ok := r.remoteGauges[name]; ok {
		return g
	}

	g := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	})
	r.Registry.MustRegister(g)
	r.remoteGauges[name] = g
	return g
}

func (r *Registry) RegisterRemoteCounter(name, help string) prometheus.Counter {
	r.mu.Lock()
	defer r.mu.Unlock()

	if c, ok := r.remoteCounters[name]; ok {
		return c
	}

	c := prometheus.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: help,
	})
	r.Registry.MustRegister(c)
	r.remoteCounters[name] = c
	return c
}

func (r *Registry) SetRemoteGauge(name string, value float64) {
	r.mu.RLock()
	g, ok := r.remoteGauges[name]
	r.mu.RUnlock()
	if ok {
		g.Set(value)
	}
}

func (r *Registry) AddRemoteCounter(name string, value float64) {
	r.mu.RLock()
	c, ok := r.remoteCounters[name]
	r.mu.RUnlock()
	if ok {
		c.Add(value)
	}
}

type ProcessMetrics struct {
	mu sync.Mutex

	StartTime    time.Time
	LastCPUStats *cpu.TimesStat
	LastCPUTime  time.Time
}

func NewProcessMetrics() *ProcessMetrics {
	return &ProcessMetrics{
		StartTime: time.Now(),
	}
}

func (p *ProcessMetrics) Collect() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make(map[string]interface{})

	vm, err := mem.VirtualMemory()
	if err == nil {
		result["memory_usage_percent"] = vm.UsedPercent
		result["memory_total_bytes"] = vm.Total
		result["memory_used_bytes"] = vm.Used
		result["memory_available_bytes"] = vm.Available
	}

	parts, err := disk.Partitions(true)
	if err == nil && len(parts) > 0 {
		usage, err := disk.Usage(parts[0].Mountpoint)
		if err == nil {
			result["disk_total_bytes"] = usage.Total
			result["disk_used_bytes"] = usage.Used
			result["disk_available_bytes"] = usage.Free
			result["disk_usage_percent"] = usage.UsedPercent
		}
	}

	cpuPercents, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercents) > 0 {
		result["cpu_usage_percent"] = cpuPercents[0]
	}

	result["uptime_seconds"] = time.Since(p.StartTime).Seconds()

	return result
}
