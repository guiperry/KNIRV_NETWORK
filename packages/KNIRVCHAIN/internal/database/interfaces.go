package database

import (
	"context"
	"time"
)

// DatabaseManager defines the interface for database management operations
type DatabaseManager interface {
	// Database lifecycle
	Initialize(config *DatabaseConfig) error
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool
	
	// Database operations
	CreateDatabase(name string) error
	DropDatabase(name string) error
	ListDatabases() ([]string, error)
	
	// Health and monitoring
	Ping() error
	GetStats() (*DatabaseStats, error)
	
	// Backup and restore
	Backup(path string) error
	Restore(path string) error
}

// LevelDBManager defines the interface for LevelDB operations
type LevelDBManager interface {
	// Basic operations
	Put(key, value []byte) error
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
	Has(key []byte) (bool, error)
	
	// Batch operations
	NewBatch() Batch
	WriteBatch(batch Batch) error
	
	// Iteration
	NewIterator(prefix []byte) Iterator
	
	// Database management
	Close() error
	CompactRange(start, limit []byte) error
	GetProperty(property string) (string, error)
}

// ChromemDBManager defines the interface for ChromemDB operations
type ChromemDBManager interface {
	// Collection operations
	CreateCollection(name string, metadata map[string]interface{}) error
	DeleteCollection(name string) error
	GetCollection(name string) (Collection, error)
	ListCollections() ([]string, error)
	
	// Document operations
	AddDocuments(collectionName string, documents []Document) error
	UpdateDocuments(collectionName string, documents []Document) error
	DeleteDocuments(collectionName string, ids []string) error
	
	// Query operations
	Query(collectionName string, query QueryRequest) (*QueryResult, error)
	Search(collectionName string, searchText string, nResults int) ([]*SearchResult, error)
	
	// Embedding operations
	GenerateEmbeddings(texts []string) ([][]float32, error)
	
	// Lifecycle
	Start(ctx context.Context) error
	Stop() error
}

// ReflectionManager defines the interface for reflection database operations
type ReflectionManager interface {
	// Reflection operations
	StoreReflection(reflection *Reflection) error
	GetReflection(id string) (*Reflection, error)
	GetReflectionsByAgent(agentID string) ([]*Reflection, error)
	DeleteReflection(id string) error
	
	// Query operations
	SearchReflections(query *ReflectionQuery) ([]*Reflection, error)
	GetReflectionHistory(agentID string, limit int) ([]*Reflection, error)
	
	// Analysis operations
	AnalyzeReflections(agentID string) (*ReflectionAnalysis, error)
	GetReflectionTrends(agentID string, timeRange TimeRange) (*ReflectionTrends, error)
	
	// Lifecycle
	Start(ctx context.Context) error
	Stop() error
}

// ConversionManager defines the interface for database conversion operations
type ConversionManager interface {
	// Conversion operations
	ConvertToChromem(sourceDB string, targetCollection string) error
	ConvertFromChromem(sourceCollection string, targetDB string) error
	
	// Migration operations
	MigrateData(source, target DatabaseConfig) error
	ValidateMigration(source, target DatabaseConfig) error
	
	// Schema operations
	ConvertSchema(sourceSchema, targetSchema interface{}) error
	ValidateSchema(schema interface{}) error
}

// Batch defines the interface for batch operations
type Batch interface {
	Put(key, value []byte)
	Delete(key []byte)
	Clear()
	Count() int
}

// Iterator defines the interface for database iteration
type Iterator interface {
	Valid() bool
	Next()
	Key() []byte
	Value() []byte
	Error() error
	Release()
}

// Collection defines the interface for ChromemDB collections
type Collection interface {
	Name() string
	Count() (int, error)
	Add(documents []Document) error
	Update(documents []Document) error
	Delete(ids []string) error
	Query(query QueryRequest) (*QueryResult, error)
	GetMetadata() map[string]interface{}
}

// Document represents a document in ChromemDB
type Document struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Embedding []float32             `json:"embedding,omitempty"`
}

// QueryRequest represents a query request
type QueryRequest struct {
	QueryTexts   []string               `json:"query_texts"`
	NResults     int                    `json:"n_results"`
	Where        map[string]interface{} `json:"where,omitempty"`
	WhereDocument map[string]interface{} `json:"where_document,omitempty"`
	Include      []string               `json:"include,omitempty"`
}

// QueryResult represents a query result
type QueryResult struct {
	IDs       [][]string               `json:"ids"`
	Documents [][]string               `json:"documents"`
	Metadatas [][]map[string]interface{} `json:"metadatas"`
	Distances [][]float32              `json:"distances"`
	Embeddings [][]float32             `json:"embeddings,omitempty"`
}

// SearchResult represents a search result
type SearchResult struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Score    float32                `json:"score"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Reflection represents a reflection entry
type Reflection struct {
	ID        string                 `json:"id"`
	AgentID   string                 `json:"agent_id"`
	Type      string                 `json:"type"`
	Content   string                 `json:"content"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Tags      []string               `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ReflectionQuery represents a reflection query
type ReflectionQuery struct {
	AgentID   string    `json:"agent_id,omitempty"`
	Type      string    `json:"type,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
}

// ReflectionAnalysis represents analysis of reflections
type ReflectionAnalysis struct {
	AgentID       string                 `json:"agent_id"`
	TotalCount    int                    `json:"total_count"`
	TypeCounts    map[string]int         `json:"type_counts"`
	TagCounts     map[string]int         `json:"tag_counts"`
	TimeRange     TimeRange              `json:"time_range"`
	Insights      []string               `json:"insights"`
	Patterns      []Pattern              `json:"patterns"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ReflectionTrends represents trends in reflections
type ReflectionTrends struct {
	AgentID    string      `json:"agent_id"`
	TimeRange  TimeRange   `json:"time_range"`
	DataPoints []DataPoint `json:"data_points"`
	Trends     []Trend     `json:"trends"`
}

// TimeRange represents a time range
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Pattern represents a detected pattern
type Pattern struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Frequency   int                    `json:"frequency"`
	Confidence  float64                `json:"confidence"`
	Examples    []string               `json:"examples"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DataPoint represents a data point in trends
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Type      string    `json:"type"`
}

// Trend represents a trend
type Trend struct {
	Type        string  `json:"type"`
	Direction   string  `json:"direction"` // increasing, decreasing, stable
	Strength    float64 `json:"strength"`  // 0-1
	Description string  `json:"description"`
}

// DatabaseStats represents database statistics
type DatabaseStats struct {
	Size          int64     `json:"size"`
	RecordCount   int64     `json:"record_count"`
	LastBackup    time.Time `json:"last_backup"`
	LastCompaction time.Time `json:"last_compaction"`
	ReadOps       int64     `json:"read_ops"`
	WriteOps      int64     `json:"write_ops"`
	ErrorCount    int64     `json:"error_count"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Type         string                 `json:"type"`
	Path         string                 `json:"path"`
	Host         string                 `json:"host,omitempty"`
	Port         int                    `json:"port,omitempty"`
	Username     string                 `json:"username,omitempty"`
	Password     string                 `json:"password,omitempty"`
	Database     string                 `json:"database,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
	MaxOpenConns int                    `json:"max_open_conns,omitempty"`
	MaxIdleConns int                    `json:"max_idle_conns,omitempty"`
	Timeout      time.Duration          `json:"timeout,omitempty"`
}

// BackupConfig represents backup configuration
type BackupConfig struct {
	Enabled      bool          `json:"enabled"`
	Interval     time.Duration `json:"interval"`
	RetentionDays int          `json:"retention_days"`
	Path         string        `json:"path"`
	Compression  bool          `json:"compression"`
}

// IndexConfig represents index configuration
type IndexConfig struct {
	Name    string   `json:"name"`
	Fields  []string `json:"fields"`
	Unique  bool     `json:"unique"`
	Sparse  bool     `json:"sparse"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// TransactionContext represents a database transaction context
type TransactionContext interface {
	Commit() error
	Rollback() error
	Put(key, value []byte) error
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
}

// DatabaseEvent represents database events
type DatabaseEvent struct {
	Type      string                 `json:"type"`
	Database  string                 `json:"database"`
	Operation string                 `json:"operation"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// EventHandler defines the function signature for database event handlers
type EventHandler func(event *DatabaseEvent) error

// DatabaseMetrics represents database performance metrics
type DatabaseMetrics struct {
	ReadLatency    time.Duration `json:"read_latency"`
	WriteLatency   time.Duration `json:"write_latency"`
	Throughput     float64       `json:"throughput"`
	CacheHitRatio  float64       `json:"cache_hit_ratio"`
	DiskUsage      int64         `json:"disk_usage"`
	MemoryUsage    int64         `json:"memory_usage"`
	ConnectionCount int          `json:"connection_count"`
}

// Error types for database operations
var (
	ErrDatabaseNotFound   = NewDatabaseError("database not found")
	ErrCollectionNotFound = NewDatabaseError("collection not found")
	ErrDocumentNotFound   = NewDatabaseError("document not found")
	ErrInvalidQuery       = NewDatabaseError("invalid query")
	ErrConnectionFailed   = NewDatabaseError("connection failed")
	ErrTransactionFailed  = NewDatabaseError("transaction failed")
	ErrBackupFailed       = NewDatabaseError("backup failed")
	ErrRestoreFailed      = NewDatabaseError("restore failed")
)

// DatabaseError represents a database-specific error
type DatabaseError struct {
	Message string
	Code    string
}

func (e *DatabaseError) Error() string {
	return e.Message
}

func NewDatabaseError(message string) *DatabaseError {
	return &DatabaseError{Message: message}
}
