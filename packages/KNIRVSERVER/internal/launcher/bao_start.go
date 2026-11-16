package launcher

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guiperry/knirv-sdk-go/secrets"
	knirvoracle "github.com/KNIRV/KNIRV_NETWORK/KNIRVORACLE"
	"go.uber.org/zap"
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

// secretsBackendEnv selects the secrets backend: auto (default), file, or bao. It is
// read the same way in every deployment class (-prod, -dev, -ent, testnet default);
// the deployment class never changes the backend choice.
const secretsBackendEnv = "KNIRV_SECRETS_BACKEND"

// secretsBackendMode returns the requested backend from KNIRV_SECRETS_BACKEND, or an
// error for an unknown value. It never silently downgrades an unknown value.
func secretsBackendMode() (string, error) {
	switch m := strings.ToLower(strings.TrimSpace(os.Getenv(secretsBackendEnv))); m {
	case "":
		return "auto", nil
	case "auto", "file", "bao":
		return m, nil
	default:
		return "", fmt.Errorf("%s=%q is not one of auto, file, bao", secretsBackendEnv, m)
	}
}

// openBaoEnabled decides whether this launcher runs OpenBao. file never runs it;
// bao always runs it (and Start fails closed if it cannot); auto follows the
// secrets.openbao.enabled config, which is the unchanged default.
func (app *ServerApp) openBaoEnabled() bool {
	mode, err := secretsBackendMode()
	if err != nil {
		return false
	}
	switch mode {
	case "file":
		return false
	case "bao":
		return true
	default:
		return app.config.Secrets.OpenBao.Enabled
	}
}

func (app *ServerApp) startOpenBao(ctx context.Context) error {
	if !app.openBaoEnabled() {
		mode, err := secretsBackendMode()
		switch {
		case err != nil:
			// The value is deliberately not logged: it is operator input and could be a secret.
			log.Printf("[SECRETS] %s has an invalid value (expected auto, file or bao); OpenBao not started, file backend in use", secretsBackendEnv)
		case mode == "file":
			log.Printf("[SECRETS] %s=file: OpenBao not started; file backend in use", secretsBackendEnv)
		}
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
	// If a bao manager was injected (tests), reuse it; otherwise build the real one.
	if app.bao == nil {
		app.bao = knirvbao.NewManager(knirvbao.Config{Enabled: true, DataDir: dir, ConfigDir: configDir}, secrets.NewFileProvider())
	}

	// The Oracle failover (spec 8.6) AppRole swap is owned by the OpenBao backend.
	// Construct the production knirvoracle.Manager through NewOracleManager so the
	// real launcher.OraclePromoter is injected — a bare knirvoracle.NewManager would
	// leave a nil promoter and failover would silently no-op. The failover service
	// (backend_server) drives Manager.Promote/Demote on app.oracleManager when this
	// node is CONFIRMED/RECLAIMED.
	app.oracleManager = NewOracleManager(&knirvoracle.ManagerConfig{
		BaoSocketPath: app.bao.Socket(),
		SocketPath:    filepath.Join(dir, "sockets", "oracle.sock"),
		DataPath:      filepath.Join(dir, "oracle"),
	}, zap.L(), OraclePromoter{})

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
	if !app.openBaoEnabled() {
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
