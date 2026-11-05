package inferencer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// TraceStore keeps every Trace as append-only JSONL under one directory, so routing and
// tool decisions are an audit trail and training data for HEAD's route head, and
// survive a restart.
//
// Layout: the active file is traces.jsonl; when it passes MaxBytes it is renamed to
// traces-<unix-nanoseconds>.jsonl and a new one starts, and only the newest MaxFiles
// rotated files are kept. Files are 0600 in a 0700 directory. Lines are never
// rewritten. A torn or corrupt line (a crash mid-write) is skipped on read.
//
// Provisional (owner-level, open): there is no fsync per line, only on rotation and
// Close, so a crash can lose the last few traces; and the default retention below is a
// guess. Both are constructor options.
type TraceStore struct {
	dir      string
	maxBytes int64
	maxFiles int

	mu     sync.Mutex
	f      *os.File
	size   int64
	errors atomic.Uint64
	lastEr atomic.Value // string
}

// Defaults for NewTraceStore: 8 MiB per file, 8 rotated files kept.
const (
	DefaultTraceFileBytes = 8 << 20
	DefaultTraceFiles     = 8
	activeTraceFile       = "traces.jsonl"
	rotatedPrefix         = "traces-"
	maxTraceLine          = 64 << 10
)

// TraceStoreOption configures NewTraceStore.
type TraceStoreOption func(*TraceStore)

// WithTraceFileLimits sets the rotation size and how many rotated files are kept.
func WithTraceFileLimits(maxBytes int64, maxFiles int) TraceStoreOption {
	return func(s *TraceStore) {
		if maxBytes > 0 {
			s.maxBytes = maxBytes
		}
		if maxFiles > 0 {
			s.maxFiles = maxFiles
		}
	}
}

// NewTraceStore opens (creating if needed) the trace directory.
func NewTraceStore(dir string, opts ...TraceStoreOption) (*TraceStore, error) {
	s := &TraceStore{dir: dir, maxBytes: DefaultTraceFileBytes, maxFiles: DefaultTraceFiles}
	for _, o := range opts {
		o(s)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("trace store: %w", err)
	}
	if err := s.openActive(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *TraceStore) openActive() error {
	f, err := os.OpenFile(filepath.Join(s.dir, activeTraceFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("trace store: %w", err)
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("trace store: %w", err)
	}
	s.f, s.size = f, st.Size()
	return nil
}

// Append writes one trace. A failure is counted (Errors) and otherwise ignored.
func (s *TraceStore) Append(t Trace) {
	line, err := json.Marshal(t)
	if err != nil {
		s.fail(err)
		return
	}
	if len(line) > maxTraceLine {
		// Keep the record, drop the free text that made it large.
		t.Notes, t.Error = nil, clip(t.Error)
		t.Fallbacks = nil
		if line, err = json.Marshal(t); err != nil || len(line) > maxTraceLine {
			s.fail(fmt.Errorf("trace %s is larger than %d bytes", t.ID, maxTraceLine))
			return
		}
	}
	line = append(line, '\n')
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.f == nil {
		s.fail(fmt.Errorf("trace store is closed"))
		return
	}
	if s.size > 0 && s.size+int64(len(line)) > s.maxBytes {
		if err := s.rotateLocked(); err != nil {
			s.fail(err)
			return
		}
	}
	n, err := s.f.Write(line)
	s.size += int64(n)
	if err != nil {
		s.fail(err)
	}
}

func (s *TraceStore) fail(err error) {
	s.errors.Add(1)
	s.lastEr.Store(err.Error())
}

// Errors is how many appends failed, and the last failure.
func (s *TraceStore) Errors() (uint64, string) {
	msg, _ := s.lastEr.Load().(string)
	return s.errors.Load(), msg
}

func (s *TraceStore) rotateLocked() error {
	_ = s.f.Sync()
	if err := s.f.Close(); err != nil {
		return err
	}
	name := fmt.Sprintf("%s%d.jsonl", rotatedPrefix, time.Now().UnixNano())
	if err := os.Rename(filepath.Join(s.dir, activeTraceFile), filepath.Join(s.dir, name)); err != nil {
		return err
	}
	if err := s.openActive(); err != nil {
		return err
	}
	files, _ := s.rotated()
	for len(files) > s.maxFiles {
		_ = os.Remove(files[0])
		files = files[1:]
	}
	return nil
}

// rotated lists rotated files oldest first.
func (s *TraceStore) rotated() ([]string, error) {
	ents, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		n := e.Name()
		if strings.HasPrefix(n, rotatedPrefix) && strings.HasSuffix(n, ".jsonl") {
			out = append(out, filepath.Join(s.dir, n))
		}
	}
	sort.Slice(out, func(i, j int) bool { return numericStamp(out[i]) < numericStamp(out[j]) })
	return out, nil
}

func numericStamp(path string) int64 {
	n := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(path), rotatedPrefix), ".jsonl")
	var v int64
	fmt.Sscanf(n, "%d", &v)
	return v
}

// Read returns up to limit stored traces, newest first (limit <= 0 reads all kept).
// Corrupt lines are skipped.
func (s *TraceStore) Read(limit int) ([]Trace, error) {
	s.mu.Lock()
	if s.f != nil {
		_ = s.f.Sync()
	}
	s.mu.Unlock()
	files, err := s.rotated()
	if err != nil {
		return nil, err
	}
	files = append(files, filepath.Join(s.dir, activeTraceFile))
	var out []Trace
	for i := len(files) - 1; i >= 0 && (limit <= 0 || len(out) < limit); i-- {
		ts, err := readTraceFile(files[i])
		if err != nil && !os.IsNotExist(err) {
			return out, err
		}
		for j := len(ts) - 1; j >= 0 && (limit <= 0 || len(out) < limit); j-- {
			out = append(out, ts[j])
		}
	}
	return out, nil
}

func readTraceFile(path string) ([]Trace, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Trace
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), maxTraceLine+1024)
	for sc.Scan() {
		var t Trace
		if json.Unmarshal(sc.Bytes(), &t) != nil || t.ID == "" {
			continue // torn or corrupt line
		}
		out = append(out, t)
	}
	return out, sc.Err()
}

// Close flushes and closes the active file. Appends after Close are counted as errors.
func (s *TraceStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.f == nil {
		return nil
	}
	_ = s.f.Sync()
	err := s.f.Close()
	s.f = nil
	return err
}
