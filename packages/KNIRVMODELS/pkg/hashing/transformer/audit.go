package transformer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type AuditRecord struct {
	InquiryHash         string   `json:"inquiry_hash"`
	WASMSha256          string   `json:"wasm_sha256"`
	WASMType            WASMType `json:"wasm_type"`
	SourceContextID     string   `json:"source_context_id"`
	ModelCheckpointHash string   `json:"model_checkpoint_hash"`
	Timestamp           string   `json:"timestamp"`
}

// Auditor writes AuditRecords as JSON-lines to a log directory.
type Auditor struct {
	dir string
	mu  sync.Mutex
}

// NewAuditor creates an Auditor that writes to dir.
func NewAuditor(dir string) *Auditor {
	return &Auditor{dir: dir}
}

// WriteAuditLog appends record to a daily audit file.
func (a *Auditor) WriteAuditLog(record *AuditRecord) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if record.Timestamp == "" {
		record.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	if err := os.MkdirAll(a.dir, 0755); err != nil {
		return fmt.Errorf("audit mkdir: %w", err)
	}

	day := time.Now().UTC().Format("2006-01-02")
	path := filepath.Join(a.dir, fmt.Sprintf("heart_audit_%s.jsonl", day))

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("audit open: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("audit marshal: %w", err)
	}

	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}