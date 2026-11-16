package launcher

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	knirvgraph "KNIRVGRAPH"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// KNIRVGRAPH lifecycle owned by the launcher.
//
// The launcher extracts the embedded knirvgraph binary, starts it before
// backend_server, and stops it after backend_server on shutdown. The backend is
// told not to start its own copy (KNIRV_GRAPH_ENABLED=false) while the launcher
// owns it, so there is exactly one KNIRVGRAPH on the graph socket and ports.
// A missing or failing binary is logged and the launcher keeps running.

// knirvGraphPorts are the KNIRVGRAPH listener ports. They match the backend's
// graph.* defaults so the socket and ports stay the same whichever side starts it.
type knirvGraphPorts struct {
	RPC, P2P, API int
}

var knirvGraphDefaultPorts = knirvGraphPorts{RPC: 7090, P2P: 7091, API: 7092}

// extractKNIRVGraphBinary writes the embedded knirvgraph into destDir and returns
// its path. It is a var so tests can simulate a missing or unusable binary.
var extractKNIRVGraphBinary = knirvgraph.ExtractEmbeddedBinary

// knirvGraphConfig is the manager config the launcher uses for appDataDir. The
// ulorad variables come from uloraServiceEnv, the same contract backend_server
// receives, so KNIRVGRAPH sees the same token, socket and data directory.
// oracledRPCURL is the backend's API base URL; it is passed as
// KNIRV_ORACLED_RPC_URL, the variable backend_server sets for its own graph.
func knirvGraphConfig(appDataDir, binaryPath, authToken, oracledRPCURL string, ports knirvGraphPorts) *knirvgraph.ManagerConfig {
	overrides := map[string]string{}
	for _, kv := range uloraServiceEnv(appDataDir, authToken) {
		k, v, _ := strings.Cut(kv, "=")
		overrides[k] = v
	}
	if oracledRPCURL != "" {
		overrides["KNIRV_ORACLED_RPC_URL"] = oracledRPCURL
	}
	return &knirvgraph.ManagerConfig{
		BinaryPath:    binaryPath,
		SocketPath:    filepath.Join(appDataDir, "sockets", "graph.sock"),
		P2PSocketPath: filepath.Join(appDataDir, "sockets", "graph-p2p.sock"),
		Port:          ports.RPC,
		P2PPort:       ports.P2P,
		APIPort:       ports.API,
		DataPath:      filepath.Join(appDataDir, "knirvgraph"),
		StartTimeout:  30 * time.Second,
		StopTimeout:   10 * time.Second,
		AppDataDir:    appDataDir,
		EnvOverrides:  overrides,
	}
}

// startKNIRVGraphWith starts a KNIRVGRAPH manager for binaryPath. It is the part
// of startKNIRVGraph that tests drive with a fake binary.
func startKNIRVGraphWith(appDataDir, authToken, binaryPath, oracledRPCURL string, ports knirvGraphPorts, logger *zap.Logger) (*knirvgraph.Manager, error) {
	m := knirvgraph.NewManager(knirvGraphConfig(appDataDir, binaryPath, authToken, oracledRPCURL, ports), logger)
	if err := m.Start(); err != nil {
		return nil, err
	}
	return m, nil
}

// startKNIRVGraph extracts the embedded KNIRVGRAPH and starts it for this app.
func (app *ServerApp) startKNIRVGraph() error {
	appDataDir, err := getAppDataDir()
	if err != nil {
		return fmt.Errorf("resolve knirvgraph app data directory: %w", err)
	}
	binPath, err := extractKNIRVGraphBinary(filepath.Join(appDataDir, "bin"))
	if err != nil {
		return fmt.Errorf("extract knirvgraph: %w", err)
	}
	m, err := startKNIRVGraphWith(appDataDir, app.internalAuthToken, binPath, app.knirvGraphOracledRPCURL(), knirvGraphDefaultPorts, newKNIRVGraphLogger())
	if err != nil {
		return err
	}
	app.knirvGraphManager = m
	return nil
}

// knirvGraphOracledRPCURL is the backend API base URL the graph child is told
// about. backend_server passes http://127.0.0.1:<its API port> for the same
// purpose, and the launcher's BackendPort is that port (KNIRV_API_PORT). It is
// empty when no port is configured, and then the variable is not set.
func (app *ServerApp) knirvGraphOracledRPCURL() string {
	if app.config == nil || app.config.BackendPort <= 0 {
		return ""
	}
	return fmt.Sprintf("http://127.0.0.1:%d", app.config.BackendPort)
}

// startKNIRVGraphOptional starts KNIRVGRAPH and degrades on failure: the error is
// logged and the launcher continues. With no launcher-owned graph the backend
// falls back to its own, so it is never left without one.
func (app *ServerApp) startKNIRVGraphOptional() {
	if err := app.startKNIRVGraph(); err != nil {
		log.Printf("WARN: KNIRVGRAPH was not started by the launcher (continuing without it): %v", err)
	}
}

// stopKNIRVGraph stops the launcher-owned KNIRVGRAPH. Call it after backend_server
// has stopped. It is a no-op when the launcher never started KNIRVGRAPH.
func (app *ServerApp) stopKNIRVGraph() {
	if app.knirvGraphManager == nil {
		return
	}
	if err := app.knirvGraphManager.Stop(); err != nil {
		log.Printf("Warning: stopping KNIRVGRAPH: %v", err)
	}
	app.knirvGraphManager = nil
}

// graphOwnershipEnv is the backend_server environment entry that hands KNIRVGRAPH
// ownership to the launcher. It is empty unless the launcher started KNIRVGRAPH,
// so a failed launcher start leaves the backend to try its own.
func (app *ServerApp) graphOwnershipEnv() []string {
	if app.knirvGraphManager == nil {
		return nil
	}
	return []string{"KNIRV_GRAPH_ENABLED=false"}
}

// newKNIRVGraphLogger writes the manager's logs to stderr. The manager logs only
// the binary path, ports and data path, never the child environment.
func newKNIRVGraphLogger() *zap.Logger {
	return zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(os.Stderr),
		zap.InfoLevel,
	))
}
