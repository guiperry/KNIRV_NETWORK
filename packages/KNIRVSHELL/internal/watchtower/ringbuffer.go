package watchtower

import "sync"

// RingBuffer is a fixed-capacity circular log store.
// Oldest entries are overwritten when capacity is exceeded.
type RingBuffer struct {
	mu   sync.RWMutex
	buf  []string
	cap  int
	head int // index of the next write slot
	full bool
}

func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{buf: make([]string, capacity), cap: capacity}
}

func (r *RingBuffer) Push(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf[r.head] = line
	r.head = (r.head + 1) % r.cap
	if r.head == 0 {
		r.full = true
	}
}

// Snapshot returns entries in chronological order (oldest first).
func (r *RingBuffer) Snapshot() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.full {
		out := make([]string, r.head)
		copy(out, r.buf[:r.head])
		return out
	}
	out := make([]string, r.cap)
	copy(out, r.buf[r.head:])
	copy(out[r.cap-r.head:], r.buf[:r.head])
	return out
}

// Last returns the most recent n entries.
func (r *RingBuffer) Last(n int) []string {
	snap := r.Snapshot()
	if len(snap) <= n {
		return snap
	}
	return snap[len(snap)-n:]
}
