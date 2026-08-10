package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	knirvagent "github.com/KNIRV/KNIRV_NETWORK/KNIRVAGENT"
	knirvmonitor "github.com/KNIRV/KNIRV_NETWORK/KNIRVSERVER/pkg/knirvmonitor"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
	"go.uber.org/zap"
	"knirv-server/internal/bootkey"
	dveevidence "knirv-server/internal/dveevidence"
	"knirv-server/internal/knirvproof"
	"knirv-server/internal/proofledger"
	"knirv-server/internal/tlsprovider"
	"knirv-server/internal/updater"
	"knirv-server/pkg/embedded"
	"knirv-server/pkg/embedded/validationchain"
	"knirv-server/pkg/embedded/validationchain/checkpoint"
)

// Embed the Next.js build output
//
//go:embed all:frontend/out/*
var embeddedFiles embed.FS

// Embed the unified backend binary
//
//go:embed bin/backend_server
var backendBinary []byte

// Embed the config files
//
//go:embed all:config/*
var configFiles embed.FS

// Version information (set by build flags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

const (
	defaultSystemAppDataDir = "/var/lib/knirvserver"
	defaultSystemConfigDir  = "/etc/knirv-server"
)

// Config represents the application configuration
type Config struct {
	Host                  string   `mapstructure:"host"`
	Port                  int      `mapstructure:"port"`
	BackendPort           int      `mapstructure:"backend_port"`
	BackendSocket         string   `mapstructure:"backend_socket"`
	GatewayPort           int      `mapstructure:"gateway_port"`
	GatewaySocket         string   `mapstructure:"gateway_socket"`
	MonitorPort           int      `mapstructure:"monitor_port"`
	LogLevel              string   `mapstructure:"log_level"`
	Testnet               bool     `mapstructure:"testnet"`
	ProofStoreDir         string   `mapstructure:"proof_store_dir"`
	ProofLedgerDir        string   `mapstructure:"proof_ledger_dir"`
	ProofMaxObjectBytes   int64    `mapstructure:"proof_max_object_bytes"`
	ProofRequiredReplicas int      `mapstructure:"proof_required_replicas"`
	ProofReplicaDirs      []string `mapstructure:"proof_replica_dirs"`
	ProofValidatorID      string   `mapstructure:"proof_validator_id"`
	ProofChainSocket      string   `mapstructure:"proof_chain_socket"`
	// AutoStartHasher is command-line only. When -hasher is set, the wrapper
	// forwards the flag to the embedded backend so it starts the training
	// pipeline during initialization.
	AutoStartHasher bool
	// NetworkMode is "testnet" (default), "production", "development", or
	// "enterprise" — set from the -prod / -dev / -ent flags (or environment),
	// not sourced from the config YAML itself.
	NetworkMode string
	// Enterprise identifies the Enterprise subscription deployment class. It is
	// intentionally distinct from the KNIRV main-network production class.
	Enterprise bool
	// UserIDTag is the DNS-safe user identity suffix used by devnet and enterprise.
	UserIDTag string
}

func normalizeUserIDTag(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var tag strings.Builder
	lastWasHyphen := false
	for _, r := range value {
		valid := r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
		if valid {
			tag.WriteRune(r)
			lastWasHyphen = false
			continue
		}
		if tag.Len() > 0 && !lastWasHyphen {
			tag.WriteByte('-')
			lastWasHyphen = true
		}
	}
	normalized := strings.Trim(tag.String(), "-")
	// enterprise- is the longest hostname prefix (11 bytes), leaving 52 bytes
	// for the tag within the DNS label's 63-byte limit.
	if len(normalized) > 52 {
		normalized = strings.TrimRight(normalized[:52], "-")
	}
	return normalized
}

// resolvePublicURL returns the single public origin shared by backend_server,
// KNIRVGATEWAY, and cloudflared. An explicit deployment value always wins.
func resolvePublicURL(cfg *Config) (string, error) {
	if configured := strings.TrimRight(strings.TrimSpace(os.Getenv("KNIRV_PUBLIC_URL")), "/"); configured != "" {
		return configured, nil
	}

	tag := normalizeUserIDTag(cfg.UserIDTag)
	if cfg.Enterprise {
		if tag == "" {
			return "", fmt.Errorf("enterprise mode requires -user-id-tag or KNIRV_USER_ID_TAG")
		}
		return fmt.Sprintf("https://enterprise-%s.knirv.network", tag), nil
	}

	switch strings.ToLower(strings.TrimSpace(cfg.NetworkMode)) {
	case "production", "prod", "mainnet":
		return "https://gateway.knirv.network", nil
	case "development", "dev", "devnet":
		if tag == "" {
			return "", fmt.Errorf("development mode requires -user-id-tag or KNIRV_USER_ID_TAG")
		}
		return fmt.Sprintf("https://devnet-%s.knirv.network", tag), nil
	default:
		return "https://testnet-gateway.knirv.network", nil
	}
}

// EmbeddedFS wraps the embedded filesystem for serving static files
type EmbeddedFS struct {
	files fs.FS
}

// NewEmbeddedFS creates a new embedded filesystem
func NewEmbeddedFS() (*EmbeddedFS, error) {
	return &EmbeddedFS{
		files: embeddedFiles,
	}, nil
}

// ServeHTTP implements http.Handler for serving embedded files
func (efs *EmbeddedFS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")

	// Build candidate paths to try in priority order.
	// Next.js static export with trailingSlash:true produces directory/index.html files,
	// so directory-style paths (ending in / or lacking an extension) must check index.html first.
	var candidates []string
	base := filepath.Base(path)
	hasExt := strings.Contains(base, ".") && !strings.HasSuffix(path, "/")

	if path == "" || strings.HasSuffix(path, "/") {
		// Root or explicit directory request
		candidates = append(candidates,
			"frontend/out/"+path+"index.html",
		)
	} else if hasExt {
		// Direct file with extension
		candidates = append(candidates,
			"frontend/out/"+path,
		)
	} else {
		// No extension — could be a Next.js route (e.g. "login", "menu")
		candidates = append(candidates,
			"frontend/out/"+path+"/index.html",
			"frontend/out/"+path+".html",
		)
	}
	// Always fall back to SPA root for client-side routing
	candidates = append(candidates, "frontend/out/index.html")

	var file fs.File
	var resolvedPath string
	for _, candidate := range candidates {
		f, err := efs.files.Open(candidate)
		if err != nil {
			continue
		}
		stat, err := f.Stat()
		if err != nil || stat.IsDir() {
			f.Close()
			continue
		}
		file = f
		resolvedPath = candidate
		break
	}

	if file == nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set content type based on resolved file extension
	ext := filepath.Ext(resolvedPath)

	// Service worker must always be revalidated so Chrome picks up updates
	if strings.HasSuffix(resolvedPath, "service-worker.js") {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Service-Worker-Allowed", "/")
	}

	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".json":
		if strings.HasSuffix(resolvedPath, "manifest.json") {
			w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
		} else {
			w.Header().Set("Content-Type", "application/json")
		}
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	case ".woff2":
		w.Header().Set("Content-Type", "font/woff2")
	case ".woff":
		w.Header().Set("Content-Type", "font/woff")
	}

	http.ServeContent(w, r, stat.Name(), stat.ModTime(), file.(io.ReadSeeker))
}

// ServerApp represents the main application
type ServerApp struct {
	config                   *Config
	router                   *gin.Engine
	server                   *http.Server
	agentControl             *AgentControlServer
	dveIngest                *dveevidence.IngestService
	dveRoutes                *gin.Engine
	proofService             *knirvproof.Service
	proofRoutes              *gin.Engine
	proofLedger              *proofledger.Ledger
	proofLedgerRoutes        *gin.Engine
	internalAuthToken        string
	proofValidatorPrivateKey string
	backendCmd               *exec.Cmd
	ipfsCmd                  *exec.Cmd
	xionCmd                  *exec.Cmd
	textEmbedderCmd          *exec.Cmd
	backendPath              string
	tempDir                  string
	upd                      *updater.Updater
	monitorManager           *knirvmonitor.Manager
}

func (app *ServerApp) startIPFS(ctx context.Context) error {
	binary := strings.TrimSpace(os.Getenv("IPFS_BINARY_PATH"))
	if binary == "" {
		var err error
		binary, err = exec.LookPath("ipfs")
		if err != nil {
			return fmt.Errorf("IPFS binary not found (set IPFS_BINARY_PATH): %w", err)
		}
	}
	appDataDir, err := getAppDataDir()
	if err != nil {
		return err
	}
	ipfsPath := strings.TrimSpace(os.Getenv("IPFS_PATH"))
	if ipfsPath == "" {
		ipfsPath = filepath.Join(appDataDir, "ipfs")
	}
	env := setChildEnv(os.Environ(), "IPFS_PATH", ipfsPath)
	run := func(args ...string) error {
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Env, cmd.Stdout, cmd.Stderr = env, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ipfs %s: %w", strings.Join(args, " "), err)
		}
		return nil
	}
	if _, err := os.Stat(filepath.Join(ipfsPath, "config")); os.IsNotExist(err) {
		if err := run("init", "--profile=server"); err != nil {
			return err
		}
	}
	for _, setting := range [][2]string{{"Addresses.API", "/ip4/127.0.0.1/tcp/5001"}, {"Addresses.Gateway", "/ip4/127.0.0.1/tcp/8081"}} {
		if err := run("config", setting[0], setting[1]); err != nil {
			return err
		}
	}
	cmd := exec.CommandContext(ctx, binary, "daemon", "--migrate=true")
	cmd.Env, cmd.Stdout, cmd.Stderr = env, os.Stdout, os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start IPFS daemon: %w", err)
	}
	app.ipfsCmd = cmd
	go func() {
		if err := cmd.Wait(); err != nil && ctx.Err() == nil {
			log.Printf("IPFS daemon exited: %v", err)
		}
	}()
	log.Printf("IPFS daemon started (API 127.0.0.1:5001, gateway 127.0.0.1:8081)")
	return nil
}

func (app *ServerApp) startXion(ctx context.Context) error {
	binary := strings.TrimSpace(os.Getenv("XION_BINARY_PATH"))
	if binary == "" {
		var err error
		binary, err = exec.LookPath("xiond")
		if err != nil {
			return fmt.Errorf("xiond binary not found (set XION_BINARY_PATH or install github.com/burnt-labs/xion): %w", err)
		}
	}
	home := strings.TrimSpace(os.Getenv("XION_HOME"))
	if home == "" {
		appDataDir, err := getAppDataDir()
		if err != nil {
			return fmt.Errorf("resolve Xion data directory: %w", err)
		}
		home = filepath.Join(appDataDir, "xion")
	}
	chainID := strings.TrimSpace(os.Getenv("XION_CHAIN_ID"))
	if chainID == "" {
		switch strings.ToLower(strings.TrimSpace(app.config.NetworkMode)) {
		case "production", "prod", "mainnet":
			chainID = "xion-mainnet-1"
		case "development", "dev", "devnet", "local":
			chainID = "xion-local-testnet-1"
		default:
			chainID = "xion-testnet-2"
		}
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		return fmt.Errorf("create Xion home: %w", err)
	}
	configDir := filepath.Join(home, "config")
	genesisPath := filepath.Join(configDir, "genesis.json")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		initCmd := exec.CommandContext(ctx, binary, "init", "knirvserver", "--home", home, "--chain-id", chainID)
		initCmd.Stdout, initCmd.Stderr = os.Stdout, os.Stderr
		initCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := initCmd.Run(); err != nil {
			return fmt.Errorf("initialize Xion home: %w", err)
		}
	}
	if _, err := os.Stat(genesisPath); os.IsNotExist(err) {
		if chainID == "xion-local-testnet-1" {
			return fmt.Errorf("local Xion genesis is missing at %s; initialize it with xiond add-genesis-account/gentx/collect-gentxs", genesisPath)
		}
		genesisURL := strings.TrimSpace(os.Getenv("XION_GENESIS_URL"))
		if genesisURL == "" {
			if chainID == "xion-mainnet-1" {
				genesisURL = "https://raw.githubusercontent.com/burnt-labs/burnt-networks/main/mainnet/xion-mainnet-1/genesis.json"
			} else if chainID == "xion-testnet-2" {
				genesisURL = "https://raw.githubusercontent.com/burnt-labs/xion-testnet-2/config/genesis.json"
			}
		}
		if genesisURL == "" {
			return fmt.Errorf("no genesis source configured for Xion chain %s", chainID)
		}
		if err := downloadXionGenesis(ctx, genesisURL, genesisPath); err != nil {
			return err
		}
	}
	if err := validateXionGenesis(genesisPath, chainID); err != nil {
		return err
	}
	if err := ensureXionKeyPermissions(home); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, binary, "start", "--home", home,
		"--chain-id", chainID, "--rpc.laddr", "tcp://127.0.0.1:26657",
		"--grpc.address", "127.0.0.1:9091", "--api.address", "tcp://127.0.0.1:1317")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start Xion: %w", err)
	}
	app.xionCmd = cmd
	go func() {
		if err := cmd.Wait(); err != nil && ctx.Err() == nil {
			log.Printf("Xion exited: %v", err)
		}
	}()
	log.Printf("Xion started (chain %s, home %s, RPC 127.0.0.1:26657, REST 127.0.0.1:1317, gRPC 127.0.0.1:9091)", chainID, home)
	return nil
}

func downloadXionGenesis(ctx context.Context, source, destination string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return fmt.Errorf("create Xion genesis request: %w", err)
	}
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("download Xion genesis: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download Xion genesis: HTTP %d from %s", resp.StatusCode, source)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 512<<20))
	if err != nil {
		return fmt.Errorf("read Xion genesis: %w", err)
	}
	var document struct {
		ChainID string `json:"chain_id"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("decode Xion genesis: %w", err)
	}
	if strings.TrimSpace(document.ChainID) == "" {
		return fmt.Errorf("Xion genesis %s has no chain_id", source)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return fmt.Errorf("create Xion config directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(destination), ".genesis-*")
	if err != nil {
		return fmt.Errorf("create temporary Xion genesis: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0644); err != nil {
		tmp.Close()
		return fmt.Errorf("set Xion genesis permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write Xion genesis: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close Xion genesis: %w", err)
	}
	if err := os.Rename(tmpName, destination); err != nil {
		return fmt.Errorf("install Xion genesis: %w", err)
	}
	return nil
}

func validateXionGenesis(path, expectedChainID string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read Xion genesis: %w", err)
	}
	var document struct {
		ChainID string `json:"chain_id"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("decode Xion genesis: %w", err)
	}
	if document.ChainID != expectedChainID {
		return fmt.Errorf("Xion genesis chain_id %q does not match configured chain %q", document.ChainID, expectedChainID)
	}
	return nil
}

func ensureXionKeyPermissions(home string) error {
	configDir := filepath.Join(home, "config")
	for _, name := range []string{"priv_validator_key.json", "node_key.json", "priv_validator_state.json"} {
		path := filepath.Join(configDir, name)
		if _, err := os.Stat(path); err == nil {
			if err := os.Chmod(path, 0600); err != nil {
				return fmt.Errorf("secure Xion key %s: %w", path, err)
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect Xion key %s: %w", path, err)
		}
	}
	for _, keyring := range []string{"keyring-file", "keyring-test"} {
		root := filepath.Join(home, keyring)
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if entry.IsDir() {
				return os.Chmod(path, 0700)
			}
			return os.Chmod(path, 0600)
		}); err != nil {
			return fmt.Errorf("secure Xion %s: %w", keyring, err)
		}
	}
	return nil
}

func (app *ServerApp) startTextEmbedder(ctx context.Context) error {
	binary := strings.TrimSpace(os.Getenv("TEXT_EMBEDDER_BINARY_PATH"))
	if binary == "" {
		for _, name := range []string{"text-embedder", "embedder"} {
			if resolved, err := exec.LookPath(name); err == nil {
				binary = resolved
				break
			}
		}
	}
	if binary == "" {
		return fmt.Errorf("text-embedder binary not found (set TEXT_EMBEDDER_BINARY_PATH)")
	}
	cmd := exec.CommandContext(ctx, binary)
	cmd.Env = setChildEnv(os.Environ(), "PORT", "8089")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start text-embedder: %w", err)
	}
	app.textEmbedderCmd = cmd
	go func() {
		if err := cmd.Wait(); err != nil && ctx.Err() == nil {
			log.Printf("text-embedder exited: %v", err)
		}
	}()
	log.Printf("Deterministic text-embedder started on 127.0.0.1:8089")
	return nil
}

func stopManagedProcess(name string, cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}
	log.Printf("Stopped %s subprocess", name)
}

const defaultAgentControlSocketName = "knirvagent-control.sock"

type agentSupervisor interface {
	SetWorkspaceResolver(func(dveID string) (string, error))
	SetExtraEnv([]string)
	StartAgent(ctx context.Context, dveID string, startTimeout time.Duration) error
	StopAgent(dveID string, stopTimeout time.Duration) error
	GetSocketPathForDVE(dveID string) (string, error)
	RunningCount() int
	IsRunning() bool
	HealthCheck(ctx context.Context) error
	GetBaseURL() string
	Stop(ctx context.Context) error
}

type AgentControlServer struct {
	socketPath string
	appDataDir string
	authToken  string
	manager    agentSupervisor
	server     *http.Server
	listener   net.Listener

	mu            sync.RWMutex
	workspacePath map[string]string
	baseExtraEnv  []string
	started       bool
}

type agentControlStartRequest struct {
	WorkspacePath  string   `json:"workspace_path,omitempty"`
	ExtraEnv       []string `json:"extra_env,omitempty"`
	StartTimeoutMS int64    `json:"start_timeout_ms,omitempty"`
}

type agentControlAgentResponse struct {
	OK            bool   `json:"ok"`
	DVEID         string `json:"dve_id,omitempty"`
	SocketPath    string `json:"socket_path,omitempty"`
	WorkspacePath string `json:"workspace_path,omitempty"`
	RunningCount  int    `json:"running_count,omitempty"`
	Error         string `json:"error,omitempty"`
}

type agentControlStatusResponse struct {
	OK           bool   `json:"ok"`
	Running      bool   `json:"running"`
	Healthy      bool   `json:"healthy"`
	RunningCount int    `json:"running_count"`
	BaseURL      string `json:"base_url"`
	SocketPath   string `json:"socket_path"`
	Timestamp    string `json:"timestamp"`
	Error        string `json:"error,omitempty"`
}

func defaultAgentControlSocketPath(appDataDir string) string {
	if strings.TrimSpace(appDataDir) == "" {
		appDataDir = "/var/lib/knirvserver"
	}
	return filepath.Join(appDataDir, "sockets", defaultAgentControlSocketName)
}

func mergeEnvSlices(base, extra []string) []string {
	seen := make(map[string]struct{}, len(base)+len(extra))
	result := make([]string, 0, len(base)+len(extra))

	appendUnique := func(list []string) {
		for _, item := range list {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}

	appendUnique(base)
	appendUnique(extra)
	return result
}

func newAgentControlServer(manager agentSupervisor, socketPath, appDataDir, authToken string) *AgentControlServer {
	srv := &AgentControlServer{
		socketPath:    socketPath,
		appDataDir:    appDataDir,
		authToken:     authToken,
		manager:       manager,
		workspacePath: make(map[string]string),
		baseExtraEnv:  collectAgentExtraEnv(),
	}

	if srv.manager != nil {
		srv.manager.SetWorkspaceResolver(srv.resolveWorkspacePath)
		srv.manager.SetExtraEnv(srv.baseExtraEnv)
	}

	return srv
}

func (app *ServerApp) startAgentControl() error {
	if app.agentControl != nil {
		return nil
	}

	appDataDir, err := getAppDataDir()
	if err != nil {
		return fmt.Errorf("failed to resolve app data directory: %w", err)
	}

	controlSocket := defaultAgentControlSocketPath(appDataDir)
	binDir := filepath.Join(appDataDir, "bin")
	agentBinaryPath, err := knirvagent.ExtractEmbeddedBinary(binDir)
	if err != nil {
		return fmt.Errorf("failed to extract KNIRVAGENT binary: %w", err)
	}

	managerCfg := knirvagent.DefaultManagerConfig()
	managerCfg.SocketPath = controlSocket
	managerCfg.BinaryPath = agentBinaryPath
	managerCfg.ExtraEnv = collectAgentExtraEnv()

	manager := knirvagent.NewManager(managerCfg, zap.NewNop())
	control := newAgentControlServer(manager, controlSocket, appDataDir, app.internalAuthToken)
	if err := control.Start(); err != nil {
		return err
	}

	app.agentControl = control
	return nil
}

func (app *ServerApp) stopAgentControl() {
	if app.agentControl == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.agentControl.Stop(ctx); err != nil {
		log.Printf("Error stopping KNIRVAGENT control server: %v", err)
	}
	app.agentControl = nil
}

func collectAgentExtraEnv() []string {
	envNames := []string{"GEMINI_API_KEY", "DEEPSEEK_API_KEY", "CEREBRAS_API_KEY"}
	extras := make([]string, 0, len(envNames))
	for _, name := range envNames {
		if val := strings.TrimSpace(os.Getenv(name)); val != "" {
			extras = append(extras, fmt.Sprintf("%s=%s", name, val))
		}
	}
	return extras
}

func (s *AgentControlServer) resolveWorkspacePath(dveID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workspacePath, ok := s.workspacePath[dveID]
	if !ok || strings.TrimSpace(workspacePath) == "" {
		return "", fmt.Errorf("workspace path not configured for DVE %s", dveID)
	}
	return workspacePath, nil
}

func (s *AgentControlServer) updateWorkspacePath(dveID, workspacePath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(workspacePath) == "" {
		return
	}
	s.workspacePath[dveID] = workspacePath
}

func (s *AgentControlServer) setMergedEnv(extra []string) {
	if s.manager == nil {
		return
	}
	s.manager.SetExtraEnv(mergeEnvSlices(s.baseExtraEnv, extra))
}

func (s *AgentControlServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/control/status", s.handleStatus)
	mux.HandleFunc("/control/agents/", s.handleAgents)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" || parts[1] != s.authToken {
			w.Header().Set("WWW-Authenticate", `Bearer realm="knirv-agent-control"`)
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *AgentControlServer) Start() error {
	if s == nil || s.manager == nil {
		return fmt.Errorf("agent control server not configured")
	}
	if s.started {
		return nil
	}

	if s.socketPath == "" {
		s.socketPath = defaultAgentControlSocketPath(s.appDataDir)
	}
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0755); err != nil {
		return fmt.Errorf("failed to create control socket directory: %w", err)
	}
	if err := os.RemoveAll(s.socketPath); err != nil {
		return fmt.Errorf("failed to remove stale control socket: %w", err)
	}

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to bind control socket %s: %w", s.socketPath, err)
	}
	if err := os.Chmod(s.socketPath, 0600); err != nil {
		_ = listener.Close()
		return fmt.Errorf("failed to chmod control socket %s: %w", s.socketPath, err)
	}

	s.listener = listener
	s.server = &http.Server{Handler: s.routes()}
	s.started = true

	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("KNIRVAGENT control server stopped with error: %v", err)
		}
	}()

	log.Printf("KNIRVAGENT control server started on %s", s.socketPath)
	return nil
}

func (s *AgentControlServer) Stop(ctx context.Context) error {
	if s == nil {
		return nil
	}

	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			log.Printf("KNIRVAGENT control server shutdown error: %v", err)
		}
	}
	if s.manager != nil {
		if err := s.manager.Stop(ctx); err != nil {
			log.Printf("KNIRVAGENT agent stop error during control shutdown: %v", err)
		}
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	if s.socketPath != "" {
		_ = os.Remove(s.socketPath)
	}
	s.started = false
	return nil
}

func (s *AgentControlServer) handleAgents(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/control/agents/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "dve id required", http.StatusBadRequest)
		return
	}

	dveID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch r.Method {
	case http.MethodPost:
		if action != "start" {
			http.Error(w, "unsupported control route", http.StatusNotFound)
			return
		}
		s.handleStart(w, r, dveID)
	case http.MethodDelete:
		if action != "" {
			http.Error(w, "unsupported control route", http.StatusNotFound)
			return
		}
		s.handleStop(w, r, dveID)
	case http.MethodGet:
		switch action {
		case "":
			s.handleAgentStatus(w, r, dveID)
		case "socket":
			s.handleSocket(w, r, dveID)
		case "health":
			s.handleAgentHealth(w, r, dveID)
		default:
			http.Error(w, "unsupported control route", http.StatusNotFound)
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *AgentControlServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	healthy := s.manager != nil && s.manager.HealthCheck(r.Context()) == nil
	running := false
	runningCount := 0
	baseURL := "http://localhost"
	if s.manager != nil {
		running = s.manager.IsRunning()
		runningCount = s.manager.RunningCount()
		baseURL = s.manager.GetBaseURL()
	}
	writeJSON(w, http.StatusOK, agentControlStatusResponse{
		OK:           true,
		Running:      running,
		Healthy:      healthy,
		RunningCount: runningCount,
		BaseURL:      baseURL,
		SocketPath:   s.socketPath,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *AgentControlServer) handleStart(w http.ResponseWriter, r *http.Request, dveID string) {
	if s.manager == nil {
		http.Error(w, "knirvagent manager not configured", http.StatusServiceUnavailable)
		return
	}

	var req agentControlStartRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != http.ErrBodyNotAllowed {
			http.Error(w, fmt.Sprintf("invalid start request: %v", err), http.StatusBadRequest)
			return
		}
	}

	workspacePath := strings.TrimSpace(req.WorkspacePath)
	if workspacePath == "" {
		if resolved, err := s.resolveWorkspacePath(dveID); err == nil {
			workspacePath = resolved
		}
	}
	if workspacePath == "" {
		http.Error(w, fmt.Sprintf("workspace path not configured for DVE %s", dveID), http.StatusBadRequest)
		return
	}

	s.updateWorkspacePath(dveID, workspacePath)
	s.setMergedEnv(req.ExtraEnv)

	startTimeout := time.Duration(req.StartTimeoutMS) * time.Millisecond
	if startTimeout <= 0 {
		startTimeout = 30 * time.Second
	}

	startCtx, cancel := context.WithTimeout(r.Context(), startTimeout)
	defer cancel()

	if err := s.manager.StartAgent(startCtx, dveID, startTimeout); err != nil {
		http.Error(w, fmt.Sprintf("failed to start knirvagent: %v", err), http.StatusInternalServerError)
		return
	}

	socketPath, err := s.manager.GetSocketPathForDVE(dveID)
	if err != nil {
		http.Error(w, fmt.Sprintf("agent started but socket unavailable: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, agentControlAgentResponse{
		OK:            true,
		DVEID:         dveID,
		SocketPath:    socketPath,
		WorkspacePath: workspacePath,
		RunningCount:  s.manager.RunningCount(),
	})
}

func (s *AgentControlServer) handleStop(w http.ResponseWriter, r *http.Request, dveID string) {
	if s.manager == nil {
		http.Error(w, "knirvagent manager not configured", http.StatusServiceUnavailable)
		return
	}

	stopTimeout := 10 * time.Second
	if err := s.manager.StopAgent(dveID, stopTimeout); err != nil {
		http.Error(w, fmt.Sprintf("failed to stop knirvagent: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, agentControlAgentResponse{
		OK:           true,
		DVEID:        dveID,
		RunningCount: s.manager.RunningCount(),
	})
}

func (s *AgentControlServer) handleSocket(w http.ResponseWriter, r *http.Request, dveID string) {
	if s.manager == nil {
		http.Error(w, "knirvagent manager not configured", http.StatusServiceUnavailable)
		return
	}

	socketPath, err := s.manager.GetSocketPathForDVE(dveID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	workspacePath, _ := s.resolveWorkspacePath(dveID)
	writeJSON(w, http.StatusOK, agentControlAgentResponse{
		OK:            true,
		DVEID:         dveID,
		SocketPath:    socketPath,
		WorkspacePath: workspacePath,
		RunningCount:  s.manager.RunningCount(),
	})
}

func (s *AgentControlServer) handleAgentStatus(w http.ResponseWriter, r *http.Request, dveID string) {
	socketPath, err := s.manager.GetSocketPathForDVE(dveID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, agentControlAgentResponse{
			OK:    false,
			DVEID: dveID,
			Error: err.Error(),
		})
		return
	}

	workspacePath, _ := s.resolveWorkspacePath(dveID)
	writeJSON(w, http.StatusOK, agentControlAgentResponse{
		OK:            true,
		DVEID:         dveID,
		SocketPath:    socketPath,
		WorkspacePath: workspacePath,
		RunningCount:  s.manager.RunningCount(),
	})
}

func (s *AgentControlServer) handleAgentHealth(w http.ResponseWriter, r *http.Request, dveID string) {
	socketPath, err := s.manager.GetSocketPathForDVE(dveID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, agentControlAgentResponse{
			OK:    false,
			DVEID: dveID,
			Error: err.Error(),
		})
		return
	}

	healthy := false
	if stat, err := os.Stat(socketPath); err == nil && !stat.IsDir() {
		conn, dialErr := net.DialTimeout("unix", socketPath, 250*time.Millisecond)
		if dialErr == nil {
			healthy = true
			_ = conn.Close()
		}
	}

	workspacePath, _ := s.resolveWorkspacePath(dveID)
	running := false
	runningCount := 0
	if s.manager != nil {
		running = s.manager.IsRunning()
		runningCount = s.manager.RunningCount()
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":             true,
		"dve_id":         dveID,
		"socket_path":    socketPath,
		"workspace_path": workspacePath,
		"healthy":        healthy,
		"running":        running,
		"running_count":  runningCount,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to encode knirvagent control response: %v", err)
	}
}

func unixSocketTransport(socketPath string) *http.Transport {
	dialer := &net.Dialer{}
	return &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", socketPath)
		},
	}
}

func backendBaseURL(cfg *Config) string {
	if cfg.BackendSocket != "" {
		return "http://localhost"
	}
	return fmt.Sprintf("http://localhost:%d", cfg.BackendPort)
}

func gatewayBaseURL(cfg *Config) string {
	port := cfg.GatewayPort
	if port == 0 {
		port = 8080
	}
	return fmt.Sprintf("http://localhost:%d", port)
}

func socketProxyTransport(socketPath string) http.RoundTripper {
	if strings.TrimSpace(socketPath) == "" {
		return http.DefaultTransport
	}
	return unixSocketTransport(socketPath)
}

type healthProbeResult struct {
	Name       string `json:"name"`
	Healthy    bool   `json:"healthy"`
	SocketPath string `json:"socket_path,omitempty"`
	URL        string `json:"url,omitempty"`
	LatencyMS  int64  `json:"latency_ms,omitempty"`
	Error      string `json:"error,omitempty"`
	Skipped    bool   `json:"skipped,omitempty"`
}

func probeHTTPHealth(url string, transport http.RoundTripper, socketPath string) healthProbeResult {
	result := healthProbeResult{
		Healthy:    false,
		SocketPath: socketPath,
		URL:        url,
	}
	if transport == nil {
		transport = http.DefaultTransport
	}

	client := &http.Client{
		Timeout:   3 * time.Second,
		Transport: transport,
	}
	start := time.Now()
	resp, err := client.Get(url)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.LatencyMS = time.Since(start).Milliseconds()
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Healthy = true
		return result
	}

	result.Error = fmt.Sprintf("unexpected status %d", resp.StatusCode)
	return result
}

func (app *ServerApp) collectDeepHealthChecks() map[string]healthProbeResult {
	checks := make(map[string]healthProbeResult)

	addProbe := func(name, socketPath, url string) {
		if strings.TrimSpace(url) == "" {
			checks[name] = healthProbeResult{
				Name:       name,
				SocketPath: socketPath,
				Skipped:    true,
				Error:      "health URL not configured",
			}
			return
		}

		transport := http.DefaultTransport
		if strings.TrimSpace(socketPath) != "" {
			if _, err := os.Stat(socketPath); err != nil {
				checks[name] = healthProbeResult{
					Name:       name,
					SocketPath: socketPath,
					URL:        url,
					Skipped:    true,
					Error:      err.Error(),
				}
				return
			}
			transport = unixSocketTransport(socketPath)
		}

		result := probeHTTPHealth(url, transport, socketPath)
		result.Name = name
		checks[name] = result
	}

	addProbe("backend", app.config.BackendSocket, backendBaseURL(app.config)+"/health")
	addProbe("gateway", app.config.GatewaySocket, gatewayBaseURL(app.config)+"/health")

	appDataDir, err := getAppDataDir()
	if err != nil {
		appDataDir = defaultSystemAppDataDir
	}
	socketDir := filepath.Join(appDataDir, "sockets")

	addProbe("oracle", filepath.Join(socketDir, "oracle.sock"), "http://localhost/health")
	addProbe("graph", filepath.Join(socketDir, "graph.sock"), "http://localhost/health")
	addProbe("chain", filepath.Join(socketDir, "chain.sock"), "http://localhost/health")
	addProbe("arena", filepath.Join(socketDir, "arena.sock"), "http://localhost/health")
	addProbe("hasher", filepath.Join(socketDir, "hasher.sock"), "http://localhost/health")

	if app.agentControl != nil {
		addProbe("agent_control", app.agentControl.socketPath, "http://localhost/control/status")
	}

	return checks
}

func newPrefixProxy(baseURL string, transport http.RoundTripper, sourcePrefix, targetPrefix string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	normalizePrefix := func(prefix string) string {
		if prefix == "" || prefix == "/" {
			return ""
		}
		return "/" + strings.Trim(strings.TrimSpace(prefix), "/")
	}

	sourcePrefix = normalizePrefix(sourcePrefix)
	targetPrefix = normalizePrefix(targetPrefix)

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host

			incomingPath := req.URL.Path
			if incomingPath == "" {
				incomingPath = "/"
			}

			trimmed := incomingPath
			if sourcePrefix != "" && strings.HasPrefix(trimmed, sourcePrefix) {
				trimmed = strings.TrimPrefix(trimmed, sourcePrefix)
			}
			if trimmed == "" {
				trimmed = "/"
			}
			if !strings.HasPrefix(trimmed, "/") {
				trimmed = "/" + trimmed
			}

			if targetPrefix != "" {
				if trimmed == "/" {
					req.URL.Path = targetPrefix
				} else {
					req.URL.Path = targetPrefix + trimmed
				}
			} else {
				req.URL.Path = trimmed
			}

			req.Host = target.Host
		},
		Transport: transport,
		ModifyResponse: func(resp *http.Response) error {
			ct := resp.Header.Get("Content-Type")
			if sourcePrefix != "" && strings.Contains(ct, "text/html") {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return err
				}
				resp.Body.Close()
				rewritten := strings.ReplaceAll(string(body), "/_next/static/", sourcePrefix+"/_next/static/")
				rewritten = strings.ReplaceAll(rewritten, `"/_next/`, `"`+sourcePrefix+`/_next/`)
				rewritten = strings.ReplaceAll(rewritten, "'/_next/", "'"+sourcePrefix+"/_next/")
				resp.Body = io.NopCloser(strings.NewReader(rewritten))
				resp.ContentLength = int64(len(rewritten))
				resp.Header.Set("Content-Length", strconv.Itoa(len(rewritten)))
			}
			return nil
		},
	}
	proxy.FlushInterval = -1

	return proxy, nil
}

// NewServerApp creates a new KNIRV-SERVER application
func NewServerApp(config *Config) (*ServerApp, error) {
	// Set Gin mode
	if config.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	internalAuthToken, err := randomInternalToken()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize internal authorization: %w", err)
	}
	app := &ServerApp{
		config:            config,
		router:            gin.New(),
		internalAuthToken: internalAuthToken,
	}

	// Extract backend binary
	if err := app.extractBackend(); err != nil {
		return nil, fmt.Errorf("failed to extract backend: %w", err)
	}

	// Setup middleware
	app.router.Use(gin.Logger())
	app.router.Use(gin.Recovery())

	// CORS middleware
	app.router.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		reqHeaders := c.GetHeader("Access-Control-Request-Headers")
		if reqHeaders != "" {
			c.Header("Access-Control-Allow-Headers", reqHeaders)
		} else {
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-Auth-Token, X-CSRF-Token")
		}

		c.Header("Access-Control-Expose-Headers", "X-Request-ID, Content-Length, Content-Range")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}

		c.Next()
	})

	// Setup routes
	if err := app.setupRoutes(); err != nil {
		return nil, fmt.Errorf("failed to setup routes: %w", err)
	}

	return app, nil
}

// decompressBinary decompresses a gzip-compressed embedded binary.
// Returns the raw bytes unchanged if the data is not gzip-compressed (e.g.
// plain binaries from development builds without Makefile compression).
func decompressBinary(data []byte) ([]byte, error) {
	if len(data) < 2 || data[0] != 0x1f || data[1] != 0x8b {
		return data, nil
	}
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer r.Close()
	decompressed, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress embedded binary: %w", err)
	}
	return decompressed, nil
}

// extractBinaries extracts all embedded binaries to the app data bin directory.
// Returns the bin directory path.
func extractBinaries() (string, error) {
	appDataDir, err := getAppDataDir()
	if err != nil {
		return "", fmt.Errorf("failed to get app data directory: %w", err)
	}

	binDir := filepath.Join(appDataDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bin directory: %w", err)
	}

	decompressed, err := decompressBinary(backendBinary)
	if err != nil {
		return "", fmt.Errorf("failed to decompress backend binary: %w", err)
	}

	type entry struct {
		name string
		data []byte
	}
	bins := []entry{
		{"backend_server", decompressed},
	}

	for _, b := range bins {
		dest := filepath.Join(binDir, b.name)
		if err := os.RemoveAll(dest); err != nil {
			return "", fmt.Errorf("failed to remove existing %s: %w", b.name, err)
		}
		if err := writeFileAtomically(dest, b.data, 0755); err != nil {
			return "", fmt.Errorf("failed to extract %s: %w", b.name, err)
		}
		log.Printf("Extracted %s to %s", b.name, dest)
	}

	return binDir, nil
}

// extractBackend extracts all embedded binaries to the app data directory and
// sets app.backendPath.
func (app *ServerApp) extractBackend() error {
	// Create temp directory for ephemeral artefacts.
	tempDir, err := os.MkdirTemp("", "knirv-server-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	if err := os.Chmod(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to chmod temp directory: %w", err)
	}
	app.tempDir = tempDir

	// Extract all binaries to the persistent app data bin directory.
	binDir, err := extractBinaries()
	if err != nil {
		return fmt.Errorf("failed to extract binaries: %w", err)
	}

	app.backendPath = filepath.Join(binDir, "backend_server")
	return nil
}

func mkdirIfUsable(path string) bool {
	return os.MkdirAll(path, 0755) == nil
}

// getAppDataDir returns the application data directory path. Privileged
// launches use a system location so sudo does not split sockets/data between
// /root and the invoking user's home directory.
func getAppDataDir() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); explicit != "" {
		if err := os.MkdirAll(explicit, 0755); err != nil {
			return "", fmt.Errorf("failed to create app data directory %s: %w", explicit, err)
		}
		return explicit, nil
	}

	if mkdirIfUsable(defaultSystemAppDataDir) {
		return defaultSystemAppDataDir, nil
	}

	if xdgDataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdgDataHome != "" {
		appDataDir := filepath.Join(xdgDataHome, "knirvserver")
		if err := os.MkdirAll(appDataDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create app data directory %s: %w", appDataDir, err)
		}
		return appDataDir, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	appDataDir := filepath.Join(homeDir, ".local", "share", "knirvserver")
	if err := os.MkdirAll(appDataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create app data directory %s: %w", appDataDir, err)
	}
	return appDataDir, nil
}

func getConfigDir() (string, error) {
	if configDir := strings.TrimSpace(os.Getenv("KNIRV_CONFIG_DIR")); configDir != "" {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create config directory %s: %w", configDir, err)
		}
		return configDir, nil
	}

	if mkdirIfUsable(defaultSystemConfigDir) {
		return defaultSystemConfigDir, nil
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to locate user config dir: %w", err)
	}

	configDir := filepath.Join(userConfigDir, "knirv-server")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}
	return configDir, nil
}

func getExtractedConfigDir() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}

	extractedConfigDir := filepath.Join(configDir, "config")
	if err := os.MkdirAll(extractedConfigDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create extracted config directory %s: %w", extractedConfigDir, err)
	}

	return extractedConfigDir, nil
}

// extractConfigFiles extracts embedded config files to the canonical config directory.
func extractConfigFiles() error {
	configDir, err := getExtractedConfigDir()
	if err != nil {
		return err
	}

	// Walk through embedded config files
	err = fs.WalkDir(configFiles, "config", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Read embedded file
		data, err := configFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		// Extract just the filename (remove "config/" prefix)
		filename := filepath.Base(path)
		destPath := filepath.Join(configDir, filename)

		// Write to local filesystem
		if err := writeFileAtomically(destPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write config file %s: %w", destPath, err)
		}

		log.Printf("Extracted config file: %s", destPath)
		return nil
	})

	return err
}

func writeFileAtomically(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}

	cleanup = false
	return nil
}

// setupRoutes configures the application routes
func (app *ServerApp) setupRoutes() error {
	// Create embedded filesystem
	embeddedFS, err := NewEmbeddedFS()
	if err != nil {
		return fmt.Errorf("failed to create embedded filesystem: %w", err)
	}

	// Debug: List embedded files
	log.Println("DEBUG: Listing embedded files:")
	if rdf, ok := embeddedFS.files.(fs.ReadDirFS); ok {
		entries, err := rdf.ReadDir(".")
		if err != nil {
			log.Printf("DEBUG: Error reading embedded dir: %v", err)
		} else {
			log.Printf("DEBUG: Found %d embedded entries", len(entries))
			for _, entry := range entries {
				log.Printf("DEBUG: Embedded file: %s (dir: %v)", entry.Name(), entry.IsDir())
			}
		}
	} else {
		log.Println("DEBUG: Embedded FS does not support ReadDir")
	}

	if err := app.setupDVEIngestRoutes(); err != nil {
		return fmt.Errorf("failed to configure DVE ingest routes: %w", err)
	}
	if err := app.setupProofRoutes(); err != nil {
		return fmt.Errorf("failed to configure native proof routes: %w", err)
	}
	if err := app.setupProofLedgerRoutes(); err != nil {
		return fmt.Errorf("failed to configure proof ledger routes: %w", err)
	}
	app.router.Any("/git/*path", func(c *gin.Context) {
		if app.proofLedgerRoutes == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "proof ledger unavailable"})
			return
		}
		app.proofLedgerRoutes.ServeHTTP(c.Writer, c.Request)
	})

	// Health check endpoint
	app.router.GET("/health", func(c *gin.Context) {
		deepChecks := app.collectDeepHealthChecks()
		status := "healthy"
		for _, check := range deepChecks {
			if !check.Skipped && !check.Healthy {
				status = "degraded"
				break
			}
		}

		c.JSON(200, gin.H{
			"status":      status,
			"version":     Version,
			"build_time":  BuildTime,
			"git_commit":  GitCommit,
			"deep_checks": deepChecks,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Version endpoint
	app.router.GET("/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version":    Version,
			"build_time": BuildTime,
			"git_commit": GitCommit,
		})
	})

	gatewayTransport := socketProxyTransport(app.config.GatewaySocket)
	gatewayBase := gatewayBaseURL(app.config)

	registerGatewayPrefix := func(prefix, upstreamPrefix string) error {
		proxy, err := newPrefixProxy(gatewayBase, gatewayTransport, prefix, upstreamPrefix)
		if err != nil {
			return err
		}
		handler := func(c *gin.Context) {
			proxy.ServeHTTP(c.Writer, c.Request)
		}
		app.router.Any(prefix, handler)
		app.router.Any(prefix+"/*path", handler)
		return nil
	}

	if err := registerGatewayPrefix("/network", "/"); err != nil {
		return fmt.Errorf("failed to configure /network proxy: %w", err)
	}
	if err := registerGatewayPrefix("/explorer", "/explorer"); err != nil {
		return fmt.Errorf("failed to configure /explorer proxy: %w", err)
	}
	if err := registerGatewayPrefix("/gateway", "/"); err != nil {
		return fmt.Errorf("failed to configure /gateway proxy: %w", err)
	}
	if err := registerGatewayPrefix("/turn", "/api/turn"); err != nil {
		return fmt.Errorf("failed to configure /turn proxy: %w", err)
	}
	// DVE public verification pages — proxied through the gateway to the backend socket
	if err := registerGatewayPrefix("/dve", "/dve"); err != nil {
		return fmt.Errorf("failed to configure /dve proxy: %w", err)
	}
	// Arena proxy — forward /arena/* to the gateway which proxies to the KNIRVARENA
	// Unix socket (arena.sock).  The exact /arena path redirects to /arena/ so that
	// the arena's SPA index is served; all sub-paths are forwarded with the path
	// unchanged so the gateway's gorilla/mux /arena/* routes handle stripping.
	{
		gatewayTarget, err := url.Parse(gatewayBase)
		if err != nil {
			return fmt.Errorf("failed to parse gateway URL for arena proxy: %w", err)
		}
		arenaPassProxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = gatewayTarget.Scheme
				req.URL.Host = gatewayTarget.Host
				req.Host = gatewayTarget.Host
			},
			Transport:     gatewayTransport,
			FlushInterval: -1,
		}
		app.router.Any("/arena", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "/arena/")
		})
		app.router.Any("/arena/*path", func(c *gin.Context) {
			arenaPassProxy.ServeHTTP(c.Writer, c.Request)
		})
	}
	// Network-monitor API routes — proxy to the gateway which has Go handler
	// equivalents for the Next.js API routes excluded from the static export.
	// These are handled inside the /api/*path catch-all below rather than as
	// explicit Gin routes, because Gin rejects a catch-all + explicit routes
	// on the same prefix.

	// API proxy to backend
	api := app.router.Group("/api")
	{
		api.Any("/*path", func(c *gin.Context) {
			// KNIRV-native encrypted CAS and proof protocol. This is kept on a
			// dedicated engine because the /api catch-all owns the Gin prefix.
			if strings.HasPrefix(c.Request.URL.Path, "/api/v1/knirv/") && app.proofRoutes != nil {
				app.proofRoutes.ServeHTTP(c.Writer, c.Request)
				return
			}
			// Update-status endpoint — handled locally; avoids proxying GitHub
			// calls through the backend subprocess.
			if c.Request.URL.Path == "/api/v1/system/update" {
				if c.Request.Method == http.MethodOptions {
					c.Status(http.StatusOK)
					return
				}
				if c.Request.Method != http.MethodGet {
					c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
					return
				}
				if app.upd == nil {
					c.JSON(http.StatusOK, gin.H{
						"available":       false,
						"current_version": GitCommit,
						"latest_tag":      "",
						"checked_at":      time.Now().UTC().Format(time.RFC3339),
					})
					return
				}
				status, err := app.upd.CheckUpdateAvailable()
				if err != nil {
					log.Printf("[updater] check failed: %v", err)
					c.JSON(http.StatusOK, gin.H{
						"available":       false,
						"current_version": GitCommit,
						"latest_tag":      "",
						"checked_at":      time.Now().UTC().Format(time.RFC3339),
					})
					return
				}
				c.JSON(http.StatusOK, status)
				return
			}

			// Update-apply endpoint — triggers self-update and restart.
			if c.Request.URL.Path == "/api/v1/system/update/apply" {
				if c.Request.Method == http.MethodOptions {
					c.Status(http.StatusOK)
					return
				}
				if c.Request.Method != http.MethodPost {
					c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
					return
				}
				if app.upd == nil {
					c.JSON(http.StatusServiceUnavailable, gin.H{"error": "updater not configured"})
					return
				}
				c.JSON(http.StatusAccepted, gin.H{"message": "applying update, server will restart"})
				// Flush the response before exec replaces the process.
				c.Writer.(http.Flusher).Flush()
				go func() {
					if err := app.upd.TriggerUpdate(); err != nil {
						log.Printf("[updater] apply error: %v", err)
					}
				}()
				return
			}

			// System-info endpoint — served locally with real /proc metrics.
			// Must come before the backend proxy fall-through so the richer
			// local response supersedes whatever the backend binary returns.
			if c.Request.URL.Path == "/api/v1/system/info" && c.Request.Method == http.MethodGet {
				c.JSON(http.StatusOK, collectSysInfo())
				return
			}

			// Detect network-monitor paths and proxy to the gateway instead
			// of the backend, since the gateway has Go handler equivalents.
			if strings.HasPrefix(c.Request.URL.Path, "/api/network-monitor/") {
				nmProxy, nmErr := newPrefixProxy(gatewayBase, gatewayTransport, "", "")
				if nmErr == nil {
					nmProxy.ServeHTTP(c.Writer, c.Request)
					return
				}
			}
			// DVE evidence ingest API: /api/dve/... — served locally by the
			// dveevidence sub-router (same catch-all constraint as below).
			if strings.HasPrefix(c.Request.URL.Path, "/api/dve/") && app.dveRoutes != nil {
				app.dveRoutes.ServeHTTP(c.Writer, c.Request)
				return
			}
			// Per-DVE agent API: /api/agent/{dveId}/... → KNIRVGATEWAY dynamicAgentProxy
			// (cannot register as a named Gin route; /api/*path catch-all owns the prefix)
			if strings.HasPrefix(c.Request.URL.Path, "/api/agent/") {
				agentProxy, agentErr := newPrefixProxy(gatewayBase, gatewayTransport, "", "")
				if agentErr == nil {
					agentProxy.ServeHTTP(c.Writer, c.Request)
					return
				}
			}
			// Construct backend URL
			backendURL := backendBaseURL(app.config) + c.Request.RequestURI
			transport := &http.Transport{}
			if app.config.BackendSocket != "" {
				transport = unixSocketTransport(app.config.BackendSocket)
			}

			// Create proxy request
			req, err := http.NewRequest(c.Request.Method, backendURL, c.Request.Body)
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to create proxy request"})
				return
			}

			// Copy headers
			for key, values := range c.Request.Header {
				for _, value := range values {
					req.Header.Add(key, value)
				}
			}

			// Detect SSE requests: no Timeout so streaming connections are never killed.
			isSSE := strings.Contains(c.GetHeader("Accept"), "text/event-stream") ||
				c.Request.URL.Query().Get("stream") == "1"

			var client *http.Client
			if isSSE {
				client = &http.Client{
					Transport: transport,
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
				}
			} else {
				client = &http.Client{
					Timeout:   60 * time.Second,
					Transport: transport,
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
				}
			}

			resp, err := client.Do(req)
			if err != nil {
				log.Printf("Proxy error: %v", err)
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Backend service unavailable"})
				return
			}
			defer resp.Body.Close()

			// Copy response headers
			for key, values := range resp.Header {
				for _, value := range values {
					c.Header(key, value)
				}
			}

			// For SSE responses flush each chunk immediately; otherwise buffer normally.
			c.Status(resp.StatusCode)
			if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
				flusher, canFlush := c.Writer.(http.Flusher)
				buf := make([]byte, 4096)
				for {
					n, readErr := resp.Body.Read(buf)
					if n > 0 {
						c.Writer.Write(buf[:n]) //nolint:errcheck
						if canFlush {
							flusher.Flush()
						}
					}
					if readErr != nil {
						break
					}
				}
			} else {
				io.Copy(c.Writer, resp.Body) //nolint:errcheck
			}
		})
	}

	app.router.Any("/tunnel", func(c *gin.Context) {
		c.Request.URL.Path = "/status"
		proxy, err := newPrefixProxy(gatewayBase, gatewayTransport, "/tunnel", "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to configure tunnel proxy"})
			return
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	})
	app.router.Any("/tunnel/*path", func(c *gin.Context) {
		trimmed := strings.TrimPrefix(c.Request.URL.Path, "/tunnel")
		if trimmed == "" || trimmed == "/" {
			c.Request.URL.Path = "/status"
		} else if strings.HasPrefix(trimmed, "/status") {
			c.Request.URL.Path = trimmed
		} else {
			c.Request.URL.Path = "/api" + trimmed
		}

		proxy, err := newPrefixProxy(gatewayBase, gatewayTransport, "", "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to configure tunnel proxy"})
			return
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	})

	// WebSocket proxy — must be registered before NoRoute so the upgrade
	// request reaches the backend instead of being served as a static file.
	// httputil.ReverseProxy handles the 101 Switching Protocols upgrade by
	// hijacking the underlying net.Conn, which works through Gin's wrapper.
	var backendWS *url.URL
	var wsTransport *http.Transport

	if app.config.BackendSocket != "" {
		backendWS, _ = url.Parse("http://localhost")
		wsTransport = unixSocketTransport(app.config.BackendSocket)
	} else {
		backendWS, _ = url.Parse(fmt.Sprintf("http://localhost:%d", app.config.BackendPort))
		wsTransport = &http.Transport{}
	}

	wsProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = backendWS.Scheme
			req.URL.Host = backendWS.Host
		},
		Transport: wsTransport,
	}
	wsProxy.FlushInterval = -1 // flush immediately; required for streaming / WebSocket
	wsProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		// The error handler is only invoked before the connection is hijacked,
		// so it is safe to write an HTTP error response here.
		log.Printf("WebSocket proxy error for %s: %v", r.URL.Path, err)
		http.Error(w, "WebSocket backend unavailable", http.StatusBadGateway)
	}

	wsHandler := func(c *gin.Context) {
		// All WebSocket paths — including /ws/dve/{dveId}/inner/* for inner agent PTY
		// streams — are handled by the backend subprocess (gorilla/mux), which has the
		// DVEInnerAgentWSHandler registered and connects directly to per-DVE sockets.
		wsProxy.ServeHTTP(c.Writer, c.Request)
	}
	app.router.GET("/ws", wsHandler)
	app.router.GET("/ws/*path", wsHandler)

	// Serve embedded frontend files.
	// Before falling through to the main frontend static files, check if
	// the request is a dynamic asset load from the gateway SPA (identified
	// by a Referer header pointing to a gateway route).  The gateway's
	// Next.js app lazy-loads route chunks with hardcoded /_next/ paths
	// that our HTML rewrite cannot intercept — these arrive here via NoRoute.
	app.router.NoRoute(func(c *gin.Context) {
		// If the Referer indicates this is a sub-request from the gateway
		// SPA (e.g. a dynamically-imported page chunk), proxy it through
		// the gateway's Unix socket so it gets the correct asset.
		if referer := c.GetHeader("Referer"); referer != "" {
			refURL, err := url.Parse(referer)
			if err == nil && (strings.HasPrefix(refURL.Path, "/gateway") ||
				refURL.Path == "/dashboard" ||
				refURL.Path == "/chain-explorer" ||
				refURL.Path == "/graph-explorer" ||
				refURL.Path == "/error-explorer") {
				gwProxy, gwErr := newPrefixProxy(gatewayBase, gatewayTransport, "", "")
				if gwErr == nil {
					gwProxy.ServeHTTP(c.Writer, c.Request)
					return
				}
			}
		}
		// Proxy known gateway SPA routes through the gateway at the same path,
		// preserving the route so the WebGUI loads the correct page.  The
		// gateway serves individual HTML pages (dashboard.html, etc.) with
		// injected __GATEWAY_BASE__ config.  Using the same path avoids the
		// "always shows /explorer" problem that a blanket redirect to /gateway
		// would cause.
		path := c.Request.URL.Path
		if path == "/dashboard" || path == "/chain-explorer" || path == "/graph-explorer" || path == "/error-explorer" {
			gwProxy, gwErr := newPrefixProxy(gatewayBase, gatewayTransport, "", "")
			if gwErr == nil {
				gwProxy.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		embeddedFS.ServeHTTP(c.Writer, c.Request)
	})

	return nil
}

func (app *ServerApp) setupDVEIngestRoutes() error {
	if app.dveIngest == nil {
		appDataDir, err := getAppDataDir()
		if err != nil {
			return err
		}
		store := dveevidence.NewFileStore(filepath.Join(appDataDir, "dve-evidence"))
		app.dveIngest = dveevidence.NewIngestService(store, dveevidence.ResolveKeyResolverFromEnv(), dveevidence.ResolvePolicyFromEnv())
	}

	// Registered on a dedicated engine, not app.router: the /api/*path
	// catch-all owns the /api prefix and Gin panics if named routes share it.
	// The catch-all dispatches /api/dve/ requests here.
	app.dveRoutes = gin.New()
	authorizer, err := knirvproof.NewBackendAuthorizer(app.config.BackendSocket, app.internalAuthToken)
	if err != nil {
		return fmt.Errorf("configure DVE ingest authorization: %w", err)
	}
	app.dveRoutes.Use(dveIngestAuthMiddleware(authorizer))
	dveevidence.RegisterDVEIngestRoutes(app.dveRoutes, app.dveIngest)
	return nil
}

func dveIngestAuthMiddleware(authorizer knirvproof.Authorizer) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		if authorizer == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "authorization service unavailable"})
			return
		}
		projectID := c.Param("dve")
		action := knirvproof.ActionProofRead
		var userID string
		if c.Request.Method == http.MethodPost && strings.HasSuffix(c.Request.URL.Path, "/sessions/ingest") {
			body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 64<<20))
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid ingest request"})
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
			var request struct {
				Bundle *struct {
					ProjectID string `json:"project_id"`
					UserID    string `json:"user_id"`
				} `json:"bundle"`
			}
			if err := json.Unmarshal(body, &request); err != nil || request.Bundle == nil || strings.TrimSpace(request.Bundle.ProjectID) == "" {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "bundle project_id is required"})
				return
			}
			projectID, userID, action = request.Bundle.ProjectID, request.Bundle.UserID, knirvproof.ActionProofSubmit
		}
		principal, err := authorizer.Authorize(c.Request.Context(), parts[1], projectID, action)
		if err != nil {
			status := http.StatusForbidden
			switch {
			case errors.Is(err, knirvproof.ErrUnauthorized):
				status = http.StatusUnauthorized
			case errors.Is(err, knirvproof.ErrAuthUnavailable):
				status = http.StatusServiceUnavailable
			}
			c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
			return
		}
		if principal == nil || principal.ID == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "authorization service omitted principal"})
			return
		}
		if userID != "" && userID != principal.ID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "bundle user_id does not match authenticated principal"})
			return
		}
		c.Set("knirv_principal_id", principal.ID)
		c.Next()
	}
}

func (app *ServerApp) setupProofRoutes() error {
	storeDir := app.config.ProofStoreDir
	if storeDir == "" {
		appDataDir, err := getAppDataDir()
		if err != nil {
			return err
		}
		storeDir = filepath.Join(appDataDir, "proof-protocol")
	}
	maxObjectBytes := app.config.ProofMaxObjectBytes
	if maxObjectBytes <= 0 {
		maxObjectBytes = 64 << 20
	}
	requiredReplicas := app.config.ProofRequiredReplicas
	if requiredReplicas <= 0 {
		if app.config.NetworkMode == "production" {
			requiredReplicas = 3
		} else {
			requiredReplicas = 1
		}
	}
	store, err := knirvproof.NewFileStore(storeDir, maxObjectBytes)
	if err != nil {
		return err
	}
	validatorPrivateKey, err := loadOrCreateProofValidatorKey(storeDir)
	if err != nil {
		return err
	}
	app.proofValidatorPrivateKey = validatorPrivateKey
	replicaStores := make([]*knirvproof.FileStore, 0, len(app.config.ProofReplicaDirs))
	for _, replicaDir := range app.config.ProofReplicaDirs {
		replicaDir = strings.TrimSpace(replicaDir)
		if replicaDir == "" || filepath.Clean(replicaDir) == filepath.Clean(storeDir) {
			continue
		}
		replicaStore, replicaErr := knirvproof.NewFileStore(replicaDir, maxObjectBytes)
		if replicaErr != nil {
			return fmt.Errorf("open proof replica %s: %w", replicaDir, replicaErr)
		}
		replicaStores = append(replicaStores, replicaStore)
	}
	replicator := knirvproof.Replicator(knirvproof.FilesystemReplicator{
		PrimaryLocation: "knirvserver-local-cas", Replicas: replicaStores,
	})
	trustClient, err := knirvproof.NewBackendTrustClient(app.config.BackendSocket, app.internalAuthToken)
	if err != nil {
		return err
	}
	validatorID := strings.TrimSpace(app.config.ProofValidatorID)
	if validatorID == "" {
		hostname, _ := os.Hostname()
		validatorID = "knirvserver:" + hostname
	}
	chainSocket := app.config.ProofChainSocket
	if chainSocket == "" {
		appDataDir, dataErr := getAppDataDir()
		if dataErr != nil {
			return dataErr
		}
		chainSocket = filepath.Join(appDataDir, "sockets", "chain.sock")
	}
	eventBundles, err := knirvproof.NewChainEventBundleFetcher(chainSocket)
	if err != nil {
		return err
	}
	verifier := &knirvproof.NativeVerifier{
		DEKs: trustClient, SigningKeys: trustClient.ResolveSigningKey, ValidatorID: validatorID,
		EventBundles: eventBundles,
	}
	minter, err := knirvproof.NewValidationChainMinter(validationChainSocketPath(), app.internalAuthToken)
	if err != nil {
		return err
	}
	service, err := knirvproof.NewService(store, verifier, replicator, minter, requiredReplicas)
	if err != nil {
		return err
	}
	authorizer, err := knirvproof.NewBackendAuthorizer(app.config.BackendSocket, app.internalAuthToken)
	if err != nil {
		return err
	}
	app.proofService = service
	app.proofRoutes = gin.New()
	app.proofRoutes.Use(gin.Recovery())
	knirvproof.RegisterRoutes(app.proofRoutes, service, authorizer)
	knirvproof.RegisterValidatorKeyRoute(app.proofRoutes, trustClient)
	if err := service.Resume(context.Background()); err != nil {
		return fmt.Errorf("resume proof operations: %w", err)
	}
	return nil
}

func (app *ServerApp) setupProofLedgerRoutes() error {
	root := strings.TrimSpace(app.config.ProofLedgerDir)
	if root == "" {
		appDataDir, err := getAppDataDir()
		if err != nil {
			return err
		}
		root = filepath.Join(appDataDir, "proof-ledger")
	}
	authorizer, err := knirvproof.NewBackendAuthorizer(app.config.BackendSocket, app.internalAuthToken)
	if err != nil {
		return err
	}
	ledger, err := proofledger.New(root, app.dveIngest, authorizer)
	if err != nil {
		return err
	}
	app.proofLedger = ledger
	app.proofLedgerRoutes = gin.New()
	app.proofLedgerRoutes.Use(gin.Recovery())
	proofledger.RegisterRoutes(app.proofLedgerRoutes, ledger)
	return nil
}

func randomInternalToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func loadOrCreateProofValidatorKey(storeDir string) (string, error) {
	if configured := strings.TrimSpace(os.Getenv("KNIRV_PROOF_VALIDATOR_X25519_PRIVATE_KEY")); configured != "" {
		raw, err := base64.StdEncoding.DecodeString(configured)
		if err != nil || len(raw) != 32 {
			return "", fmt.Errorf("KNIRV_PROOF_VALIDATOR_X25519_PRIVATE_KEY must encode 32 bytes")
		}
		return configured, nil
	}
	trustDir := filepath.Join(storeDir, "trust")
	if err := os.MkdirAll(trustDir, 0o700); err != nil {
		return "", err
	}
	keyPath := filepath.Join(trustDir, "validator-x25519.key")
	if existing, err := os.ReadFile(keyPath); err == nil {
		encoded := strings.TrimSpace(string(existing))
		raw, decodeErr := base64.StdEncoding.DecodeString(encoded)
		if decodeErr != nil || len(raw) != 32 {
			return "", fmt.Errorf("persisted proof validator key is invalid")
		}
		return encoded, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString(raw)
	file, err := os.OpenFile(keyPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if errors.Is(err, os.ErrExist) {
		return loadOrCreateProofValidatorKey(storeDir)
	}
	if err != nil {
		return "", err
	}
	if _, err := file.WriteString(encoded + "\n"); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return encoded, nil
}

// ── Real-time system metrics (/api/v1/system/info) ───────────────────────────
// Handled locally in the /api/*path catch-all so we can read /proc directly
// and return disk I/O, network I/O, and process counts that the backend binary
// does not expose.

type sysMemoryInfo struct {
	TotalMB     float64 `json:"total_mb"`
	UsedMB      float64 `json:"used_mb"`
	AvailableMB float64 `json:"available_mb"`
	Percentage  float64 `json:"percentage"`
}

type sysInfoPayload struct {
	CPU              float64       `json:"cpu"`
	Memory           sysMemoryInfo `json:"memory"`
	UptimeSeconds    int64         `json:"uptime_seconds"`
	OS               string        `json:"os"`
	Arch             string        `json:"arch"`
	Hostname         string        `json:"hostname"`
	Processes        int           `json:"processes"`
	DiskReadBytesPS  uint64        `json:"disk_read_bytes_per_sec"`
	DiskWriteBytesPS uint64        `json:"disk_write_bytes_per_sec"`
	NetRxBytesPS     uint64        `json:"net_rx_bytes_per_sec"`
	NetTxBytesPS     uint64        `json:"net_tx_bytes_per_sec"`
}

// ioSnapshot holds raw cumulative counters for rate calculation.
type ioSnapshot struct {
	ts        time.Time
	cpuTotal  uint64
	cpuIdle   uint64
	diskRead  uint64 // cumulative sectors read across all whole-disk devices
	diskWrite uint64 // cumulative sectors written
	netRx     uint64 // cumulative bytes received (all non-loopback interfaces)
	netTx     uint64 // cumulative bytes sent
}

var (
	sysInfoSnapshotMu sync.Mutex
	sysInfoPrevSnap   *ioSnapshot
)

func readProcCPUStat() (total, idle uint64) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		l := s.Text()
		if !strings.HasPrefix(l, "cpu ") {
			continue
		}
		fields := strings.Fields(l)
		// fields: cpu user nice system idle iowait irq softirq steal guest guest_nice
		var vals [10]uint64
		for i := 1; i < len(fields) && i-1 < len(vals); i++ {
			vals[i-1], _ = strconv.ParseUint(fields[i], 10, 64)
		}
		idle = vals[3] + vals[4] // idle + iowait
		for _, v := range vals {
			total += v
		}
		return
	}
	return
}

func readProcMeminfo() sysMemoryInfo {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return sysMemoryInfo{}
	}
	defer f.Close()
	m := make(map[string]uint64)
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		val, _ := strconv.ParseUint(fields[1], 10, 64)
		m[key] = val
	}
	total := m["MemTotal"]
	avail := m["MemAvailable"]
	used := total - avail
	pct := 0.0
	if total > 0 {
		pct = float64(used) / float64(total) * 100
	}
	return sysMemoryInfo{
		TotalMB:     float64(total) / 1024,
		UsedMB:      float64(used) / 1024,
		AvailableMB: float64(avail) / 1024,
		Percentage:  pct,
	}
}

func readProcUptime() int64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}
	f, _ := strconv.ParseFloat(fields[0], 64)
	return int64(f)
}

// readProcDiskstats returns cumulative sectors read/written across all
// whole-disk block devices (entries that have a matching /sys/block/<name> dir,
// which excludes partitions).
func readProcDiskstats() (readSectors, writeSectors uint64) {
	f, err := os.Open("/proc/diskstats")
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) < 10 {
			continue
		}
		name := fields[2]
		// Only aggregate whole-disk devices — partitions don't appear in /sys/block.
		if _, serr := os.Stat("/sys/block/" + name); serr != nil {
			continue
		}
		r, _ := strconv.ParseUint(fields[5], 10, 64) // sectors read
		w, _ := strconv.ParseUint(fields[9], 10, 64) // sectors written
		readSectors += r
		writeSectors += w
	}
	return
}

// readProcNetDev returns cumulative bytes received/sent across all non-loopback
// network interfaces.
func readProcNetDev() (rx, tx uint64) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		l := strings.TrimSpace(s.Text())
		if !strings.Contains(l, ":") {
			continue
		}
		parts := strings.SplitN(l, ":", 2)
		if strings.TrimSpace(parts[0]) == "lo" {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 9 {
			continue
		}
		r, _ := strconv.ParseUint(fields[0], 10, 64) // rx bytes
		t, _ := strconv.ParseUint(fields[8], 10, 64) // tx bytes
		rx += r
		tx += t
	}
	return
}

func readProcProcessCount() int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(e.Name()); err == nil {
			count++
		}
	}
	return count
}

func collectSysInfo() sysInfoPayload {
	cpuTotal, cpuIdle := readProcCPUStat()
	diskRead, diskWrite := readProcDiskstats()
	netRx, netTx := readProcNetDev()
	now := time.Now()

	var cpuPct float64
	var diskReadPS, diskWritePS, netRxPS, netTxPS uint64

	sysInfoSnapshotMu.Lock()
	prev := sysInfoPrevSnap
	sysInfoPrevSnap = &ioSnapshot{
		ts:        now,
		cpuTotal:  cpuTotal,
		cpuIdle:   cpuIdle,
		diskRead:  diskRead,
		diskWrite: diskWrite,
		netRx:     netRx,
		netTx:     netTx,
	}
	sysInfoSnapshotMu.Unlock()

	if prev != nil && cpuTotal > prev.cpuTotal {
		dt := now.Sub(prev.ts).Seconds()
		if dt > 0 {
			totalDelta := float64(cpuTotal - prev.cpuTotal)
			idleDelta := float64(cpuIdle - prev.cpuIdle)
			if totalDelta > 0 {
				cpuPct = (1 - idleDelta/totalDelta) * 100
				if cpuPct < 0 {
					cpuPct = 0
				}
			}
			if diskRead >= prev.diskRead {
				diskReadPS = uint64(float64(diskRead-prev.diskRead) * 512 / dt)
			}
			if diskWrite >= prev.diskWrite {
				diskWritePS = uint64(float64(diskWrite-prev.diskWrite) * 512 / dt)
			}
			if netRx >= prev.netRx {
				netRxPS = uint64(float64(netRx-prev.netRx) / dt)
			}
			if netTx >= prev.netTx {
				netTxPS = uint64(float64(netTx-prev.netTx) / dt)
			}
		}
	}

	hostname, _ := os.Hostname()
	return sysInfoPayload{
		CPU:              cpuPct,
		Memory:           readProcMeminfo(),
		UptimeSeconds:    readProcUptime(),
		OS:               runtime.GOOS,
		Arch:             runtime.GOARCH,
		Hostname:         hostname,
		Processes:        readProcProcessCount(),
		DiskReadBytesPS:  diskReadPS,
		DiskWriteBytesPS: diskWritePS,
		NetRxBytesPS:     netRxPS,
		NetTxBytesPS:     netTxPS,
	}
}

// killStaleBackend sends SIGTERM then SIGKILL to any running backend_server processes
// so port 4001 (P2P/libp2p) and other ports are free before a new instance starts.
func killStaleBackend(binaryPath string) {
	binaryName := filepath.Base(binaryPath)
	if binaryName == "" || binaryName == "." {
		binaryName = "backend_server"
	}
	if cmd := exec.Command("pkill", "-TERM", "-x", binaryName); cmd.Run() == nil {
		time.Sleep(2 * time.Second)
	}
	exec.Command("pkill", "-KILL", "-x", binaryName).Run() //nolint:errcheck
}

// waitForPortFreeMain blocks until the given TCP port is no longer occupied or the deadline.
func waitForPortFreeMain(port int, deadline time.Duration) bool {
	end := time.Now().Add(deadline)
	for time.Now().Before(end) {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return true
		}
		time.Sleep(300 * time.Millisecond)
	}
	return false
}

// killTCPPortListenersMain aggressively terminates anything listening on the given TCP port.
// This is intentionally best-effort and non-interactive so startup can self-heal stale runs.
func killTCPPortListenersMain(port int) {
	portSpec := fmt.Sprintf("%d/tcp", port)

	// Prefer fuser when available because it directly targets the socket and does not
	// require us to infer PIDs from tool output.
	if _, err := exec.LookPath("fuser"); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		if out, err := exec.CommandContext(ctx, "fuser", "-k", "-TERM", portSpec).CombinedOutput(); err == nil {
			if len(strings.TrimSpace(string(out))) > 0 {
				log.Printf("fuser terminated listeners on port %d: %s", port, strings.TrimSpace(string(out)))
			} else {
				log.Printf("fuser terminated listeners on port %d", port)
			}
		}

		// Give listeners a moment to exit, then escalate if needed.
		time.Sleep(1 * time.Second)
		if waitForPortFreeMain(port, 500*time.Millisecond) {
			return
		}

		_, _ = exec.CommandContext(ctx, "fuser", "-k", "-KILL", portSpec).CombinedOutput()
		time.Sleep(500 * time.Millisecond)
		if waitForPortFreeMain(port, 500*time.Millisecond) {
			return
		}
	}

	pids := listenerPIDsOnTCPPortMain(port)
	if len(pids) == 0 {
		log.Printf("No listener PIDs found for port %d", port)
		return
	}

	log.Printf("Port %d is in use by PID(s): %v", port, pids)
	for _, pid := range pids {
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			log.Printf("Failed to SIGTERM PID %d on port %d: %v", pid, port, err)
		}
	}

	time.Sleep(1 * time.Second)

	stillAlive := make([]int, 0, len(pids))
	for _, pid := range pids {
		if err := syscall.Kill(pid, 0); err == nil {
			stillAlive = append(stillAlive, pid)
		}
	}

	if len(stillAlive) > 0 {
		log.Printf("Escalating to SIGKILL for PID(s): %v", stillAlive)
		for _, pid := range stillAlive {
			if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
				log.Printf("Failed to SIGKILL PID %d on port %d: %v", pid, port, err)
			}
		}
	}
}

func listenerPIDsOnTCPPortMain(port int) []int {
	pids := listenerPIDsViaLsofMain(port)
	if len(pids) > 0 {
		return uniqueIntsMain(pids)
	}

	return uniqueIntsMain(listenerPIDsViaSSMain(port))
}

func listenerPIDsViaLsofMain(port int) []int {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "lsof", "-nP", "-t", fmt.Sprintf("-iTCP:%d", port), "-sTCP:LISTEN").Output()
	if err != nil && len(out) == 0 {
		return nil
	}

	return parsePIDLinesMain(string(out))
}

func listenerPIDsViaSSMain(port int) []int {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "ss", "-H", "-ltnp").Output()
	if err != nil && len(out) == 0 {
		return nil
	}

	pidPattern := regexp.MustCompile(`pid=([0-9]+)`)
	target := fmt.Sprintf(":%d", port)
	seen := make(map[int]struct{})
	var pids []int

	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, target) || !strings.Contains(line, "pid=") {
			continue
		}
		matches := pidPattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			pid, err := strconv.Atoi(match[1])
			if err != nil {
				continue
			}
			if _, ok := seen[pid]; ok {
				continue
			}
			seen[pid] = struct{}{}
			pids = append(pids, pid)
		}
	}

	return pids
}

func parsePIDLinesMain(output string) []int {
	seen := make(map[int]struct{})
	var pids []int

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		pids = append(pids, pid)
	}

	return pids
}

func uniqueIntsMain(values []int) []int {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[int]struct{}, len(values))
	out := make([]int, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func setChildEnv(env []string, key, value string) []string {
	prefix := key + "="
	filtered := env[:0]
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return append(filtered, prefix+value)
}

// ensureMainHTTPPortFree kills any existing listeners on the main HTTP port before startup.
func ensureMainHTTPPortFree(port int) error {
	if waitForPortFreeMain(port, 250*time.Millisecond) {
		return nil
	}

	log.Printf("Port %d is already in use; attempting automatic cleanup...", port)
	killTCPPortListenersMain(port)

	deadline := 10 * time.Second
	if waitForPortFreeMain(port, deadline) {
		log.Printf("Port %d is now free", port)
		return nil
	}

	return fmt.Errorf("port %d is still in use after automatic cleanup", port)
}

// validationChainSocketPath and transactionChainSocketPath resolve the Unix
// domain sockets KNIRVSERVER binds these embedded subprocesses to.
// backend_server/KNIRVGATEWAY proxy requests to them over these same
// sockets; neither process exposes a TCP port.
func validationChainSocketPath() string {
	return embeddedChainSocketPath("KNIRV_VALIDATION_CHAIN_SOCKET_PATH", "validationchain.sock")
}

func transactionChainSocketPath() string {
	return embeddedChainSocketPath("KNIRV_TRANSACTION_CHAIN_SOCKET_PATH", "transactionchain.sock")
}

func embeddedChainSocketPath(overrideEnv, filename string) string {
	if override := strings.TrimSpace(os.Getenv(overrideEnv)); override != "" {
		return override
	}
	if appDataDir, err := getAppDataDir(); err == nil {
		return filepath.Join(appDataDir, "sockets", filename)
	}
	return filepath.Join("/var/lib/knirvserver", "sockets", filename)
}

// unixHTTPClient builds an http.Client that dials a Unix domain socket
// regardless of the URL host given to it — callers use the "http://unix"
// base URL by convention (matching pkg/knirvoracle/client.go's pattern).
func unixHTTPClient(socketPath string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
			},
		},
	}
}

// startValidationChain starts the Validation Chain subprocess that
// KNIRVSERVER owns directly (pkg/embedded/validationchain — mirroring the
// using the shared embedded-service lifecycle), bound
// to a Unix domain socket. backend_server (KNIRV_CORP) only holds an HTTP
// client dialing that same socket; it does not spawn its own copy and
// nothing exposes a TCP port for it. On success, also starts the
// checkpoint-posting runtime that registers Validation Chain with
// KNIRVORACLE and periodically submits signed checkpoints — making it the
// merkle source in place of KNIRVCHAIN.
//
// Must be called last, after backend_server (and the KNIRVORACLE it spawns)
// and Transaction Chain have started, so KNIRVORACLE has already established
// its public tunnel endpoint through KNIRVGATEWAY by the time the checkpoint
// runtime attempts its first registration — registering against a gateway
// whose tunnel isn't up yet just wastes the first attempt. Failure here is
// logged and non-fatal either way, since ensureRegistered retries on every
// subsequent poll.
//
// LoadCheckpointSigner requires an explicitly provisioned service key
// (KNIRV_VALIDATION_SERVICE_KEY_FILE/_PRIVATE_KEY). startBackend already
// derives and sets that env var from root.key (Root) or boot.key (Bootnode)
// before this runs, so operators only need to set it explicitly on Client
// nodes with neither key.
func (app *ServerApp) startValidationChain(ctx context.Context) error {
	socketPath := validationChainSocketPath()
	signer, err := validationchain.LoadCheckpointSigner()
	if err != nil {
		return fmt.Errorf("load explicitly provisioned Validation Chain service signer: %w", err)
	}
	servicePrivateKey := fmt.Sprintf("%064x", signer.D)
	if err := embedded.GetManager().StartValidationChain(ctx, socketPath, servicePrivateKey); err != nil {
		return fmt.Errorf("start Validation Chain: %w", err)
	}
	log.Printf("Validation Chain started on socket %s with registered service signer %s", socketPath, validationchain.CheckpointAddress(&signer.PublicKey))
	gatewayURL, gwErr := resolvePublicURL(app.config)
	if gwErr != nil {
		gatewayURL = checkpoint.DefaultOracleGatewayURL
	}
	gatewayURL = strings.TrimRight(gatewayURL, "/")
	oracleURL := strings.TrimRight(strings.TrimSpace(os.Getenv("KNIRV_ORACLE_URL")), "/")
	if oracleURL == "" {
		oracleURL = gatewayURL
	}

	if !waitForOracleHealth(ctx, 60*time.Second, oracleURL) {
		log.Printf("Warning: KNIRVORACLE did not become healthy in time; Validation Chain checkpoint registration will retry in the background")
	}

	runtime := checkpoint.NewRuntime(socketPath, gatewayURL, oracleURL, signer)
	go runtime.Run(ctx, 30*time.Second)
	log.Printf("Validation Chain checkpoint runtime started (posting to KNIRVORACLE via gateway %s, oracle %s)", gatewayURL, oracleURL)
	return nil
}

// oracleGatewayURL resolves KNIRVORACLE's public address. KNIRVORACLE only
// runs on the root node — every instance (root or not) must reach it
// through the public KNIRVGATEWAY, never a local socket, so this always
// reuses resolvePublicURL()'s network-mode-aware resolution (testnet →
// testnet-gateway.knirv.network, production → gateway.knirv.network, devnet
// → devnet-<tag>.knirv.network, enterprise → enterprise-<tag>.knirv.network)
// instead of assuming co-location. KNIRV_ORACLE_URL overrides everything,
// for whatever exceptional deployment needs it.
func oracleGatewayURL(cfg *Config) string {
	if override := strings.TrimSpace(os.Getenv("KNIRV_ORACLE_URL")); override != "" {
		return strings.TrimRight(override, "/")
	}
	if url, err := resolvePublicURL(cfg); err == nil {
		return url
	}
	return checkpoint.DefaultOracleGatewayURL
}

// waitForOracleHealth polls KNIRVORACLE's health endpoint through the
// public gateway with a bounded retry. Non-fatal: Transaction Chain still
// starts if Oracle never comes up, it just skips the funding step (which
// requires Oracle's Faucet).
func waitForOracleHealth(ctx context.Context, timeout time.Duration, oracleURL string) bool {
	client := &http.Client{Timeout: 3 * time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, oracleURL+"/oracle/v3/health", nil)
		if err == nil {
			if resp, err := client.Do(req); err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return true
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
	return false
}

// startTransactionChain starts the Transaction Chain subprocess bound to a
// Unix domain socket. Must be called after backend_server has started, per
// the requirement that KNIRVORACLE funds every Transaction Chain wallet's
// provisional balance pool — see fundRootKeyHolderWallet. KNIRVORACLE itself
// is reached through the public gateway (it only runs on the root node),
// not assumed local even when this happens to be the root node.
func (app *ServerApp) startTransactionChain(ctx context.Context) error {
	socketPath := transactionChainSocketPath()
	if err := embedded.GetManager().StartTransactionChain(ctx, socketPath, app.internalAuthToken); err != nil {
		return fmt.Errorf("start Transaction Chain: %w", err)
	}
	log.Printf("Transaction Chain started on socket %s", socketPath)

	oracleURL := oracleGatewayURL(app.config)
	if !waitForOracleHealth(ctx, 60*time.Second, oracleURL) {
		log.Printf("Warning: KNIRVORACLE did not become healthy in time; skipping Transaction Chain wallet funding")
		return nil
	}
	// StartTransactionChain only guarantees the process was spawned, not that
	// its HTTP server has finished booting (sqlite table creation + chain
	// load happen asynchronously before app.listen()) — wait for its own
	// /health before crediting, or the very first request loses this race.
	if !waitForTransactionChainHealth(ctx, socketPath, 30*time.Second) {
		return fmt.Errorf("Transaction Chain did not become healthy within 30s")
	}
	demo := strings.TrimSpace(os.Getenv("KNIRV_ENABLE_DEMO"))
	if demo == "1" || strings.EqualFold(demo, "true") {
		app.fundRootKeyHolderWallet(ctx, oracleURL)
	} else {
		log.Printf("Transaction Chain demo funding disabled; balances must be established through Oracle settlement")
	}
	return nil
}

// waitForTransactionChainHealth polls the Transaction Chain's own /health
// endpoint over its Unix domain socket with a bounded retry.
func waitForTransactionChainHealth(ctx context.Context, socketPath string, timeout time.Duration) bool {
	client := unixHTTPClient(socketPath, 3*time.Second)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix/health", nil)
		if err == nil {
			if resp, err := client.Do(req); err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return true
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

// testFundingAmountNRN is the flat provisional balance credited to the
// root.key holder's Transaction Chain wallet on startup, for testing.
// There is no per-address staking lookup or root.key-to-wallet linkage
// anywhere in this codebase today (see pkg/embedded/validationchain.
// LoadCheckpointSigner's doc comment) — sizing this off "current
// stake" is future work once that infrastructure exists.
const testFundingAmountNRN = 100_000_000

// fundRootKeyHolderWallet derives the root.key holder's address from the
// same checkpoint signer identity (LoadCheckpointSigner) and credits
// its Transaction Chain wallet with a flat provisional NRN balance, sourced
// from KNIRVORACLE's Faucet via the public gateway. One-time per process
// start; safe to call repeatedly since both the Faucet mint and the wallet
// credit are additive (a restart adds another testFundingAmountNRN, which
// is fine for testing).
func (app *ServerApp) fundRootKeyHolderWallet(ctx context.Context, oracleURL string) {
	signer, err := validationchain.LoadCheckpointSigner()
	if err != nil {
		log.Printf("Warning: failed to load checkpoint signer for wallet funding: %v", err)
		return
	}
	address := validationchain.CheckpointAddress(&signer.PublicKey)

	if err := requestOracleFaucet(ctx, oracleURL, address, testFundingAmountNRN); err != nil {
		log.Printf("Warning: failed to fund %s from KNIRVORACLE faucet: %v", address, err)
		return
	}
	if err := creditTransactionChainWallet(ctx, transactionChainSocketPath(), app.internalAuthToken, address, testFundingAmountNRN); err != nil {
		log.Printf("Warning: failed to credit Transaction Chain wallet for %s: %v", address, err)
		return
	}
	log.Printf("Funded root.key holder wallet %s with %d NRN on Transaction Chain", address, testFundingAmountNRN)
}

func requestOracleFaucet(ctx context.Context, oracleURL, address string, amount int64) error {
	body, err := json.Marshal(map[string]any{"address": address, "amount": amount})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, oracleURL+"/test/faucet", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("oracle faucet returned %d", resp.StatusCode)
	}
	return nil
}

func creditTransactionChainWallet(ctx context.Context, txChainSocketPath, internalAuthToken, address string, amount int64) error {
	body, err := json.Marshal(map[string]any{"address": address, "amount": amount, "reason": "root-key-holder-provisional-funding"})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://unix/wallet/credit", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+internalAuthToken)
	resp, err := unixHTTPClient(txChainSocketPath, 15*time.Second).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("transaction chain /wallet/credit returned %d", resp.StatusCode)
	}
	return nil
}

// startBackend starts the embedded unified backend service
func (app *ServerApp) startBackend() error {
	if app.config.BackendSocket != "" {
		log.Printf("Starting unified backend service on socket %s...", app.config.BackendSocket)
	} else {
		log.Printf("Starting unified backend service on port %d...", app.config.BackendPort)
	}

	// Kill any stale backend_server processes from previous runs so they do not
	// hold the P2P port (4001) or other resources that would cause the new
	// instance to crash on startup.
	log.Println("Killing any stale backend_server processes...")
	killStaleBackend(app.backendPath)
	if !waitForPortFreeMain(4001, 8*time.Second) {
		log.Println("Warning: P2P port 4001 still occupied after kill — proceeding anyway")
	}

	// Pass the config file path used by the wrapper to the backend.
	// This ensures the backend uses the same configuration.
	configFile := viper.ConfigFileUsed()
	backendArgs := backendCommandArgs(configFile, app.config.AutoStartHasher)
	if configFile != "" {
		log.Printf("Passing config file to backend: %s", configFile)
	}
	if app.config.AutoStartHasher {
		log.Println("Hasher auto-start requested; forwarding -hasher to backend")
	}
	app.backendCmd = exec.Command(app.backendPath, backendArgs...)

	// Resolve the browser-facing origin before spawning backend_server. The
	// backend passes this same value to KNIRVGATEWAY, whose cloudflared tunnel
	// uses it as the authoritative public hostname during initialization.
	publicURL, err := resolvePublicURL(app.config)
	if err != nil {
		return err
	}
	publicOrigin, err := url.Parse(publicURL)
	if err != nil || publicOrigin.Hostname() == "" {
		return fmt.Errorf("invalid KNIRV public URL %q", publicURL)
	}
	userIDTag := normalizeUserIDTag(app.config.UserIDTag)
	authPublicURL := strings.TrimRight(strings.TrimSpace(os.Getenv("KNIRV_AUTH_PUBLIC_URL")), "/")
	if authPublicURL == "" {
		// Browser authentication is served by the KNIRVSERVER wrapper, not by
		// KNIRVGATEWAY's SPA. Remote installations can override this origin.
		authPublicURL = fmt.Sprintf("http://127.0.0.1:%d", app.config.Port)
	}
	env := append(os.Environ(),
		fmt.Sprintf("KNIRV_API_PORT=%d", app.config.BackendPort),
		fmt.Sprintf("KNIRV_API_SOCKET=%s", app.config.BackendSocket),
		fmt.Sprintf("KNIRV_API_SOCKET_PATH=%s", app.config.BackendSocket),
		fmt.Sprintf("KNIRV_INTERNAL_AUTH_TOKEN=%s", app.internalAuthToken),
		fmt.Sprintf("KNIRV_PROOF_VALIDATOR_X25519_PRIVATE_KEY=%s", app.proofValidatorPrivateKey),
		fmt.Sprintf("KNIRV_SERVER_BASE_URL=http://127.0.0.1:%d", app.config.Port),
		fmt.Sprintf("KNIRV_AGENT_CONTROL_GATEWAY_URL=http://127.0.0.1:%d/internal/agent-control", app.config.GatewayPort),
		fmt.Sprintf("KNIRV_AGENT_CONTROL_TOKEN=%s", app.internalAuthToken),
		fmt.Sprintf("KNIRV_PUBLIC_URL=%s", publicURL),
		fmt.Sprintf("KNIRV_AUTH_PUBLIC_URL=%s", authPublicURL),
		fmt.Sprintf("PUBLIC_HOST=%s", publicOrigin.Hostname()),
		fmt.Sprintf("KNIRV_USER_ID_TAG=%s", userIDTag),
		fmt.Sprintf("KNIRV_ENTERPRISE=%t", app.config.Enterprise),
		fmt.Sprintf("KNIRV_PROOF_REQUIRED_REPLICAS=%d", app.config.ProofRequiredReplicas),
		fmt.Sprintf("KNIRV_CONFIG_FILE=%s", configFile),
		"KNIRV_API_HOST=127.0.0.1",
	)

	// Only inject testnet defaults in testnet mode.
	// Auth is always required — testnet.yaml has no auth_required flag to disable it,
	// so this must not set KNIRV_SECURITY_AUTH_REQUIRED=false either.
	if app.config.Testnet {
		env = append(env,
			"KNIRV_SECURITY_JWT_SECRET=testnet-jwt-secret-change-this-in-production",
			"KNIRV_JWT_SECRET=testnet-jwt-secret-change-this-in-production",
		)
	}

	// Propagate paths to bundled binaries so the backend does not rely on CWD
	// or system PATH to locate knirvgateway and knirvshell.  Also propagate the
	// app data directory so the gateway runtime extracts files there instead of
	// /tmp, and so the backend can locate config/log directories consistently.
	if appDataDir, err := getAppDataDir(); err == nil {
		binDir := filepath.Join(appDataDir, "bin")
		env = append(env,
			fmt.Sprintf("KNIRV_APP_DATA_DIR=%s", appDataDir),
			fmt.Sprintf("KNIRV_GATEWAY_BINARY_PATH=%s", filepath.Join(binDir, "knirvgateway")),
			fmt.Sprintf("KNIRV_GATEWAY_BINARY_DIR=%s", binDir),
			fmt.Sprintf("KNIRV_CHAIN_BINARY_DIR=%s", binDir),
			fmt.Sprintf("KNIRV_GRAPH_BINARY_DIR=%s", binDir),
			fmt.Sprintf("KNIRV_ORACLE_BINARY_DIR=%s", binDir),
			fmt.Sprintf("KNIRV_KNIRVCLI_PATH=%s", filepath.Join(binDir, "knirvshell")),
			fmt.Sprintf("KNIRV_ARENA_SOCKET_PATH=%s", filepath.Join(appDataDir, "sockets", "arena.sock")),
		)
	}
	if configDir, err := getConfigDir(); err == nil {
		env = append(env, fmt.Sprintf("KNIRV_CONFIG_DIR=%s", configDir))
	}

	// Pass the project log directory as an absolute path so the backend writes
	// logs to <cwd>/logs/server.log regardless of where the backend binary runs from.
	if cwd, err := os.Getwd(); err == nil {
		env = append(env, fmt.Sprintf("KNIRV_PROJECT_LOG_DIR=%s", filepath.Join(cwd, "logs")))
	}

	// Add testnet environment variable if enabled
	if app.config.Testnet {
		env = append(env,
			"KNIRV_TESTNET=true",
			"KNIRV_MODE=headless",
		)
		log.Println("Starting backend in testnet mode (auth still required)")
	}

	// Propagate network mode and node role to sub-processes (backend_server, KNIRVGATEWAY).
	// KNIRVGATEWAY inherits these via os.Environ() and provisions its tunnel
	// from KNIRV_PUBLIC_URL, keeping device authorization and public ingress on
	// one origin for testnet, production, devnet, and enterprise deployments.
	networkMode := app.config.NetworkMode
	if networkMode == "" {
		networkMode = "testnet"
	}
	env = append(env, "KNIRV_NETWORK_MODE="+networkMode)
	// Keep the embedded Gateway, CLI-facing services, and xiond subprocess on
	// the same Xion network selected by the server wrapper.
	xionChainID := strings.TrimSpace(os.Getenv("XION_CHAIN_ID"))
	if xionChainID == "" {
		switch strings.ToLower(strings.TrimSpace(networkMode)) {
		case "production", "prod", "mainnet":
			xionChainID = "xion-mainnet-1"
		case "development", "dev", "devnet", "local":
			xionChainID = "xion-local-testnet-1"
		default:
			xionChainID = "xion-testnet-2"
		}
	}
	env = append(env, "XION_CHAIN_ID="+xionChainID)
	if strings.TrimSpace(os.Getenv("XION_HOME")) == "" {
		if appDataDir, err := getAppDataDir(); err == nil {
			env = append(env, "XION_HOME="+filepath.Join(appDataDir, "xion"))
		}
	}
	// Node role: root.key present → Root; boot.key present → Bootnode; neither → Client.
	// Both keys are discovered at runtime from standard filesystem locations.
	if rootKeyPath := bootkey.FindRootKey(); rootKeyPath != "" {
		env = setChildEnv(env, "CHAIN_NODE_ROLE", "Root")
		// Pass the resolved path so backend_server can find and decrypt root.key
		// without embedding or hard-coding paths.
		env = append(env, "ORACLE_KEY_PATH="+rootKeyPath)
		log.Printf("[KEY] Root node: root.key at %s", rootKeyPath)

		// Propagate Cloudflare credentials from root.key so KNIRVGATEWAY can start
		// its tunnels on Root nodes the same way it does for Bootnodes (from
		// boot.key, below). Requires ORACLE_KEY_PASSWORD to already be set.
		if rootCreds, rootErr := bootkey.LoadRootKeyCloudflareCreds(); rootCreds != nil {
			if rootCreds.CloudflareAPIToken != "" {
				env = append(env, "CLOUDFLARE_API_TOKEN="+rootCreds.CloudflareAPIToken)
			}
			if rootCreds.CloudflareZoneID != "" {
				env = append(env, "CLOUDFLARE_ZONE_ID="+rootCreds.CloudflareZoneID)
			}
			if rootCreds.CloudflareAccountID != "" {
				env = append(env, "CLOUDFLARE_ACCOUNT_ID="+rootCreds.CloudflareAccountID)
			}
			if rootCreds.CloudflareTunnelToken != "" {
				env = append(env, "CLOUDFLARE_GATEWAY_TUNNEL_TOKEN="+rootCreds.CloudflareTunnelToken)
			}
			if rootCreds.CloudflareOracleTunnelTok != "" {
				env = append(env, "CLOUDFLARE_ORACLE_TUNNEL_TOKEN="+rootCreds.CloudflareOracleTunnelTok)
			}
			log.Println("[KEY] Cloudflare credentials loaded from root.key")

			// Derive the Validation Chain checkpoint signer from root.key's root
			// private key — the same identity KNIRVORACLE itself represents on a
			// Root node — unless the operator already set an explicit override.
			// Set in this process's own environment (not the child env slice
			// above) since startValidationChain runs here in KNIRVSERVER, not in
			// the backend_server subprocess.
			if strings.TrimSpace(os.Getenv("KNIRV_VALIDATION_SERVICE_PRIVATE_KEY")) == "" &&
				strings.TrimSpace(os.Getenv("KNIRV_VALIDATION_SERVICE_KEY_FILE")) == "" &&
				rootCreds.RootPrivateKeyHex != "" {
				os.Setenv("KNIRV_VALIDATION_SERVICE_PRIVATE_KEY", rootCreds.RootPrivateKeyHex)
				log.Println("[KEY] Validation Chain checkpoint signer derived from root.key")
			}
		} else if rootErr != nil {
			log.Printf("[KEY] root.key found but Cloudflare credentials could not be loaded: %v", rootErr)
			log.Printf("[KEY] Set ORACLE_KEY_PASSWORD to unlock root.key credentials, or export CLOUDFLARE_API_TOKEN/CLOUDFLARE_ZONE_ID directly")
		}
	} else {
		bootContent, bootErr := bootkey.Load()
		if bootContent != nil {
			env = setChildEnv(env, "CHAIN_NODE_ROLE", "Bootnode")
			if bootContent.RegistrationID != "" {
				env = append(env, "KNIRV_NODE_REGISTRATION_ID="+bootContent.RegistrationID)
			}
			if bootContent.CloudflareAPIToken != "" {
				env = append(env, "CLOUDFLARE_API_TOKEN="+bootContent.CloudflareAPIToken)
			}
			if bootContent.CloudflareZoneID != "" {
				env = append(env, "CLOUDFLARE_ZONE_ID="+bootContent.CloudflareZoneID)
			}
			if bootContent.CloudflareAccountID != "" {
				env = append(env, "CLOUDFLARE_ACCOUNT_ID="+bootContent.CloudflareAccountID)
			}
			if bootContent.CloudflareTunnelToken != "" {
				env = append(env, "CLOUDFLARE_GATEWAY_TUNNEL_TOKEN="+bootContent.CloudflareTunnelToken)
			}
			log.Printf("Boot node detected: registration_id=%s", bootContent.RegistrationID)

			// Derive the Validation Chain checkpoint signer from boot.key's master
			// wallet key on Bootnodes that have no root.key, mirroring the Root
			// node derivation above, unless the operator already set an explicit
			// override.
			if strings.TrimSpace(os.Getenv("KNIRV_VALIDATION_SERVICE_PRIVATE_KEY")) == "" &&
				strings.TrimSpace(os.Getenv("KNIRV_VALIDATION_SERVICE_KEY_FILE")) == "" &&
				bootContent.MasterWalletKeyHex != "" {
				os.Setenv("KNIRV_VALIDATION_SERVICE_PRIVATE_KEY", bootContent.MasterWalletKeyHex)
				log.Println("[KEY] Validation Chain checkpoint signer derived from boot.key")
			}
		} else if bootkey.Exists() {
			// boot.key present but couldn't decrypt — still mark as bootnode role
			env = setChildEnv(env, "CHAIN_NODE_ROLE", "Bootnode")
			log.Printf("Boot node detected (boot.key found) but decryption failed: %v", bootErr)
			log.Printf("Set BOOT_KEY_PASSWORD to unlock boot.key credentials at startup")
		} else {
			// Never inherit a privileged role without its corresponding key file.
			env = setChildEnv(env, "CHAIN_NODE_ROLE", "Client")
		}
	}

	if controlSocket := defaultAgentControlSocketPath(mustAppDataDir()); controlSocket != "" {
		env = append(env, fmt.Sprintf("KNIRV_AGENT_CONTROL_SOCKET=%s", controlSocket))
	}

	// Propagate ORACLE_KEY_PASSWORD from the parent environment so the
	// backend_server subprocess can decrypt root.key without needing an
	// interactive terminal.  Without this the backend silently skips all
	// root.key secrets (API keys for Gemini, DeepSeek, Cerebras, etc.).
	if pwd := os.Getenv("ORACLE_KEY_PASSWORD"); pwd != "" {
		env = append(env, fmt.Sprintf("ORACLE_KEY_PASSWORD=%s", pwd))
	}

	app.backendCmd.Env = env
	app.backendCmd.Stdout = os.Stdout
	app.backendCmd.Stderr = os.Stderr

	if err := app.backendCmd.Start(); err != nil {
		return fmt.Errorf("failed to start unified backend: %w", err)
	}

	log.Printf("Unified backend started (PID: %d)", app.backendCmd.Process.Pid)

	// Wait for backend to accept connections on its health endpoint.
	var healthURL string
	var client *http.Client

	if app.config.BackendSocket != "" {
		healthURL = "http://localhost/health"
		client = &http.Client{
			Timeout:   2 * time.Second,
			Transport: unixSocketTransport(app.config.BackendSocket),
		}
	} else {
		healthURL = fmt.Sprintf("http://localhost:%d/health", app.config.BackendPort)
		client = &http.Client{Timeout: 2 * time.Second}
	}

	// backend_server starts KNIRVGATEWAY synchronously as part of its own Start()
	// sequence (KNIRV_CORP/packages/server/backend_server/cmd/backend_server/main.go),
	// and KNIRVGATEWAY's own health wait budgets up to StartTimeout (30s, see
	// pkg/knirvgateway/manager.go) before giving up and letting backend_server
	// continue. So backend_server's own /health endpoint can legitimately take
	// well over 30s to come up — a 30s budget here fired spuriously even on runs
	// that succeeded a bit over a minute later. 150s covers gateway's worst case
	// (stale-process/port cleanup + its own 30s health timeout) plus the graph,
	// chain, hasher, arena, and inference-provider setup that follows it.
	const backendHealthTimeout = 150 * time.Second
	deadline := time.Now().Add(backendHealthTimeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				if app.config.BackendSocket != "" {
					log.Printf("Backend ready on socket %s", app.config.BackendSocket)
				} else {
					log.Printf("Backend ready on port %d", app.config.BackendPort)
				}
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("unified backend did not become healthy within %s", backendHealthTimeout)
}

func backendCommandArgs(configFile string, autoStartHasher bool) []string {
	args := make([]string, 0, 3)
	if configFile != "" {
		args = append(args, "--config", configFile)
	}
	if autoStartHasher {
		args = append(args, "-hasher")
	}
	return args
}

func mustAppDataDir() string {
	if dir, err := getAppDataDir(); err == nil {
		return dir
	}
	return filepath.Join(os.TempDir(), "knirvserver", "data")
}

// stopBackend stops the unified backend service
func (app *ServerApp) stopBackend() {
	if app.backendCmd != nil && app.backendCmd.Process != nil {
		pid := app.backendCmd.Process.Pid
		log.Printf("Stopping unified backend (PID: %d)", pid)

		// Send SIGTERM for graceful shutdown
		if err := app.backendCmd.Process.Signal(syscall.SIGTERM); err != nil {
			log.Printf("Failed to signal backend PID %d: %v — force killing", pid, err)
			app.backendCmd.Process.Kill()
			app.backendCmd.Wait() // reap the zombie
			return
		}

		// Wait with a timeout, then escalate to SIGKILL.
		// Without this timeout a stuck backend would block shutdown forever.
		done := make(chan error, 1)
		go func() { done <- app.backendCmd.Wait() }()
		select {
		case err := <-done:
			if err != nil {
				log.Printf("Backend PID %d stopped: %v", pid, err)
			} else {
				log.Printf("Backend PID %d stopped gracefully", pid)
			}
		case <-time.After(30 * time.Second):
			log.Printf("Backend PID %d did not stop within 30s — sending SIGKILL", pid)
			app.backendCmd.Process.Kill()
			// Wait briefly for the goroutine to reap the zombie.
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				log.Printf("Warning: backend PID %d Wait() did not complete after Kill() — zombie possible", pid)
			}
			log.Printf("Backend PID %d force killed", pid)
		}
	}
}

// validateProductionCredentials ensures that either root.key (root node) or
// boot.key (bootnode) is present and valid before the server starts in production
// mode. Returns a non-nil error with a specific, actionable message on failure.
//
// Root node: root.key must exist; warns if ORACLE_KEY_PASSWORD is unset.
// Bootnode: boot.key must decrypt successfully; JWTSecret, CloudflareAPIToken,
// and CloudflareZoneID must be non-empty.
func validateProductionCredentials() error {
	rootKeyPath := bootkey.FindRootKey()
	bootKeyExists := bootkey.Exists()

	if rootKeyPath == "" && !bootKeyExists {
		searched := make([]string, 0, 20)
		for _, p := range bootkey.RootKeyCandidatePaths() {
			searched = append(searched, "  root.key: "+p)
		}
		for _, p := range bootkey.CandidatePaths() {
			searched = append(searched, "  boot.key: "+p)
		}
		return fmt.Errorf(
			"production mode requires root.key (root node) or boot.key (bootnode).\n"+
				"Neither was found in any of these locations:\n%s\n\n"+
				"Options:\n"+
				"  • Root node:  place root.key in one of the locations above\n"+
				"  • Bootnode:   place boot.key in one of the locations above and set BOOT_KEY_PASSWORD\n"+
				"  • Testnet:    omit -prod/-dev to bypass credential checks (testnet is the default)\n"+
				"  • Override:   set ORACLE_KEY_PATH or KNIRV_BOOT_KEY_PATH env vars",
			strings.Join(searched, "\n"),
		)
	}

	if rootKeyPath != "" {
		if os.Getenv("ORACLE_KEY_PASSWORD") == "" {
			log.Printf("[KEY] WARNING: root.key found at %s but ORACLE_KEY_PASSWORD is not set", rootKeyPath)
			log.Printf("[KEY]          The oracle service will start degraded — set ORACLE_KEY_PASSWORD to enable it")
		} else {
			log.Printf("[KEY] Root node credentials: root.key at %s", rootKeyPath)
		}
		return nil
	}

	// Bootnode validation.
	if os.Getenv("BOOT_KEY_PASSWORD") == "" {
		return fmt.Errorf(
			"boot.key found but BOOT_KEY_PASSWORD is not set.\n" +
				"Set BOOT_KEY_PASSWORD to the decryption password for boot.key.\n" +
				"The server cannot start in production mode without decrypted credentials.",
		)
	}

	content, err := bootkey.Load()
	if err != nil {
		return fmt.Errorf(
			"boot.key decryption failed: %w\n"+
				"Verify BOOT_KEY_PASSWORD is correct for this boot.key file",
			err,
		)
	}

	var missing []string
	if content.JWTSecret == "" {
		missing = append(missing, "JWTSecret (field 10)")
	}
	if content.CloudflareAPIToken == "" {
		missing = append(missing, "CloudflareAPIToken (field 18)")
	}
	if content.CloudflareZoneID == "" {
		missing = append(missing, "CloudflareZoneID (field 19)")
	}
	if content.CloudflareAccountID == "" {
		missing = append(missing, "CloudflareAccountID (field 20)")
	}
	if len(missing) > 0 {
		return fmt.Errorf(
			"boot.key is missing required fields: %s\n"+
				"Re-provision boot.key using the KNIRV onboarding CLI.\n"+
				"All required fields must be populated before the server can start in production mode.",
			strings.Join(missing, ", "),
		)
	}

	log.Printf("[KEY] Bootnode credentials valid (registration_id=%s)", content.RegistrationID)
	return nil
}

// Start starts the KNIRV-SERVER application
func (app *ServerApp) Start() error {
	if err := ensureMainHTTPPortFree(app.config.Port); err != nil {
		return err
	}

	if err := app.startAgentControl(); err != nil {
		return err
	}

	// Validate credentials before any service starts. Fail fast in production.
	if !app.config.Testnet {
		if err := validateProductionCredentials(); err != nil {
			return fmt.Errorf("credential validation failed: %w", err)
		}
	}

	// Provision TLS certificate from Cloudflare Origin CA when running in production
	// with boot.key credentials. Safe no-op when credentials are absent (dev/testnet).
	if !app.config.Testnet {
		if bootContent, _ := bootkey.Load(); bootContent != nil {
			certFile := viper.GetString("security.tls_cert_file")
			keyFile := viper.GetString("security.tls_key_file")
			if certFile == "" {
				certFile = tlsprovider.DefaultCertFile
			}
			if keyFile == "" {
				keyFile = tlsprovider.DefaultKeyFile
			}
			tlsCtx, tlsCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer tlsCancel()
			if err := tlsprovider.EnsureCertificate(tlsCtx, bootContent.CloudflareAPIToken, bootContent.CloudflareZoneID, certFile, keyFile); err != nil {
				log.Printf("[TLS] WARNING: certificate provisioning failed: %v", err)
				log.Printf("[TLS] Server will continue without TLS — configure a reverse proxy or fix credentials")
			} else {
				// Background goroutine renews the cert before it expires.
				// Uses the same context lifecycle as the server; cancelled on Stop().
				tlsprovider.StartAutoRenew(context.Background(), bootContent.CloudflareAPIToken, bootContent.CloudflareZoneID, certFile, keyFile, 0)
			}
		}
	}

	production := strings.EqualFold(strings.TrimSpace(app.config.NetworkMode), "production")
	requiredServices := []struct {
		name  string
		start func(context.Context) error
	}{
		{"text-embedder", app.startTextEmbedder},
		{"IPFS", app.startIPFS},
		{"Xion", app.startXion},
	}
	for _, service := range requiredServices {
		if err := service.start(context.Background()); err != nil {
			if production {
				return fmt.Errorf("required production service %s failed: %w", service.name, err)
			}
			log.Printf("Warning: %s was not started: %v", service.name, err)
		}
	}

	// Start backend (spawns KNIRVORACLE among other services)
	if err := app.startBackend(); err != nil {
		return err
	}

	// Transaction Chain starts only after KNIRVORACLE is confirmed healthy,
	// since Oracle must fund every Transaction Chain wallet's provisional
	// balance pool (see startTransactionChain / fundRootKeyHolderWallet).
	if err := app.startTransactionChain(context.Background()); err != nil {
		return err
	}

	// Validation Chain is initialized last, after KNIRVORACLE has had a
	// chance to start and establish its tunnel endpoint through
	// KNIRVGATEWAY — its checkpoint runtime registers with KNIRVORACLE on
	// first run, and doing that before Oracle's tunnel is reachable just
	// wastes the first attempt (see startValidationChain).
	if err := app.startValidationChain(context.Background()); err != nil {
		return err
	}

	// Create HTTP server
	// WriteTimeout is 0 (disabled) so SSE and long-lived streaming responses are
	// not killed by the server. ReadTimeout still guards against slow-loris attacks.
	app.server = &http.Server{
		Addr:        fmt.Sprintf("%s:%d", app.config.Host, app.config.Port),
		Handler:     app.router,
		ReadTimeout: 15 * time.Second,
		IdleTimeout: 120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting KNIRV-SERVER on %s:%d", app.config.Host, app.config.Port)
		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for server to be ready
	time.Sleep(2 * time.Second)

	return nil
}

// Stop stops the KNIRV-SERVER application
func (app *ServerApp) Stop() error {
	// Stop HTTP server
	if app.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := app.server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}

	// Stop backend
	app.stopBackend()
	stopManagedProcess("Xion", app.xionCmd)
	stopManagedProcess("IPFS", app.ipfsCmd)
	stopManagedProcess("text-embedder", app.textEmbedderCmd)

	// Stop the embedded Validation Chain / Transaction Chain subprocesses.
	if err := embedded.GetManager().Shutdown(); err != nil {
		log.Printf("Warning: embedded chain shutdown error: %v", err)
	}

	// Stop agent control plane and all running agents.
	app.stopAgentControl()

	// Clean up temp directory
	if app.tempDir != "" {
		os.RemoveAll(app.tempDir)
	}

	return nil
}

// loadConfig loads configuration from file and environment
func loadConfig() (*Config, error) {
	// Load the shared environment before resolving the network mode or reading
	// encrypted node keys. Existing process environment values always win.
	// In particular, this makes ORACLE_KEY_PASSWORD / BOOT_KEY_PASSWORD
	// available in clean testnet launches just as they are in production.
	_ = gotenv.Load()

	// Parse command line flags
	var (
		configFile = flag.String("config", "", "Path to configuration file")
		prodFlag   = flag.Bool("prod", false, "Run in production mode")
		devFlag    = flag.Bool("dev", false, "Run in development mode")
		entFlag    = flag.Bool("ent", false, "Run as an enterprise node (uses enterprise-{UserIDTag}.knirv.network)")
		userIDTag  = flag.String("user-id-tag", "", "User identity tag for devnet or enterprise public hostname")
		hasherFlag = flag.Bool("hasher", false, "Start the KNIRVHASHER training pipeline during server initialization")
		port       = flag.Int("port", 0, "Server port (overrides config)")
		host       = flag.String("host", "", "Server host (overrides config)")
	)
	flag.Parse()

	selectedClasses := 0
	for _, selected := range []bool{*prodFlag, *devFlag, *entFlag} {
		if selected {
			selectedClasses++
		}
	}
	if selectedClasses > 1 {
		log.Fatal("Choose exactly one deployment class: -prod, -dev, or -ent")
	}
	enterpriseEnv := strings.EqualFold(strings.TrimSpace(os.Getenv("KNIRV_ENTERPRISE")), "true") || os.Getenv("KNIRV_ENTERPRISE") == "1"

	// Deployment class defaults to testnet. -prod / -dev / -ent override it; when
	// neither is passed, KNIRV_ENV or KNIRV_NETWORK_MODE may still select
	// production, development, or enterprise. This mode is propagated to sub-processes
	// via KNIRV_NETWORK_MODE (see startBackend) so they follow the same
	// convention instead of resolving their own network mode independently.
	networkMode := "testnet"
	switch {
	case *prodFlag:
		networkMode = "production"
	case *devFlag:
		networkMode = "development"
	case *entFlag || enterpriseEnv:
		networkMode = "enterprise"
	case os.Getenv("KNIRV_ENV") != "":
		networkMode = os.Getenv("KNIRV_ENV")
		log.Printf("Using environment from KNIRV_ENV: %s", networkMode)
	case os.Getenv("KNIRV_NETWORK_MODE") != "":
		networkMode = os.Getenv("KNIRV_NETWORK_MODE")
	}

	// Set config file if provided
	if *configFile != "" {
		viper.SetConfigFile(*configFile)
	} else {
		// Enterprise is a distinct deployment class, but currently shares the
		// hardened production configuration baseline. Its identity remains
		// "enterprise" in Config and all child-process environment variables.
		configName := networkMode
		if networkMode == "enterprise" {
			configName = "production"
		}
		viper.SetConfigName(configName)
		viper.SetConfigType("yaml")

		// Add canonical config directory first (highest priority)
		if configDir, err := getExtractedConfigDir(); err == nil {
			viper.AddConfigPath(configDir)
		}

		// Add local paths as fallback
		viper.AddConfigPath("./config")
		viper.AddConfigPath(".")
	}

	// Set default values
	viper.SetDefault("host", "0.0.0.0")
	viper.SetDefault("port", 8090)
	viper.SetDefault("backend_port", 8082)
	if appDataDir, err := getAppDataDir(); err == nil {
		viper.SetDefault("backend_socket", filepath.Join(appDataDir, "sockets", "backend.sock"))
	}
	viper.SetDefault("gateway_port", 8080)
	viper.SetDefault("log_level", "info")
	viper.SetDefault("testnet", networkMode == "testnet")
	viper.SetDefault("proof_max_object_bytes", int64(64<<20))
	viper.SetDefault("proof_validator_id", "")
	viper.SetDefault("proof_replica_dirs", []string{})
	if appDataDir, err := getAppDataDir(); err == nil {
		viper.SetDefault("proof_ledger_dir", filepath.Join(appDataDir, "proof-ledger"))
		viper.SetDefault("proof_chain_socket", filepath.Join(appDataDir, "sockets", "chain.sock"))
	}
	if networkMode == "production" || networkMode == "enterprise" {
		viper.SetDefault("proof_required_replicas", 3)
	} else {
		viper.SetDefault("proof_required_replicas", 1)
	}

	// Enable environment variable support
	viper.AutomaticEnv()
	viper.SetEnvPrefix("NEXUS")

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		log.Println("No config file found, using defaults and environment variables")
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}
	if replicaDirs := strings.TrimSpace(os.Getenv("KNIRV_PROOF_REPLICA_DIRS")); replicaDirs != "" {
		config.ProofReplicaDirs = strings.Split(replicaDirs, string(os.PathListSeparator))
	}

	// Initialize BackendSocket if empty
	if config.BackendSocket == "" {
		if appDataDir, err := getAppDataDir(); err == nil {
			config.BackendSocket = filepath.Join(appDataDir, "sockets", "backend.sock")
		}
	}
	// KNIRVGATEWAY is TCP-only on GatewayPort — it no longer listens on a Unix
	// socket, so GatewaySocket stays empty unless explicitly configured, and
	// socketProxyTransport() falls back to http.DefaultTransport (real TCP).
	if config.GatewaySocket == "" {
		config.GatewaySocket = viper.GetString("gateway.socket_path")
	}
	if config.GatewayPort == 0 {
		config.GatewayPort = viper.GetInt("gateway.port")
		if config.GatewayPort == 0 {
			config.GatewayPort = 8080
		}
	}

	// Network mode and Testnet are derived from -prod/-dev/-ent (or their env
	// fallbacks) above, not from the config YAML, so they always win here.
	config.NetworkMode = networkMode
	config.Testnet = networkMode == "testnet"
	config.AutoStartHasher = *hasherFlag
	config.Enterprise = networkMode == "enterprise"
	config.UserIDTag = strings.TrimSpace(*userIDTag)
	if config.UserIDTag == "" {
		config.UserIDTag = strings.TrimSpace(os.Getenv("KNIRV_USER_ID_TAG"))
	}

	// Override with command line flags
	if *port != 0 {
		config.Port = *port
	}
	if *host != "" {
		config.Host = *host
	}

	return &config, nil
}

func main() {
	// Print version information
	fmt.Printf("KNIRV-SERVER v%s (built %s, commit %s)\n", Version, BuildTime, GitCommit)

	// Extract embedded config files to app data directory
	if err := extractConfigFiles(); err != nil {
		log.Printf("Warning: Failed to extract config files: %v", err)
	}

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Log testnet mode if enabled
	if config.Testnet {
		log.Println("🧪 Starting KNIRV-SERVER in testnet mode")
	}

	// Create application
	app, err := NewServerApp(config)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Initialize and start the updater (if enabled)
	selfPath, _ := os.Executable()

	// Load GitHub token from environment for security
	githubToken := os.Getenv("DEFAULT_GITHUB_TOKEN")

	upd := updater.New(updater.Config{
		Enabled:       viper.GetBool("updater.enabled"),
		PollInterval:  viper.GetDuration("updater.poll_interval"),
		GitHubRepo:    viper.GetString("updater.github_repo"),
		GitHubToken:   githubToken,
		AssetName:     "knirv-server",
		CurrentCommit: GitCommit,
		BinaryPath:    selfPath,
	})
	app.upd = upd
	go upd.Start()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start application
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	// Wait for shutdown signal
	select {
	case <-sigChan:
	}
	log.Println("Shutting down...")

	if err := app.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("KNIRV-SERVER stopped")
}
