package monitoring

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricsInstance *Metrics
	metricsOnce     sync.Once
)

type Metrics struct {
	BlocksCommitted          prometheus.Counter
	BlockCommitDuration      prometheus.Histogram
	SkillMintOps             prometheus.Counter
	CapabilityMintOps        prometheus.Counter
	PropertyMakeOps          prometheus.Counter
	MiningDuration           prometheus.Histogram
	MiningRewards            prometheus.Counter
	ValidationOps            prometheus.Counter
	ValidationDuration       prometheus.Histogram
	ActiveConnections        prometheus.Gauge
	NRNBalance               prometheus.Gauge
	QueryLatency             prometheus.Histogram
	ErrorCount               prometheus.Counter
	NodeStoreSize            prometheus.Gauge
	SkillNodesTotal          prometheus.Gauge
	CapabilityNodesTotal     prometheus.Gauge
	PropertyNodesTotal       prometheus.Gauge
	TransactionPoolSize      prometheus.Gauge
	ProofVerifications       prometheus.Counter
	ProofVerificationLatency prometheus.Histogram
	NoveltyChecks            prometheus.Counter
	NoveltyCheckDuration     prometheus.Histogram
}

func NewMetrics() *Metrics {
	metricsOnce.Do(func() {
		metricsInstance = &Metrics{
			BlocksCommitted: promauto.NewCounter(prometheus.CounterOpts{
				Name: "knirvchain_blocks_committed_total",
				Help: "Total number of blocks committed to the chain",
			}),
			BlockCommitDuration: promauto.NewHistogram(prometheus.HistogramOpts{
				Name:    "knirvchain_block_commit_duration_seconds",
				Help:    "Time taken to commit a block",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
			}),
			SkillMintOps: promauto.NewCounter(prometheus.CounterOpts{
				Name: "knirvchain_skill_mint_ops_total",
				Help: "Total number of SkillNode mint operations",
			}),
			CapabilityMintOps: promauto.NewCounter(prometheus.CounterOpts{
				Name: "knirvchain_capability_mint_ops_total",
				Help: "Total number of CapabilityNode mint operations",
			}),
			PropertyMakeOps: promauto.NewCounter(prometheus.CounterOpts{
				Name: "knirvchain_property_make_ops_total",
				Help: "Total number of PropertyNode make operations",
			}),
			MiningDuration: promauto.NewHistogram(prometheus.HistogramOpts{
				Name:    "knirvchain_mining_duration_seconds",
				Help:    "Time taken for node transformation mining",
				Buckets: prometheus.ExponentialBuckets(0.01, 2, 10),
			}),
			MiningRewards: promauto.NewCounter(prometheus.CounterOpts{
				Name: "knirvchain_mining_rewards_total",
				Help: "Total NRN rewards distributed for mining",
			}),
			ValidationOps: promauto.NewCounter(prometheus.CounterOpts{
				Name: "knirvchain_validation_ops_total",
				Help: "Total number of validation operations",
			}),
			ValidationDuration: promauto.NewHistogram(prometheus.HistogramOpts{
				Name:    "knirvchain_validation_duration_seconds",
				Help:    "Time taken for validation operations",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
			}),
			ActiveConnections: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "knirvchain_active_connections",
				Help: "Number of active P2P connections",
			}),
			NRNBalance: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "knirvchain_nrn_balance",
				Help: "Current NRN token balance",
			}),
			QueryLatency: promauto.NewHistogram(prometheus.HistogramOpts{
				Name:    "knirvchain_query_latency_seconds",
				Help:    "Query latency distribution",
				Buckets: prometheus.ExponentialBuckets(0.01, 2, 10),
			}),
			ErrorCount: promauto.NewCounter(prometheus.CounterOpts{
				Name: "knirvchain_errors_total",
				Help: "Total number of errors",
			}),
			NodeStoreSize: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "knirvchain_node_store_size_bytes",
				Help: "Size of the node store in bytes",
			}),
			SkillNodesTotal: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "knirvchain_skill_nodes_total",
				Help: "Total number of SkillNodes in the network",
			}),
			CapabilityNodesTotal: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "knirvchain_capability_nodes_total",
				Help: "Total number of CapabilityNodes in the network",
			}),
			PropertyNodesTotal: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "knirvchain_property_nodes_total",
				Help: "Total number of PropertyNodes in the network",
			}),
			TransactionPoolSize: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "knirvchain_transaction_pool_size",
				Help: "Number of pending transactions in the pool",
			}),
			ProofVerifications: promauto.NewCounter(prometheus.CounterOpts{
				Name: "knirvchain_proof_verifications_total",
				Help: "Total number of proof verifications",
			}),
			ProofVerificationLatency: promauto.NewHistogram(prometheus.HistogramOpts{
				Name:    "knirvchain_proof_verification_duration_seconds",
				Help:    "Time taken for proof verification",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
			}),
			NoveltyChecks: promauto.NewCounter(prometheus.CounterOpts{
				Name: "knirvchain_novelty_checks_total",
				Help: "Total number of novelty checks performed",
			}),
			NoveltyCheckDuration: promauto.NewHistogram(prometheus.HistogramOpts{
				Name:    "knirvchain_novelty_check_duration_seconds",
				Help:    "Time taken for novelty checks",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
			}),
		}
	})
	return metricsInstance
}
