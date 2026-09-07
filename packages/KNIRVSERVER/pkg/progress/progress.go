// Package progress renders lightweight, non-destructive terminal progress
// indicators for the long blocking holds that occur during KNIRV-SERVER
// initialization (backend decompression, subprocess startup, and bytes-total
// download steps). It degrades gracefully: when the output is not a terminal
// the same rolling statuses are emitted as ordinary lines, so piped output
// stays readable.
//
// Two rendering modes exist:
//
//   - Bar mode (the default when out is a terminal): a single line is
//     overwritten in place with a progress bar, a percentage (when a timeout is
//     known), a rolling status label, optional stats, and the elapsed time.
//   - Line mode (non-terminal output, or forced via NewLine): each tick emits a
//     fresh full line, which is safe when other writers are streaming to the
//     same destination.
package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	fullBlock    = '█'
	emptyBlock   = '░'
	defaultWidth = 80
)

// Spinner renders rolling initialization progress to a writer.
//
// A Spinner is not safe for concurrent use unless its receiver methods are
// serialized internally; all exported methods take the internal lock.
type Spinner struct {
	mu       sync.Mutex
	out      io.Writer
	barMode  bool
	interval time.Duration
	maxWidth int

	running  bool
	stopCh   chan struct{}
	doneCh   chan struct{}
	barDrawn bool

	title    string
	statuses []string
	status   int
	stats    []string
	started  time.Time
	timeout  time.Duration
}

func newSpinner(out io.Writer, barMode bool) *Spinner {
	interval := 100 * time.Millisecond
	return &Spinner{
		out:      out,
		barMode:  barMode,
		interval: interval,
		maxWidth: defaultWidth,
	}
}

// New returns a spinner writing to out. Bar mode is used when out is a
// terminal (character device); otherwise it degrades to line mode.
func New(out io.Writer) *Spinner {
	return newSpinner(out, isCharDevice(out))
}

// NewLine returns a spinner forced into line mode regardless of out. Use this
// when other writers are streaming to the same destination concurrently and an
// in-place bar would corrupt their output.
func NewLine(out io.Writer) *Spinner {
	return newSpinner(out, false)
}

// isCharDevice reports whether w is an interactive terminal (a character
// device). It is a conservative check: non-*os.File writers are never treated
// as terminals.
func isCharDevice(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// Start begins indeterminate progress under title, cycling through the
// provided rolling status labels at each tick.
func (s *Spinner) Start(title string, statuses ...string) {
	s.StartWithTimeout(title, 0, statuses...)
}

// StartWithTimeout begins progress that fills toward timeout (0 means
// indeterminate). statuses cycles as rolling status labels.
func (s *Spinner) StartWithTimeout(title string, timeout time.Duration, statuses ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(statuses) == 0 {
		statuses = []string{"working..."}
	}
	if s.running {
		return
	}
	s.title = title
	s.statuses = statuses
	s.status = 0
	s.started = time.Now()
	s.timeout = timeout
	s.running = true
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.stats = s.stats[:0]

	go s.run()
}

// run owns the render loop. It must be invoked with s.mu held by Start, but it
// re-acquires the lock on every tick to keep drawing and external mutations
// coherent.
func (s *Spinner) run() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	defer close(s.doneCh)
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			if s.running {
				s.drawLocked()
			}
			s.mu.Unlock()
		}
	}
}

// SetStatuses replaces the rolling status labels and restarts the rotation at
// the first entry. Safe to call while running.
func (s *Spinner) SetStatuses(statuses ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(statuses) == 0 {
		return
	}
	s.statuses = statuses
	s.status = 0
}

// SetStats replaces the comma-separated stats that trail the status label. Safe
// to call while running.
func (s *Spinner) SetStats(stats ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats = append(s.stats[:0], stats...)
}

// Log writes line as a full terminal line. In bar mode the in-place bar is
// cleared first and redrawn on the next tick; in line mode it is emitted
// directly. Interleaving with subprocess output is therefore always safe.
func (s *Spinner) Log(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.barMode {
		s.clearLocked()
	}
	s.writeLine(strings.TrimRight(line, "\r\n"))
	if s.barMode && s.running {
		s.drawLocked()
	}
}

// Stop halts the spinner and emits a final summary line when summary is
// non-empty. An empty summary just clears the bar cleanly without a final line.
func (s *Spinner) Stop(summary string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		s.running = false
		close(s.stopCh)
		// Wait for the render goroutine to exit so no tick lands after the bar
		// is cleared below. The lock is released to let the goroutine finish.
		s.mu.Unlock()
		<-s.doneCh
		s.mu.Lock()
	}
	if s.barMode && s.barDrawn {
		s.clearLocked()
	}
	if summary != "" {
		s.writeLine(strings.TrimRight(summary, "\r\n"))
	}
}

// drawLocked renders the current tick. The status label cycles on every draw
// so motion is visible even when only elapsed time changes.
func (s *Spinner) drawLocked() {
	elapsed := time.Since(s.started)
	frac := 0.0
	if s.timeout > 0 {
		frac = float64(elapsed) / float64(s.timeout)
		if frac > 1 {
			frac = 1
		}
	}

	label := s.statuses[s.status%len(s.statuses)]
	if len(s.statuses) > 1 {
		s.status++
	}

	line := s.renderLine(frac, label, elapsed)
	s.barDrawn = true
	if s.barMode {
		fmt.Fprint(s.out, "\r"+line)
	} else {
		fmt.Fprintln(s.out, line)
	}
}

func (s *Spinner) renderLine(frac float64, label string, elapsed time.Duration) string {
	var b strings.Builder
	b.WriteByte('[')
	barWidth := 14
	filled := int(frac * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	for i := 0; i < barWidth; i++ {
		if i < filled {
			b.WriteRune(fullBlock)
		} else if i == filled && s.timeout > 0 {
			b.WriteRune('>')
		} else {
			b.WriteRune(emptyBlock)
		}
	}
	b.WriteString("] ")
	if s.timeout > 0 {
		fmt.Fprintf(&b, "%3.0f%% ", frac*100)
	}
	b.WriteString(s.title)
	if label != "" {
		b.WriteString(": ")
		b.WriteString(label)
	}
	if len(s.stats) > 0 {
		b.WriteString(" · ")
		b.WriteString(strings.Join(s.stats, ", "))
	}
	fmt.Fprintf(&b, " (%s)", formatElapsed(elapsed))

	out := strings.TrimRight(b.String(), " ")
	if r := []rune(out); len(r) > s.maxWidth {
		out = string(r[:s.maxWidth])
	}
	return out
}

// clearLocked erases the in-place bar so the following output starts on a
// clean line.
func (s *Spinner) clearLocked() {
	if !s.barDrawn {
		return
	}
	fmt.Fprintf(s.out, "\r%s\r", strings.Repeat(" ", s.maxWidth))
	s.barDrawn = false
}

func (s *Spinner) writeLine(line string) {
	fmt.Fprintln(s.out, line)
}

// formatElapsed renders a duration compactly, e.g. "25.3s" or "1m02s".
func formatElapsed(d time.Duration) string {
	d = d.Truncate(100 * time.Millisecond)
	return d.String()
}

// ByteReporter is a thin helper for reporting transfer progress on a stream
// with a known total.
type ByteReporter struct {
	Spinner *Spinner
	Total   int64
	Current int64
	mu      sync.Mutex
}

// Add reports that n additional bytes were read and refreshes the stats.
func (r *ByteReporter) Add(n int) {
	if r == nil || r.Spinner == nil || n <= 0 {
		return
	}
	r.mu.Lock()
	r.Current += int64(n)
	cur := r.Current
	total := r.Total
	r.mu.Unlock()

	stats := []string{formatBytes(cur)}
	if total > 0 {
		stats = append(stats, "/ "+formatBytes(total))
		pct := int(float64(cur) / float64(total) * 100)
		if pct > 100 {
			pct = 100
		}
		stats = append(stats, fmt.Sprintf("%d%%", pct))
	}
	r.Spinner.SetStats(stats...)
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), "KMGTPE"[exp])
}