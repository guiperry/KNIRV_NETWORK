package knirvproof

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

var (
	ErrUnauthorized    = errors.New("authentication required")
	ErrForbidden       = errors.New("project action forbidden")
	ErrAuthUnavailable = errors.New("authorization service unavailable")
)

const (
	ActionCASRead     = "cas:read"
	ActionCASWrite    = "cas:write"
	ActionProofRead   = "proof:read"
	ActionProofSubmit = "proof:submit"
	ActionProofShare  = "proof:share"
)

type Principal struct {
	ID string `json:"id"`
}

type Authorizer interface {
	Authorize(context.Context, string, string, string) (*Principal, error)
}

type AuthorizerFunc func(context.Context, string, string, string) (*Principal, error)

func (fn AuthorizerFunc) Authorize(ctx context.Context, token, projectID, action string) (*Principal, error) {
	return fn(ctx, token, projectID, action)
}

type BackendAuthorizer struct {
	client        *http.Client
	baseURL       string
	internalToken string
}

type BackendTrustClient struct {
	client        *http.Client
	baseURL       string
	internalToken string
}

func NewBackendAuthorizer(socketPath, internalToken string) (*BackendAuthorizer, error) {
	client, err := newBackendUnixHTTPClient(socketPath, internalToken)
	if err != nil {
		return nil, err
	}
	return &BackendAuthorizer{
		client: client, baseURL: "http://backend", internalToken: internalToken,
	}, nil
}

func NewBackendTrustClient(socketPath, internalToken string) (*BackendTrustClient, error) {
	client, err := newBackendUnixHTTPClient(socketPath, internalToken)
	if err != nil {
		return nil, err
	}
	return &BackendTrustClient{
		client: client, baseURL: "http://backend", internalToken: internalToken,
	}, nil
}

func newBackendUnixHTTPClient(socketPath, internalToken string) (*http.Client, error) {
	if strings.TrimSpace(socketPath) == "" {
		return nil, fmt.Errorf("backend socket is required")
	}
	if strings.TrimSpace(internalToken) == "" {
		return nil, fmt.Errorf("internal authorization token is required")
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	return &http.Client{Transport: transport, Timeout: 5 * time.Second}, nil
}

func (a *BackendAuthorizer) Authorize(ctx context.Context, bearerToken, projectID, action string) (*Principal, error) {
	if strings.TrimSpace(bearerToken) == "" {
		return nil, ErrUnauthorized
	}
	body, err := json.Marshal(map[string]string{"project_id": projectID, "action": action})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/internal/v1/knirv/authorize", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+bearerToken)
	request.Header.Set("X-KNIRV-Internal-Token", a.internalToken)
	response, err := a.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthUnavailable, err)
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusForbidden, http.StatusNotFound:
		return nil, ErrForbidden
	case http.StatusOK:
	default:
		return nil, fmt.Errorf("%w: backend returned %d", ErrAuthUnavailable, response.StatusCode)
	}
	var result struct {
		Allowed     bool   `json:"allowed"`
		PrincipalID string `json:"principal_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%w: invalid backend response", ErrAuthUnavailable)
	}
	if !result.Allowed {
		return nil, ErrForbidden
	}
	if result.PrincipalID == "" {
		return nil, fmt.Errorf("%w: backend omitted principal", ErrAuthUnavailable)
	}
	return &Principal{ID: result.PrincipalID}, nil
}

func (c *BackendTrustClient) ResolveDEK(ctx context.Context, submission ProofSubmission) ([]byte, error) {
	body, err := json.Marshal(struct {
		ProjectID    string        `json:"project_id"`
		ProofRoot    string        `json:"proof_root"`
		KeyEnvelopes []KeyEnvelope `json:"key_envelopes"`
	}{ProjectID: submission.ProjectID, ProofRoot: submission.ProofRoot, KeyEnvelopes: submission.KeyEnvelopes})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/v1/knirv/unwrap-proof-key", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-KNIRV-Internal-Token", c.internalToken)
	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("key broker returned %d", response.StatusCode)
	}
	var result struct {
		DEK string `json:"dek"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, err
	}
	dek, err := base64.StdEncoding.DecodeString(result.DEK)
	if err != nil {
		return nil, fmt.Errorf("decode data-encryption key: %w", err)
	}
	return dek, nil
}

func (c *BackendTrustClient) ValidatorKey(ctx context.Context) (*ValidatorKey, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/internal/v1/knirv/validator-key", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-KNIRV-Internal-Token", c.internalToken)
	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("key broker returned %d", response.StatusCode)
	}
	var key ValidatorKey
	if err := json.NewDecoder(response.Body).Decode(&key); err != nil {
		return nil, err
	}
	if key.RecipientID == "" || key.KeyID == "" || key.Algorithm != "x25519-xchacha20poly1305" || key.WrapSchema != "knirv.proof-key-envelope.v1" || key.PublicKey == "" {
		return nil, fmt.Errorf("key broker returned invalid validator metadata")
	}
	return &key, nil
}

func (c *BackendTrustClient) ResolveSigningKey(keyID string) (ed25519.PublicKey, bool) {
	if strings.TrimSpace(keyID) == "" {
		return nil, false
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.baseURL+"/internal/v1/knirv/signing-keys/"+keyID, nil)
	if err != nil {
		return nil, false
	}
	request.Header.Set("X-KNIRV-Internal-Token", c.internalToken)
	response, err := c.client.Do(request)
	if err != nil {
		return nil, false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, false
	}
	var result struct {
		PublicKey string `json:"public_key"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, false
	}
	raw, err := base64.StdEncoding.DecodeString(result.PublicKey)
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, false
	}
	return ed25519.PublicKey(raw), true
}

func bearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", ErrUnauthorized
	}
	return parts[1], nil
}
