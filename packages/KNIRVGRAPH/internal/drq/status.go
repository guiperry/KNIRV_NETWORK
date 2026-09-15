package drq

// Status is the observable state of the knowledge-mining loop.
//
// It lives here rather than in the app package so both the app that builds it
// and the RPC layer that serves it can reference the same type without the
// network package importing app (which would invert the dependency).
type Status struct {
	Enabled             bool           `json:"enabled"`
	DisabledReason      string         `json:"disabled_reason,omitempty"`
	Ticks               uint64         `json:"ticks"`
	IngestedErrors      int            `json:"ingested_errors"`
	Clusters            int            `json:"clusters"`
	ClustersByStatus    map[string]int `json:"clusters_by_status"`
	ClustersDetail      []ClusterView  `json:"clusters_detail"`
	ValidatorConfigured bool           `json:"validator_configured"`
	TrainerBackend      string         `json:"trainer_backend"`
	SimilarityThreshold float64        `json:"similarity_threshold"`
	// LastErrors maps a cluster id to the lifecycle error currently blocking it.
	// Each distinct failure is reported once, so this reflects the *current*
	// blocker rather than a repeat count.
	LastErrors map[string]string `json:"last_errors,omitempty"`
}

// ClusterView is the per-cluster status detail.
type ClusterView struct {
	ClusterID      string   `json:"cluster_id"`
	Status         string   `json:"status"`
	Members        int      `json:"members"`
	Validated      int      `json:"validated_solutions"`
	MeanSimilarity float64  `json:"mean_similarity"`
	AgeSeconds     float64  `json:"age_seconds"`
	Ready          bool     `json:"ready_for_training"`
	NotReadyReason string   `json:"not_ready_reason,omitempty"`
	ValidationTask string   `json:"validation_task_id,omitempty"`
	MintedSkillID  string   `json:"minted_skill_id,omitempty"`
	ResolvedErrors []string `json:"resolved_errors,omitempty"`
}
