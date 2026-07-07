package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/bits"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"knirvhasher/pkg/hashing/jitter"

	"github.com/lab/hasher/data-trainer/internal/config"
	"github.com/lab/hasher/data-trainer/internal/logging"
	"github.com/lab/hasher/data-trainer/pkg/deployment"
	"github.com/lab/hasher/data-trainer/pkg/simulator"
	"github.com/lab/hasher/data-trainer/pkg/storage"
	"github.com/lab/hasher/data-trainer/pkg/training"
	"github.com/lab/hasher/data-trainer/pkg/validator"
)

var (
	configFile     = flag.String("config", "", "Path to configuration file")
	dataPath       = flag.String("data", "", "Path to data directory (default: app data directory)")
	maxEpochs      = flag.Int("epochs", 10, "Maximum number of training epochs")
	population     = flag.Int("population", 256, "Population size for evolution")
	maxGenerations = flag.Int("generations", 2000, "Maximum number of generations")
	difficultyBits = flag.Int("difficulty-bits", config.DefaultDifficultyBits, fmt.Sprintf("Number of leading bits that must match (%d-%d)", config.MinDifficultyBits, config.MaxDifficultyBits))
	verbose        = flag.Bool("verbose", false, "Enable verbose logging")
	sequential     = flag.Bool("sequential", false, "Process tokens sequentially (cleaner logs)")
	hashMethod     = flag.String("hash-method", "auto", "Hash method to use: auto, software, cuda")
)

type seedWriterInterface interface {
	AddSeedWrite(sourceFile string, slots [12]uint32, targetTokenID int32, bestSeed []byte) error
	WriteBack() error
}

type TrainingOrchestrator struct {
	logger           *logging.Logger
	simulator        simulator.HashSimulator
	storage          *storage.CSVStorage
	harness          *training.EvolutionaryHarness
	validator        validator.Validator
	flashManager     *deployment.FlashManager
	checkpointMgr    *storage.CheckpointManager
	dataIngestor     *storage.DataIngestor
	trainingData     []*training.TrainingRecord
	seedWriter       seedWriterInterface
	config           *config.Config
	dataPath         string
	sequential       bool
	runID            string
	runStatusPath    string
	trainingDataPath string
	currentBatch     int
	batchesDone      int
	totalBatches     int
	pipelineErrors   []string
	allRecordsSeeded bool
}

type TrainingRunStatus struct {
	RunID            string    `json:"run_id"`
	Status           string    `json:"status"`
	Message          string    `json:"message,omitempty"`
	DataPath         string    `json:"data_path"`
	TrainingDataPath string    `json:"training_data_path,omitempty"`
	RecordsLoaded    int       `json:"records_loaded"`
	CheckpointTokens int       `json:"checkpoint_tokens"`
	AverageFitness   float64   `json:"average_fitness"`
	CurrentBatch     int       `json:"current_batch"`
	BatchesDone      int       `json:"batches_done"`
	TotalBatches     int       `json:"total_batches"`
	LastError        string    `json:"last_error,omitempty"`
	Errors           []string  `json:"errors,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ingestionLogger wraps the internal logging.Logger to implement storage.IngestionLogger
type ingestionLogger struct {
	logger *logging.Logger
}

func (il *ingestionLogger) Info(format string, args ...interface{}) {
	il.logger.Info(format, args...)
}

func (il *ingestionLogger) Debug(format string, args ...interface{}) {
	il.logger.Debug(format, args...)
}

func (il *ingestionLogger) Warn(format string, args ...interface{}) {
	il.logger.Warn(format, args...)
}

func (il *ingestionLogger) Error(format string, args ...interface{}) {
	il.logger.Error(format, args...)
}

// getAppDataDir returns the OS-specific application data directory.
// Privileged launches use a system location so sudo does not split data
// between /root and the invoking user's home directory.
func getAppDataDir() (string, error) {
	var basePath string

	if runtime.GOOS == "windows" {
		// Windows: %APPDATA% or %LOCALAPPDATA%
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			basePath = localAppData
		} else if appData := os.Getenv("APPDATA"); appData != "" {
			basePath = appData
		} else {
			// Fallback to user profile
			userProfile := os.Getenv("USERPROFILE")
			if userProfile == "" {
				currentUser, err := user.Current()
				if err != nil {
					return "", fmt.Errorf("failed to get current user: %w", err)
				}
				userProfile = currentUser.HomeDir
			}
			basePath = filepath.Join(userProfile, "AppData", "Local")
		}
		basePath = filepath.Join(basePath, "knirvserver", "knirvhasher", "data")
	} else {
		// KNIRV_APP_DATA_DIR takes highest precedence
		if explicit := os.Getenv("KNIRV_APP_DATA_DIR"); explicit != "" {
			p := filepath.Join(explicit, "knirvhasher", "data")
			if err := os.MkdirAll(p, 0755); err == nil {
				return p, nil
			}
		}
		// System location
		if err := os.MkdirAll("/var/lib/knirvserver/knirvhasher/data", 0755); err == nil {
			return "/var/lib/knirvserver/knirvhasher/data", nil
		}
		// Unix-like systems: ~/.local/share
		home := os.Getenv("HOME")
		if home == "" {
			currentUser, err := user.Current()
			if err != nil {
				return "", fmt.Errorf("failed to get current user: %w", err)
			}
			home = currentUser.HomeDir
		}

		dataHome := os.Getenv("XDG_DATA_HOME")
		if dataHome == "" {
			dataHome = filepath.Join(home, ".local", "share")
		}
		basePath = filepath.Join(dataHome, "knirvserver", "knirvhasher", "data")
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create app data directory: %w", err)
	}

	return basePath, nil
}

func main() {
	flag.Parse()

	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Load configuration file if specified
	cfg, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	loggingConfig := &logging.LoggingConfig{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	}

	logger, err := logging.NewLogger(loggingConfig)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	logger.Info("Starting HASHER Data Trainer v1.0")

	// Determine data path - use flag if provided, otherwise default to app data dir
	effectiveDataPath := *dataPath
	if effectiveDataPath == "" {
		effectiveDataPath, err = getAppDataDir()
		if err != nil {
			logger.Fatal("Failed to get app data directory: %v", err)
		}
		logger.Info("Using default data directory: %s", effectiveDataPath)
	} else {
		logger.Info("Using specified data directory: %s", effectiveDataPath)
	}

	orchestrator, err := NewTrainingOrchestrator(logger, cfg, effectiveDataPath, *sequential)
	if err != nil {
		logger.Fatal("Failed to initialize orchestrator: %v", err)
	}
	defer orchestrator.Shutdown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("Received signal %v, shutting down...", sig)
		cancel()

		// Force exit if graceful shutdown takes too long
		go func() {
			time.Sleep(5 * time.Second)
			logger.Warn("Graceful shutdown timed out, forcing exit")
			os.Exit(1)
		}()
	}()

	if err := orchestrator.Run(ctx, *maxEpochs, *population); err != nil {
		orchestrator.recordPipelineError(err.Error())
		_ = orchestrator.writeRunStatus("failed", err.Error())
		logger.Fatal("Training failed: %v", err)
	}

	_ = orchestrator.writeRunStatus("completed", "Training completed successfully")
	logger.Info("Training completed successfully")
}

func NewTrainingOrchestrator(logger *logging.Logger, cfg *config.Config, dataPath string, sequential bool) (*TrainingOrchestrator, error) {
	orchestrator := &TrainingOrchestrator{
		logger:        logger,
		config:        cfg,
		dataPath:      dataPath,
		sequential:    sequential,
		runID:         time.Now().UTC().Format("20060102T150405.000000000Z"),
		runStatusPath: filepath.Join(dataPath, "runs", "latest_training_run.json"),
	}

	if err := orchestrator.initializeComponents(); err != nil {
		return nil, err
	}

	return orchestrator, nil
}

func selectTrainingDataPath(dataPath string) (path string, seededOutputOnly bool, err error) {
	framesDir := filepath.Join(dataPath, "frames")
	candidates := []string{
		filepath.Join(framesDir, "training_frames.json"),
		filepath.Join(framesDir, "training_frames.arrow"),
		filepath.Join(framesDir, "training_frames.parquet"),
	}

	for _, candidate := range candidates {
		info, statErr := os.Stat(candidate)
		if statErr == nil && !info.IsDir() && info.Size() > 0 {
			return candidate, false, nil
		}
	}

	seededOutput := filepath.Join(framesDir, "training_frames_with_seeds.json")
	info, statErr := os.Stat(seededOutput)
	if statErr == nil && !info.IsDir() && info.Size() > 0 {
		return seededOutput, true, nil
	}

	return "", false, fmt.Errorf("training data not found in %s - please run the data encoder first", framesDir)
}

func (to *TrainingOrchestrator) initializeComponents() error {
	// Get OS-specific application data directory
	appDataDir, err := getAppDataDir()
	if err != nil {
		return fmt.Errorf("failed to get app data directory: %w", err)
	}
	to.logger.Info("Using app data directory: %s", appDataDir)
	if to.config != nil {
		to.logger.Info("Configuration loaded from file")
	}

	to.logger.Info("Initializing simulator with hash-method=%s...", *hashMethod)
	simConfig := &simulator.SimulatorConfig{
		DeviceType:     fmt.Sprintf("hasher_%s", *hashMethod),
		MaxConcurrency: 100,
		TargetHashRate: 500000000,
		CacheSize:      10000,
		GPUDevice:      0,
		Timeout:        30,
	}
	// Use the new HasherWrapper which implements HashSimulator using hasher's HashMethod
	sim := simulator.NewHasherWrapper(simConfig)
	if err := sim.Initialize(simConfig); err != nil {
		return fmt.Errorf("failed to initialize simulator: %w", err)
	}
	to.logger.Info("Simulator initialized with hash method: %s", sim.GetHashMethod().Name())
	to.simulator = sim

	to.logger.Info("Initializing storage...")
	storagePath := filepath.Join(appDataDir, "weights")
	csvStorage := storage.NewCSVStorage(storagePath, 1000)
	if err := csvStorage.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	to.storage = csvStorage

	to.logger.Info("Initializing checkpoint manager...")
	checkpointPath := filepath.Join(appDataDir, "checkpoints")
	if err := os.MkdirAll(checkpointPath, 0755); err != nil {
		return fmt.Errorf("failed to create checkpoint directory: %w", err)
	}
	checkpointFile := filepath.Join(checkpointPath, "checkpoints.db")
	to.logger.Info("Checkpoint database: %s", checkpointFile)
	checkpointMgr := storage.NewCheckpointManager(checkpointFile)
	if err := checkpointMgr.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize checkpoint manager: %w", err)
	}
	to.checkpointMgr = checkpointMgr

	// Initialize data ingestion - REQUIRED
	to.logger.Info("Initializing data ingestion...")
	if to.sequential {
		to.logger.Info("Sequential processing enabled (cleaner logs)")
	}
	trainingDataPath, seededOutputOnly, err := selectTrainingDataPath(to.dataPath)
	if err != nil {
		return err
	}

	to.trainingDataPath = trainingDataPath
	if err := to.writeRunStatus("initializing", "training data selected"); err != nil {
		to.logger.Warn("Failed to write run status: %v", err)
	}
	to.logger.Info("Found training data at: %s", trainingDataPath)
	dataIngestor := storage.NewJSONDataIngestor(filepath.Dir(trainingDataPath))
	dataIngestor.SetLogger(&ingestionLogger{to.logger})
	dataIngestor.SetCheckpointManager(to.checkpointMgr)
	dataIngestor.SetChunkSize(1000) // Process 1000 records at a time
	dataIngestor.SetFiles([]string{trainingDataPath})
	to.dataIngestor = dataIngestor.DataIngestor

	to.logger.Info("Initializing seed writer for write-back...")
	seedWriter := storage.NewDualSeedWriter(to.dataPath)
	to.seedWriter = seedWriter

	trainingRecords, err := dataIngestor.ProcessAllFiles(nil)
	if err != nil {
		return fmt.Errorf("failed to ingest training data: %w - please check that the parquet file is valid", err)
	}

	if len(trainingRecords) == 0 {
		if seededOutputOnly {
			to.allRecordsSeeded = true
			to.logger.Info("No unseeded records remain in %s; treating training as already complete", trainingDataPath)
			if err := to.writeRunStatus("completed", "all records already have seeds"); err != nil {
				to.logger.Warn("Failed to write run status: %v", err)
			}
			return nil
		}
		return fmt.Errorf("no training records found in %s - file may be empty or corrupted", trainingDataPath)
	}

	to.trainingData = trainingRecords
	to.logger.Info("Successfully ingested %d training records", len(trainingRecords))
	if err := to.writeRunStatus("running", "training records ingested"); err != nil {
		to.logger.Warn("Failed to write run status: %v", err)
	}

	// Validate training data
	report, err := dataIngestor.ValidateTrainingData(trainingRecords)
	if err != nil {
		return fmt.Errorf("failed to validate training data: %w", err)
	}

	to.logger.Info("Training data validation: %s", report.GetSummary())
	if !report.Valid {
		return fmt.Errorf("training data validation failed: %s", report.GetSummary())
	}

	to.logger.Info("Initializing evolutionary harness...")
	harness := training.NewEvolutionaryHarness(*population)
	// Set difficulty mask based on requested bit count
	if *difficultyBits >= config.MinDifficultyBits && *difficultyBits <= config.MaxDifficultyBits {
		mask := uint32(0xFFFFFFFF) << (32 - *difficultyBits)
		harness.SetDifficultyMask(mask)
		to.logger.Info("Difficulty set to %d bits (mask: 0x%08X)", *difficultyBits, mask)
	} else {
		to.logger.Warn("Invalid difficulty bits: %d (must be between %d and %d), using default: %d",
			*difficultyBits, config.MinDifficultyBits, config.MaxDifficultyBits, config.DefaultDifficultyBits)
		// Calculate mask at runtime to avoid constant overflow
		var defaultMask uint32 = 0xFFF00000 // 12-bit default mask
		harness.SetDifficultyMask(defaultMask)
	}
	to.harness = harness

	to.logger.Info("Initializing validator...")
	validatorConfig := &validator.ValidatorConfig{
		Timeout:            30 * time.Second,
		MaxConcurrency:     10,
		RetryAttempts:      3,
		ToleranceThreshold: 0.01,
		EnableASIC:         false,
	}
	validator := validator.NewCrossHardwareValidator(sim, validatorConfig)
	if err := validator.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize validator: %w", err)
	}
	to.validator = validator

	to.logger.Info("Initializing flash manager...")
	flashConfig := &deployment.FlashConfig{
		BPFMapPath:        "/sys/fs/bpf/hasher_weights",
		DeploymentTimeout: 300 * time.Second,
		MaxRetries:        3,
		RetryDelay:        5 * time.Second,
		ValidationMode:    "strict",
		BackupEnabled:     true,
		BackupPath:        filepath.Join(appDataDir, "backups"),
		RollbackEnabled:   true,
	}
	flashManager := deployment.NewFlashManager(csvStorage, flashConfig)
	if err := flashManager.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize flash manager: %w", err)
	}
	to.flashManager = flashManager

	// Populate FlashManager Knowledge Base
	to.logger.Info("Populating FlashManager Knowledge Base...")
	to.flashManager.SetKnowledgeBase(to.trainingData)

	// Start Jitter RPC Server
	to.logger.Info("Starting Jitter RPC Server on /tmp/jitter.sock...")
	jitterServer := jitter.NewServer("/tmp/jitter.sock", to.flashManager.GetAssociativeJitter)
	if err := jitterServer.Start(); err != nil {
		to.logger.Warn("Failed to start Jitter Server: %v", err)
	} else {
		to.logger.Info("Jitter Server active")
	}

	return nil
}

func (to *TrainingOrchestrator) Run(ctx context.Context, maxEpochs, populationSize int) error {
	to.logger.Info("Starting training with %d epochs, population size %d", maxEpochs, populationSize)
	if err := to.writeRunStatus("running", "training started"); err != nil {
		to.logger.Warn("Failed to write run status: %v", err)
	}

	if to.allRecordsSeeded {
		to.logger.Info("All training records already have seeds. Nothing to train.")
		return to.finalizeTraining()
	}

	if len(to.trainingData) == 0 {
		// Fallback to synthetic training
		to.logger.Warn("No training data available, falling back to synthetic training")
		tokenMap := to.createTokenMap()
		return to.runSyntheticTraining(ctx, maxEpochs, populationSize, tokenMap)
	}

	to.logger.Info("Training with %d ingested records", len(to.trainingData))
	return to.runTrainingWithData(ctx, maxEpochs, populationSize, to.trainingData)
}

func (to *TrainingOrchestrator) runSyntheticTraining(ctx context.Context, maxEpochs, _ int, tokenMap map[int32]bool) error {
	for epoch := 0; epoch < maxEpochs; epoch++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		to.logger.Info("Starting epoch %d/%d", epoch+1, maxEpochs)
		to.harness.UpdateDifficulty(epoch + 1)

		if err := to.runEpoch(ctx, epoch, tokenMap); err != nil {
			to.logger.Error("Epoch %d failed: %v", epoch+1, err)
			return err
		}

		if epoch%5 == 0 {
			if err := to.saveProgress(epoch); err != nil {
				to.logger.Warn("Failed to save progress: %v", err)
			}
		}
	}

	to.logger.Info("Synthetic training completed all epochs")
	return to.finalizeTraining()
}

func (to *TrainingOrchestrator) runTrainingWithData(ctx context.Context, maxEpochs, populationSize int, records []*training.TrainingRecord) error {
	batchSize := populationSize
	if batchSize <= 0 {
		batchSize = 1
	}
	batchesPerEpoch := (len(records) + batchSize - 1) / batchSize
	to.totalBatches = batchesPerEpoch * maxEpochs
	to.currentBatch = 0
	to.batchesDone = 0
	if err := to.writeRunStatus("running", "training batches scheduled"); err != nil {
		to.logger.Warn("Failed to write run status: %v", err)
	}

	// Shuffle records for training
	for epoch := 0; epoch < maxEpochs; epoch++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		to.logger.Info("Starting epoch %d/%d with %d records", epoch+1, maxEpochs, len(records))
		to.harness.UpdateDifficulty(epoch + 1)

		// Process records in batches
		epochTrainedCount := 0
		batchIndex := 0
		for i := 0; i < len(records); i += batchSize {
			end := i + batchSize
			if end > len(records) {
				end = len(records)
			}

			to.currentBatch = epoch*batchesPerEpoch + batchIndex + 1
			if err := to.writeRunStatus("running", fmt.Sprintf("training batch %d/%d", to.currentBatch, to.totalBatches)); err != nil {
				to.logger.Warn("Failed to write run status: %v", err)
			}

			batch := records[i:end]
			trained, err := to.trainBatch(ctx, batch)
			if err != nil {
				msg := fmt.Sprintf("Batch %d-%d failed: %v", i, end-1, err)
				to.recordPipelineError(msg)
				to.logger.Error("%s", msg)
				_ = to.writeRunStatus("failed", msg)
				return err
			}
			to.batchesDone = to.currentBatch
			epochTrainedCount += trained

			if i%(batchSize*10) == 0 {
				to.logger.Debug("Processed %d/%d records", i, len(records))
			}
			if err := to.writeRunStatus("running", fmt.Sprintf("completed batch %d/%d", to.batchesDone, to.totalBatches)); err != nil {
				to.logger.Warn("Failed to write run status: %v", err)
			}
			batchIndex++
		}

		if epochTrainedCount == 0 && epoch > 0 {
			to.logger.Info("All records already have winning seeds for current difficulty. Training complete.")
			break
		}

		if epoch%2 == 0 {
			if err := to.saveProgress(epoch); err != nil {
				to.logger.Warn("Failed to save progress: %v", err)
			}
		}
		if err := to.writeRunStatus("running", fmt.Sprintf("completed epoch %d/%d", epoch+1, maxEpochs)); err != nil {
			to.logger.Warn("Failed to write run status: %v", err)
		}
	}

	to.logger.Info("Data training completed all epochs")
	return to.finalizeTraining()
}

func (to *TrainingOrchestrator) runEpoch(ctx context.Context, epoch int, tokenMap map[int32]bool) error {
	targetTokens := to.getTargetTokens(epoch)

	for i, targetToken := range targetTokens {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if i%50 == 0 {
			to.logger.Debug("Processing token %d/%d in epoch %d", i+1, len(targetTokens), epoch+1)
		}

		if err := to.trainToken(ctx, targetToken, tokenMap); err != nil {
			to.logger.Warn("Failed to train token %d: %v", targetToken, err)
			continue
		}
	}

	return nil
}

func (to *TrainingOrchestrator) trainBatch(ctx context.Context, records []*training.TrainingRecord) (int, error) {
	trainedCount := 0
	for _, record := range records {
		select {
		case <-ctx.Done():
			return trainedCount, ctx.Err()
		default:
		}

		// Check if we already have a winning seed for this token
		hasCheckpoint, err := to.checkpointMgr.HasCheckpoint(record.TargetToken)
		if err == nil && hasCheckpoint {
			continue
		}

		if err := to.trainRecord(ctx, record); err != nil {
			to.logger.Warn("Failed to train record for token %d: %v", record.TargetToken, err)
			continue
		}
		trainedCount++
	}

	return trainedCount, nil
}

func (to *TrainingOrchestrator) trainRecord(ctx context.Context, record *training.TrainingRecord) error {
	contextHash := training.ComputeContextHash(record.TokenSequence, 5)
	pop := training.NewSeedPopulation(record.TargetToken, contextHash, *population)

	tokenMap := map[int32]bool{record.TargetToken: true}

	for gen := 0; gen < *maxGenerations; gen++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		results, err := to.harness.EvaluatePopulation(pop, record, tokenMap, to.simulator)
		if err != nil {
			return fmt.Errorf("failed to evaluate population: %w", err)
		}

		eliteSeeds := to.harness.GetEliteSeeds(results)
		// Check for winning seed using difficulty mask (partial match) or high advantage
		if len(eliteSeeds) > 0 {
			bestSeed := eliteSeeds[0]
			// Winning condition: passes difficulty mask OR has very high advantage
			if to.harness.IsWinningSeed(bestSeed.HashOutput, uint32(record.TargetToken)) {
				to.logger.Info("[WIN] Token %d: Found winning seed in generation %d (%d-bit prefix match)", record.TargetToken, gen, *difficultyBits)
				return to.saveWinningSeed(record, bestSeed, gen)
			}
			// Also accept if advantage is very high AND meets a minimum quality threshold
			if bestSeed.Advantage > 2.0 {
				diff := bestSeed.HashOutput ^ uint32(record.TargetToken)
				matchingBits := bits.LeadingZeros32(diff)
				if matchingBits >= *difficultyBits { // Require at least difficulty-bits of similarity for a high-advantage win
					to.logger.Info("[WIN] Token %d: Found winning seed in gen %d (high advantage=%.2f, %d bits)", record.TargetToken, gen, bestSeed.Advantage, matchingBits)
					return to.saveWinningSeed(record, bestSeed, gen)
				}
			}
		}

		pop.Seeds = to.harness.SelectAndMutate(results, pop.Seeds)
		pop.Generation++
		to.harness.Generation = gen

		if gen%10 == 0 || gen == *maxGenerations-1 {
			bestPrefix := 0
			bestHamming := 0
			if len(eliteSeeds) > 0 {
				diff := eliteSeeds[0].HashOutput ^ uint32(record.TargetToken)
				bestPrefix = bits.LeadingZeros32(diff)
				bestHamming = 32 - bits.OnesCount32(diff)
			}
			stats := fmt.Sprintf("fit=%.4f prefix=%d ham=%d", pop.Fitness, bestPrefix, bestHamming)
			to.logger.ProgressBar(gen+1, *maxGenerations, fmt.Sprintf("Token %d", record.TargetToken), stats)
		}

		if gen%50 == 0 && *verbose {
			bestMatch := 0
			if len(eliteSeeds) > 0 {
				diff := eliteSeeds[0].HashOutput ^ uint32(record.TargetToken)
				bestMatch = bits.LeadingZeros32(diff)
			}
			to.logger.Debug("Token %d gen %d: fitness=%.4f, best_match=%d bits", record.TargetToken, gen, pop.Fitness, bestMatch)
		}
	}

	to.logger.Warn("Token %d: No %d-bit match after %d generations (try increasing -generations)", record.TargetToken, *difficultyBits, *maxGenerations)
	return nil
}

func (to *TrainingOrchestrator) trainToken(ctx context.Context, targetToken int32, tokenMap map[int32]bool) error {
	hasCheckpoint, err := to.checkpointMgr.HasCheckpoint(targetToken)
	if err != nil {
		return fmt.Errorf("failed to check checkpoint: %w", err)
	}

	if hasCheckpoint {
		to.logger.Debug("Token %d already has checkpoint, skipping", targetToken)
		return nil
	}

	contextHash := training.ComputeContextHash([]int32{targetToken}, 5)
	pop := training.NewSeedPopulation(targetToken, contextHash, *population)

	// Create a TrainingRecord for the targetToken
	record := &training.TrainingRecord{
		SourceFile:    "synthetic",
		TargetToken:   targetToken,
		TokenSequence: []int32{targetToken},
		ContextHash:   training.ComputeContextHash([]int32{targetToken}, 5),
		FeatureVector: [12]uint32{}, // Initialize with zeros
	}

	for gen := 0; gen < *maxGenerations; gen++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		results, err := to.harness.EvaluatePopulation(pop, record, tokenMap, to.simulator)
		if err != nil {
			return fmt.Errorf("failed to evaluate population: %w", err)
		}

		eliteSeeds := to.harness.GetEliteSeeds(results)
		// Check for winning seed using difficulty mask (partial match) or high advantage
		if len(eliteSeeds) > 0 {
			bestSeed := eliteSeeds[0]
			// Winning condition: passes difficulty mask OR has very high advantage
			if to.harness.IsWinningSeed(bestSeed.HashOutput, uint32(targetToken)) {
				to.logger.Info("[WIN] Token %d: Found winning seed in generation %d (%d-bit prefix match)", targetToken, gen, *difficultyBits)
				return to.saveWinningSeed(record, bestSeed, gen)
			}
			// Also accept if advantage is very high (converged solution)
			if bestSeed.Advantage > 2.0 {
				to.logger.Info("[WIN] Token %d: Found winning seed in generation %d (high advantage=%.2f)", targetToken, gen, bestSeed.Advantage)
				return to.saveWinningSeed(record, bestSeed, gen)
			}
		}

		pop.Seeds = to.harness.SelectAndMutate(results, pop.Seeds)
		pop.Generation++
		to.harness.Generation = gen

		if gen%10 == 0 || gen == *maxGenerations-1 {
			bestPrefix := 0
			bestHamming := 0
			if len(eliteSeeds) > 0 {
				diff := eliteSeeds[0].HashOutput ^ uint32(targetToken)
				bestPrefix = bits.LeadingZeros32(diff)
				bestHamming = 32 - bits.OnesCount32(diff)
			}
			stats := fmt.Sprintf("fit=%.4f prefix=%d ham=%d", pop.Fitness, bestPrefix, bestHamming)
			to.logger.ProgressBar(gen+1, *maxGenerations, fmt.Sprintf("Token %d", targetToken), stats)
		}

		if gen%50 == 0 && *verbose {
			bestMatch := 0
			if len(eliteSeeds) > 0 {
				diff := eliteSeeds[0].HashOutput ^ uint32(targetToken)
				bestMatch = bits.LeadingZeros32(diff)
			}
			to.logger.Debug("Token %d gen %d: fitness=%.4f, best_match=%d bits", targetToken, gen, pop.Fitness, bestMatch)
		}
	}

	to.logger.Warn("Token %d: No %d-bit match after %d generations (try increasing -generations)", targetToken, *difficultyBits, *maxGenerations)
	return nil
}

func (to *TrainingOrchestrator) saveWinningSeed(record *training.TrainingRecord, seed training.SeedResult, generation int) error {
	weightRecord := storage.WeightRecord{
		TokenID:      record.TargetToken,
		BestSeed:     seed.Seed,
		FitnessScore: seed.Reward,
		Generation:   int32(generation),
		ContextKey:   record.ContextHash,
	}

	layerID := int32(record.TargetToken / 100)
	if err := to.storage.SaveWeights([]storage.WeightRecord{weightRecord}, layerID); err != nil {
		return fmt.Errorf("failed to save weight: %w", err)
	}

	checkpointEntry := training.CheckpointEntry{
		TokenID:      record.TargetToken,
		SeedHash:     storage.ComputeSeedHash(seed.Seed),
		BestSeed:     seed.Seed,
		FitnessScore: seed.Reward,
		LastUpdated:  time.Now().Unix(),
	}

	if err := to.checkpointMgr.SaveCheckpoint(checkpointEntry); err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	// Immediately write best seed back to storage
	to.logger.Info("[DEBUG] saveWinningSeed: Token %d, Slot0: %d, Source: %s",
		record.TargetToken, record.FeatureVector[0], record.SourceFile)

	// Ensure we pass the actual record's feature vector and source file for matching
	if err := to.seedWriter.AddSeedWrite(record.SourceFile, record.FeatureVector, record.TargetToken, seed.Seed); err != nil {
		return fmt.Errorf("failed to queue seed write-back for token %d: %w", record.TargetToken, err)
	} else {
		if err := to.seedWriter.WriteBack(); err != nil {
			msg := fmt.Sprintf("Seed materialization incomplete for token %d: %v", record.TargetToken, err)
			to.recordPipelineError(msg)
			to.logger.Warn("%s", msg)
			if statusErr := to.writeRunStatus("running", msg); statusErr != nil {
				to.logger.Warn("Failed to write run status: %v", statusErr)
			}
		} else {
			to.logger.Info("Wrote best seed for token %d to storage", record.TargetToken)
		}
	}

	to.logger.Info("Saved checkpoint for token %d (fitness=%.4f) [gen=%d]", record.TargetToken, seed.Reward, generation)
	return nil
}

func (to *TrainingOrchestrator) saveProgress(epoch int) error {
	summary, err := to.checkpointMgr.GetCheckpointSummary()
	if err != nil {
		return err
	}

	to.logger.Info("Progress checkpoint at epoch %d: %d tokens, avg_fitness=%.4f",
		epoch, summary.TotalTokens, summary.AverageFitness)

	// Seeds are now written back immediately after each win, nothing to do here
	return nil
}

func (to *TrainingOrchestrator) finalizeTraining() error {
	to.logger.Info("Finalizing training...")

	// Seeds are written back immediately after each win, just do final checkpoint save
	if err := to.checkpointMgr.Close(); err != nil {
		to.logger.Warn("Failed to save final checkpoints: %v", err)
	}

	layers, err := to.storage.ListLayers()
	if err != nil {
		to.logger.Warn("Failed to list layers: %v", err)
	} else {
		to.logger.Info("Created %d weight layers", len(layers))
	}

	stats, err := to.validator.GetValidatorStats()
	if err != nil {
		to.logger.Warn("Failed to get validator stats: %v", err)
	} else {
		to.logger.Info("Validator stats: consistency=%.2f%%, validations=%d",
			stats.ConsistencyRate*100, stats.TotalValidations)
	}

	if err := to.writeRunStatus("completed", "finalized training artifacts"); err != nil {
		to.logger.Warn("Failed to write run status: %v", err)
	}
	return nil
}

func (to *TrainingOrchestrator) recordPipelineError(message string) {
	if message == "" {
		return
	}
	const maxPipelineErrors = 20
	to.pipelineErrors = append(to.pipelineErrors, message)
	if len(to.pipelineErrors) > maxPipelineErrors {
		to.pipelineErrors = to.pipelineErrors[len(to.pipelineErrors)-maxPipelineErrors:]
	}
}

func (to *TrainingOrchestrator) writeRunStatus(status, message string) error {
	if to.runStatusPath == "" {
		return nil
	}

	checkpointTokens := 0
	averageFitness := 0.0
	if to.checkpointMgr != nil {
		if summary, err := to.checkpointMgr.GetCheckpointSummary(); err == nil {
			checkpointTokens = summary.TotalTokens
			averageFitness = summary.AverageFitness
		}
	}

	runStatus := TrainingRunStatus{
		RunID:            to.runID,
		Status:           status,
		Message:          message,
		DataPath:         to.dataPath,
		TrainingDataPath: to.trainingDataPath,
		RecordsLoaded:    len(to.trainingData),
		CheckpointTokens: checkpointTokens,
		AverageFitness:   averageFitness,
		CurrentBatch:     to.currentBatch,
		BatchesDone:      to.batchesDone,
		TotalBatches:     to.totalBatches,
		Errors:           append([]string(nil), to.pipelineErrors...),
		UpdatedAt:        time.Now().UTC(),
	}
	if len(to.pipelineErrors) > 0 {
		runStatus.LastError = to.pipelineErrors[len(to.pipelineErrors)-1]
	}

	if err := os.MkdirAll(filepath.Dir(to.runStatusPath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(runStatus, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(to.runStatusPath, data, 0644)
}

func (to *TrainingOrchestrator) Shutdown() {
	to.logger.Info("Shutting down training orchestrator...")

	if to.validator != nil {
		to.validator.Close()
	}

	if to.checkpointMgr != nil {
		to.checkpointMgr.Close()
	}

	if to.simulator != nil {
		to.simulator.Shutdown()
	}

	if to.logger != nil {
		to.logger.Info("Shutdown complete")
	}
}

func loadConfig(filename string) (*config.Config, error) {
	// Get default app data directory for config paths
	defaultDataPath, err := getAppDataDir()
	if err != nil {
		defaultDataPath = "data" // Fallback to local directory if app data dir can't be determined
	}

	config := &config.Config{
		Simulator: &config.SimulatorConfig{
			DeviceType:     "vhasher",
			MaxConcurrency: 100,
			TargetHashRate: 500000000,
			CacheSize:      10000,
			GPUDevice:      0,
			Timeout:        30,
		},
		Storage: &config.StorageConfig{
			BasePath:  filepath.Join(defaultDataPath, "weights"),
			LayerSize: 1000,
		},
		Training: &config.TrainingConfig{
			PopulationSize:  256,
			MaxGenerations:  500,
			EliteRatio:      0.25,
			MutationRate:    0.05,
			TargetFitness:   0.80,
			ValidationSplit: 0.1,
		},
		Deployment: &config.DeploymentConfig{
			BPFMapPath:        "/sys/fs/bpf/hasher_weights",
			DeploymentTimeout: "300s",
			MaxRetries:        3,
			RetryDelay:        "5s",
			ValidationMode:    "strict",
			BackupEnabled:     true,
			BackupPath:        filepath.Join(defaultDataPath, "backups"),
			RollbackEnabled:   true,
		},
		Validation: &config.ValidationConfig{
			Timeout:            "30s",
			MaxConcurrency:     10,
			RetryAttempts:      3,
			ToleranceThreshold: 0.01,
			EnableASIC:         false,
		},
		Logging: &config.LoggingConfig{
			Level:      "info",
			Format:     "text",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 10,
			MaxAge:     30,
		},
	}

	if filename == "" {
		return config, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return config, nil
}

func (to *TrainingOrchestrator) createTokenMap() map[int32]bool {
	tokenMap := make(map[int32]bool)

	for i := int32(1); i <= 1000; i++ {
		tokenMap[i] = true
	}

	return tokenMap
}

func (to *TrainingOrchestrator) getTargetTokens(epoch int) []int32 {
	tokens := make([]int32, 0, 50) // Smaller set for testing

	start := int32(epoch*50 + 1)
	end := start + 49
	if end > 500 {
		end = 500
	}

	for i := start; i <= end; i++ {
		tokens = append(tokens, i)
	}

	return tokens
}
