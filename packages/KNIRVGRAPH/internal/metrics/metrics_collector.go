package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsCollector exports Prometheus metrics
type MetricsCollector struct {
	// Clustering metrics
	ClusteringLatency    prometheus.Histogram
	ClusterCount         prometheus.Gauge
	ErrorQueueSize       prometheus.Gauge
	
	// DRQ metrics
	QValueSyncLatency    prometheus.Histogram
	ConvergenceRate      prometheus.Gauge
	PhenotypeDrift       prometheus.Gauge
	RoundNumber          prometheus.Counter
	
	// Network metrics
	NodeCount            prometheus.Gauge
	HubNodeCount         prometheus.Gauge
	AvgPathLength        prometheus.Gauge
	
	// Training metrics
	LoRATrainingTime     prometheus.Histogram
	SkillsMinted         prometheus.Counter
	ValidationSuccess    prometheus.Counter
	
	// Consensus performance
	BlockTime            prometheus.Histogram
	TxThroughput         prometheus.Gauge
	ValidatorCount       prometheus.Gauge
}

// NewMetricsCollector initializes and registers Prometheus metrics
func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		ClusteringLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "knirvgraph_clustering_latency_seconds",
			Help: "Latency of error clustering operations.",
			Buckets: prometheus.DefBuckets,
		}),
		ClusterCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "knirvgraph_cluster_count",
			Help: "Current number of error clusters.",
		}),
		ErrorQueueSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "knirvgraph_error_queue_size",
			Help: "Current size of the error queue.",
		}),
		QValueSyncLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "knirvgraph_qvalue_sync_latency_seconds",
			Help: "Latency of DRQ Q-value synchronization.",
			Buckets: prometheus.DefBuckets,
		}),
		ConvergenceRate: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "knirvgraph_drq_convergence_rate",
			Help: "Convergence rate of DRQ Q-values.",
		}),
		PhenotypeDrift: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "knirvgraph_drq_phenotype_drift",
			Help: "Phenotype drift in DRQ evolution.",
		}),
		RoundNumber: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "knirvgraph_drq_round_number",
			Help: "Current DRQ round number.",
		}),
		NodeCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "knirvgraph_network_node_count",
			Help: "Current number of nodes in the network.",
		}),
		HubNodeCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "knirvgraph_network_hub_node_count",
			Help: "Current number of hub nodes in the network.",
		}),
		AvgPathLength: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "knirvgraph_network_avg_path_length",
			Help: "Average path length in the network.",
		}),
		LoRATrainingTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "knirvgraph_lora_training_time_seconds",
			Help: "Time taken for LoRA adapter training.",
			Buckets: prometheus.DefBuckets,
		}),
		SkillsMinted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "knirvgraph_skills_minted_total",
			Help: "Total number of skills minted.",
		}),
		ValidationSuccess: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "knirvgraph_validation_success_total",
			Help: "Total number of successful validations.",
		}),
		BlockTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "knirvgraph_consensus_block_time_seconds",
			Help: "Time taken for block production.",
			Buckets: prometheus.DefBuckets,
		}),
		TxThroughput: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "knirvgraph_consensus_tx_throughput",
			Help: "Transaction throughput in transactions per second.",
		}),
		ValidatorCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "knirvgraph_consensus_validator_count",
			Help: "Current number of validators.",
		}),
	}

	prometheus.MustRegister(
		mc.ClusteringLatency,
		mc.ClusterCount,
		mc.ErrorQueueSize,
		mc.QValueSyncLatency,
		mc.ConvergenceRate,
		mc.PhenotypeDrift,
		mc.RoundNumber,
		mc.NodeCount,
		mc.HubNodeCount,
		mc.AvgPathLength,
		mc.LoRATrainingTime,
		mc.SkillsMinted,
		mc.ValidationSuccess,
		mc.BlockTime,
		mc.TxThroughput,
		mc.ValidatorCount,
	)
	return mc
}

// ServeHTTP exports Prometheus endpoint
func (mc *MetricsCollector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}
