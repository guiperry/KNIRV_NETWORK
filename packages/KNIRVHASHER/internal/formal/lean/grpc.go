package lean

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	pb "knirvhasher/internal/proto/hasher/v1"
	"knirvhasher/pkg/hashing/proofasset"
)

func receiptToProto(r *proofasset.VerificationReceipt) *pb.VerificationReceipt {
	if r == nil {
		return nil
	}
	return &pb.VerificationReceipt{
		SchemaVersion:     r.SchemaVersion,
		ProofAssetId:      r.ProofAssetID,
		Status:            r.Status,
		CheckerDigest:     r.CheckerDigest,
		EnvironmentDigest: r.EnvironmentDigest,
		CheckedAt:         r.CheckedAt.Format(time.RFC3339Nano),
		DiagnosticDigest:  r.DiagnosticDigest,
	}
}

func assetToProto(a *proofasset.ProofAsset) (*pb.SubmitProofRequest, error) {
	canonical, err := proofasset.CanonicalProofAssetBytes(a)
	if err != nil {
		return nil, err
	}
	return &pb.SubmitProofRequest{
		CanonicalProofAsset: canonical,
	}, nil
}

// GRPCServer implements the FormalVerificationService gRPC service.
type GRPCServer struct {
	pb.UnimplementedFormalVerificationServiceServer
	worker    *Worker
	store     ProofAssetStore
	receipts  map[string]*proofasset.VerificationReceipt
	assets    map[string]*proofasset.ProofAsset
	metrics   *VerificationMetrics
	mu        sync.RWMutex
}

// ProofAssetStore persists and retrieves proof assets and receipts.
type ProofAssetStore interface {
	StoreProofAsset(asset *proofasset.ProofAsset, receipt *proofasset.VerificationReceipt) error
	GetProofAsset(id string) (*proofasset.ProofAsset, *proofasset.VerificationReceipt, error)
	ListProofAssets() ([]*proofasset.ProofAsset, error)
}

// FileSystemProofAssetStore persists proof assets to the filesystem.
type FileSystemProofAssetStore struct {
	baseDir string
	mu      sync.RWMutex
}

func NewFileSystemProofAssetStore(baseDir string) (*FileSystemProofAssetStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create proof asset store directory: %w", err)
	}
	return &FileSystemProofAssetStore{baseDir: baseDir}, nil
}

func (s *FileSystemProofAssetStore) StoreProofAsset(asset *proofasset.ProofAsset, receipt *proofasset.VerificationReceipt) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := proofasset.ComputeProofAssetID(asset)
	if err != nil {
		return fmt.Errorf("compute proof asset ID: %w", err)
	}

	assetPath := filepath.Join(s.baseDir, id+".json")
	assetData, err := json.MarshalIndent(asset, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal proof asset: %w", err)
	}
	if err := os.WriteFile(assetPath, assetData, 0644); err != nil {
		return fmt.Errorf("write proof asset: %w", err)
	}

	if receipt != nil {
		receiptPath := filepath.Join(s.baseDir, id+"_receipt.json")
		receiptData, err := json.MarshalIndent(receipt, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal receipt: %w", err)
		}
		if err := os.WriteFile(receiptPath, receiptData, 0644); err != nil {
			return fmt.Errorf("write receipt: %w", err)
		}
	}

	return nil
}

func (s *FileSystemProofAssetStore) GetProofAsset(id string) (*proofasset.ProofAsset, *proofasset.VerificationReceipt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	assetPath := filepath.Join(s.baseDir, id+".json")
	data, err := os.ReadFile(assetPath)
	if err != nil {
		return nil, nil, err
	}
	var asset proofasset.ProofAsset
	if err := json.Unmarshal(data, &asset); err != nil {
		return nil, nil, err
	}

	receiptPath := filepath.Join(s.baseDir, id+"_receipt.json")
	receiptData, err := os.ReadFile(receiptPath)
	if err != nil {
		return &asset, nil, nil
	}
	var receipt proofasset.VerificationReceipt
	if err := json.Unmarshal(receiptData, &receipt); err != nil {
		return &asset, nil, nil
	}

	return &asset, &receipt, nil
}

func (s *FileSystemProofAssetStore) ListProofAssets() ([]*proofasset.ProofAsset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, err
	}

	var assets []*proofasset.ProofAsset
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" || filepath.Ext(entry.Name()) == "_receipt.json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.baseDir, entry.Name()))
		if err != nil {
			continue
		}
		var asset proofasset.ProofAsset
		if err := json.Unmarshal(data, &asset); err != nil {
			continue
		}
		assets = append(assets, &asset)
	}

	return assets, nil
}

// VerificationMetrics tracks verification queue and execution metrics.
type VerificationMetrics struct {
	TotalSubmissions  uint64
	VerifiedCount     uint64
	RejectedCount     uint64
	ErrorCount        uint64
	QueueDepth        uint64
	MedianCheckTimeMs float64
	CheckTimes        []float64
	mu                sync.RWMutex
}

func (m *VerificationMetrics) RecordCheck(durationMs float64, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalSubmissions++
	m.CheckTimes = append(m.CheckTimes, durationMs)
	if len(m.CheckTimes) > 1000 {
		m.CheckTimes = m.CheckTimes[len(m.CheckTimes)-1000:]
	}
	switch status {
	case proofasset.StatusFormallyVerified:
		m.VerifiedCount++
	case proofasset.StatusFormallyRejected:
		m.RejectedCount++
	default:
		m.ErrorCount++
	}
	if len(m.CheckTimes) > 0 {
		m.MedianCheckTimeMs = median(m.CheckTimes)
	}
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

// NewGRPCServer creates a new FormalVerificationService gRPC server.
func NewGRPCServer(worker *Worker, store ProofAssetStore) *GRPCServer {
	if store == nil {
		store = &MemoryProofAssetStore{}
	}
	return &GRPCServer{
		worker:   worker,
		store:    store,
		receipts: make(map[string]*proofasset.VerificationReceipt),
		assets:   make(map[string]*proofasset.ProofAsset),
		metrics:  &VerificationMetrics{},
	}
}

// MemoryProofAssetStore is an in-memory proof asset store for testing.
type MemoryProofAssetStore struct {
	assets  map[string]*proofasset.ProofAsset
	receipts map[string]*proofasset.VerificationReceipt
	mu      sync.RWMutex
}

func NewMemoryProofAssetStore() *MemoryProofAssetStore {
	return &MemoryProofAssetStore{
		assets:   make(map[string]*proofasset.ProofAsset),
		receipts: make(map[string]*proofasset.VerificationReceipt),
	}
}

func (s *MemoryProofAssetStore) StoreProofAsset(asset *proofasset.ProofAsset, receipt *proofasset.VerificationReceipt) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, err := proofasset.ComputeProofAssetID(asset)
	if err != nil {
		return err
	}
	cp := *asset
	s.assets[id] = &cp
	if receipt != nil {
		rc := *receipt
		s.receipts[id] = &rc
	}
	return nil
}

func (s *MemoryProofAssetStore) GetProofAsset(id string) (*proofasset.ProofAsset, *proofasset.VerificationReceipt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	asset, ok := s.assets[id]
	if !ok {
		return nil, nil, fmt.Errorf("proof asset not found: %s", id)
	}
	receipt := s.receipts[id]
	return asset, receipt, nil
}

func (s *MemoryProofAssetStore) ListProofAssets() ([]*proofasset.ProofAsset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var assets []*proofasset.ProofAsset
	for _, a := range s.assets {
		cp := *a
		assets = append(assets, &cp)
	}
	return assets, nil
}

// SubmitProof implements the gRPC SubmitProof RPC.
func (s *GRPCServer) SubmitProof(ctx context.Context, req *pb.SubmitProofRequest) (*pb.SubmitProofResponse, error) {
	s.metrics.mu.Lock()
	s.metrics.QueueDepth++
	s.metrics.mu.Unlock()
	defer func() {
		s.metrics.mu.Lock()
		s.metrics.QueueDepth--
		s.metrics.mu.Unlock()
	}()

	if s.worker == nil {
		s.metrics.RecordCheck(0, proofasset.StatusCheckerUnavailable)
		return &pb.SubmitProofResponse{
			ProofAssetId: "",
			Status:       pb.FormalProofStatus_CHECKER_UNAVAILABLE,
			Diagnostic:   "formal verifier worker not initialized",
		}, nil
	}

	var asset proofasset.ProofAsset
	if err := json.Unmarshal(req.CanonicalProofAsset, &asset); err != nil {
		s.metrics.RecordCheck(0, proofasset.StatusCheckerUnavailable)
		return &pb.SubmitProofResponse{
			ProofAssetId: "",
			Status:       pb.FormalProofStatus_CHECKER_UNAVAILABLE,
			Diagnostic:   fmt.Sprintf("invalid proof asset JSON: %v", err),
		}, nil
	}

	start := time.Now()
	result, err := s.worker.SubmitProof(&asset)
	durationMs := float64(time.Since(start).Milliseconds())

	if err != nil {
		s.metrics.RecordCheck(durationMs, proofasset.StatusCheckerUnavailable)
		return &pb.SubmitProofResponse{
			ProofAssetId: "",
			Status:       pb.FormalProofStatus_CHECKER_UNAVAILABLE,
			Diagnostic:   fmt.Sprintf("checker error: %v", err),
		}, nil
	}

	proofAssetID, _ := proofasset.ComputeProofAssetID(&asset)

	status := pb.FormalProofStatus_PROOF_PENDING
	switch result.Receipt.Status {
	case proofasset.StatusFormallyVerified:
		status = pb.FormalProofStatus_FORMALLY_VERIFIED
	case proofasset.StatusFormallyRejected:
		status = pb.FormalProofStatus_FORMALLY_REJECTED
	case proofasset.StatusCheckerUnavailable:
		status = pb.FormalProofStatus_CHECKER_UNAVAILABLE
	case proofasset.StatusProofPending:
		status = pb.FormalProofStatus_PROOF_PENDING
	}

	s.metrics.RecordCheck(durationMs, result.Receipt.Status)

	s.mu.Lock()
	s.assets[proofAssetID] = &asset
	s.receipts[proofAssetID] = result.Receipt
	s.mu.Unlock()

	if err := s.store.StoreProofAsset(&asset, result.Receipt); err != nil {
		log.Printf("WARNING: failed to store proof asset: %v", err)
	}

	return &pb.SubmitProofResponse{
		ProofAssetId: proofAssetID,
		Status:       status,
		Diagnostic:   result.Diagnostic,
	}, nil
}

// GetProofStatus implements the gRPC GetProofStatus RPC.
func (s *GRPCServer) GetProofStatus(ctx context.Context, req *pb.GetProofStatusRequest) (*pb.GetProofStatusResponse, error) {
	s.mu.RLock()
	receipt, ok := s.receipts[req.ProofAssetId]
	s.mu.RUnlock()

	if !ok {
		asset, r, err := s.store.GetProofAsset(req.ProofAssetId)
		if err != nil {
			return &pb.GetProofStatusResponse{
				ProofAssetId: req.ProofAssetId,
				Status:       pb.FormalProofStatus_FORMAL_PROOF_UNKNOWN,
			}, nil
		}
		if r != nil {
			receipt = r
			s.mu.Lock()
			s.receipts[req.ProofAssetId] = receipt
			if asset != nil {
				s.assets[req.ProofAssetId] = asset
			}
			s.mu.Unlock()
		}
	}

	if receipt == nil {
		return &pb.GetProofStatusResponse{
			ProofAssetId: req.ProofAssetId,
			Status:       pb.FormalProofStatus_FORMAL_PROOF_UNKNOWN,
		}, nil
	}

	status := pb.FormalProofStatus_PROOF_PENDING
	switch receipt.Status {
	case proofasset.StatusFormallyVerified:
		status = pb.FormalProofStatus_FORMALLY_VERIFIED
	case proofasset.StatusFormallyRejected:
		status = pb.FormalProofStatus_FORMALLY_REJECTED
	case proofasset.StatusCheckerUnavailable:
		status = pb.FormalProofStatus_CHECKER_UNAVAILABLE
	case proofasset.StatusProofPending:
		status = pb.FormalProofStatus_PROOF_PENDING
	}

	return &pb.GetProofStatusResponse{
		ProofAssetId: req.ProofAssetId,
		Status:       status,
		Receipt:      receiptToProto(receipt),
	}, nil
}

// GetProofAsset implements the gRPC GetProofAsset RPC.
func (s *GRPCServer) GetProofAsset(ctx context.Context, req *pb.GetProofAssetRequest) (*pb.GetProofAssetResponse, error) {
	s.mu.RLock()
	asset, ok := s.assets[req.ProofAssetId]
	s.mu.RUnlock()

	if !ok {
		a, _, err := s.store.GetProofAsset(req.ProofAssetId)
		if err != nil {
			return &pb.GetProofAssetResponse{
				ProofAssetId: req.ProofAssetId,
				Status:       pb.FormalProofStatus_FORMAL_PROOF_UNKNOWN,
			}, nil
		}
		asset = a
		if asset != nil {
			s.mu.Lock()
			s.assets[req.ProofAssetId] = asset
			s.mu.Unlock()
		}
	}

	if asset == nil {
		return &pb.GetProofAssetResponse{
			ProofAssetId: req.ProofAssetId,
			Status:       pb.FormalProofStatus_FORMAL_PROOF_UNKNOWN,
		}, nil
	}

	canonical, err := proofasset.CanonicalProofAssetBytes(asset)
	if err != nil {
		return nil, fmt.Errorf("canonicalize proof asset: %w", err)
	}

	s.mu.RLock()
	receipt := s.receipts[req.ProofAssetId]
	s.mu.RUnlock()

	status := pb.FormalProofStatus_PROOF_PENDING
	if receipt != nil {
		switch receipt.Status {
		case proofasset.StatusFormallyVerified:
			status = pb.FormalProofStatus_FORMALLY_VERIFIED
		case proofasset.StatusFormallyRejected:
			status = pb.FormalProofStatus_FORMALLY_REJECTED
		case proofasset.StatusCheckerUnavailable:
			status = pb.FormalProofStatus_CHECKER_UNAVAILABLE
		case proofasset.StatusProofPending:
			status = pb.FormalProofStatus_PROOF_PENDING
		}
	}

	return &pb.GetProofAssetResponse{
		ProofAssetId:      req.ProofAssetId,
		CanonicalProofAsset: canonical,
		Status:            status,
		Receipt:           receiptToProto(receipt),
	}, nil
}

// Metrics implements the gRPC Metrics RPC.
func (s *GRPCServer) Metrics(ctx context.Context, req *pb.MetricsRequest) (*pb.MetricsResponse, error) {
	m := s.metrics
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &pb.MetricsResponse{
		TotalSubmissions:  m.TotalSubmissions,
		VerifiedCount:     m.VerifiedCount,
		RejectedCount:     m.RejectedCount,
		ErrorCount:        m.ErrorCount,
		QueueDepth:        m.QueueDepth,
		MedianCheckTimeMs: float32(m.MedianCheckTimeMs),
	}, nil
}
