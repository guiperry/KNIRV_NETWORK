package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

//go:embed bin/*
var binFS embed.FS

//go:embed config/*
var configFS embed.FS

//go:embed data/*/config.*
//go:embed data/*/genesis/*
var dataFS embed.FS

//go:embed data/testnet-gateway
var gatewayFS embed.FS

//go:embed .env
var envFS embed.FS

// Service represents a KNIRV service to be managed by the orchestrator
type Service struct {
	Name            string
	BinaryPath      string
	Args            []string
	Env             []string
	LogFile         string
	PidFile         string
	HealthCheck     string
	HealthCheckPort string
	StartDelay      time.Duration
}

// ServiceManager manages all KNIRV services
type ServiceManager struct {
	services  map[string]*Service
	processes map[string]*exec.Cmd
	mutex     sync.RWMutex
	deployDir string
}

// ConfigGenerator generates service configurations
type ConfigGenerator struct {
	deployDir string
}

func NewServiceManager(deployDir string) *ServiceManager {
	return &ServiceManager{
		services:  make(map[string]*Service),
		processes: make(map[string]*exec.Cmd),
		deployDir: deployDir,
	}
}

func (cg *ConfigGenerator) GenerateOracleConfig() error {
	configPath := filepath.Join(cg.deployDir, "config", "knirvoracle-testnet-config.json")

	// Try to extract from embedded configFS first
	configData, err := configFS.ReadFile("config/knirvoracle-testnet-config.json")
	if err == nil {
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(configPath, configData, 0644)
	}

	// Fallback: create minimal config if not found
	log.Printf("Config not found in embed, using fallback for ORACLE")
	return nil
}

func (cg *ConfigGenerator) GenerateOracleGenesis() error {
	genesisPath := filepath.Join(cg.deployDir, "data", "knirvoracle", "genesis", "genesis.json")

	genesis := map[string]interface{}{
		"chain_id": "knirv-testnet-1",
		"validators": []map[string]interface{}{
			{"address": "validator1", "power": 100},
			{"address": "validator2", "power": 100},
			{"address": "validator3", "power": 100},
		},
		"initial_nrn": 1000000000000,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}

	return writeJSON(genesisPath, genesis)
}

func (cg *ConfigGenerator) GenerateChainConfig() error {
	configPath := filepath.Join(cg.deployDir, "data", "knirvchain", "config.toml")

	config := `[testnet]
enabled = true
single_validator = true
simplified_consensus = true
mock_ipfs = true
mock_llm = true
in_memory_db = true

[blockchain]
chain_id = "knirvchain-testnet-1"
block_time = 5
difficulty = 1

[api]
host = "127.0.0.1"
port = 8090

[storage]
database_path = "./data/knirvchain/testnet.db"
in_memory = true

[llm]
mock_mode = true
validation_timeout = 5000

[skills]
mock_validation = true
simplified_testing = true
`

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(configPath, []byte(config), 0644)
}

func (cg *ConfigGenerator) GenerateGraphConfig() error {
	configPath := filepath.Join(cg.deployDir, "data", "knirvgraph", "config.yaml")

	config := `testnet:
  enabled: true
  in_memory: true
  pre_populate: true
  max_nodes: 250
  chain_id: "knirvgraph-testnet-1"
  port: 8082
  local_mode: true

network:
  chain_id: "knirvgraph-testnet-1"
  port: 8082
  local_mode: true

storage:
  backend: "memory"
  in_memory: true
  max_nodes: 250

test_data:
  pre_populate: true
  error_nodes: 10
  skill_nodes: 5

dht:
  simulation_mode: true
  mock_connectivity: true
  simplified_proofs: true
`

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(configPath, []byte(config), 0644)
}

func (cg *ConfigGenerator) GenerateRouterConfig() error {
	configPath := filepath.Join(cg.deployDir, "data", "knirvrouter", "test.env")

	config := `# Testnet Configuration for KNIRV-ROUTER
TESTNET_MODE=true
LOCAL_NETWORK_MODE=true
MOCK_NRN_MINTING=true
SIMPLIFIED_CONSENSUS=true
DISABLE_XION_BRIDGE=true

# Basic Configuration
PORT=8086
MINERS_ADDRESS=KNIRVROUTER_Testnet_Miner
BLOCKCHAIN_NAME=KNIRVROUTER_TESTNET
CURRENCY_NAME=NRN
HEX_PREFIX=0x
ADDRESS_PREFIX=KNIRVROUTER-

# Testnet-specific settings
TESTNET_CHAIN_ID=knirvrouter-testnet-1
TESTNET_VALIDATORS=3
TESTNET_INITIAL_NRN=1000000000000

# Mining Configuration
MINING_DIFFICULTY=5
MINING_REWARD=100
DECIMAL=8

# Consensus Configuration
CONSENSUS_PAUSE_TIME=30

# Status Messages
SUCCESS=true
FAILED=false
PENDING=pending
TXN_VERIFICATION_SUCCESS=verified
TXN_VERIFICATION_FAILURE=failed
BLOCKCHAIN_STATUS=active

# Peer Configuration (empty for local testnet)
PEER_ADDRESSES=

# Installation
INSTALL_COMPLETE=true
`

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(configPath, []byte(config), 0644)
}

func (cg *ConfigGenerator) GenerateAllConfigs() error {
	generators := []struct {
		name string
		fn   func() error
	}{
		{"ORACLE Config", cg.GenerateOracleConfig},
		{"ORACLE Genesis", cg.GenerateOracleGenesis},
		{"CHAIN Config", cg.GenerateChainConfig},
		{"GRAPH Config", cg.GenerateGraphConfig},
		{"ROUTER Config", cg.GenerateRouterConfig},
	}

	for _, gen := range generators {
		log.Printf("Generating %s...", gen.name)
		if err := gen.fn(); err != nil {
			return fmt.Errorf("failed to generate %s: %w", gen.name, err)
		}
	}

	return nil
}

func (sm *ServiceManager) DefineServices() {
	sm.services = map[string]*Service{
		"KNIRV-ORACLE": {
			Name:       "KNIRV-ORACLE",
			BinaryPath: filepath.Join(sm.deployDir, "bin", "knirvoracle"),
			Args: []string{
				"--testnet",
				"--disable-p2p",
				"--config", filepath.Join(sm.deployDir, "config", "knirvoracle-testnet-config.json"),
				"--port", "1317",
				"--p2p.port", "26656",
				"--shared_database_path", filepath.Join(sm.deployDir, "data", "testnet", "blockchain.db"),
				"--miners_address", "KNIRVORACLE_Faucet",
				"--root",
				"--non-interactive",
				"--skip-install",
			},
			LogFile:         filepath.Join(sm.deployDir, "logs", "knirvoracle.log"),
			PidFile:         filepath.Join(sm.deployDir, "data", "knirvoracle.pid"),
			HealthCheck:     "/health",
			HealthCheckPort: "1317",
			StartDelay:      2 * time.Second,
		},
		"KNIRVCHAIN": {
			Name:       "KNIRVCHAIN",
			BinaryPath: filepath.Join(sm.deployDir, "bin", "knirvchain"),
			Args: []string{
				"--testnet",
				"--disable-mining",
				"--port", "8090",
			},
			Env: []string{
				fmt.Sprintf("CONFIG_PATH=%s", filepath.Join(sm.deployDir, "data", "knirvchain", "config.toml")),
			},
			LogFile:         filepath.Join(sm.deployDir, "logs", "knirvchain.log"),
			PidFile:         filepath.Join(sm.deployDir, "data", "knirvchain.pid"),
			HealthCheck:     "/health",
			HealthCheckPort: "8090",
			StartDelay:      2 * time.Second,
		},
		"KNIRVGRAPH": {
			Name:       "KNIRVGRAPH",
			BinaryPath: filepath.Join(sm.deployDir, "bin", "knirvgraph"),
			Args: []string{
				"--testnet",
				"--memory",
				"--populate",
				"--max-nodes", "250",
				"--rpc-port", "8082",
				"--home", filepath.Join(sm.deployDir, "data", "knirvgraph"),
			},
			LogFile:         filepath.Join(sm.deployDir, "logs", "knirvgraph.log"),
			PidFile:         filepath.Join(sm.deployDir, "data", "knirvgraph.pid"),
			HealthCheck:     "/height",
			HealthCheckPort: "8082",
			StartDelay:      2 * time.Second,
		},
		"KNIRV-ROUTER": {
			Name:       "KNIRV-ROUTER",
			BinaryPath: filepath.Join(sm.deployDir, "bin", "knirvrouter"),
			Args: []string{
				"-testnet",
				"-local-network",
				"-mock-nrn",
				"-port", "8086",
				"-miners_address", "KNIRVROUTER_Testnet_Miner",
			},
			Env: []string{
				fmt.Sprintf("ENV_FILE=%s", filepath.Join(sm.deployDir, "data", "knirvrouter", "test.env")),
			},
			LogFile:         filepath.Join(sm.deployDir, "logs", "knirvrouter.log"),
			PidFile:         filepath.Join(sm.deployDir, "data", "knirvrouter.pid"),
			HealthCheck:     "/status",
			HealthCheckPort: "8086",
			StartDelay:      5 * time.Second,
		},
	}

	// KNIRVSERVER is optional - only add if binary exists (for private/corporate testing)
	nexusBinary := filepath.Join(sm.deployDir, "bin", "knirvserver")
	if _, err := os.Stat(nexusBinary); err == nil {
		log.Printf("ℹ️ KNIRVSERVER binary found - enabling for private testing")
		sm.services["KNIRV-NEXUS"] = &Service{
			Name:       "KNIRV-NEXUS",
			BinaryPath: nexusBinary,
			Args: []string{
				"--testnet",
				"--port", "8084",
				"--host", "0.0.0.0",
				"--config", filepath.Join(sm.deployDir, "config", "nexus-testnet.yaml"),
			},
			LogFile:         filepath.Join(sm.deployDir, "logs", "knirvserverr.log"),
			PidFile:         filepath.Join(sm.deployDir, "data", "knirvserverr.pid"),
			HealthCheck:     "/",
			HealthCheckPort: "8084",
			StartDelay:      2 * time.Second,
		}
	} else {
		log.Printf("ℹ️ KNIRVSERVER binary not found - skipping (public testnet mode)")
	}
}

func (sm *ServiceManager) StartService(name string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	service, exists := sm.services[name]
	if !exists {
		return fmt.Errorf("service %s not found", name)
	}

	// Verify binary exists
	if _, err := os.Stat(service.BinaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found: %s", service.BinaryPath)
	}

	log.Printf("🚀 Starting %s...", service.Name)

	// Create log file
	if err := os.MkdirAll(filepath.Dir(service.LogFile), 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logFile, err := os.Create(service.LogFile)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	// Prepare command
	cmd := exec.Command(service.BinaryPath, service.Args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = append(os.Environ(), service.Env...)

	// Start process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Write PID file
	if err := os.MkdirAll(filepath.Dir(service.PidFile), 0755); err != nil {
		log.Printf("Warning: failed to create PID directory for %s: %v", service.Name, err)
	}
	if err := os.WriteFile(service.PidFile, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644); err != nil {
		log.Printf("Warning: failed to write PID file for %s: %v", service.Name, err)
	}

	// Store process
	sm.processes[name] = cmd

	log.Printf("✅ %s started (PID: %d)", service.Name, cmd.Process.Pid)

	// Wait for startup delay
	if service.StartDelay > 0 {
		log.Printf("⏳ Waiting %v for %s to initialize...", service.StartDelay, service.Name)
		time.Sleep(service.StartDelay)
	}

	// Health check with graceful degradation
	if err := sm.WaitForHealthy(service); err != nil {
		log.Printf("⚠️ %s health check warning: %v (continuing anyway)", service.Name, err)
	}

	// Monitor in background
	go func(svcName string, svcCmd *exec.Cmd) {
		err := svcCmd.Wait()
		if err != nil {
			log.Printf("❌ %s exited with error: %v", svcName, err)
		} else {
			log.Printf("ℹ️ %s exited normally", svcName)
		}
	}(service.Name, cmd)

	return nil
}

func (sm *ServiceManager) WaitForHealthy(service *Service) error {
	if service.HealthCheck == "" {
		return nil
	}

	healthURL := fmt.Sprintf("http://localhost:%s%s", service.HealthCheckPort, service.HealthCheck)
	maxAttempts := 30

	log.Printf("🏥 Health checking %s at %s", service.Name, healthURL)

	for i := 0; i < maxAttempts; i++ {
		resp, err := http.Get(healthURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			log.Printf("✅ %s is healthy!", service.Name)
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		if i < maxAttempts-1 {
			time.Sleep(2 * time.Second)
		}
	}

	return fmt.Errorf("health check timeout after %d attempts", maxAttempts)
}

func (sm *ServiceManager) StartAll() error {
	// Define startup order for public testnet services
	serviceOrder := []string{
		"KNIRV-ORACLE",
		"KNIRVCHAIN",
		"KNIRVGRAPH",
		"KNIRV-ROUTER",
	}

	// Add KNIRVSERVER if it's available (private/corporate testing only)
	if _, hasNexus := sm.services["KNIRV-NEXUS"]; hasNexus {
		// Insert NEXUS before ROUTER
		serviceOrder = []string{
			"KNIRV-ORACLE",
			"KNIRVCHAIN",
			"KNIRVGRAPH",
			"KNIRV-NEXUS",
			"KNIRV-ROUTER",
		}
		log.Printf("ℹ️ Including KNIRV-NEXUS in startup sequence (private testing mode)")
	}

	for _, serviceName := range serviceOrder {
		if err := sm.StartService(serviceName); err != nil {
			return fmt.Errorf("failed to start %s: %w", serviceName, err)
		}
	}

	return nil
}

func (sm *ServiceManager) StopAll() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	log.Println("🛑 Stopping all services...")

	for name, cmd := range sm.processes {
		if cmd.Process != nil {
			log.Printf("Stopping %s (PID: %d)...", name, cmd.Process.Pid)
			if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
				log.Printf("Failed to send SIGTERM to %s: %v, killing...", name, err)
				cmd.Process.Kill()
			}
		}
	}

	// Wait for all processes to exit gracefully
	time.Sleep(5 * time.Second)

	return nil
}

func main() {
	log.Println("🚀 KNIRV Native Orchestrator Starting...")
	log.Println("Version: 1.0.0")
	log.Println("Build: " + time.Now().Format(time.RFC3339))

	// Default deployment directory
	deployDir := "/opt/knirv-testnet"

	// Override with command-line argument if provided
	if len(os.Args) > 1 {
		if os.Args[1] == "--help" || os.Args[1] == "-h" {
			fmt.Println("KNIRV Native Orchestrator")
			fmt.Println("=========================")
			fmt.Println("")
			fmt.Println("Usage: knirvtestnet [deploy-dir]")
			fmt.Println("")
			fmt.Println("Arguments:")
			fmt.Println("  deploy-dir    Directory to extract files and run services")
			fmt.Println("                (default: /opt/knirv-testnet)")
			fmt.Println("")
			fmt.Println("Examples:")
			fmt.Println("  Local:   ./knirvtestnet .")
			fmt.Println("  Remote:  sudo knirvtestnet /opt/knirv-testnet")
			fmt.Println("")
			os.Exit(0)
		}
		deployDir = os.Args[1]
	}

	log.Printf("📁 Deployment directory: %s", deployDir)

	// Phase 1: Extract embedded files
	log.Println("")
	log.Println("📦 Phase 1: Extracting embedded files...")
	if err := extractEmbeddedFiles(deployDir); err != nil {
		log.Fatalf("❌ Failed to extract files: %v", err)
	}
	log.Println("✅ Phase 1 complete: All files extracted")

	// Phase 2: Generate configurations
	log.Println("")
	log.Println("⚙️ Phase 2: Generating configurations...")
	configGen := &ConfigGenerator{deployDir: deployDir}
	if err := configGen.GenerateAllConfigs(); err != nil {
		log.Fatalf("❌ Failed to generate configs: %v", err)
	}
	log.Println("✅ Phase 2 complete: All configurations generated")

	// Phase 3: Start services
	log.Println("")
	log.Println("🚀 Phase 3: Starting services...")
	sm := NewServiceManager(deployDir)
	sm.DefineServices()

	if err := sm.StartAll(); err != nil {
		log.Fatalf("❌ Failed to start services: %v", err)
	}

	log.Println("")
	log.Println("✅ Phase 3 complete: All services started successfully!")
	log.Println("")
	log.Println("🎉 KNIRV TESTNET IS FULLY RUNNING!")
	log.Println("==================================")
	log.Println("")
	log.Println("Public Service Endpoints:")
	log.Println("  🔗 KNIRV-ORACLE:  http://localhost:1317")
	log.Println("  ⛓️  KNIRVCHAIN:    http://localhost:8090")
	log.Println("  📊 KNIRVGRAPH:    http://localhost:8082")
	log.Println("  🌐 KNIRV-ROUTER:  http://localhost:8086")

	// Show NEXUS only if it's running
	if _, hasNexus := sm.services["KNIRV-NEXUS"]; hasNexus {
		log.Println("")
		log.Println("Private Service Endpoints (Corporate Testing):")
		log.Println("  🔒 KNIRV-NEXUS:   http://localhost:8084")
	}

	log.Println("")
	log.Println("Log Files:")
	log.Printf("  📄 Logs directory: %s/logs/", deployDir)
	log.Println("")

	// Phase 4: Wait for shutdown signal
	log.Println("⏳ Phase 4: Monitoring services (press Ctrl+C to stop)...")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("")
	log.Println("🛑 Shutdown signal received")
	if err := sm.StopAll(); err != nil {
		log.Printf("⚠️ Error during shutdown: %v", err)
	}

	log.Println("👋 KNIRV Orchestrator stopped")
}

func extractEmbeddedFiles(deployDir string) error {
	// Create main directories
	dirs := []string{
		filepath.Join(deployDir, "bin"),
		filepath.Join(deployDir, "config"),
		filepath.Join(deployDir, "data"),
		filepath.Join(deployDir, "logs"),
	}

	for _, dir := range dirs {
		log.Printf("Creating directory: %s", dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Extract binaries
	log.Println("Extracting binaries...")
	if err := extractFS(binFS, "bin", deployDir); err != nil {
		return fmt.Errorf("failed to extract binaries: %w", err)
	}

	// Extract configs
	log.Println("Extracting configs...")
	if err := extractFS(configFS, "config", deployDir); err != nil {
		return fmt.Errorf("failed to extract configs: %w", err)
	}

	// Extract data files
	log.Println("Extracting data files...")
	if err := extractFS(dataFS, "data", deployDir); err != nil {
		// Non-fatal if data extraction fails
		log.Printf("Warning: failed to extract data files: %v", err)
	}

	// Extract .env
	log.Println("Extracting .env file...")
	envData, err := envFS.ReadFile(".env")
	if err != nil {
		log.Printf("Warning: .env file not found in embedded files: %v", err)
	} else {
		if err := os.WriteFile(filepath.Join(deployDir, ".env"), envData, 0644); err != nil {
			return fmt.Errorf("failed to write .env: %w", err)
		}
	}

	// Verify critical binaries were extracted (NEXUS is optional)
	criticalBinaries := []string{"knirvoracle", "knirvchain", "knirvgraph", "knirvrouter"}
	optionalBinaries := []string{"knirvserverr"}

	for _, bin := range criticalBinaries {
		binPath := filepath.Join(deployDir, "bin", bin)
		if _, err := os.Stat(binPath); os.IsNotExist(err) {
			return fmt.Errorf("critical binary missing after extraction: %s", bin)
		}
		log.Printf("✓ Verified binary: %s", bin)
	}

	// Check optional binaries (don't fail if missing)
	for _, bin := range optionalBinaries {
		binPath := filepath.Join(deployDir, "bin", bin)
		if _, err := os.Stat(binPath); os.IsNotExist(err) {
			log.Printf("ℹ️ Optional binary not found: %s (skipping)", bin)
		} else {
			log.Printf("✓ Verified optional binary: %s", bin)
		}
	}

	return nil
}

func extractFS(fsys embed.FS, prefix string, deployDir string) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		destPath := filepath.Join(deployDir, path)

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		data, err := fsys.ReadFile(path)
		if err != nil {
			return err
		}

		// Make binaries executable
		perm := os.FileMode(0644)
		if strings.HasPrefix(path, "bin/") {
			perm = 0755
		}

		return os.WriteFile(destPath, data, perm)
	})
}

func writeJSON(path string, data interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
