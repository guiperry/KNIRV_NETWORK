package inferencer

import (
	"sync"
	"time"
)

// Breaker is a per-model circuit breaker. After `threshold` consecutive failures it
// opens and refuses calls for `cooldown`; then it lets exactly one probe through
// (half-open). A success closes it, a failed probe reopens it.
type Breaker struct {
	mu        sync.Mutex
	threshold int
	cooldown  time.Duration
	now       func() time.Time

	failures int
	openedAt time.Time
	open     bool
	probing  bool
}

// NewBreaker makes a breaker opening after threshold consecutive failures.
func NewBreaker(threshold int, cooldown time.Duration) *Breaker {
	if threshold < 1 {
		threshold = 1
	}
	return &Breaker{threshold: threshold, cooldown: cooldown, now: time.Now}
}

// Allow reports whether a call could go ahead now, without taking the half-open probe.
// It is for deciding who is eligible; Acquire is for actually calling.
func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.open {
		return true
	}
	return b.now().Sub(b.openedAt) >= b.cooldown && !b.probing
}

// Acquire is Allow that also takes the half-open probe: only one caller gets true
// until its outcome is reported with Success or Failure.
func (b *Breaker) Acquire() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.open {
		return true
	}
	if b.now().Sub(b.openedAt) < b.cooldown || b.probing {
		return false
	}
	b.probing = true
	return true
}

// Success closes the breaker.
func (b *Breaker) Success() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures, b.open, b.probing = 0, false, false
}

// Failure counts a failed call and opens the breaker at the threshold.
func (b *Breaker) Failure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	b.probing = false
	if b.open || b.failures >= b.threshold {
		b.open = true
		b.openedAt = b.now()
	}
}

// State is "closed", "open" or "half-open".
func (b *Breaker) State() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch {
	case !b.open:
		return "closed"
	case b.now().Sub(b.openedAt) >= b.cooldown:
		return "half-open"
	}
	return "open"
}
