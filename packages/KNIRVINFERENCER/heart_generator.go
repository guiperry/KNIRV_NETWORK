package inferencer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// heartGenerateRequest is sent to HEART's /heart/generate endpoint.
type heartGenerateRequest struct {
	Prompt string `json:"prompt"`
}

// heartGenerateResponse is returned by HEART's /heart/generate endpoint.
type heartGenerateResponse struct {
	Text   string `json:"text,omitempty"`
	Error  string `json:"error,omitempty"`
	Ready  bool   `json:"ready"`
}

// heartHealthResponse is returned by HEART's /heart/health endpoint.
type heartHealthResponse struct {
	Status   string  `json:"status"`
	Ready    bool    `json:"ready"`
	Progress float64 `json:"progress"`
}

// HeartTextGenerator implements InternalTextGenerator by calling the
// HEART HTTP service (separate process, no cross-package import needed).
type HeartTextGenerator struct {
	baseURL    string
	httpClient *http.Client
	ready      bool
	progress   float64
	lastCheck  time.Time
	mu         sync.RWMutex
}

// NewHeartTextGenerator creates a generator that delegates to the HEART
// service at the given base URL (e.g. "http://localhost:8090").
func NewHeartTextGenerator(baseURL string) *HeartTextGenerator {
	return &HeartTextGenerator{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// IsReady checks HEART's /heart/health to see if the internal model is trained.
func (g *HeartTextGenerator) IsReady() bool {
	g.mu.RLock()
	now := time.Now()
	if now.Before(g.lastCheck.Add(switcherCheckInterval)) {
		ready := g.ready
		g.mu.RUnlock()
		return ready
	}
	g.mu.RUnlock()

	g.mu.Lock()
	defer g.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Since(g.lastCheck) <= switcherCheckInterval {
		return g.ready
	}

	url := g.baseURL + "/heart/health"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Printf("[heart-generator] health check request error: %v", err)
		g.ready = false
		g.lastCheck = time.Now()
		return false
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		log.Printf("[heart-generator] health check failed: %v", err)
		g.ready = false
		g.lastCheck = time.Now()
		return false
	}
	defer resp.Body.Close()

	var health heartHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		log.Printf("[heart-generator] health decode error: %v", err)
		g.ready = false
		g.lastCheck = time.Now()
		return false
	}

	g.ready = health.Ready
	g.progress = health.Progress
	g.lastCheck = time.Now()

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

// GenerateText sends a prompt to HEART's /heart/generate endpoint and
// returns the generated text.
func (g *HeartTextGenerator) GenerateText(ctx context.Context, prompt string) (string, error) {
	url := g.baseURL + "/heart/generate"

	body := heartGenerateRequest{Prompt: prompt}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("heart marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("heart create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("heart request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("heart read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("heart returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result heartGenerateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("heart unmarshal response: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("heart generate error: %s", result.Error)
	}

	return result.Text, nil
}
