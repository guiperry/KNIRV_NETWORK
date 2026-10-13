package adapters

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// writeGGUF writes a metadata-only GGUF (no tensors), which is all ReadInfo reads.
func writeGGUF(t *testing.T, dir, name string, kv map[string]any) string {
	t.Helper()
	var b []byte
	le := binary.LittleEndian
	u32 := func(v uint32) { b = le.AppendUint32(b, v) }
	u64 := func(v uint64) { b = le.AppendUint64(b, v) }
	str := func(s string) { u64(uint64(len(s))); b = append(b, s...) }
	b = append(b, "GGUF"...)
	u32(3)
	u64(0)
	u64(uint64(len(kv)))
	for k, v := range kv {
		str(k)
		switch x := v.(type) {
		case string:
			u32(8)
			str(x)
		case float32:
			u32(6)
			u32(math.Float32bits(x))
		}
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func lora(t *testing.T, dir, name, base string) string {
	kv := map[string]any{"general.type": "adapter", "adapter.type": "lora", "general.architecture": "llama",
		"adapter.lora.alpha": float32(16), "knirv.skill_name": name}
	if base != "" {
		kv["knirv.base_sha256"] = base
	}
	return writeGGUF(t, dir, name+".gguf", kv)
}

type fakeLlama struct {
	mu    sync.Mutex
	sets  [][]string
	fail  func(set []Entry) error
	calls int
}

func (f *fakeLlama) restart(_ context.Context, set []Entry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.fail != nil {
		if err := f.fail(set); err != nil {
			return err
		}
	}
	var names []string
	for _, e := range set {
		names = append(names, e.Name)
	}
	f.sets = append(f.sets, names)
	return nil
}

func newMgr(t *testing.T, fl *fakeLlama) (*Manager, string) {
	t.Helper()
	dir := t.TempDir()
	st, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	base := Base{Model: "m", SHA256: func() (string, error) { return strings.Repeat("b", 64), nil }}
	m := NewManager(st, base, fl.restart, NewGate())
	m.DrainTimeout = 300 * time.Millisecond
	return m, dir
}

var baseSHA = strings.Repeat("b", 64)

func TestReadInfo(t *testing.T) {
	p := lora(t, t.TempDir(), "x", baseSHA)
	info, err := ReadInfo(p)
	if err != nil || info.Type != "adapter" || info.AdapterType != "lora" || info.BaseSHA256 != baseSHA || info.Alpha != 16 || info.Skill != "x" {
		t.Fatalf("%+v %v", info, err)
	}
	if _, err := ReadInfo(filepath.Join(t.TempDir(), "none")); err == nil {
		t.Fatal("missing file")
	}
}

func TestRegisterLoadsInOrderAndPersists(t *testing.T) {
	fl := &fakeLlama{}
	m, dir := newMgr(t, fl)
	ctx := context.Background()
	for _, n := range []string{"alpha", "beta"} {
		if _, err := m.Register(ctx, n, lora(t, dir, n, baseSHA)); err != nil {
			t.Fatal(err)
		}
	}
	if id, ok := m.Resolve("beta"); !ok || id != 1 {
		t.Fatalf("beta id %d %v", id, ok)
	}
	// replacing keeps the slot
	if _, err := m.Register(ctx, "alpha", lora(t, dir, "alpha", baseSHA)); err != nil {
		t.Fatal(err)
	}
	if id, _ := m.Resolve("alpha"); id != 0 || len(m.Loaded()) != 2 {
		t.Fatal("replace changed the order")
	}
	// a new manager over the same dir restores both
	st2, _ := OpenStore(dir)
	fl2 := &fakeLlama{}
	m2 := NewManager(st2, m.base, fl2.restart, NewGate())
	if err := m2.Restore(ctx); err != nil || len(m2.Loaded()) != 2 || len(fl2.sets) != 1 || len(fl2.sets[0]) != 2 {
		t.Fatalf("restore: %v %+v", err, fl2.sets)
	}
}

func TestRegisterRefusals(t *testing.T) {
	fl := &fakeLlama{}
	m, dir := newMgr(t, fl)
	ctx := context.Background()
	cases := map[string]struct {
		name, path string
		want       error
	}{
		"other base":     {"a", lora(t, dir, "a", strings.Repeat("c", 64)), ErrBaseMismatch},
		"no base hash":   {"b", lora(t, dir, "b", ""), ErrInvalid},
		"bad name":       {"Bad Name", lora(t, dir, "c", baseSHA), ErrInvalid},
		"missing":        {"d", filepath.Join(dir, "none.gguf"), ErrInvalid},
		"not an adapter": {"e", writeGGUF(t, dir, "model.gguf", map[string]any{"general.architecture": "llama"}), ErrInvalid},
	}
	for n, c := range cases {
		if _, err := m.Register(ctx, c.name, c.path); !errors.Is(err, c.want) {
			t.Errorf("%s: %v, want %v", n, err, c.want)
		}
	}
	if fl.calls != 0 {
		t.Fatalf("a refused adapter restarted llama %d times", fl.calls)
	}
}

func TestFailedReloadRestoresThePreviousSet(t *testing.T) {
	fl := &fakeLlama{}
	m, dir := newMgr(t, fl)
	ctx := context.Background()
	if _, err := m.Register(ctx, "good", lora(t, dir, "good", baseSHA)); err != nil {
		t.Fatal(err)
	}
	fl.fail = func(set []Entry) error {
		for _, e := range set {
			if e.Name == "bad" {
				return errors.New("tensor shape mismatch")
			}
		}
		return nil
	}
	_, err := m.Register(ctx, "bad", lora(t, dir, "bad", baseSHA))
	if err == nil || !strings.Contains(err.Error(), "previous set restored") {
		t.Fatalf("%v", err)
	}
	if got := m.Loaded(); len(got) != 1 || got[0].Name != "good" {
		t.Fatalf("loaded %v", got)
	}
	if reg := m.Registered(); len(reg) != 1 {
		t.Fatalf("a failed adapter was persisted: %v", reg)
	}
	last := fl.sets[len(fl.sets)-1]
	if len(last) != 1 || last[0] != "good" {
		t.Fatalf("llama was left running with %v", last)
	}
}

func TestRemoveAndUnmanaged(t *testing.T) {
	fl := &fakeLlama{}
	m, dir := newMgr(t, fl)
	ctx := context.Background()
	m.Register(ctx, "a", lora(t, dir, "a", baseSHA))
	m.Register(ctx, "b", lora(t, dir, "b", baseSHA))
	if err := m.Remove(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if id, _ := m.Resolve("b"); id != 0 {
		t.Fatalf("b should shift to id 0, got %d", id)
	}
	if err := m.Remove(ctx, "zzz"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("%v", err)
	}
	st, _ := OpenStore(t.TempDir())
	un := NewManager(st, m.base, nil, NewGate())
	if _, err := un.Register(ctx, "a", lora(t, dir, "a", baseSHA)); !errors.Is(err, ErrUnmanaged) {
		t.Fatalf("%v", err)
	}
}

func TestRestoreSkipsMissingFilesAndSurvivesABadAdapter(t *testing.T) {
	fl := &fakeLlama{}
	m, dir := newMgr(t, fl)
	ctx := context.Background()
	m.Register(ctx, "keep", lora(t, dir, "keep", baseSHA))
	gone := lora(t, dir, "gone", baseSHA)
	m.Register(ctx, "gone", gone)
	os.Remove(gone)

	st, _ := OpenStore(dir)
	fl2 := &fakeLlama{fail: func(set []Entry) error {
		if len(set) > 0 {
			return errors.New("llama rejected")
		}
		return nil
	}}
	m2 := NewManager(st, m.base, fl2.restart, NewGate())
	if err := m2.Restore(ctx); err != nil {
		t.Fatalf("one bad adapter must not stop the base model: %v", err)
	}
	if len(m2.Loaded()) != 0 {
		t.Fatalf("loaded %v", m2.Loaded())
	}
}

func TestGateDrainsInFlightThenBlocksNewRequests(t *testing.T) {
	g := NewGate()
	leave, err := g.Enter(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	// busy: Pause times out and reports it, without leaving the gate paused
	if _, err := g.Pause(50 * time.Millisecond); err == nil {
		t.Fatal("paused with a request in flight")
	} else {
		var busy *StillBusyError
		if !errors.As(err, &busy) || busy.InFlight != 1 {
			t.Fatalf("%v", err)
		}
	}
	if l2, err := g.Enter(context.Background(), 50*time.Millisecond); err != nil {
		t.Fatalf("a failed pause must not block requests: %v", err)
	} else {
		l2()
	}
	var resumed atomic.Bool
	paused := make(chan func(), 1)
	go func() {
		resume, err := g.Pause(2 * time.Second)
		if err != nil {
			t.Error(err)
			return
		}
		paused <- func() { resumed.Store(true); resume() }
	}()
	time.Sleep(50 * time.Millisecond)
	leave() // the in-flight request finishes -> pause completes
	resume := <-paused
	if _, err := g.Enter(context.Background(), 50*time.Millisecond); !errors.Is(err, ErrReloading) {
		t.Fatalf("a request during a reload must be refused after its wait: %v", err)
	}
	done := make(chan struct{})
	go func() {
		l, err := g.Enter(context.Background(), 2*time.Second)
		if err == nil {
			l()
		}
		close(done)
	}()
	time.Sleep(30 * time.Millisecond)
	resume()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("a waiting request was not admitted after resume")
	}
}
