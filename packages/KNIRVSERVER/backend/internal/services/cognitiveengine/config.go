package cognitiveengine

import "time"

// EngineConfig holds all tunable parameters for the Cognitive Engine.
// Values are populated from the server config or set to defaults by
// DefaultEngineConfig(). The running engine reads from this struct only
// at startup, so a restart is required for changes to take effect unless
// the caller uses the live-update setters below.
type EngineConfig struct {
	// Background loop intervals
	LearningInterval        time.Duration `mapstructure:"learning_interval"`
	MetricsInterval         time.Duration `mapstructure:"metrics_interval"`
	PatternAnalysisInterval time.Duration `mapstructure:"pattern_analysis_interval"`

	// Worker pool for concurrent validation result processing
	WorkerPoolSize    int `mapstructure:"worker_pool_size"`
	TaskQueueCapacity int `mapstructure:"task_queue_capacity"`

	// Guardrail subsystem
	GuardrailCheckInterval   time.Duration `mapstructure:"guardrail_check_interval"`
	MaxViolationsBeforePanic int           `mapstructure:"max_violations_before_panic"`

	// eBPF telemetry polling
	EBPFTelemetryInterval time.Duration `mapstructure:"ebpf_telemetry_interval"`

	// Ontology / KNIRVGRAPH sync
	OntologyUpdateInterval time.Duration `mapstructure:"ontology_update_interval"`

	// Periodic adaptation gate
	AdaptationMinInterval time.Duration `mapstructure:"adaptation_min_interval"`
}

// DefaultEngineConfig returns production-ready defaults.
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		LearningInterval:         30 * time.Second,
		MetricsInterval:          60 * time.Second,
		PatternAnalysisInterval:  5 * time.Minute,
		WorkerPoolSize:           4,
		TaskQueueCapacity:        256,
		GuardrailCheckInterval:   10 * time.Second,
		MaxViolationsBeforePanic: 5,
		EBPFTelemetryInterval:    15 * time.Second,
		OntologyUpdateInterval:   2 * time.Minute,
		AdaptationMinInterval:    24 * time.Hour,
	}
}
