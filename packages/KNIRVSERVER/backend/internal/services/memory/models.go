package memory

import "time"

// ErrorNode represents a documented failure state in the network
type ErrorNode struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Context     map[string]interface{} `json:"context"`
	Timestamp    time.Time              `json:"timestamp"`
	BadgeID     string                 `json:"badge_id,omitempty"` // active badge at failure time (Gap 4)
}

// SolutionNode represents executable logic to resolve an ErrorNode
type SolutionNode struct {
	ID        string                 `json:"id"`
	ErrorID   string                 `json:"error_id"`
	Language  string                 `json:"language"`
	Code      string                 `json:"code"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
}

// OntologyEntityType classifies knowledge-graph entities
type OntologyEntityType string

const (
	EntityTypeNode       OntologyEntityType = "dve_node"
	EntityTypeTask       OntologyEntityType = "validation_task"
	EntityTypeResult     OntologyEntityType = "validation_result"
	EntityTypeAdaptation OntologyEntityType = "adaptation_event"
	EntityTypePattern    OntologyEntityType = "failure_pattern"
	EntityTypePolicy     OntologyEntityType = "guardrail_policy"
	EntityTypeViolation  OntologyEntityType = "policy_violation"
)

// OntologyEntity is a typed node in the knowledge graph
type OntologyEntity struct {
	ID         string
	Type       OntologyEntityType
	Label      string
	Properties map[string]interface{}
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// OntologyRelation is a directed, typed edge in the knowledge graph
type OntologyRelation struct {
	SourceID   string
	TargetID   string
	RelType    string
	Properties map[string]interface{}
	CreatedAt  time.Time
}

// QueryRequest represents a cross-backend query
type QueryRequest struct {
	Query   string   `json:"query"`
	Mode    string   `json:"mode"`    // "graphrag", "ontology", "hybrid"
	Limit   int      `json:"limit"`
	Sources []string `json:"sources"` // which backends to query
}

// QueryResult represents the result of a cross-backend query
type QueryResult struct {
	ID        string            `json:"id"`
	Query     string            `json:"query"`
	Mode      string            `json:"mode"`
	Entities  []*OntologyEntity `json:"entities,omitempty"`
	Nodes     []GraphNode       `json:"nodes,omitempty"`
	Edges     []GraphEdge       `json:"edges,omitempty"`
	Chunks    []TextChunk      `json:"chunks,omitempty"`
	Summary   string            `json:"summary"`
	Score     float64           `json:"score"`
	Timestamp time.Time         `json:"timestamp"`
}

// GraphNode represents a node in the GraphRAG index
type GraphNode struct {
	ID    string                 `json:"id"`
	Type  string                 `json:"type"`
	Data  map[string]interface{} `json:"data"`
	Score float64                `json:"score"`
}

// GraphEdge represents an edge in the GraphRAG index
type GraphEdge struct {
	Source     string                 `json:"source"`
	Target     string                 `json:"target"`
	Type       string                 `json:"type"`
	Weight     float64                `json:"weight"`
	Attributes map[string]interface{} `json:"attributes"`
}

// TextChunk represents a text chunk from the knowledge base
type TextChunk struct {
	ID        string  `json:"id"`
	Content   string  `json:"content"`
	Relevance float64 `json:"relevance"`
	Source    string  `json:"source,omitempty"`
}

// KnowledgeBase represents a GraphRAG-powered knowledge base
type KnowledgeBase struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Version        string                 `json:"version"`
	Author         string                 `json:"author"`
	Type           string                 `json:"type"`
	Status         string                 `json:"status"`
	FilePath       string                 `json:"file_path"`
	FileSize       int64                  `json:"file_size"`
	FileHash       string                 `json:"file_hash"`
	Capabilities   []string               `json:"capabilities"`
	Configuration  map[string]interface{} `json:"configuration"`
	Metadata       map[string]interface{} `json:"metadata"`
	Tags           []string               `json:"tags"`
	EmbeddingModel string                 `json:"embedding_model"`
	GraphIndex     string                 `json:"graph_index"`
	UploadedAt     time.Time              `json:"uploaded_at"`
	DeployedAt     *time.Time             `json:"deployed_at,omitempty"`
	LastModified   time.Time              `json:"last_modified"`
	LastActivity   *time.Time             `json:"last_activity,omitempty"`
	UploadedBy     string                 `json:"uploaded_by"`
	DeployedBy     *string                `json:"deployed_by,omitempty"`
}

// IndexStatus represents the status of a GraphRAG index build
type IndexStatus struct {
	KBID         string    `json:"kb_id"`
	Status       string    `json:"status"`
	Progress     float64   `json:"progress"`
	NodesCount   int       `json:"nodes_count"`
	EdgesCount   int       `json:"edges_count"`
	ChunksCount  int       `json:"chunks_count"`
	LastUpdated  time.Time `json:"last_updated"`
	ErrorMessage string    `json:"error_message,omitempty"`
}
