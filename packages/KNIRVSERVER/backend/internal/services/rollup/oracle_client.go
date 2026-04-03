package rollup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OracleSettlementClient interface {
	SubmitRollup(batch *RollupBatch) (string, error)
	GetRollup(id string) (map[string]interface{}, error)
	FinalizeRollup(id string) error
	DisputeRollup(id string, reason string) error
}

type HTTPOracleClient struct {
	baseURL string
	client  *http.Client
}

func NewOracleClient(baseURL string) *HTTPOracleClient {
	return &HTTPOracleClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *HTTPOracleClient) post(path string, payload interface{}, out interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal oracle payload: %w", err)
	}

	resp, err := c.client.Post(c.baseURL+path, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("oracle request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("oracle request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	if out == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode oracle response: %w", err)
	}
	return nil
}

func (c *HTTPOracleClient) SubmitRollup(batch *RollupBatch) (string, error) {
	var result struct {
		ID string `json:"id"`
	}
	if err := c.post("/oracle/v3/rollups/submit", batch, &result); err != nil {
		return "", err
	}
	return result.ID, nil
}

func (c *HTTPOracleClient) GetRollup(id string) (map[string]interface{}, error) {
	resp, err := c.client.Get(c.baseURL + "/oracle/v3/rollups/" + id)
	if err != nil {
		return nil, fmt.Errorf("oracle get rollup failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("oracle get rollup failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode oracle rollup response: %w", err)
	}
	return result, nil
}

func (c *HTTPOracleClient) FinalizeRollup(id string) error {
	return c.post("/oracle/v3/rollups/"+id+"/finalize", map[string]string{"id": id}, nil)
}

func (c *HTTPOracleClient) DisputeRollup(id string, reason string) error {
	return c.post("/oracle/v3/rollups/"+id+"/dispute", map[string]string{
		"id":     id,
		"reason": reason,
	}, nil)
}
