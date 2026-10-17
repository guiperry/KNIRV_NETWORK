package knirvllama

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// tailRing keeps the last lines the child printed so a failed start can say why.
type tailRing struct {
	mu    sync.Mutex
	lines []string
	max   int
}

func newTailRing(max int) *tailRing { return &tailRing{max: max} }

func (r *tailRing) add(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lines = append(r.lines, line)
	if len(r.lines) > r.max {
		r.lines = r.lines[len(r.lines)-r.max:]
	}
}

func (r *tailRing) tail(n int) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n > len(r.lines) {
		n = len(r.lines)
	}
	return append([]string(nil), r.lines[len(r.lines)-n:]...)
}

// lineWriter prefixes every line of child output, forwards it, and records it in the
// ring. It splits on \n and \r (download progress rewrites one line with \r), and
// flushes a runaway unterminated line rather than buffering it forever.
type lineWriter struct {
	prefix string
	out    io.Writer
	ring   *tailRing
	mu     sync.Mutex
	buf    []byte
}

const maxLine = 4096

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = append(w.buf, p...)
	for {
		i := strings.IndexAny(string(w.buf), "\r\n")
		if i < 0 {
			if len(w.buf) > maxLine {
				w.emit(string(w.buf))
				w.buf = w.buf[:0]
			}
			break
		}
		if i > 0 {
			w.emit(string(w.buf[:i]))
		}
		w.buf = w.buf[i+1:]
	}
	return len(p), nil
}

func (w *lineWriter) emit(line string) {
	fmt.Fprintf(w.out, "%s%s\n", w.prefix, line)
	w.ring.add(line)
}

// Flush emits any trailing partial line (call once the child has exited).
func (w *lineWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buf) > 0 {
		w.emit(string(w.buf))
		w.buf = nil
	}
}

// StartError describes a start attempt that failed, including what the child said.
type StartError struct {
	Attempt  int
	Exited   bool // the process ended (as opposed to a health timeout)
	ExitErr  error
	Elapsed  time.Duration
	Tail     []string
	Timeout  bool
	Underlay error
}

func (e *StartError) Error() string {
	var b strings.Builder
	switch {
	case e.Exited:
		fmt.Fprintf(&b, "KNIRVLLAMA exited %s into startup (attempt %d): %v", e.Elapsed.Round(time.Millisecond), e.Attempt, e.ExitErr)
	case e.Timeout:
		fmt.Fprintf(&b, "KNIRVLLAMA did not become healthy after %s (attempt %d)", e.Elapsed.Round(time.Second), e.Attempt)
	default:
		fmt.Fprintf(&b, "KNIRVLLAMA start failed (attempt %d): %v", e.Attempt, e.Underlay)
	}
	if len(e.Tail) > 0 {
		fmt.Fprintf(&b, "; last output: %s", strings.Join(e.Tail, " | "))
	}
	return b.String()
}

func (e *StartError) Unwrap() error {
	if e.ExitErr != nil {
		return e.ExitErr
	}
	return e.Underlay
}

// retryable reports whether another attempt can help. A crash during startup often
// is a cold-cache or contention failure that a warm second attempt survives; a
// health timeout, a missing binary or a cancelled context is not.
func (e *StartError) retryable() bool { return e.Exited }

// permanent marks failures that no retry can fix (binary not found, bad data dir).
type permanent struct{ err error }

func (p permanent) Error() string { return p.err.Error() }
func (p permanent) Unwrap() error { return p.err }

// removeStaleSocket deletes a leftover unix socket file from a previous run. A socket
// that still accepts connections belongs to a live process and is an error.
func removeStaleSocket(path string) error {
	if path == "" {
		return nil
	}
	st, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if st.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("%s exists and is not a socket", path)
	}
	if c, err := net.DialTimeout("unix", path, time.Second); err == nil {
		c.Close()
		return fmt.Errorf("socket %s is held by a live process", path)
	}
	return os.Remove(path)
}

func backoffFor(schedule []time.Duration, attempt int) time.Duration {
	if len(schedule) == 0 {
		return 0
	}
	if attempt-1 < len(schedule) {
		return schedule[attempt-1]
	}
	return schedule[len(schedule)-1]
}
