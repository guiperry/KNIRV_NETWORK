package enroll

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// IssuedResponse is the body returned by the internal issue route.
type IssuedResponse struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	Issued Issued `json:"issued"`
}

// RequestIssuedChallenge asks the enrollment server on its Unix socket to issue a
// fresh nonce for an approved registration. It returns the nonce the registrant
// must sign and the root must approve.
func RequestIssuedChallenge(ctx context.Context, socketPath, registrationID, chainID string, ttl time.Duration) (Issued, error) {
	body, err := json.Marshal(map[string]any{
		"registration_id": registrationID,
		"chain_id":        chainID,
		"ttl_seconds":     int64(ttl / time.Second),
	})
	if err != nil {
		return Issued{}, err
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socketPath)
			},
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://unix"+IssuePath, bytes.NewReader(body))
	if err != nil {
		return Issued{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return Issued{}, fmt.Errorf("issue request: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	var out IssuedResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return Issued{}, fmt.Errorf("decode issue response (HTTP %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK || !out.OK {
		return Issued{}, fmt.Errorf("challenge issuance refused (HTTP %d): %s", resp.StatusCode, out.Error)
	}
	return out.Issued, nil
}
