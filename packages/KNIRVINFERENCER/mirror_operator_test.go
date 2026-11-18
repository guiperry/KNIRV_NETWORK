package inferencer

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

// mirrorService builds a running InferenceService whose harness holds the three daemon
// models served by mirrorFake (HEAD, HEART, HASHER) plus extra models. The flag is set
// through the environment and read at construction, as in production.
func mirrorService(t *testing.T, flag string, f *mirrorFake, extra ...HarnessModel) (*InferenceService, *Harness) {
	t.Helper()
	t.Setenv(envMirroredGeneration, flag)
	s, err := NewInferenceService(nil)
	if err != nil {
		t.Fatal(err)
	}
	head, heart, hasher := DaemonModels(f.server(t))
	all := append([]HarnessModel{head, heart, hasher}, extra...)
	h := NewHarness(all, WithTraceCapacity(8))
	s.SetHarness(h)
	s.isRunning = true
	return s, h
}

// routedService builds a running service whose only model is a fake HEART that answers
// "heart answer" on the routed chain, for the flag-off baseline.
func routedService(t *testing.T, flag string) (*InferenceService, *Harness, *fakeModel) {
	t.Helper()
	t.Setenv(envMirroredGeneration, flag)
	s, err := NewInferenceService(nil)
	if err != nil {
		t.Fatal(err)
	}
	heart := up("heart")
	h := NewHarness([]HarnessModel{heart}, WithTraceCapacity(8))
	s.SetHarness(h)
	s.isRunning = true
	return s, h, heart
}

func cachedTexts(s *InferenceService) []string {
	s.responseCacheMu.Lock()
	defer s.responseCacheMu.Unlock()
	var out []string
	for _, e := range s.responseCache {
		out = append(out, e.response)
	}
	return out
}

func TestMirrorFlagParsing(t *testing.T) {
	for val, want := range map[string]bool{"1": true, "0": false, "": false, "true": false} {
		t.Setenv(envMirroredGeneration, val)
		s, err := NewInferenceService(nil)
		if err != nil {
			t.Fatal(err)
		}
		if s.mirroredGeneration != want {
			t.Fatalf("%s=%q: mirroredGeneration = %v, want %v", envMirroredGeneration, val, s.mirroredGeneration, want)
		}
	}
}

// Flag off: default generation takes the routed chain exactly as before. HEAD is never
// called, no mirror counters move, the trace has no MirrorTrace, and the routed output
// is cached as it always was.
func TestMirrorFlagOffKeepsRoutedPath(t *testing.T) {
	s, h, heart := routedService(t, "")

	text, err := s.GenerateTextWithContext(context.Background(), "", "p", "")
	if err != nil {
		t.Fatal(err)
	}
	if text != "heart answer" || heart.Calls() != 1 {
		t.Fatalf("flag off must answer through the routed chain: text=%q calls=%d", text, heart.Calls())
	}
	if c := h.MirrorCounters(); c != (MirrorCounters{}) {
		t.Fatalf("flag off must leave mirror counters at zero: %+v", c)
	}
	tr := h.Traces(1)[0]
	if tr.Mirror != nil {
		t.Fatalf("flag off trace must have no MirrorTrace: %+v", tr.Mirror)
	}
	if got := cachedTexts(s); len(got) != 1 || got[0] != "heart answer" {
		t.Fatalf("flag off must cache the routed answer as before: %v", got)
	}
}

// Flag on: default generation runs the mirror path. Both candidates are generated and
// verified, HEART wins the tie, the verified winner is cached, and the trace carries the
// MirrorTrace.
func TestMirrorFlagOnTakesMirrorPath(t *testing.T) {
	f := &mirrorFake{headVerdict: "supported", heartVerdict: "supported", decideName: "mirror", decideSelected: "heart"}
	s, h := mirrorService(t, "1", f)

	text, err := s.GenerateTextWithContext(context.Background(), "", "p", "")
	if err != nil {
		t.Fatal(err)
	}
	if text != "from heart" {
		t.Fatalf("mirror tie must go to HEART, got %q", text)
	}
	f.mu.Lock()
	headCalled, heartDecided := f.headBody != "", strings.Contains(f.heartBody, "decision")
	f.mu.Unlock()
	if !headCalled {
		t.Fatal("flag on must take the mirror path, which calls HEAD")
	}
	if !heartDecided {
		t.Fatal("mirror HEART candidate must be decision-conditioned")
	}
	c := h.MirrorCounters()
	if c.Runs != 1 || c.Arbitrated != 1 || c.WonHeart != 1 || c.WonHead != 0 || c.FallbackToRouted != 0 {
		t.Fatalf("unexpected arbitration counters: %+v", c)
	}
	tr := h.Traces(1)[0]
	if tr.Mirror == nil || len(tr.Mirror.Candidates) != 2 {
		t.Fatalf("mirror trace must record both candidates: %+v", tr.Mirror)
	}
	if tr.Chosen != "heart" || !tr.Cached {
		t.Fatalf("verified winner must be the chosen, cached trace: chosen=%q cached=%v", tr.Chosen, tr.Cached)
	}
	if got := cachedTexts(s); len(got) != 1 || got[0] != "from heart" {
		t.Fatalf("verified winner must be cached: %v", got)
	}
}

// A REFUTED or unverified candidate is never written to responseCache. Each case has a
// positive control in the same table where the winner is verified and is cached.
func TestMirrorRefutedOrUnverifiedNeverCached(t *testing.T) {
	cases := []struct {
		name       string
		fake       *mirrorFake
		wantText   string
		wantCached []string
	}{
		{
			name:       "refuted heart loses; verified head is cached",
			fake:       &mirrorFake{headVerdict: "supported", heartVerdict: "refuted"},
			wantText:   "from head",
			wantCached: []string{"from head"},
		},
		{
			name:       "unverified candidates are returned but not cached",
			fake:       &mirrorFake{hasherUnverified: true},
			wantText:   "from heart",
			wantCached: nil,
		},
		{
			name:       "degraded HASHER verdict is returned but not cached",
			fake:       &mirrorFake{headVerdict: "supported", heartVerdict: "supported", hasherDegraded: true},
			wantText:   "from heart",
			wantCached: nil,
		},
		{
			name:       "verified tie is cached (control)",
			fake:       &mirrorFake{headVerdict: "supported", heartVerdict: "supported"},
			wantText:   "from heart",
			wantCached: []string{"from heart"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := mirrorService(t, "1", tc.fake)
			text, err := s.GenerateTextWithContext(context.Background(), "", "p", "")
			if err != nil {
				t.Fatal(err)
			}
			if text != tc.wantText {
				t.Fatalf("text = %q, want %q", text, tc.wantText)
			}
			got := cachedTexts(s)
			if len(got) != len(tc.wantCached) || (len(got) == 1 && got[0] != tc.wantCached[0]) {
				t.Fatalf("cache = %v, want %v", got, tc.wantCached)
			}
		})
	}
}

// The mirror counters record what HASHER said about each candidate: a refuted HEART
// candidate is counted and HEAD wins.
func TestMirrorArbitrationCounters(t *testing.T) {
	f := &mirrorFake{headVerdict: "supported", heartVerdict: "refuted"}
	s, h := mirrorService(t, "1", f)
	if _, err := s.GenerateTextWithContext(context.Background(), "", "p", ""); err != nil {
		t.Fatal(err)
	}
	c := h.MirrorCounters()
	if c.RefutedCandidates != 1 || c.WonHead != 1 || c.Arbitrated != 1 || c.NotCached != 0 {
		t.Fatalf("refutation must be counted and head must win: %+v", c)
	}
}

// The trace JSON includes the MirrorTrace when the mirror path ran, and it never carries
// candidate text. A routed trace has no mirror key at all.
func TestMirrorTraceInTraceJSON(t *testing.T) {
	f := &mirrorFake{headVerdict: "supported", heartVerdict: "refuted"}
	s, h := mirrorService(t, "1", f)
	if _, err := s.GenerateTextWithContext(context.Background(), "", "p", ""); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(h.Traces(1)[0])
	if err != nil {
		t.Fatal(err)
	}
	js := string(raw)
	if !strings.Contains(js, `"mirror":{`) || !strings.Contains(js, `"candidates":[`) {
		t.Fatalf("trace JSON must include the mirror trace: %s", js)
	}
	if strings.Contains(js, "from head") || strings.Contains(js, "from heart") {
		t.Fatalf("mirror trace must not carry candidate text: %s", js)
	}
	if !strings.Contains(js, `"refuted":true`) {
		t.Fatalf("mirror trace must record the HASHER refutation: %s", js)
	}

	s2, h2, _ := routedService(t, "")
	if _, err := s2.GenerateTextWithContext(context.Background(), "", "p", ""); err != nil {
		t.Fatal(err)
	}
	raw2, _ := json.Marshal(h2.Traces(1)[0])
	if strings.Contains(string(raw2), `"mirror"`) {
		t.Fatalf("routed trace must not carry a mirror key: %s", raw2)
	}
}

// When no mirror candidate survives (HEAD and HEART both down) the routed chain answers,
// the mirror counters say so, and that unarbitrated answer is not cached.
func TestMirrorNoCandidateFallsBackUncached(t *testing.T) {
	f := &mirrorFake{headDown: true, heartDown: true, headVerdict: "supported", heartVerdict: "supported"}
	s, h := mirrorService(t, "1", f, up("llama"))
	text, err := s.GenerateTextWithContext(context.Background(), "", "p", "")
	if err != nil {
		t.Fatal(err)
	}
	if text != "llama answer" {
		t.Fatalf("routed fallback must answer, got %q", text)
	}
	if got := cachedTexts(s); len(got) != 0 {
		t.Fatalf("unarbitrated fallback must not be cached: %v", got)
	}
	c := h.MirrorCounters()
	if c.NoCandidate != 1 || c.FallbackToRouted != 1 || c.Arbitrated != 0 {
		t.Fatalf("unexpected counters for no-candidate fallback: %+v", c)
	}
	traces := h.Traces(2)
	if len(traces) != 2 || traces[1].Mirror == nil || traces[1].Error == "" {
		t.Fatalf("failed mirror run must still leave a mirror trace with its error: %+v", traces)
	}
}

// With no head model registered the mirror path cannot run; the routed chain answers and
// the unavailability is counted.
func TestMirrorUnavailableWithoutHead(t *testing.T) {
	t.Setenv(envMirroredGeneration, "1")
	s, err := NewInferenceService(nil)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHarness([]HarnessModel{up("heart")}, WithTraceCapacity(4))
	s.SetHarness(h)
	s.isRunning = true
	text, err := s.GenerateTextWithContext(context.Background(), "", "p", "")
	if err != nil {
		t.Fatal(err)
	}
	if text != "heart answer" {
		t.Fatalf("routed fallback must answer, got %q", text)
	}
	c := h.MirrorCounters()
	if c.Unavailable != 1 || c.FallbackToRouted != 1 || c.Runs != 1 {
		t.Fatalf("unexpected counters without head: %+v", c)
	}
	if got := cachedTexts(s); len(got) != 0 {
		t.Fatalf("fallback must not be cached: %v", got)
	}
}

// mirrorCacheable is the single cache gate. arbitrate never lets a REFUTED candidate
// win, so the refuted guard is checked here directly on a synthetic winner as well.
func TestMirrorCacheableRules(t *testing.T) {
	cand := func(c MirrorCandidateTrace) *MirroredResult {
		c.Model = "heart"
		return &MirroredResult{Winner: "heart", Trace: &MirrorTrace{Candidates: []MirrorCandidateTrace{c}}}
	}
	cases := map[string]struct {
		mr   *MirroredResult
		want bool
	}{
		"verified":          {cand(MirrorCandidateTrace{Verified: true}), true},
		"refuted winner":    {cand(MirrorCandidateTrace{Verified: true, Refuted: true}), false},
		"unverified":        {cand(MirrorCandidateTrace{Verified: true, Unverified: true}), false},
		"degraded":          {cand(MirrorCandidateTrace{Verified: true, Degraded: true}), false},
		"verify error":      {cand(MirrorCandidateTrace{Error: "boom"}), false},
		"not verified":      {cand(MirrorCandidateTrace{}), false},
		"result refuted":    {&MirroredResult{Winner: "heart", Refuted: true, Trace: &MirrorTrace{Candidates: []MirrorCandidateTrace{{Model: "heart", Verified: true}}}}, false},
		"no trace":          {&MirroredResult{Winner: "heart"}, false},
		"winner not listed": {&MirroredResult{Winner: "head", Trace: &MirrorTrace{Candidates: []MirrorCandidateTrace{{Model: "heart", Verified: true}}}}, false},
		"nil result":        {nil, false},
	}
	for name, tc := range cases {
		if got := mirrorCacheable(tc.mr); got != tc.want {
			t.Errorf("%s: mirrorCacheable = %v, want %v", name, got, tc.want)
		}
	}
}

// The arbitration counters are shared by concurrent mirror runs and by readers of the
// status endpoint. 50 writers x 20 bumps plus concurrent MirrorCounters() reads must
// total exactly 1000 with no data race (run under -race; removing the mutex fails it).
func TestMirrorCountersConcurrentSafe(t *testing.T) {
	h := NewHarness(nil, WithTraceCapacity(8))
	var wg sync.WaitGroup
	for g := 0; g < 50; g++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				h.noteMirrorFallback()
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				_ = h.MirrorCounters()
			}
		}()
	}
	wg.Wait()
	if c := h.MirrorCounters(); c.FallbackToRouted != 1000 {
		t.Fatalf("FallbackToRouted = %d, want 1000", c.FallbackToRouted)
	}
}
