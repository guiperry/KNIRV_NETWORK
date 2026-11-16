package launcher

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

// TestSetupRoutesAnswers404ForGitOnTCP runs the real setupRoutes wiring, so it fails if
// the registerGitTCPNotFound call in setupRoutes is removed. Without that call, /git/*
// on the public TCP router would fall through to the SPA NoRoute and serve HTML.
//
// Everything is isolated: app data, proof stores and sockets live in a temp dir, HOME
// and XDG_DATA_HOME point there too, and the validator key comes from a fixed dummy
// value supplied through the env var, so no real key, .env or secret is read or written.
func TestSetupRoutesAnswers404ForGitOnTCP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "xdg"))
	t.Setenv("KNIRV_APP_DATA_DIR", filepath.Join(tmp, "appdata"))
	dummyKey := make([]byte, 32)
	for i := range dummyKey {
		dummyKey[i] = 1
	}
	t.Setenv("KNIRV_PROOF_VALIDATOR_X25519_PRIVATE_KEY", base64.StdEncoding.EncodeToString(dummyKey))

	prevFiles := embeddedFiles
	// Same layout as the real embedded frontend: the SPA fallback is frontend/out/index.html.
	embeddedFiles = fstest.MapFS{"frontend/out/index.html": {Data: []byte("<html>spa</html>")}}
	t.Cleanup(func() { embeddedFiles = prevFiles })

	app := &ServerApp{
		config: &Config{
			BackendSocket:    filepath.Join(tmp, "backend.sock"),
			GatewaySocket:    filepath.Join(tmp, "gateway.sock"),
			ProofStoreDir:    filepath.Join(tmp, "proof-protocol"),
			ProofLedgerDir:   filepath.Join(tmp, "proof-ledger"),
			ProofChainSocket: filepath.Join(tmp, "chain.sock"),
			NetworkMode:      "development",
		},
		router:            gin.New(),
		internalAuthToken: "test-internal-token",
	}
	if err := app.setupRoutes(); err != nil {
		t.Fatalf("setupRoutes: %v", err)
	}

	for _, p := range []string{"/git/proj.git/info/refs", "/git/proj/info/refs", "/git/"} {
		w := httptest.NewRecorder()
		app.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("GET %s = %d, want 404", p, w.Code)
		}
		if strings.Contains(w.Body.String(), "spa") {
			t.Fatalf("GET %s served SPA body", p)
		}
	}
}
