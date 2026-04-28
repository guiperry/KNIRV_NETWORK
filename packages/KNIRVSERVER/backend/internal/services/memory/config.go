package memory

import "time"

// MemoryConfig defines configuration for the UnifiedMemorySystem
type MemoryConfig struct {
	EnabledBackends []string      `json:"enabled_backends"` // "markdown", "graphrag", "ontology"
	PQCEncryption    bool          `json:"pqc_encryption"`
	ArrowStreaming   bool          `json:"arrow_streaming"`
	GraphRAGModel    string        `json:"graphrag_model"`
	SyncInterval     time.Duration `json:"sync_interval"`
	EnableAutoSync   bool          `json:"enable_auto_sync"`
	KNIRVGRAPHPath  string        `json:"knirvgraph_path"`
}
