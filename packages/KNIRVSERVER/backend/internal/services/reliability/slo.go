package reliability

import (
	"sync"
	"time"
)

type SLOConfig struct {
	Name             string        `json:"name"`
	Target           float64       `json:"target"`
	Window           time.Duration `json:"window"`
	MetricName       string        `json:"metric_name"`
	ComparisonOp     string        `json:"comparison_op"`
}

type SLOStatus struct {
	Config     SLOConfig `json:"config"`
	Current    float64   `json:"current"`
	Compliant  bool      `json:"compliant"`
	BurnRate   float64   `json:"burn_rate"`
}

type SLOBudget struct {
	mu       sync.RWMutex
	slos     map[string]*SLOConfig
	metrics  map[string][]float64
	timestamps map[string][]time.Time
}

func NewSLOBudget() *SLOBudget {
	return &SLOBudget{
		slos:       make(map[string]*SLOConfig),
		metrics:    make(map[string][]float64),
		timestamps: make(map[string][]time.Time),
	}
}

func (sb *SLOBudget) DefineSLO(config SLOConfig) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.slos[config.Name] = &config
}

func (sb *SLOBudget) RecordMetric(name string, value float64) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	sb.metrics[name] = append(sb.metrics[name], value)
	sb.timestamps[name] = append(sb.timestamps[name], time.Now())
}

func (sb *SLOBudget) GetStatus(name string) *SLOStatus {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	config, ok := sb.slos[name]
	if !ok {
		return nil
	}

	vals := sb.metrics[name]
	if len(vals) == 0 {
		return &SLOStatus{Config: *config, Current: 0, Compliant: true}
	}

	var sum float64
	for _, v := range vals {
		sum += v
	}
	current := sum / float64(len(vals))

	compliant := false
	switch config.ComparisonOp {
	case "gte":
		compliant = current >= config.Target
	case "lte":
		compliant = current <= config.Target
	default:
		compliant = current >= config.Target
	}

	var burnRate float64
	if len(vals) > 1 {
		burnRate = 1.0 - (current / config.Target)
		if burnRate < 0 {
			burnRate = 0
		}
	}

	return &SLOStatus{
		Config:    *config,
		Current:   current,
		Compliant: compliant,
		BurnRate:  burnRate,
	}
}

func (sb *SLOBudget) ListSLOs() []*SLOConfig {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	result := make([]*SLOConfig, 0, len(sb.slos))
	for _, c := range sb.slos {
		result = append(result, c)
	}
	return result
}
