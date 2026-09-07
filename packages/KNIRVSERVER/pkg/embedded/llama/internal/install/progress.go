// Minimal terminal progress indicator for llama provisioning.
//
// The install package lives in its own Go module (github.com/guiperry/knirv/llama)
// so it cannot import the KNIRVSERVER progress package. This is a compact,
// dependency-free equivalent with the same bar/line behavior: it renders an
// in-place progress bar when stderr is a terminal and degrades to rolling log
// lines otherwise.
package install

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type progress struct {
	mu       sync.Mutex
	out      io.Writer
	barMode  bool
	interval time.Duration

	running  bool
	statuses []string
	status   int
	stats    []string
	title    string
	timeout  time.Duration
	started  time.Time
	last     string
	stopCh   chan struct{}
	doneCh   chan struct{}
}

func newProgress(out io.Writer) *progress {
	return &progress{
		out:      out,
		barMode:  isCharDevice(out),
		interval: 100 * time.Millisecond,
	}
}

// NewProgress returns a progress indicator that renders an in-place bar on a
// terminal and rolling log lines otherwise. Attach it to Installer.Prog to make
// KNIRVLLAMA provisioning visible.
func NewProgress(out io.Writer) *progress {
	return newProgress(out)
}

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

func (p *progress) start(title string, timeout time.Duration, statuses ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(statuses) == 0 {
		statuses = []string{"working..."}
	}
	if p.running {
		return
	}
	p.title = title
	p.statuses = statuses
	p.status = 0
	p.timeout = timeout
	p.started = time.Now()
	p.stats = p.stats[:0]
	p.running = true
	p.stopCh = make(chan struct{})
	p.doneCh = make(chan struct{})
	go p.run()
}

func (p *progress) run() {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	defer close(p.doneCh)
	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.mu.Lock()
			if p.running {
				p.drawLocked()
			}
			p.mu.Unlock()
		}
	}
}

func (p *progress) setStats(stats ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats = append(p.stats[:0], stats...)
}

func (p *progress) stop(summary string) {
	p.mu.Lock()
	wasRunning := p.running
	if wasRunning {
		p.running = false
		close(p.stopCh)
	}
	if wasRunning {
		p.mu.Unlock()
		<-p.doneCh
		p.mu.Lock()
	}
	if p.barMode && p.last != "" {
		fmt.Fprintf(p.out, "\r%s\r", strings.Repeat(" ", len(p.last)))
		p.last = ""
	}
	if summary != "" {
		fmt.Fprintln(p.out, summary)
	}
	p.mu.Unlock()
}

func (p *progress) drawLocked() {
	elapsed := time.Since(p.started)
	frac := 0.0
	if p.timeout > 0 {
		frac = elapsed.Seconds() / p.timeout.Seconds()
		if frac > 1 {
			frac = 1
		}
	}
	label := p.statuses[p.status%len(p.statuses)]
	if len(p.statuses) > 1 {
		p.status++
	}

	var b strings.Builder
	b.WriteByte('[')
	const barWidth = 14
	filled := int(frac * barWidth)
	if filled > barWidth {
		filled = barWidth
	}
	for i := 0; i < barWidth; i++ {
		switch {
		case i < filled:
			b.WriteRune('█')
		case i == filled && p.timeout > 0:
			b.WriteRune('>')
		default:
			b.WriteRune('░')
		}
	}
	b.WriteString("] ")
	if p.timeout > 0 {
		fmt.Fprintf(&b, "%3.0f%% ", frac*100)
	}
	b.WriteString(p.title)
	if label != "" {
		b.WriteString(": ")
		b.WriteString(label)
	}
	if len(p.stats) > 0 {
		b.WriteString(" · ")
		b.WriteString(strings.Join(p.stats, ", "))
	}
	fmt.Fprintf(&b, " (%s)", elapsed.Truncate(100*time.Millisecond).String())

	out := strings.TrimRight(b.String(), " ")
	if len(out) > 80 {
		out = out[:80]
	}
	p.last = out
	fmt.Fprint(p.out, "\r"+out)
}

// logf emits a full terminal line, clearing and redrawing the in-place bar so
// interleaved progress lines never corrupt it.
func (p *progress) logf(format string, args ...any) {
	p.mu.Lock()
	barMode := p.barMode
	current := p.last
	p.mu.Unlock()

	if barMode && current != "" {
		p.mu.Lock()
		if p.running {
			fmt.Fprintf(p.out, "\r%s\r", strings.Repeat(" ", len(current)))
			p.last = ""
		}
		p.mu.Unlock()
	}
	fmt.Fprintf(p.out, format, args...)
	if !strings.HasSuffix(format, "\n") {
		fmt.Fprintln(p.out)
	}
	if barMode {
		p.mu.Lock()
		if p.running {
			p.drawLocked()
		}
		p.mu.Unlock()
	}
}

// countingReader reports bytes pulled from an underlying reader so downloads
// show transferred size and percentage.
type countingReader struct {
	r     io.Reader
	prog  *progress
	total int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if n > 0 && c.prog != nil {
		c.total += int64(n)
		c.prog.setStats(fmt.Sprintf("%s downloaded", fmtBytes(c.total)))
	}
	return n, err
}

func fmtBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(n)/float64(div), "KMGTPE"[exp])
}