package authority

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

const sessionsFile = "sessions.jsonl"

// SessionEvent is one line of sessions.jsonl. It holds PUBLIC facts only: the
// session private key is never persisted, so a restart re-runs the ceremony
// (plan step 7). The log gives the next epoch number, the retirement times a
// verifier registry needs, and an audit trail of which key was live when.
type SessionEvent struct {
	Event      string `json:"event"` // "start" or "retire"
	Epoch      uint64 `json:"epoch,omitempty"`
	SessionPub string `json:"session_pub"`
	Address    string `json:"address,omitempty"`
	NotBefore  int64  `json:"not_before,omitempty"`
	NotAfter   int64  `json:"not_after,omitempty"`
	Entropy    string `json:"entropy,omitempty"`
	At         int64  `json:"at"`
	Reason     string `json:"reason,omitempty"`
}

// SessionStatus is the session part of /authority/status.
type SessionStatus struct {
	State      string `json:"state"` // "active", "expired" or "none"
	Epoch      uint64 `json:"epoch,omitempty"`
	Address    string `json:"address,omitempty"`
	NotAfter   int64  `json:"not_after,omitempty"`
	Entropy    string `json:"entropy,omitempty"`
	Rotations  uint64 `json:"rotations"`
	Retired    int    `json:"retired"`
	LastError  string `json:"last_error,omitempty"`
	LastReason string `json:"last_reason,omitempty"`
}

// SessionConfig configures a SessionManager.
type SessionConfig struct {
	LongTerm knirvsigning.RemoteSigner
	Entropy  EntropySource
	// Scope is the certificate scope ("*" for root, the node's address for boot).
	Scope string
	// TTL is the epoch length: the certificate window and the rotation period.
	// Default 24 h comes from policy cert_ttl_seconds (D-A2 leaves it a parameter).
	TTL time.Duration
	// EpochFloor is the lowest epoch to use (the policy epoch).
	EpochFloor uint64
	Dir        string
	Now        func() time.Time
	Logf       func(format string, args ...any)
}

// SessionManager owns the node's current ephemeral session key.
//
// A session lives TTL. When a quarter of that remains the manager runs a new
// ceremony and retires the old key; if the ceremony fails the old key keeps
// working until it truly expires, and after that there is NO session (fail
// closed, plan step 10) until a ceremony succeeds.
type SessionManager struct {
	cfg   SessionConfig
	start time.Time // monotonic origin for the key material's monotonic_ns

	mu        sync.Mutex
	cur       *Session
	epoch     uint64 // last epoch used
	retired   map[string]int64
	rotations uint64
	lastErr   string
	lastWhy   string
	retiredN  int
}

// NewSessionManager loads the public history and prepares, but does not start, a
// session. Call Ensure to run the first ceremony.
func NewSessionManager(cfg SessionConfig) (*SessionManager, error) {
	if cfg.LongTerm == nil || cfg.Entropy == nil || cfg.Dir == "" {
		return nil, errors.New("authority: a session manager needs a long-term signer, an entropy source and a directory")
	}
	if cfg.TTL < 5*time.Minute {
		return nil, fmt.Errorf("authority: session TTL %s is shorter than five minutes", cfg.TTL)
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Logf == nil {
		cfg.Logf = func(string, ...any) {}
	}
	m := &SessionManager{cfg: cfg, start: cfg.Now(), retired: map[string]int64{}}
	events, err := readSessionEvents(cfg.Dir)
	if err != nil {
		return nil, err
	}
	started := map[string]SessionEvent{}
	for _, e := range events {
		switch e.Event {
		case "start":
			if e.Epoch > m.epoch {
				m.epoch = e.Epoch
			}
			started[e.SessionPub] = e
		case "retire":
			m.retired[e.SessionPub] = e.At
		}
	}
	// A key from a previous process cannot be live again: its private half existed
	// only in that process's memory. Anything still unretired in the log is retired
	// now, with the reason "restart", so the history and the verifier registry say
	// what is true.
	for pub, e := range started {
		if _, done := m.retired[pub]; done {
			continue
		}
		at := cfg.Now().Unix()
		m.retired[pub] = at
		if err := appendSessionEvent(cfg.Dir, SessionEvent{Event: "retire", SessionPub: pub, Address: e.Address, At: at, Reason: "restart"}); err != nil {
			cfg.Logf("WARN authority: could not record the retirement of a previous session key: %v", err)
		}
	}
	m.retiredN = len(m.retired)
	return m, nil
}

// Current returns the live session, or nil when there is none (never started, or
// expired after a failed rotation).
func (m *SessionManager) Current(now time.Time) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.liveLocked(now)
}

func (m *SessionManager) liveLocked(now time.Time) *Session {
	if m.cur == nil || now.Unix() >= m.cur.Cert.NotAfter || now.Unix() < m.cur.Cert.NotBefore {
		return nil
	}
	return m.cur
}

// Ensure returns a live session, running a ceremony if there is none or less than
// a quarter of the window remains. A failed scheduled rotation is not fatal while
// the old key is still valid.
func (m *SessionManager) Ensure(ctx context.Context) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.cfg.Now()
	if cur := m.liveLocked(now); cur != nil && now.Unix() < cur.Cert.NotAfter-int64(m.cfg.TTL/time.Second)/4 {
		return cur, nil
	}
	reason := "scheduled"
	if m.cur == nil {
		reason = "start"
	}
	next, err := m.rotateLocked(ctx, now, reason)
	if err != nil {
		if cur := m.liveLocked(now); cur != nil {
			m.cfg.Logf("WARN authority: scheduled session rotation failed (%v); the current session key stays in use until it expires at %d", err, cur.Cert.NotAfter)
			return cur, nil
		}
		return nil, err
	}
	return next, nil
}

// Rotate forces a new ceremony now and retires the current key, for example after a
// suspected theft (plan test "stolen-session revocation by rotation").
func (m *SessionManager) Rotate(ctx context.Context, reason string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if reason == "" {
		reason = "manual"
	}
	return m.rotateLocked(ctx, m.cfg.Now(), reason)
}

func (m *SessionManager) rotateLocked(ctx context.Context, now time.Time, reason string) (*Session, error) {
	epoch := m.epoch + 1
	if epoch < m.cfg.EpochFloor {
		epoch = m.cfg.EpochFloor
	}
	sess, err := RunCeremony(ctx, CeremonyParams{
		LongTerm: m.cfg.LongTerm, Entropy: m.cfg.Entropy, Epoch: epoch, Scope: m.cfg.Scope,
		Now: now, TTL: m.cfg.TTL, Monotime: now.Sub(m.start).Nanoseconds(),
	})
	if err != nil {
		m.lastErr = err.Error()
		m.cfg.Logf("WARN authority: session ceremony failed: %v", err)
		return nil, err
	}
	// Record the new key first, then retire the old one, so a crash between the
	// two leaves an unretired entry (harmless) rather than a retired-but-absent key.
	if err := appendSessionEvent(m.cfg.Dir, SessionEvent{
		Event: "start", Epoch: epoch, SessionPub: hex.EncodeToString(sess.Cert.SessionPub),
		Address: sess.Signer.Address(), NotBefore: sess.Cert.NotBefore, NotAfter: sess.Cert.NotAfter,
		Entropy: sess.Kind, At: now.Unix(), Reason: reason,
	}); err != nil {
		return nil, fmt.Errorf("record session start: %w", err)
	}
	if old := m.cur; old != nil {
		pub := hex.EncodeToString(old.Cert.SessionPub)
		m.retired[pub] = now.Unix()
		m.retiredN = len(m.retired)
		if err := appendSessionEvent(m.cfg.Dir, SessionEvent{Event: "retire", SessionPub: pub, Address: old.Signer.Address(), At: now.Unix(), Reason: reason}); err != nil {
			m.cfg.Logf("WARN authority: could not record session retirement: %v", err)
		}
	}
	m.cur, m.epoch, m.lastErr, m.lastWhy = sess, epoch, "", reason
	m.rotations++
	m.cfg.Logf("authority: session ceremony complete (%s): epoch %d, key %s, entropy %s, valid until %d", reason, epoch, sess.Signer.Address(), sess.Kind, sess.Cert.NotAfter)
	return sess, nil
}

// Retired maps hex(session_pub) to its retirement time, for a verifier registry.
// Note the limit recorded in spec section 10 (D1): a verifier can only use this to
// reject certificates that START after retirement, not individual late records.
func (m *SessionManager) Retired() map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]int64, len(m.retired))
	for k, v := range m.retired {
		out[k] = v
	}
	return out
}

// LastEpoch is the newest epoch this node has used (0 if none).
func (m *SessionManager) LastEpoch() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.epoch
}

// Status reports the session state.
func (m *SessionManager) Status(now time.Time) SessionStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := SessionStatus{State: "none", Rotations: m.rotations, Retired: len(m.retired), LastError: m.lastErr, LastReason: m.lastWhy}
	if m.cur != nil {
		st.Epoch, st.Address, st.NotAfter, st.Entropy = m.cur.Epoch, m.cur.Signer.Address(), m.cur.Cert.NotAfter, m.cur.Kind
		st.State = "expired"
		if m.liveLocked(now) != nil {
			st.State = "active"
		}
	}
	return st
}

// Run keeps the session fresh until ctx ends: it re-checks every interval and
// rotates when a quarter of the window remains. Failures are logged, never fatal.
func (m *SessionManager) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if _, err := m.Ensure(ctx); err != nil {
				m.cfg.Logf("WARN authority: no live session key and the ceremony failed: %v (T2 and T3 are refused until it succeeds)", err)
			}
		}
	}
}

// ---- sessions.jsonl ----

var sessionLogMu sync.Mutex

func appendSessionEvent(dir string, e SessionEvent) error {
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	sessionLogMu.Lock()
	defer sessionLogMu.Unlock()
	f, err := os.OpenFile(filepath.Join(dir, sessionsFile), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

func readSessionEvents(dir string) ([]SessionEvent, error) {
	data, err := os.ReadFile(filepath.Join(dir, sessionsFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []SessionEvent
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 1<<16), 1<<20)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var e SessionEvent
		if err := json.Unmarshal(line, &e); err != nil {
			// A torn final line is tolerated; the log only feeds the epoch counter
			// and retirement times, and losing one trailing event cannot let an
			// epoch repeat because every start is written before it is used.
			continue
		}
		out = append(out, e)
	}
	return out, sc.Err()
}

// registryRetired exposes the retirement map in the SDK's registry shape.
func registryRetired(m *SessionManager) map[string]int64 {
	if m == nil {
		return nil
	}
	return m.Retired()
}
