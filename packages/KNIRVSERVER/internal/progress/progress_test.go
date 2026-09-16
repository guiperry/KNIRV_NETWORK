package progress

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// TestBarShortLived ensures a bar started and stopped with no tick in between
// still terminates cleanly and can be restarted.
func TestBarShortLived(t *testing.T) {
	var out bytes.Buffer
	sp := New(&out)
	sp.Start("job", "step one", "step two")
	sp.Stop("")
	if out.Len() != 0 {
		t.Fatalf("non-tty writer should not contain bar output, got %q", out.String())
	}
	sp.Start("restarted", "again")
	sp.Stop("")
}

// TestLineModeEmitsRollingLines verifies line mode writes a full line with a
// newline for each render (the non-tty fallback).
func TestLineModeEmitsRollingLines(t *testing.T) {
	var out bytes.Buffer
	sp := New(&out)
	sp.Start("boot", "status")
	sp.SetStats("attempt 1")
	time.Sleep(150 * time.Millisecond)
	sp.Stop("")

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) == 0 {
		t.Fatal("expected at least one rolled status line")
	}
	for _, l := range lines {
		if !strings.Contains(l, "boot") {
			t.Errorf("line %q missing title", l)
		}
	}
	if !strings.Contains(out.String(), "attempt") {
		t.Errorf("expected attempt stat in lines, got %q", out.String())
	}
}

// TestDeterminateTimeoutProgress checks that a timeout-bound bar reaches the
// timeout cap and reports 100%.
func TestDeterminateTimeoutProgress(t *testing.T) {
	var out bytes.Buffer
	sp := NewLine(&out)
	sp.StartWithTimeout("wait", 1*time.Second, "polling")
	sp.SetStats("attempt 5")
	deadline := time.Now().Add(1200 * time.Millisecond)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		sp.SetStats("attempt 7")
	}
	sp.Stop("")
	if !strings.Contains(out.String(), "100%") {
		t.Fatalf("expected capped 100%% bar at timeout, got:\n%s", out.String())
	}
}

// TestLogClearsBarRedraw verifies logf-style interlacing doesn't corrupt the
// final line: the writer must end with the summary on its own line.
func TestLogClearsBarRedraw(t *testing.T) {
	var out bytes.Buffer
	sp := NewLine(&out)
	sp.Start("extract", "decompressing")
	time.Sleep(120 * time.Millisecond)
	sp.Stop("Extracted backend_server")

	// Line mode must never contain a bare \r carriage return as a leftover.
	if strings.Contains(out.String(), "\r") {
		t.Fatalf("line mode produced stray carriage returns:\n%q", out.String())
	}
	if !strings.Contains(out.String(), "Extracted backend_server\n") {
		t.Fatalf("expected summary line, got:\n%q", out.String())
	}
}

// TestStatsInjection checks stats appear in rendered output.
func TestStatsInjection(t *testing.T) {
	var out bytes.Buffer
	sp := NewLine(&out)
	sp.Start("download", "fetching")
	sp.SetStats("12.0MiB downloaded")
	time.Sleep(120 * time.Millisecond)
	sp.Stop("")
	if !strings.Contains(out.String(), "12.0MiB downloaded") {
		t.Fatalf("expected injected stat, got:\n%q", out.String())
	}
}

// TestByteReporter formats cumulative bytes as stats.
func TestByteReporter(t *testing.T) {
	var out bytes.Buffer
	sp := NewLine(&out)
	sp.Start("download", "fetching")
	br := &ByteReporter{Spinner: sp}
	br.Add(1024)
	br.Add(512 * 1024)
	time.Sleep(120 * time.Millisecond)
	sp.Stop("")
	if !strings.Contains(out.String(), "KiB") {
		t.Fatalf("expected KiB stat after >1KiB, got:\n%q", out.String())
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0B"},
		{512, "512B"},
		{1024, "1.0KiB"},
		{3 << 20, "3.0MiB"},
		{1 << 30, "1.0GiB"},
	}
	for _, c := range cases {
		if got := formatBytes(c.n); got != c.want {
			t.Errorf("formatBytes(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}