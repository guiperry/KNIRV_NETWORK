package launcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"knirv-server/internal/enroll"
)

// backendEnrollmentAdmitter appends a verified enrollment to the registry-backed
// failover validator set by calling backend_server over the node-local Unix
// socket. The backend owns the failover publisher key; this process only holds
// the per-boot internal proof token.
type backendEnrollmentAdmitter struct {
	socketPath string
	token      string
	client     *http.Client
}

func newBackendEnrollmentAdmitter(socketPath, token string) *backendEnrollmentAdmitter {
	socketPath = strings.TrimSpace(socketPath)
	return &backendEnrollmentAdmitter{
		socketPath: socketPath,
		token:      strings.TrimSpace(token),
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "unix", socketPath)
				},
			},
			Timeout: 20 * time.Second,
		},
	}
}

func (a *backendEnrollmentAdmitter) Admit(ctx context.Context, admission enroll.ValidatorAdmission) error {
	if a.socketPath == "" {
		return fmt.Errorf("backend socket is not configured")
	}
	if a.token == "" {
		return fmt.Errorf("internal authorization token is not configured")
	}
	payload, err := json.Marshal(struct {
		ChainID        string `json:"chain_id"`
		RegistrationID string `json:"registration_id"`
		ValidatorID    string `json:"validator_id"`
		PublicKey      string `json:"public_key"`
	}{
		ChainID:        admission.ChainID,
		RegistrationID: admission.RegistrationID,
		ValidatorID:    admission.ValidatorID,
		PublicKey:      admission.PublicKey,
	})
	if err != nil {
		return fmt.Errorf("encode admission request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://backend/internal/v1/knirv/enroll/register", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build admission request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-KNIRV-Internal-Token", a.token)

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("call validator admission: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		return fmt.Errorf("validator admission failed (%d): %s", resp.StatusCode, message)
	}
	return nil
}
