package server

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

// KnirvServerAppForwardHeader marks a request the gateway forwarded to KNIRVSERVER's
// app socket. KNIRVSERVER's router reads it so a forwarded request is never sent back
// to the gateway (its referer-based NoRoute proxy would otherwise loop on static assets).
const KnirvServerAppForwardHeader = "X-KNIRV-Gateway-Forwarded"

// knirvServerAppPages are the KNIRVSERVER web-app pages the gateway forwards to the
// app socket. The gateway's own WebGUI owns "/" and the explorer pages.
var knirvServerAppPages = []string{"login", "menu", "terminal", "error-resolution"}

// newKnirvServerAppProxy returns a handler that forwards a request unchanged (method,
// path, query, body, upgrade headers) to KNIRVSERVER's internal app socket. It returns
// nil when no socket is configured, so the routes below are not registered.
func newKnirvServerAppProxy(socketPath string) http.Handler {
	if socketPath == "" {
		return nil
	}
	proxy := newSocketProxy(socketPath, "http://knirvserver-app")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set(KnirvServerAppForwardHeader, "1")
		proxy.ServeHTTP(w, r)
	})
}

// registerKnirvServerAppRoutes forwards the KNIRVSERVER-owned public paths that are
// not WebSocket routes: the system info/update API, the DVE evidence API, the tunnel
// status routes and the KNIRVSERVER web-app pages. It must be registered before the
// generic /api/v1/ backend proxy, which would otherwise take the system routes.
func registerKnirvServerAppRoutes(r *mux.Router, app http.Handler) {
	r.PathPrefix("/api/v1/system/update").Handler(app)
	r.PathPrefix("/api/v1/system/info").Handler(app)
	r.PathPrefix("/api/dve/").Handler(app)
	r.Path("/tunnel").Handler(app)
	r.PathPrefix("/tunnel/").Handler(app)
	for _, page := range knirvServerAppPages {
		r.Path("/" + page).Handler(app)
		r.PathPrefix("/" + page + "/").Handler(app)
	}
	// The launcher web app is built under /app (its basePath). Everything under it,
	// including its /_next/ chunks, is answered by the app socket.
	r.Path(knirvServerAppBasePath).Handler(app)
	r.PathPrefix(knirvServerAppBasePath + "/").Handler(app)
	// The launcher's own rewrites for these prefixes (/network to the gateway root,
	// /turn to /api/turn, /gateway to the gateway root) are served only through the
	// app socket. /gateway/model-proxy/ stays on the gateway (see knirvServerAppGatewayPrefixMatcher).
	r.Path("/network").Handler(app)
	r.PathPrefix("/network/").Handler(app)
	r.Path("/turn").Handler(app)
	r.PathPrefix("/turn/").Handler(app)
	r.Path("/gateway").Handler(app)
	r.PathPrefix("/gateway/").MatcherFunc(knirvServerAppGatewayPrefixMatcher).Handler(app)
}

// knirvServerAppBasePath is the URL prefix of the launcher web app. It matches the
// Next.js basePath in KNIRVSERVER/frontend/next.config.ts.
const knirvServerAppBasePath = "/app"

// knirvServerAppGatewayPrefixMatcher matches /gateway/ paths that belong to the launcher
// rewrite, and excludes the gateway-owned model-call proxy. The model-call routes are
// registered later, so without this exclusion the broader prefix would shadow them.
func knirvServerAppGatewayPrefixMatcher(req *http.Request, _ *mux.RouteMatch) bool {
	p := req.URL.Path
	return !(p == "/gateway/model-proxy" || strings.HasPrefix(p, "/gateway/model-proxy/"))
}

// serverBaseURLLoopsToGateway reports whether KNIRV_SERVER_BASE_URL points at the
// gateway's own public origin. The launcher now sets that variable to the public origin
// for KNIRVAGENT, and the gateway inherits it. Proxying the legacy fallback there would
// loop through the public tunnel back into this gateway, so the fallback is skipped.
func serverBaseURLLoopsToGateway(serverBaseURL, publicURL string) bool {
	sb, err1 := url.Parse(strings.TrimSpace(serverBaseURL))
	pu, err2 := url.Parse(strings.TrimSpace(publicURL))
	if err1 != nil || err2 != nil || sb.Host == "" || pu.Host == "" {
		return false
	}
	return strings.EqualFold(sb.Scheme, pu.Scheme) && strings.EqualFold(sb.Host, pu.Host)
}

// registerKnirvServerAppWSRoutes forwards the unified /ws endpoints to KNIRVSERVER's
// app socket. Request upgrades pass through httputil.ReverseProxy, which hijacks the
// connection after the 101 response. It must be registered after the gateway's
// specific /ws/dve/... tunnels, which stay on the gateway.
func registerKnirvServerAppWSRoutes(r *mux.Router, app http.Handler) {
	r.Path("/ws").Handler(app)
	r.PathPrefix("/ws/").Handler(app)
}

// webguiFileOrAppFallback serves a file from the WebGUI static export. A file that is
// missing there is forwarded to KNIRVSERVER's app socket when one is configured (the
// web app's /_next/ chunks and public/ assets). Without a socket it answers 404 as
// the static file server does.
func webguiFileOrAppFallback(webguiDir string, app http.Handler) http.Handler {
	fileServer := http.StripPrefix("/", http.FileServer(http.Dir(webguiDir)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app != nil {
			if fi, err := os.Stat(filepath.Join(webguiDir, filepath.Clean("/"+r.URL.Path))); err != nil || fi.IsDir() {
				app.ServeHTTP(w, r)
				return
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}
