package processing

import (
	"testing"

	"KNIRVGRAPH/internal/types"
)

func TestExtractorEntities(t *testing.T) {
	extractor := NewExtractor(types.ExtractionConfig{
		EnableEntities: true,
		MinConfidence:  0.5,
	})
	text := "Contact us at support@example.com or visit https://example.com. Server IP is 192.168.1.1."
	entities, _, err := extractor.Extract("doc1", text)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}
	if len(entities) == 0 {
		t.Fatal("Expected at least one entity")
	}
	found := false
	for _, e := range entities {
		if e.Type == "EMAIL" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected EMAIL entity")
	}
}

func TestExtractorRelationships(t *testing.T) {
	extractor := NewExtractor(types.ExtractionConfig{
		EnableEntities:     true,
		EnableRelationships: true,
		MinConfidence:      0.5,
	})
	text := "Alice works for Acme Corp. Bob owns Acme Corp. Acme Corp is located in New York."
	_, relationships, err := extractor.Extract("doc2", text)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}
	if len(relationships) == 0 {
		t.Fatal("Expected at least one relationship")
	}
}

func TestExtractorDisabled(t *testing.T) {
	extractor := NewExtractor(types.ExtractionConfig{
		EnableEntities:      false,
		EnableRelationships: false,
	})
	_, _, err := extractor.Extract("doc3", "some text")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}
}
