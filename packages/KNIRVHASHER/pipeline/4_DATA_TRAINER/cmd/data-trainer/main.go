package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"time"

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
	flag.Parse()

	if cfg.NumEpochs <= 0 {
		return nil, fmt.Errorf("epochs must be > 0")
	}
	if cfg.LearningRate <= 0 {
		return nil, fmt.Errorf("learning rate must be > 0")
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
	gpt := transformer.NewGPT(gptConfig)

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

		for i, f := range frames {
			inputIDs := int32SliceToInt(f.TokenSequence)
			targetIDs := buildTargetIDs(inputIDs, f.TargetTokenID)

			loss, err := gpt.TrainStep(inputIDs, targetIDs, float32(cfg.LearningRate))
			if err != nil {
				log.Printf("frame %d (source=%s): train step error: %v", i, f.SourceFile, err)
				continue
			}
			if math.IsNaN(float64(loss)) || math.IsInf(float64(loss), 0) {
				log.Printf("frame %d: non-finite loss %.4f, skipping", i, loss)
				continue
			}

			epochLoss += loss
			steps++
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
