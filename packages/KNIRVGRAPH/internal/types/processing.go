package types

import "time"

type ChunkingStrategy string

const (
	ChunkStrategyRecursive ChunkingStrategy = "recursive"
	ChunkStrategyToken     ChunkingStrategy = "token"
	ChunkStrategySemantic  ChunkingStrategy = "semantic"
)

type Chunk struct {
	ID        string                 `json:"id"`
	DocumentID string                `json:"document_id"`
	Text      string                 `json:"text"`
	Index     int                    `json:"index"`
	StartOffset int                  `json:"start_offset"`
	EndOffset   int                  `json:"end_offset"`
	Metadata    map[string]interface{} `json:"metadata"`
	Embedding   []float32             `json:"embedding,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
}

type ProcessedDocument struct {
	ID          string                 `json:"id"`
	SourceID    string                 `json:"source_id"`
	Content     string                 `json:"content"`
	Chunks      []Chunk                `json:"chunks"`
	Entities    []ExtractedEntity      `json:"entities"`
	Relationships []ExtractedRelationship `json:"relationships"`
	Metadata    map[string]interface{} `json:"metadata"`
	Status      DocumentStatus         `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type DocumentStatus string

const (
	DocumentStatusPending    DocumentStatus = "pending"
	DocumentStatusProcessing DocumentStatus = "processing"
	DocumentStatusIndexed    DocumentStatus = "indexed"
	DocumentStatusFailed     DocumentStatus = "failed"
)

type ExtractedEntity struct {
	ID         string                 `json:"id"`
	DocumentID string                 `json:"document_id"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	Confidence float64                `json:"confidence"`
	ChunkIDs   []string               `json:"chunk_ids"`
	CreatedAt  time.Time              `json:"created_at"`
}

type ExtractedRelationship struct {
	ID         string                 `json:"id"`
	DocumentID string                 `json:"document_id"`
	Source     string                 `json:"source"`
	Target     string                 `json:"target"`
	Type       string                 `json:"type"`
	Weight     float64                `json:"weight"`
	Evidence   string                 `json:"evidence"`
	Confidence float64                `json:"confidence"`
	CreatedAt  time.Time              `json:"created_at"`
}

type ChunkingConfig struct {
	Strategy    ChunkingStrategy `json:"strategy"`
	ChunkSize   int              `json:"chunk_size"`
	Overlap     int              `json:"overlap"`
	Separators  []string         `json:"separators"`
	LengthFunc  func(string) int `json:"-"`
}

type ExtractionConfig struct {
	EnableEntities     bool     `json:"enable_entities"`
	EnableRelationships bool    `json:"enable_relationships"`
	EntityTypes        []string `json:"entity_types"`
	MinConfidence      float64  `json:"min_confidence"`
	LLMEndpoint        string   `json:"llm_endpoint"`
	LLMModel           string   `json:"llm_model"`
}

type EmbeddingProviderType string

const (
	EmbeddingProviderDeterministic EmbeddingProviderType = "deterministic"
	EmbeddingProviderOllama       EmbeddingProviderType = "ollama"
	EmbeddingProviderTextEmbedder EmbeddingProviderType = "text_embedder"
	EmbeddingProviderStub         EmbeddingProviderType = "stub"
)

type EmbeddingProviderConfig struct {
	Type           EmbeddingProviderType `json:"type"`
	Endpoint       string                `json:"endpoint"`
	Model          string                `json:"model"`
	Dimension      int                   `json:"dimension"`
	BatchSize      int                   `json:"batch_size"`
	TimeoutSeconds int                   `json:"timeout_seconds"`
	APIKey         string                `json:"api_key"`
}

type VectorSearchResult struct {
	ID       string                 `json:"id"`
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
	Vector   []float32              `json:"vector,omitempty"`
}

type RetrievalQuery struct {
	Query        string                 `json:"query"`
	TopK         int                    `json:"top_k"`
	Filters      map[string]interface{} `json:"filters"`
	IncludeVector bool                  `json:"include_vector"`
	HybridWeight float64                `json:"hybrid_weight"`
}

type RetrievalResult struct {
	Query      string                `json:"query"`
	Results    []VectorSearchResult  `json:"results"`
	TotalFound int                   `json:"total_found"`
	LatencyMs  int64                 `json:"latency_ms"`
	Strategy   string                `json:"strategy"`
}

type SynthesisRequest struct {
	Query        string                 `json:"query"`
	Contexts     []RetrievalResult      `json:"contexts"`
	MaxTokens    int                    `json:"max_tokens"`
	Temperature  float64                `json:"temperature"`
	LLMEndpoint  string                 `json:"llm_endpoint"`
	LLMModel     string                 `json:"llm_model"`
}

type SynthesisResponse struct {
	Answer       string `json:"answer"`
	Reasoning    string `json:"reasoning,omitempty"`
	Confidence   float64 `json:"confidence"`
	Sources      []string `json:"sources"`
	LatencyMs    int64   `json:"latency_ms"`
}

type IndexDocumentRequest struct {
	Document  ProcessedDocument `json:"document"`
	Overwrite bool              `json:"overwrite"`
}

type QueryRequest struct {
	Query      string                 `json:"query"`
	TopK       int                    `json:"top_k"`
	Filters    map[string]interface{} `json:"filters"`
	UseHybrid  bool                   `json:"use_hybrid"`
	UseRerank  bool                   `json:"use_rerank"`
	Synthesize bool                   `json:"synthesize"`
}

type QueryResponse struct {
	Answer     string                `json:"answer"`
	Reasoning  string                `json:"reasoning,omitempty"`
	Results    []RetrievalResult     `json:"results"`
	Sources    []string              `json:"sources"`
	Confidence float64               `json:"confidence"`
	LatencyMs  int64                 `json:"latency_ms"`
}
