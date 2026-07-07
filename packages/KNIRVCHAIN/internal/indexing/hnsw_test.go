package indexing

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHNSWIndex_AddSingle(t *testing.T) {
	index := NewHNSWIndex(4, 16, 200)

	id := uuid.New()
	vector := []float32{1.0, 0.0, 0.0, 0.0}
	err := index.Add(id, vector)
	require.NoError(t, err)
}

func TestHNSWIndex_MultipleVectors(t *testing.T) {
	index := NewHNSWIndex(4, 16, 200)

	vectors := []struct {
		id     uuid.UUID
		vector []float32
	}{
		{uuid.New(), []float32{1.0, 0.0, 0.0, 0.0}},
		{uuid.New(), []float32{0.0, 1.0, 0.0, 0.0}},
		{uuid.New(), []float32{0.0, 0.0, 1.0, 0.0}},
		{uuid.New(), []float32{0.0, 0.0, 0.0, 1.0}},
	}

	for _, v := range vectors {
		err := index.Add(v.id, v.vector)
		require.NoError(t, err)
	}

	results, err := index.Search([]float32{1.0, 0.0, 0.0, 0.0}, 2)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestHNSWIndex_SimilarVectors(t *testing.T) {
	index := NewHNSWIndex(4, 16, 200)

	id1 := uuid.New()
	index.Add(id1, []float32{1.0, 0.0, 0.0, 0.0})

	id2 := uuid.New()
	index.Add(id2, []float32{0.9, 0.1, 0.0, 0.0})

	results, err := index.Search([]float32{1.0, 0.0, 0.0, 0.0}, 2)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestHNSWIndex_CosineSimilarity(t *testing.T) {
	index := NewHNSWIndex(4, 16, 200)

	index.Add(uuid.New(), []float32{1.0, 0.0, 0.0, 0.0})
	index.Add(uuid.New(), []float32{-1.0, 0.0, 0.0, 0.0})

	results, err := index.Search([]float32{1.0, 0.0, 0.0, 0.0}, 1)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestHNSWIndex_SearchEmpty(t *testing.T) {
	index := NewHNSWIndex(4, 16, 200)

	results, err := index.Search([]float32{1.0, 0.0, 0.0, 0.0}, 1)
	require.NoError(t, err)
	assert.Equal(t, 0, len(results))
}

func TestHNSWIndex_SearchWithK(t *testing.T) {
	index := NewHNSWIndex(4, 16, 200)

	for i := 0; i < 5; i++ {
		index.Add(uuid.New(), []float32{float32(i), 0.0, 0.0, 0.0})
	}

	results, err := index.Search([]float32{1.0, 0.0, 0.0, 0.0}, 3)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), 3)
}
