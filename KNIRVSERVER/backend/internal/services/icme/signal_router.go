package icme

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ActiveMemoryInterface interface {
	RecordIntentionalSignal(ctx context.Context, signal *IntentionalSignal) error
}

type SignalRouter struct {
	intentRegistry *IntentRegistry
	nerClient      *NERProvider
	embedProvider  *EmbeddingProvider
	graphEngine    *TemporalHypergraph
	indexManager   *FAISSIndexManager
	activeMemory   ActiveMemoryInterface
	logger         *zap.Logger
	signalCh       chan *IntentionalSignal
}

func NewSignalRouter(
	intentRegistry *IntentRegistry,
	nerClient *NERProvider,
	embedProvider *EmbeddingProvider,
	graphEngine *TemporalHypergraph,
	indexManager *FAISSIndexManager,
	activeMemory ActiveMemoryInterface,
	logger *zap.Logger,
) *SignalRouter {
	r := &SignalRouter{
		intentRegistry: intentRegistry,
		nerClient:      nerClient,
		embedProvider:  embedProvider,
		graphEngine:    graphEngine,
		indexManager:   indexManager,
		activeMemory:   activeMemory,
		logger:         logger,
		signalCh:       make(chan *IntentionalSignal, 512),
	}
	return r
}

func (r *SignalRouter) Start(ctx context.Context) {
	for i := 0; i < 4; i++ {
		go r.processLoop(ctx)
	}
}

func (r *SignalRouter) Ingest(agentID, dveID string, source SignalSource, content string) {
	sig := &IntentionalSignal{
		ID:        uuid.NewString(),
		AgentID:   agentID,
		DVEID:     dveID,
		Source:    source,
		Content:   content,
		Timestamp: time.Now(),
	}

	select {
	case r.signalCh <- sig:
	default:
		r.logger.Warn("icme signal channel full, dropping signal",
			zap.String("agent_id", agentID),
			zap.String("source", string(source)),
		)
	}
}

func (r *SignalRouter) processLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case sig := <-r.signalCh:
			if err := r.enrich(ctx, sig); err != nil {
				r.logger.Error("icme signal enrichment failed",
					zap.String("signal_id", sig.ID),
					zap.Error(err),
				)
			}
		}
	}
}

func (r *SignalRouter) enrich(ctx context.Context, sig *IntentionalSignal) error {
	sig.Scope = ScopeGlobal
	if sig.DVEID != "" {
		obj := r.intentRegistry.GetObjectiveForDVE(sig.AgentID, sig.DVEID)
		if obj != nil {
			sig.Scope = ScopeDVE
			sig.ObjectiveName = obj.Name
			sig.AuthorizedActs = obj.AuthorizedActions
			sig.TradeOffWeights = obj.TradeOffs
			sig.HardBoundaries = obj.HardBoundaries
		}
	}

	if sig.ObjectiveName == "" {
		obj := r.intentRegistry.GetGlobalObjectiveForAgent(sig.AgentID)
		if obj != nil {
			sig.ObjectiveName = obj.Name
			sig.AuthorizedActs = obj.AuthorizedActions
			sig.TradeOffWeights = obj.TradeOffs
			sig.HardBoundaries = obj.HardBoundaries
		}
	}

	ents, rels, err := r.nerClient.ExtractEntitiesAndRelations(sig.Content)
	if err != nil {
		r.logger.Debug("icme ner skipped", zap.Error(err))
	} else {
		sig.Entities = ents
		sig.Relations = rels
	}

	embedding, err := r.embedProvider.GetEmbedding(sig.Content)
	if err != nil {
		return fmt.Errorf("embedding generation: %w", err)
	}

	summary := sig.Content
	if len(summary) > 200 {
		summary = summary[:200]
	}
	vecID, err := r.indexManager.Add(sig.ID, embedding, VectorMeta{
		SignalID:  sig.ID,
		AgentID:   sig.AgentID,
		Summary:   summary,
		Objective: sig.ObjectiveName,
		DVEID:     sig.DVEID,
	})
	if err != nil {
		return fmt.Errorf("faiss index add: %w", err)
	}
	sig.EmbeddingID = vecID

	r.graphEngine.InsertSignal(sig)

	if r.activeMemory != nil {
		r.activeMemory.RecordIntentionalSignal(ctx, sig)
	}

	r.logger.Debug("icme signal enriched",
		zap.String("signal_id", sig.ID),
		zap.String("objective", sig.ObjectiveName),
		zap.Int("entities", len(sig.Entities)),
		zap.Int64("embedding_id", sig.EmbeddingID),
	)
	return nil
}
