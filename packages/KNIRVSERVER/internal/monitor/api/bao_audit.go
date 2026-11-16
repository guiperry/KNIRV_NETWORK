package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Bao audit view (OpenBao.md §7: every signature and secret read gets an audit
// record). The monitor reads OpenBao's file audit device log read-only and returns a
// whitelisted projection of each record. Tokens, accessors, client addresses and
// request/response bodies are never returned: only time, type, operation, path,
// mount type and error text.

const (
	baoAuditDefaultLimit = 50
	baoAuditMaxLimit     = 200
	// baoAuditTailBytes bounds how much of the log is read per request.
	baoAuditTailBytes = 256 * 1024
)

// BaoAuditEntry is the redacted projection returned to callers.
type BaoAuditEntry struct {
	Time      string `json:"time"`
	Type      string `json:"type"`
	Operation string `json:"operation,omitempty"`
	Path      string `json:"path,omitempty"`
	MountType string `json:"mount_type,omitempty"`
	Error     string `json:"error,omitempty"`
}

// BaoAuditResponse is the handler payload.
type BaoAuditResponse struct {
	Enabled      bool            `json:"enabled"`
	Entries      []BaoAuditEntry `json:"entries"`
	Scanned      int             `json:"scanned"`
	OperationCnt map[string]int  `json:"operation_counts"`
	Timestamp    string          `json:"timestamp"`
}

// rawBaoAudit mirrors only the fields we project; everything else is discarded.
type rawBaoAudit struct {
	Time    string `json:"time"`
	Type    string `json:"type"`
	Error   string `json:"error"`
	Request struct {
		Operation string `json:"operation"`
		Path      string `json:"path"`
		MountType string `json:"mount_type"`
	} `json:"request"`
}

// readBaoAuditTail returns the complete JSON lines in the last baoAuditTailBytes of
// the audit log, newest last. It refuses a symlink or non-regular file.
func readBaoAuditTail(path string) ([][]byte, bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, err
	}
	if !info.Mode().IsRegular() {
		return nil, true, os.ErrInvalid
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, true, err
	}
	defer f.Close()
	size := info.Size()
	var start int64
	if size > baoAuditTailBytes {
		start = size - baoAuditTailBytes
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return nil, true, err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, true, err
	}
	lines := bytes.Split(data, []byte{'\n'})
	if start > 0 && len(lines) > 0 {
		lines = lines[1:] // first line may be a partial record
	}
	out := make([][]byte, 0, len(lines))
	for _, l := range lines {
		if len(bytes.TrimSpace(l)) > 0 {
			out = append(out, l)
		}
	}
	return out, true, nil
}

// projectBaoAudit parses the newest `limit` records (newest first) and counts
// operations across the whole scanned window.
func projectBaoAudit(lines [][]byte, limit int) ([]BaoAuditEntry, map[string]int, int) {
	counts := map[string]int{}
	var entries []BaoAuditEntry
	scanned := 0
	for i := len(lines) - 1; i >= 0; i-- {
		var raw rawBaoAudit
		if err := json.Unmarshal(lines[i], &raw); err != nil {
			continue
		}
		scanned++
		if raw.Request.Operation != "" {
			counts[raw.Request.Operation]++
		}
		if len(entries) < limit {
			entries = append(entries, BaoAuditEntry{
				Time: raw.Time, Type: raw.Type, Operation: raw.Request.Operation,
				Path: raw.Request.Path, MountType: raw.Request.MountType, Error: raw.Error,
			})
		}
	}
	return entries, counts, scanned
}

// handleBaoAudit serves GET /api/v1/monitor/bao-audit?limit=N (default 50, max 200).
func (s *Server) handleBaoAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMonitorError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit := baoAuditDefaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			writeMonitorError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		if n > baoAuditMaxLimit {
			n = baoAuditMaxLimit
		}
		limit = n
	}
	resp := BaoAuditResponse{Entries: []BaoAuditEntry{}, OperationCnt: map[string]int{}, Timestamp: time.Now().UTC().Format(time.RFC3339)}
	if s.config == nil || s.config.BaoAuditLogPath == "" {
		writeJSON(w, resp)
		return
	}
	lines, present, err := readBaoAuditTail(s.config.BaoAuditLogPath)
	if err != nil {
		writeMonitorError(w, http.StatusInternalServerError, "bao audit log unreadable")
		return
	}
	if !present {
		writeJSON(w, resp)
		return
	}
	resp.Enabled = true
	resp.Entries, resp.OperationCnt, resp.Scanned = projectBaoAudit(lines, limit)
	writeJSON(w, resp)
}

// writeJSON writes a 200 JSON body.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v)
}
