package dveevidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	. "github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/dveevidence"
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

type persistedSession struct {
	DVEID         string            `json:"dve_id"`
	SessionID     string            `json:"session_id"`
	BundleAddress string            `json:"bundle_address"`
	Evidence      *Evidence         `json:"evidence,omitempty"`
	Report        *ValidationReport `json:"report"`
}

type sessionStore interface {
	PutSession(*Bundle, *Evidence, *ValidationReport, string) error
	GetSession(dveID, sessionID string) (*persistedSession, bool)
}

func NewFileStore(baseDir string) *FileStore {
	_ = os.MkdirAll(baseDir, 0o755)
	return &FileStore{baseDir: baseDir}
}

func (s *FileStore) Put(b *Bundle) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
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

func (s *FileStore) PutSession(b *Bundle, evidence *Evidence, report *ValidationReport, address string) error {
	record := persistedSession{
		DVEID: b.DVEID, SessionID: b.SessionID, BundleAddress: address,
		Evidence: evidence, Report: report,
	}
	return s.writeSessionRecord(record)
}

func (s *FileStore) GetSession(dveID, sessionID string) (*persistedSession, bool) {
	raw, err := os.ReadFile(s.sessionPath(dveID, sessionID))
	if err != nil {
		return nil, false
	}
	var record persistedSession
	if err := json.Unmarshal(raw, &record); err != nil || record.DVEID != dveID || record.SessionID != sessionID {
		return nil, false
	}
	return &record, true
}

func (s *FileStore) writeSessionRecord(record persistedSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(s.baseDir, "sessions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".session-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.sessionPath(record.DVEID, record.SessionID))
}

func (s *FileStore) sessionPath(dveID, sessionID string) string {
	digest := sha256.Sum256([]byte(dveID + "\x00" + sessionID))
	return filepath.Join(s.baseDir, "sessions", hex.EncodeToString(digest[:])+".json")
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

	ingestMu sync.Mutex
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
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()

	if existing, ok := s.BundleFor(b.DVEID, b.SessionID); ok {
		existingEvidence, hasEvidence := s.EvidenceFor(b.DVEID, b.SessionID)
		if reflect.DeepEqual(existing, b) && ((ev == nil && !hasEvidence) || (hasEvidence && reflect.DeepEqual(existingEvidence, ev))) {
			if report, found := s.ReportFor(b.DVEID, b.SessionID); found {
				return report, nil
			}
		}
		return nil, &VerifyError{Code: "session_conflict", Message: "session id already contains different evidence"}
	}
	report, err := VerifyBundle(b, VerifyOptions{Resolver: s.resolver, Policy: s.policy, Evidence: ev})
	if err != nil {
		return nil, err
	}
	addr, err := s.store.Put(b)
	if err != nil {
		return nil, err
	}
	if durable, ok := s.store.(sessionStore); ok {
		if err := durable.PutSession(b, ev, report, addr); err != nil {
			return nil, err
		}
	}
	key := sessionKey(b.DVEID, b.SessionID)
	s.mu.Lock()
	s.reports[key] = report
	s.sessions[key] = addr
	if ev != nil {
		s.evidence[key] = ev
	}
	s.mu.Unlock()
	return report, nil
}

func (s *IngestService) Report(sessionID string) (*ValidationReport, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for key, report := range s.reports {
		if strings.HasSuffix(key, "\x00"+sessionID) {
			return report, true
		}
	}
	return nil, false
}

func (s *IngestService) Bundle(sessionID string) (*Bundle, bool) {
	s.mu.RLock()
	var addr string
	for key, candidate := range s.sessions {
		if strings.HasSuffix(key, "\x00"+sessionID) {
			addr = candidate
			break
		}
	}
	s.mu.RUnlock()
	if addr == "" {
		return nil, false
	}
	return s.store.Get(addr)
}

func (s *IngestService) Evidence(sessionID string) (*Evidence, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for key, evidence := range s.evidence {
		if strings.HasSuffix(key, "\x00"+sessionID) {
			return evidence, true
		}
	}
	return nil, false
}

func sessionKey(dveID, sessionID string) string { return dveID + "\x00" + sessionID }

func (s *IngestService) BundleFor(dveID, sessionID string) (*Bundle, bool) {
	key := sessionKey(dveID, sessionID)
	s.mu.RLock()
	addr, ok := s.sessions[key]
	s.mu.RUnlock()
	if ok {
		return s.store.Get(addr)
	}
	if durable, ok := s.store.(sessionStore); ok {
		if record, found := durable.GetSession(dveID, sessionID); found {
			return s.store.Get(record.BundleAddress)
		}
	}
	return nil, false
}

func (s *IngestService) EvidenceFor(dveID, sessionID string) (*Evidence, bool) {
	key := sessionKey(dveID, sessionID)
	s.mu.RLock()
	evidence, ok := s.evidence[key]
	s.mu.RUnlock()
	if ok {
		return evidence, true
	}
	if durable, ok := s.store.(sessionStore); ok {
		if record, found := durable.GetSession(dveID, sessionID); found && record.Evidence != nil {
			return record.Evidence, true
		}
	}
	return nil, false
}

func (s *IngestService) ReportFor(dveID, sessionID string) (*ValidationReport, bool) {
	key := sessionKey(dveID, sessionID)
	s.mu.RLock()
	report, ok := s.reports[key]
	s.mu.RUnlock()
	if ok {
		return report, true
	}
	if durable, ok := s.store.(sessionStore); ok {
		if record, found := durable.GetSession(dveID, sessionID); found && record.Report != nil {
			return record.Report, true
		}
	}
	return nil, false
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
	} else if req.Bundle.DVEID != dveID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path and bundle DVE ids differ"})
		return
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
	b, ok := s.BundleFor(c.Param("dve"), c.Param("session"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	c.JSON(http.StatusOK, b)
}

func (s *IngestService) handleEvidence(c *gin.Context) {
	b, ok := s.BundleFor(c.Param("dve"), c.Param("session"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	ev, _ := s.EvidenceFor(c.Param("dve"), c.Param("session"))
	var events []Event
	var artifactHashes []string
	if ev != nil {
		events = ev.Events
		artifactHashes = ev.ArtifactHashes
	}
	c.JSON(http.StatusOK, gin.H{
		"schema_version":  b.SchemaVersion,
		"session_id":      b.SessionID,
		"eventlog_root":   b.EventLogRoot,
		"merkle_root":     b.ArtifactMerkleRoot,
		"artifacts":       b.Artifacts,
		"memvid_refs":     b.MemvidRefs,
		"tool_runs":       b.ToolRuns,
		"events":          events,
		"artifact_hashes": artifactHashes,
	})
}

func (s *IngestService) handleProof(c *gin.Context) {
	b, ok := s.BundleFor(c.Param("dve"), c.Param("session"))
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
	r, ok := s.ReportFor(c.Param("dve"), c.Param("session"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	c.JSON(http.StatusOK, r)
}


// sha256Hex is adapter-local storage addressing. Verification/hash semantics
// are supplied exclusively by the SDK package above.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
