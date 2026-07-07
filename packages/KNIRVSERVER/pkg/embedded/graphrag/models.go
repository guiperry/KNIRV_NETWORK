package graphrag

import "time"

type GraphQuery struct {
	Query  string                 `json:"query"`
	Mode   string                 `json:"mode"`
	Limit  int                    `json:"limit"`
	Params map[string]interface{} `json:"params"`
}

type GraphResult struct {
	ID        string      `json:"id"`
	Query     string      `json:"query"`
	Mode      string      `json:"mode"`
	Nodes     []Node      `json:"nodes"`
	Edges     []Edge      `json:"edges"`
	Chunks    []Chunk     `json:"chunks"`
	Summary   string      `json:"summary"`
	Score     float64     `json:"score"`
	Timestamp time.Time   `json:"timestamp"`
}

type Node struct {
	ID    string                 `json:"id"`
	Type  string                 `json:"type"`
	Data  map[string]interface{} `json:"data"`
	Score float64                `json:"score"`
}

type Edge struct {
	Source     string                 `json:"source"`
	Target     string                 `json:"target"`
	Type       string                 `json:"type"`
	Weight     float64                `json:"weight"`
	Attributes map[string]interface{} `json:"attributes"`
}

type Chunk struct {
	ID        string  `json:"id"`
	Content   string  `json:"content"`
	Relevance float64 `json:"relevance"`
	Source    string  `json:"source,omitempty"`
}

type IndexStatus struct {
	KBId         string    `json:"kb_id"`
	Status       string    `json:"status"`
	Progress     float64   `json:"progress"`
	NodesCount   int       `json:"nodes_count"`
	EdgesCount   int       `json:"edges_count"`
	ChunksCount  int       `json:"chunks_count"`
	LastUpdated  time.Time `json:"last_updated"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

type ExtractionResult struct {
	Entities      []ExtractedEntity      `json:"entities"`
	Relationships []ExtractedRelationship `json:"relationships"`
	Chunks        []ExtractedChunk       `json:"chunks"`
}

type ExtractedEntity struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
}

type ExtractedRelationship struct {
	ID       string  `json:"id"`
	SourceID string  `json:"source_id"`
	TargetID string  `json:"target_id"`
	Type     string  `json:"type"`
	Weight   float64 `json:"weight"`
}

type ExtractedChunk struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Order   int    `json:"order"`
}
