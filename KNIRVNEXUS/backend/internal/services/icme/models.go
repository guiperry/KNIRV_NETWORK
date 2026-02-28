package icme

import "time"

type SignalSource string

const (
	SourceValidation SignalSource = "validation"
	SourceError      SignalSource = "error"
	SourceSolution   SignalSource = "solution"
	SourceAgent      SignalSource = "agent"
	SourceFactuality SignalSource = "factuality"
	SourceEBPF       SignalSource = "ebpf"
)

type IntentScope string

const (
	ScopeGlobal IntentScope = "global"
	ScopeDVE    IntentScope = "dve"
)

type IntentionalSignal struct {
	ID              string              `json:"id"`
	AgentID         string              `json:"agent_id"`
	DVEID           string              `json:"dve_id"`
	Source          SignalSource        `json:"source"`
	Content         string              `json:"content"`
	Timestamp       time.Time           `json:"timestamp"`
	Scope           IntentScope         `json:"scope"`
	ObjectiveName   string              `json:"objective_name"`
	AuthorizedActs  []string            `json:"authorized_acts"`
	TradeOffWeights map[string]float64  `json:"trade_off_weights"`
	HardBoundaries  []string            `json:"hard_boundaries"`
	AlignmentScore  float64             `json:"alignment_score"`
	Entities        []ExtractedEntity   `json:"entities"`
	Relations       []ExtractedRelation `json:"relations"`
	EmbeddingID     int64               `json:"embedding_id"`
}

type ExtractedEntity struct {
	ID    string  `json:"id"`
	Text  string  `json:"text"`
	Label string  `json:"label"`
	Score float32 `json:"score"`
	Start int     `json:"start"`
	End   int     `json:"end"`
}

type ExtractedRelation struct {
	FromEntityID string  `json:"from_entity_id"`
	ToEntityID   string  `json:"to_node_id"`
	RelationType string  `json:"relation_type"`
	Confidence   float32 `json:"confidence"`
}

type IntentObjective struct {
	Name              string             `json:"name"`
	Scope             IntentScope        `json:"scope"`
	DVEID             string             `json:"dve_id"`
	Description       string             `json:"description"`
	Signals           []string           `json:"signals"`
	DataSources       []string           `json:"data_sources"`
	AuthorizedActions []string           `json:"authorized_actions"`
	TradeOffs         map[string]float64 `json:"trade_offs"`
	HardBoundaries    []string           `json:"hard_boundaries"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
	Version           int                `json:"version"`
}

type AlignmentRecord struct {
	ID             string    `json:"id"`
	AgentID        string    `json:"agent_id"`
	DVEID          string    `json:"dve_id"`
	ObjectiveName  string    `json:"objective_name"`
	SignalID       string    `json:"signal_id"`
	Decision       string    `json:"decision"`
	Outcome        string    `json:"outcome"`
	AlignmentScore float64   `json:"alignment_score"`
	FidelityScore  float64   `json:"fidelity_score"`
	Timestamp      time.Time `json:"timestamp"`
}

type VectorMeta struct {
	VectorID  int64  `json:"vector_id"`
	SignalID  string `json:"signal_id"`
	AgentID   string `json:"agent_id"`
	DVEID     string `json:"dve_id"`
	Summary   string `json:"summary"`
	Objective string `json:"objective"`
}

type HybridResult struct {
	SignalID       string       `json:"signal_id"`
	AgentID        string       `json:"agent_id"`
	DVEID          string       `json:"dve_id"`
	Content        string       `json:"content"`
	ObjectiveName  string       `json:"objective_name"`
	VectorScore    float32      `json:"vector_score"`
	GraphHops      int          `json:"graph_hops"`
	CombinedScore  float64      `json:"combined_score"`
	Nodes          []*HyperNode `json:"related_nodes"`
	AlignmentScore float64      `json:"alignment_score"`
}

type DecisionContext struct {
	AgentID      string
	DVEID        string
	Action       string
	CustomerTier string
	Amount       float64
	Custom       map[string]interface{}
}

type DecisionResult struct {
	Approved bool
	Action   string
	Reason   string
}

type HyperNode struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Text       string                 `json:"text"`
	Attributes map[string]interface{} `json:"attributes"`
	FirstSeen  time.Time              `json:"first_seen"`
	LastSeen   time.Time              `json:"last_seen"`
	SignalIDs  []string               `json:"signal_ids"`
}

type HyperEdge struct {
	ID             string                 `json:"id"`
	FromNodeID     string                 `json:"from_node_id"`
	ToNodeID       string                 `json:"to_node_id"`
	RelationType   string                 `json:"relation_type"`
	Timestamp      time.Time              `json:"timestamp"`
	SignalID       string                 `json:"signal_id"`
	AgentID        string                 `json:"agent_id"`
	DVEID          string                 `json:"dve_id"`
	ObjectiveName  string                 `json:"objective_name"`
	AlignmentScore float64                `json:"alignment_score"`
	Metadata       map[string]interface{} `json:"metadata"`
}
