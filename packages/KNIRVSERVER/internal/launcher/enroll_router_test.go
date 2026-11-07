package launcher

import (
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

// enrollPaths are the three DEC-2 boot-node enrollment endpoints. Rule 2 forbids
// them on the public HTTP listener; they are served only over the internal Unix
// socket (see internal/enroll.Server.ServeUnix).
var enrollPaths = []string{"/enroll/transfer", "/enroll/bond", "/enroll/register"}

// newRouterForEnrollTest builds the launcher's REAL public router (app.router)
// through the production constructor NewServerApp — the exact code path main uses.
// A fresh, empty mux proves nothing about production, so we construct the genuine
// router here.
func newRouterForEnrollTest(t *testing.T) *gin.Engine {
	t.Helper()
	t.Setenv("KNIRV_APP_DATA_DIR", t.TempDir())
	d := t.TempDir()
	app, err := NewServerApp(&Config{
		Host:          "127.0.0.1",
		Port:          0,
		LogLevel:      "error",
		BackendSocket: filepath.Join(d, "backend.sock"),
		GatewaySocket: filepath.Join(d, "gateway.sock"),
	})
	if err != nil {
		t.Fatalf("NewServerApp (production router): %v", err)
	}
	return app.router
}

// TestEnrollNotOnRealPublicRouter proves Rule 2 against the launcher's real public
// router: none of the enrollment paths are registered on it. This is the guard that
// the hollow empty-mux test could not provide — a regression that registers
// /enroll/* on the public listener is caught here.
func TestEnrollNotOnRealPublicRouter(t *testing.T) {
	router := newRouterForEnrollTest(t)

	registered := map[string]string{}
	for _, r := range router.Routes() {
		registered[r.Path] = r.Method
	}
	for _, p := range enrollPaths {
		if m, ok := registered[p]; ok {
			t.Fatalf("public router MUST NOT register %s (Rule 2); found route %s %s", p, m, p)
		}
	}
}

// TestEnrollPublicRouterMutation documents the failure mode TestEnrollNotOnRealPublicRouter
// guards against: re-add an enrollment path to the real production router and the
// guard above must fail. We demonstrate that the mutated route is now present, i.e.
// the absence assertion in TestEnrollNotOnRealPublicRouter would report failure.
func TestEnrollPublicRouterMutation(t *testing.T) {
	router := newRouterForEnrollTest(t)

	// MUTATION: re-add an enrollment route to the real production router.
	router.POST("/enroll/transfer", func(c *gin.Context) { c.Status(200) })

	registered := map[string]string{}
	for _, r := range router.Routes() {
		registered[r.Path] = r.Method
	}
	if _, ok := registered["/enroll/transfer"]; !ok {
		t.Fatal("mutation scenario: expected /enroll/transfer registered on the real router (the guard test must now fail)")
	}
}
