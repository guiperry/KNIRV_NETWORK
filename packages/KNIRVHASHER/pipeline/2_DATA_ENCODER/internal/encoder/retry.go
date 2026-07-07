package encoder

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"
)

const (
	maxRetries  = 5
	baseBackoff = 100 * time.Millisecond
	maxBackoff  = 30 * time.Second
)

func retryWithBackoff(ctx context.Context, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * baseBackoff
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			log.Printf("retry: attempt %d/%d, backing off %v", attempt, maxRetries, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return fmt.Errorf("retry cancelled: %w", ctx.Err())
			}
		}
		if err := fn(); err != nil {
			lastErr = err
			log.Printf("retry: attempt %d failed: %v", attempt, err)
			continue
		}
		return nil
	}
	return fmt.Errorf("all %d retries failed: %w", maxRetries, lastErr)
}
