package launcher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"knirv-server/internal/knirvbao"
)

type fakeBaoManager struct {
	status  knirvbao.Status
	stopped bool
}

func (f *fakeBaoManager) Start(context.Context) error { return nil }
func (f *fakeBaoManager) Stop(context.Context) error  { f.stopped = true; return nil }
func (f *fakeBaoManager) Health() knirvbao.Status     { return f.status }
func (f *fakeBaoManager) Socket() string              { return "/test/bao.sock" }

func TestOpenBaoHealthHTTP(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KNIRV_APP_DATA_DIR", dir)
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name             string
		enabled          bool
		manager          *fakeBaoManager
		status           string
		present, skipped bool
	}{
		{"disabled", false, nil, "healthy", false, false},
		{"not started", true, nil, "degraded", true, false},
		{"ready", true, &fakeBaoManager{status: knirvbao.Status{State: knirvbao.Ready}}, "healthy", true, false},
		{"sealed", true, &fakeBaoManager{status: knirvbao.Status{State: knirvbao.Degraded, Reason: "sealed"}}, "degraded", true, false},
		{"client", true, &fakeBaoManager{status: knirvbao.Status{State: knirvbao.Absent}}, "healthy", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{BackendSocket: filepath.Join(dir, "backend.sock"), GatewaySocket: filepath.Join(dir, "gateway.sock"), Secrets: SecretsConfig{OpenBao: OpenBaoConfig{Enabled: tc.enabled}}}
			app := &ServerApp{config: cfg}
			if tc.manager != nil {
				app.bao = tc.manager
			}
			router := gin.New()
			router.GET("/health", app.handleHealth)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("HTTP status = %d", recorder.Code)
			}
			var response struct {
				Status string                       `json:"status"`
				Checks map[string]healthProbeResult `json:"deep_checks"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			check, present := response.Checks["secrets"]
			if response.Status != tc.status || present != tc.present || (present && check.Skipped != tc.skipped) {
				t.Fatalf("unexpected health: status=%s, present=%v, skipped=%v", response.Status, present, check.Skipped)
			}
			if tc.status == "degraded" && check.Error == "" {
				t.Fatal("degraded secret status has no reason")
			}
		})
	}
}

func TestOpenBaoConfigAndDisabledStart(t *testing.T) {
	v := viper.New()
	v.Set("secrets.openbao.enabled", true)
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatal(err)
	}
	if !cfg.Secrets.OpenBao.Enabled {
		t.Fatal("nested OpenBao setting was not decoded")
	}
	app := &ServerApp{config: &Config{}}
	if err := app.startOpenBao(context.Background()); err != nil || app.bao != nil {
		t.Fatal("disabled OpenBao changed startup")
	}
	manager := &fakeBaoManager{}
	app.bao = manager
	if err := app.stopOpenBao(); err != nil || !manager.stopped {
		t.Fatal("managed OpenBao was not stopped")
	}
}
