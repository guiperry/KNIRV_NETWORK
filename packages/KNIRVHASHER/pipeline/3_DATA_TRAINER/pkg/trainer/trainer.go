package trainer

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"math"
	"strings"

	"github.com/lab/hasher/data-trainer/internal/config"
)

// Trainer implements curriculum-based training with stage-gated data selection.
type Trainer struct {
	config       *config.CurriculumConfig
	currentStage config.CurriculumStage
	epoch        int
	loss         float64
	dataSelector func(stage config.CurriculumStage, data []TrainingSample) []TrainingSample
}

// TrainingSample represents a single training data point.
type TrainingSample struct {
	ID    string
	Input []float64
	Label float64
	Difficulty float64 // Used for curriculum gating
}

// NewTrainer creates a new curriculum trainer with the given config.
func NewTrainer(cfg *config.CurriculumConfig) *Trainer {
	if cfg == nil {
		cfg = &config.CurriculumConfig{
			Stage:          config.CurriculumApprentice,
			LearningRate:   0.01,
			AdvanceTrigger: "loss<0.5",
			MaxEpochs:      100,
		}
	}
	return &Trainer{
		config:       cfg,
		currentStage: cfg.Stage,
		epoch:        0,
		loss:         math.Inf(1),
		dataSelector: defaultDataSelector,
	}
}

// SetDataSelector sets a custom function for stage-gated data selection.
func (t *Trainer) SetDataSelector(selector func(stage config.CurriculumStage, data []TrainingSample) []TrainingSample) {
	t.dataSelector = selector
}

// TrainEpoch runs one epoch of curriculum-gated training.
// Returns the loss for this epoch and whether the stage should advance.
func (t *Trainer) TrainEpoch(data []TrainingSample) (float64, bool, error) {
	t.epoch++

	// Stage-gated data selection: filter data based on current stage
	selected := t.dataSelector(t.currentStage, data)
	if len(selected) == 0 {
		return 0, false, fmt.Errorf("no training data available for stage %s", t.currentStage)
	}

	// Simulate training: compute average loss on selected data
	// In production, this would actually run the model forward/backward pass
	loss := t.simulateTraining(selected)
	t.loss = loss

	// Check if we should advance to next stage
	shouldAdvance := t.shouldAdvanceStage(loss)

	return loss, shouldAdvance, nil
}

// shouldAdvanceStage checks if the advance trigger condition is met.
func (t *Trainer) shouldAdvanceStage(loss float64) bool {
	trigger := t.config.AdvanceTrigger
	if trigger == "" {
		return false
	}

	// Parse simple trigger conditions like "loss<0.5" or "epoch>10"
	// For now, support "loss<X" format
	if strings.HasPrefix(trigger, "loss<") {
		var threshold float64
		_, err := fmt.Sscanf(trigger, "loss<%f", &threshold)
		if err == nil && loss < threshold {
			return true
		}
	}
	return false
}

// AdvanceStage moves to the next curriculum stage.
// Returns the new stage and false if already at the highest stage.
func (t *Trainer) AdvanceStage() (config.CurriculumStage, bool) {
	switch t.currentStage {
	case config.CurriculumApprentice:
		t.currentStage = config.CurriculumJourneyman
		t.config.Stage = t.currentStage
		return t.currentStage, true
	case config.CurriculumJourneyman:
		t.currentStage = config.CurriculumExpert
		t.config.Stage = t.currentStage
		return t.currentStage, true
	case config.CurriculumExpert:
		return t.currentStage, false // Already at highest stage
	default:
		return t.currentStage, false
	}
}

// GetCurrentStage returns the current curriculum stage.
func (t *Trainer) GetCurrentStage() config.CurriculumStage {
	return t.currentStage
}

// GetEpoch returns the current epoch number.
func (t *Trainer) GetEpoch() int {
	return t.epoch
}

// GetLoss returns the current loss.
func (t *Trainer) GetLoss() float64 {
	return t.loss
}

// defaultDataSelector implements stage-gated data selection.
// - Apprentice: uses easy samples (low difficulty)
// - Journeyman: uses medium+ difficulty samples
// - Expert: uses all samples
func defaultDataSelector(stage config.CurriculumStage, data []TrainingSample) []TrainingSample {
	var filtered []TrainingSample
	for _, sample := range data {
		switch stage {
		case config.CurriculumApprentice:
			if sample.Difficulty <= 0.3 {
				filtered = append(filtered, sample)
			}
		case config.CurriculumJourneyman:
			if sample.Difficulty <= 0.7 {
				filtered = append(filtered, sample)
			}
		case config.CurriculumExpert:
			filtered = append(filtered, sample) // All data
		default:
			filtered = append(filtered, sample)
		}
	}
	return filtered
}

// simulateTraining simulates one training epoch.
// In production, this would integrate with the actual model.
func (t *Trainer) simulateTraining(data []TrainingSample) float64 {
	if len(data) == 0 {
		return 0
	}
	// Simple simulation: loss decreases with more data and easier samples
	// and as we progress through epochs
	baseLoss := 1.0
	difficultySum := 0.0
	for _, d := range data {
		difficultySum += d.Difficulty
	}
	avgDifficulty := difficultySum / float64(len(data))

	// Loss decreases with more epochs and easier data
	epochFactor := math.Exp(-float64(t.epoch) * 0.01)
	difficultyFactor := 1.0 - avgDifficulty*0.5

	loss := baseLoss * epochFactor * difficultyFactor
	if loss < 0.01 {
		loss = 0.01 // Minimum loss
	}
	return loss
}

// HashNetwork provides hashing utilities for the trainer.
type HashNetwork struct{}

// SHA256Hash computes the SHA256 hash of various input types.
func SHA256Hash(input interface{}) [32]byte {
	var data []byte
	switch v := input.(type) {
	case string:
		data = []byte(v)
	case []float64:
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		enc.Encode(v)
		data = buf.Bytes()
	case []byte:
		data = v
	default:
		data = []byte{}
	}
	return sha256.Sum256(data)
}
