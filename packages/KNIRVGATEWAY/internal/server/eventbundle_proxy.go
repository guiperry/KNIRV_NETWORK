package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

// newEventBundleMintProxy wraps chainProxy so CLI clients can mint a
// KNIRVCHAIN event-bundle NFT (chain_refactor.md §3.2/§4 Phase 2) through the
// gateway's single public entry point, without ever holding the internal
// service-to-service token KNIRVCHAIN's mint endpoint requires.
//
// The gateway's other proxies (chainProxy, backendProxy, oracleProxy) are
// deliberately dumb network-topology bridges — auth is the proxied
// service's job. KNIRVCHAIN's /api/v1/event-bundles/mint is intentionally
// internal-only (same gate as /api/v1/validation-proofs/mint), so a bare
// passthrough would force every CLI installation to carry
// KNIRV_INTERNAL_AUTH_TOKEN, defeating the point of that gate. This handler
// is the one exception: it requires the caller to already present their own
// KNIRV session Authorization header (proving they authenticated normally),
// then attaches the internal token on their behalf before forwarding.
//
// The caller's bearer token is validated against backend_server's protected
// /api/auth/me endpoint before the gateway adds the internal service token.
// This preserves the zero-trust boundary: possession of an arbitrary string
// in Authorization never grants access to the internal KNIRVCHAIN mint API.
// Bundle lookup remains public read access to already-minted, non-sensitive
// data.
type sessionAuthorizer interface {
	Authorize(context.Context, string) error
}

type backendSessionAuthorizer struct {
	client *http.Client
}

func newBackendSessionAuthorizer(socketPath string) sessionAuthorizer {
	if strings.TrimSpace(socketPath) == "" {
		return nil
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
	}}
	return &backendSessionAuthorizer{client: &http.Client{Transport: transport, Timeout: 5 * time.Second}}
}

func (a *backendSessionAuthorizer) Authorize(ctx context.Context, authorization string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://backend/api/auth/me", nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", authorization)
	response, err := a.client.Do(request)
	if err != nil {
		return fmt.Errorf("authorization service unavailable: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
			return errInvalidSession
		}
		return fmt.Errorf("authorization service returned %d", response.StatusCode)
	}
	return nil
}

var errInvalidSession = fmt.Errorf("invalid or expired session token")

func newEventBundleMintProxy(chainProxy *httputil.ReverseProxy, internalAuthToken string, authorizer sessionAuthorizer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			authorization := strings.TrimSpace(r.Header.Get("Authorization"))
			parts := strings.Fields(authorization)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
			if authorizer == nil {
				http.Error(w, "authorization service is not configured", http.StatusServiceUnavailable)
				return
			}
			if err := authorizer.Authorize(r.Context(), authorization); err != nil {
				if errors.Is(err, errInvalidSession) {
					http.Error(w, err.Error(), http.StatusUnauthorized)
				} else {
					http.Error(w, err.Error(), http.StatusServiceUnavailable)
				}
				return
			}
			if internalAuthToken == "" {
				http.Error(w, "event bundle minting is not configured", http.StatusServiceUnavailable)
				return
			}
			r.Header.Set("X-KNIRV-Internal-Token", internalAuthToken)
		}
		chainProxy.ServeHTTP(w, r)
	})
}
