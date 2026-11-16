package launcher

import (
	"net/http"
	"os"
	"strings"
)

// knirvServerAppForwardHeader is set by KNIRVGATEWAY on every request it forwards to
// this server's app socket. A forwarded request must never be proxied back to the
// gateway, which would loop on web-app assets (see setupRoutes NoRoute).
const knirvServerAppForwardHeader = "X-KNIRV-Gateway-Forwarded"

// knirvServerAppSocketPath is the internal Unix socket for this server's full HTTP
// router (the web app, /api/*, /arena/*, /tunnel, /ws). It sits beside the backend
// socket, as the other route sockets do. KNIRVGATEWAY forwards the public paths to it.
// An empty BackendSocket yields "", so no socket is created in the working directory.
func (app *ServerApp) knirvServerAppSocketPath() string {
	return app.socketInSocketsDir("knirvserver-app.sock")
}

// publicTCPHandler is the handler of the public TCP listener. Only /health and /version
// stay public on TCP. Every other path, including the web app, /api/*, /arena/*,
// /tunnel, /ws, and /git/*, answers 404 here and is served on the internal app socket
// through KNIRVGATEWAY.
func publicTCPHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (r.Method == http.MethodGet || r.Method == http.MethodHead) &&
			(r.URL.Path == "/health" || r.URL.Path == "/version") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		if r.Method != http.MethodHead {
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		}
	})
}

// spaAppBasePath is the URL prefix the launcher web app is built under (Next.js
// basePath in frontend/next.config.ts). KNIRVGATEWAY forwards /app/* to the app socket,
// and this prefix is stripped before the embedded export is looked up.
const spaAppBasePath = "/app"

// spaExportPath maps a request path to a path inside the embedded web-app export.
// /app and /app/... address the export root; any other path is the export path itself,
// so the public pages (/login, /menu, /terminal, /error-resolution) keep their URLs.
func spaExportPath(urlPath string) string {
	p := strings.TrimPrefix(urlPath, "/")
	if p == strings.TrimPrefix(spaAppBasePath, "/") {
		return ""
	}
	return strings.TrimPrefix(p, strings.TrimPrefix(spaAppBasePath, "/")+"/")
}

// authPublicURLFor is the origin browser authentication links are built on. The
// KNIRV_AUTH_PUBLIC_URL override wins. Otherwise it is the public gateway origin, because
// the device-login page is served on the public origin, not on the launcher's TCP port
// (which answers only /health and /version).
func authPublicURLFor(publicURL string) string {
	if configured := strings.TrimRight(strings.TrimSpace(os.Getenv("KNIRV_AUTH_PUBLIC_URL")), "/"); configured != "" {
		return configured
	}
	return strings.TrimRight(publicURL, "/")
}

// knirvServerBaseURLFor is the KNIRVSERVER origin that backend_server hands to KNIRVAGENT
// as KNIRV_SERVER_URL. The agent's /dve/* and /api/v1/dve/* calls must reach the public
// gateway, which routes them to the backend, so this is the public origin, not the
// launcher's TCP port (which now answers only /health and /version).
func knirvServerBaseURLFor(publicURL string) string {
	return strings.TrimRight(publicURL, "/")
}
