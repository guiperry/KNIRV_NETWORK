package query

import (
	"KNIRVGRAPH/internal/embeddings"
	"KNIRVGRAPH/internal/indexing"
	"KNIRVGRAPH/internal/retrieval"
	"KNIRVGRAPH/internal/synthesis"
	"KNIRVGRAPH/internal/types"
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type QueryProcessor struct {
	embeddingService *embeddings.EmbeddingService
	indexManager     *indexing.IndexManager
	pipeline         *retrieval.RetrievalPipeline
	synthesizer      *synthesis.Synthesizer
	reranker         *retrieval.Reranker
	logger           *zap.Logger
}

func NewQueryProcessor(
	embeddingService *embeddings.EmbeddingService,
	indexManager *indexing.IndexManager,
	pipeline *retrieval.RetrievalPipeline,
	synthesizer *synthesis.Synthesizer,
	reranker *retrieval.Reranker,
	logger *zap.Logger,
) *QueryProcessor {
	return &QueryProcessor{
		embeddingService: embeddingService,
		indexManager:     indexManager,
		pipeline:         pipeline,
		synthesizer:      synthesizer,
		reranker:         reranker,
		logger:           logger,
	}
}

func (p *QueryProcessor) Process(ctx context.Context, req types.QueryRequest) (*types.QueryResponse, error) {
	start := time.Now()
	if p.embeddingService == nil {
		return nil, fmt.Errorf("embedding service is not configured")
	}
	if p.pipeline == nil {
		return nil, fmt.Errorf("retrieval pipeline is not configured")
	}
	queryVec, err := p.embeddingService.Embed(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	results, err := p.pipeline.SearchFiltered(req.Query, queryVec, req.TopK, req.Filters)
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}
	if req.UseRerank && p.reranker != nil {
		results, err = p.reranker.Rerank(ctx, req.Query, results, req.TopK)
		if err != nil {
			p.logger.Warn("rerank failed, using original results", zap.Error(err))
		}
	}
	var answer string
	var reasoning string
	var sources []string
	var confidence float64
	if req.Synthesize && p.synthesizer != nil {
		synthReq := types.SynthesisRequest{
			Query:    req.Query,
			Contexts: []types.RetrievalResult{{Query: req.Query, Results: results}},
		}
		resp, err := p.synthesizer.Synthesize(ctx, synthReq)
		if err == nil {
			answer = resp.Answer
			reasoning = resp.Reasoning
			sources = resp.Sources
			confidence = resp.Confidence
		} else {
			p.logger.Warn("synthesis failed", zap.Error(err))
		}
	}
	latency := time.Since(start).Milliseconds()
	return &types.QueryResponse{
		Answer:     answer,
		Reasoning:  reasoning,
		Results:    []types.RetrievalResult{{Query: req.Query, Results: results}},
		Sources:    sources,
		Confidence: confidence,
		LatencyMs:  latency,
	}, nil
}
