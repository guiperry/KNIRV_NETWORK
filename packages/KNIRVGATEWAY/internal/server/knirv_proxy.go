package server

import (
	"net/http"

	"github.com/gorilla/mux"
)

// registerKnirvSocketRoutes exposes KNIRVSERVER's native proof API through the gateway.
// KNIRVSERVER serves it on an internal Unix socket and never on its TCP listener:
//
//   - /api/v1/knirv/* goes to proofSocket (KNIRVPROOF_SOCKET_PATH), path unchanged.
//
// The socket owner authenticates each request itself, so the gateway adds no
// credentials. An empty socket path registers a 404 for that prefix instead of a
// proxy. The prefix must not fall through to the generic /api/v1/ backend proxy.
func registerKnirvSocketRoutes(r *mux.Router, proofSocket string) {
	if proofSocket != "" {
		r.PathPrefix("/api/v1/knirv/").Handler(newSocketProxy(proofSocket, "http://knirvproof"))
	} else {
		r.PathPrefix("/api/v1/knirv/").HandlerFunc(http.NotFound)
	}
}

// registerGitPublicNotFound answers /git/* on the gateway's public listener with 404.
// The proof ledger's git routes (KNIRVSERVER's git.sock) are internal only: the gateway
// does not proxy them publicly, and the SPA catch-all must never serve HTML for them.
// Git clients therefore have no public push path until an owner-approved bridge exists.
func registerGitPublicNotFound(r *mux.Router) {
	r.PathPrefix("/git/").HandlerFunc(http.NotFound)
}
