package inferencer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	models "github.com/guiperry/knirv-sdk-go/models"
)

// HeartTextGenerator implements InternalTextGenerator over the models daemon's
// HEART routes (/models/heart/*). The daemon is reached through the SDK client
// (a unix socket in production), so the generator carries no URL or port of its
// own.
type HeartTextGenerator struct {
	client *models.Client

	mu        sync.RWMutex
	ready     bool
	progress  float64
	lastCheck time.Time
}

// NewHeartTextGenerator creates a generator that delegates to HEART through c.
func NewHeartTextGenerator(c *models.Client) *HeartTextGenerator {
	return &HeartTextGenerator{client: c}
}

// IsReady asks HEART whether its internal model is trained. It never builds HEART
// (the daemon answers "idle" until something has needed it), and an unreachable
// daemon is simply "not ready".
func (g *HeartTextGenerator) IsReady() bool {
	g.mu.RLock()
	if time.Since(g.lastCheck) <= switcherCheckInterval {
		ready := g.ready
		g.mu.RUnlock()
		return ready
	}
	g.mu.RUnlock()

	g.mu.Lock()
	defer g.mu.Unlock()
	if time.Since(g.lastCheck) <= switcherCheckInterval {
		return g.ready
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	health, err := g.client.HeartHealth(ctx)
	g.lastCheck = time.Now()
	if err != nil {
		log.Printf("[heart-generator] health check failed: %v", err)
		g.ready, g.progress = false, 0
		return false
	}
	g.ready, g.progress = health.Ready, health.Progress
	if g.ready {
		log.Printf("[heart-generator] internal model ready (progress=%.1f%%)", g.progress*100)
	}
	return g.ready
}

// Progress returns the last known training progress from HEART.
func (g *HeartTextGenerator) Progress() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.progress
}

// GenerateText returns HEART's text for the prompt. When HEART answers "not
// ready" the cached readiness is dropped so the switcher falls back to the
// external chain at once instead of waiting out its check interval.
func (g *HeartTextGenerator) GenerateText(ctx context.Context, prompt string) (string, error) {
	text, err := g.client.HeartGenerate(ctx, prompt)
	if errors.Is(err, models.ErrNotReady) {
		g.mu.Lock()
		g.ready = false
		g.mu.Unlock()
		return "", fmt.Errorf("heart not ready: %w", err)
	}
	if err != nil {
		return "", fmt.Errorf("heart generate: %w", err)
	}
	return text, nil
}
