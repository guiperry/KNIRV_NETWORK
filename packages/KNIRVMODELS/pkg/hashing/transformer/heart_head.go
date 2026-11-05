package transformer

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	sdk "github.com/guiperry/knirv-sdk-go/models"
	"knirvmodels/pkg/heart/decision"
)

// DefaultDecisionDir is where HEART's decision model lives
// (data/heart/decision).
const DefaultDecisionDir = "/var/lib/knirvserver/knirvmodels/data/heart/decision"

// decisionDir resolves the directory the decision service READS from.
// KNIRV_HEART_DECISION_DIR (explicit override) ->
// <app data>/knirvmodels/data/heart/decision ->
// DefaultDecisionDir.
func decisionDir() string {
	if d := strings.TrimSpace(os.Getenv("KNIRV_HEART_DECISION_DIR")); d != "" {
		return d
	}
	if root := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); root != "" {
		return filepath.Join(root, "knirvmodels", "data", "heart", "decision")
	}
	return DefaultDecisionDir
}

func (hs *HEARTService) headService() *decision.Service {
	hs.headOnce.Do(func() {
		hs.headSvc = decision.NewService(decisionDir())
	})
	return hs.headSvc
}

// SetHEADService overrides the model source (tests).
func (hs *HEARTService) SetHEADService(s *decision.Service) {
	hs.headOnce.Do(func() {})
	hs.headSvc = s
}

// registerHEARTDecision mounts the decision routes. They carry no CORS headers and are meant
// to be reached only through backend_server, which authenticates callers.
//
// The decision core lives under HEART at /models/heart/decide (sdk.HeartDecisionPrefix).
// Phase 3 removed the legacy /models/head decision mount: HEAD is now the vision model
// and owns /models/head/models and /models/head/weights/{ver} as VLM artefacts, so the
// decision manifest/weights are reachable only via the HEART decision prefix and never
// collide with HEAD's VLM routes.
func (hs *HEARTService) registerHEARTDecision(mux *http.ServeMux) {
	svc := hs.headService()
	decision.RoutesAt(svc, mux, sdk.HeartDecisionPrefix)
}
