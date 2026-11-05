package launcher

import (
	"context"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guiperry/knirv-sdk-go/secrets"
	"knirv-server/internal/knirvbao"
)

type SecretsConfig struct {
	OpenBao OpenBaoConfig `mapstructure:"openbao"`
}
type OpenBaoConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type baoManager interface {
	Start(context.Context) error
	Stop(context.Context) error
	Health() knirvbao.Status
	Socket() string
}

func (app *ServerApp) handleHealth(c *gin.Context) {
	checks := app.collectDeepHealthChecks()
	status := "healthy"
	for _, check := range checks {
		if !check.Skipped && !check.Healthy {
			status = "degraded"
			break
		}
	}
	roles := gin.H{"available": false}
	if resolved := app.roleResolution.Load(); resolved != nil {
		roles = gin.H{"available": true, "kind": resolved.Kind, "chain": resolved.ChainRole, "oracle": resolved.Oracle, "validator": resolved.Validator, "degraded": resolved.Degraded}
	}
	replication := map[string]any{"available": false}
	if dir, err := getAppDataDir(); err == nil {
		replication = replicationHealth(c.Request.Context(), filepath.Join(dir, "sockets", "oracle.sock"))
	}
	c.JSON(200, gin.H{
		"status": status, "version": Version, "build_time": BuildTime,
		"git_commit": GitCommit, "deep_checks": checks, "roles": roles, "replication": replication, "timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (app *ServerApp) startOpenBao(ctx context.Context) error {
	if !app.config.Secrets.OpenBao.Enabled {
		return nil
	}
	dir, err := getAppDataDir()
	if err != nil {
		return err
	}
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}
	app.bao = knirvbao.NewManager(knirvbao.Config{Enabled: true, DataDir: dir, ConfigDir: configDir}, secrets.NewFileProvider())
	return app.bao.Start(ctx)
}

func (app *ServerApp) stopOpenBao() error {
	if app.bao == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return app.bao.Stop(ctx)
}

func (app *ServerApp) addBaoHealth(checks map[string]healthProbeResult) {
	if !app.config.Secrets.OpenBao.Enabled {
		return
	}
	check := healthProbeResult{Name: "secrets"}
	if app.bao == nil {
		check.Error = "OpenBao has not started"
		checks["secrets"] = check
		return
	}
	status := app.bao.Health()
	check.SocketPath = app.bao.Socket()
	check.Healthy = status.State == knirvbao.Ready
	check.Skipped = status.State == knirvbao.Absent
	check.Error = status.Reason
	if !check.Healthy && !check.Skipped && check.Error == "" {
		check.Error = "OpenBao is degraded"
	}
	checks["secrets"] = check
}
