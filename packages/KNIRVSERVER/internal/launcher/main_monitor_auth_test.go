package launcher

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/spf13/viper"
)

func TestMonitorAdminBackendUnixSocket(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("environment", "production")
	socket := filepath.Join(t.TempDir(), "backend.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	backend := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/me" || r.Header.Get("Authorization") != "Bearer session-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"role":"admin"}`))
	}))
	_ = backend.Listener.Close()
	backend.Listener = listener
	backend.Start()
	defer backend.Close()
	app := &ServerApp{config: &Config{BackendSocket: socket}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitor/metrics", nil)
	if app.isMonitorAdminRequest(req) {
		t.Fatal("missing credential authorized")
	}
	req.Header.Set("Authorization", "Bearer session-token")
	if !app.isMonitorAdminRequest(req) {
		t.Fatal("admin session rejected over Unix socket")
	}
}

func TestMonitorAdminBackendCredentials(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("environment", "production")
	for _, tc := range []struct {
		name, credential, body string
		status                 int
		want                   bool
	}{
		{"admin session", "opaque-admin-session", `{"role":"admin"}`, 200, true},
		{"member session", "opaque-member-session", `{"role":"user"}`, 200, false},
		{"invalid credential", "invalid", `{"role":"admin"}`, 401, false},
		{"backend failure", "opaque-admin-session", `{"role":"admin"}`, 503, false},
		{"malformed identity", "opaque-admin-session", `{`, 200, false},
		{"missing role", "opaque-admin-session", `{}`, 200, false},
		{"redirect", "opaque-admin-session", `{"role":"admin"}`, 302, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/auth/me" || r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer "+tc.credential {
					t.Errorf("unexpected identity request: %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer backend.Close()
			u, _ := url.Parse(backend.URL)
			port, _ := strconv.Atoi(u.Port())
			app := &ServerApp{config: &Config{BackendPort: port}}
			req := httptest.NewRequest(http.MethodGet, "/api/v1/monitor/metrics", nil)
			req.Header.Set("Authorization", "Bearer "+tc.credential)
			if got := app.isMonitorAdminRequest(req); got != tc.want {
				t.Fatalf("authorization = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsAdminRequestTestnetAdminToken(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	req := httptest.NewRequest("GET", "/api/v1/knirvbase/metrics", nil)
	req.Header.Set("Authorization", "Bearer TESTNET_ADMIN_TOKEN")

	viper.Set("environment", "testnet")
	if !isAdminRequest(req) {
		t.Fatal("testnet admin token should access monitor endpoints in testnet")
	}

	viper.Set("environment", "production")
	viper.Set("testnet", false)
	if isAdminRequest(req) {
		t.Fatal("testnet admin token must not access monitor endpoints outside testnet")
	}
}
