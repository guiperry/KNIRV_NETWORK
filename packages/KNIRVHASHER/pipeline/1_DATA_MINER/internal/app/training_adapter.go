package app

import "fmt"

// UnifiedTrainingAdapter routes SourceRecords to the appropriate downstream handler
// based on their SourceType.
type UnifiedTrainingAdapter struct {
	handlers map[SourceType]RecordHandler
}

// RecordHandler processes a single SourceRecord.
type RecordHandler func(record *SourceRecord) error

// NewUnifiedTrainingAdapter creates an adapter with no handlers registered.
func NewUnifiedTrainingAdapter() *UnifiedTrainingAdapter {
	return &UnifiedTrainingAdapter{
		handlers: make(map[SourceType]RecordHandler),
	}
}

// Register attaches a handler for a specific SourceType.
func (a *UnifiedTrainingAdapter) Register(src SourceType, h RecordHandler) {
	a.handlers[src] = h
}

// Route dispatches record to the registered handler for its SourceType.
// Falls back to the SourceTypeUnknown handler if no exact match is found.
func (a *UnifiedTrainingAdapter) Route(record *SourceRecord) error {
	if record == nil {
		return fmt.Errorf("nil record")
	}
	if h, ok := a.handlers[record.Source]; ok {
		return h(record)
	}
	if h, ok := a.handlers[SourceTypeUnknown]; ok {
		return h(record)
	}
	return fmt.Errorf("no handler registered for source type %q", record.Source)
}
