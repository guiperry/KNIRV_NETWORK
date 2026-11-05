package headless

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// The backend reaches every model over the daemon's socket, so the daemon's mux
// must carry the whole /models/* namespace after the migration: hasher (validator),
// heart (generator), head (vision), and the HEART-hosted decision routes.
func TestDaemonServesModelsNamespace(t *testing.T) {
	t.Setenv("KNIRV_APP_DATA_DIR", t.TempDir())
	s := NewServer(t.TempDir()+"/h.sock", 0o660)
	for _, p := range []string{
		sdk.RegistryPath, sdk.HealthPath,
		sdk.HeadPrefix + "/models", sdk.HeadPrefix + "/capabilities", sdk.HeadPrefix + "/status",
		sdk.HeartPrefix + "/health",
		sdk.HeartDecisionPrefix + "/models", sdk.HeartDecisionPrefix + "/matrix", sdk.HeartDecisionPrefix + "/reason",
		sdk.HasherPrefix + "/status",
	} {
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code == http.StatusNotFound && rec.Body.String() == "404 page not found\n" {
			t.Fatalf("%s is not routed", p)
		}
	}
	// HEAD is the vision model now: its manifest route is routed and returns the
	// VLM manifest (200) whether or not a backend is promoted.
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, sdk.HeadPrefix+"/models", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: got %d, want 200 (routed VLM manifest)", sdk.HeadPrefix+"/models", rec.Code)
	}
}

// End to end over a real unix socket with the SDK client the backend uses: the
// registry lists all four models (hasher validator, heart generator, head vision,
// hands server_tasks) and asking about HEART does not build it.
func TestRegistryOverUnixSocketWithSDKClient(t *testing.T) {
	t.Setenv("KNIRV_APP_DATA_DIR", t.TempDir())
	t.Setenv("KNIRV_AUDIT_DIR", t.TempDir())
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	sock := t.TempDir() + "/m.sock"
	s := NewServer(sock, 0o660)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: s.mux}
	go srv.Serve(ln)
	defer srv.Close()

	c := sdk.NewClient("http://models", sdk.NewUnixHTTPClient(sock, 5*time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := c.Registry(ctx)
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{}
	for _, r := range rows {
		roles[r.Name] = r.Role
	}
	want := map[string]string{
		"hasher": sdk.RoleValidator,
		"heart":  sdk.RoleGenerator,
		"head":   sdk.RoleVision,
		"hands":  sdk.RoleHands,
	}
	if len(roles) != len(want) {
		t.Fatalf("registry has %d models, want %d: %+v", len(roles), len(want), rows)
	}
	for name, role := range want {
		if roles[name] != role {
			t.Fatalf("registry role %s = %q, want %q (%+v)", name, roles[name], role, rows)
		}
	}
	h, err := c.HeartHealth(ctx)
	if err != nil || h.Loaded {
		t.Fatalf("heart health %+v %v", h, err)
	}
	rows, _ = c.Registry(ctx)
	for _, r := range rows {
		if r.Name == "heart" && r.Loaded {
			t.Fatal("asking about HEART built it")
		}
	}
	if hs, err := c.HasherStatus(ctx); err != nil || hs.Loaded {
		t.Fatalf("hasher status %+v %v", hs, err)
	}
	res, err := c.HasherValidate(ctx, sdk.HasherValidateRequest{Prediction: 1, Domain: "no-such-domain"})
	if err != nil || res.Domain != "no-such-domain" {
		t.Fatalf("validate %+v %v", res, err)
	}
	// HEAD is the vision model now: its /models/head/models manifest route is routed
	// and returns the VLM manifest (200) whether or not a backend is promoted.
	if _, err := c.Manifest(ctx); err != nil {
		t.Fatalf("manifest: %v", err)
	}
	// HANDS reaches /models/hands/status and reports as server_tasks in the registry.
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, sdk.HandsPrefix+"/status", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: got %d, want 200 (HANDS reachable)", sdk.HandsPrefix+"/status", rec.Code)
	}
}
