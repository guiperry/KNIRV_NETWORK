package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath" // Ensure filepath is imported
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	// GUI functionality has been removed

	"KNIRVCHAIN/internal/dataengine"
	"KNIRVCHAIN/internal/inference"
	"KNIRVCHAIN/internal/inference/agentify"
	"KNIRVCHAIN/internal/network"

	"github.com/joho/godotenv"

	"KNIRVCHAIN/config" // Use your actual module path
	// Local package imports for types used in the code

	"KNIRVCHAIN/internal/services/monitoring"
	"KNIRVCHAIN/internal/utils"
)

// nodeRole is set by build tags in role-specific files (main_root.go, main_bootnode.go, etc.)
// or determined at runtime from command-line flags
var nodeRole config.Role = config.RoleClient // Default to Client if no build tag or flag is specified

// Use sync.Map for thread-safe access to global variables
var mainChromemManager sync.Map

// Global variables for agent mode components
var globalAgentInferencer *agentify.AgentInferencer
var globalInferenceService *inference.InferenceService

// Missing type definitions for main.go
type LevelDB struct {
	// Placeholder for LevelDB
}

func (ldb *LevelDB) GetBytes(key string) ([]byte, error) {
	return []byte("placeholder"), nil
}

func (ldb *LevelDB) PutBytes(key string, value []byte) error {
	return nil
}

type WAN struct {
	// Placeholder
}

func (w *WAN) PutValue(ctx context.Context, key string, value []byte) error {
	return nil
}

type DHT struct {
	WAN *WAN
}

type HostID struct {
	// Placeholder
}

func (hid *HostID) String() string {
	return "placeholder-id"
}

type Host struct {
	// Placeholder
}

func (h *Host) ID() *HostID {
	return &HostID{}
}

type DiscoveryManager struct {
	host *Host
	dht  *DHT
	ctx  context.Context
	// Placeholder for DiscoveryManager
}

type BlockchainStruct struct {
	ChainID          string
	p2pConsensusMgr  *P2PConsensusManager
	ConsensusManager *ConsensusManager
	// Placeholder for BlockchainStruct
}

func (bc *BlockchainStruct) Shutdown() {
	// Placeholder
}

type P2PConsensusManager struct {
	// Placeholder for P2PConsensusManager
}

func (p2p *P2PConsensusManager) Start() {
	// Placeholder
}

func (p2p *P2PConsensusManager) Stop() {
	// Placeholder
}

func (p2p *P2PConsensusManager) GetPeerCount() int {
	// Placeholder - return 0 peers
	return 0
}

type WalletManager struct {
	// Placeholder for WalletManager
}

// WalletManager methods
func (wm *WalletManager) LoadWallet(address string, role interface{}) (*Wallet, error) {
	return &Wallet{}, nil
}

func (wm *WalletManager) LoadMasterWallet(address string, role interface{}) (*Wallet, error) {
	return &Wallet{}, nil
}

func (wm *WalletManager) CreateWallet(role interface{}) (*Wallet, error) {
	return &Wallet{}, nil
}

func (wm *WalletManager) CreateMasterWallet(role interface{}) (*Wallet, error) {
	return &Wallet{}, nil
}

func (wm *WalletManager) SaveWallet(wallet *Wallet, role interface{}) error {
	return nil
}

func (wm *WalletManager) SaveMasterWallet(wallet *Wallet, role interface{}) error {
	return nil
}

// Missing function definitions
func fetchAndStorePublicIPInfo(_ interface{}, _ interface{}) error {
	return fmt.Errorf("fetchAndStorePublicIPInfo not implemented")
}

func NewChromemManager(config interface{}) (interface{}, error) {
	return &struct{}{}, nil
}

func NewWalletManager(encKey interface{}) (*WalletManager, error) {
	return &WalletManager{}, nil
}

// Missing types
type Wallet struct {
	// Placeholder for Wallet
}

// Wallet methods
func (w *Wallet) GetAddress() string {
	return "placeholder-address"
}

func (w *Wallet) GetPublicKeyHex() string {
	return "placeholder-public-key"
}

// More missing functions
func NewWalletFromPrivateKeyHex(privateKey string) *Wallet {
	return &Wallet{}
}

func LoadPaymentProcessorConfig(config interface{}) {
	// Placeholder
}

func Install(configPath string, bootnode bool, role interface{}, nonInteractive bool, walletPath string) (*config.Config, error) {
	return &config.Config{}, nil
}

type FailoverManager struct {
	// Placeholder
}

func (fm *FailoverManager) StartMonitoring() {
	// Placeholder
}

func (fm *FailoverManager) StopMonitoring() {
	// Placeholder
}

func NewFailoverManager(rootAPIURL string, cfg interface{}, configPath string, wm *WalletManager, wallet *Wallet, param interface{}, cancel interface{}) *FailoverManager {
	return &FailoverManager{}
}

func SetGlobalFailoverManager(fm interface{}) {
	// Placeholder
}

func GetGlobalFailoverManager() *FailoverManager {
	return &FailoverManager{}
}

// Missing types and functions
type GoReverseProxy struct {
	// Placeholder
}

func (grp *GoReverseProxy) Start() error {
	return nil
}

func NewGoReverseProxy(config interface{}, frontendURL, backendURL string, dataEngine interface{}) (*GoReverseProxy, error) {
	return &GoReverseProxy{}, nil
}

type ChromemManager struct {
	// Placeholder
}

func NewLevelDB(path string) (*LevelDB, error) {
	return &LevelDB{}, nil
}

func NewDiscoveryManager(chainID string, port int, clientOnly, isBootnode bool, role interface{}, config interface{}) (*DiscoveryManager, error) {
	return &DiscoveryManager{}, nil
}

// Add methods to types
func (dm *DiscoveryManager) Close() {
	// Placeholder
}

func (dm *DiscoveryManager) Run(duration time.Duration) {
	// Placeholder
}

func (ldb *LevelDB) Close() error {
	return nil
}

// Missing variables and functions
var trueGenesisBlock interface{} = &struct{}{}

func NewBlockchain(genesisBlock interface{}, chainID, minersAddress string, db *LevelDB, chromemMgr *ChromemManager, searchablePath string, cerebrasConfig interface{}) (*BlockchainStruct, error) {
	return &BlockchainStruct{ChainID: chainID}, nil
}

func initPaymentProcessor(_ interface{}, _ *LevelDB, _ interface{}) (*PaymentProcessor, error) {
	return &PaymentProcessor{}, nil
}

func NewP2PConsensusManager(bc *BlockchainStruct, db *LevelDB, discoveryMgr *DiscoveryManager, role interface{}) (*P2PConsensusManager, error) {
	return &P2PConsensusManager{}, nil
}

func EnableRelayOnHost(ctx interface{}, host interface{}, dht interface{}, config interface{}) {
	// Placeholder
}

// Remove duplicate - use the one below

type ConsensusManager struct {
	// Placeholder
}

func (cm *ConsensusManager) Stop() {
	// Placeholder
}

func NewConsensusManager(bc *BlockchainStruct, reflectURLs []string, selfURL string) *ConsensusManager {
	return &ConsensusManager{}
}

type WalletServer struct {
	portChan chan uint64
	// Placeholder
}

func (ws *WalletServer) Start() func() {
	return func() {} // Return stop function
}

func NewWalletServer(port uint64, url string) *WalletServer {
	return &WalletServer{portChan: make(chan uint64, 1)}
}

func NewNetworkMonitorManager(cfg interface{}) *monitoring.NetworkMonitorManager {
	return &monitoring.NetworkMonitorManager{}
}

type BlockchainServer struct {
	server           *http.Server
	discoveryManager *DiscoveryManager
	// Placeholder
}

func (bs *BlockchainServer) Prepare() (uint64, error) {
	bs.server = &http.Server{}
	return 8080, nil
}

func (bs *BlockchainServer) StartListenAndServe() error {
	return nil
}

func (bs *BlockchainServer) Stop(ctx context.Context) error {
	return nil
}

func NewBlockchainServer(port uint64, bc *BlockchainStruct, db *LevelDB, discoveryMgr *DiscoveryManager, p2pPort int) *BlockchainServer {
	return &BlockchainServer{}
}

type PaymentProcessor struct {
	// Placeholder
}

func (pp *PaymentProcessor) Stop() error {
	return nil
}

type EconomicsIntegration struct {
	// Placeholder
}

func (ei *EconomicsIntegration) IsLocalMode() bool {
	return false
}

func (ei *EconomicsIntegration) StopLocalEconomicsService() {
	// Placeholder
}

func initEconomicsIntegration(_ interface{}) (*EconomicsIntegration, error) {
	return &EconomicsIntegration{}, nil
}

func HandleFailoverPromotion(configPath string, cfg interface{}, wm *WalletManager) error {
	return nil
}

// getKeys returns a slice of keys from a map[string]interface{}
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// convertRelayConfig converts config.RelayConfig to network.RelayConfig
func convertRelayConfig(cfg config.RelayConfig) network.RelayConfig {
	return network.RelayConfig{
		Enabled:            cfg.Enabled,
		Resources:          cfg.Resources,
		AdvertiseInterval:  cfg.AdvertiseInterval,
		DiscoveryNamespace: cfg.DiscoveryNamespace,
	}
}

// AppVersion is the current version of the application.
// This should be set at build time using ldflags:
// go build -ldflags="-X main.AppVersion=v1.0.1"
var AppVersion = "dev" // Default if not set by ldflags

// Constants for custom update mechanism
const (
	UpdateSignalTopic = "update_signals"
	UpdateConsensusTimeout = 30 * time.Second
	UpdateDownloadTimeout = 5 * time.Minute
)

// UpdateSignal represents an update signal from the root chain
type UpdateSignal struct {
	Version     string    `json:"version"`
	DownloadURL string    `json:"download_url"`
	Checksum    string    `json:"checksum"`
	InitiatorID string    `json:"initiator_id"`
	Timestamp   time.Time `json:"timestamp"`
}

// UpdateConsensus represents consensus on an update
type UpdateConsensus struct {
	Signal      UpdateSignal `json:"signal"`
	Votes       int          `json:"votes"`
	Required    int          `json:"required"`
	Consensus   bool         `json:"consensus"`
	Timestamp   time.Time    `json:"timestamp"`
}

// handleUpdateSignal processes update signals from the root chain
func handleUpdateSignal(signal UpdateSignal, p2pMgr *P2PConsensusManager) {
	log.Printf("Received update signal for version %s from %s", signal.Version, signal.InitiatorID)

	// Seek consensus from network
	consensus := seekUpdateConsensus(signal, p2pMgr)
	if !consensus.Consensus {
		log.Printf("Update consensus not reached for version %s", signal.Version)
		return
	}

	log.Printf("Update consensus reached for version %s", signal.Version)

	// Warn chain owner
	warnChainOwner(signal)

	// Download and apply update
	if err := downloadAndApplyUpdate(signal); err != nil {
		log.Printf("Failed to download and apply update: %v", err)
		return
	}

	// Shutdown and restart
	shutdownAndRestart()
}

// seekUpdateConsensus seeks consensus from the network for an update
func seekUpdateConsensus(signal UpdateSignal, p2pMgr *P2PConsensusManager) UpdateConsensus {
	log.Printf("Seeking consensus for update to version %s", signal.Version)

	consensus := UpdateConsensus{
		Signal:    signal,
		Votes:     1, // Count our own vote
		Required:  getRequiredConsensusVotes(p2pMgr),
		Timestamp: time.Now(),
	}

	// TODO: Implement actual consensus gathering from network
	// Broadcast consensus request (implementation pending)
	_ = map[string]interface{}{
		"type":      "update_consensus_request",
		"signal":    signal,
		"timestamp": consensus.Timestamp,
	}
	// For now, assume consensus is reached if we have enough peers
	peerCount := p2pMgr.GetPeerCount()
	if peerCount >= consensus.Required-1 { // -1 because we count our own vote
		consensus.Votes = peerCount + 1
		consensus.Consensus = true
	}

	return consensus
}

// getRequiredConsensusVotes determines how many votes are required for consensus
func getRequiredConsensusVotes(p2pMgr *P2PConsensusManager) int {
	peerCount := p2pMgr.GetPeerCount()
	// Require majority consensus (more than 50%)
	required := (peerCount + 1) / 2 + 1
	if required < 2 { // Minimum 2 votes required
		required = 2
	}
	return required
}

// warnChainOwner warns the chain owner about the pending update
func warnChainOwner(signal UpdateSignal) {
	log.Printf("WARNING: Chain update to version %s is pending. The node will shut down and restart automatically.", signal.Version)
	log.Printf("Update details: %s", signal.DownloadURL)

	// TODO: Implement additional warning mechanisms (email, UI notifications, etc.)
	// For now, just log the warning
}

// downloadAndApplyUpdate downloads and applies the update
func downloadAndApplyUpdate(signal UpdateSignal) error {
	log.Printf("Downloading update from %s", signal.DownloadURL)

	_, cancel := context.WithTimeout(context.Background(), UpdateDownloadTimeout)
	defer cancel()

	// Create HTTP client
	client := &http.Client{
		Timeout: UpdateDownloadTimeout,
	}

	// Download the update
	resp, err := client.Get(signal.DownloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Create temporary file for the update
	tempFile, err := os.CreateTemp("", "knirvchain-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy download to temp file
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save update: %w", err)
	}

	// Verify checksum if provided
	if signal.Checksum != "" {
		if err := verifyChecksum(tempFile.Name(), signal.Checksum); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	// Get current executable path
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create backup of current executable
	backupPath := executablePath + ".backup"
	if err := copyFile(executablePath, backupPath); err != nil {
		log.Printf("Warning: Failed to create backup: %v", err)
		// Continue anyway
	}

	// Replace executable with update
	if err := os.Rename(tempFile.Name(), executablePath); err != nil {
		return fmt.Errorf("failed to apply update: %w", err)
	}

	// Make executable
	if err := os.Chmod(executablePath, 0755); err != nil {
		return fmt.Errorf("failed to make executable: %w", err)
	}

	log.Printf("Update applied successfully")
	return nil
}

// verifyChecksum verifies the downloaded file against the expected checksum
func verifyChecksum(filePath, expectedChecksum string) error {
	// TODO: Implement checksum verification
	// For now, just log that verification is skipped
	log.Printf("Checksum verification skipped for %s", filePath)
	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// shutdownAndRestart shuts down the current process and restarts with the updated binary
func shutdownAndRestart() {
	log.Printf("Shutting down for update...")

	executablePath, err := os.Executable()
	if err != nil {
		log.Printf("Could not get executable path: %v. Please restart manually.", err)
		os.Exit(1)
		return
	}

	var scriptContent, scriptName, shellCmd, shellArg string

	if runtime.GOOS == "windows" {
		scriptName = "restart_wrapper.bat"
		scriptContent = fmt.Sprintf(`
@echo off
echo Restarting application with update...
timeout /t 2 /nobreak > nul
start "" /B "%s" %s
(goto) 2>nul & del "%%~f0"
`, executablePath, strings.Join(os.Args[1:], " "))
		shellCmd = "cmd.exe"
		shellArg = "/C"
	} else { // Linux & macOS
		scriptName = "restart_wrapper.sh"
		argsString := ""
		for _, arg := range os.Args[1:] {
			argsString += fmt.Sprintf(" '%s'", strings.ReplaceAll(arg, "'", "'\\''"))
		}
		scriptContent = fmt.Sprintf(`#!/bin/sh
echo "Restarting application with update..."
sleep 2
nohup "%s"%s > /dev/null 2>&1 &
rm -- "$0"
`, executablePath, argsString)
		shellCmd = "/bin/sh"
	}

	tempDir := os.TempDir()
	scriptPath := filepath.Join(tempDir, scriptName)

	err = os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		log.Printf("Failed to write restart script: %v. Please restart manually.", err)
		os.Exit(1)
		return
	}
	log.Printf("Restart script written to: %s", scriptPath)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command(shellCmd, shellArg, scriptPath)
	} else {
		cmd = exec.Command(scriptPath)
	}

	err = cmd.Start()
	if err != nil {
		log.Printf("Failed to start restart wrapper: %v. Please restart manually.", err)
		_ = os.Remove(scriptPath)
		os.Exit(1)
		return
	}

	log.Printf("Restart wrapper started with PID %d. Old application exiting.", cmd.Process.Pid)
	os.Exit(0)
}

// Define a global log file variable
var appLogFile *os.File

// configureLogLevel sets the global log level based on the provided level string
func configureLogLevel(level string) error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
	}

	normalizedLevel := strings.ToLower(level)
	if !validLevels[normalizedLevel] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, error, or fatal)", level)
	}

	// Set the log level in the global logger
	// Note: Go's standard log package doesn't have built-in levels, but we can use prefixes
	// For more advanced logging, consider using logrus or zap in the future
	log.Printf("Log level configured to: %s", normalizedLevel)

	// Store the log level for use by other components
	os.Setenv("KNIRV_LOG_LEVEL", normalizedLevel)

	return nil
}

// Removed setupDirectBootnodeSignalHandler to prevent signal handling conflicts
// All roles now use the unified waitForShutdownSignal() function

// Removed isBootnodeMode() function - no longer needed since signal handling is unified

func main() {
	// Signal handling will be unified for all roles using waitForShutdownSignal()
	// No need for separate bootnode signal handler that conflicts with the main one

	err := godotenv.Load(".key") // Loads .key file from the current directory
	if err != nil {
		log.Println("Info: .key file not found or could not be loaded. Using system environment variables or config files.")
	}

	// --- Setup Application Logging to File and Console ---

	var logFilePath = "logs/KNIRVCHAIN.log" // You can make this configurable
	var errOpenLog error

	// Ensure logs directory exists
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Failed to create log directory %s: %v", logDir, err)
	}

	// walletSrv variable is now handled directly where needed
	appLogFile, errOpenLog = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if errOpenLog != nil {
		log.Fatalf("Failed to open log file %s: %v", logFilePath, errOpenLog)
	}
	defer appLogFile.Close() // Defer close immediately after successful open
	// We will close appLogFile at the end of main

	// GUI functionality has been removed

	// Initial log before setting multiwriter
	log.Printf("***********STARTING KNIRVCHAIN***********")
	log.Printf("VERSION: %s", AppVersion)
	log.Printf("OS: %s", runtime.GOOS)
	log.Printf("Arch: %s", runtime.GOARCH)
	log.Printf("LOGFILE: %s", logFilePath)
	// Log to both Stderr (console) and the application log file
	// Set log output to both console and file
	log.SetOutput(io.MultiWriter(os.Stderr, appLogFile))

	// --- Define Flags ---
	// Viper will handle config file paths, but we can allow an override.
	cliConfigPath := flag.String("config", "", "Path to config file (e.g., config.json, config.yaml)")
	var loadedConfigPath string // Track where config was loaded from

	// Define ALL flags with default values first
	// These flags can override Viper-loaded values.
	httpPortFlag := flag.Uint("port", 0, "HTTP port (overrides config if set, 0 means use config)")
	p2pPortFlag := flag.Uint("p2p.port", 0, "Libp2p host port (overrides config if set, 0 means use config)")
	walletPortFlag := flag.Uint("wallet_port", 0, "HTTP port for the wallet server (overrides config if set, 0 means use config)")
	dbPathFlag := flag.String("shared_database_path", "", "Filepath for the chain's database (overrides config if set)")
	minerAddressFlag := flag.String("miners_address", "", "Miner's address for rewards (overrides config if set)")
	noWalletServer := flag.Bool("no-wallet-server", false, "Disable wallet server startup")
	clientOnly := flag.Bool("client-only", false, "Run as a client-only node (reduced resource usage)")
	useGUI := flag.Bool("gui", false, "DEPRECATED: GUI functionality has been removed")
	runNetworkMode := flag.Bool("network", false, "Run in multi-node network mode (main + reflection)")
	runPeerMode := flag.Bool("dev", false, "Run as a dev reflection node (requires install_complete in config)")
	bootnode := flag.Bool("bootnode", false, "Run as a bootnode for the KNIRVCHAIN network")
	root := flag.Bool("root", false, "") // Hidden backdoor flag
	agent := flag.Bool("agent", false, "Run in agent mode with inference engine and data engine sharing DHT resources")
	nonInteractive := flag.Bool("non-interactive", false, "Automatically accept all default and randomly created values for installation")
	skipInstall := flag.Bool("skip-install", false, "Skip installation process even if InstallComplete is false")
	testnetMode := flag.Bool("testnet", false, "Run in testnet mode with simplified configuration")
	disableP2P := flag.Bool("disable-p2p", false, "Disable P2P messaging and consensus (reduces memory usage and network traffic)")

	// The -role flag helps determine which section of the config to load if other role flags aren't set.
	roleFlag := flag.String("role", "", "Node role (Root, Bootnode, Peer, Client) - overrides auto-detection if other role flags are absent.")

	logLevelFlag := flag.String("logLevel", "info", "Set the logging level (e.g., debug, info, warn, error)")
	// var reflectFlags []string // Viper can load string slices from config.
	// flag.Func("reflect", "Add a reflection URL (can be specified multiple times)", func(url string) error {
	// reflectFlags = append(reflectFlags, url)
	// return nil
	// })

	// Parse ALL flags ONCE
	flag.Parse()

	// Configure logging level from flag if set
	if *logLevelFlag != "" {
		log.Printf("Setting log level to: %s", *logLevelFlag)
		if err := configureLogLevel(*logLevelFlag); err != nil {
			log.Printf("Warning: Failed to set log level to %s: %v", *logLevelFlag, err)
		}
	}

	// --- Determine Node Role based on flags ---
	// Priority: build tags > explicit role flags > -role flag > default from config/Viper
	// Note: If built with role-specific build tags, nodeRole will already be set in init()

	// Only override the role from build tags if explicit flags are provided
	if *root {
		nodeRole = config.Root
		log.Println("Role override from command line: Root")
	} else if *bootnode {
		nodeRole = config.RoleBootnode
		log.Println("Role override from command line: Bootnode")
	} else if *runPeerMode {
		nodeRole = config.RolePeer
		log.Println("Role override from command line: Peer")
	} else if *clientOnly {
		nodeRole = config.RoleClient
		log.Println("Role override from command line: Client")
	} else if *roleFlag != "" {
		// Validate the role from -role flag
		validRole := config.Role(*roleFlag)
		if validRole == config.Root || validRole == config.RoleBootnode || validRole == config.RolePeer || validRole == config.RoleClient {
			nodeRole = validRole
			log.Printf("Role override from -role flag: %s", validRole)
		} else {
			log.Printf("Warning: Invalid role '%s' provided via -role flag. Using default role.", *roleFlag)
		}
	}

	// Handle agent flag - enables both inference engine and data engine with DHT integration
	if *agent {
		log.Println("Agent mode enabled: Inference engine and data engine will be initialized with DHT integration")
	}

	log.Printf("Using node role: %s", nodeRole.String())

	// --- Load Configuration using Viper ---
	var cfg *config.Config
	var viperErr error
	cfg, loadedConfigPath, viperErr = config.LoadConfigurationViper(nodeRole, *cliConfigPath)
	if viperErr != nil {
		log.Fatalf("Error loading configuration via Viper: %v", viperErr)
	} else if nodeRole == config.Root {
		log.Println("[INFO] Root using configuration directly from constants (no config file operations).")
	} else {
		log.Printf("Successfully loaded config from: %s", loadedConfigPath)
	}
	if cfg == nil {
		log.Fatalf("Configuration could not be loaded or initialized.")
	}

	// For Root role, load parameters from env.local
	if nodeRole == config.Root {
		if err := loadRootNodeParameters(cfg); err != nil {
			log.Printf("WARNING: Failed to load parameters from env.local: %v", err)
			// Continue with default or CLI-provided parameters
		}
	}

	// Fetch and store public IP info AFTER config is loaded and role is known
	if err := fetchAndStorePublicIPInfo(cfg, nodeRole); err != nil {
		log.Printf("Warning: Could not initialize public IP information: %v", err)
		// This is not a fatal error, the application can continue
	}

	// If this is a Root node and ServerPublicHost is not set, use the discovered public IP.
	if nodeRole == config.Root && (cfg.NodeJSServices.TunnelRegistry.ServerPublicHost == "" || cfg.NodeJSServices.TunnelRegistry.ServerPublicHost == "localhost") {
		var discoveredIP string
		if ipInfo, ok := cfg.PublicIPInfo["ip"].(string); ok && ipInfo != "" {
			discoveredIP = ipInfo
		} else if utils.LastIPInfoResponse != nil && utils.LastIPInfoResponse.IP != "" {
			discoveredIP = utils.LastIPInfoResponse.IP
		}

		if discoveredIP != "" {
			cfg.NodeJSServices.TunnelRegistry.ServerPublicHost = discoveredIP
			log.Printf("Updated NodeJSServices.TunnelRegistry.ServerPublicHost to discovered public IP: %s", discoveredIP)
		}
	}

	// Initialize ChromemManager and store it in the sync.Map for global access
	ChromemCfgForManager := &config.ChromemConfig{
		Path:           cfg.SearchableDatabasePath,
		CerebrasConfig: (*config.CerebrasConfig)(cfg.Chromem.CerebrasConfig),
	}

	// Initialize ChromemManager
	chromemManager, err := NewChromemManager(ChromemCfgForManager) // Create the ChromemManager
	if err != nil {
		log.Fatalf("Failed to initialize ChromemManager: %v", err)
	}

	// Store in the sync.Map for global access
	mainChromemManager.Store("chromemManager", chromemManager)

	// Create wallet manager with encryption key
	encKey := utils.DeriveEncryptionKey()
	wm, err := NewWalletManager(encKey)

	if err != nil {
		log.Fatalf("Failed to initialize wallet manager: %v", err)
	}

	// Declare wallet variables
	var wallet *Wallet
	var masterWallet *Wallet

	// Implement consistency checks for non-Root roles
	if nodeRole != config.Root {
		// Check MinersAddress and wallet.dat consistency
		if cfg.MinersAddress != "" {
			// Try to load wallet using wallet_manager
			_, err = wm.LoadWallet(cfg.MinersAddress, nodeRole)
			if err != nil {
				if os.IsNotExist(err) || strings.Contains(err.Error(), "wallet address mismatch") { // Catch mismatch too
					log.Printf("WARNING: MinersAddress '%s' is configured but wallet file not found or mismatched.", cfg.MinersAddress)
					// Try to load any wallet instead of forcing reinstallation
					walletPath, _ := config.GetWalletPath(nodeRole)
					if _, statErr := os.Stat(walletPath); statErr == nil {
						// Wallet file exists, try to load it
						loadedWallet, loadErr := wm.LoadWallet("", nodeRole)
						if loadErr != nil {
							log.Printf("ERROR: Found wallet file but failed to load: %v", loadErr)
							log.Println("This node requires reinstallation. Forcing installer.")
							cfg.InstallComplete = false // Force installer
						} else {
							// Update config with the loaded wallet address
							cfg.MinersAddress = loadedWallet.GetAddress()
							wallet = loadedWallet // Assign to the main wallet variable
							log.Printf("Loaded wallet with address '%s', updating config", cfg.MinersAddress)
							config.SaveConfigToUserDir(cfg, nodeRole)
						}
					} else {
						log.Println("This node requires reinstallation. Forcing installer.")
						cfg.InstallComplete = false // Force installer
					}
				} else {
					log.Printf("ERROR: Failed to load wallet for address '%s': %v. This is a non-fatal error, attempting to continue.", cfg.MinersAddress, err)
					// Try to continue instead of fatal exit
				}
			} else {
				log.Printf("Successfully validated wallet for address '%s'", cfg.MinersAddress)
			}
		} else {
			// MinersAddress is empty, try to load any wallet
			walletPath, _ := config.GetWalletPath(nodeRole) // Assuming GetWalletPath is appropriate for non-root here
			if _, statErr := os.Stat(walletPath); statErr == nil {
				// Wallet file exists, try to load it
				loadedWallet, loadErr := wm.LoadWallet("", nodeRole) // Use temp var
				if loadErr != nil {
					log.Printf("ERROR: Found wallet file but failed to load: %v", loadErr)
					log.Println("This node requires reinstallation. Forcing installer.")
					cfg.InstallComplete = false // Force installer
				} else {
					// Update config with the loaded wallet address
					cfg.MinersAddress = loadedWallet.GetAddress()
					wallet = loadedWallet // Assign to the main wallet variable
					log.Printf("Loaded wallet with address '%s', updating config", cfg.MinersAddress)
					config.SaveConfigToUserDir(cfg, nodeRole)
				}
			} else {
				log.Println("WARNING: No wallet file found and MinersAddress not configured.")
				log.Println("This node requires reinstallation. Forcing installer.")
				cfg.InstallComplete = false // Force installer
			}
		}

		// For Bootnode role check MasterAddress and master_wallet.dat consistency
		if nodeRole == config.RoleBootnode {
			if cfg.MasterAddress != "" {
				// Try to load master wallet using wallet_manager
				if _, err := wm.LoadMasterWallet(cfg.MasterAddress, nodeRole); err != nil {
					if os.IsNotExist(err) {
						log.Printf("WARNING: MasterAddress '%s' is configured but master wallet file not found.", cfg.MasterAddress)
						log.Println("This node requires reinstallation. Forcing installer.")
						cfg.InstallComplete = false // Force installer instead of exiting
					} else if strings.Contains(err.Error(), "master wallet address mismatch") {
						log.Printf("WARNING: %v", err)
						log.Println("This node requires reinstallation. Forcing installer.")
						cfg.InstallComplete = false // Force installer instead of exiting
					} else {
						log.Printf("WARNING: Failed to load master wallet for address '%s': %v", cfg.MasterAddress, err)
						log.Println("This node requires reinstallation. Forcing installer.")
						cfg.InstallComplete = false // Force installer instead of exiting
					}
				} else {
					log.Printf("Successfully validated master wallet for address '%s'", cfg.MasterAddress)
				}
			} else {
				// MasterAddress is empty, try to load any master wallet
				masterWalletPath, _ := config.GetMasterWalletPath(nodeRole)
				if _, statErr := os.Stat(masterWalletPath); statErr == nil {
					// Master wallet file exists, try to load it
					masterWallet, err = wm.LoadMasterWallet("", nodeRole)
					if err != nil {
						log.Printf("WARNING: Found master wallet file but failed to load: %v", err)
						log.Println("This node requires reinstallation. Forcing installer.")
						cfg.InstallComplete = false // Force installer instead of exiting
					} else {
						// Update config with the loaded master wallet address
						cfg.MasterAddress = masterWallet.GetAddress()
						log.Printf("Loaded master wallet with address '%s', updating config", cfg.MasterAddress)
						config.SaveConfigToUserDir(cfg, nodeRole)
					}
				} else if os.IsNotExist(statErr) {
					log.Println("WARNING: No master wallet file found and MasterAddress not configured for bootnode.")
					log.Println("This node requires reinstallation. Forcing installer.")
					cfg.InstallComplete = false // Force installer instead of exiting
				}
			}
		}
	} else {
		// For Root role, skip wallet file operations

		if cfg.MinersAddress == utils.BLOCKCHAIN_ADDRESS || cfg.MinersAddress == "_Faucet" {
			log.Printf("Root: Configured MinersAddress ('%s') matches predefined blockchain identity. Using hardcoded private key from constants.go.", cfg.MinersAddress)
			wallet = NewWalletFromPrivateKeyHex(utils.BLOCKCHAIN_PRIVATE_KEY)
			if wallet.GetAddress() != utils.BLOCKCHAIN_ADDRESS {
				log.Fatalf("FATAL: BLOCKCHAIN_PRIVATE_KEY in constants.go does not correspond to BLOCKCHAIN_ADDRESS. Expected %s, got %s. Please check constants.go.", utils.BLOCKCHAIN_ADDRESS, wallet.GetAddress())
			}
			log.Printf("[INFO] - Root: Successfully initialized wallet using hardcoded key for %s.", cfg.MinersAddress)
		} else {
			// Skip all wallet file operations for root role
			log.Printf("Root: Skipping wallet file operations")
			_ = NewWalletFromPrivateKeyHex(utils.BLOCKCHAIN_PRIVATE_KEY) // Still need a wallet instance
		}
		log.Printf("[INFO] - Root wallet setup process complete.")
	}

	// Track which flags were explicitly set
	flagsSet := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		flagsSet[f.Name] = true
	})

	// Apply flag overrides to the Viper-loaded config
	if !cfg.IsPeer {
		if flagsSet["port"] && *httpPortFlag != 0 { // Check if flag was set and not its default
			cfg.Port = uint64(*httpPortFlag)
		}
		if flagsSet["p2p.port"] && *p2pPortFlag != 0 {
			cfg.P2PPort = uint64(*p2pPortFlag)
		}
		if flagsSet["wallet_port"] && *walletPortFlag != 0 {
			cfg.WalletPort = uint64(*walletPortFlag)
		}
		if flagsSet["shared_shared_database_path"] && *dbPathFlag != "" {
			cfg.BlockchainDatabasePath = *dbPathFlag
		}
		if flagsSet["miners_address"] && *minerAddressFlag != "" {
			cfg.MinersAddress = *minerAddressFlag
		}
		if flagsSet["client-only"] {
			cfg.ClientOnly = *clientOnly
		}
		// Set IsPeer based on runPeerMode flag
		if flagsSet["dev"] {
			cfg.IsPeer = *runPeerMode
		}
		// No Wallet Server is always applied if present
		if flagsSet["no-wallet-server"] {
			cfg.NoWalletServer = *noWalletServer
		}
		// Always apply GUI flag override, regardless of mode, if it was set
		if flagsSet["gui"] {
			cfg.UseGUI = *useGUI
		}
		// cfg.ClientOnly and cfg.IsRoot are now primarily set by viper_loader based on nodeRole.
		// Flags -client-only and -root are used to determine nodeRole initially.

		// if len(reflectFlags) > 0 { // Viper loads ReflectionURLs from config file
		// cfg.ReflectionURLs = reflectFlags
		// }

		// Update nodeRole based on the config after all flag overrides have been applied
		// This ensures nodeRole is consistent with the final config state
		updatedRole := config.DetermineRoleFromConfig(cfg)
		if updatedRole != nodeRole {
			log.Printf("Updating node role from %s to %s based on final config state", nodeRole.String(), updatedRole.String())
			nodeRole = updatedRole
		}

		// Set bootnode flag in config if -bootnode flag was used
		if flagsSet["bootnode"] && *bootnode {
			cfg.IsBootnode = true
			cfg.Bootnode.Enabled = true
			log.Println("Running in bootnode mode")
		}

		// Set agent flag in config if -agent flag was used
		if flagsSet["agent"] && *agent {
			cfg.AgentMode.Enabled = true
			cfg.InferenceEngine.Enabled = true
			cfg.DataEngine.Enabled = true
			cfg.InferenceEngine.ShareDHTMetrics = true
			log.Println("Running in agent mode with inference engine and data engine enabled")
		}

		// Set testnet flag in config if -testnet flag was used
		if flagsSet["testnet"] && *testnetMode {
			cfg.Testnet.Enabled = true
			config.ApplyTestnetDefaults(cfg)
			log.Println("Running in testnet mode with simplified configuration")
		}

		// If root flag is set, enable payment processor
		if nodeRole == config.Root {
			// Load payment processor config from environment variables
			LoadPaymentProcessorConfig(&cfg.PaymentProcessor)
		}
	}

	// Database path resolution is now handled inside viper_loader.go's resolveDynamicPathsViper
	// It considers the role and existing cfg.BlockchainDatabasePath.
	// If dbPathFlag was set, it would have overridden cfg.BlockchainDatabasePath before resolveDynamicPathsViper.
	log.Printf("Final BlockchainDatabasePath after Viper and flag processing: %s", cfg.BlockchainDatabasePath)
	if cfg.BlockchainDatabasePath != "" {
		log.Printf("Final BlockchainDatabasePath: %s", cfg.BlockchainDatabasePath)
	}

	// Set ChainID for the main node if not in dev mode
	if !cfg.IsPeer {
		// ChainID is now set by viper_loader based on role and config
	}

	// --- Mode Selection ---
	// Determine if we're running in network mode (requires explicit -network flag)
	isNetworkMode := *runNetworkMode

	// Check if installation is needed
	if !cfg.InstallComplete && !*skipInstall {
		log.Println("Installation not complete - running installer...")
		// Pass the current cfg, which might have come from default_config.json or be a fresh default
		updatedCfg, err := Install(loadedConfigPath, *bootnode, nodeRole, *nonInteractive, "") // Pass empty wallet path
		if err != nil {
			log.Fatalf("Failed to run installer: %v", err)
		}
		cfg = updatedCfg // Use the config potentially modified and saved by the installer
		log.Printf("Installer finished. Using updated configuration for ChainID: %s", cfg.ChainID)

		// Ensure InstallComplete is set to true and save the config again
		cfg.InstallComplete = true
		log.Printf("Ensuring InstallComplete is set to true for role %s", nodeRole.String())
		config.SaveConfigToUserDir(cfg, nodeRole)

		// Re-apply GUI flag override after reload
		if flagsSet["gui"] {
			cfg.UseGUI = *useGUI
		}

		// Update nodeRole based on the config after installation
		// This ensures nodeRole is consistent with the installed config
		updatedRole := config.DetermineRoleFromConfig(cfg)
		if updatedRole != nodeRole {
			log.Printf("Updating node role from %s to %s based on installed config", nodeRole.String(), updatedRole.String())
			nodeRole = updatedRole
		}
	} else {
		// If installation is complete or skip-install flag is set, ensure we continue with node initialization
		if *skipInstall {
			log.Printf("Skipping installation due to --skip-install flag. Continuing with node initialization...")
		} else {
			log.Printf("Installation is complete. Continuing with node initialization...")
		}
	}
	// --- Context and WaitGroup ---
	mainCtx := context.Background()

	ctx, cancel := context.WithCancel(mainCtx)
	var wg sync.WaitGroup

	// Store cancel function and wait group in global variables for GUI access
	// Note: These are declared in altgui.go and used for GUI coordination

	// --- Initialize FailoverManager for Bootnode role ---
	if nodeRole == config.RoleBootnode && cfg.IsBootnode {
		// Initialize FailoverManager to monitor root node
		var rootAPIURL string
		if cfg.CurrentOracleNodeAPIURL != "" {
			rootAPIURL = cfg.CurrentOracleNodeAPIURL
		} else {
			// Try to construct from root configuration if available
			log.Println("[FailoverManager] No CurrentOracleNodeAPIURL configured, failover monitoring will be disabled")
		}

		if rootAPIURL != "" {
			fm := NewFailoverManager(rootAPIURL, cfg, loadedConfigPath, wm, wallet, nil, cancel)
			if fm != nil {
				SetGlobalFailoverManager(fm)
				log.Printf("[FailoverManager] Initialized for monitoring root at: %s", rootAPIURL)
				// Start monitoring in a separate goroutine
				go fm.StartMonitoring()
				// Ensure cleanup on shutdown
				defer func() {
					if fm := GetGlobalFailoverManager(); fm != nil {
						fm.StopMonitoring()
					}
				}()
			}
		}
	}

	// --- Declare variables needed for GUI pre-initialization ---
	var guiNodeConfig *config.Config = nil // Which config is for the GUI node
	var guiDB *LevelDB
	var guiDiscoveryMgr *DiscoveryManager
	var guiBC *BlockchainStruct
	// p2pConsensusMgr is now handled locally where needed
	var guiInitErr error
	// var chromemManager *ChromemManager // Already declared and initialized above
	var goProxy *GoReverseProxy
	var dataEngine *dataengine.DataEngine

	// Channel for synchronizing P2P manager initialization
	guiP2PReadyChan := make(chan *P2PConsensusManager, 1) // Buffered channel

	// --- Determine Node Configurations and GUI Target ---

	// --- ADD: Variable to hold the final dev config if in dev mode ---
	var finalPeerConfig *config.Config
	var actualReflectionNodeConfig *config.Config // To store the reflection config if GUI is on it

	// --- Initialize DataEngine if enabled ---
	if cfg.DataEngine.Enabled {
		log.Println("Initializing DataEngine...")

		// Parse duration strings
		windowSize, err := time.ParseDuration(cfg.DataEngine.WindowSize)
		if err != nil || windowSize <= 0 {
			windowSize = 5 * time.Minute // Default window size
		}

		metricsInterval, err := time.ParseDuration(cfg.DataEngine.MetricsInterval)
		if err != nil || metricsInterval <= 0 {
			metricsInterval = 10 * time.Second // Default metrics interval
		}

		// Create DataEngine configuration
		dataEngineConfig := dataengine.DataEngineConfig{
			KafkaBrokers:     cfg.DataEngine.KafkaBrokers,
			KafkaClientID:    cfg.DataEngine.KafkaClientID,
			ChromaDBURL:      cfg.DataEngine.ChromaDBURL,
			ChromaCollection: cfg.DataEngine.ChromaCollection,
			EnableKafka:      cfg.DataEngine.EnableKafka,
			EnableChromaDB:   cfg.DataEngine.EnableChromaDB,
			EnableWebSocket:  cfg.DataEngine.EnableWebSocket && !cfg.ReverseProxy.EmbedDataEngine,
			EnableRESTAPI:    cfg.DataEngine.EnableRESTAPI && !cfg.ReverseProxy.EmbedDataEngine,
			WebSocketPort:    cfg.DataEngine.WebSocketPort,
			RESTAPIPort:      cfg.DataEngine.RESTAPIPort,
			WindowSize:       windowSize,
			MetricsInterval:  metricsInterval,
		}

		// Create DataEngine
		dataEngine = dataengine.NewDataEngine(dataEngineConfig)

		// Start DataEngine
		err = dataEngine.Start()
		if err != nil {
			log.Printf("Warning: Failed to start DataEngine: %v", err)
		} else {
			log.Println("DataEngine started successfully")
		}
	}

	// --- Initialize InferenceEngine if enabled ---
	if cfg.InferenceEngine.Enabled {
		log.Println("Initializing InferenceEngine...")

		// Create plugins directory if it doesn't exist
		pluginsDir := cfg.InferenceEngine.PluginsDir
		if pluginsDir == "" {
			pluginsDir = "plugins"
		}
		if err := os.MkdirAll(pluginsDir, 0755); err != nil {
			log.Printf("Warning: Failed to create plugins directory %s: %v", pluginsDir, err)
		}

		// Initialize AgentInferencer
		globalAgentInferencer = agentify.NewAgentInferencer(pluginsDir)
		log.Printf("AgentInferencer initialized with plugins directory: %s", pluginsDir)

		// Initialize InferenceService with database accessor
		// We'll use the main database as the accessor
		var dbAccessor inference.DatabaseAccessor
		// For now, we'll pass nil and handle this when we have the actual database initialized
		// This will be properly connected when the database is available
		globalInferenceService, err = inference.NewInferenceService(dbAccessor)
		if err != nil {
			log.Printf("Warning: Failed to create InferenceService: %v", err)
		} else {
			log.Println("InferenceService created successfully")
		}

		// Start the inference service
		err = globalInferenceService.Start()
		if err != nil {
			log.Printf("Warning: Failed to start InferenceService: %v", err)
		} else {
			log.Println("InferenceService started successfully")
		}

		// If agent mode is enabled and DHT metrics sharing is enabled,
		// we'll integrate with DHT when the discovery manager is available
		if cfg.AgentMode.Enabled && cfg.InferenceEngine.ShareDHTMetrics {
			log.Println("Agent mode enabled: Inference engine will share metrics with DHT")
		}
	}

	// --- Initialize Go Reverse Proxy if enabled ---
	if cfg.ReverseProxy.Enabled {
		// Determine target URLs for frontend and backend
		// These will listen on localhost when the proxy is active.
		frontendTargetPort := cfg.AltGUIPort // Assuming AltGUIPort is where Next.js will run
		if frontendTargetPort == 0 {
			frontendTargetPort = 3000 // Default Next.js port
		}
		backendTargetPort := cfg.Port // The Go API port

		frontendURL := fmt.Sprintf("http://127.0.0.1:%d", frontendTargetPort)
		backendURL := fmt.Sprintf("http://127.0.0.1:%d", backendTargetPort)

		var errProxy error
		goProxy, errProxy = NewGoReverseProxy(&cfg.ReverseProxy, frontendURL, backendURL, dataEngine)
		if errProxy != nil {
			log.Fatalf("Failed to initialize Go Reverse Proxy: %v", errProxy)
		}
	}

	if cfg.IsPeer {
		// *** PEER MODE ***
		log.Println("Configuring Peer Mode...")

		// Use the configuration loaded from config.json (set during install)
		devCfg := *cfg // Start with a copy of the loaded config

		// *** CRITICAL: Use Generic Ports for Peers ***
		// The installer now sets the generic Port and P2PPort fields for devs.
		// These should be loaded directly into devCfg.Port and devCfg.P2PPort by viper_loader.
		if devCfg.Port == 0 || devCfg.P2PPort == 0 {
			log.Fatalf("FATAL: Peer ports (Port: %d, P2PPort: %d) not set or zero in config.json. Run installation or set manually.", devCfg.Port, devCfg.P2PPort)
		}
		log.Printf("Peer mode: Using generic Port (%d) and P2PPort (%d) from loaded config.", devCfg.Port, devCfg.P2PPort)

		// --- Start the Go Reverse Proxy if it was initialized ---
		if goProxy != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := goProxy.Start(); err != nil {
					log.Printf("Go Reverse Proxy failed: %v", err)
					// Optionally, call cancel() here if proxy failure is critical
				}
			}()
		}

		// *** Revised DB Path Logic for Peer ***
		finalPeerDbPath := ""
		if devCfg.SearchableDatabasePath != "" { // 1. Check specific dev path first
			finalPeerDbPath = devCfg.SearchableDatabasePath
			log.Printf("Using SearchableDatabasePath from config: %s", finalPeerDbPath)
		} else { // 2. Construct default path using helper
			if devCfg.ChainID == "" {
				log.Fatal("FATAL: ChainID is empty in config.json, cannot determine dev database path. Run installation.")
			}
			defaultPeerDbPath, err := config.GetSearchableDatabasePath(nil, devCfg.ChainID, nodeRole)
			if err != nil {
				log.Fatalf("Failed to determine default dev database path: %v", err)
			}
			finalPeerDbPath = defaultPeerDbPath
			log.Printf("SearchableDatabasePath empty, using default path: %s", finalPeerDbPath)
		}
		devCfg.SearchableDatabasePath = finalPeerDbPath // Assign the chosen path back for use

		log.Printf("Peer database path finalized: %s", devCfg.SearchableDatabasePath)

		// Ensure dev-specific settings
		devCfg.NoWalletServer = true // Peers usually don't run wallets
		devCfg.ClientOnly = false    // Peers are full participants

		// Determine if this dev node runs the GUI
		if devCfg.UseGUI {
			guiNodeConfig = &devCfg
			log.Printf("GUI enabled for Peer Node (%s)", guiNodeConfig.ChainID)
		} else {
			log.Println("GUI is disabled for Peer Node.")
		}

		// --- Store the fully prepared config ---
		finalPeerConfig = &devCfg

	} else if isNetworkMode {
		// *** NETWORK MODE (Root + Reflection) ***
		log.Println("Configuring Multi-Node Network Mode...")

		// If network mode and root mode are both enabled, disable GUI to simplify.
		// The root (root) node will run headless.
		if *root && *useGUI {
			log.Println("INFO: Network mode with --root flag. GUI will be disabled. Root node (root) and Reflection node will run headless.")
			*useGUI = false // Force GUI off
		}

		// Main node uses the potentially flag-overridden config
		mainNodeConfig := *cfg

		// Define Reflection Node Config that might run the GUI
		reflectionNodeConfig := &config.Config{
			Port:                   mainNodeConfig.Port + 1,
			P2PPort:                mainNodeConfig.P2PPort + 1,
			WalletPort:             mainNodeConfig.WalletPort + 1, // Not used, but keep consistent
			ReflectionDatabasePath: "",                            // Will be determined by path helpers
			MinersAddress:          "reflection_miner_addr",
			ReflectionURLs:         []string{fmt.Sprintf("http://127.0.0.1:%d", mainNodeConfig.Port)}, // Reflects main node
			ChainID:                fmt.Sprintf("agent-reflection-%d", mainNodeConfig.Port+1),
			ClientOnly:             false,
			UseGUI:                 mainNodeConfig.UseGUI,           // Inherit GUI flag initially
			IsRoot:                 mainNodeConfig.IsRoot,           // Inherit IsRoot flag
			PaymentProcessor:       mainNodeConfig.PaymentProcessor, // Inherit PaymentProcessor config
			NoWalletServer:         true,
		}

		// Use the new helper function for reflection DB path
		reflectionDbPath, err := config.GetReflectionDatabasePath(nodeRole)
		if err != nil {
			log.Fatalf("Failed to determine reflection database path: %v", err)
		}
		reflectionNodeConfig.ReflectionDatabasePath = reflectionDbPath

		// Determine GUI target in network mode
		if mainNodeConfig.UseGUI {
			guiNodeConfig = &mainNodeConfig                   // GUI runs on main node
			actualReflectionNodeConfig = reflectionNodeConfig // Store this specific instance
			mainNodeConfig.UseGUI = true                      // Main node runs with GUI
			reflectionNodeConfig.UseGUI = false               // Reflection node runs headless
			log.Printf("GUI enabled for Main Node (%s)", guiNodeConfig.ChainID)
		} else {
			// GUI is disabled for both
			mainNodeConfig.UseGUI = false
			reflectionNodeConfig.UseGUI = false
			log.Println("GUI is disabled for Network Mode.")
		}

		// Store the final configs for starting nodes later
		cfg = &mainNodeConfig // Update cfg to be the main node's config for starting
		// actualReflectionNodeConfig will be used if GUI is on reflection

	} else {
		// *** SINGLE NODE MODE ***
		log.Println("Configuring Single-Node Mode...")
		// Use the potentially flag-overridden config
		mainNodeConfig := *cfg
		if mainNodeConfig.UseGUI {
			guiNodeConfig = &mainNodeConfig
			log.Printf("GUI enabled for Main Node (%s)", guiNodeConfig.ChainID)
		} else {
			log.Println("GUI is disabled for Single Node.")
		}
		// cfg is already the mainNodeConfig
	}

	// If running as Peer or Bootnode and -gui flag was NOT explicitly set, force headless.
	// This ensures waitForShutdownSignal is used for these roles by default.
	if (nodeRole == config.RolePeer || nodeRole == config.RoleBootnode) && !flagsSet["gui"] {
		log.Printf("INFO: Role is %s and -gui flag not set. Forcing headless mode (guiNodeConfig will be nil).", nodeRole.String())
		guiNodeConfig = nil // Override any previous guiNodeConfig assignment
	}

	// --- Initialize Components for GUI Node (if applicable) ---
	if guiNodeConfig != nil { // If any node needs a GUI, initialize components
		log.Printf("[%s] Pre-initializing components for GUI node...", guiNodeConfig.ChainID)

		// Determine the appropriate database path based on node type
		var dbPath string
		if strings.HasPrefix(guiNodeConfig.ChainID, "agent-reflection-") {
			// For reflection nodes, use ReflectionDatabasePath
			dbPath = guiNodeConfig.ReflectionDatabasePath
			log.Printf("Using reflection database path for GUI node: '%s'", dbPath)
		} else {
			// For other nodes, use BlockchainDatabasePath
			dbPath = guiNodeConfig.BlockchainDatabasePath
			log.Printf("Using shared database path for GUI node: '%s'", dbPath)
		}

		// Ensure the specific DB directory for the GUI node exists
		if dbPath == "" {
			guiInitErr = fmt.Errorf("database path for GUI node '%s' is empty", guiNodeConfig.ChainID)
		} else {
			dbDir := filepath.Dir(dbPath)
			if err := os.MkdirAll(dbDir, os.ModePerm); err != nil {
				guiInitErr = fmt.Errorf("failed to create GUI DB dir '%s': %w", dbDir, err)
			} else {
				// Path is not empty, proceed with initialization
				log.Printf("DEBUG: Attempting to use DB path for GUI node: '%s'", dbPath)
				guiDB, guiInitErr = NewLevelDB(dbPath)
			}
		}
		if guiInitErr == nil { // Pass guiNodeConfig to NewDiscoveryManager
			guiDiscoveryMgr, guiInitErr = NewDiscoveryManager(guiNodeConfig.ChainID, int(guiNodeConfig.P2PPort), guiNodeConfig.ClientOnly, guiNodeConfig.IsBootnode, nodeRole, guiNodeConfig)
		}
		if guiInitErr == nil {
			// Get the global ChromemManager from the sync.Map
			var chromemMgr *ChromemManager
			val, ok := mainChromemManager.Load("chromemManager")
			if ok {
				chromemMgr, _ = val.(*ChromemManager)
			}

			// Pass the chromemMgr to NewBlockchain to avoid multiple instances
			guiBC, guiInitErr = NewBlockchain(trueGenesisBlock, guiNodeConfig.ChainID, guiNodeConfig.MinersAddress, guiDB, chromemMgr, guiNodeConfig.SearchableDatabasePath, guiNodeConfig.Chromem.CerebrasConfig)
			if guiInitErr == nil {
				guiBC.ChainID = guiNodeConfig.ChainID // Assign ChainID
			}
		}

		if guiInitErr != nil {
			log.Fatalf("FATAL: Failed to initialize components for GUI node %s: %v", guiNodeConfig.ChainID, guiInitErr)
			// Cleanup partially initialized components
			if guiDiscoveryMgr != nil {
				guiDiscoveryMgr.Close()
			}
			if guiDB != nil {
				guiDB.Close()
			}
			return // Exit main
		}
		log.Printf("[%s] GUI node components pre-initialized.", guiNodeConfig.ChainID)

		// The global chromemManager (initialized earlier with the initial cfg)
		// will be passed to InitializeGUI if guiNodeConfig is not nil.
		log.Printf("[%s] ChromemManager initialized for GUI node.", guiNodeConfig.ChainID)
	}

	// Custom update mechanism will be triggered by update signals from root chain
	log.Println("Custom update mechanism initialized - updates will be triggered by root chain signals")

	// --- Start Nodes Based on Mode ---
	if cfg.IsPeer {
		// *** START PEER NODE ***
		log.Println("Starting Peer Reflection Node...")

		// --- Use the finalPeerConfig prepared earlier ---
		if finalPeerConfig == nil {
			// This should not happen if the earlier logic ran correctly
			log.Fatal("FATAL: finalPeerConfig is nil when trying to start dev node.")
		}

		// Log using the correct config
		log.Printf("Starting Peer Node (HTTP: %d, P2P: %d, GUI: %t, DB: %s)",
			finalPeerConfig.Port, finalPeerConfig.P2PPort, finalPeerConfig.UseGUI, finalPeerConfig.BlockchainDatabasePath)

		// Check if this dev is the GUI node
		if guiNodeConfig != nil && guiNodeConfig.ChainID == finalPeerConfig.ChainID {
			// Start dev node with pre-initialized GUI components in goroutine
			go func() {
				// Pass the correct finalPeerConfig
				mgr, errStart := startNodeWithComponents(ctx, &wg, *finalPeerConfig, finalPeerConfig.NoWalletServer, *disableP2P, *runNetworkMode, guiDB, guiDiscoveryMgr, guiBC)
				if errStart != nil {
					log.Printf("Failed to start dev node with components: %v", errStart)
					guiP2PReadyChan <- nil // Signal failure
					cancel()               // Cancel context on failure
				} else {
					guiP2PReadyChan <- mgr // Signal success with manager
				}
			}()
		} else {
			// Start dev node without GUI components
			// Pass the correct finalPeerConfig
			startNode(ctx, &wg, *finalPeerConfig, nodeRole, finalPeerConfig.NoWalletServer, *disableP2P, *runNetworkMode)
		}

	} else if nodeRole == config.Root && *runNetworkMode {
		// *** START NETWORK MODE (Root + Reflection) ***
		log.Println("Starting in multi-node network mode...")

		// mainNodeConfig is already set to cfg (root node's config)
		rootNodeEffectiveConfig := *cfg
		rootNodeEffectiveConfig.UseGUI = *useGUI // Set GUI flag based on command line parameter

		// Start Main Node (With GUI if UseGUI is true)
		log.Printf("Starting Main Node (HTTP: %d, P2P: %d, GUI: %t, DB: %s)", // Use rootNodeEffectiveConfig for logging
			rootNodeEffectiveConfig.Port, rootNodeEffectiveConfig.P2PPort, rootNodeEffectiveConfig.UseGUI, rootNodeEffectiveConfig.BlockchainDatabasePath)
		// If GUI is enabled, start the main node with GUI components
		if rootNodeEffectiveConfig.UseGUI {
			go func() {
				mgr, errStart := startNodeWithComponents(ctx, &wg, rootNodeEffectiveConfig, rootNodeEffectiveConfig.NoWalletServer, *disableP2P, *runNetworkMode, guiDB, guiDiscoveryMgr, guiBC)
				if errStart != nil {
					log.Printf("Failed to start main node with components: %v", errStart)
					guiP2PReadyChan <- nil // Signal failure
					cancel()               // Cancel context on failure
				} else {
					guiP2PReadyChan <- mgr // Signal success with manager
				}
			}()
		} else {
			// Start without GUI
			startNode(ctx, &wg, rootNodeEffectiveConfig, nodeRole, rootNodeEffectiveConfig.NoWalletServer, *disableP2P, *runNetworkMode)
		}

		// Start Reflection Node (Potentially with GUI)
		// Use actualReflectionNodeConfig if the GUI is on it.
		// actualReflectionNodeConfig is the one prepped for GUI with correct IsRoot flags.
		if actualReflectionNodeConfig == nil { // Should not happen if -gui is passed in network mode
			// Initialize reflection node config if it's nil
			reflectionDbPath, err := config.GetReflectionDatabasePath()
			if err != nil {
				log.Printf("Warning: could not determine reflection database path: %v", err)
				reflectionDbPath = "agent_reflection.db" // Fallback to local path
			}
			actualReflectionNodeConfig = &config.Config{
				Port:                   rootNodeEffectiveConfig.Port + 1,
				P2PPort:                rootNodeEffectiveConfig.P2PPort + 1,
				WalletPort:             rootNodeEffectiveConfig.WalletPort + 1,
				ReflectionDatabasePath: reflectionDbPath,
				MinersAddress:          "reflection_miner_addr",
				ReflectionURLs:         []string{fmt.Sprintf("http://127.0.0.1:%d", rootNodeEffectiveConfig.Port)},
				ChainID:                fmt.Sprintf("agent-reflection-%d", rootNodeEffectiveConfig.Port+1),
				ClientOnly:             false,
				UseGUI:                 false, // Reflection node should always run headless now
				IsRoot:                 rootNodeEffectiveConfig.IsRoot,
				PaymentProcessor:       rootNodeEffectiveConfig.PaymentProcessor,
				NoWalletServer:         true,
			}
			log.Printf("WARNING: actualReflectionNodeConfig was nil in network mode with GUI. Created default config.")
		}
		log.Printf("Starting Reflection Node (HTTP: %d, P2P: %d, GUI: %t, DB: %s, Root: %t)",
			actualReflectionNodeConfig.Port, actualReflectionNodeConfig.P2PPort, actualReflectionNodeConfig.UseGUI, actualReflectionNodeConfig.ReflectionDatabasePath, actualReflectionNodeConfig.IsRoot)
		// Reflection node should always run headless now
		startNode(ctx, &wg, *actualReflectionNodeConfig, nodeRole, actualReflectionNodeConfig.NoWalletServer, *disableP2P, *runNetworkMode)
		log.Printf("Reflection Node started successfully (ChainID: %s)", actualReflectionNodeConfig.ChainID)
	} else {
		// *** START SINGLE NODE MODE ***
		log.Println("Starting in single-node mode...")
		mainNodeConfig := *cfg // Use the potentially flag-overridden config

		log.Printf("Starting Main Node (HTTP: %d, P2P: %d, GUI: %t, DB: %s)",
			mainNodeConfig.Port, mainNodeConfig.P2PPort, mainNodeConfig.UseGUI, mainNodeConfig.BlockchainDatabasePath)

		if guiNodeConfig != nil { // Single node mode WITH GUI
			go func() {
				mgr, errStart := startNodeWithComponents(ctx, &wg, *guiNodeConfig, guiNodeConfig.NoWalletServer, *disableP2P, *runNetworkMode, guiDB, guiDiscoveryMgr, guiBC)
				if errStart != nil {
					log.Printf("Failed to start node with components: %v", errStart)
					guiP2PReadyChan <- nil // Signal failure
					cancel()               // Cancel context on failure
				} else {
					guiP2PReadyChan <- mgr // Signal success with manager
				}
			}()
		} else { // Single node mode WITHOUT GUI
			startNode(ctx, &wg, mainNodeConfig, nodeRole, mainNodeConfig.NoWalletServer, *disableP2P, *runNetworkMode)
		}
	}

	// --- Wait for Signal ---
	// Wait for P2P Manager initialization if needed
	if guiNodeConfig != nil && !*disableP2P {
		log.Printf("[%s] Waiting for P2P Consensus Manager initialization...", guiNodeConfig.ChainID)
		select {
		case p2pMgr := <-guiP2PReadyChan:
			if p2pMgr == nil {
				log.Fatalf("FATAL: Failed to initialize P2P Consensus Manager for node %s.", guiNodeConfig.ChainID)
				// Context should already be canceled if mgr is nil
				wg.Wait()
				return
			}
			// p2pConsensusMgr is no longer needed since GUI has been removed
			log.Printf("[%s] P2P Consensus Manager initialized successfully.", guiNodeConfig.ChainID)

		case <-time.After(30 * time.Second): // Add a timeout
			log.Fatalf("FATAL: Timeout waiting for P2P Consensus Manager initialization for node %s.", guiNodeConfig.ChainID)
			cancel()
			wg.Wait()
			return
		}
	} else if guiNodeConfig != nil && *disableP2P {
		log.Printf("[%s] P2P disabled - skipping P2P Consensus Manager wait", guiNodeConfig.ChainID)
	}

	// Initialize payment processor if in root mode and enabled (moved outside the P2P check)
	if guiNodeConfig != nil {
		// Initialize payment processor if in root mode and enabled
		if guiNodeConfig.IsRoot && guiNodeConfig.PaymentProcessor.Enabled {
			var err error
			_, err = initPaymentProcessor(guiNodeConfig, guiDB, nodeRole)
			if err != nil {
				log.Printf("[%s] Warning: Failed to initialize payment processor: %v", guiNodeConfig.ChainID, err)
			} else {
				log.Printf("[%s] Payment processor initialized successfully", guiNodeConfig.ChainID)
			}
		}
	}

	// Wait for SIGINT/SIGTERM for all modes
	log.Println("Application started. Press Ctrl+C to exit.")

	// All roles will now use the graceful waitForShutdownSignal.
	waitForShutdownSignal(cancel, &wg, loadedConfigPath, cfg, wm)
	log.Println("All nodes shut down. Exiting.")

	log.Println("Exiting main KNIRVCHAIN function.")
}

// --- End of main function ---

// --- startNodeWithComponents, startNode, waitForShutdownSignal (remain unchanged) ---
// ... (rest of the functions startNodeWithComponents, startNode, waitForShutdownSignal) ...

// startNodeWithComponents is a variation that accepts pre-initialized core components
func startNodeWithComponents(
	ctx context.Context,
	wg *sync.WaitGroup,
	cfg config.Config,
	disableWallet bool,
	disableP2P bool,
	isNetworkMode bool,
	db *LevelDB, // Pre-initialized
	discoveryMgr *DiscoveryManager, // Pre-initialized
	bc *BlockchainStruct, // Pre-initialized
) (*P2PConsensusManager, error) { // Return the manager
	var p2pConsensusMgr *P2PConsensusManager

	// Create P2P Consensus Manager first (skip if disabled)
	if !disableP2P {
		var err error
		p2pConsensusMgr, err = NewP2PConsensusManager(bc, db, discoveryMgr, nodeRole)
		if err != nil {
			// Clean up already initialized components if manager fails
			// Note: Closing shared components here might be problematic if they are used elsewhere.
			// The caller (main) should handle cleanup based on where the error occurred.
			// discoveryMgr.Close() // Avoid closing shared manager here
			// db.Close() // Avoid closing shared DB here
			return nil, fmt.Errorf("[%s] failed to create P2P consensus manager: %w", cfg.ChainID, err)
		}
		bc.p2pConsensusMgr = p2pConsensusMgr // Assign to blockchain struct
	} else {
		log.Printf("[%s] P2P messaging disabled - skipping P2P consensus manager initialization", cfg.ChainID)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("[%s] Initializing node with pre-initialized components...", cfg.ChainID)
		log.Printf("[DEBUG] About to check NodeJS Services - IsRoot: %v, IsBootnode: %v, Enabled: %v", cfg.IsRoot, cfg.IsBootnode, cfg.NodeJSServices.Enabled)

		// Defer cleanup of components specific to this node's lifecycle
		defer func() {
			log.Printf("[%s] Shutting down components for node...", cfg.ChainID)
			_, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer shutdownCancel()

			// Stop blockchain operations (including mining)
			if bc != nil {
				log.Printf("[%s] Shutting down blockchain...", cfg.ChainID)
				bc.Shutdown()
			}

			// Stop P2P Consensus Manager
			if p2pConsensusMgr != nil {
				p2pConsensusMgr.Stop()
			}
			// Discovery Manager and DB are pre-initialized, assume external close handled in main's defer/cleanup.
			// Close the discovery manager if it's not nil
			// This is critical for proper shutdown in all modes
			if discoveryMgr != nil {
				log.Printf("[%s] Closing discovery manager...", cfg.ChainID)
				discoveryMgr.Close()
			}

			// DB is pre-initialized, assume external close handled in main's defer/cleanup.
			log.Printf("[%s] Node components shutdown sequence initiated.", cfg.ChainID)
		}()

		// Start P2P Consensus Manager (only if not disabled)
		if p2pConsensusMgr != nil {
			p2pConsensusMgr.Start()
		} else {
			log.Printf("[%s] P2P Consensus Manager disabled - skipping start", cfg.ChainID)
		}

		// Start Discovery Manager background tasks (if not already running)
		// Assuming Run handles multiple calls or is only called once externally if pre-initialized.
		// go discoveryMgr.Run(30 * time.Second) // Might need adjustment based on DiscoveryManager design

		// Relay (if enabled for this node)
		if cfg.Relay.Enabled && discoveryMgr.host != nil && discoveryMgr.dht != nil {
			// Use the WAN DHT and convert the config type
			relayConfig := convertRelayConfig(cfg.Relay)
			EnableRelayOnHost(ctx, discoveryMgr.host, discoveryMgr.dht.WAN, relayConfig)
		}

		// Legacy Consensus Manager (conditionally initialized)
		var consensusMgr *ConsensusManager
		if cfg.IsRoot && isNetworkMode { // isNetworkMode now passed as parameter
			selfURL := fmt.Sprintf("http://127.0.0.1:%d", cfg.Port)
			reflectURLsArray := make([]string, len(cfg.ReflectionURLs))
			copy(reflectURLsArray, cfg.ReflectionURLs)
			consensusMgr = NewConsensusManager(bc, reflectURLsArray, selfURL)
			bc.ConsensusManager = consensusMgr // Assign to blockchain struct
		}
		if consensusMgr != nil {
			log.Printf("[%s] Legacy Consensus Manager initialized.", cfg.ChainID)
			defer func() {
				log.Printf("[%s] Stopping Legacy Consensus Manager...", cfg.ChainID)
				consensusMgr.Stop()
				log.Printf("[%s] Legacy Consensus Manager stopped.", cfg.ChainID)
			}()
		}

		// Wallet Server (Optional) - Should be disabled for dev/reflection
		var walletSrv *WalletServer
		var stopWallet func()
		if !disableWallet {
			walletSrv = NewWalletServer(uint64(cfg.WalletPort), fmt.Sprintf("http://localhost:%d", cfg.Port))
			go func() {
				log.Printf("[%s] Starting Wallet Server on port %d...", cfg.ChainID, cfg.WalletPort)
				stopWallet = walletSrv.Start()
			}()
			// Wait for the actual port to be determined
			select {
			case actualPort := <-walletSrv.portChan:
				if actualPort != uint64(cfg.WalletPort) {
					log.Printf("[%s] Wallet Server is using port %d instead of configured port %d", cfg.ChainID, actualPort, cfg.WalletPort)
					// Update the config with the actual port used
					cfg.WalletPort = actualPort
				}
			case <-time.After(5 * time.Second):
				log.Printf("[%s] Warning: Timeout waiting for wallet server port signal", cfg.ChainID)
			}
			defer func() {
				if stopWallet != nil {
					log.Printf("[%s] Stopping Wallet Server...", cfg.ChainID)
					stopWallet()
					log.Printf("[%s] Wallet Server stopped.", cfg.ChainID)
				}
			}()
		}

		// 7.1. Node.js services (LEGACY - DISABLED)
		// The old Node.js services have been replaced by embedded services
		// Keeping this section commented for reference
		log.Printf("[DEBUG] NodeJS Services Config - IsRoot: %v, IsBootnode: %v, Enabled: %v", cfg.IsRoot, cfg.IsBootnode, cfg.NodeJSServices.Enabled)
		log.Printf("[DEBUG] TunnelRegistry Enabled: %v, PaymentGateway Enabled: %v", cfg.NodeJSServices.TunnelRegistry.Enabled, cfg.NodeJSServices.PaymentGateway.Enabled)
		log.Printf("[DEBUG] OperatorRegistry Enabled: %v, WebGUI Enabled: %v", cfg.NodeJSServices.OperatorRegistry.Enabled, cfg.NodeJSServices.WebGUI.Enabled)

		// 7.2. Embedded Node.js services (new embedded approach) - DISABLED
		// Node.js services have been moved to separate components
		log.Printf("[%s][%s] Node.js services moved to separate components - skipping embedded initialization", cfg.ChainID, nodeRole.String())

		// Blockchain HTTP Server
		blockchainSrv := NewBlockchainServer(uint64(cfg.Port), bc, db, discoveryMgr, int(cfg.P2PPort))

		// Prepare the server first to initialize the server field
		actualHTTPPort, err := blockchainSrv.Prepare()
		if err != nil {
			log.Printf("[%s] ERROR: Failed to prepare blockchain server: %v", cfg.ChainID, err)
			log.Fatalf("[%s] FATAL: Failed to prepare blockchain server: %v", cfg.ChainID, err)
		}

		// Update the configuration with the actual port that will be used
		if actualHTTPPort != cfg.Port {
			log.Printf("[%s] Port %d is in use, using port %d instead", cfg.ChainID, cfg.Port, actualHTTPPort)
			cfg.Port = actualHTTPPort

			// If this is the root node, save the new port to env.local
			if nodeRole == config.Root {
				rootDataDir, err := config.GetDataDir(config.Root)
				if err == nil {
					envLocalPath := filepath.Join(rootDataDir, "env.local")
					if err := utils.UpdateEnvVariable(envLocalPath, "HTTP_PORT", fmt.Sprintf("%d", actualHTTPPort)); err != nil {
						log.Printf("[%s] WARNING: Failed to update HTTP_PORT in env.local: %v", cfg.ChainID, err)
					} else {
						log.Printf("[%s] Successfully saved HTTP_PORT=%d to env.local", cfg.ChainID, actualHTTPPort)
					}
				}
			}
		}

		// Set the listen address for BlockchainServer
		if cfg.ReverseProxy.Enabled {
			blockchainSrv.server.Addr = fmt.Sprintf("127.0.0.1:%d", cfg.Port) // Listen on localhost if proxy is active
		} else {
			blockchainSrv.server.Addr = fmt.Sprintf(":%d", cfg.Port) // Listen on all interfaces otherwise
		}
		log.Printf("[%s] Blockchain Server initialized to listen on %s.", cfg.ChainID, blockchainSrv.server.Addr)

		// Save essential root node parameters to env.local if this is the root node
		if nodeRole == config.Root {
			rootDataDir, err := config.GetDataDir(config.Root)
			if err != nil {
				log.Printf("[%s] WARNING: Failed to get root data directory: %v", cfg.ChainID, err)
			} else {
				if err := saveRootNodeParameters(&cfg, rootDataDir); err != nil {
					log.Printf("[%s] WARNING: Failed to save root node parameters: %v", cfg.ChainID, err)
				}
			}
		}

		serverStopped := make(chan struct{})
		go func() {
			log.Printf("[%s] Starting Server on port %d...", cfg.ChainID, cfg.Port)
			if err := blockchainSrv.StartListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("[%s] ERROR: Blockchain HTTP Server failed: %v", cfg.ChainID, err)
			}
			log.Printf("[%s] Blockchain HTTP Server stopped.", cfg.ChainID)
			// --- Update the webgui backend.config file with the actual HTTP port ---
			webguiEnvPath := filepath.Join("..", "..", "internal", "embedded", "nodejs", "webgui", "webGUI", "backend.config") // Use filepath.Join for cross-platform compatibility

			var backendURL string
			if cfg.ReverseProxy.Enabled {
				// If reverse proxy is on, Next.js (served from root of proxy)
				// should call /api for the backend.
				backendURL = "/api"
			} else {
				// No reverse proxy, Next.js calls the Go API directly using its public IP/port
				var publicIP string
				// The public IP should have been fetched by fetchAndStorePublicIPInfo in main()
				// and stored in cfg.PublicIPInfo or LastIPInfoResponse.
				if cfg.IsRoot {
					if utils.LastIPInfoResponse != nil && utils.LastIPInfoResponse.IP != "" {
						publicIP = utils.LastIPInfoResponse.IP
					} else {
						log.Printf("[%s][%s] WARNING: Public IP for Root node not available from initial fetch. Using localhost for backendURL.", nodeRole.String(), cfg.ChainID)
						publicIP = "localhost" // Fallback
					}
				} else {
					if cfg.PublicIPInfo != nil {
						if ip, ok := cfg.PublicIPInfo["ip"].(string); ok && ip != "" {
							publicIP = ip
						} else {
							log.Printf("[%s][%s] WARNING: Public IP not found or invalid in cfg.PublicIPInfo. Using localhost for backendURL.", nodeRole.String(), cfg.ChainID)
							publicIP = "localhost" // Fallback
						}
					} else {
						log.Printf("[%s][%s] WARNING: cfg.PublicIPInfo is nil. Using localhost for backendURL.", nodeRole.String(), cfg.ChainID)
						publicIP = "localhost" // Fallback
					}
				}
				backendURL = fmt.Sprintf("http://%s:%d", publicIP, cfg.Port)
			}

			log.Printf("[%s][%s] Attempting to update NEXT_PUBLIC_BACKEND_URL in %s to %s", nodeRole.String(), cfg.ChainID, webguiEnvPath, backendURL)

			if updateErr := utils.UpdateEnvVariable(webguiEnvPath, "NEXT_PUBLIC_BACKEND_URL", backendURL); updateErr != nil {
				log.Printf("[%s][%s] WARNING: Failed to update GUI backend URL for %s: %v", nodeRole.String(), cfg.ChainID, webguiEnvPath, updateErr)
				// This is a non-fatal error, node can continue
			} else {
				log.Printf("[%s][%s] Successfully updated GUI backend URL for %s", nodeRole.String(), cfg.ChainID, webguiEnvPath)
			}
			close(serverStopped)
		}()

		defer func() {
			log.Printf("[%s] Stopping Blockchain HTTP Server...", cfg.ChainID)
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer shutdownCancel()
			if err := blockchainSrv.Stop(shutdownCtx); err != nil {
				log.Printf("[%s] WARNING: Blockchain Server shutdown error: %v", cfg.ChainID, err)
			}
			<-serverStopped
			log.Printf("[%s] Blockchain HTTP Server confirmed stopped.", cfg.ChainID)
		}()

		// Wait for Shutdown Signal
		<-ctx.Done()
		log.Printf("[%s] Node goroutine received shutdown signal. Initiating deferred cleanup...", cfg.ChainID)
	}()

	return p2pConsensusMgr, nil // Return the initialized manager
}

// startNode initializes its own components
// and starts the node with the given configuration.
func startNode(ctx context.Context, wg *sync.WaitGroup, cfg config.Config, role config.Role, disableWallet bool, disableP2P bool, isNetworkMode bool) {
	wg.Add(1)

	// Get access to the global chromemManager
	var chromemMgr *ChromemManager
	val, ok := mainChromemManager.Load("chromemManager")
	if ok {
		chromemMgr, _ = val.(*ChromemManager)
	}

	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[%s][%s] PANIC: Node goroutine panicked: %v", role.String(), cfg.ChainID, r)
			}
		}()

		log.Printf("[%s][%s] Initializing node with role %s...", role.String(), cfg.ChainID, role.String())

		// 1. Database initialization
		log.Printf("[%s][%s] Initializing database at: %s", role.String(), cfg.ChainID, cfg.BlockchainDatabasePath)
		db, err := NewLevelDB(cfg.BlockchainDatabasePath)
		if err != nil {
			log.Printf("[%s][%s] FATAL: Failed to initialize database: %v", role.String(), cfg.ChainID, err)
			return
		}
		defer func() {
			log.Printf("[%s][%s] Closing database...", role.String(), cfg.ChainID)
			if err := db.Close(); err != nil {
				log.Printf("[%s][%s] WARNING: Error closing database: %v", role.String(), cfg.ChainID, err)
			}
		}()

		// 2. Blockchain initialization
		// Pass the chromemMgr to NewBlockchain to avoid multiple instances
		bc, err := NewBlockchain(trueGenesisBlock, cfg.ChainID, cfg.MinersAddress, db, chromemMgr, cfg.SearchableDatabasePath, cfg.Chromem.CerebrasConfig)
		if err != nil {
			log.Printf("[%s][%s] ERROR: Failed to initialize blockchain: %v", role.String(), cfg.ChainID, err)
			return
		}

		// Add blockchain shutdown to defer cleanup
		defer func() {
			log.Printf("[%s][%s] Shutting down blockchain...", role.String(), cfg.ChainID)
			bc.Shutdown()
		}()
		bc.ChainID = cfg.ChainID

		// Initialize inference service with database if agent mode is enabled
		if cfg.AgentMode.Enabled && cfg.InferenceEngine.Enabled {
			log.Printf("[%s][%s] Initializing inference service with database...", role.String(), cfg.ChainID)
			if err := initializeInferenceServiceWithDB(db, &cfg); err != nil {
				log.Printf("[%s][%s] Warning: Failed to initialize inference service with database: %v", role.String(), cfg.ChainID, err)
			} else {
				log.Printf("[%s][%s] Inference service initialized with database successfully", role.String(), cfg.ChainID)
			}

			// Log agent status after initialization
			agentStatus := getAgentStatus()
			statusJSON, err := json.MarshalIndent(agentStatus, "", "  ")
			if err != nil {
				log.Printf("[%s][%s] Failed to marshal agent status: %v", role.String(), cfg.ChainID, err)
			} else {
				log.Printf("[%s][%s] Agent status:\n%s", role.String(), cfg.ChainID, string(statusJSON))
			}
		}

		// 3. Blockchain HTTP Server preparation
		blockchainSrv := NewBlockchainServer(uint64(cfg.Port), bc, db, nil, int(cfg.P2PPort))
		actualHTTPPort, err := blockchainSrv.Prepare()
		if err != nil {
			log.Printf("[%s][%s] FATAL: Failed to prepare blockchain server: %v", role.String(), cfg.ChainID, err)
			return
		}

		// Update the configuration with the actual port that will be used
		if actualHTTPPort != cfg.Port {
			log.Printf("[%s] Port %d is in use, using port %d instead", cfg.ChainID, cfg.Port, actualHTTPPort)
			cfg.Port = actualHTTPPort

			// If this is the root node, save the new port to env.local
			if role == config.Root {
				rootDataDir, err := config.GetDataDir(config.Root)
				if err == nil {
					envLocalPath := filepath.Join(rootDataDir, "env.local")
					if err := utils.UpdateEnvVariable(envLocalPath, "HTTP_PORT", fmt.Sprintf("%d", actualHTTPPort)); err != nil {
						log.Printf("[%s] WARNING: Failed to update HTTP_PORT in env.local: %v", cfg.ChainID, err)
					} else {
						log.Printf("[%s] Successfully saved HTTP_PORT=%d to env.local", cfg.ChainID, actualHTTPPort)
					}
				}
			}
		} else {
			cfg.Port = actualHTTPPort
		}

		// 4. Discovery Manager
		discoveryMgr, err := NewDiscoveryManager(cfg.ChainID, int(cfg.P2PPort), cfg.ClientOnly, cfg.IsBootnode, role, &cfg) // Pass &cfg

		if err != nil {
			log.Printf("[%s][%s] ERROR: Failed to initialize discovery manager: %v", role.String(), cfg.ChainID, err)
			return
		}
		blockchainSrv.discoveryManager = discoveryMgr
		defer func() {
			log.Printf("[%s][%s] Closing discovery manager...", role.String(), cfg.ChainID)
			discoveryMgr.Close()
		}()
		go discoveryMgr.Run(30 * time.Second)

		// Initialize DHT integration for inference engine if agent mode is enabled
		if cfg.AgentMode.Enabled && cfg.InferenceEngine.Enabled && cfg.InferenceEngine.ShareDHTMetrics {
			log.Printf("[%s][%s] Integrating inference engine with DHT for metrics sharing...", role.String(), cfg.ChainID)
			if err := integrateInferenceEngineWithDHT(discoveryMgr, &cfg); err != nil {
				log.Printf("[%s][%s] Warning: Failed to integrate inference engine with DHT: %v", role.String(), cfg.ChainID, err)
			} else {
				log.Printf("[%s][%s] Inference engine successfully integrated with DHT", role.String(), cfg.ChainID)
			}
		}

		// Relay (if enabled for this node)
		if cfg.Relay.Enabled && discoveryMgr.host != nil && discoveryMgr.dht != nil {
			// Use the WAN DHT and convert the config type
			relayConfig := convertRelayConfig(cfg.Relay)
			EnableRelayOnHost(ctx, discoveryMgr.host, discoveryMgr.dht.WAN, relayConfig)
		}

		// Legacy consensus manager
		var consensusMgr *ConsensusManager
		// Legacy Consensus Manager (only for Root node in Network mode)
		if cfg.IsRoot && isNetworkMode {
			selfURL := fmt.Sprintf("http://127.0.0.1:%d", cfg.Port)
			reflectURLsArray := make([]string, len(cfg.ReflectionURLs))
			copy(reflectURLsArray, cfg.ReflectionURLs)
			log.Printf("[%s][%s] Initializing legacy consensus manager with MinersAddress: %s", role.String(), cfg.ChainID, cfg.MinersAddress)
			log.Printf("[%s][%s] ChromemDB path: %s", role.String(), cfg.ChainID, cfg.SearchableDatabasePath)
			consensusMgr = NewConsensusManager(bc, reflectURLsArray, selfURL)
			bc.ConsensusManager = consensusMgr
		}
		if consensusMgr != nil {
			defer func() {
				log.Printf("[%s][%s] Stopping legacy consensus manager...", role.String(), cfg.ChainID)
				consensusMgr.Stop()
			}()
		}

		// P2P consensus manager (skip if disabled)
		if !disableP2P {
			p2pConsensusMgr, err := NewP2PConsensusManager(bc, db, discoveryMgr, role)
			if err != nil {
				log.Printf("[%s][%s] WARNING: Failed to initialize P2P consensus manager: %v", role.String(), cfg.ChainID, err)
			} else {
				bc.p2pConsensusMgr = p2pConsensusMgr
				log.Printf("[%s][%s] Starting P2P consensus manager with MinersAddress: %s", role.String(), cfg.ChainID, cfg.MinersAddress)
				log.Printf("[%s][%s] ChromemDB path: %s", role.String(), cfg.ChainID, cfg.SearchableDatabasePath)
				p2pConsensusMgr.Start()
				defer func() {
					log.Printf("[%s][%s] Stopping P2P consensus manager...", role.String(), cfg.ChainID)
					p2pConsensusMgr.Stop()
				}()
			}
		} else {
			log.Printf("[%s][%s] P2P messaging disabled - skipping P2P consensus manager initialization", role.String(), cfg.ChainID)
		}

		// 6. Wallet server (optional)
		var walletSrv *WalletServer
		var stopWallet func()
		if !disableWallet {
			walletSrv = NewWalletServer(uint64(cfg.WalletPort), fmt.Sprintf("http://localhost:%d", cfg.Port))
			wg.Add(1)
			go func() {
				defer wg.Done()
				log.Printf("[%s][%s] Starting wallet server on port %d...", role.String(), cfg.ChainID, cfg.WalletPort)
				stopWallet = walletSrv.Start()
				log.Printf("[%s][%s] Wallet server stopped", role.String(), cfg.ChainID)
			}()
			// Wait for the actual port to be determined
			select {
			case actualPort := <-walletSrv.portChan:
				if actualPort != uint64(cfg.WalletPort) {
					log.Printf("[%s][%s] Wallet Server is using port %d instead of configured port %d", role.String(), cfg.ChainID, actualPort, cfg.WalletPort)
					// Update the config with the actual port used
					cfg.WalletPort = actualPort
				}
			case <-time.After(5 * time.Second):
				log.Printf("[%s][%s] Warning: Timeout waiting for wallet server port signal", role.String(), cfg.ChainID)
			}
			defer func() {
				if stopWallet != nil {
					log.Printf("[%s][%s] Stopping wallet server...", role.String(), cfg.ChainID)
					stopWallet()
				}
			}()
		}

		// 7. Payment processor (root mode only)
		var paymentProcessor *PaymentProcessor
		if cfg.IsRoot && cfg.PaymentProcessor.Enabled {
			paymentProcessor, err = initPaymentProcessor(&cfg, db, role)
			if err != nil {
				log.Printf("[%s][%s] ERROR: Failed to initialize payment processor: %v", role.String(), cfg.ChainID, err)
				return
			}
			defer func() {
				log.Printf("[%s][%s] Stopping payment processor...", role.String(), cfg.ChainID)
				if err := paymentProcessor.Stop(); err != nil {
					log.Printf("[%s][%s] WARNING: Error stopping payment processor: %v", role.String(), cfg.ChainID, err)
				}
			}()
		}

		// 7.2. Economics integration service
		var economicsIntegration *EconomicsIntegration
		economicsIntegration, err = initEconomicsIntegration(&cfg)
		if err != nil {
			log.Printf("[%s][%s] ERROR: Failed to initialize economics integration: %v", role.String(), cfg.ChainID, err)
			// Continue execution even if economics integration fails to start
		}
		if economicsIntegration != nil {
			defer func() {
				log.Printf("[%s][%s] Stopping economics integration...", role.String(), cfg.ChainID)
				if economicsIntegration.IsLocalMode() {
					economicsIntegration.StopLocalEconomicsService()
				}
				log.Printf("[%s][%s] Economics integration stopped", role.String(), cfg.ChainID)
			}()
		}

		// 8. Start blockchain HTTP server
		serverStopped := make(chan struct{})
		go func() {
			log.Printf("[%s][%s] Starting blockchain HTTP server on port %d...", role.String(), cfg.ChainID, cfg.Port)
			if err := blockchainSrv.StartListenAndServe(); err != nil {
				log.Printf("[%s][%s] ERROR: Blockchain HTTP server failed: %v", role.String(), cfg.ChainID, err)
			}
			close(serverStopped)
		}()

		// 9. Save essential root node parameters to env.local if this is the root node
		if role == config.Root {
			rootDataDir, err := config.GetDataDir(config.Root)
			if err != nil {
				log.Printf("[%s][%s] WARNING: Failed to get root data directory: %v", role.String(), cfg.ChainID, err)
			} else {
				if err := saveRootNodeParameters(&cfg, rootDataDir); err != nil {
					log.Printf("[%s][%s] WARNING: Failed to save root node parameters: %v", role.String(), cfg.ChainID, err)
				}
			}
		}

		defer func() {
			log.Printf("[%s][%s] Stopping blockchain HTTP server...", role.String(), cfg.ChainID)
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := blockchainSrv.Stop(shutdownCtx); err != nil {
				log.Printf("[%s][%s] WARNING: Error stopping server: %v", role.String(), cfg.ChainID, err)
			}
			<-serverStopped
		}()

		// Wait for shutdown signal
		<-ctx.Done()
		log.Printf("[%s][%s] Received shutdown signal, cleaning up...", role.String(), cfg.ChainID)
	}()
}

// saveRootNodeParameters saves essential root node parameters to env.local
func saveRootNodeParameters(cfg *config.Config, rootDataDir string) error {
	envLocalPath := filepath.Join(rootDataDir, "env.local")
	log.Printf("Saving essential root node parameters to %s", envLocalPath)

	// Create a map of all essential parameters to save
	// Note: CHAIN_ID and MINERS_ADDRESS are constants and should not be saved in env.local
	params := map[string]string{
		"HTTP_PORT":     fmt.Sprintf("%d", cfg.Port),
		"P2P_PORT":      fmt.Sprintf("%d", cfg.P2PPort),
		"WALLET_PORT":   fmt.Sprintf("%d", cfg.WalletPort),
		"ALT_GUI_PORT":  fmt.Sprintf("%d", cfg.AltGUIPort),
		"DATABASE_PATH": cfg.BlockchainDatabasePath,
	}

	// Add IP info if available
	if utils.LastIPInfoResponse != nil && utils.LastIPInfoResponse.IP != "" {
		params["NEXT_PUBLIC_IP_INFO"] = utils.LastIPInfoResponse.IP
	}

	// Save each parameter to env.local
	for key, value := range params {
		if value != "" {
			if err := utils.UpdateEnvVariable(envLocalPath, key, value); err != nil {
				log.Printf("WARNING: Failed to update %s in env.local: %v", key, err)
				// Continue with other parameters even if one fails
			}
		}
	}

	log.Printf("Successfully saved essential root node parameters to env.local")
	return nil
}

// loadRootNodeParameters loads essential root node parameters from env.local
func loadRootNodeParameters(cfg *config.Config) error {
	rootDataDir, err := config.GetDataDir(config.Root)
	if err != nil {
		return fmt.Errorf("failed to get root data directory: %w", err)
	}

	envLocalPath := filepath.Join(rootDataDir, "env.local")
	log.Printf("Loading essential root node parameters from %s", envLocalPath)

	// Check if env.local exists
	if _, err := os.Stat(envLocalPath); os.IsNotExist(err) {
		log.Printf("env.local file not found at %s, will use default parameters", envLocalPath)
		return nil
	}

	// Read env.local file
	content, err := os.ReadFile(envLocalPath)
	if err != nil {
		return fmt.Errorf("failed to read env.local file: %w", err)
	}

	// Parse env.local file
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Skip malformed lines
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, "\"'")

		// Update config based on key
		switch key {
		case "HTTP_PORT":
			if port, err := strconv.ParseUint(value, 10, 64); err == nil {
				cfg.Port = port
				log.Printf("Loaded HTTP_PORT=%d from env.local", cfg.Port)
			}
		case "P2P_PORT":
			if port, err := strconv.ParseUint(value, 10, 64); err == nil {
				cfg.P2PPort = port
				log.Printf("Loaded P2P_PORT=%d from env.local", cfg.P2PPort)
			}
		case "WALLET_PORT":
			if port, err := strconv.ParseUint(value, 10, 64); err == nil {
				cfg.WalletPort = port
				log.Printf("Loaded WALLET_PORT=%d from env.local", cfg.WalletPort)
			}
		case "ALT_GUI_PORT":
			if port, err := strconv.ParseUint(value, 10, 64); err == nil {
				cfg.AltGUIPort = uint64(port)
				log.Printf("Loaded ALT_GUI_PORT=%d from env.local", cfg.AltGUIPort)
			}
		case "DATABASE_PATH":
			cfg.BlockchainDatabasePath = value
			log.Printf("Loaded DATABASE_PATH=%s from env.local", cfg.BlockchainDatabasePath)
			// Note: CHAIN_ID and MINERS_ADDRESS are constants and should not be loaded from env.local
		}
	}

	log.Printf("Successfully loaded root node parameters from env.local")
	return nil
}

// waitForShutdownSignal waits for SIGINT or SIGTERM and initiates shutdown
func waitForShutdownSignal(cancel context.CancelFunc, wg *sync.WaitGroup, configPath string, cfg *config.Config, wm *WalletManager) {
	// This function is now used by all roles for graceful shutdown.

	// Create a buffered channel to avoid signal loss
	signalChan := make(chan os.Signal, 2) // Buffer of 2 to handle multiple signals

	// Register for SIGINT (Ctrl+C) and SIGTERM to be sent to signalChan
	// Note: We don't call signal.Reset() here to avoid conflicts with other handlers
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Ensure we clean up signal handling when this function exits
	defer signal.Stop(signalChan)

	// Wait for first signal
	sig := <-signalChan
	log.Printf("Received signal: %s. Initiating graceful shutdown...", sig)

	// Signal all node goroutines to stop by canceling the context
	log.Println("Canceling main context to signal all components to shut down...")
	cancel()
	log.Println("Main context canceled. Waiting for components to shut down...")

	// Start a goroutine to wait for all components to shut down
	waitDone := make(chan struct{})
	go func() {
		log.Println("Waiting for all components to complete their shutdown sequences...")
		wg.Wait() // Wait for all node goroutines (incremented in startNode/startNodeWithComponents)
		log.Println("All components have completed their shutdown sequences")
		close(waitDone)
	}()

	// Set up a timeout for graceful shutdown
	shutdownTimeout := 15 * time.Second // Reduced timeout for faster response

	// Start a watchdog goroutine to log waiting goroutines if shutdown takes too long
	go func() {
		select {
		case <-waitDone:
			return // Normal shutdown, exit this goroutine
		case <-time.After(shutdownTimeout / 3):
			// If we're 1/3 through the timeout and still waiting, log more details
			log.Println("WARNING: Shutdown taking longer than expected. Waiting for remaining components...")
		}
	}()

	// Wait for either completion or timeout
	select {
	case <-waitDone:
		log.Println("All components shut down gracefully.")
	case sig := <-signalChan: // Listen on signalChan again for a second signal
		log.Printf("Received second signal: %s. Forcing immediate exit.", sig)
		log.Println("WARNING: Some resources may not be properly cleaned up.")
		os.Exit(1)
	case <-time.After(shutdownTimeout):
		log.Println("WARN: Timeout waiting for all nodes to shut down. Forcing exit.")
		log.Println("WARNING: Some resources may not be properly cleaned up.")
		os.Exit(1)
	}

	log.Println("All shutdown procedures completed.")

	// Check if this was a failover-triggered shutdown and handle promotion
	if err := HandleFailoverPromotion(configPath, cfg, wm); err != nil {
		log.Printf("Failover promotion failed: %v", err)
		log.Println("Exiting normally without promotion.")
	}

	log.Println("Exiting normally.")
}

// integrateInferenceEngineWithDHT integrates the inference engine with the DHT for sharing metrics
func integrateInferenceEngineWithDHT(discoveryMgr *DiscoveryManager, cfg *config.Config) error {
	if discoveryMgr == nil || discoveryMgr.dht == nil {
		return fmt.Errorf("discovery manager or DHT is not available")
	}

	// Create a metrics sharing key for this node
	nodeID := discoveryMgr.host.ID().String()
	metricsKey := fmt.Sprintf("/KNIRVCHAIN/inference-metrics/%s", nodeID)

	log.Printf("Setting up DHT metrics sharing with key: %s", metricsKey)

	// Start a goroutine to periodically share inference metrics via DHT
	go func() {
		ticker := time.NewTicker(30 * time.Second) // Share metrics every 30 seconds
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Get available agents from globalAgentInferencer if it's available
				var availableAgents []string
				if globalAgentInferencer != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					agents, err := globalAgentInferencer.ListAvailableAgents(ctx)
					cancel()
					if err == nil {
						availableAgents = agents
					}
				}

				// Create metrics data to share
				metricsData := map[string]interface{}{
					"node_id":           nodeID,
					"timestamp":         time.Now().Unix(),
					"inference_enabled": cfg.InferenceEngine.Enabled,
					"agent_mode":        cfg.AgentMode.Enabled,
					"plugins_dir":       cfg.InferenceEngine.PluginsDir,
					"api_port":          cfg.InferenceEngine.APIPort,
					"available_agents":  availableAgents,
					"agent_count":       len(availableAgents),
				}

				// Convert to JSON
				metricsJSON, err := json.Marshal(metricsData)
				if err != nil {
					log.Printf("Failed to marshal inference metrics: %v", err)
					continue
				}

				// Store in DHT
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				err = discoveryMgr.dht.WAN.PutValue(ctx, metricsKey, metricsJSON)
				cancel()

				if err != nil {
					log.Printf("Failed to store inference metrics in DHT: %v", err)
				} else {
					log.Printf("Successfully shared inference metrics via DHT (agents: %d)", len(availableAgents))
				}

			case <-discoveryMgr.ctx.Done():
				log.Println("Stopping inference metrics sharing due to discovery manager shutdown")
				return
			}
		}
	}()

	return nil
}

// LevelDBAdapter adapts LevelDB to implement the inference.DatabaseAccessor interface
type LevelDBAdapter struct {
	db *LevelDB
}

// GetValue implements the DatabaseAccessor interface
func (adapter *LevelDBAdapter) GetValue(key string) (string, error) {
	data, err := adapter.db.GetBytes(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SetValue implements the DatabaseAccessor interface
func (adapter *LevelDBAdapter) SetValue(key, value string) error {
	return adapter.db.PutBytes(key, []byte(value))
}

// initializeInferenceServiceWithDB initializes the inference service with the database
func initializeInferenceServiceWithDB(db *LevelDB, _ *config.Config) error {
	// Create an adapter to make LevelDB compatible with DatabaseAccessor interface
	dbAdapter := &LevelDBAdapter{db: db}

	// If we already have a global inference service, stop it first
	if globalInferenceService != nil {
		log.Println("Stopping existing inference service...")
		globalInferenceService.Stop()
	}

	// Create a new inference service with the database
	log.Println("Creating new inference service with database...")
	var err error
	globalInferenceService, err = inference.NewInferenceService(dbAdapter)
	if err != nil {
		return fmt.Errorf("failed to create inference service: %w", err)
	}

	// Start the inference service
	if err := globalInferenceService.Start(); err != nil {
		return fmt.Errorf("failed to start inference service: %w", err)
	}

	log.Println("Inference service initialized and started with database successfully")
	return nil
}

// getAgentStatus returns the current status of the agent mode components
func getAgentStatus() map[string]interface{} {
	status := map[string]interface{}{
		"agent_mode_enabled": false,
		"inference_service":  nil,
		"agent_inferencer":   nil,
		"available_agents":   []string{},
		"agent_count":        0,
	}

	// Check if agent inferencer is available
	if globalAgentInferencer != nil {
		status["agent_inferencer"] = map[string]interface{}{
			"initialized": true,
		}

		// Try to get available agents
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		agents, err := globalAgentInferencer.ListAvailableAgents(ctx)
		cancel()

		if err == nil {
			status["available_agents"] = agents
			status["agent_count"] = len(agents)
		} else {
			log.Printf("Failed to list available agents: %v", err)
		}
	}

	// Check if inference service is available
	if globalInferenceService != nil {
		status["inference_service"] = map[string]interface{}{
			"initialized": true,
		}
		status["agent_mode_enabled"] = true
	}

	return status
}
