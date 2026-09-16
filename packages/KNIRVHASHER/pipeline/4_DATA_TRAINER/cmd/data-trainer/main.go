package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guiperry/text-embedder/pkg/embed"
	tiktoken "github.com/pkoukk/tiktoken-go"
	"knirvhasher/pkg/hashing/schema"
	"knirvhasher/pkg/hashing/semanticmemory"
	"knirvhasher/pkg/hashing/transformer"

	"github.com/lab/hasher/data-trainer/internal/loader"
)

type Config struct {
	InputPath     string
	CheckpointDir string
	NumEpochs     int
	LearningRate  float64
	SaveFreq      int
	ResumeFrom    string
	ProgressEvery time.Duration
	Mode          string
	MaxPrototypes int
	// ModelConfig overrides the default Gorgonite config. When nil,
	// DefaultGorgoniteConfig is used. Exposed for testing with small models.
	ModelConfig *transformer.GorgoniteConfig
}

func loadConfig() (*Config, error) {
	cfg := &Config{}

	flag.StringVar(&cfg.InputPath, "input", "training_frames.json", "Path to training_frames.json")
	flag.StringVar(&cfg.CheckpointDir, "checkpoint-dir", "checkpoints", "Directory for model checkpoints")
	flag.IntVar(&cfg.NumEpochs, "epochs", 3, "Number of training epochs")
	flag.Float64Var(&cfg.LearningRate, "lr", 0.0003, "Learning rate")
	flag.IntVar(&cfg.SaveFreq, "save-freq", 1, "Save checkpoint every N epochs")
	flag.StringVar(&cfg.ResumeFrom, "resume", "", "Path to checkpoint to resume from")
	flag.DurationVar(&cfg.ProgressEvery, "progress-every", 15*time.Second, "Emit training progress at this interval (0 disables)")
	flag.StringVar(&cfg.Mode, "mode", "semantic", "Training mode: semantic (safe default) or gpt (experimental, memory-intensive)")
	flag.IntVar(&cfg.MaxPrototypes, "max-prototypes", semanticmemory.DefaultMaxPrototypes, "Maximum semantic target prototypes to retain")
	flag.Parse()

	if cfg.NumEpochs <= 0 {
		return nil, fmt.Errorf("epochs must be > 0")
	}
	if cfg.LearningRate <= 0 {
		return nil, fmt.Errorf("learning rate must be > 0")
	}
	if cfg.Mode != "semantic" && cfg.Mode != "gpt" {
		return nil, fmt.Errorf("mode must be semantic or gpt")
	}
	if cfg.MaxPrototypes <= 0 {
		return nil, fmt.Errorf("max-prototypes must be > 0")
	}

	return cfg, nil
}

func buildTargetIDs(tokenIDs []int, targetTokenID int32) []int {
	targets := make([]int, len(tokenIDs))
	for i := 0; i < len(targets)-1; i++ {
		targets[i] = tokenIDs[i+1]
	}
	targets[len(targets)-1] = int(targetTokenID)
	return targets
}

func int32SliceToInt(s []int32) []int {
	out := make([]int, len(s))
	for i, v := range s {
		out[i] = int(v)
	}
	return out
}

// newTokenContextDecoder is injectable in tests. Production uses the same
// cl100k_base tokenization that data-encoder used to create TrainingFrame.
var newTokenContextDecoder = func() (func([]int32) string, error) {
	encoding, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return nil, err
	}
	return func(tokens []int32) string {
		return encoding.Decode(int32SliceToInt(tokens))
	}, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if err := Run(cfg); err != nil {
		log.Fatalf("train: %v", err)
	}
}

// Run executes the training pipeline for the given config. Exposed for testing.
func Run(cfg *Config) error {
	if cfg.Mode == "" {
		cfg.Mode = "semantic"
	}
	if cfg.MaxPrototypes == 0 {
		cfg.MaxPrototypes = semanticmemory.DefaultMaxPrototypes
	}
	if cfg.Mode == "semantic" {
		return runSemantic(cfg)
	}
	return runGPT(cfg)
}

// runGPT preserves the original full-transformer trainer behind an explicit
// opt-in. It is unsuitable for the pipeline's large frame archive because it
// constructs a massive Gorgonia graph for every frame.
func runGPT(cfg *Config) error {
	if err := os.MkdirAll(cfg.CheckpointDir, 0755); err != nil {
		return fmt.Errorf("create checkpoint dir: %w", err)
	}

	frames, err := loader.LoadFrames(cfg.InputPath)
	if err != nil {
		return fmt.Errorf("load frames: %w", err)
	}
	if len(frames) == 0 {
		return fmt.Errorf("no training frames loaded")
	}

	gptConfig := transformer.DefaultGorgoniteConfig()
	if cfg.ModelConfig != nil {
		gptConfig = cfg.ModelConfig
	}
	log.Printf("initializing model (vocab=%d embed=%d layers=%d heads=%d context=%d)",
		gptConfig.VocabSize, gptConfig.EmbedDim, gptConfig.NumLayers, gptConfig.NumHeads, gptConfig.ContextLen)
	gpt := transformer.NewGPT(gptConfig)
	log.Printf("model initialized; beginning training over %d frames", len(frames))

	if cfg.ResumeFrom != "" {
		if err := transformer.LoadModel(gpt, cfg.ResumeFrom); err != nil {
			return fmt.Errorf("resume from checkpoint: %w", err)
		}
		log.Printf("resumed from %s", cfg.ResumeFrom)
	}

	start := time.Now()
	totalSteps := 0

	for epoch := 0; epoch < cfg.NumEpochs; epoch++ {
		var epochLoss float32
		steps := 0
		skipped := 0
		epochStart := time.Now()
		lastProgress := epochStart
		log.Printf("epoch %d/%d: starting frame 1 of %d", epoch+1, cfg.NumEpochs, len(frames))

		for i, f := range frames {
			inputIDs := int32SliceToInt(f.TokenSequence)
			targetIDs := buildTargetIDs(inputIDs, f.TargetTokenID)

			loss, err := gpt.TrainStep(inputIDs, targetIDs, float32(cfg.LearningRate))
			if err != nil {
				log.Printf("frame %d (source=%s): train step error: %v", i, f.SourceFile, err)
				skipped++
				lastProgress = logTrainingProgress(epoch, cfg.NumEpochs, i+1, len(frames), steps, skipped, epochStart, lastProgress, cfg.ProgressEvery)
				continue
			}
			if math.IsNaN(float64(loss)) || math.IsInf(float64(loss), 0) {
				log.Printf("frame %d: non-finite loss %.4f, skipping", i, loss)
				skipped++
				lastProgress = logTrainingProgress(epoch, cfg.NumEpochs, i+1, len(frames), steps, skipped, epochStart, lastProgress, cfg.ProgressEvery)
				continue
			}

			epochLoss += loss
			steps++
			lastProgress = logTrainingProgress(epoch, cfg.NumEpochs, i+1, len(frames), steps, skipped, epochStart, lastProgress, cfg.ProgressEvery)
		}

		if steps == 0 {
			return fmt.Errorf("epoch %d: no valid training steps completed", epoch)
		}

		avgLoss := epochLoss / float32(steps)
		log.Printf("epoch %d/%d: avg loss=%.4f (%d steps)", epoch+1, cfg.NumEpochs, avgLoss, steps)
		totalSteps += steps

		if cfg.SaveFreq > 0 && (epoch+1)%cfg.SaveFreq == 0 {
			path := filepath.Join(cfg.CheckpointDir, fmt.Sprintf("checkpoint_epoch_%d.bin", epoch+1))
			if err := transformer.SaveModel(gpt, path); err != nil {
				log.Printf("checkpoint save failed at epoch %d: %v", epoch+1, err)
			} else {
				log.Printf("saved checkpoint: %s", path)
			}
		}
	}

	latestPath := filepath.Join(cfg.CheckpointDir, "model_latest.bin")
	if err := transformer.SaveModel(gpt, latestPath); err != nil {
		return fmt.Errorf("save latest model: %v", err)
	}

	elapsed := time.Since(start)
	log.Printf("training complete: %d epochs, %d steps, %.2fs total, checkpoint at %s",
		cfg.NumEpochs, totalSteps, elapsed.Seconds(), latestPath)
	return nil
}

// runSemantic trains a bounded semantic next-token memory in streaming mode.
// It shares the encoder's hash-ngram-v1 embedder, keeps only target centroids,
// and never materializes a transformer graph or the complete frame archive.
func runSemantic(cfg *Config) error {
	if err := os.MkdirAll(cfg.CheckpointDir, 0755); err != nil {
		return fmt.Errorf("create checkpoint dir: %w", err)
	}

	counts, err := loader.CountFrames(cfg.InputPath)
	if err != nil {
		return fmt.Errorf("count frames: %w", err)
	}
	if counts.Accepted == 0 {
		return fmt.Errorf("no training frames loaded")
	}
	log.Printf("semantic trainer: %d frames (%d empty skipped), embedder=%s dims=%d, max prototypes=%d",
		counts.Accepted, counts.Skipped, embed.ModelID, embed.Dims, cfg.MaxPrototypes)

	decodeContext, err := newTokenContextDecoder()
	if err != nil {
		return fmt.Errorf("initialize cl100k tokenizer: %w", err)
	}

	memoryPath := filepath.Join(cfg.CheckpointDir, "semantic_memory.json")
	model, err := semanticmemory.New(embed.Dims, cfg.MaxPrototypes, embed.ModelID)
	if err != nil {
		return fmt.Errorf("create semantic memory: %w", err)
	}
	if cfg.ResumeFrom != "" {
		model, err = semanticmemory.Load(cfg.ResumeFrom)
		if err != nil {
			return fmt.Errorf("resume semantic memory: %w", err)
		}
		if model.Dimensions != embed.Dims || model.Embedder != embed.ModelID {
			return fmt.Errorf("resume semantic memory is incompatible with %s", embed.ModelID)
		}
		log.Printf("resumed semantic memory from %s", cfg.ResumeFrom)
	}

	start := time.Now()
	totalSteps := 0
	for epoch := 0; epoch < cfg.NumEpochs; epoch++ {
		epochStart := time.Now()
		lastProgress := epochStart
		steps, skipped := 0, 0
		log.Printf("semantic epoch %d/%d: streaming %d frames", epoch+1, cfg.NumEpochs, counts.Accepted)

		_, err := loader.StreamFrames(cfg.InputPath, func(index int, frame schema.TrainingFrame) error {
			text := decodeContext(frame.TokenSequence)
			if strings.TrimSpace(text) == "" {
				skipped++
				return nil
			}
			if err := model.Update(embed.Embed(text), frame.TargetTokenID); err != nil {
				return fmt.Errorf("frame %d: update semantic memory: %w", index, err)
			}
			steps++
			lastProgress = logTrainingProgress(epoch, cfg.NumEpochs, index+1, counts.Accepted, steps, skipped, epochStart, lastProgress, cfg.ProgressEvery)
			return nil
		})
		if err != nil {
			return fmt.Errorf("stream semantic frames: %w", err)
		}
		if steps == 0 {
			return fmt.Errorf("semantic epoch %d: no valid training steps completed", epoch)
		}
		totalSteps += steps
		log.Printf("semantic epoch %d/%d complete: %d steps, %d skipped, %d prototypes (%d evictions)",
			epoch+1, cfg.NumEpochs, steps, skipped, len(model.Prototypes), model.Evictions)

		if cfg.SaveFreq > 0 && (epoch+1)%cfg.SaveFreq == 0 {
			if err := model.Save(memoryPath); err != nil {
				return fmt.Errorf("save semantic memory: %w", err)
			}
			log.Printf("saved semantic memory: %s", memoryPath)
		}
	}
	if err := model.Save(memoryPath); err != nil {
		return fmt.Errorf("save semantic memory: %w", err)
	}
	log.Printf("semantic training complete: %d epochs, %d steps, %.2fs total, memory at %s",
		cfg.NumEpochs, totalSteps, time.Since(start).Seconds(), memoryPath)
	return nil
}

// logTrainingProgress emits a heartbeat while a large input file is being
// processed. Training a frame can take seconds with the default model, so an
// epoch-only summary otherwise makes a healthy trainer look stalled.
func logTrainingProgress(epoch, totalEpochs, completed, totalFrames, steps, skipped int, started, last time.Time, interval time.Duration) time.Time {
	now := time.Now()
	if interval <= 0 || now.Sub(last) < interval {
		return last
	}

	elapsed := now.Sub(started)
	rate := float64(completed) / elapsed.Seconds()
	eta := time.Duration(0)
	if rate > 0 {
		eta = time.Duration(float64(totalFrames-completed) / rate * float64(time.Second))
	}
	log.Printf("epoch %d/%d progress: %d/%d frames (steps=%d skipped=%d, %.2f frames/s, ETA %s)",
		epoch+1, totalEpochs, completed, totalFrames, steps, skipped, rate, eta.Round(time.Second))
	return now
}
