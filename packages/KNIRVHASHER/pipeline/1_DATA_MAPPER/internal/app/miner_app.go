package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"data-mapper/internal/cleaner"
	"data-mapper/internal/normalizer"
	"data-mapper/internal/writer"
)

// MinerApp reads raw .md files written by 0_DATA_CONNECTOR, cleans and
// normalises each document via the SpaCy NLP pipeline, then writes the output
// as .arrow IPC files into processedDir for consumption by 2_DATA_ENCODER.
type MinerApp struct {
	rawDir     string
	normalizer *normalizer.SecurityNormalizer
	cleaner    *cleaner.Cleaner
	writer     *writer.ArrowWriter
}

// NewMinerApp constructs a MinerApp.
//   - rawDir      – directory containing raw .md files from 0_DATA_CONNECTOR
//   - processedDir – directory to write processed .arrow files
func NewMinerApp(rawDir, processedDir string) *MinerApp {
	return &MinerApp{
		rawDir:     rawDir,
		normalizer: normalizer.NewSecurityNormalizer(),
		cleaner:    cleaner.New(),
		writer:     writer.NewArrowWriter(processedDir),
	}
}

// Run iterates over all .md files in rawDir, runs each through the
// cleaner → normaliser chain, and writes the resulting SecurityRecords as a
// .arrow IPC file to processedDir.
func (m *MinerApp) Run(ctx context.Context) error {
	pattern := filepath.Join(m.rawDir, "*.md")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob %s: %w", pattern, err)
	}

	for _, path := range files {
		if err := ctx.Err(); err != nil {
			return err
		}

		docID := filepath.Base(path)

		raw, err := os.ReadFile(path)
		if err != nil {
			log.Printf("miner: read %s: %v", docID, err)
			continue
		}

		cleaned, err := m.cleaner.CleanMarkdown(string(raw))
		if err != nil {
			log.Printf("miner: cleaner skip %s: %v", docID, err)
			continue
		}

		records, err := m.normalizer.Process(cleaned)
		if err != nil {
			log.Printf("miner: normalizer skip %s: %v", docID, err)
			continue
		}

		if err := m.writer.WriteBatch(docID, records); err != nil {
			log.Printf("miner: arrow writer %s: %v", docID, err)
		}
	}
	return nil
}
