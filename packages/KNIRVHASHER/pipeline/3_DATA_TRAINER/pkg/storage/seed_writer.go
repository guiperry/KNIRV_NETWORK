package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// JSONSeedWriter handles writing best seeds back to JSON with precise ASIC slot matching
type JSONSeedWriter struct {
	dataPath      string
	mu            sync.Mutex
	pendingWrites map[string]map[string][]byte // sourceFile -> (compositeKey -> best_seed)
	cachedRecords map[string][]map[string]interface{}
}

// NewJSONSeedWriter creates a new JSONSeedWriter
func NewJSONSeedWriter(dataPath string) *JSONSeedWriter {
	return &JSONSeedWriter{
		dataPath:      dataPath,
		pendingWrites: make(map[string]map[string][]byte),
		cachedRecords: make(map[string][]map[string]interface{}),
	}
}

// asicKey creates a unique key based on 12 ASIC slots and target token ID.
// Shared by JSONSeedWriter, ArrowSeedWriter, and the seed ledger so all three
// agree on how a record is identified.
func asicKey(slots [12]uint32, targetTokenID int32) string {
	return fmt.Sprintf("%d:%d:%d:%d:%d:%d:%d:%d:%d:%d:%d:%d:%d",
		slots[0], slots[1], slots[2], slots[3],
		slots[4], slots[5], slots[6], slots[7],
		slots[8], slots[9], slots[10], slots[11],
		targetTokenID)
}

// generateAsicKey creates a unique key based on 12 ASIC slots and target token ID
func (sw *JSONSeedWriter) generateAsicKey(slots [12]uint32, targetTokenID int32) string {
	return asicKey(slots, targetTokenID)
}

// ledgerSeedsByKey reads the append-only seed_writes.jsonl ledger and returns
// the best_seed bytes seen for each composite ASIC+token key. The ledger is
// written for every winning seed ever found (via DualSeedWriter.appendLedger),
// regardless of whether a given batch's JSON/Arrow output file successfully
// matched a record for it, so it is the authoritative history to backfill
// from when (re)building an output file from a fresh source batch.
func ledgerSeedsByKey(dataPath string) map[string][]byte {
	seeds := make(map[string][]byte)

	data, err := os.ReadFile(filepath.Join(dataPath, "frames", "seed_writes.jsonl"))
	if err != nil {
		return seeds
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry SeedWriteLedgerEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		seed, err := base64.StdEncoding.DecodeString(entry.BestSeed)
		if err != nil || len(seed) == 0 {
			continue
		}
		// Later entries for the same key overwrite earlier ones, so the most
		// recently discovered seed for a given record wins.
		seeds[asicKey(entry.AsicSlots, entry.TargetTokenID)] = seed
	}

	return seeds
}

func (sw *JSONSeedWriter) AddSeedWrite(sourceFile string, slots [12]uint32, targetTokenID int32, bestSeed []byte) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if len(bestSeed) == 0 {
		return fmt.Errorf("cannot write empty seed")
	}

	// Normalize sourceFile to its base name
	sourceFile = filepath.Base(sourceFile)
	if sourceFile == "" || sourceFile == "." || sourceFile == "synthetic" {
		sourceFile = "training_frames.json"
	}

	if _, ok := sw.pendingWrites[sourceFile]; !ok {
		sw.pendingWrites[sourceFile] = make(map[string][]byte)
	}

	key := sw.generateAsicKey(slots, targetTokenID)
	fmt.Printf("[DEBUG] JSONSeedWriter: Queuing win for Token %d in %s\n", targetTokenID, sourceFile)
	sw.pendingWrites[sourceFile][key] = bestSeed
	return nil
}

func sourceAliases(sourceFile string) []string {
	base := strings.TrimSuffix(filepath.Base(sourceFile), filepath.Ext(sourceFile))
	base = strings.TrimSuffix(base, "_with_seeds")
	if base == "" || base == "." || base == "synthetic" {
		base = "training_frames"
	}

	aliases := []string{base}
	if base != "training_frames" {
		aliases = append(aliases, "training_frames")
	}

	seen := make(map[string]bool, len(aliases))
	out := aliases[:0]
	for _, alias := range aliases {
		if alias == "" || seen[alias] {
			continue
		}
		seen[alias] = true
		out = append(out, alias)
	}
	return out
}

func jsonSourcePaths(dataPath string, aliases []string) []string {
	var paths []string
	for _, base := range aliases {
		paths = append(paths,
			filepath.Join(dataPath, "frames", base+".json"),
			filepath.Join(dataPath, "json", base+".json"),
			filepath.Join(dataPath, "frames", "archive", base+".json"),
		)
	}
	return paths
}

func outputBaseForSource(sourceFile string) string {
	return sourceAliases(sourceFile)[0]
}

func numericToUint32(v interface{}) (uint32, bool) {
	switch val := v.(type) {
	case float64:
		if math.IsNaN(val) || math.IsInf(val, 0) {
			return 0, false
		}
		if val < 0 {
			return uint32(int32(val)), true
		}
		return uint32(uint64(val)), true
	case int:
		return uint32(int32(val)), true
	case int32:
		return uint32(val), true
	case int64:
		return uint32(int32(val)), true
	case json.Number:
		i, err := val.Int64()
		if err == nil {
			return uint32(int32(i)), true
		}
		f, err := val.Float64()
		if err != nil {
			return 0, false
		}
		return numericToUint32(f)
	default:
		return 0, false
	}
}

// recordKey derives the composite ASIC-slot+target-token key for a JSON
// record, the same way generateAsicKey does for a queued seed write. Returns
// ok=false if the record is missing the slots or target token needed to
// build the key.
func (sw *JSONSeedWriter) recordKey(record map[string]interface{}) (string, bool) {
	var slots [12]uint32

	// Try "feature_vector" array first
	if fv, ok := record["feature_vector"].([]interface{}); ok && len(fv) >= 12 {
		for j := 0; j < 12; j++ {
			val, ok := numericToUint32(fv[j])
			if !ok {
				return "", false
			}
			slots[j] = val
		}
	} else {
		// Fallback to individual slot keys
		for j := 0; j < 12; j++ {
			key := fmt.Sprintf("asic_slot_%d", j)
			valRaw, ok := record[key]
			if !ok {
				key = fmt.Sprintf("AsicSlots%d", j)
				valRaw, ok = record[key]
			}
			if !ok {
				return "", false
			}
			val, ok := numericToUint32(valRaw)
			if !ok {
				return "", false
			}
			slots[j] = val
		}
	}

	var targetTokenID int32
	if tID, ok := record["target_token"].(float64); ok {
		targetTokenID = int32(tID)
	} else if tID, ok := record["target_token_id"].(float64); ok {
		targetTokenID = int32(tID)
	} else if tID, ok := record["TargetTokenID"].(float64); ok {
		targetTokenID = int32(tID)
	} else {
		return "", false
	}

	return sw.generateAsicKey(slots, targetTokenID), true
}

func setBestSeed(record map[string]interface{}, seed []byte) {
	encoded := base64.StdEncoding.EncodeToString(seed)
	if _, exists := record["best_seed"]; exists {
		record["best_seed"] = encoded
	} else if _, exists := record["BestSeed"]; exists {
		record["BestSeed"] = encoded
	} else {
		record["best_seed"] = encoded
	}
}

// WriteBack commits all pending writes to the appropriate output JSON files
func (sw *JSONSeedWriter) WriteBack() error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if len(sw.pendingWrites) == 0 {
		return nil
	}

	for sourceFile, writes := range sw.pendingWrites {
		if len(writes) == 0 {
			continue
		}

		potentialPaths := jsonSourcePaths(sw.dataPath, sourceAliases(sourceFile))

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
		outputFile := filepath.Join(sw.dataPath, "frames", cleanBase+"_with_seeds.json")

		// 1. Initialize cache for this file if empty
		if _, ok := sw.cachedRecords[sourceFile]; !ok {
			// Always base the cache on the current batch's source records. Each
			// pipeline batch mines a brand new training_frames.json with different
			// documents and target tokens; a "_with_seeds.json" left over from an
			// earlier batch must never be used as the base record set, since its
			// records would never match this batch's target tokens and every
			// write would silently no-op forever (the bug this replaced).
			if sourcePath == "" {
				fmt.Printf("[WARN] JSONSeedWriter: No source path found for %s among %v\n", sourceFile, potentialPaths)
				continue
			}
			if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
				fmt.Printf("[WARN] JSONSeedWriter: Source file not found: %s\n", sourcePath)
				continue
			}

			fmt.Printf("[DEBUG] JSONSeedWriter: Initializing cache from %s\n", sourcePath)
			data, err := os.ReadFile(sourcePath)
			if err != nil {
				return fmt.Errorf("failed to read JSON source %s: %w", sourcePath, err)
			}

			var records []map[string]interface{}
			if err := json.Unmarshal(data, &records); err != nil {
				return fmt.Errorf("failed to unmarshal JSON source %s: %w", sourcePath, err)
			}

			// Carry forward any best_seed values already recorded in an existing
			// output file (e.g. data-trainer restarted mid-run on this same
			// batch) by matching on the composite ASIC+token key.
			migrated := 0
			if prevData, err := os.ReadFile(outputFile); err == nil {
				var prevRecords []map[string]interface{}
				if err := json.Unmarshal(prevData, &prevRecords); err == nil {
					prevSeeds := make(map[string]interface{}, len(prevRecords))
					for _, pr := range prevRecords {
						seed, hasSeed := pr["best_seed"]
						if !hasSeed {
							seed, hasSeed = pr["BestSeed"]
						}
						if !hasSeed {
							continue
						}
						if key, ok := sw.recordKey(pr); ok {
							prevSeeds[key] = seed
						}
					}
					for i := range records {
						if key, ok := sw.recordKey(records[i]); ok {
							if seed, ok := prevSeeds[key]; ok {
								records[i]["best_seed"] = seed
								migrated++
							}
						}
					}
				}
			}

			// Also migrate any winning seed ever discovered for these records,
			// even ones that predate this fix and never made it into an output
			// file (the "updated 0 records" bug queued wins into the ledger via
			// DualSeedWriter.appendLedger but the stale-file bug meant they were
			// never actually matched against a record). The ledger is the
			// authoritative, ever-growing history, so replay it here to backfill
			// any record in this batch that already has a known winning seed.
			ledgerSeeds := ledgerSeedsByKey(sw.dataPath)
			for i := range records {
				if _, hasSeed := records[i]["best_seed"]; hasSeed {
					continue
				}
				key, ok := sw.recordKey(records[i])
				if !ok {
					continue
				}
				if seed, ok := ledgerSeeds[key]; ok {
					setBestSeed(records[i], seed)
					migrated++
				}
			}
			if migrated > 0 {
				fmt.Printf("[DEBUG] JSONSeedWriter: Migrated %d previously-known winning seed(s) into %s\n", migrated, filepath.Base(outputFile))
			}

			sw.cachedRecords[sourceFile] = records
		}

		// 2. Update cached records
		records := sw.cachedRecords[sourceFile]
		updated := 0
		for i := range records {
			record := records[i]

			key, ok := sw.recordKey(record)
			if !ok {
				continue
			}

			if seed, ok := writes[key]; ok {
				setBestSeed(record, seed)
				updated++
			}
		}

		// 3. Persist to output file
		newData, err := json.MarshalIndent(records, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal output JSON: %w", err)
		}

		if err := os.WriteFile(outputFile, newData, 0644); err != nil {
			return fmt.Errorf("failed to write output JSON %s: %w", outputFile, err)
		}

		fmt.Printf("Successfully updated %d records in %s (total: %d)\n", updated, filepath.Base(outputFile), len(records))
		if updated == 0 {
			fmt.Printf("[WARN] JSONSeedWriter: no records updated in %s for %d pending seed writes; seed ledger retains wins\n", filepath.Base(outputFile), len(writes))
		}

		// Clear writes for this file
		delete(sw.pendingWrites, sourceFile)
	}

	return nil
}

// DualSeedWriter handles dual persistence to JSON and Arrow
type DualSeedWriter struct {
	jsonWriter  *JSONSeedWriter
	arrowWriter *ArrowSeedWriter
	dataPath    string
}

type SeedWriteLedgerEntry struct {
	Timestamp     time.Time  `json:"timestamp"`
	SourceFile    string     `json:"source_file"`
	TargetTokenID int32      `json:"target_token_id"`
	AsicSlots     [12]uint32 `json:"asic_slots"`
	BestSeed      string     `json:"best_seed"`
	SeedBytes     int        `json:"seed_bytes"`
}

// NewDualSeedWriter creates a new DualSeedWriter
func NewDualSeedWriter(dataPath string) *DualSeedWriter {
	return &DualSeedWriter{
		jsonWriter:  NewJSONSeedWriter(dataPath),
		arrowWriter: NewArrowSeedWriter(dataPath),
		dataPath:    dataPath,
	}
}

// AddSeedWrite queues a win
func (dsw *DualSeedWriter) AddSeedWrite(sourceFile string, slots [12]uint32, targetTokenID int32, bestSeed []byte) error {
	if err := dsw.appendLedger(sourceFile, slots, targetTokenID, bestSeed); err != nil {
		return err
	}
	if err := dsw.jsonWriter.AddSeedWrite(sourceFile, slots, targetTokenID, bestSeed); err != nil {
		return err
	}
	return dsw.arrowWriter.AddSeedWrite(sourceFile, slots, targetTokenID, bestSeed)
}

func (dsw *DualSeedWriter) appendLedger(sourceFile string, slots [12]uint32, targetTokenID int32, bestSeed []byte) error {
	if len(bestSeed) == 0 {
		return fmt.Errorf("cannot ledger empty seed")
	}

	ledgerDir := filepath.Join(dsw.dataPath, "frames")
	if err := os.MkdirAll(ledgerDir, 0755); err != nil {
		return fmt.Errorf("create seed ledger directory: %w", err)
	}
	entry := SeedWriteLedgerEntry{
		Timestamp:     time.Now().UTC(),
		SourceFile:    filepath.Base(sourceFile),
		TargetTokenID: targetTokenID,
		AsicSlots:     slots,
		BestSeed:      base64.StdEncoding.EncodeToString(bestSeed),
		SeedBytes:     len(bestSeed),
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal seed ledger entry: %w", err)
	}
	f, err := os.OpenFile(filepath.Join(ledgerDir, "seed_writes.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open seed ledger: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("append seed ledger: %w", err)
	}
	return nil
}

// WriteBack commits all pending wins
func (dsw *DualSeedWriter) WriteBack() error {
	var failures []string
	if err := dsw.jsonWriter.WriteBack(); err != nil {
		failures = append(failures, err.Error())
	}
	if err := dsw.arrowWriter.WriteBack(); err != nil {
		failures = append(failures, err.Error())
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return nil
}
