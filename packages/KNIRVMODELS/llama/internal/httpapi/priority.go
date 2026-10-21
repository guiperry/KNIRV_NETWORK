package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Priority classes, lowest number first (P0 interactive ... P3 teacher).
const (
	classes       = 4
	defaultClass  = 1
	agingInterval = time.Minute // a waiter moves up one class per interval, so P3 cannot starve
	maxQueued     = 64
)

// ErrQueueFull is returned when too many calls are already waiting.
var ErrQueueFull = errors.New("llama admission queue is full")

// Admission hands out a fixed number of slots to waiters in priority order
// (FIFO within a class). It stands in front of llama-server so an interactive
// call never waits behind a teacher job that arrived earlier.
type Admission struct {
	mu      sync.Mutex
	free    int
	waiters []*waiter
	now     func() time.Time

	admitted [classes]uint64
	waitNs   [classes]int64
}

type waiter struct {
	class int
	since time.Time
	ready chan struct{}
}

// NewAdmission returns a queue with n slots (n < 1 is 1).
func NewAdmission(n int) *Admission {
	if n < 1 {
		n = 1
	}
	return &Admission{free: n, now: time.Now}
}

func clampClass(c int) int {
	if c < 0 {
		return 0
	}
	if c >= classes {
		return classes - 1
	}
	return c
}

// Acquire blocks until a slot is free for the class, the context ends, or the
// queue is full. The returned func releases the slot.
func (a *Admission) Acquire(ctx context.Context, class int) (func(), error) {
	class = clampClass(class)
	a.mu.Lock()
	if a.free > 0 && len(a.waiters) == 0 {
		a.free--
		a.admitted[class]++
		a.mu.Unlock()
		return a.release, nil
	}
	if len(a.waiters) >= maxQueued {
		a.mu.Unlock()
		return nil, ErrQueueFull
	}
	w := &waiter{class: class, since: a.now(), ready: make(chan struct{})}
	a.waiters = append(a.waiters, w)
	a.mu.Unlock()

	select {
	case <-w.ready:
		return a.release, nil
	case <-ctx.Done():
		a.mu.Lock()
		defer a.mu.Unlock()
		for i, x := range a.waiters {
			if x == w {
				a.waiters = append(a.waiters[:i], a.waiters[i+1:]...)
				return nil, ctx.Err()
			}
		}
		// Already granted while we were cancelling: give the slot back.
		select {
		case <-w.ready:
			a.grantLocked()
		default:
		}
		return nil, ctx.Err()
	}
}

func (a *Admission) release() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.free++
	a.grantLocked()
}

// grantLocked gives free slots to the best waiters.
func (a *Admission) grantLocked() {
	for a.free > 0 && len(a.waiters) > 0 {
		best := 0
		for i, w := range a.waiters {
			if a.effective(w) < a.effective(a.waiters[best]) {
				best = i
			}
		}
		w := a.waiters[best]
		a.waiters = append(a.waiters[:best], a.waiters[best+1:]...)
		a.free--
		a.admitted[w.class]++
		a.waitNs[w.class] += int64(a.now().Sub(w.since))
		close(w.ready)
	}
}

// effective is the class minus one per agingInterval waited. Ties go to the
// earlier arrival because the scan keeps the first minimum.
func (a *Admission) effective(w *waiter) int {
	return w.class - int(a.now().Sub(w.since)/agingInterval)
}

// Snapshot is for /v1/queue/stats.
func (a *Admission) Snapshot() map[string]any {
	a.mu.Lock()
	defer a.mu.Unlock()
	depth := [classes]int{}
	for _, w := range a.waiters {
		depth[w.class]++
	}
	return map[string]any{"free_slots": a.free, "depth": depth, "admitted": a.admitted, "total_wait_ns": a.waitNs}
}

// withAdmission queues each call by its priority header before it reaches the
// proxy. A caller that gives up (context cancelled) leaves the queue.
func withAdmission(next http.Handler, a *Admission) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		class := defaultClass
		if strings.EqualFold(r.Header.Get(RoleHeader), RoleTeacher) {
			class = classes - 1 // P3: teacher batches never delay interactive work
		}
		if v, err := strconv.Atoi(r.Header.Get(PriorityHeader)); err == nil {
			class = v
		}
		release, err := a.Acquire(r.Context(), class)
		if err != nil {
			if errors.Is(err, ErrQueueFull) {
				w.Header().Set("Retry-After", "5")
				jsonError(w, http.StatusTooManyRequests, err.Error())
			}
			return
		}
		defer release()
		next.ServeHTTP(w, r)
	})
}

func registerQueueRoutes(mux *http.ServeMux, a *Admission) {
	mux.HandleFunc("GET /v1/queue/stats", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(a.Snapshot())
	})
}
