package harness

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Admission is the inference-daemon's common gate. P0 is interactive, P1 is
// query-time work and P3 is bulk/teacher work. Aging keeps P3 from starving.
type Admission struct {
	mu      sync.Mutex
	free    int
	waiters []*admissionWaiter
	now     func() time.Time
}
type admissionWaiter struct {
	class int
	since time.Time
	ready chan struct{}
}

var ErrAdmissionFull = errors.New("inference admission queue is full")

const admissionMaxQueued = 64

func NewAdmission(slots int) *Admission {
	if slots < 1 {
		slots = 1
	}
	return &Admission{free: slots, now: time.Now}
}
func admissionClass(v int) int {
	if v < 0 {
		return 0
	}
	if v > 3 {
		return 3
	}
	return v
}
func (a *Admission) Acquire(ctx context.Context, class int) (func(), error) {
	class = admissionClass(class)
	a.mu.Lock()
	if a.free > 0 && len(a.waiters) == 0 {
		a.free--
		a.mu.Unlock()
		return a.release, nil
	}
	if len(a.waiters) >= admissionMaxQueued {
		a.mu.Unlock()
		return nil, ErrAdmissionFull
	}
	w := &admissionWaiter{class: class, since: a.now(), ready: make(chan struct{})}
	a.waiters = append(a.waiters, w)
	a.mu.Unlock()
	select {
	case <-w.ready:
		return a.release, nil
	case <-ctx.Done():
		a.mu.Lock()
		for i, queued := range a.waiters {
			if queued == w {
				a.waiters = append(a.waiters[:i], a.waiters[i+1:]...)
				break
			}
		}
		a.mu.Unlock()
		return nil, ctx.Err()
	}
}
func (a *Admission) release() { a.mu.Lock(); a.free++; a.grant(); a.mu.Unlock() }
func (a *Admission) grant() {
	for a.free > 0 && len(a.waiters) > 0 {
		best := 0
		bestClass := a.effective(a.waiters[0])
		for i := 1; i < len(a.waiters); i++ {
			if c := a.effective(a.waiters[i]); c < bestClass {
				best, bestClass = i, c
			}
		}
		w := a.waiters[best]
		a.waiters = append(a.waiters[:best], a.waiters[best+1:]...)
		a.free--
		close(w.ready)
	}
}
func (a *Admission) effective(w *admissionWaiter) int {
	age := int(a.now().Sub(w.since) / time.Minute)
	c := w.class - age
	if c < 0 {
		return 0
	}
	return c
}
