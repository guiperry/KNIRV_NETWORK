package dvemanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"backend_server/internal/objects"
)

// DVEClientResponse represents the response from a remote DVE instance
type DVEClientResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Message string                 `json:"message,omitempty"`
}

// DVENodeListResponse represents a list of DVE nodes from a remote instance
type DVENodeListResponse struct {
	Nodes []objects.DVENode `json:"nodes"`
	Total int               `json:"total"`
}

// DVENodeResponse represents a single DVE node response from a remote instance
type DVENodeResponse struct {
	Node *objects.DVENode `json:"node"`
}

// DVEHealthResponse represents health status of a remote DVE instance
type DVEHealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
}

// DVEClient handles communication with remote DVE instances
type DVEClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int
	backoff    time.Duration
}

// NewDVEClient creates a new DVE client for a remote instance
func NewDVEClient(baseURL string, timeout time.Duration, maxRetries int, backoff time.Duration) *DVEClient {
	return &DVEClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout:    timeout,
		maxRetries: maxRetries,
		backoff:    backoff,
	}
}

// HealthCheck checks if the remote DVE instance is healthy
func (dc *DVEClient) HealthCheck(ctx context.Context) (bool, error) {
	resp, err := dc.doRequest(ctx, "GET", "/api/health", nil)
	if err != nil {
		return false, fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("health check returned status %d: %s", resp.StatusCode, string(body))
	}

	var healthResp DVEHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return false, fmt.Errorf("failed to decode health response: %w", err)
	}

	return healthResp.Status == "healthy", nil
}

// GetNodes fetches available nodes from the remote DVE instance
func (dc *DVEClient) GetNodes(ctx context.Context, status string, limit int) ([]objects.DVENode, error) {
	url := fmt.Sprintf("/api/dve-nodes?status=%s&limit=%d", status, limit)
	resp, err := dc.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get nodes returned status %d: %s", resp.StatusCode, string(body))
	}

	var nodeResp DVENodeListResponse
	if err := json.NewDecoder(resp.Body).Decode(&nodeResp); err != nil {
		return nil, fmt.Errorf("failed to decode nodes response: %w", err)
	}

	return nodeResp.Nodes, nil
}

// GetNode fetches a specific node by ID from the remote DVE instance
func (dc *DVEClient) GetNode(ctx context.Context, nodeID string) (*objects.DVENode, error) {
	url := fmt.Sprintf("/api/dve-nodes/%s", nodeID)
	resp, err := dc.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get node returned status %d: %s", resp.StatusCode, string(body))
	}

	var nodeResp DVENodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&nodeResp); err != nil {
		return nil, fmt.Errorf("failed to decode node response: %w", err)
	}

	return nodeResp.Node, nil
}

// GetNodeMetrics fetches performance metrics for a node
func (dc *DVEClient) GetNodeMetrics(ctx context.Context, nodeID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("/api/dve-nodes/%s/metrics", nodeID)
	resp, err := dc.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get node metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get metrics returned status %d: %s", resp.StatusCode, string(body))
	}

	var metricsResp struct {
		Metrics map[string]interface{} `json:"metrics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&metricsResp); err != nil {
		return nil, fmt.Errorf("failed to decode metrics response: %w", err)
	}

	return metricsResp.Metrics, nil
}

// SendHeartbeat sends a heartbeat to the remote DVE instance
func (dc *DVEClient) SendHeartbeat(ctx context.Context, nodeID string) error {
	payload := map[string]interface{}{
		"timestamp": time.Now(),
		"node_id":   nodeID,
	}

	url := fmt.Sprintf("/api/dve-nodes/%s/heartbeat", nodeID)
	resp, err := dc.doRequest(ctx, "POST", url, payload)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("heartbeat returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SubmitValidationTask submits a validation task to a remote DVE node
func (dc *DVEClient) SubmitValidationTask(ctx context.Context, nodeID string, task *objects.ValidationTask) error {
	url := fmt.Sprintf("/api/dve-nodes/%s/tasks", nodeID)
	resp, err := dc.doRequest(ctx, "POST", url, task)
	if err != nil {
		return fmt.Errorf("failed to submit task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("submit task returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetValidationResults retrieves validation results from the remote DVE instance
func (dc *DVEClient) GetValidationResults(ctx context.Context, taskID string) (*objects.ValidationResult, error) {
	url := fmt.Sprintf("/api/validation-results/%s", taskID)
	resp, err := dc.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get validation results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get results returned status %d: %s", resp.StatusCode, string(body))
	}

	var result objects.ValidationResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode results response: %w", err)
	}

	return &result, nil
}

// doRequest performs an HTTP request with retry logic
func (dc *DVEClient) doRequest(ctx context.Context, method string, endpoint string, payload interface{}) (*http.Response, error) {
	url := dc.baseURL + endpoint
	var attempt int
	var lastErr error

	for attempt = 0; attempt <= dc.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(dc.backoff * time.Duration(attempt)):
				// Wait for backoff before retrying
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		var body io.Reader
		if payload != nil {
			jsonData, err := json.Marshal(payload)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal payload: %w", err)
			}
			body = bytes.NewReader(jsonData)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "KNIRV-DVE-Manager/1.0")

		resp, err := dc.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: %w", attempt+1, err)
			log.Printf("DVE client request failed: %v", lastErr)
			continue
		}

		// Success on 2xx status codes
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		// Check if we should retry on error status codes
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusRequestTimeout {
			// Server error or timeout - retry
			resp.Body.Close()
			lastErr = fmt.Errorf("attempt %d: server returned status %d", attempt+1, resp.StatusCode)
			continue
		}

		// Client error (4xx) - don't retry
		return resp, nil
	}

	return nil, fmt.Errorf("all %d retries failed: %w", dc.maxRetries+1, lastErr)
}

// Close closes the DVE client connection
func (dc *DVEClient) Close() error {
	// HTTP client doesn't need explicit closing, but we can implement any cleanup if needed
	return nil
}
