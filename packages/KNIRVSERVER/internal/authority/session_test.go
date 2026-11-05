package authority

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

// toggleEntropy is the OS CSPRNG that can be made to fail.
type toggleEntropy struct {
	mu   sync.Mutex
	fail bool
}

func (e *toggleEntropy) set(fail bool) { e.mu.Lock(); e.fail = fail; e.mu.Unlock() }
func (e *toggleEntropy) Draw(ctx context.Context, c [32]byte) ([]byte, string, error) {
	e.mu.Lock()
	fail := e.fail
	e.mu.Unlock()
	if fail {
		return nil, "", errors.New("entropy device unavailable")
	}
	return CSPRNG{}.Draw(ctx, c)
}

type sessionRig struct {
	m     *SessionManager
	clock *clock
	ent   *toggleEntropy
	logs  *logBuf
	dir   string
	long  knirvsigning.RemoteSigner
}

func newSessionRig(t *testing.T, dir string, floor uint64) *sessionRig {
	t.Helper()
	r := &sessionRig{clock: newClock(), ent: &toggleEntropy{}, logs: &logBuf{}, dir: dir, long: testSigner(t, 1)}
	m, err := NewSessionManager(SessionConfig{
		LongTerm: r.long, Entropy: r.ent, Scope: "*", TTL: 24 * time.Hour, EpochFloor: floor,
		Dir: dir, Now: r.clock.Now, Logf: r.logs.Logf,
	})
	if err != nil {
		t.Fatal(err)
	}
	r.m = m
	return r
}

func events(t *testing.T, dir string) []SessionEvent {
	t.Helper()
	ev, err := readSessionEvents(dir)
	if err != nil {
		t.Fatal(err)
	}
	return ev
}

func TestFirstEnsureRunsOneCeremonyAndReusesIt(t *testing.T) {
	r := newSessionRig(t, t.TempDir(), 1)
	if r.m.Current(r.clock.Now()) != nil {
		t.Fatal("a session exists before any ceremony")
	}
	a, err := r.m.Ensure(context.Background())
	if err != nil || a.Epoch != 1 {
		t.Fatalf("%+v %v", a, err)
	}
	b, _ := r.m.Ensure(context.Background())
	if b != a {
		t.Fatal("a second Ensure inside the window ran another ceremony")
	}
	if st := r.m.Status(r.clock.Now()); st.State != "active" || st.Rotations != 1 || st.Epoch != 1 {
		t.Fatalf("%+v", st)
	}
	if ev := events(t, r.dir); len(ev) != 1 || ev[0].Event != "start" || ev[0].Reason != "start" {
		t.Fatalf("history %+v", ev)
	}
}

func TestSessionRotatesWhenAQuarterOfTheWindowRemains(t *testing.T) {
	r := newSessionRig(t, t.TempDir(), 1)
	first, _ := r.m.Ensure(context.Background())
	r.clock.Advance(17 * time.Hour) // 7h of 24h left: still more than a quarter
	if cur, _ := r.m.Ensure(context.Background()); cur != first {
		t.Fatal("rotated too early")
	}
	r.clock.Advance(2 * time.Hour) // 5h left: inside the last quarter
	second, err := r.m.Ensure(context.Background())
	if err != nil || second == first || second.Epoch != 2 {
		t.Fatalf("expected a rotation to epoch 2: %+v %v", second, err)
	}
	if bytes.Equal(first.Cert.SessionPub, second.Cert.SessionPub) {
		t.Fatal("rotation kept the same key")
	}
	retired := r.m.Retired()
	if at, ok := retired[hex.EncodeToString(first.Cert.SessionPub)]; !ok || at != r.clock.Now().Unix() {
		t.Fatalf("old key not retired at the rotation time: %v", retired)
	}
	if _, ok := retired[hex.EncodeToString(second.Cert.SessionPub)]; ok {
		t.Fatal("the live key is marked retired")
	}
	ev := events(t, r.dir)
	if len(ev) != 3 || ev[1].Event != "start" || ev[2].Event != "retire" || ev[2].Reason != "scheduled" {
		t.Fatalf("history %+v", ev)
	}
}

func TestFailedScheduledRotationKeepsTheOldKeyUntilItExpiresThenFailsClosed(t *testing.T) {
	r := newSessionRig(t, t.TempDir(), 1)
	first, _ := r.m.Ensure(context.Background())
	r.ent.set(true)

	r.clock.Advance(20 * time.Hour) // last quarter: a rotation is due and fails
	cur, err := r.m.Ensure(context.Background())
	if err != nil || cur != first {
		t.Fatalf("the old key should stay usable while valid: %v", err)
	}
	if !strings.Contains(r.logs.String(), "stays in use") {
		t.Fatalf("a failed rotation must warn: %q", r.logs.String())
	}
	if st := r.m.Status(r.clock.Now()); st.State != "active" || st.LastError == "" {
		t.Fatalf("%+v", st)
	}

	r.clock.Advance(5 * time.Hour) // past not_after
	if _, err := r.m.Ensure(context.Background()); err == nil {
		t.Fatal("an expired session with a failing ceremony must give no session")
	}
	if r.m.Current(r.clock.Now()) != nil {
		t.Fatal("Current returned an expired session")
	}
	if st := r.m.Status(r.clock.Now()); st.State != "expired" {
		t.Fatalf("%+v", st)
	}

	r.ent.set(false) // entropy recovers: the next Ensure heals
	healed, err := r.m.Ensure(context.Background())
	if err != nil || healed.Epoch != 2 || r.m.Status(r.clock.Now()).State != "active" {
		t.Fatalf("did not recover: %+v %v", healed, err)
	}
}

func TestRestartReRunsTheCeremonyAndNeverRepeatsAnEpoch(t *testing.T) {
	dir := t.TempDir()
	r1 := newSessionRig(t, dir, 1)
	a, _ := r1.m.Ensure(context.Background())
	r1.clock.Advance(20 * time.Hour)
	b, _ := r1.m.Ensure(context.Background()) // epoch 2
	if a.Epoch != 1 || b.Epoch != 2 {
		t.Fatal("setup")
	}

	r2 := newSessionRig(t, dir, 1) // "restart": a fresh manager over the same directory
	if r2.m.Current(r2.clock.Now()) != nil {
		t.Fatal("a session survived a restart: its private key cannot have")
	}
	if r2.m.LastEpoch() != 2 {
		t.Fatalf("the epoch counter was not restored: %d", r2.m.LastEpoch())
	}
	retired := r2.m.Retired()
	if _, ok := retired[hex.EncodeToString(b.Cert.SessionPub)]; !ok {
		t.Fatal("the previous process's live key must be recorded as retired by the restart")
	}
	c, err := r2.m.Ensure(context.Background())
	if err != nil || c.Epoch != 3 || bytes.Equal(c.Cert.SessionPub, b.Cert.SessionPub) {
		t.Fatalf("after restart: %+v %v", c, err)
	}
	var restartRetires int
	for _, e := range events(t, dir) {
		if e.Event == "retire" && e.Reason == "restart" {
			restartRetires++
		}
	}
	if restartRetires != 1 {
		t.Fatalf("expected exactly one restart retirement (the other key was already retired), got %d", restartRetires)
	}
}

func TestForcedRotationRetiresTheCurrentKey(t *testing.T) {
	r := newSessionRig(t, t.TempDir(), 1)
	a, _ := r.m.Ensure(context.Background())
	b, err := r.m.Rotate(context.Background(), "suspected theft")
	if err != nil || b.Epoch != 2 || bytes.Equal(a.Cert.SessionPub, b.Cert.SessionPub) {
		t.Fatalf("%+v %v", b, err)
	}
	if _, ok := r.m.Retired()[hex.EncodeToString(a.Cert.SessionPub)]; !ok {
		t.Fatal("not retired")
	}
	if ev := events(t, r.dir); ev[len(ev)-1].Reason != "suspected theft" {
		t.Fatalf("reason not recorded: %+v", ev)
	}
	// A failed forced rotation does not silently keep the compromised key's slot.
	r.ent.set(true)
	if _, err := r.m.Rotate(context.Background(), "again"); err == nil {
		t.Fatal("a failing forced rotation must report the failure")
	}
}

func TestEpochFloorAndConstructionChecks(t *testing.T) {
	r := newSessionRig(t, t.TempDir(), 7)
	if s, _ := r.m.Ensure(context.Background()); s.Epoch != 7 {
		t.Fatalf("epoch floor ignored: %d", s.Epoch)
	}
	long := testSigner(t, 1)
	for name, cfg := range map[string]SessionConfig{
		"no signer":  {Entropy: CSPRNG{}, Dir: t.TempDir(), TTL: time.Hour},
		"no entropy": {LongTerm: long, Dir: t.TempDir(), TTL: time.Hour},
		"no dir":     {LongTerm: long, Entropy: CSPRNG{}, TTL: time.Hour},
		"short ttl":  {LongTerm: long, Entropy: CSPRNG{}, Dir: t.TempDir(), TTL: time.Minute},
	} {
		if _, err := NewSessionManager(cfg); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestSessionHistoryToleratesATornFinalLine(t *testing.T) {
	dir := t.TempDir()
	r := newSessionRig(t, dir, 1)
	r.m.Ensure(context.Background())
	f, _ := os.OpenFile(filepath.Join(dir, sessionsFile), os.O_APPEND|os.O_WRONLY, 0o600)
	f.WriteString(`{"event":"start","epoch":9,"session_pub":"ab`)
	f.Close()
	r2 := newSessionRig(t, dir, 1)
	if r2.m.LastEpoch() != 1 {
		t.Fatalf("a torn line must be ignored, epoch = %d", r2.m.LastEpoch())
	}
}

func TestConcurrentEnsureRunsOneCeremony(t *testing.T) {
	r := newSessionRig(t, t.TempDir(), 1)
	var wg sync.WaitGroup
	got := make(chan *Session, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := r.m.Ensure(context.Background())
			if err != nil {
				t.Error(err)
			}
			got <- s
		}()
	}
	wg.Wait()
	close(got)
	var first *Session
	for s := range got {
		if first == nil {
			first = s
		} else if s != first {
			t.Fatal("concurrent callers got different sessions")
		}
	}
	if r.m.Status(r.clock.Now()).Rotations != 1 {
		t.Fatal("more than one ceremony ran")
	}
}

func TestRunRotatesInTheBackgroundAndStops(t *testing.T) {
	r := newSessionRig(t, t.TempDir(), 1)
	r.m.Ensure(context.Background())
	r.clock.Advance(20 * time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { r.m.Run(ctx, 10*time.Millisecond); close(done) }()
	deadline := time.Now().Add(10 * time.Second)
	for r.m.Status(r.clock.Now()).Epoch < 2 {
		if time.Now().After(deadline) {
			t.Fatal("the background loop never rotated")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not stop")
	}
}
