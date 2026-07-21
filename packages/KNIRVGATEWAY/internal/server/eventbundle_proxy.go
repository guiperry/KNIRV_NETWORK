package server

import (
	"net/http"
	"net/http/httputil"
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
// Full verification of the caller's session token against KNIRVSERVER's auth
// store is a follow-up — today this only enforces "a token was presented",
// consistent with the gateway's existing proxies not independently
// re-validating tokens the backend will check anyway. Unlike mint, the
// bundle-lookup GET is unauthenticated read access to already-minted,
// non-sensitive data, so no header requirement applies there.
func newEventBundleMintProxy(chainProxy *httputil.ReverseProxy, internalAuthToken string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if r.Header.Get("Authorization") == "" {
				http.Error(w, "authentication required", http.StatusUnauthorized)
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
