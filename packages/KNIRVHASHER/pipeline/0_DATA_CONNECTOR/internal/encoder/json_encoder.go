package encoder

import (
	"encoding/json"
	"fmt"
	"os"

	"data-connector/internal/normalizer"
)

type JSONEncoder struct{}

func NewJSONEncoder() *JSONEncoder {
	return &JSONEncoder{}
}

func (e *JSONEncoder) Encode(records []*normalizer.SecurityRecord) ([]byte, error) {
	type JSONRecord struct {
		FileName     string    `json:"file_name"`
		ChunkID      int32     `json:"chunk_id"`
		Content      string    `json:"content"`
		Tokens       []string  `json:"tokens"`
		POSTags      []int     `json:"pos_tags"`
		DepHashes    []uint32  `json:"dep_hashes"`
		SecurityTags []string  `json:"security_tags"`
		Slot4        int       `json:"slot4"`
		Slot10       int       `json:"slot10"`
		Weight       float64   `json:"weight"`
		Embedding    []float32 `json:"embedding"`
	}

	jsonRecords := make([]JSONRecord, len(records))
	for i, rec := range records {
		jsonRecords[i] = JSONRecord{
			FileName:     rec.FileName,
			ChunkID:      rec.ChunkID,
			Content:      rec.Content,
			Tokens:       rec.Tokens,
			POSTags:      rec.POSTags,
			DepHashes:    rec.DepHashes,
			SecurityTags: rec.SecurityTags,
			Slot4:        rec.Slot4,
			Slot10:       rec.Slot10,
			Weight:       rec.Weight,
			Embedding:    rec.GetEmbedding(),
		}
	}

	return json.MarshalIndent(jsonRecords, "", "  ")
}

func (e *JSONEncoder) EncodeToFile(records []*normalizer.SecurityRecord, path string) error {
	data, err := e.Encode(records)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

type JSONFrameFile struct {
	Version   string       `json:"version"`
	CreatedAt string       `json:"created_at"`
	Count     int          `json:"count"`
	Records   []JSONRecord `json:"records"`
}

type JSONRecord struct {
	FileName     string    `json:"file_name"`
	ChunkID      int32     `json:"chunk_id"`
	Content      string    `json:"content"`
	Tokens       []string  `json:"tokens"`
	POSTags      []int     `json:"pos_tags"`
	DepHashes    []uint32  `json:"dep_hashes"`
	SecurityTags []string  `json:"security_tags"`
	Slot4        int       `json:"slot4"`
	Slot10       int       `json:"slot10"`
	Weight       float64   `json:"weight"`
	Embedding    []float32 `json:"embedding"`
}
