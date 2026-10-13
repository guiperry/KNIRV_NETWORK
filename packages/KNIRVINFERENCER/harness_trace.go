package inferencer

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"sync"
	"time"
)

// Candidate is one model the routing table listed for a task and what became of it.
type Candidate struct {
	Model    string `json:"model"`
	Version  string `json:"version,omitempty"`
	Excluded string `json:"excluded,omitempty"` // why it was not eligible; empty = eligible
}

// Advice records the consultation of the advisory chooser (HEAD).
type Advice struct {
	Asked      bool    `json:"asked"`
	Applied    bool    `json:"applied"` // the recommendation changed the order
	Model      string  `json:"model,omitempty"`
	Confidence float32 `json:"confidence,omitempty"`
	Version    string  `json:"version,omitempty"`
	Note       string  `json:"note,omitempty"`
}

// Fallback is one hop that failed and was passed over.
type Fallback struct {
	From      string  `json:"from"`
	Error     string  `json:"error"`
	LatencyMs float64 `json:"latency_ms"`
}

// Trace is the complete account of one Harness.Do call.
type Trace struct {
	ID         string      `json:"id"`
	Time       time.Time   `json:"time"`
	Kind       Kind        `json:"kind"`
	Skill      string      `json:"skill,omitempty"`
	LocalOnly  bool        `json:"local_only,omitempty"`
	NoSpend    bool        `json:"no_spend,omitempty"`
	Candidates []Candidate `json:"candidates"`
	Advice     *Advice     `json:"advice,omitempty"`
	Chosen     string      `json:"chosen,omitempty"`
	Reason     string      `json:"reason,omitempty"`
	Version    string      `json:"model_version,omitempty"`
	// Adapter is the ULoRA adapter that actually ran. Empty for a skill task means the
	// skill did NOT run; Degraded says so explicitly.
	Adapter   string     `json:"adapter,omitempty"`
	Degraded  bool       `json:"degraded,omitempty"`
	Notes     []string   `json:"notes,omitempty"`
	Fallbacks []Fallback `json:"fallbacks,omitempty"`
	LatencyMs float64    `json:"latency_ms"`
	Error     string     `json:"error,omitempty"`
}

func newTraceID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// TraceLog is a fixed-size ring of recent traces.
type TraceLog struct {
	mu   sync.Mutex
	buf  []Trace
	next int
	full bool
}

// NewTraceLog keeps the last n traces (n < 1 keeps one).
func NewTraceLog(n int) *TraceLog {
	if n < 1 {
		n = 1
	}
	return &TraceLog{buf: make([]Trace, n)}
}

// Add stores a trace, evicting the oldest when full.
func (l *TraceLog) Add(t Trace) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buf[l.next] = t
	l.next = (l.next + 1) % len(l.buf)
	if l.next == 0 {
		l.full = true
	}
}

// Recent returns up to limit traces, newest first (limit <= 0 = all kept).
func (l *TraceLog) Recent(limit int) []Trace {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := l.next
	if l.full {
		n = len(l.buf)
	}
	out := make([]Trace, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, l.buf[(l.next-i+len(l.buf))%len(l.buf)])
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// ModelStats are per-model counters since the harness started.
type ModelStats struct {
	Calls      uint64  `json:"calls"`
	Failures   uint64  `json:"failures"`
	LastError  string  `json:"last_error,omitempty"`
	LastCallAt string  `json:"last_call_at,omitempty"`
	P50Ms      float64 `json:"p50_ms"`
	P95Ms      float64 `json:"p95_ms"`
	Circuit    string  `json:"circuit"`
}

type modelStats struct {
	calls, failures uint64
	lastErr         string
	last            time.Time
	lat             [128]float64 // ring of recent latencies, ms
	n               int
}

func (s *modelStats) record(d time.Duration, err error) {
	s.calls++
	s.last = time.Now()
	if err != nil {
		s.failures++
		s.lastErr = err.Error()
	}
	s.lat[s.n%len(s.lat)] = float64(d.Microseconds()) / 1000
	s.n++
}

func (s *modelStats) snapshot() ModelStats {
	out := ModelStats{Calls: s.calls, Failures: s.failures, LastError: s.lastErr}
	if !s.last.IsZero() {
		out.LastCallAt = s.last.UTC().Format(time.RFC3339)
	}
	k := s.n
	if k > len(s.lat) {
		k = len(s.lat)
	}
	if k > 0 {
		v := append([]float64(nil), s.lat[:k]...)
		sort.Float64s(v)
		out.P50Ms = v[(k-1)*50/100]
		out.P95Ms = v[(k-1)*95/100]
	}
	return out
}
