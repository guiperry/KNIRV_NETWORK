package server

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

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
	"go.uber.org/zap"
)

type modelCallPolicyRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type modelCallPolicyResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason"`
}

type meResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type creationView struct {
	OwnerID  string `json:"owner_id"`
	FailOpen bool   `json:"fail_open"`
}

func newModelCallProxy(cfg *config.Config, logger *zap.Logger) (http.Handler, error) {
	if cfg.CLIProxyAPIBaseURL == "" {
		return nil, nil
	}
	proxy, err := newHTTPProxy(cfg.CLIProxyAPIBaseURL)
	if err != nil {
		return nil, fmt.Errorf("configure CLIProxyAPI proxy: %w", err)
	}

	authorizer := newBackendSessionAuthorizer(cfg.BackendSocketPath)
	backendClient := newBackendClient(cfg.BackendSocketPath)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/gateway/model-proxy/"), "/", 3)
		if len(vars) < 2 {
			http.Error(w, "invalid model-proxy path", http.StatusBadRequest)
			return
		}
		creationID := vars[0]
		provider := vars[1]

		authorization := r.Header.Get("Authorization")
		if authorization == "" {
			authorization = r.Header.Get("x-api-key")
		}
		authorization = strings.TrimSpace(authorization)
		if authorization == "" {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		parts := strings.Fields(authorization)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			http.Error(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}
		bearer := authorization

		if authorizer == nil {
			http.Error(w, "authorization service is not configured", http.StatusServiceUnavailable)
			return
		}
		if err := authorizer.Authorize(r.Context(), bearer); err != nil {
			if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
				http.Error(w, "invalid or expired session token", http.StatusUnauthorized)
			} else {
				http.Error(w, "authorization service unavailable: "+err.Error(), http.StatusServiceUnavailable)
			}
			return
		}

		creation, err := fetchCreation(backendClient, creationID, bearer)
		if err != nil {
			status := http.StatusInternalServerError
			if err == errCreationNotFound {
				status = http.StatusNotFound
			} else if err == errForbidden {
				status = http.StatusForbidden
			}
			http.Error(w, err.Error(), status)
			return
		}

		me, err := fetchMe(backendClient, bearer)
		if err != nil {
			http.Error(w, "failed to resolve authenticated user: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		if me.UserID != creation.OwnerID {
			http.Error(w, "forbidden: you do not own this creation", http.StatusForbidden)
			return
		}

		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, err = io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read request body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		var model string
		if len(bodyBytes) > 0 {
			var req modelCallPolicyRequest
			if err := json.Unmarshal(bodyBytes, &req); err == nil {
				model = req.Model
			}
		}

		allowed, reason, err := evaluateModelPolicy(backendClient, creationID, provider, model, bearer)
		if err != nil {
			if creation.FailOpen {
				logger.Warn("model policy evaluation failed, failing open",
					zap.String("creation_id", creationID),
					zap.Error(err))
				allowed = true
			} else {
				logger.Warn("model policy evaluation failed, failing closed",
					zap.String("creation_id", creationID),
					zap.Error(err))
				http.Error(w, fmt.Sprintf("policy evaluation unavailable: %v", err), http.StatusServiceUnavailable)
				return
			}
		}
		if !allowed {
			http.Error(w, fmt.Sprintf("model call blocked by policy: %s", reason), http.StatusForbidden)
			return
		}

		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/gateway/model-proxy/"+creationID+"/"+provider)
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
		r.URL.RawPath = ""
		proxy.ServeHTTP(w, r)
	}), nil
}

func newBackendClient(socketPath string) *http.Client {
	if strings.TrimSpace(socketPath) == "" {
		return http.DefaultClient
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
	}}
	return &http.Client{Transport: transport, Timeout: 10 * time.Second}
}

var errCreationNotFound = fmt.Errorf("creation not found")
var errForbidden = fmt.Errorf("forbidden")

func fetchCreation(client *http.Client, creationID, bearer string) (*creationView, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://backend/api/dve-creation/nodes/"+creationID, nil)
	if err != nil {
		return nil, fmt.Errorf("create creation request: %w", err)
	}
	req.Header.Set("Authorization", bearer)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch creation: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusNotFound {
		return nil, errCreationNotFound
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, errForbidden
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch creation returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var raw struct {
		Success bool `json:"success"`
		Data    struct {
			OwnerID string `json:"owner_id"`
			Policy  struct {
				FailOpen bool `json:"fail_open"`
			} `json:"policy"`
		} `json:"data"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode creation: %w", err)
	}
	if !raw.Success {
		return nil, fmt.Errorf("creation fetch failed: %s", raw.Error)
	}
	return &creationView{
		OwnerID:  raw.Data.OwnerID,
		FailOpen: raw.Data.Policy.FailOpen,
	}, nil
}

func fetchMe(client *http.Client, bearer string) (*meResponse, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://backend/api/auth/me", nil)
	if err != nil {
		return nil, fmt.Errorf("create me request: %w", err)
	}
	req.Header.Set("Authorization", bearer)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch me: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("me endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var me meResponse
	if err := json.Unmarshal(body, &me); err != nil {
		return nil, fmt.Errorf("decode me: %w", err)
	}
	return &me, nil
}

func evaluateModelPolicy(client *http.Client, creationID, provider, model, bearer string) (bool, string, error) {
	reqBody := modelCallPolicyRequest{Provider: provider, Model: model}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return false, "", fmt.Errorf("marshal policy request: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		"http://backend/api/dve-creation/nodes/"+creationID+"/policy/evaluate",
		bytes.NewReader(bodyBytes))
	if err != nil {
		return false, "", fmt.Errorf("create policy request: %w", err)
	}
	req.Header.Set("Authorization", bearer)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("policy evaluation request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("policy evaluation returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var policyResp modelCallPolicyResponse
	if err := json.Unmarshal(respBody, &policyResp); err != nil {
		return false, "", fmt.Errorf("decode policy response: %w", err)
	}
	return policyResp.Allowed, policyResp.Reason, nil
}
