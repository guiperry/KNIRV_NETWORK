package app

import "fmt"

// TrainingPair is a (input, target) pair ready for the trainer.
type TrainingPair struct {
	Input    DocumentRecord `json:"input"`
	Target   DocumentRecord `json:"target"`
	Weight   float32        `json:"weight"`
	SourceID string         `json:"source_id"`
}

// PairHandler converts a SourceRecord into a TrainingPair.
type PairHandler func(record *SourceRecord) (TrainingPair, error)

// UnifiedTrainingAdapter routes SourceRecords to the appropriate downstream handler
// based on their SourceType.
type UnifiedTrainingAdapter struct {
	handlers map[SourceType]PairHandler
}

// NewUnifiedTrainingAdapter creates an adapter with no handlers registered.
func NewUnifiedTrainingAdapter() *UnifiedTrainingAdapter {
	return &UnifiedTrainingAdapter{
		handlers: make(map[SourceType]PairHandler),
	}
}

// Register attaches a handler for a specific SourceType.
func (a *UnifiedTrainingAdapter) Register(src SourceType, h PairHandler) {
	a.handlers[src] = h
}

// Route dispatches record to the registered handler and returns a TrainingPair.
// Falls back to the SourceTypeUnknown handler if no exact match is found.
func (a *UnifiedTrainingAdapter) Route(record *SourceRecord) (TrainingPair, error) {
	if record == nil {
		return TrainingPair{}, fmt.Errorf("nil record")
	}
	if h, ok := a.handlers[record.Source]; ok {
		return h(record)
	}
	if h, ok := a.handlers[SourceTypeUnknown]; ok {
		return h(record)
	}
	return TrainingPair{}, fmt.Errorf("no handler registered for source type %q", record.Source)
}
