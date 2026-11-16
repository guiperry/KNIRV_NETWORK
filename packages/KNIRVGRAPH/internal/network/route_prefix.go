package network

import (
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Resolution Locator route group (DVE-U4). The canonical HTTP prefix is
// /graph/*. The legacy /nrv/* prefix is kept as an alias so existing callers
// (KNIRVSERVER, the CLI before this change, gateway clients) keep working. The
// alias is removed one release after this change; each alias logs a deprecation
// warning once, on first use.
const (
	locatorRoutePrefix       = "/graph"
	legacyLocatorRoutePrefix = "/nrv"
)

// handleLocatorRoute registers path (for example "/errors/commit") under the
// canonical /graph prefix and under the deprecated /nrv alias, with the same
// handler and methods for both.
func handleLocatorRoute(router *mux.Router, logger *zap.Logger, path string, h http.HandlerFunc, methods ...string) {
	router.HandleFunc(locatorRoutePrefix+path, h).Methods(methods...)
	router.HandleFunc(legacyLocatorRoutePrefix+path, deprecatedLocatorAlias(logger, path, h)).Methods(methods...)
}

// deprecatedLocatorAlias wraps a handler served under the legacy /nrv prefix.
func deprecatedLocatorAlias(logger *zap.Logger, path string, h http.HandlerFunc) http.HandlerFunc {
	var once sync.Once
	return func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() {
			if logger != nil {
				logger.Warn("deprecated route alias in use; switch to the /graph prefix",
					zap.String("legacy_route", legacyLocatorRoutePrefix+path),
					zap.String("route", locatorRoutePrefix+path),
				)
			}
		})
		h(w, r)
	}
}
