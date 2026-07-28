package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"data-encoder/internal"
	"data-encoder/pkg/analyzer"
	"data-encoder/pkg/checkpoint"
	"data-encoder/pkg/embeddings"
	"data-encoder/pkg/mapper"
	"data-encoder/pkg/nrvio"
	"data-encoder/pkg/schema"
	"data-encoder/pkg/sliding"
	"data-encoder/pkg/tokenizer"
)

// Config holds the application configuration
type Config struct {
	InputFile  string
	OutputFile string
	MapperSeed int64
	TokenTest  bool // Run token test and exit

	// Worker configuration
	NumWorkers int // Default: 4 workers

	// NEW: Sliding window configuration
	WindowSize   int // Default: 128 tokens
	WindowStride int // Default: 1 (no overlap)
	BatchSize    int // Default: 32 contexts per API call

	// NEW: Checkpoint and quota configuration
	QuotaLimit       int  // Maximum embeddings quota (default: 5000)
	EnableCheckpoint bool // Enable checkpoint/resume functionality
}

type varianceRecord struct {
	FileName    string      `json:"file_name"`
	ChunkID     int         `json:"chunk_id"`
	Content     string      `json:"content"`
	Text        string      `json:"text"`
	Instruction string      `json:"instruction"`
	Input       string      `json:"input"`
	Output      string      `json:"output"`
	Embedding   interface{} `json:"embedding"`
}

func main() {
	// Parse command line flags
	config := parseFlags()

	// Token test mode - run tokenization test and exit
	if config.TokenTest {
		if err := internal.GetTokens(); err != nil {
			log.Fatalf("Token test failed: %v", err)
		}
		return
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Run the encoder
	if err := runEncoder(config); err != nil {
		log.Fatalf("Encoding failed: %v", err)
	}

	log.Println("✅ Encoding Complete. Ready for Evo-GRPO.")
}

// getAppDataDir returns the application data directory.
// Checks writability by creating the frames subdirectory so downstream
// code doesn't fail with permission denied on root-owned directories.
func getAppDataDir() string {
	candidates := []string{}
	if appData := os.Getenv("KNIRV_APP_DATA_DIR"); appData != "" {
		candidates = append(candidates, filepath.Join(appData, "knirvhasher", "data"))
	}
	candidates = append(candidates, "/var/lib/knirvserver/knirvhasher/data")
	homeDir, err := os.UserHomeDir()
	if err == nil {
		candidates = append(candidates,
			filepath.Join(homeDir, ".local", "share", "knirvserver", "knirvhasher", "data"),
			filepath.Join(homeDir, ".local", "share", "knirvhasher", "data"),
		)
	}

	for _, dir := range candidates {
		if err := os.MkdirAll(dir, 0755); err != nil {
			continue
		}
		// Verify writability by creating the frames subdirectory that
		// will be needed later for output.
		if err := os.MkdirAll(filepath.Join(dir, "frames"), 0755); err != nil {
			continue
		}
		return dir
	}
	return "."
}

func parseFlags() *Config {
	config := &Config{}

	// Use the same app data dir as data-mapper so paths are consistent
	appDataDir := getAppDataDir()
	defaultJSONInput := filepath.Join(appDataDir, "json", "ai_knowledge_base.json")
	defaultOutput := filepath.Join(appDataDir, "frames", "training_frames.json")

	flag.StringVar(&config.InputFile, "input", defaultJSONInput, "Input JSON file path")
	flag.StringVar(&config.OutputFile, "output", defaultOutput, "Output JSON file path")
	flag.Int64Var(&config.MapperSeed, "seed", 1337, "Random seed for mapper (for reproducibility)")
	flag.IntVar(&config.NumWorkers, "workers", 4, "Number of worker goroutines")

	// NEW: Sliding window configuration
	flag.IntVar(&config.WindowSize, "window-size", 128, "Sliding window size in tokens")
	flag.IntVar(&config.WindowStride, "window-stride", 1, "Stride between sliding windows")
	flag.IntVar(&config.BatchSize, "batch-size", 32, "Batch size for embedding API calls")

	// NEW: Quota and checkpoint configuration
	// Quota is now disabled by default (0 = unlimited) since the encoder uses
	// deterministic embeddings and no longer consumes a web/Cloudflare daily
	// embedding quota.
	flag.IntVar(&config.QuotaLimit, "quota", 0, "Maximum embeddings quota per run (0 = unlimited)")
	flag.BoolVar(&config.EnableCheckpoint, "checkpoint", true, "Enable checkpoint/resume functionality")

	// Token test mode
	flag.BoolVar(&config.TokenTest, "tokens", false, "Run tokenization test and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Data Encoder - Transform embeddings into hardware-ready Neural Frames\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nDefault input file:\n")
		fmt.Fprintf(os.Stderr, "  %s (JSON)\n", defaultJSONInput)
		fmt.Fprintf(os.Stderr, "\nDefault output file:\n")
		fmt.Fprintf(os.Stderr, "  %s (JSON)\n", defaultOutput)
	}

	flag.Parse()

	return config
}

// replaceFileExtension replaces the file extension in a path
func replaceFileExtension(path, newExt string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + newExt
}

func validateConfig(config *Config) error {
	if strings.TrimSpace(config.InputFile) == "" {
		return fmt.Errorf("input file is required")
	}

	// Only auto-detect for the default placeholder. A user-provided missing
	// path should fail loudly instead of silently training on stale live data.
	if isDefaultInput(config.InputFile) {
		detectedFile := AutoDetectInputFile()
		if detectedFile != "" {
			config.InputFile = detectedFile
			log.Printf("🔍 Auto-detected input file: %s", config.InputFile)
		}
	}

	// Detect file type
	fileType := detectFileType(config.InputFile)
	log.Printf("📁 Input file type: %s (%s)", filepath.Base(config.InputFile), fileType)

	// Check if input file exists after detection attempt
	if _, err := os.Stat(config.InputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", config.InputFile)
	}

	// Ensure application data directory exists
	appDataDir := getAppDataDir()
	if err := os.MkdirAll(appDataDir, 0755); err != nil {
		return fmt.Errorf("failed to create application data directory: %w", err)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(config.OutputFile)
	if outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	return nil
}

// isDefaultInput checks if the provided path is one of our default placeholder paths
func isDefaultInput(path string) bool {
	base := filepath.Base(path)
	return base == "ai_knowledge_base.json"
}

// AutoDetectInputFile searches for available knowledge base files in priority order:
// 1. Goat Alpaca (from GOAT dataset)
// 2. Standard Alpaca (from PDF neural processing)
// 3. Generic Knowledge Base
//
// It checks both the app data directory (from getAppDataDir) and ~/.local/share/hasher
// to handle the case where the data-mapper writes to the server's data dir.
func AutoDetectInputFile() string {
	priorityFiles := []string{
		"mapper/mined_records.json",
		"ai_knowledge_base_goat_alpaca.json",
		"ai_knowledge_base_alpaca.json",
		"ai_knowledge_base.json",
		"ai_knowledge_base_goat_alpaca.arrow",
		"ai_knowledge_base_alpaca.arrow",
		"ai_knowledge_base.arrow",
	}

	// Search paths: app data directory first (where data-mapper writes), then home dir
	searchDirs := []string{
		getAppDataDir(),
		filepath.Join(getAppDataDir(), "json"),
		filepath.Join(getAppDataDir(), "..", "data", "json"),
	}
	if homeDir, err := os.UserHomeDir(); err == nil {
		searchDirs = append(searchDirs,
			filepath.Join(homeDir, ".local", "share", "knirvhasher", "connector"),
			filepath.Join(homeDir, ".local", "share", "knirvhasher", "mapper"),
			filepath.Join(homeDir, ".local", "share", "knirvhasher", "data", "json"),
		)
	}

	for _, dir := range searchDirs {
		for _, fileName := range priorityFiles {
			fullPath := filepath.Join(dir, fileName)
			if _, err := os.Stat(fullPath); err == nil {
				return fullPath
			}
		}
	}

	return ""
}

// detectFileType determines if a file is JSON, JSONL, or Arrow based on extension
func detectFileType(filePath string) string {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".arrow" {
		return "arrow"
	}
	if ext == ".json" || ext == ".jsonl" {
		return "json"
	}

	return "json" // Default to JSON
}

// performVarianceAnalysis samples real BGE-compatible embeddings to identify
// high-variance dimensions. It prefers embeddings already present in the mined
// data and falls back to the configured embedding service for text-only records.
func performVarianceAnalysis(inputFile string, ew embeddings.EmbeddingService) ([]int, error) {
	va := analyzer.NewVarianceAnalyzer()
	maxSamples := 1000
	sampleCount := 0

	// Read JSON file
	content, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON for variance analysis: %w", err)
	}

	contentStr := string(content)
	trimmed := strings.TrimSpace(contentStr)
	isJSONArray := strings.HasPrefix(trimmed, "[")

	if isJSONArray {
		// Process JSON array
		var records []varianceRecord
		if err := json.Unmarshal([]byte(contentStr), &records); err != nil {
			return nil, fmt.Errorf("failed to parse JSON array for variance analysis: %w", err)
		}

		for _, record := range records {
			if sampleCount >= maxSamples {
				break
			}
			if sampleRecord(record, va, ew) {
				sampleCount++
			}
		}
	} else {
		// Process JSONL format
		scanner := bufio.NewScanner(strings.NewReader(contentStr))
		scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
		for scanner.Scan() && sampleCount < maxSamples {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var record varianceRecord
			if err := json.Unmarshal([]byte(line), &record); err == nil {
				if sampleRecord(record, va, ew) {
					sampleCount++
				}
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed reading JSONL for variance analysis: %w", err)
		}
	}

	if sampleCount == 0 {
		return nil, fmt.Errorf("no usable text or embeddings found for variance analysis")
	}

	// Calculate variance and get indices
	if err := va.Calculate(); err != nil {
		return nil, fmt.Errorf("variance calculation failed after %d samples: %w", sampleCount, err)
	}

	log.Printf("📊 Variance analysis: sampled %d real embeddings", sampleCount)
	if stats, _ := va.GetStats(); stats != nil {
		log.Printf("📊 Top variance: %.6f, Mean variance: %.6f", stats.TopVariance, stats.MeanVariance)
	}

	if err := saveVarianceAnalysis(inputFile, va); err != nil {
		log.Printf("⚠️  Failed to persist variance analysis: %v", err)
	}

	return va.GetSignalIndices(), nil
}

// sampleRecord creates a variance sample from a record using real embeddings.
func sampleRecord(record varianceRecord, va *analyzer.VarianceAnalyzer, ew embeddings.EmbeddingService) bool {
	if record.Embedding != nil {
		if err := va.SampleFromRecord(record.Embedding); err == nil {
			return true
		}
	}

	text := varianceSampleText(record)
	if text == "" || ew == nil {
		return false
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		embedding, err := ew.GetEmbedding(text)
		if err == nil {
			if err := va.Sample(embedding); err == nil {
				return true
			} else {
				lastErr = err
			}
		} else {
			lastErr = err
		}
		time.Sleep(time.Duration(attempt*5) * time.Millisecond)
	}

	log.Printf("⚠️  Variance sample embedding failed for %s#%d after retries: %v", record.FileName, record.ChunkID, lastErr)
	return false
}

func varianceSampleText(record varianceRecord) string {
	parts := make([]string, 0, 4)
	for _, part := range []string{record.Content, record.Text, record.Instruction, record.Input, record.Output} {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func saveVarianceAnalysis(inputFile string, va *analyzer.VarianceAnalyzer) error {
	outputDir := filepath.Dir(inputFile)
	appDataDir := getAppDataDir()
	if appDataDir != "" {
		framesDir := filepath.Join(appDataDir, "frames")
		if isWritableDir(framesDir) {
			outputDir = framesDir
		}
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	outputPath := filepath.Join(outputDir, analyzer.VarianceOutputFile)
	if err := va.SaveToFile(outputPath, embeddings.CloudflareWorkersBGE); err != nil {
		return err
	}

	indices := append([]int(nil), va.GetSignalIndices()...)
	sort.Ints(indices)
	log.Printf("📊 Saved variance signal indices to %s (sorted=%v)", outputPath, indices)
	return nil
}

func isWritableDir(dir string) bool {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false
	}
	testFile, err := os.CreateTemp(dir, ".variance-write-test-*")
	if err != nil {
		return false
	}
	name := testFile.Name()
	_ = testFile.Close()
	_ = os.Remove(name)
	return true
}

// getDefaultVarianceIndices returns default variance indices
func getDefaultVarianceIndices() []int {
	return spreadVarianceIndices()
}

func runEncoder(config *Config) error {
	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Handle signals in a goroutine
	go func() {
		sig := <-sigChan
		log.Printf("\n🛑 Received signal: %v. Initiating graceful shutdown...", sig)
		cancel()
	}()

	if config.QuotaLimit <= 0 {
		log.Printf("🔧 Initializing Data Encoder (seed=%d, window=%d, stride=%d, batch=%d, quota=unlimited)...",
			config.MapperSeed, config.WindowSize, config.WindowStride, config.BatchSize)
	} else {
		log.Printf("🔧 Initializing Data Encoder (seed=%d, window=%d, stride=%d, batch=%d, quota=%d)...",
			config.MapperSeed, config.WindowSize, config.WindowStride, config.BatchSize, config.QuotaLimit)
	}

	// 0. Initialize Checkpoint Manager (for resume capability and quota tracking)
	var cpManager *checkpoint.Manager
	if config.EnableCheckpoint {
		var err error
		cpManager, err = checkpoint.NewManager(config.OutputFile)
		if err != nil {
			log.Printf("⚠️  Failed to initialize checkpoint manager: %v", err)
			// Continue without checkpointing
			cpManager = nil
		} else {
			// Set quota limit from config
			cpManager.SetQuotaLimit(config.QuotaLimit)
			// Check if quota should be reset (new day)
			cpManager.ResetDailyQuota()
			log.Printf("✓ Checkpoint manager initialized: %s", cpManager.GetCheckpointPath())
		}
	}

	// 1. Initialize Services
	tk, err := tokenizer.New()
	if err != nil {
		return fmt.Errorf("failed to initialize tokenizer: %w", err)
	}

	var ew embeddings.EmbeddingService = embeddings.NewWithBatchSize(config.BatchSize)

	log.Printf("✓ Embeddings service initialized with batch size %d", config.BatchSize)
	log.Printf("✓ Sliding window: size=%d, stride=%d", config.WindowSize, config.WindowStride)

	// 1.5 Variance Analysis - Identify high-signal dimensions for BGE embeddings
	log.Printf("📊 Performing variance analysis on BGE embeddings...")
	varianceIndices, err := performVarianceAnalysis(config.InputFile, ew)
	if err != nil {
		log.Printf("⚠️  Variance analysis failed: %v", err)
		varianceIndices = varianceFallbackIndices(config.InputFile)
		log.Printf("⚠️  Continuing with cached/fallback signal indices: %v", varianceIndices)
	} else {
		log.Printf("✅ Variance analysis complete: using top 24 high-signal dimensions: %v", varianceIndices)
	}

	// Create TensorPacker (replaces old mapper and varianceMapper)
	tp := mapper.NewTensorPacker(varianceIndices)
	log.Printf("✓ TensorPacker initialized with %d signal indices", len(varianceIndices))

	// Check initial quota status
	if cpManager != nil {
		used, limit, _ := cpManager.GetQuotaStatus()
		if limit <= 0 {
			log.Printf("📊 Quota: unlimited (used: %d)", used)
		} else {
			_, _, remaining := cpManager.GetQuotaStatus()
			log.Printf("📊 Quota status: %d/%d used, %d remaining", used, limit, remaining)
		}

		if !cpManager.HasQuotaAvailable() {
			log.Printf("⚠️  Quota already exhausted (%d/%d). Nothing to process.", used, limit)
			return nil
		}
	}

	log.Printf("💾 Output will be saved to: %s", config.OutputFile)

	// Create JSON array to collect training frames
	var frames []schema.TrainingFrame
	var mu sync.Mutex

	processInput := func(cp *checkpoint.Manager) (int64, error) {
		fileType := detectFileType(config.InputFile)
		if fileType == "arrow" {
			return processArrowFile(config.InputFile, tp, tk, ew, &frames, config, cp)
		}
		return processJSONFile(config.InputFile, tp, tk, ew, &frames, config, cp)
	}

	// Use WaitGroup to coordinate processing and cleanup
	var wg sync.WaitGroup
	var frameCount int64
	var processErr error

	// Start processing in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()

		var err error
		frameCount, err = processInput(cpManager)
		if err != nil {
			mu.Lock()
			processErr = fmt.Errorf("failed to process file: %w", err)
			mu.Unlock()
			return
		}
	}()

	// Wait for either processing to complete or context to be cancelled
	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		// Processing completed normally
	case <-ctx.Done():
		// Signal received, wait for goroutine to finish
		log.Printf("⏳ Waiting for processing to stop...")
		<-waitChan
	}

	// Save final checkpoint before cleanup
	if cpManager != nil {
		if err := cpManager.Save(); err != nil {
			log.Printf("⚠️  Failed to save final checkpoint: %v", err)
		}

		// Check if quota was exhausted
		if !cpManager.HasQuotaAvailable() {
			log.Printf("🛑 Processing stopped: quota exhausted")
		} else if ctx.Err() != nil {
			log.Printf("🛑 Processing stopped: interrupted by user")
		} else {
			log.Printf("✅ Processing completed - checkpoint saved for resume")
		}
	}

	mu.Lock()
	err = processErr
	mu.Unlock()
	if err != nil {
		return err
	}

	if len(frames) == 0 && frameCount == 0 && cpManager != nil && ctx.Err() == nil {
		log.Printf("⚠️  Checkpoint resume produced 0 frames; rerunning encoder without checkpoint to rebuild output artifacts")
		frames = nil
		frameCount, err = processInput(nil)
		if err != nil {
			return fmt.Errorf("failed to rebuild encoder output without checkpoint: %w", err)
		}
	}

	if len(frames) == 0 {
		return fmt.Errorf("encoder generated 0 training frames from %s; refusing to overwrite %s with empty output", config.InputFile, config.OutputFile)
	}

	// Write JSON output
	log.Printf("💾 Writing JSON output...")
	if err := writeJSONOutput(config.OutputFile, frames); err != nil {
		return fmt.Errorf("failed to write JSON output: %w", err)
	}
	log.Printf("💾 JSON output written successfully")

	// Write Arrow IPC stream output
	arrowPath := replaceFileExtension(config.OutputFile, ".arrow")
	log.Printf("💾 Writing Arrow IPC stream output...")
	if err := schema.WriteTrainingFramesToArrowIPC(arrowPath, frames); err != nil {
		return fmt.Errorf("failed to write Arrow IPC stream output: %w", err)
	}
	log.Printf("💾 Arrow IPC stream output written successfully: %s", arrowPath)
	nrvPath := replaceFileExtension(config.OutputFile, ".nrv")
	if err := writeNRV(nrvPath, frames); err != nil {
		return fmt.Errorf("failed to write NRV output: %w", err)
	}
	log.Printf("💾 NRV output written successfully: %s", nrvPath)

	// Check for processing errors
	mu.Lock()
	err = processErr
	mu.Unlock()
	if err != nil {
		return err
	}

	// Verify the output file exists and has content
	fileInfo, err := os.Stat(config.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to stat output file: %w", err)
	}
	log.Printf("✅ Output file created: %s (%d bytes)", config.OutputFile, fileInfo.Size())

	log.Printf("📈 Total: %d training frames generated", frameCount)
	return nil
}

func writeNRV(path string, frames []schema.TrainingFrame) error {
	w, err := nrvio.NewWriter(path)
	if err != nil {
		return err
	}
	ticker := nrvio.NewFrameTicker(w, time.Second)
	for _, frame := range frames {
		slots := frame.GetAsicSlots()
		var b nrvio.Bracket
		copy(b.Projections[:], internal.SlotsToProjections(slots[:4]))
		b.POSTag = uint8(slots[4])
		b.DepHead = uint8(slots[5])
		b.IntentFlags = uint8(slots[9])
		b.DomainSig = uint16(slots[10])
		b.GoldenSeed = uint32(frame.TargetTokenID)
		b.LSHSalt = uint32(slots[11])
		for i := 0; i < 3; i++ {
			binary.LittleEndian.PutUint32(b.Memory[i*4:], slots[6+i])
		}
		b.SubSecondUS = uint32(frame.WindowStart)
		if err := ticker.Append(b); err != nil {
			return err
		}
	}
	if err := ticker.Close(); err != nil {
		return err
	}
	return w.Close()
}

func varianceFallbackIndices(inputFile string) []int {
	candidateDirs := []string{filepath.Dir(inputFile)}
	if appDataDir := getAppDataDir(); appDataDir != "" {
		candidateDirs = append([]string{filepath.Join(appDataDir, "frames")}, candidateDirs...)
	}

	for _, dir := range candidateDirs {
		varianceFile := filepath.Join(dir, analyzer.VarianceOutputFile)
		if _, err := os.Stat(varianceFile); err == nil {
			indices, err := analyzer.GetOrCreateSignalIndices(dir)
			if err == nil && len(indices) == 24 {
				return indices
			}
		}
	}

	return spreadVarianceIndices()
}

func spreadVarianceIndices() []int {
	indices := make([]int, 24)
	for i := range indices {
		indices[i] = i * 32
	}
	return indices
}

// processJSONFallback handles JSON processing
func processJSONFallback(filePath string, tp *mapper.TensorPacker, tk *tokenizer.Service, ew *embeddings.Service, frames *[]schema.TrainingFrame, config *Config, cpManager *checkpoint.Manager) (int64, error) {
	log.Printf("🔄 Attempting JSON processing from: %s", filePath)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read JSON file: %w", err)
	}

	contentStr := string(content)
	trimmed := strings.TrimSpace(contentStr)
	isJSONArray := strings.HasPrefix(trimmed, "[")

	if isJSONArray {
		fixedContent, needsFix := fixJSONContent(contentStr)
		if needsFix {
			log.Printf("🔧 Fixed JSON array structure")
			if writeErr := os.WriteFile(filePath, []byte(fixedContent), 0644); writeErr != nil {
				log.Printf("⚠️  Failed to update fixed file: %v", writeErr)
			}
			contentStr = fixedContent
		}

		var records []schema.MinedRecord
		if err := json.Unmarshal([]byte(contentStr), &records); err != nil {
			return 0, fmt.Errorf("failed to parse JSON array: %w", err)
		}

		log.Printf("📋 Detected JSON array format with %d records", len(records))
		return processMinedRecords(&records, tp, tk, ew, frames, config, cpManager)
	} else {
		log.Printf("📋 Processing as JSONL format")
		return processJSONLRecords(contentStr, tp, tk, ew, frames, config, cpManager)
	}
}

// processJSONFile processes JSON files with proper error handling
func processJSONFile(filePath string, tp *mapper.TensorPacker, tk *tokenizer.Service, ew embeddings.EmbeddingService, frames *[]schema.TrainingFrame, config *Config, cpManager *checkpoint.Manager) (int64, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read input file: %w", err)
	}

	contentStr := string(content)
	trimmed := strings.TrimSpace(contentStr)
	isJSONArray := strings.HasPrefix(trimmed, "[")

	if isJSONArray {
		fixedContent, needsFix := fixJSONContent(contentStr)
		if needsFix {
			log.Printf("🔧 Fixed JSON array structure")
			if writeErr := os.WriteFile(filePath, []byte(fixedContent), 0644); writeErr != nil {
				log.Printf("⚠️  Failed to update fixed file: %v", writeErr)
			}
			contentStr = fixedContent
		}

		var records []schema.MinedRecord
		if err := json.Unmarshal([]byte(contentStr), &records); err != nil {
			return 0, fmt.Errorf("failed to parse JSON array: %w", err)
		}

		log.Printf("📋 Detected JSON array format with %d records", len(records))
		return processMinedRecords(&records, tp, tk, ew, frames, config, cpManager)
	} else {
		log.Printf("📋 Processing as JSONL format")
		return processJSONLRecords(contentStr, tp, tk, ew, frames, config, cpManager)
	}
}

// processDocumentRecords processes DocumentRecord chunks
func processDocumentRecords(records []schema.DocumentRecord, tp *mapper.TensorPacker, tk *tokenizer.Service, ew embeddings.EmbeddingService, frames *[]schema.TrainingFrame, config *Config, cpManager *checkpoint.Manager) (int64, error) {
	var frameCount int64
	var recordCount int64
	log.Printf("[BATCH] Starting to process %d DocumentRecords...", len(records))

	for i := range records {
		record := &records[i]

		// Check if we should skip this record (already processed)
		if cpManager != nil && cpManager.ShouldSkipRecord(record.FileName, record.ChunkID, 0) {
			log.Printf("[BATCH] Skipping already processed record %d/%d: %s (chunk %d)",
				i+1, len(records), record.FileName, record.ChunkID)
			recordCount++
			continue
		}

		log.Printf("[BATCH] Processing record %d/%d: %s (chunk %d, content: %d chars)",
			i+1, len(records), record.FileName, record.ChunkID, len(record.Content))

		// Convert DocumentRecord to MinedRecord format for processing
		minedRecord := schema.MinedRecord{
			FileName:     record.FileName,
			ChunkID:      int(record.ChunkID),
			Heading:      record.Heading,
			Content:      record.Content,
			Instruction:  record.Instruction,
			Input:        record.Input,
			Output:       record.Output,
			Tokens:       record.Tokens,
			TokenOffsets: record.TokenOffsets,
			POSTags:      record.POSTags,
			Tenses:       record.Tenses,
			DepHashes:    record.DepHashes,
		}

		if err := processSingleRecordWithSlidingWindow(&minedRecord, &frameCount, tp, tk, ew, frames, config, cpManager); err != nil {
			return 0, fmt.Errorf("failed to process DocumentRecord %d: %w", i, err)
		}

		recordCount++
		log.Printf("[BATCH] Completed record %d/%d, total frames: %d", i+1, len(records), frameCount)

		if cpManager != nil {
			cpManager.MarkRecordProcessed(record.FileName, record.ChunkID, 0)
		}

		// Update checkpoint every 10 records
		if cpManager != nil && i%10 == 0 {
			cpManager.UpdateProgress(record.FileName, record.ChunkID, 0)
			cpManager.IncrementStats(1, frameCount-cpManager.GetCheckpoint().FramesGenerated)
			if err := cpManager.Save(); err != nil {
				log.Printf("⚠️  Failed to save checkpoint: %v", err)
			}
		}

		// Progress logging every 10 records (more frequent)
		if i%10 == 0 {
			log.Printf("📊 Processed %d/%d DocumentRecords, generated %d frames...", i, len(records), frameCount)
		}
	}

	log.Printf("📈 Total: %d DocumentRecords processed, %d training frames generated", recordCount, frameCount)
	return frameCount, nil
}

// processMinedRecords processes MinedRecord chunks from JSON array
func processMinedRecords(records *[]schema.MinedRecord, tp *mapper.TensorPacker, tk *tokenizer.Service, ew embeddings.EmbeddingService, frames *[]schema.TrainingFrame, config *Config, cpManager *checkpoint.Manager) (int64, error) {
	var frameCount int64
	var processedCount int64

	for i := range *records {
		record := &(*records)[i]

		// Check if we should skip this record
		if cpManager != nil && cpManager.ShouldSkipRecord(record.FileName, int32(record.ChunkID), 0) {
			log.Printf("[BATCH] Skipping already processed record %d/%d: %s", i+1, len(*records), record.FileName)
			processedCount++
			continue
		}

		if err := processSingleRecordWithSlidingWindow(record, &frameCount, tp, tk, ew, frames, config, cpManager); err != nil {
			return 0, fmt.Errorf("failed to process record %d: %w", i, err)
		}

		processedCount++

		if cpManager != nil {
			cpManager.MarkRecordProcessed(record.FileName, int32(record.ChunkID), 0)
		}

		// Update checkpoint every 10 records
		if cpManager != nil && i%10 == 0 {
			cpManager.UpdateProgress(record.FileName, int32(record.ChunkID), 0)
			cpManager.IncrementStats(1, frameCount-cpManager.GetCheckpoint().FramesGenerated)
			if err := cpManager.Save(); err != nil {
				log.Printf("⚠️  Failed to save checkpoint: %v", err)
			}
		}

		// Progress logging every 100 records
		if i%100 == 0 {
			log.Printf("📊 Processed %d records, generated %d frames...", i, frameCount)
		}
	}

	log.Printf("📈 Total: %d records processed, %d training frames generated", processedCount, frameCount)
	return frameCount, nil
}

// processJSONLRecords processes JSONL format records
func processJSONLRecords(content string, tp *mapper.TensorPacker, tk *tokenizer.Service, ew embeddings.EmbeddingService, frames *[]schema.TrainingFrame, config *Config, cpManager *checkpoint.Manager) (int64, error) {
	var recordCount, frameCount, fixedCount int64
	scanner := bufio.NewScanner(strings.NewReader(content))
	var updatedLines []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Try to parse the JSON record
		var record schema.MinedRecord
		err := json.Unmarshal([]byte(line), &record)

		if err != nil {
			// Try to fix the JSON
			fixedJSON := fixJSONString(line)
			if parseErr := json.Unmarshal([]byte(fixedJSON), &record); parseErr != nil {
				if recordCount%10 == 0 || recordCount < 10 {
					log.Printf("⚠️  Skipping unfixable record %d: %v", recordCount, err)
				}
				continue
			}

			// Use the fixed version for file update
			updatedLines = append(updatedLines, fixedJSON)
			fixedCount++
			if fixedCount <= 10 {
				log.Printf("🔧 Fixed record %d", recordCount)
			}
		} else {
			updatedLines = append(updatedLines, line)
		}

		recordCount++

		// Check if we should skip this record
		if cpManager != nil && cpManager.ShouldSkipRecord(record.FileName, int32(record.ChunkID), 0) {
			log.Printf("[BATCH] Skipping already processed record %d: %s", recordCount, record.FileName)
			continue
		}

		// Process the record with sliding windows
		if err := processSingleRecordWithSlidingWindow(&record, &frameCount, tp, tk, ew, frames, config, cpManager); err != nil {
			return 0, fmt.Errorf("failed to process record %d: %w", recordCount, err)
		}

		if cpManager != nil {
			cpManager.MarkRecordProcessed(record.FileName, int32(record.ChunkID), 0)
		}

		// Update checkpoint every 10 records
		if cpManager != nil && recordCount%10 == 0 {
			cpManager.UpdateProgress(record.FileName, int32(record.ChunkID), 0)
			cpManager.IncrementStats(1, frameCount-cpManager.GetCheckpoint().FramesGenerated)
			if err := cpManager.Save(); err != nil {
				log.Printf("⚠️  Failed to save checkpoint: %v", err)
			}
		}

		// Progress logging every 100 records
		if recordCount%100 == 0 {
			log.Printf("📊 Processed %d records, generated %d frames (fixed: %d)...", recordCount, frameCount, fixedCount)
		}
	}

	// Update file with fixed content if any records were fixed
	if fixedCount > 0 {
		fixedContent := strings.Join(updatedLines, "\n") + "\n"
		if writeErr := os.WriteFile(config.InputFile, []byte(fixedContent), 0644); writeErr != nil {
			log.Printf("⚠️  Failed to update fixed file: %v", writeErr)
		} else {
			log.Printf("🔧 Updated file with %d fixed records", fixedCount)
		}
	}

	log.Printf("📈 Total: %d records processed, %d training frames generated", recordCount, frameCount)
	if fixedCount > 0 {
		log.Printf("🔧 JSON issues fixed and updated: %d records", fixedCount)
	}

	return frameCount, nil
}

func fixJSONContent(content string) (string, bool) {
	original := content
	fixed := fixJSONString(content)

	// Check if JSON array is properly closed
	trimmed := strings.TrimSpace(fixed)
	if strings.HasPrefix(trimmed, "[") && !strings.HasSuffix(trimmed, "]") {
		// Add missing closing bracket
		fixed = fixed + "\n]"
	}

	return fixed, original != fixed
}

func processSingleRecordWithSlidingWindow(record *schema.MinedRecord, frameCount *int64, tp *mapper.TensorPacker, tk *tokenizer.Service, ew embeddings.EmbeddingService, frames *[]schema.TrainingFrame, config *Config, cpManager *checkpoint.Manager) error {
	log.Printf("[PROCESS] Starting record %d from %s (content length: %d chars)", record.ChunkID, record.FileName, len(record.Content))

	// Check quota availability before processing
	if cpManager != nil && !cpManager.HasQuotaAvailable() {
		log.Printf("🛑 Quota exhausted (%d/%d embeddings used). Stopping processing.",
			cpManager.GetCheckpoint().QuotaUsed, cpManager.GetCheckpoint().QuotaLimit)
		return fmt.Errorf("quota exhausted")
	}

	// 1. Tokenize the entire content once
	log.Printf("[PROCESS] Step 1/6: Tokenizing content...")
	allTokens := tk.Encode(record.Content)
	log.Printf("[PROCESS] Tokenized to %d tokens", len(allTokens))

	if len(allTokens) < 2 {
		log.Printf("⚠️  Insufficient tokens (%d) for sliding window in record %d from %s", len(allTokens), record.ChunkID, record.FileName)
		return nil
	}

	// Pre-calculate token character offsets for mapping to SpaCy metadata
	// This is necessary because tiktoken and SpaCy use different tokenization
	tokenStartOffsets := make([]int, len(allTokens))
	currentPos := 0
	for i, tokenID := range allTokens {
		tokenStartOffsets[i] = currentPos
		tokenText := tk.Decode([]int{tokenID})
		currentPos += len(tokenText)
	}

	// 2. Generate sliding windows
	windowStride := effectiveWindowStride(record, config)
	log.Printf("[PROCESS] Step 2/6: Generating sliding windows (size=%d, stride=%d)...", config.WindowSize, windowStride)
	sg := sliding.NewGenerator(config.WindowSize, windowStride)
	windows := sg.GenerateWindows(allTokens)
	log.Printf("[PROCESS] Generated %d sliding windows", len(windows))

	if len(windows) == 0 {
		log.Printf("⚠️  No windows generated for record %d from %s", record.ChunkID, record.FileName)
		return nil
	}

	// 3. Process windows in batches for API efficiency
	log.Printf("[PROCESS] Step 3/6: Processing %d windows in batches of %d...", len(windows), config.BatchSize)
	for batchStart := 0; batchStart < len(windows); batchStart += config.BatchSize {
		batchEnd := batchStart + config.BatchSize
		if batchEnd > len(windows) {
			batchEnd = len(windows)
		}
		batch := windows[batchStart:batchEnd]
		batchSize := len(batch)
		log.Printf("[PROCESS]   Processing batch %d-%d of %d windows...", batchStart, batchEnd, len(windows))

		// Check quota before this batch
		if cpManager != nil {
			if !cpManager.UseQuota(batchSize) {
				used, limit, _ := cpManager.GetQuotaStatus()
				log.Printf("🛑 Quota exhausted before batch (need %d, have %d/%d). Stopping.",
					batchSize, used, limit)
				return fmt.Errorf("quota exhausted")
			}
			used, limit, remaining := cpManager.GetQuotaStatus()
			if limit <= 0 {
				log.Printf("📊 Quota: unlimited (used: %d, using %d for this batch)",
					used, batchSize)
			} else {
				log.Printf("📊 Quota: %d/%d used, %d remaining (using %d for this batch)",
					used, limit, remaining, batchSize)
			}
		}

		// 4. Extract context texts for batch embedding
		log.Printf("[PROCESS] Step 4/6: Extracting %d context texts...", len(batch))
		contextTexts := make([]string, len(batch))
		for i, window := range batch {
			contextTexts[i] = tk.Decode(window.ContextTokens)
		}

		// 5. Get batch embeddings from Cloudflare
		log.Printf("[PROCESS] Step 5/6: Requesting embeddings from endpoint...")
		batchEmbeddings, err := ew.GetBatchEmbeddings(contextTexts)
		if err != nil {
			log.Printf("[PROCESS] ❌ Embedding request FAILED: %v", err)
			return fmt.Errorf("batch embedding failed for record %d: %w", record.ChunkID, err)
		}

		// 6. Process each window in the batch
		log.Printf("[PROCESS] Step 6/6: Writing %d frames to JSON array...", len(batch))

		// Maintain a rolling XOR history for memory slots (6-8)
		// We use a simple 3-uint32 state that we XOR with each new header
		memoryState := [3]uint32{0x5F3759DF, 0x12345678, 0x87654321} // Initial seeds

		for i, window := range batch {
			// Find corresponding SpaCy metadata for the target token
			targetTokenIdx := window.EndPos
			targetOffset := int32(tokenStartOffsets[targetTokenIdx])

			var pos uint8 = 0
			var tense uint8 = 0
			var depHash uint32 = 0

			// More robust search: find the SpaCy token that most closely matches or contains this offset
			bestMatchIdx := -1
			minDiff := int32(1000000)

			for j, spacyOffset := range record.TokenOffsets {
				diff := spacyOffset - targetOffset
				if diff < 0 {
					diff = -diff
				}

				if diff < minDiff {
					minDiff = diff
					bestMatchIdx = j
				}

				// Perfect match found
				if diff == 0 {
					break
				}
			}

			// If we found a reasonably close match (within 2 characters), use its metadata
			if bestMatchIdx != -1 && minDiff <= 2 {
				pos = record.POSTags[bestMatchIdx]
				tense = record.Tenses[bestMatchIdx]
				depHash = record.DepHashes[bestMatchIdx]
			}

			// Pack the frame into the 12-slot specification
			domainHeading := strings.TrimSpace(record.Heading)
			if strings.HasPrefix(record.FileName, "arxiv:") {
				domainHeading = "arXiv heading: " + domainHeading
			}
			if domainHeading == "" {
				domainHeading = record.Instruction
			}
			asicSlots := tp.PackFrame(
				batchEmbeddings[i],
				pos,
				tense,
				depHash,
				memoryState[:],
				uint16(targetTokenIdx),
				domainHeading,
				record.Input,
			)

			// Update memory state for next window (XOR with current slots 0-2)
			memoryState[0] ^= asicSlots[0]
			memoryState[1] ^= asicSlots[1]
			memoryState[2] ^= asicSlots[2]

			// Create training frame
			frame := schema.TrainingFrame{
				SourceFile:    record.FileName,
				ChunkID:       int32(record.ChunkID),
				WindowStart:   int32(window.StartPos),
				WindowEnd:     int32(window.EndPos),
				ContextLength: int32(len(window.ContextTokens)),
				TargetTokenID: int32(window.TargetToken),
				BestSeed:      nil,
			}
			frame.SetAsicSlots(asicSlots)

			*frames = append(*frames, frame)
			*frameCount++
		}
	}

	log.Printf("[PROCESS] ✅ Completed record %d: %d windows -> %d frames", record.ChunkID, len(windows), *frameCount)
	return nil
}

func effectiveWindowStride(record *schema.MinedRecord, config *Config) int {
	stride := config.WindowStride
	if stride <= 0 {
		stride = 1
	}
	if strings.HasPrefix(record.FileName, "arxiv:") && stride < config.WindowSize {
		return config.WindowSize
	}
	return stride
}

func processStreamingJSON(jsonFile *os.File, tp *mapper.TensorPacker, tk *tokenizer.Service, ew *embeddings.Service, frames *[]schema.TrainingFrame, config *Config, cpManager *checkpoint.Manager) error {
	decoder := json.NewDecoder(jsonFile)
	decoder.DisallowUnknownFields()

	var recordCount, frameCount, fixedCount int64

	// Processing Loop
	log.Printf("🚀 Starting encoding from %s...", config.InputFile)

	for decoder.More() {
		var record schema.MinedRecord
		if err := decoder.Decode(&record); err != nil {
			// Try to fix JSON errors by reading raw and fixing
			if jsonErr := fixAndRetryRecord(decoder, &record, config, &fixedCount); jsonErr != nil {
				if recordCount%10 == 0 || recordCount < 10 {
					log.Printf("⚠️  Skipping bad record %d: %v", recordCount, jsonErr)
				}
				continue
			}
		}

		recordCount++

		// Process with sliding windows
		if err := processSingleRecordWithSlidingWindow(&record, &frameCount, tp, tk, ew, frames, config, cpManager); err != nil {
			return fmt.Errorf("failed to process record %d: %w", recordCount, err)
		}

		// Progress logging every 100 records
		if recordCount%100 == 0 {
			log.Printf("📊 Processed %d records, generated %d frames (fixed: %d)...", recordCount, frameCount, fixedCount)
		}
	}

	log.Printf("📈 Total: %d records processed, %d training frames generated", recordCount, frameCount)
	if fixedCount > 0 {
		log.Printf("🔧 JSON issues fixed: %d records", fixedCount)
	}

	return nil
}

func fixAndRetryRecord(decoder *json.Decoder, record *schema.MinedRecord, config *Config, fixedCount *int64) error {
	// Read the raw JSON bytes and try to fix
	var rawRecord map[string]interface{}
	if err := decoder.Decode(&rawRecord); err != nil {
		return fmt.Errorf("failed to read raw record: %w", err)
	}

	// Convert back to JSON and fix common issues
	jsonBytes, err := json.Marshal(rawRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal raw record: %w", err)
	}

	fixedJSON := fixJSONString(string(jsonBytes))

	// Try to parse the fixed JSON
	if err := json.Unmarshal([]byte(fixedJSON), record); err != nil {
		return fmt.Errorf("failed to parse fixed record: %w", err)
	}

	*fixedCount++
	return nil
}

// processArrowFile processes Arrow IPC stream files
func processArrowFile(filePath string, tp *mapper.TensorPacker, tk *tokenizer.Service, ew embeddings.EmbeddingService, frames *[]schema.TrainingFrame, config *Config, cpManager *checkpoint.Manager) (int64, error) {
	log.Printf("🔄 Attempting Arrow IPC stream processing from: %s", filePath)

	// Read records from Arrow IPC stream
	records, err := schema.ReadDocumentRecordsFromArrowIPC(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read Arrow IPC stream: %w", err)
	}

	log.Printf("📋 Detected Arrow IPC format with %d records", len(records))
	return processDocumentRecords(records, tp, tk, ew, frames, config, cpManager)
}

func writeJSONOutput(filePath string, frames []schema.TrainingFrame) error {
	// Write to JSON
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(frames); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

func fixJSONString(jsonStr string) string {
	// Step 1: Fix invalid \x escape sequences by removing them completely
	invalidEscapeRegex := regexp.MustCompile(`\\x[0-9a-fA-F]{2}`)
	fixed := invalidEscapeRegex.ReplaceAllString(jsonStr, "")

	// Step 2: Remove control characters that break JSON
	controlCharRegex := regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F]`)
	fixed = controlCharRegex.ReplaceAllString(fixed, "")

	// Step 3: Fix invalid escape sequences
	// Handle escaped backslash followed by invalid char FIRST: \\ + invalid -> just invalid
	// This matches \\ (escaped backslash) followed by any invalid char
	fixed = regexp.MustCompile(`\\\\([^"\\bfnrt/])`).ReplaceAllString(fixed, "$1")

	// Then handle single backslash followed by invalid char: \ + invalid -> just invalid
	fixed = regexp.MustCompile(`\\([^"\\bfnrt/])`).ReplaceAllString(fixed, "$1")

	return fixed
}
