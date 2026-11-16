package launcher

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestSPAExportPath checks the /app basePath mapping: /app and /app/... address the export
// root, and the public pages (/login, /menu, /terminal, /error-resolution) keep their URLs.
func TestSPAExportPath(t *testing.T) {
	cases := map[string]string{
		"/app":                        "",
		"/app/":                       "",
		"/app/login":                  "login",
		"/app/_next/static/css/x.css": "_next/static/css/x.css",
		"/app/menu/index.txt":         "menu/index.txt",
		"/login":                      "login",
		"/menu":                       "menu",
		"/terminal/":                  "terminal/",
		"/error-resolution":           "error-resolution",
		"/application/x":              "application/x",
		"/":                           "",
	}
	for in, want := range cases {
		if got := spaExportPath(in); got != want {
			t.Errorf("spaExportPath(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestEmbeddedFSServesAppBasePath checks the embedded export answers the same document
// under /app as at its public URL, so the forwarded /app/* traffic resolves.
func TestEmbeddedFSServesAppBasePath(t *testing.T) {
	// The embed is assigned at startup (main), so tests read the built export from disk
	// through the same frontend/out/... paths the embed uses.
	prev := embeddedFiles
	embeddedFiles = os.DirFS("../..")
	t.Cleanup(func() { embeddedFiles = prev })
	efs, err := NewEmbeddedFS()
	if err != nil {
		t.Fatal(err)
	}
	get := func(target string) (int, string) {
		rec := httptest.NewRecorder()
		efs.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		return rec.Code, rec.Body.String()
	}
	codeA, bodyA := get("/app/login/")
	codeB, bodyB := get("/login/")
	if codeA != http.StatusOK || codeB != http.StatusOK || bodyA == "" || bodyA != bodyB {
		t.Fatalf("/app/login/ = %d, /login/ = %d, bodies equal = %v", codeA, codeB, bodyA == bodyB)
	}
	if !strings.Contains(bodyA, `/app/_next/`) {
		t.Fatalf("exported login page does not reference assets under /app/_next/")
	}
	if code, body := get("/app/"); code != http.StatusOK || !strings.Contains(body, "/app/_next/") {
		t.Fatalf("/app/ = %d, want the SPA home referencing /app/_next/", code)
	}
}

// TestAuthPublicURLDefaultsToPublicGateway checks the device-login origin: the public
// gateway origin by default, and the KNIRV_AUTH_PUBLIC_URL override when set.
func TestAuthPublicURLDefaultsToPublicGateway(t *testing.T) {
	t.Setenv("KNIRV_AUTH_PUBLIC_URL", "")
	if got := authPublicURLFor("https://testnet-gateway.example.net/"); got != "https://testnet-gateway.example.net" {
		t.Fatalf("default auth URL = %q, want the public gateway origin", got)
	}
	t.Setenv("KNIRV_AUTH_PUBLIC_URL", " https://auth.example.net/ ")
	if got := authPublicURLFor("https://testnet-gateway.example.net"); got != "https://auth.example.net" {
		t.Fatalf("override auth URL = %q, want https://auth.example.net", got)
	}
}

// TestKnirvServerBaseURLIsPublicOrigin checks that KNIRVAGENT's KNIRV_SERVER_URL is the
// public gateway origin, never the launcher's TCP port.
func TestKnirvServerBaseURLIsPublicOrigin(t *testing.T) {
	if got := knirvServerBaseURLFor("https://testnet-gateway.example.net/"); got != "https://testnet-gateway.example.net" {
		t.Fatalf("KNIRV_SERVER_BASE_URL = %q, want the public gateway origin", got)
	}
}

// TestBackendEnvUsesPublicOriginHelpers is a source-level guard that backend_server's
// environment takes the auth and KNIRVAGENT origins from the public-origin helpers. A
// revert to 127.0.0.1:<Port> would send device login and agent calls to the launcher TCP
// port, which answers only /health and /version.
func TestBackendEnvUsesPublicOriginHelpers(t *testing.T) {
	src, err := os.ReadFile("launcher.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	for _, want := range []string{
		"authPublicURL := authPublicURLFor(publicURL)",
		`fmt.Sprintf("KNIRV_SERVER_BASE_URL=%s", knirvServerBaseURLFor(publicURL))`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("launcher.go missing %q", want)
		}
	}
	if strings.Contains(text, `KNIRV_SERVER_BASE_URL=http://127.0.0.1:`) {
		t.Errorf("launcher.go still sets KNIRV_SERVER_BASE_URL to the launcher TCP port")
	}
}
