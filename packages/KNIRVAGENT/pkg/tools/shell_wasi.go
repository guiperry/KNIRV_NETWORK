//go:build wasip1 && wasm

package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func executeHostCommand(ctx context.Context, req HostCommandRequest) (string, error) {
	bridgeURL := strings.TrimRight(os.Getenv("KNIRVAGENT_HOST_BRIDGE_URL"), "/")
	if bridgeURL == "" {
		return "", fmt.Errorf("exec is unavailable in WASI without KNIRVAGENT_HOST_BRIDGE_URL")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	timeout := time.Duration(req.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, bridgeURL+"/exec", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("host bridge exec failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("host bridge exec returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result struct {
		Output string `json:"output"`
		Stdout string `json:"stdout"`
		Stderr string `json:"stderr"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return string(respBody), nil
	}
	if result.Error != "" {
		return "", fmt.Errorf(result.Error)
	}
	if result.Output != "" {
		return result.Output, nil
	}
	if result.Stderr != "" {
		return result.Stdout + "\nSTDERR:\n" + result.Stderr, nil
	}
	return result.Stdout, nil
}
