package rollup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ChainReader interface {
	GetChainSnapshot(ctx context.Context) (*ChainSnapshot, error)
}

type HTTPTransactionChainReader struct {
	baseURL string
	client  *http.Client
}

func NewHTTPTransactionChainReader(baseURL string) *HTTPTransactionChainReader {
	return &HTTPTransactionChainReader{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (r *HTTPTransactionChainReader) GetChainSnapshot(ctx context.Context) (*ChainSnapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL+"/chain", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create chain snapshot request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chain snapshot: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chain snapshot request failed with status %d", resp.StatusCode)
	}

	var snapshot ChainSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		return nil, fmt.Errorf("failed to decode chain snapshot: %w", err)
	}
	return &snapshot, nil
}

type Service struct {
	reader        ChainReader
	oracleClient  OracleSettlementClient
	mu            sync.RWMutex
	lastProcessed uint64
	batches       map[string]*RollupBatch
	persistPath   string
}

func NewService(reader ChainReader, oracleClient OracleSettlementClient) *Service {
	return &Service{
		reader:       reader,
		oracleClient: oracleClient,
		batches:      make(map[string]*RollupBatch),
	}
}

type persistedState struct {
	LastProcessed uint64         `json:"last_processed"`
	Batches       []*RollupBatch `json:"batches"`
}

func (s *Service) SetPersistencePath(path string) error {
	if path == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create rollup persistence directory: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.persistPath = path
	return s.loadStateLocked()
}

func (s *Service) BuildNextBatch(ctx context.Context) (*RollupBatch, error) {
	snapshot, err := s.reader.GetChainSnapshot(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	lastProcessed := s.lastProcessed
	s.mu.RUnlock()

	var candidateBlocks []BlockRecord
	for _, block := range snapshot.Blocks {
		if block.BlockNumber > lastProcessed {
			candidateBlocks = append(candidateBlocks, block)
		}
	}

	if len(candidateBlocks) == 0 {
		return nil, nil
	}

	sort.Slice(candidateBlocks, func(i, j int) bool {
		return candidateBlocks[i].BlockNumber < candidateBlocks[j].BlockNumber
	})

	batch := &RollupBatch{
		ID:          "rollup-" + uuid.NewString(),
		ChainID:     snapshot.ChainID,
		StartHeight: candidateBlocks[0].BlockNumber,
		EndHeight:   candidateBlocks[len(candidateBlocks)-1].BlockNumber,
		Blocks:      candidateBlocks,
		Status:      StatusBuilt,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	batch.BatchRoot = computeBatchRoot(candidateBlocks)

	s.mu.Lock()
	s.batches[batch.ID] = batch
	if err := s.persistStateLocked(); err != nil {
		delete(s.batches, batch.ID)
		s.mu.Unlock()
		return nil, err
	}
	s.mu.Unlock()

	return batch, nil
}

func (s *Service) SubmitBatch(ctx context.Context, batchID string) (*RollupBatch, error) {
	s.mu.RLock()
	batch, ok := s.batches[batchID]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("rollup batch not found: %s", batchID)
	}

	settlementID, err := s.oracleClient.SubmitRollup(batch)
	if err != nil {
		s.mu.Lock()
		batch.Status = StatusFailed
		batch.Error = err.Error()
		batch.UpdatedAt = time.Now().UTC()
		s.mu.Unlock()
		return nil, err
	}

	txCount := 0
	for _, block := range batch.Blocks {
		txCount += len(block.Transactions)
	}

	s.mu.Lock()
	batch.Status = StatusSubmitted
	batch.UpdatedAt = time.Now().UTC()
	batch.Settlement = SettlementMetadata{
		BatchID:      batch.ID,
		BatchRoot:    batch.BatchRoot,
		ChainID:      batch.ChainID,
		StartHeight:  batch.StartHeight,
		EndHeight:    batch.EndHeight,
		BlockCount:   len(batch.Blocks),
		TxCount:      txCount,
		SubmittedAt:  time.Now().UTC(),
		SettlementID: settlementID,
	}
	s.lastProcessed = batch.EndHeight
	if err := s.persistStateLocked(); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.mu.Unlock()

	return batch, nil
}

func (s *Service) MarkSettled(batchID string, settledAt time.Time) (*RollupBatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	batch, ok := s.batches[batchID]
	if !ok {
		return nil, fmt.Errorf("rollup batch not found: %s", batchID)
	}

	batch.Status = StatusSettled
	batch.UpdatedAt = settledAt.UTC()
	batch.Settlement.SettledAt = settledAt.UTC()
	if err := s.persistStateLocked(); err != nil {
		return nil, err
	}
	return batch, nil
}

func (s *Service) MarkDisputed(batchID string, reason string) (*RollupBatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	batch, ok := s.batches[batchID]
	if !ok {
		return nil, fmt.Errorf("rollup batch not found: %s", batchID)
	}

	batch.Status = StatusDisputed
	batch.Error = reason
	batch.UpdatedAt = time.Now().UTC()
	if err := s.persistStateLocked(); err != nil {
		return nil, err
	}
	return batch, nil
}

func (s *Service) GetBatch(batchID string) (*RollupBatch, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	batch, ok := s.batches[batchID]
	return batch, ok
}

func (s *Service) ListBatches(limit int) []*RollupBatch {
	s.mu.RLock()
	defer s.mu.RUnlock()

	batches := make([]*RollupBatch, 0, len(s.batches))
	for _, batch := range s.batches {
		copyBatch := *batch
		copyBatch.Blocks = append([]BlockRecord(nil), batch.Blocks...)
		batches = append(batches, &copyBatch)
	}

	sort.Slice(batches, func(i, j int) bool {
		return batches[i].CreatedAt.After(batches[j].CreatedAt)
	})

	if limit > 0 && len(batches) > limit {
		batches = batches[:limit]
	}

	return batches
}

func (s *Service) GetStatus() ServiceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := ServiceStatus{
		LastProcessed:   s.lastProcessed,
		BatchCount:      len(s.batches),
		BatchesByStatus: make(map[BatchStatus]int),
	}

	var latest *RollupBatch
	for _, batch := range s.batches {
		status.BatchesByStatus[batch.Status]++
		if latest == nil || batch.UpdatedAt.After(latest.UpdatedAt) {
			latest = batch
		}
	}

	if latest != nil {
		status.LatestBatchID = latest.ID
		status.LatestBatchStatus = latest.Status
		status.LatestUpdatedAt = latest.UpdatedAt
	}

	return status
}

func (s *Service) ReconcileWithOracle(ctx context.Context) (int, error) {
	s.mu.RLock()
	candidateIDs := make([]string, 0, len(s.batches))
	for id, batch := range s.batches {
		if batch.Status == StatusSubmitted {
			candidateIDs = append(candidateIDs, id)
		}
	}
	s.mu.RUnlock()

	updated := 0
	for _, batchID := range candidateIDs {
		select {
		case <-ctx.Done():
			return updated, ctx.Err()
		default:
		}

		record, err := s.oracleClient.GetRollup(batchID)
		if err != nil {
			return updated, fmt.Errorf("failed to fetch oracle rollup %s: %w", batchID, err)
		}

		status, _ := record["status"].(string)
		switch strings.ToLower(status) {
		case "finalized":
			settledAt := extractTimestamp(record["finalized_at"], time.Now().UTC())
			if _, err := s.MarkSettled(batchID, settledAt); err != nil {
				return updated, fmt.Errorf("failed to mark batch %s settled: %w", batchID, err)
			}
			updated++
		case "disputed":
			reason, _ := record["dispute"].(string)
			if reason == "" {
				reason = "oracle marked batch as disputed"
			}
			if _, err := s.MarkDisputed(batchID, reason); err != nil {
				return updated, fmt.Errorf("failed to mark batch %s disputed: %w", batchID, err)
			}
			updated++
		}
	}

	return updated, nil
}

func computeBatchRoot(blocks []BlockRecord) string {
	payload, _ := json.Marshal(blocks)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func (s *Service) loadStateLocked() error {
	if s.persistPath == "" {
		return nil
	}

	data, err := os.ReadFile(s.persistPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read rollup state: %w", err)
	}

	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to decode rollup state: %w", err)
	}

	s.lastProcessed = state.LastProcessed
	s.batches = make(map[string]*RollupBatch, len(state.Batches))
	for _, batch := range state.Batches {
		if batch == nil {
			continue
		}
		s.batches[batch.ID] = batch
	}

	return nil
}

func (s *Service) persistStateLocked() error {
	if s.persistPath == "" {
		return nil
	}

	batches := make([]*RollupBatch, 0, len(s.batches))
	for _, batch := range s.batches {
		batches = append(batches, batch)
	}

	sort.Slice(batches, func(i, j int) bool {
		return batches[i].CreatedAt.Before(batches[j].CreatedAt)
	})

	payload, err := json.MarshalIndent(persistedState{
		LastProcessed: s.lastProcessed,
		Batches:       batches,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode rollup state: %w", err)
	}

	tempPath := s.persistPath + ".tmp"
	if err := os.WriteFile(tempPath, payload, 0644); err != nil {
		return fmt.Errorf("failed to write rollup state temp file: %w", err)
	}

	if err := os.Rename(tempPath, s.persistPath); err != nil {
		return fmt.Errorf("failed to move rollup state into place: %w", err)
	}

	return nil
}

func extractTimestamp(value interface{}, fallback time.Time) time.Time {
	switch v := value.(type) {
	case string:
		if parsed, err := time.Parse(time.RFC3339, v); err == nil {
			return parsed.UTC()
		}
	case float64:
		return time.Unix(int64(v), 0).UTC()
	case json.Number:
		if seconds, err := strconv.ParseInt(v.String(), 10, 64); err == nil {
			return time.Unix(seconds, 0).UTC()
		}
	}

	return fallback.UTC()
}
