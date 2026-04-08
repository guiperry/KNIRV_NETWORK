package knirvoracle

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	client  *http.Client
}

func NewClient(socketPath string) *Client {
	if socketPath == "" {
		socketPath = "/var/run/knirv/oracle.sock"
	}

	return &Client{
		baseURL: "http://unix",
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					dialer := net.Dialer{}
					return dialer.DialContext(ctx, "unix", socketPath)
				},
			},
		},
	}
}

func (c *Client) GetBalance(address string) (string, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/oracle/v3/token/balance/%s", c.baseURL, address))
	if err != nil {
		return "", fmt.Errorf("failed to get balance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oracle returned status %d", resp.StatusCode)
	}

	var result struct {
		Balance string `json:"balance"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Balance, nil
}

func (c *Client) GetTokenInfo() (map[string]interface{}, error) {
	resp, err := c.client.Get(c.baseURL + "/oracle/v3/token/info")
	if err != nil {
		return nil, fmt.Errorf("failed to get token info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oracle returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

func (c *Client) SubmitRollup(batchID, data string) (string, error) {
	return "", nil
}

func (c *Client) GetRollup(id string) (map[string]interface{}, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/oracle/v3/rollups/%s", c.baseURL, id))
	if err != nil {
		return nil, fmt.Errorf("failed to get rollup: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oracle returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

func (c *Client) Health() error {
	resp, err := c.client.Get(c.baseURL + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("oracle unhealthy: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) GetStatus() (map[string]interface{}, error) {
	resp, err := c.client.Get(c.baseURL + "/oracle/v3/status")
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oracle returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
