package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lab/hasher/data-seeder/pkg/training"
)

// ArrowSeedWriter handles incremental updates to Arrow files
type ArrowSeedWriter struct {
	dataPath      string
	mu            sync.Mutex
	pendingWrites map[string]map[string][]byte
	cachedRecords map[string][]*training.TrainingRecord
}

// NewArrowSeedWriter creates a new ArrowSeedWriter
func NewArrowSeedWriter(dataPath string) *ArrowSeedWriter {
	return &ArrowSeedWriter{
		dataPath:      dataPath,
		pendingWrites: make(map[string]map[string][]byte),
		cachedRecords: make(map[string][]*training.TrainingRecord),
	}
}

// generateAsicKey creates a unique key based on 12 ASIC slots and target token ID
func (aw *ArrowSeedWriter) generateAsicKey(slots [12]uint32, targetTokenID int32) string {
	return fmt.Sprintf("%d:%d:%d:%d:%d:%d:%d:%d:%d:%d:%d:%d:%d",
		slots[0], slots[1], slots[2], slots[3],
		slots[4], slots[5], slots[6], slots[7],
		slots[8], slots[9], slots[10], slots[11],
		targetTokenID)
}

// AddSeedWrite queues a win
func (aw *ArrowSeedWriter) AddSeedWrite(sourceFile string, slots [12]uint32, targetTokenID int32, bestSeed []byte) error {
	aw.mu.Lock()
	defer aw.mu.Unlock()

	if len(bestSeed) == 0 {
		return fmt.Errorf("cannot write empty seed")
	}

	// Normalize sourceFile to its base name
	sourceFile = filepath.Base(sourceFile)
	if sourceFile == "" || sourceFile == "." || sourceFile == "synthetic" {
		sourceFile = "training_frames.arrow"
	}

	if _, ok := aw.pendingWrites[sourceFile]; !ok {
		aw.pendingWrites[sourceFile] = make(map[string][]byte)
	}

	key := aw.generateAsicKey(slots, targetTokenID)
	aw.pendingWrites[sourceFile][key] = bestSeed
	return nil
}

// WriteBack commits all pending wins incrementally
func (aw *ArrowSeedWriter) WriteBack() error {
	aw.mu.Lock()
	defer aw.mu.Unlock()

	if len(aw.pendingWrites) == 0 {
		return nil
	}

	for sourceFile, writes := range aw.pendingWrites {
		if len(writes) == 0 {
			continue
		}

		var potentialPaths []string
		for _, base := range sourceAliases(sourceFile) {
			potentialPaths = append(potentialPaths,
				filepath.Join(aw.dataPath, "frames", base+".arrow"),
				filepath.Join(aw.dataPath, "json", base+".arrow"),
				filepath.Join(aw.dataPath, "frames", "archive", base+".arrow"),
			)
		}

		sourcePath := ""
		for _, p := range potentialPaths {
			if _, err := os.Stat(p); err == nil {
				sourcePath = p
				break
			}
		}
		cleanBase := outputBaseForSource(sourceFile)
		if sourcePath != "" {
			cleanBase = strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
		}
		outputFile := filepath.Join(aw.dataPath, "frames", cleanBase+"_with_seeds.arrow")

		// 1. Initialize cache for this file if empty
		if _, ok := aw.cachedRecords[sourceFile]; !ok {
			// Always base the cache on the current batch's source records. Each
			// pipeline batch mines a brand new training_frames.arrow with different
			// documents and target tokens; a "_with_seeds.arrow" left over from an
			// earlier batch must never be used as the base record set, since its
			// records would never match this batch's target tokens and every
			// write would silently no-op forever (the bug this replaced).
			if sourcePath == "" {
				fmt.Printf("[WARN] ArrowSeedWriter: No source path found for %s among %v\n", sourceFile, potentialPaths)
				continue
			}
			if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
				fmt.Printf("[WARN] ArrowSeedWriter: Source file not found: %s\n", sourcePath)
				continue
			}

			fmt.Printf("[DEBUG] ArrowSeedWriter: Initializing cache from %s\n", sourcePath)
			records, err := ReadTrainingRecordsFromArrowIPC(sourcePath)
			if err != nil {
				return fmt.Errorf("failed to read Arrow source %s: %w", sourcePath, err)
			}
			if len(records) == 0 {
				fmt.Printf("[WARN] ArrowSeedWriter: %s contained no trainable records; seed ledger retains wins\n", sourcePath)
				delete(aw.pendingWrites, sourceFile)
				continue
			}

			// Carry forward any best_seed values already recorded in an existing
			// output file (e.g. data-seeder restarted mid-run on this same
			// batch) by matching on the composite ASIC+token key.
			migrated := 0
			if prevRecords, err := ReadTrainingRecordsFromArrowIPC(outputFile); err == nil {
				prevSeeds := make(map[string][]byte, len(prevRecords))
				for _, pr := range prevRecords {
					if len(pr.BestSeed) > 0 {
						prevSeeds[aw.generateAsicKey(pr.FeatureVector, pr.TargetToken)] = pr.BestSeed
					}
				}
				for _, record := range records {
					if seed, ok := prevSeeds[aw.generateAsicKey(record.FeatureVector, record.TargetToken)]; ok {
						record.BestSeed = seed
						migrated++
					}
				}
			}

			// Also migrate any winning seed ever discovered for these records
			// from the seed ledger, even ones that predate this fix and never
			// made it into an output file because of the stale-file bug this
			// replaced. The ledger is the authoritative, ever-growing history.
			ledgerSeeds := ledgerSeedsByKey(aw.dataPath)
			for _, record := range records {
				if len(record.BestSeed) > 0 {
					continue
				}
				if seed, ok := ledgerSeeds[aw.generateAsicKey(record.FeatureVector, record.TargetToken)]; ok {
					record.BestSeed = seed
					migrated++
				}
			}
			if migrated > 0 {
				fmt.Printf("[DEBUG] ArrowSeedWriter: Migrated %d previously-known winning seed(s) into %s\n", migrated, filepath.Base(outputFile))
			}

			aw.cachedRecords[sourceFile] = records
		}

		// 2. Update records
		records := aw.cachedRecords[sourceFile]
		updated := 0
		for _, record := range records {
			key := aw.generateAsicKey(record.FeatureVector, record.TargetToken)

			if seed, ok := writes[key]; ok {
				record.BestSeed = seed
				updated++
			}
		}

		// 3. Write back
		if err := WriteTrainingRecordsToArrowIPC(outputFile, records); err != nil {
			return fmt.Errorf("failed to write output Arrow file %s: %w", outputFile, err)
		}

		fmt.Printf("Successfully updated %d records in %s (total: %d)\n", updated, filepath.Base(outputFile), len(records))
		if updated == 0 {
			fmt.Printf("[WARN] ArrowSeedWriter: no records updated in %s for %d pending seed writes; seed ledger retains wins\n", filepath.Base(outputFile), len(writes))
		}

		// Clear writes for this file
		delete(aw.pendingWrites, sourceFile)
	}

	return nil
}
