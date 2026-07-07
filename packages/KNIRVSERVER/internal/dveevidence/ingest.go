package dveevidence

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type Store interface {
	Put(b *Bundle) (addr string, err error)
	Get(addr string) (*Bundle, bool)
	Has(addr string) bool
}

type MemStore struct {
	mu sync.RWMutex
	m  map[string]*Bundle
}

func NewMemStore() *MemStore { return &MemStore{m: make(map[string]*Bundle)} }

func (s *MemStore) Put(b *Bundle) (string, error) {
	raw, err := json.Marshal(b)
	if err != nil {
		return "", err
	}
	addr := sha256Hex(raw)
	s.mu.Lock()
	s.m[addr] = b
	s.mu.Unlock()
	return addr, nil
}

func (s *MemStore) Get(addr string) (*Bundle, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.m[addr]
	return b, ok
}

func (s *MemStore) Has(addr string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.m[addr]
	return ok
}

type FileStore struct {
	baseDir string
	mu      sync.Mutex
}

func NewFileStore(baseDir string) *FileStore {
	_ = os.MkdirAll(baseDir, 0o755)
	return &FileStore{baseDir: baseDir}
}

func (s *FileStore) Put(b *Bundle) (string, error) {
	raw, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return "", err
	}
	addr := sha256Hex(raw)
	tmp, err := os.CreateTemp(s.baseDir, ".bundle-*.json")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	finalPath := filepath.Join(s.baseDir, strings.TrimPrefix(addr, "sha256:")+".json")
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return addr, nil
}

func (s *FileStore) Get(addr string) (*Bundle, bool) {
	path := filepath.Join(s.baseDir, strings.TrimPrefix(addr, "sha256:")+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var b Bundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, false
	}
	return &b, true
}

func (s *FileStore) Has(addr string) bool {
	_, ok := s.Get(addr)
	return ok
}

type IngestService struct {
	store    Store
	resolver KeyResolver
	policy   *Policy

	mu       sync.RWMutex
	reports  map[string]*ValidationReport
	sessions map[string]string
	evidence map[string]*Evidence
}

func NewIngestService(store Store, resolver KeyResolver, policy *Policy) *IngestService {
	if store == nil {
		store = NewMemStore()
	}
	return &IngestService{
		store:    store,
		resolver: resolver,
		policy:   policy,
		reports:  make(map[string]*ValidationReport),
		sessions: make(map[string]string),
		evidence: make(map[string]*Evidence),
	}
}

type ingestRequest struct {
	Bundle   *Bundle   `json:"bundle"`
	Evidence *Evidence `json:"evidence,omitempty"`
}

func (s *IngestService) Ingest(b *Bundle, ev *Evidence) (*ValidationReport, error) {
	if b == nil {
		return nil, &VerifyError{Code: "ingest", Message: "nil bundle"}
	}
	report, err := VerifyBundle(b, VerifyOptions{Resolver: s.resolver, Policy: s.policy, Evidence: ev})
	if err != nil {
		return nil, err
	}
	addr, err := s.store.Put(b)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.reports[b.SessionID] = report
	s.sessions[b.SessionID] = addr
	if ev != nil {
		s.evidence[b.SessionID] = ev
	}
	s.mu.Unlock()
	return report, nil
}

func (s *IngestService) Report(sessionID string) (*ValidationReport, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.reports[sessionID]
	return r, ok
}

func (s *IngestService) Bundle(sessionID string) (*Bundle, bool) {
	s.mu.RLock()
	addr, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return s.store.Get(addr)
}

func (s *IngestService) Evidence(sessionID string) (*Evidence, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ev, ok := s.evidence[sessionID]
	return ev, ok
}

func RegisterDVEIngestRoutes(r gin.IRouter, svc *IngestService) {
	r.POST("/api/dve/:dve/sessions/ingest", svc.handleIngest)
	r.GET("/api/dve/:dve/sessions/:session", svc.handleSession)
	r.GET("/api/dve/:dve/sessions/:session/evidence", svc.handleEvidence)
	r.GET("/api/dve/:dve/sessions/:session/proof", svc.handleProof)
	r.GET("/api/dve/:dve/sessions/:session/report", svc.handleReport)
}

func (s *IngestService) handleIngest(c *gin.Context) {
	dveID := c.Param("dve")
	var req ingestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}
	if req.Bundle == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing bundle"})
		return
	}
	if strings.TrimSpace(req.Bundle.DVEID) == "" {
		req.Bundle.DVEID = dveID
	}
	report, err := s.Ingest(req.Bundle, req.Evidence)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status := http.StatusOK
	if report.Status == StatusRejected {
		status = http.StatusUnprocessableEntity
	}
	c.JSON(status, report)
}

func (s *IngestService) handleSession(c *gin.Context) {
	b, ok := s.Bundle(c.Param("session"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	c.JSON(http.StatusOK, b)
}

func (s *IngestService) handleEvidence(c *gin.Context) {
	b, ok := s.Bundle(c.Param("session"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	ev, _ := s.Evidence(c.Param("session"))
	c.JSON(http.StatusOK, gin.H{
		"schema_version": b.SchemaVersion,
		"session_id":     b.SessionID,
		"eventlog_root":  b.EventLogRoot,
		"merkle_root":    b.ArtifactMerkleRoot,
		"artifacts":      b.Artifacts,
		"memvid_refs":    b.MemvidRefs,
		"tool_runs":      b.ToolRuns,
		"events":         ev,
	})
}

func (s *IngestService) handleProof(c *gin.Context) {
	b, ok := s.Bundle(c.Param("session"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"session_id":           b.SessionID,
		"signature":            b.Signature,
		"policy_hash":          b.PolicyHash,
		"eventlog_root":        b.EventLogRoot,
		"artifact_merkle_root": b.ArtifactMerkleRoot,
		"workspace_base_hash":  b.WorkspaceBaseHash,
		"workspace_final_hash": b.WorkspaceFinalHash,
	})
}

func (s *IngestService) handleReport(c *gin.Context) {
	r, ok := s.Report(c.Param("session"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	c.JSON(http.StatusOK, r)
}

func ResolverFromPublicKeys(keys map[string]ed25519.PublicKey) KeyResolver {
	return func(keyID string) (ed25519.PublicKey, bool) {
		pub, ok := keys[keyID]
		return pub, ok
	}
}
