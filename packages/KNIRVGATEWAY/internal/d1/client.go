// Package d1 is a minimal client for Cloudflare's D1 HTTP query API
// (https://api.cloudflare.com/client/v4/accounts/{account_id}/d1/database/{database_id}/query).
// KNIRVGATEWAY is a plain Go binary, not a Cloudflare Worker, so it cannot
// use D1's native binding (that only exists inside the Workers/Pages
// Functions runtime) — this REST client is the only way for a Go process to
// reach D1 directly. See internal/rootkey for where the account ID and API
// token this client needs come from.
package d1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.cloudflare.com"

// Client queries one D1 database over Cloudflare's REST API.
type Client struct {
	AccountID  string
	DatabaseID string
	APIToken   string

	// BaseURL overrides the Cloudflare API origin; empty means
	// defaultBaseURL. Only ever set in tests.
	BaseURL string

	HTTPClient *http.Client
}

// NewClient builds a Client with a sane default HTTP timeout.
func NewClient(accountID, databaseID, apiToken string) *Client {
	return &Client{
		AccountID:  accountID,
		DatabaseID: databaseID,
		APIToken:   apiToken,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type queryRequest struct {
	SQL    string `json:"sql"`
	Params []any  `json:"params,omitempty"`
}

type cloudflareError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type queryResultMeta struct {
	Changes   int64 `json:"changes"`
	LastRowID int64 `json:"last_row_id"`
}

type queryResultEntry struct {
	Results []map[string]any `json:"results"`
	Success bool             `json:"success"`
	Meta    queryResultMeta  `json:"meta"`
}

type queryResponse struct {
	Success  bool               `json:"success"`
	Errors   []cloudflareError  `json:"errors"`
	Result   []queryResultEntry `json:"result"`
	Messages []any              `json:"messages"`
}

// QueryResult is the caller-facing shape of a single statement's outcome.
type QueryResult struct {
	// Rows holds every returned row as column-name -> value. Empty (not
	// nil) for statements that return no rows (INSERT/UPDATE/DELETE).
	Rows []map[string]any
	// Changes is the number of rows affected by an INSERT/UPDATE/DELETE.
	Changes int64
	// LastRowID is the SQLite last_insert_rowid() after an INSERT.
	LastRowID int64
}

func (c *Client) baseURL() string {
	if strings.TrimSpace(c.BaseURL) != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return defaultBaseURL
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// Query executes a single SQL statement with positional `?` placeholders
// bound to params, in order. Works for both reads (SELECT — Rows populated)
// and writes (INSERT/UPDATE/DELETE — Changes/LastRowID populated).
//
// Every caller in this codebase MUST use `?` placeholders and pass user
// input via params, never string-concatenate SQL — this is the only
// injection boundary D1's REST API gives us.
func (c *Client) Query(ctx context.Context, sql string, params ...any) (*QueryResult, error) {
	if strings.TrimSpace(c.AccountID) == "" {
		return nil, fmt.Errorf("d1: AccountID is empty")
	}
	if strings.TrimSpace(c.DatabaseID) == "" {
		return nil, fmt.Errorf("d1: DatabaseID is empty")
	}
	if strings.TrimSpace(c.APIToken) == "" {
		return nil, fmt.Errorf("d1: APIToken is empty")
	}

	reqBody, err := json.Marshal(queryRequest{SQL: sql, Params: params})
	if err != nil {
		return nil, fmt.Errorf("d1: marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/client/v4/accounts/%s/d1/database/%s/query", c.baseURL(), c.AccountID, c.DatabaseID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("d1: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIToken)

	resp, err := c.httpClient().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("d1: request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("d1: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("d1: HTTP %d: %s", resp.StatusCode, truncate(string(bodyBytes), 500))
	}

	var parsed queryResponse
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil, fmt.Errorf("d1: decode response: %w", err)
	}

	if !parsed.Success {
		return nil, fmt.Errorf("d1: query failed: %s", formatCloudflareErrors(parsed.Errors))
	}
	if len(parsed.Result) == 0 {
		return &QueryResult{Rows: []map[string]any{}}, nil
	}

	entry := parsed.Result[0]
	if !entry.Success {
		return nil, fmt.Errorf("d1: statement failed: %s", formatCloudflareErrors(parsed.Errors))
	}

	rows := entry.Results
	if rows == nil {
		rows = []map[string]any{}
	}
	return &QueryResult{
		Rows:      rows,
		Changes:   entry.Meta.Changes,
		LastRowID: entry.Meta.LastRowID,
	}, nil
}

func formatCloudflareErrors(errs []cloudflareError) string {
	if len(errs) == 0 {
		return "unknown error"
	}
	parts := make([]string, 0, len(errs))
	for _, e := range errs {
		parts = append(parts, fmt.Sprintf("[%d] %s", e.Code, e.Message))
	}
	return strings.Join(parts, "; ")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
