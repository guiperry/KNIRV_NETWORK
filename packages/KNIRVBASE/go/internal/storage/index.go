package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/knirvcorp/knirvbase/go/internal/indexing"
	typ "github.com/knirvcorp/knirvbase/go/internal/types"
)

// IndexType represents the type of index
type IndexType string

const (
	IndexTypeBTree IndexType = "btree"
	IndexTypeGIN   IndexType = "gin"
	IndexTypeHNSW  IndexType = "hnsw"
	IndexTypeTag   IndexType = "tag"
)

// Index represents a secondary index
type Index struct {
	Name        string                 `json:"name"`
	Collection  string                 `json:"collection"`
	Type        IndexType              `json:"type"`
	Fields      []string               `json:"fields"`
	Unique      bool                   `json:"unique"`
	PartialExpr string                 `json:"partial_expr,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`

	// In-memory data structures
	btreeIndex *BTreeIndex
	ginIndex   *GINIndex
	hnswIndex  *HNSWIndex
	tagIndex   *indexing.TagIndex

	mu sync.RWMutex
}

// BTreeIndex for B-Tree indexes
type BTreeIndex struct {
	data map[string][]string // value -> []documentID
}

// GINIndex for Generalized Inverted Index (JSONB)
type GINIndex struct {
	data map[string][]string // token -> []documentID
}

// HNSWIndex for Hierarchical Navigable Small World (vectors)
type HNSWIndex struct {
	index   *indexing.HNSWIndex  // reference to the actual HNSW index implementation
	vectors map[string][]float32 // documentID -> vector for fast lookup
}

// IndexManager manages all indexes for a storage backend
type IndexManager struct {
	baseDir string
	indexes map[string]*Index // collection:indexName -> index
	mu      sync.RWMutex
}

// NewIndexManager creates a new index manager
func NewIndexManager(baseDir string) *IndexManager {
	return &IndexManager{
		baseDir: baseDir,
		indexes: make(map[string]*Index),
	}
}

// CreateIndex creates a new index
func (im *IndexManager) CreateIndex(collection, name string, indexType IndexType, fields []string, unique bool, partialExpr string, options map[string]interface{}) error {
	key := fmt.Sprintf("%s:%s", collection, name)

	im.mu.Lock()
	defer im.mu.Unlock()

	if _, exists := im.indexes[key]; exists {
		return fmt.Errorf("index %s already exists", key)
	}

	index := &Index{
		Name:        name,
		Collection:  collection,
		Type:        indexType,
		Fields:      fields,
		Unique:      unique,
		PartialExpr: partialExpr,
		Options:     options,
	}

	// Initialize index data structure
	switch indexType {
	case IndexTypeBTree:
		index.btreeIndex = &BTreeIndex{data: make(map[string][]string)}
	case IndexTypeGIN:
		index.ginIndex = &GINIndex{data: make(map[string][]string)}
	case IndexTypeHNSW:
		dim := 768 // default
		if d, ok := options["dimensions"].(float64); ok {
			dim = int(d)
		} else if d, ok := options["dimensions"].(int); ok {
			dim = d
		}
		m := 16 // default
		if mOpt, ok := options["m"].(float64); ok {
			m = int(mOpt)
		} else if mOpt, ok := options["m"].(int); ok {
			m = mOpt
		}
		efConstruction := 200 // default
		if efOpt, ok := options["ef_construction"].(float64); ok {
			efConstruction = int(efOpt)
		} else if efOpt, ok := options["ef_construction"].(int); ok {
			efConstruction = efOpt
		}
		index.hnswIndex = &HNSWIndex{
			index:   indexing.NewHNSWIndex(dim, m, efConstruction),
			vectors: make(map[string][]float32),
		}
	case IndexTypeTag:
		index.tagIndex = indexing.NewTagIndex()
	}

	im.indexes[key] = index

	// Save index metadata
	return im.saveIndexMetadata(index)
}

// DropIndex removes an index
func (im *IndexManager) DropIndex(collection, name string) error {
	key := fmt.Sprintf("%s:%s", collection, name)

	im.mu.Lock()
	defer im.mu.Unlock()

	if _, exists := im.indexes[key]; !exists {
		return fmt.Errorf("index %s does not exist", key)
	}

	delete(im.indexes, key)

	// Remove index files
	indexDir := filepath.Join(im.baseDir, collection, "indexes", name)
	os.RemoveAll(indexDir)

	return nil
}

// GetIndex returns an index by collection and name
func (im *IndexManager) GetIndex(collection, name string) *Index {
	key := fmt.Sprintf("%s:%s", collection, name)

	im.mu.RLock()
	defer im.mu.RUnlock()

	return im.indexes[key]
}

// GetIndexesForCollection returns all indexes for a collection
func (im *IndexManager) GetIndexesForCollection(collection string) []*Index {
	im.mu.RLock()
	defer im.mu.RUnlock()

	var indexes []*Index
	for _, idx := range im.indexes {
		if idx.Collection == collection {
			indexes = append(indexes, idx)
		}
	}
	return indexes
}

// Insert updates all relevant indexes for a document
func (im *IndexManager) Insert(collection string, doc map[string]interface{}) error {
	indexes := im.GetIndexesForCollection(collection)

	for _, idx := range indexes {
		if !im.matchesPartial(idx, doc) {
			continue
		}

		docID := doc["id"].(string)

		switch idx.Type {
		case IndexTypeBTree:
			im.insertBTree(idx, docID, doc)
		case IndexTypeGIN:
			im.insertGIN(idx, docID, doc)
		case IndexTypeHNSW:
			im.insertHNSW(idx, docID, doc)
		case IndexTypeTag:
			im.insertTag(idx, docID, doc)
		}
	}

	return nil
}

// Delete removes document from all relevant indexes
func (im *IndexManager) Delete(collection, docID string) error {
	indexes := im.GetIndexesForCollection(collection)

	for _, idx := range indexes {
		switch idx.Type {
		case IndexTypeBTree:
			im.deleteBTree(idx, docID)
		case IndexTypeGIN:
			im.deleteGIN(idx, docID)
		case IndexTypeHNSW:
			im.deleteHNSW(idx, docID)
		case IndexTypeTag:
			im.deleteTag(idx, docID)
		}
	}

	return nil
}

// QueryIndex performs an index-based query
func (im *IndexManager) QueryIndex(collection, indexName string, query map[string]interface{}) ([]string, error) {
	idx := im.GetIndex(collection, indexName)
	if idx == nil {
		return nil, fmt.Errorf("index %s:%s not found", collection, indexName)
	}

	switch idx.Type {
	case IndexTypeBTree:
		return im.queryBTree(idx, query)
	case IndexTypeGIN:
		return im.queryGIN(idx, query)
	case IndexTypeHNSW:
		return im.queryHNSW(idx, query)
	case IndexTypeTag:
		return im.queryTag(idx, query)
	default:
		return nil, fmt.Errorf("unsupported index type: %s", idx.Type)
	}
}

// matchesPartial checks if document matches partial index expression
func (im *IndexManager) matchesPartial(idx *Index, doc map[string]interface{}) bool {
	if idx.PartialExpr == "" {
		return true
	}

	// Simple partial expression evaluation (can be extended)
	// For now, support basic field=value expressions
	parts := strings.Split(idx.PartialExpr, "=")
	if len(parts) != 2 {
		return true
	}

	field := strings.TrimSpace(parts[0])
	expected := strings.TrimSpace(parts[1])

	if val, ok := doc[field]; ok {
		return fmt.Sprintf("%v", val) == expected
	}

	return false
}

// Tag index operations
func (im *IndexManager) insertTag(idx *Index, docID string, doc map[string]interface{}) {
	if idx.tagIndex == nil {
		return
	}
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Ensure the document has the correct ID field
	if doc["id"] == nil {
		doc = make(map[string]interface{}, len(doc)+1)
		for k, v := range doc {
			doc[k] = v
		}
		doc["id"] = docID
	}
	block := &blockWrapper{doc: doc}
	idx.tagIndex.Add(context.TODO(), block)
}

func (im *IndexManager) deleteTag(idx *Index, docID string) {
	if idx.tagIndex == nil {
		return
	}
	idx.mu.Lock()
	defer idx.mu.Unlock()

	id, _ := uuid.Parse(docID)
	idx.tagIndex.Remove(context.TODO(), id)
}

func (im *IndexManager) queryTag(idx *Index, query map[string]interface{}) ([]string, error) {
	if idx.tagIndex == nil {
		return nil, fmt.Errorf("tag index not initialized")
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	tags, ok := query["tags"].([]string)
	if !ok {
		// Try single tag
		if tag, ok := query["tag"].(string); ok {
			tags = []string{tag}
		} else {
			return nil, fmt.Errorf("invalid query for tag index")
		}
	}

	resultIDs, err := idx.tagIndex.Search(context.TODO(), tags)
	if err != nil {
		return nil, err
	}

	out := make([]string, len(resultIDs))
	for i, id := range resultIDs {
		out[i] = id.String()
	}
	return out, nil
}

// blockWrapper adapts map[string]interface{} to indexing.Block and indexing.Taggable
type blockWrapper struct {
	doc map[string]interface{}
}

func (w *blockWrapper) GetBlockID() uuid.UUID {
	idStr, _ := w.doc["id"].(string)
	id, _ := uuid.Parse(idStr)
	return id
}

func (w *blockWrapper) GetTimestamp() int64 {
	ts, _ := w.doc["_timestamp"].(int64)
	return ts
}

func (w *blockWrapper) GetCategory() typ.MemoryCategory {
	cat, _ := w.doc["category"].(string)
	return typ.MemoryCategory(cat)
}

func (w *blockWrapper) GetSemanticVector() []float32 {
	if payload, ok := w.doc["payload"].(map[string]interface{}); ok {
		if vector, ok := payload["vector"].([]interface{}); ok {
			vec := make([]float32, len(vector))
			for i, v := range vector {
				if f, ok := v.(float64); ok {
					vec[i] = float32(f)
				}
			}
			return vec
		}
	}
	return nil
}

func (w *blockWrapper) GetTags() []string {
	if payload, ok := w.doc["payload"].(map[string]interface{}); ok {
		if tags, ok := payload["tags"].([]interface{}); ok {
			out := make([]string, len(tags))
			for i, t := range tags {
				out[i], _ = t.(string)
			}
			return out
		}
	}
	return nil
}

// B-Tree index operations
func (im *IndexManager) insertBTree(idx *Index, docID string, doc map[string]interface{}) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Create composite key from fields
	key := im.buildCompositeKey(idx.Fields, doc)

	if idx.Unique {
		// Check uniqueness
		if existing, exists := idx.btreeIndex.data[key]; exists && len(existing) > 0 {
			// For unique indexes, only one document per key
			return
		}
	}

	idx.btreeIndex.data[key] = append(idx.btreeIndex.data[key], docID)
}

func (im *IndexManager) deleteBTree(idx *Index, docID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	for key, docIDs := range idx.btreeIndex.data {
		for i, id := range docIDs {
			if id == docID {
				idx.btreeIndex.data[key] = append(docIDs[:i], docIDs[i+1:]...)
				if len(idx.btreeIndex.data[key]) == 0 {
					delete(idx.btreeIndex.data, key)
				}
				break
			}
		}
	}
}

func (im *IndexManager) queryBTree(idx *Index, query map[string]interface{}) ([]string, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// For now, support exact match queries
	if value, ok := query["value"]; ok {
		key := fmt.Sprintf("%v", value)
		if docIDs, exists := idx.btreeIndex.data[key]; exists {
			return docIDs, nil
		}
	}

	return []string{}, nil
}

// GIN index operations (simplified)
func (im *IndexManager) insertGIN(idx *Index, docID string, doc map[string]interface{}) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Tokenize JSON fields
	tokens := im.tokenizeJSON(doc)
	for _, token := range tokens {
		idx.ginIndex.data[token] = append(idx.ginIndex.data[token], docID)
	}
}

func (im *IndexManager) deleteGIN(idx *Index, docID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	for token, docIDs := range idx.ginIndex.data {
		for i, id := range docIDs {
			if id == docID {
				idx.ginIndex.data[token] = append(docIDs[:i], docIDs[i+1:]...)
				if len(idx.ginIndex.data[token]) == 0 {
					delete(idx.ginIndex.data, token)
				}
				break
			}
		}
	}
}

func (im *IndexManager) queryGIN(idx *Index, query map[string]interface{}) ([]string, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if token, ok := query["token"].(string); ok {
		if docIDs, exists := idx.ginIndex.data[token]; exists {
			return docIDs, nil
		}
	}

	return []string{}, nil
}

// HNSW index operations
func (im *IndexManager) insertHNSW(idx *Index, docID string, doc map[string]interface{}) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Extract vector from document
	if payload, ok := doc["payload"].(map[string]interface{}); ok {
		if vector, ok := payload["vector"].([]interface{}); ok {
			vec := make([]float32, len(vector))
			for i, v := range vector {
				if f, ok := v.(float64); ok {
					vec[i] = float32(f)
				}
			}
			idx.hnswIndex.vectors[docID] = vec
			// Add to HNSW index
			idx.hnswIndex.index.Add(uuid.MustParse(docID), vec)
		}
	}
}

func (im *IndexManager) deleteHNSW(idx *Index, docID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	delete(idx.hnswIndex.vectors, docID)
	// Remove from HNSW index
	idx.hnswIndex.index.Remove(uuid.MustParse(docID))
}

func (im *IndexManager) queryHNSW(idx *Index, query map[string]interface{}) ([]string, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if queryVec, ok := query["vector"].([]float64); ok {
		// Convert to float32
		queryVec32 := make([]float32, len(queryVec))
		for i, v := range queryVec {
			queryVec32[i] = float32(v)
		}

		// Use HNSW index for search
		limit := 10
		if l, ok := query["limit"].(int); ok {
			limit = l
		}

		// Search the HNSW index
		neighborIDs, err := idx.hnswIndex.index.Search(queryVec32, limit)
		if err != nil {
			return nil, err
		}

		// Convert uuid.UUID to string
		var docIDs []string
		for _, id := range neighborIDs {
			docIDs = append(docIDs, id.String())
		}

		return docIDs, nil
	}

	return []string{}, nil
}

// Helper functions

func (im *IndexManager) buildCompositeKey(fields []string, doc map[string]interface{}) string {
	var parts []string
	for _, field := range fields {
		if val, ok := doc[field]; ok {
			parts = append(parts, fmt.Sprintf("%v", val))
		}
	}
	return strings.Join(parts, "|")
}

func (im *IndexManager) tokenizeJSON(doc map[string]interface{}) []string {
	var tokens []string
	im.tokenizeValue(doc, &tokens)
	return tokens
}

func (im *IndexManager) tokenizeValue(val interface{}, tokens *[]string) {
	switch v := val.(type) {
	case string:
		// Simple tokenization: split by spaces and lowercase
		words := strings.Fields(strings.ToLower(v))
		*tokens = append(*tokens, words...)
	case map[string]interface{}:
		for _, value := range v {
			im.tokenizeValue(value, tokens)
		}
	case []interface{}:
		for _, item := range v {
			im.tokenizeValue(item, tokens)
		}
	}
}

// CosineSimilarity computes the cosine similarity between two vectors
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (normA * normB)
}

func (im *IndexManager) saveIndexMetadata(idx *Index) error {
	indexDir := filepath.Join(im.baseDir, idx.Collection, "indexes", idx.Name)
	os.MkdirAll(indexDir, 0755)

	metaPath := filepath.Join(indexDir, "metadata.json")
	data, err := json.Marshal(idx)
	if err != nil {
		return err
	}

	return os.WriteFile(metaPath, data, 0644)
}

// LoadIndexes loads all index metadata from disk
func (im *IndexManager) LoadIndexes() error {
	collectionsDir := im.baseDir

	entries, err := os.ReadDir(collectionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		collection := entry.Name()
		indexesDir := filepath.Join(collectionsDir, collection, "indexes")

		if _, err := os.Stat(indexesDir); os.IsNotExist(err) {
			continue
		}

		indexEntries, err := os.ReadDir(indexesDir)
		if err != nil {
			continue
		}

		for _, idxEntry := range indexEntries {
			if !idxEntry.IsDir() {
				continue
			}

			indexName := idxEntry.Name()
			metaPath := filepath.Join(indexesDir, indexName, "metadata.json")

			data, err := os.ReadFile(metaPath)
			if err != nil {
				continue
			}

			var idx Index
			if err := json.Unmarshal(data, &idx); err != nil {
				continue
			}

			// Reinitialize data structures
			switch idx.Type {
			case IndexTypeBTree:
				idx.btreeIndex = &BTreeIndex{data: make(map[string][]string)}
			case IndexTypeGIN:
				idx.ginIndex = &GINIndex{data: make(map[string][]string)}
			case IndexTypeHNSW:
				dim := 768
				if d, ok := idx.Options["dimensions"].(float64); ok {
					dim = int(d)
				}
				m := 16
				if mOpt, ok := idx.Options["m"].(float64); ok {
					m = int(mOpt)
				}
				efConstruction := 200
				if efOpt, ok := idx.Options["ef_construction"].(float64); ok {
					efConstruction = int(efOpt)
				}
				idx.hnswIndex = &HNSWIndex{
					index:   indexing.NewHNSWIndex(dim, m, efConstruction),
					vectors: make(map[string][]float32),
				}
			}

			key := fmt.Sprintf("%s:%s", collection, indexName)
			im.indexes[key] = &idx
		}
	}

	return nil
}
