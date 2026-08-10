package metrics

import "github.com/prometheus/client_golang/prometheus"

type RAGMetrics struct {
	Requests         *prometheus.CounterVec
	Latency          *prometheus.HistogramVec
	IndexedDocuments prometheus.Counter
	Failures         *prometheus.CounterVec
	Vectors          prometheus.Gauge
}

func NewRAGMetrics(reg prometheus.Registerer) *RAGMetrics {
	m := &RAGMetrics{Requests: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "knirvgraph_rag_requests_total", Help: "RAG HTTP requests."}, []string{"operation", "status"}), Latency: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "knirvgraph_rag_request_duration_seconds", Help: "RAG HTTP request duration.", Buckets: prometheus.DefBuckets}, []string{"operation"}), IndexedDocuments: prometheus.NewCounter(prometheus.CounterOpts{Name: "knirvgraph_rag_documents_indexed_total", Help: "Documents accepted for indexing."}), Failures: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "knirvgraph_rag_failures_total", Help: "RAG operation failures."}, []string{"operation"}), Vectors: prometheus.NewGauge(prometheus.GaugeOpts{Name: "knirvgraph_rag_vectors", Help: "Vectors currently indexed."})}
	reg.MustRegister(m.Requests, m.Latency, m.IndexedDocuments, m.Failures, m.Vectors)
	return m
}
