package app

import (
	"context"
	"fmt"
	"log"

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/ipc"
	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/knirvcorp/knirvbase"
	"github.com/knirvcorp/knirvhasher/pipeline/1_DATA_MINER/internal/cleaner"
	"github.com/knirvcorp/knirvhasher/pipeline/1_DATA_MINER/internal/normalizer"
	"github.com/knirvcorp/knirvhasher/pipeline/1_DATA_MINER/internal/writer"
)

// MinerApp reads raw .md files from the KNIRVBASE connector_raw collection,
// cleans and normalises each document via SpaCy NLP, then writes the output
// as .arrow IPC files into miner_processed for 2_DATA_ENCODER.
type MinerApp struct {
	db         knirvbase.DB
	normalizer *normalizer.SecurityNormalizer
	cleaner    *cleaner.Cleaner
	writer     *writer.ArrowWriter
}

func NewMinerApp(db knirvbase.DB) *MinerApp {
	return &MinerApp{
		db:         db,
		normalizer: normalizer.NewSecurityNormalizer(),
		cleaner:    cleaner.New(),
		writer:     writer.NewArrowWriter(db.Collection("miner_processed")),
	}
}

// Run opens a Flight stream on the connector_raw collection (raw .md files),
// processes each document through the cleaner → normalizer chain, and writes
// the resulting SecurityRecords as a .arrow IPC batch to miner_processed.
func (m *MinerApp) Run(ctx context.Context) error {
	stream, err := m.db.Collection("connector_raw").FlightStream(ctx)
	if err != nil {
		return fmt.Errorf("open connector_raw flight stream: %w", err)
	}

	for mdDoc := range stream {
		// mdDoc.RawData is the decrypted Markdown text written by 0_DATA_CONNECTOR.
		cleaned, err := m.cleaner.CleanMarkdown(mdDoc.RawData)
		if err != nil {
			log.Printf("cleaner: skip %s: %v", mdDoc.ID, err)
			continue
		}

		records, err := m.normalizer.Process(cleaned)
		if err != nil {
			log.Printf("normalizer: skip %s: %v", mdDoc.ID, err)
			continue
		}

		// Write the batch of SecurityRecords as a single .arrow IPC file.
		if err := m.writer.WriteBatch(mdDoc.ID, records); err != nil {
			log.Printf("arrow writer: %v", err)
		}
	}
	return nil
}
