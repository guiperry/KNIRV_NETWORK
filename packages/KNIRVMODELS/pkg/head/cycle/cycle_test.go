package cycle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func bronze(t *testing.T, dir string, n int) {
	t.Helper()
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `{"event_id":"e%d","session_id":"s%d","instance_id":"i","sequence":1,"event_kind":"gate","source_scope":"cli","tenant_scope":"local","workspace":{"fingerprint":"f%d"},"task":{"request":"review change %d","intent":"review","risk":"low"},"state_before":{"phase":"develop","budget":{}},"policy_bundle_hash":"p","trace":{"agent":"a","proposal":"ok"},"decision":{"recommended_action":"request_review","next_phase":"develop_review"},"outcome":{},"license":"operator-generated","collected_at":"2026-01-01T00:00:00Z"}`+"\n", i, i, i, i)
	}
	if err := os.WriteFile(filepath.Join(dir, "2026-01-01.jsonl"), []byte(b.String()), 0o640); err != nil {
		t.Fatal(err)
	}
}

// fakeTrainer exits with `code` for training and 0 for convert, logging argv.
func fakeTrainer(t *testing.T, code int) (bin, log string) {
	t.Helper()
	d := t.TempDir()
	bin, log = filepath.Join(d, "head-trainer"), filepath.Join(d, "argv")
	script := fmt.Sprintf("#!/bin/sh\necho \"$@\" >> %s\nif [ \"$1\" = convert ]; then exit 0; fi\nexit %d\n", log, code)
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return
}

func cfg(t *testing.T, bin string) Config {
	return Config{BronzeDir: t.TempDir(), WorkDir: t.TempDir(), OutDir: t.TempDir(), TrainerBin: bin, MinNew: 2, MinGap: time.Hour}
}

func TestSkipsWithoutBronze(t *testing.T) {
	bin, _ := fakeTrainer(t, 0)
	st, err := Run(context.Background(), cfg(t, bin))
	if err != nil || st.Outcome != Skipped {
		t.Fatalf("%+v %v", st, err)
	}
}

func TestPromotedRefusedAndGating(t *testing.T) {
	for _, tc := range []struct {
		code int
		want string
	}{{0, Promoted}, {3, Refused}} {
		bin, log := fakeTrainer(t, tc.code)
		c := cfg(t, bin)
		bronze(t, c.BronzeDir, 3)
		st, err := Run(context.Background(), c)
		if err != nil || st.Outcome != tc.want || st.Trained != 3 {
			t.Fatalf("code %d: %+v %v", tc.code, st, err)
		}
		argv, _ := os.ReadFile(log)
		if !strings.Contains(string(argv), "convert --episodes") || !strings.Contains(string(argv), "--preset small") {
			t.Fatalf("argv: %s", argv)
		}
		// Immediately again: too soon, and the recorded state is untouched.
		st2, _ := Run(context.Background(), c)
		if st2.Outcome != Skipped || LoadState(c.WorkDir).Outcome != tc.want {
			t.Fatalf("%+v / %+v", st2, LoadState(c.WorkDir))
		}
		// Past the gap but no new data: still skipped.
		c.MinGap = time.Nanosecond
		time.Sleep(2 * time.Millisecond)
		if st3, _ := Run(context.Background(), c); st3.Outcome != Skipped || !strings.Contains(st3.Reason, "new episodes") {
			t.Fatalf("%+v", st3)
		}
	}
}

func TestTrainerErrorIsFailed(t *testing.T) {
	bin, _ := fakeTrainer(t, 1)
	c := cfg(t, bin)
	bronze(t, c.BronzeDir, 3)
	st, err := Run(context.Background(), c)
	if err == nil || st.Outcome != Failed {
		t.Fatalf("%+v %v", st, err)
	}
	if st.Trained != 0 {
		t.Fatal("a failed cycle must not mark data as consumed")
	}
}

func TestSoftLabelsPassedOnlyWhenPresent(t *testing.T) {
	bin, log := fakeTrainer(t, 0)
	c := cfg(t, bin)
	bronze(t, c.BronzeDir, 3)
	c.SoftLabels = filepath.Join(t.TempDir(), "missing.jsonl")
	c.Force = true
	if _, err := Run(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(log); strings.Contains(string(b), "--soft") {
		t.Fatal("--soft passed for a missing file")
	}
}
