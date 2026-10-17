package adapters

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrReloading is returned to a request that arrived while llama-server was being
// restarted and could not be served within the wait budget.
var ErrReloading = errors.New("llama-server is reloading its adapter set; retry shortly")

// Gate lets requests run concurrently but lets a restart wait for them all to
// finish, and hold new ones back while it happens.
type Gate struct {
	mu       sync.Mutex
	cond     *sync.Cond
	inflight int
	paused   bool
}

func NewGate() *Gate {
	g := &Gate{}
	g.cond = sync.NewCond(&g.mu)
	return g
}

// Enter admits a request, waiting out a pause for at most wait.
func (g *Gate) Enter(ctx context.Context, wait time.Duration) (leave func(), err error) {
	deadline := time.Now().Add(wait)
	g.mu.Lock()
	defer g.mu.Unlock()
	for g.paused {
		if time.Now().After(deadline) || ctx.Err() != nil {
			return nil, ErrReloading
		}
		g.waitUntil(deadline)
	}
	g.inflight++
	return func() {
		g.mu.Lock()
		g.inflight--
		g.cond.Broadcast()
		g.mu.Unlock()
	}, nil
}

// waitUntil is cond.Wait with a wake-up at deadline (sync.Cond has no timeout).
func (g *Gate) waitUntil(deadline time.Time) {
	t := time.AfterFunc(time.Until(deadline), func() {
		g.mu.Lock()
		g.cond.Broadcast()
		g.mu.Unlock()
	})
	g.cond.Wait()
	t.Stop()
}

// Pause stops admitting requests and waits for in-flight ones to finish, up to
// drain. On timeout it resumes and reports how many were still running, so a
// restart never kills work silently.
func (g *Gate) Pause(drain time.Duration) (resume func(), err error) {
	deadline := time.Now().Add(drain)
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.paused {
		return nil, errors.New("a reload is already in progress")
	}
	g.paused = true
	for g.inflight > 0 {
		if time.Now().After(deadline) {
			n := g.inflight
			g.paused = false
			g.cond.Broadcast()
			return nil, &StillBusyError{InFlight: n}
		}
		g.waitUntil(deadline)
	}
	return func() {
		g.mu.Lock()
		g.paused = false
		g.cond.Broadcast()
		g.mu.Unlock()
	}, nil
}

// StillBusyError means in-flight requests did not finish within the drain budget.
type StillBusyError struct{ InFlight int }

func (e *StillBusyError) Error() string {
	return "requests are still in flight; the adapter set was not changed"
}
