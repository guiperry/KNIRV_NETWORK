package icme

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type AlignmentLoop struct {
	registry          *IntentRegistry
	factualityAdapter *FactualityAdapter
	logger            *zap.Logger
	evalInterval      time.Duration
	driftThreshold    float64
}

func NewAlignmentLoop(
	registry *IntentRegistry,
	factualityAdapter *FactualityAdapter,
	evalInterval time.Duration,
	driftThreshold float64,
	logger *zap.Logger,
) *AlignmentLoop {
	return &AlignmentLoop{
		registry:          registry,
		factualityAdapter: factualityAdapter,
		evalInterval:      evalInterval,
		driftThreshold:    driftThreshold,
		logger:            logger,
	}
}

func (l *AlignmentLoop) Start(ctx context.Context) {
	ticker := time.NewTicker(l.evalInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.runEvaluation(ctx)
		}
	}
}

func (l *AlignmentLoop) runEvaluation(ctx context.Context) {
	objectives := l.registry.ListObjectives("")
	for _, obj := range objectives {
		records := l.registry.GetRecentAlignmentRecords(obj.Name, 10)
		if len(records) < 2 {
			continue
		}

		var drift float64
		for i := 1; i < len(records); i++ {
			drift += records[i].AlignmentScore - records[i-1].AlignmentScore
		}
		drift /= float64(len(records) - 1)

		if drift < -l.driftThreshold {
			l.logger.Warn("alignment drift detected",
				zap.String("objective", obj.Name),
				zap.Float64("drift", drift),
			)
		}
	}
}

func (l *AlignmentLoop) Evaluate(ctx context.Context, signal *IntentionalSignal) (*AlignmentRecord, error) {
	alignScore, err := l.factualityAdapter.ComputeAlignmentScore(ctx, signal)
	if err != nil {
		return nil, err
	}

	obj := l.registry.GetObjectiveForAgent(signal.AgentID, signal.DVEID)
	decision := "approved"
	if obj != nil {
		authorized := false
		for _, a := range obj.AuthorizedActions {
			if a == signal.Content {
				authorized = true
				break
			}
		}
		if !authorized {
			decision = "escalated"
			alignScore *= 0.8
		}
	}

	rec := &AlignmentRecord{
		ID:             "",
		AgentID:        signal.AgentID,
		DVEID:          signal.DVEID,
		ObjectiveName:  signal.ObjectiveName,
		SignalID:       signal.ID,
		Decision:       decision,
		Outcome:        "completed",
		AlignmentScore: alignScore,
		FidelityScore:  alignScore,
		Timestamp:      time.Now(),
	}

	if err := l.registry.RecordAlignment(rec); err != nil {
		l.logger.Error("failed to record alignment", zap.Error(err))
	}

	return rec, nil
}
