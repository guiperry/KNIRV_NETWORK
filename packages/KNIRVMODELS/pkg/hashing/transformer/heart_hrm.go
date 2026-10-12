package transformer

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"knirvmodels/pkg/hashing/hrm"
)

// DefaultHRMDir is where 5_HRM_TRAINER publishes (`--out .../data` -> `.../data/hrm`).
const DefaultHRMDir = "/var/lib/knirvserver/knirvmodels/data/hrm"

// hrmDir is where HEART reads the served model. It must be the directory stage 5
// publishes to, which is <app data>/knirvmodels/data/hrm with the app data root
// from KNIRV_APP_DATA_DIR when set (as embedded.GetAppDataDir does).
func hrmDir() string {
	if d := os.Getenv("KNIRV_HRM_DIR"); d != "" {
		return d
	}
	if root := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); root != "" {
		return filepath.Join(root, "knirvmodels", "data", "hrm")
	}
	return DefaultHRMDir
}

func (hs *HEARTService) hrmService() *hrm.Service {
	hs.hrmOnce.Do(func() {
		hs.hrmSvc = hrm.NewService(hrmDir())
	})
	return hs.hrmSvc
}

// SetHRMService overrides the model source (tests).
func (hs *HEARTService) SetHRMService(s *hrm.Service) {
	hs.hrmOnce.Do(func() {})
	hs.hrmSvc = s
}

// registerHRM mounts the HRM routes. They carry no CORS headers and are meant to
// be reached only through backend_server, which authenticates callers.
func (hs *HEARTService) registerHRM(mux *http.ServeMux) {
	hrm.Routes(hs.hrmService(), mux)
}
