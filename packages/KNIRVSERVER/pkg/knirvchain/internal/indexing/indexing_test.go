package indexing

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHNSWIndex(t *testing.T) {
	index := NewHNSWIndex(128, 16, 200)
	assert.NotNil(t, index)
	assert.Equal(t, 128, index.dimension)
	assert.Equal(t, 16, index.M)
}

func TestHNSWIndex_AddAndSearch(t *testing.T) {
	index := NewHNSWIndex(4, 16, 200)

	id1 := uuid.New()
	vector1 := []float32{1.0, 0.0, 0.0, 0.0}
	err := index.Add(id1, vector1)
	require.NoError(t, err)

	id2 := uuid.New()
	vector2 := []float32{0.0, 1.0, 0.0, 0.0}
	err = index.Add(id2, vector2)
	require.NoError(t, err)

	id3 := uuid.New()
	vector3 := []float32{0.9, 0.1, 0.0, 0.0}
	err = index.Add(id3, vector3)
	require.NoError(t, err)

	results, err := index.Search([]float32{1.0, 0.0, 0.0, 0.0}, 2)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestHNSWIndex_DimensionMismatch(t *testing.T) {
	index := NewHNSWIndex(4, 16, 200)

	id := uuid.New()
	vector := []float32{1.0, 0.0}
	err := index.Add(id, vector)
	assert.Error(t, err)
	assert.Equal(t, ErrDimensionMismatch, err)
}

func TestHNSWIndex_SearchDimensionMismatch(t *testing.T) {
	index := NewHNSWIndex(4, 16, 200)

	id := uuid.New()
	vector := []float32{1.0, 0.0, 0.0, 0.0}
	err := index.Add(id, vector)
	require.NoError(t, err)

	_, err = index.Search([]float32{1.0, 0.0}, 1)
	assert.Error(t, err)
}

func TestHNSWIndex_Remove(t *testing.T) {
	index := NewHNSWIndex(4, 16, 200)

	id := uuid.New()
	vector := []float32{1.0, 0.0, 0.0, 0.0}
	err := index.Add(id, vector)
	require.NoError(t, err)

	err = index.Remove(id)
	require.NoError(t, err)

	results, err := index.Search(vector, 1)
	require.NoError(t, err)
	assert.Equal(t, 0, len(results))
}

func TestHNSWIndex_SetEf(t *testing.T) {
	index := NewHNSWIndex(4, 16, 200)
	assert.Equal(t, 200, index.ef)

	index.SetEf(50)
	assert.Equal(t, 50, index.ef)
}

func TestHNSWIndex_EmptySearch(t *testing.T) {
	index := NewHNSWIndex(4, 16, 200)

	results, err := index.Search([]float32{1.0, 0.0, 0.0, 0.0}, 1)
	require.NoError(t, err)
	assert.Equal(t, 0, len(results))
}

func TestNewSemanticIndex(t *testing.T) {
	index := NewSemanticIndex(128)
	assert.NotNil(t, index)
	assert.NotNil(t, index.hnsw)
}

func TestNewTemporalIndex(t *testing.T) {
	index := NewTemporalIndex()
	assert.NotNil(t, index)
	assert.NotNil(t, index.timeline)
}

func TestNewCategoryIndex(t *testing.T) {
	index := NewCategoryIndex()
	assert.NotNil(t, index)
	assert.NotNil(t, index.categories)
}

func TestNewMultiIndexManager(t *testing.T) {
	manager := NewMultiIndexManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.indexes)
}

func TestMultiIndexManager_RegisterAndGetIndex(t *testing.T) {
	manager := NewMultiIndexManager()

	semanticIdx := NewSemanticIndex(128)
	manager.RegisterIndex(IndexTypeSemantic, semanticIdx)

	retrieved := manager.GetIndex(IndexTypeSemantic)
	assert.Equal(t, semanticIdx, retrieved)
}

func TestMultiIndexManager_GetIndexNotFound(t *testing.T) {
	manager := NewMultiIndexManager()

	retrieved := manager.GetIndex(IndexTypeSemantic)
	assert.Nil(t, retrieved)
}
