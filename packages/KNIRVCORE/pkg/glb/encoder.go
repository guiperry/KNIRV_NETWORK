package glb

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type Metadata struct {
	Category  string
	Tags      []string
	Timestamp string
}

type GLBData struct {
	Asset struct {
		Version   string `json:"version"`
		Generator string `json:"generator"`
	} `json:"asset"`
	ExtensionsUsed []string               `json:"extensionsUsed"`
	Extensions     map[string]interface{} `json:"extensions"`
	Buffers        []Buffer               `json:"buffers"`
}

type Buffer struct {
	ByteLength int    `json:"byteLength"`
	URI        string `json:"uri"`
}

type Encoder struct{}

func NewEncoder() *Encoder {
	return &Encoder{}
}

func (e *Encoder) Encode(content string, metadata Metadata, embedding []float32) ([]byte, error) {
	glb := GLBData{}
	glb.Asset.Version = "2.0"
	glb.Asset.Generator = "KNIRVBASE v1.0"
	glb.ExtensionsUsed = []string{"KNIRV_memory_metadata"}
	glb.Extensions = map[string]interface{}{
		"KNIRV_memory_metadata": map[string]interface{}{
			"content_type":  "text/plain",
			"category":      metadata.Category,
			"tags":          metadata.Tags,
			"confidence":    0.87,
			"relationships": []string{},
			"embedding":     embedding,
		},
	}
	glb.Buffers = []Buffer{
		{
			ByteLength: len(content),
			URI:        fmt.Sprintf("data:text/plain;base64,%s", base64.StdEncoding.EncodeToString([]byte(content))),
		},
	}

	return json.Marshal(glb)
}

type Decoded struct {
	Content  string
	Metadata map[string]interface{}
}

func (e *Encoder) Decode(data []byte) (*Decoded, error) {
	var glb GLBData
	if err := json.Unmarshal(data, &glb); err != nil {
		return nil, err
	}

	metadata := glb.Extensions["KNIRV_memory_metadata"].(map[string]interface{})

	// Extract content from buffer
	if len(glb.Buffers) == 0 {
		return nil, fmt.Errorf("no buffers in GLB")
	}

	buffer := glb.Buffers[0]
	if len(buffer.URI) < 23 || buffer.URI[:23] != "data:text/plain;base64," {
		return nil, fmt.Errorf("invalid buffer URI")
	}

	contentBytes, err := base64.StdEncoding.DecodeString(buffer.URI[23:])
	if err != nil {
		return nil, err
	}

	return &Decoded{
		Content:  string(contentBytes),
		Metadata: metadata,
	}, nil
}
