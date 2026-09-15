package drq

import (
	"fmt"
	"time"

	"KNIRVGRAPH/internal/protocol/proto"
)

// ErrorNode is the canonical error node, generated from
// shared-proto/graph/v1/graph.proto into internal/protocol/proto.
//
// DRQ previously declared its own ErrorNode right here — the third divergent
// shape for one concept, alongside nrv's copy and KNIRVCHAIN's types. The
// canonical definition carries everything DRQ needs: the Clustering group
// (cluster_id, embedding, priority, queue_position) and the Red Queen dynamics
// group (adaptation_round, generation_score, phenotype_drift).
type ErrorNode = proto.ErrorNode

// TrainingExample is used for LoRA training
type TrainingExample struct {
	Input  string
	Output string
	Weight float64
}

// Solution represents agent-submitted resolution
type Solution struct {
	SolutionID   string
	ErrorID      string
	ClusterID    string
	AgentID      string
	CodePackage  string // LoRA adapter code or other solution
	ProposalTime time.Time

	// Validation state
	Validated       bool
	ValidationScore float64 // DVE consensus score
	DVEAttestation  []byte  // Cryptographic proof

	// DRQ metrics
	FitnessValue    float64 // Performance in multi-agent sim
	BehaviorDesc    BehaviorDescriptor
	GenotypeDist    float64 // Distance from cluster centroid
	ContributedData TrainingExample
}

// BehaviorDescriptor characterizes a solution's behavior
type BehaviorDescriptor struct {
	SpawnedProcesses int // Thread spawning behavior
	MemoryCoverage   int // Spatial footprint
	ExecutionTime    time.Duration
	ResourceUsage    ResourceProfile
}

// ResourceProfile tracks resource usage
type ResourceProfile struct {
	CPUUsage    float64
	MemoryUsage uint64
	DiskIO      uint64
	NetworkIO   uint64
}

// ErrorCluster with competitive dynamics
type ErrorCluster struct {
	ClusterID     string
	Errors        []*ErrorNode
	Centroid      []float64
	AgentCounts   map[string]int // Agent -> solution count
	Solutions     map[string][]*Solution
	Status        ClusterStatus
	CreatedAt     time.Time
	AvgComplexity float64
	// DRQ state tracking
	RoundCreated    int     // Which DRQ round
	CurrentFitness  float64 // Cluster performance
	HistoricalBest  float64 // Best fitness achieved
	GeneralityScore float64 // Generality against held-out set

	// Training state
	LoRAInProgress  bool
	TrainingJobID   string
	ValidationProof []byte

	// Validation state. ValidationTaskID is the platform DVE validation
	// task created for this cluster by ClusterManager.submitForValidation;
	// ValidationProof holds the platform's validation certificate for it.
	ValidationTaskID string
	// ValidationDegraded records that the validation service answered from a
	// fallback rather than a real model. A degraded result is never treated as
	// validation.
	ValidationDegraded bool

	// Minting result. Set by ClusterManager.mintSkill once
	// SkillMintingProtocol.MintSkillFromCluster succeeds.
	MintedSkillID string
	MintedSkill   *SkillNode

	// RewardsDistributed guards the resolved-state payout so a cluster that
	// stays CLUSTER_RESOLVED across ticks is only paid once.
	RewardsDistributed bool

	// Ownership and rewards
	OwnerAgent  string // Most solutions
	SecondPlace string // Runner-up
	TotalBounty uint64 // Accumulated NRN

	// Convergence metrics
	PhenotypeVar float64 // Variance across independent runs
	GenotypeVar  float64 // Source code variance

	// Network topology
	HubNodeID       string  // Hosting hub node
	TopologicalRank float64 // PageRank score
}

// AgentType defines the type of agent
type AgentType int

const (
	AGENT_KNIRV_SHELL AgentType = iota
	AGENT_HUMAN_DEVELOPER
	AGENT_AUTONOMOUS_CONTROLLER
)

// AgentProfile for competitive clustering
type AgentProfile struct {
	AgentID        string
	Type           AgentType
	Specialization []string

	// Performance metrics
	ReputationScore float64
	TotalSolutions  int
	ValidationRate  float64

	// Current assignment
	CurrentCluster string
	SolutionCount  int // In current cluster
	ClusterRank    int // Position in cluster

	// Cluster ownership
	OwnedClusters    []string // Clusters where agent is owner
	SkillInvocations uint64   // Total invocations across skills

	// DRQ history
	RoundsActive int
	BestGenScore float64
	AvgFitness   float64

	// Red Queen adaptation
	StrategyVector []float64 // Behavioral embedding
	AdaptationRate float64   // Phenotype change velocity

	CreatedAt  time.Time
	LastActive time.Time
}

// ActionType for DRQ actions
type ActionType int

const (
	ASSIGN_NEW_AGENT ActionType = iota
	REASSIGN_AGENT
	MERGE_CLUSTERS
	SPLIT_CLUSTER
	ESCALATE_PRIORITY
)

// String returns the string representation of the ActionType.
func (a ActionType) String() string {
	switch a {
	case ASSIGN_NEW_AGENT:
		return "ASSIGN_NEW_AGENT"
	case REASSIGN_AGENT:
		return "REASSIGN_AGENT"
	case MERGE_CLUSTERS:
		return "MERGE_CLUSTERS"
	case SPLIT_CLUSTER:
		return "SPLIT_CLUSTER"
	case ESCALATE_PRIORITY:
		return "ESCALATE_PRIORITY"
	default:
		return fmt.Sprintf("ActionType(%d)", int(a))
	}
}

// DRQAction represents possible agent assignments
type DRQAction struct {
	Type          ActionType
	AgentID       string
	TargetCluster string
	Priority      float64
	ResourceQuota ResourceAllocation
}

// ResourceAllocation defines resources for an action
type ResourceAllocation struct {
	DVENodeCount  int
	ComputeBudget uint64
	TimeLimit     time.Duration
}

// ClusterStatus defines the lifecycle of an error cluster
type ClusterStatus int

const (
	CLUSTER_ACTIVE     ClusterStatus = iota
	CLUSTER_TRAINING                 // LoRA training in progress
	CLUSTER_VALIDATING               // DVE validation
	CLUSTER_RESOLVED                 // Skill minted
	CLUSTER_ARCHIVED
)

// ErrorClusterState represents the RL state for DRQ
type ErrorClusterState struct {
	ClusterID         string
	ErrorFingerprints []string       // Error hashes in cluster
	AgentAssignments  map[string]int // Agent -> solution count
	ClusterCentroid   []float64      // Embedding centroid (768-dim)
	ClusterDensity    float64        // Node count in cluster radius
	ComplexityScore   float64        // Average error complexity
	SolutionVelocity  float64        // Solutions/hour
	TopologicalRank   float64        // PageRank in error graph
}

// ActionOutcome defines the result of a DRQ action
type ActionOutcome struct {
	SolutionValidated     bool
	ValidationScore       float64
	SolutionsGenerated    int
	AgentHours            float64
	SkillMinted           bool
	DependencyCount       float64
	DownstreamResolutions float64
	WastedDVEHours        float64
}
