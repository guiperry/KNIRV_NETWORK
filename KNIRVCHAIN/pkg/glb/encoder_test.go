package glb

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEncoder(t *testing.T) {
	encoder := NewEncoder()
	assert.NotNil(t, encoder)
}

func TestEncoder_Encode(t *testing.T) {
	encoder := NewEncoder()

	content := "test content"
	metadata := Metadata{
		Category:  "GENERAL",
		Tags:      []string{"test", "memory"},
		Timestamp: "2023-01-01T00:00:00Z",
	}
	embedding := []float32{0.1, 0.2, 0.3}

	data, err := encoder.Encode(content, metadata, embedding)
	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.True(t, len(data) > 0)

	// Verify it's valid JSON
	assert.Contains(t, string(data), "KNIRV_memory_metadata")
	assert.Contains(t, string(data), "GENERAL")
	assert.Contains(t, string(data), "data:text/plain;base64")
}

func TestEncoder_Decode(t *testing.T) {
	encoder := NewEncoder()

	content := "test content"
	metadata := Metadata{
		Category:  "GENERAL",
		Tags:      []string{"test", "memory"},
		Timestamp: "2023-01-01T00:00:00Z",
	}
	embedding := []float32{0.1, 0.2, 0.3}

	// Encode first
	data, err := encoder.Encode(content, metadata, embedding)
	assert.NoError(t, err)

	// Decode
	decoded, err := encoder.Decode(data)
	assert.NoError(t, err)
	assert.NotNil(t, decoded)
	assert.Equal(t, content, decoded.Content)
	assert.NotNil(t, decoded.Metadata)

	// Check metadata
	assert.Equal(t, "GENERAL", decoded.Metadata["category"])
	assert.Equal(t, []interface{}{"test", "memory"}, decoded.Metadata["tags"])
}

func TestEncoder_Decode_InvalidData(t *testing.T) {
	encoder := NewEncoder()

	// Test invalid JSON
	_, err := encoder.Decode([]byte("invalid json"))
	assert.Error(t, err)

	// Test empty data
	_, err = encoder.Decode([]byte{})
	assert.Error(t, err)
}

func TestEncoder_Decode_NoBuffers(t *testing.T) {
	encoder := NewEncoder()

	// Create GLB data without buffers
	glbData := GLBData{
		Asset: struct {
			Version   string `json:"version"`
			Generator string `json:"generator"`
		}{
			Version:   "2.0",
			Generator: "test",
		},
		ExtensionsUsed: []string{"KNIRV_memory_metadata"},
		Extensions: map[string]interface{}{
			"KNIRV_memory_metadata": map[string]interface{}{
				"category": "GENERAL",
			},
		},
		Buffers: []Buffer{}, // Empty buffers
	}

	data, _ := json.Marshal(glbData)
	_, err := encoder.Decode(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no buffers")
}

func TestEncoder_Decode_InvalidBufferURI(t *testing.T) {
	encoder := NewEncoder()

	// Create GLB data with invalid buffer URI
	glbData := GLBData{
		Asset: struct {
			Version   string `json:"version"`
			Generator string `json:"generator"`
		}{
			Version:   "2.0",
			Generator: "test",
		},
		ExtensionsUsed: []string{"KNIRV_memory_metadata"},
		Extensions: map[string]interface{}{
			"KNIRV_memory_metadata": map[string]interface{}{
				"category": "GENERAL",
			},
		},
		Buffers: []Buffer{
			{
				ByteLength: 10,
				URI:        "invalid-uri",
			},
		},
	}

	data, _ := json.Marshal(glbData)
	_, err := encoder.Decode(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid buffer URI")
}

func TestEncoder_Decode_InvalidBase64(t *testing.T) {
	encoder := NewEncoder()

	// Create GLB data with invalid base64
	glbData := GLBData{
		Asset: struct {
			Version   string `json:"version"`
			Generator string `json:"generator"`
		}{
			Version:   "2.0",
			Generator: "test",
		},
		ExtensionsUsed: []string{"KNIRV_memory_metadata"},
		Extensions: map[string]interface{}{
			"KNIRV_memory_metadata": map[string]interface{}{
				"category": "GENERAL",
			},
		},
		Buffers: []Buffer{
			{
				ByteLength: 10,
				URI:        "data:text/plain;base64,invalid-base64!!!",
			},
		},
	}

	data, _ := json.Marshal(glbData)
	_, err := encoder.Decode(data)
	assert.Error(t, err)
}
