package payment

import (
	"sync"
	"time"
)

// FaucetLimiter manages faucet request rate limiting
type FaucetLimiter struct {
	cooldown time.Duration
	requests map[string]time.Time
	mu       sync.RWMutex
}

// NewFaucetLimiter creates a new faucet limiter
func NewFaucetLimiter(cooldown time.Duration) *FaucetLimiter {
	return &FaucetLimiter{
		cooldown: cooldown,
		requests: make(map[string]time.Time),
	}
}

// Allow checks if a request is allowed for the given network and address
func (fl *FaucetLimiter) Allow(network, address string) bool {
	fl.mu.RLock()
	defer fl.mu.RUnlock()

	key := network + ":" + address
	lastRequest, exists := fl.requests[key]

	if !exists {
		return true
	}

	return time.Since(lastRequest) >= fl.cooldown
}

// TimeLeft returns the time left before the next request is allowed
func (fl *FaucetLimiter) TimeLeft(network, address string) time.Duration {
	fl.mu.RLock()
	defer fl.mu.RUnlock()

	key := network + ":" + address
	lastRequest, exists := fl.requests[key]

	if !exists {
		return 0
	}

	elapsed := time.Since(lastRequest)
	if elapsed >= fl.cooldown {
		return 0
	}

	return fl.cooldown - elapsed
}

// RecordRequest records a successful faucet request
func (fl *FaucetLimiter) RecordRequest(network, address string) {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	key := network + ":" + address
	fl.requests[key] = time.Now()
}

// Cleanup removes old entries (optional, for memory management)
func (fl *FaucetLimiter) Cleanup() {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	now := time.Now()
	for key, timestamp := range fl.requests {
		if now.Sub(timestamp) > fl.cooldown*2 {
			delete(fl.requests, key)
		}
	}
}
