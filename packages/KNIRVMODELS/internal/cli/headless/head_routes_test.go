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
// must carry the whole /models/* namespace.
func TestDaemonServesModelsNamespace(t *testing.T) {
	t.Setenv("KNIRV_APP_DATA_DIR", t.TempDir())
	s := NewServer(t.TempDir()+"/h.sock", 0o660)
	for _, p := range []string{
		sdk.RegistryPath, sdk.HealthPath,
		sdk.HeadPrefix + "/models", sdk.HeadPrefix + "/matrix",
		sdk.HeartPrefix + "/health", sdk.HasherPrefix + "/status",
	} {
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code == http.StatusNotFound && rec.Body.String() == "404 page not found\n" {
			t.Fatalf("%s is not routed", p)
		}
	}
	for _, p := range []string{sdk.HeadPrefix + "/models"} {
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code != http.StatusServiceUnavailable { // routed, but no model published yet
			t.Fatalf("%s: got %d: %s", p, rec.Code, rec.Body)
		}
	}
}

// End to end over a real unix socket with the SDK client the backend uses: the
// registry lists all three models and asking about HEART does not build it.
func TestRegistryOverUnixSocketWithSDKClient(t *testing.T) {
	t.Setenv("KNIRV_APP_DATA_DIR", t.TempDir())
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
	want := map[string]string{"hasher": sdk.RoleValidator, "heart": sdk.RoleGenerator, "head": sdk.RoleReasoner}
	if len(roles) != 3 || roles["hasher"] != want["hasher"] || roles["heart"] != want["heart"] || roles["head"] != want["head"] {
		t.Fatalf("registry %+v", rows)
	}
	h, err := c.HeartHealth(ctx)
	if err != nil || h.Loaded || h.Status != "idle" {
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
	if _, err := c.Manifest(ctx); err == nil {
		t.Fatal("HEAD with no published model must report ErrNoModel")
	}
}
