package server

import (
	"net/http"

	"github.com/gorilla/mux"
)

// registerEnrollRoute proxies the DEC-2 boot-node enrollment protocol to the
// enrollment server's Unix socket. Only POST /enroll/{step} is forwarded, with the
// path unchanged. The enrollment protocol authenticates each step (a signed challenge
// from the new bootnode key and a root approval), so the gateway does not add
// credentials of its own. With an empty socket path no route is registered, and
// /enroll/* is not served.
func registerEnrollRoute(r *mux.Router, socketPath string) {
	if socketPath == "" {
		return
	}
	r.PathPrefix("/enroll/").Methods(http.MethodPost).Handler(newSocketProxy(socketPath, "http://knirvenroll"))
}
