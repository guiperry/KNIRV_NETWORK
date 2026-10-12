// Package connector contains the source-independent ingestion primitives used
// by Stage 0.  Keep this package free of KNIRVBASE imports: Stage 0 produces
// records, while the standalone KNIRVBASE process owns persistence.
package connector

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// RawRecord is the canonical hand-off between the connector and data-mapper.
type RawRecord struct {
	DatasetID string   `json:"dataset_id" parquet:"dataset_id"`
	Split     string   `json:"split" parquet:"split"`
	Index     int64    `json:"index" parquet:"index"`
	Heading   string   `json:"heading,omitempty" parquet:"heading"`
	Text      string   `json:"text" parquet:"text"`
	Tags      []string `json:"tags" parquet:"tags"`
}

// HuggingFaceConfig controls the source connector without coupling callers to
// the HuggingFace API implementation.
type HuggingFaceConfig struct {
	DatasetIDs   []string
	Splits       []string
	MaxRowsPerDS int
	ShardWorkers int
	CacheDir     string
	Token        string
}

// NormalizeRow applies the documented first-match-wins text rules.
func NormalizeRow(row map[string]any) string {
	if s := stringValue(row["text"]); s != "" {
		return s
	}
	if instruction, ok := stringValueOK(row["instruction"]); ok {
		if output, ok := stringValueOK(row["output"]); ok {
			return "### Instruction:\n" + instruction + "\n\n### Response:\n" + output
		}
	}
	if input, ok := stringValueOK(row["input"]); ok {
		if output, ok := stringValueOK(row["output"]); ok {
			return input + "\n" + output
		}
	}
	if messages, ok := row["messages"].([]any); ok {
		parts := make([]string, 0, len(messages))
		for _, message := range messages {
			if obj, ok := message.(map[string]any); ok {
				if content, ok := stringValueOK(obj["content"]); ok {
					parts = append(parts, content)
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	keys := make([]string, 0, len(row))
	for key, value := range row {
		if _, ok := stringValueOK(value); ok {
			keys = append(keys, key)
		}
	}
	// JSON object iteration is deliberately sorted so fallback output is stable.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, stringValue(row[key]))
	}
	return strings.Join(parts, "\n")
}

func stringValue(v any) string { s, _ := stringValueOK(v); return s }
func stringValueOK(v any) (string, bool) {
	s, ok := v.(string)
	return strings.TrimSpace(s), ok && strings.TrimSpace(s) != ""
}

type ontologyStats struct {
	EntityCount int64 `json:"entityCount"`
}

// OntologyAvailable checks the server's real HTTP endpoint. Any unavailable
// or malformed response is treated as unavailable so source fallback remains
// deterministic and non-fatal.
func OntologyAvailable(client *http.Client, baseURL string, minimum int64) (bool, error) {
	if client == nil {
		client = http.DefaultClient
	}
	u, err := url.JoinPath(strings.TrimRight(baseURL, "/"), "/api/ontology/stats")
	if err != nil {
		return false, err
	}
	resp, err := client.Get(u)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("ontology stats: HTTP %s", resp.Status)
	}
	var stats ontologyStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return false, err
	}
	if minimum < 1 {
		minimum = 1
	}
	return stats.EntityCount >= minimum, nil
}

// NextOntologyURL builds a page URL while keeping pagination policy in one
// place for the entity and relation endpoints.
func NextOntologyURL(baseURL, resource string, offset, limit int) string {
	u, _ := url.JoinPath(strings.TrimRight(baseURL, "/"), "/api/ontology/"+resource)
	q := url.Values{}
	q.Set("offset", strconv.Itoa(offset))
	q.Set("limit", strconv.Itoa(limit))
	return u + "?" + q.Encode()
}
