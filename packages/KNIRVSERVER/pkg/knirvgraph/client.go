package knirvgraph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	client  *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type GraphNode struct {
	NodeID    string                 `json:"node_id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

type GraphEdge struct {
	EdgeID    string                 `json:"edge_id"`
	SourceID  string                 `json:"source_id"`
	TargetID  string                 `json:"target_id"`
	Type      string                 `json:"type"`
	Weight    float64                `json:"weight"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

type CommitRequest struct {
	Node    GraphNode `json:"node"`
	Message string    `json:"message"`
	Author  string    `json:"author"`
}

type CommitResponse struct {
	Success    bool   `json:"success"`
	CommitHash string `json:"commit_hash,omitempty"`
	Error      string `json:"error,omitempty"`
}

type BulkEdgeRequest struct {
	Edges   []GraphEdge `json:"edges"`
	Message string      `json:"message"`
	Author  string      `json:"author"`
}

type BulkEdgeResponse struct {
	Success    bool   `json:"success"`
	CommitHash string `json:"commit_hash,omitempty"`
	EdgeCount  int    `json:"edge_count"`
	Error      string `json:"error,omitempty"`
}

func (c *Client) CommitNode(ctx context.Context, node GraphNode, message, author string) error {
	reqBody := CommitRequest{
		Node:    node,
		Message: message,
		Author:  author,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/commit", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) CreateEdge(ctx context.Context, edge GraphEdge) error {
	body, err := json.Marshal(edge)
	if err != nil {
		return fmt.Errorf("failed to marshal edge: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/edge", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create edge request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send edge request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var be BulkEdgeResponse
		if json.NewDecoder(resp.Body).Decode(&be) == nil && be.Error != "" {
			return fmt.Errorf("edge create failed: %s", be.Error)
		}
		return fmt.Errorf("edge create returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) BulkCommit(ctx context.Context, edges []GraphEdge, message, author string) (*BulkEdgeResponse, error) {
	reqBody := BulkEdgeRequest{
		Edges:   edges,
		Message: message,
		Author:  author,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bulk edge request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/edges/bulk", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create bulk edge request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send bulk edge request: %w", err)
	}
	defer resp.Body.Close()

	var result BulkEdgeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode bulk edge response: %w", err)
	}

	return &result, nil
}

func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}
