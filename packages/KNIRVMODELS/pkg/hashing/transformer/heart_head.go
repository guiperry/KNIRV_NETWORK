package transformer

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"knirvmodels/pkg/hashing/head"
)

// DefaultHEADDir is where 5_HEAD_TRAINER publishes (`--out .../data` -> `.../data/head`).
const DefaultHEADDir = "/var/lib/knirvserver/knirvmodels/data/head"

// headDir is where HEART reads the served model. It must be the directory stage 5
// publishes to, which is <app data>/knirvmodels/data/head with the app data root
// from KNIRV_APP_DATA_DIR when set (as embedded.GetAppDataDir does).
func headDir() string {
	if d := os.Getenv("KNIRV_HEAD_DIR"); d != "" {
		return d
	}
	if root := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); root != "" {
		return filepath.Join(root, "knirvmodels", "data", "head")
	}
	return DefaultHEADDir
}

func (hs *HEARTService) headService() *head.Service {
	hs.headOnce.Do(func() {
		hs.headSvc = head.NewService(headDir())
	})
	return hs.headSvc
}

// SetHEADService overrides the model source (tests).
func (hs *HEARTService) SetHEADService(s *head.Service) {
	hs.headOnce.Do(func() {})
	hs.headSvc = s
}

// registerHEAD mounts the HEAD routes. They carry no CORS headers and are meant to
// be reached only through backend_server, which authenticates callers.
func (hs *HEARTService) registerHEAD(mux *http.ServeMux) {
	head.Routes(hs.headService(), mux)
}
