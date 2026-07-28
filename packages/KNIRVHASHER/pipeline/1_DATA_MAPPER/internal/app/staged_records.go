package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"data-mapper/internal/embedder"
)

// RawRecord mirrors Stage 0's canonical schema. It is intentionally duplicated
// across the independently-built stage modules to keep their Go modules
// decoupled; changes must be made in both schemas together.
type RawRecord struct {
	DatasetID string   `json:"dataset_id"`
	Split     string   `json:"split"`
	Index     int64    `json:"index"`
	Text      string   `json:"text"`
	Tags      []string `json:"tags"`
}

func stagedInputFiles(dir string) ([]string, error) {
	var files []string
	for _, pattern := range []string{"*.jsonl", "*.ndjson", "*.json"} {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}
	return files, nil
}

func stagedInputDir(config *Config) string {
	if override := os.Getenv("KNIRV_CONNECTOR_DIR"); override != "" {
		return override
	}
	if home, err := os.UserHomeDir(); err == nil {
		primary := filepath.Join(home, ".local", "share", "knirvhasher", "connector")
		if entries, _ := os.ReadDir(primary); len(entries) > 0 {
			return primary
		}
	}
	return filepath.Join(config.AppDataDir, "connector")
}

func readRawRecords(path string) ([]RawRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var records []RawRecord
	if filepath.Ext(path) == ".json" {
		if err := json.NewDecoder(f).Decode(&records); err != nil {
			return nil, err
		}
		return records, nil
	}
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64*1024), 2*1024*1024)
	for s.Scan() {
		var r RawRecord
		if err := json.Unmarshal(s.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		records = append(records, r)
	}
	return records, s.Err()
}

// ProcessStagedRecords is the Stage 1 path. It consumes only records emitted
// by Stage 0 and never performs source selection or network fetching.
func ProcessStagedRecords(ctx context.Context, config *Config) error {
	inputDir := stagedInputDir(config)
	files, err := stagedInputFiles(inputDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	provider := embedder.NewEmbeddingProvider(config.EmbeddingBackend, config.CloudflareEndpoint, strings.TrimSuffix(config.OllamaHost, "/")+"/api/embeddings", config.OllamaModel, config.CloudflareLimit)
	nlp, nlpErr := NewNLPBridge()
	if nlpErr != nil {
		log.Printf("staged mapper: %v", nlpErr)
	} else {
		defer nlp.Close()
	}
	results := make(chan DocumentRecord)
	errCh := make(chan error, 1)
	go func() {
		defer close(results)
		for _, path := range files {
			records, e := readRawRecords(path)
			if e != nil {
				errCh <- e
				return
			}
			for _, raw := range records {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				default:
				}
				text := strings.TrimSpace(raw.Text)
				if len(text) == 0 {
					continue
				}
				chunks := []string{text}
				if len(text) > 512 {
					chunks = ChunkText(text, config.ChunkSize, config.ChunkOverlap)
					if len(chunks) == 0 {
						chunks = []string{text}
					}
				}
				embeddings, e := provider.GetBatchEmbeddings(chunks)
				if e != nil {
					log.Printf("staged mapper embedding %s/%d: %v", raw.DatasetID, raw.Index, e)
					continue
				}
				for i, chunk := range chunks {
					var tokens []string
					var offsets []int32
					var pos, tense []int
					var deps []uint32
					if nlp != nil {
						tokens, offsets, pos, tense, deps = nlp.ProcessText(chunk)
					}
					results <- DocumentRecord{FileName: fmt.Sprintf("%s/%s/%d", raw.DatasetID, raw.Split, raw.Index), ChunkID: int32(i), Content: chunk, Embedding: embeddings[i], Tokens: tokens, TokenOffsets: offsets, POSTags: pos, Tenses: tense, DepHashes: deps, DatasetID: raw.DatasetID, Split: raw.Split, SourceIndex: raw.Index}
				}
			}
		}
	}()
	if err := writeOutput(config.OutputFile, results); err != nil {
		return err
	}
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}
