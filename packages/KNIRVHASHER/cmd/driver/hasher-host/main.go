// Hasher: Neural Inference Engine Powered by SHA-256 ASICs
// Copyright (C) 2026  Guillermo Perry
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"knirvhasher/internal/config"
	"knirvhasher/internal/discovery"
	devicepkg "knirvhasher/internal/driver/device"
	"knirvhasher/internal/host"
	"knirvhasher/internal/formal/lean"
	hasherapi "knirvhasher/pkg/hashing/api"
	pb "knirvhasher/internal/proto/hasher/v1"
	"knirvhasher/pkg/hashing/proofasset"
	"knirvhasher/pkg/hashing/inference"
	jitterpkg "knirvhasher/pkg/hashing/jitter"
	hashermath "knirvhasher/pkg/hashing/math"
	"knirvhasher/pkg/hashing/methods/asic"
	"knirvhasher/pkg/hashing/neural"
	"knirvhasher/pkg/hashing/tokenizer"
	"knirvhasher/pkg/hashing/transformer"
)

const (
	portFile    = "/tmp/hasher-host.port"
	socketFile  = "/tmp/hasher-host-grpc.sock"
	socketDir   = "/tmp/knirvhasher"
)

// writePortFile writes the port to a temporary file for the CLI to discover.
func writePortFile(port int) error {
	log.Printf("Writing port %d to %s", port, portFile)
	return os.WriteFile(portFile, []byte(fmt.Sprintf("%d", port)), 0644)
}

// cleanupPortFile removes the temporary port file.
func cleanupPortFile() {
	log.Printf("Cleaning up port file: %s", portFile)
	os.Remove(portFile)
}

// writeSocketFile writes the Unix socket path to a temporary file for discovery.
func writeSocketFile(socketPath string) error {
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return fmt.Errorf("create socket directory: %w", err)
	}
	log.Printf("Writing socket path %s to %s", socketPath, socketFile)
	return os.WriteFile(socketFile, []byte(socketPath), 0644)
}

// cleanupSocketFile removes the temporary socket file and socket itself.
func cleanupSocketFile(socketPath string) {
	if socketPath != "" {
		log.Printf("Cleaning up socket file: %s", socketPath)
		os.Remove(socketPath)
	}
	log.Printf("Cleaning up socket path file: %s", socketFile)
	os.Remove(socketFile)
}

func init() {
	if deviceStr := config.GetDeviceIP(); deviceStr != "" && *device == "" {
		*device = deviceStr
	}
	if passwordStr := config.GetDevicePassword(); passwordStr != "" && *serverSSHPassword == "" {
		*serverSSHPassword = passwordStr
	}
}

// Configuration flags
var (
	// Server configuration
	port      = flag.Int("port", 0, "HTTP API server port (0 = auto-find open port)")
	device    = flag.String("device", "", "ASIC device IP (empty = auto-discover/deploy). Deploys hasher-server first. Ex: 192.168.12.151")
	asicAddr  = flag.String("asic-addr", "", "hasher-server gRPC address (empty = auto-discover). Deploys hasher-server first. Ex: 192.168.12.151:8888")
	enableAPI = flag.Bool("api", true, "enable REST API server")

	// CLI modes (for direct command-line testing)
	mode      = flag.String("mode", "api", "operation mode: api, single, batch, stream, metrics, info")
	count     = flag.Int("count", 10, "number of hashes to compute (CLI modes)")
	batchSize = flag.Int("batch", 4, "batch size for batch mode")
	dataSize  = flag.Int("size", 64, "size of data to hash")

	// Inference configuration
	passes       = flag.Int("passes", 21, "number of temporal ensemble passes")
	jitter       = flag.Float64("jitter", 0.01, "input jitter factor [0, 1]")
	seedRotation = flag.Bool("seed-rotation", true, "enable seed rotation per pass")

	// Network architecture
	inputSize  = flag.Int("input-size", 784, "network input dimension")
	hidden1    = flag.Int("hidden1", 128, "hidden layer 1 size")
	hidden2    = flag.Int("hidden2", 64, "hidden layer 2 size")
	outputSize = flag.Int("output-size", 10, "network output dimension")

	// Crypto-transformer configuration
	enableCrypto     = flag.Bool("crypto", true, "enable crypto-transformer")
	vocabSize        = flag.Int("vocab-size", 100256, "transformer vocabulary size")
	embedDim         = flag.Int("embed-dim", 256, "transformer embedding dimension")
	numLayers        = flag.Int("num-layers", 4, "transformer number of layers")
	numHeads         = flag.Int("num-heads", 8, "transformer attention heads")
	ffnHiddenDim     = flag.Int("ffn-hidden", 512, "transformer feed-forward hidden dim")
	cryptoActivation = flag.String("crypto-activation", "hash", "transformer activation: hash, tanh, sigmoid")

	// Network discovery configuration
	discoverNetwork  = flag.Bool("discover", true, "enable network discovery for hasher-server")
	discoverySubnet  = flag.String("subnet", "", "network subnet to scan (CIDR, empty = auto-detect)")
	discoveryPort    = flag.Int("discovery-port", 80, "port to scan for hasher-server")
	discoveryTimeout = flag.Duration("discovery-timeout", 2*time.Second, "timeout for each server probe")
	skipLocalhost    = flag.Bool("skip-localhost", false, "skip localhost during discovery")

	// Auto-deployment configuration
	autoDeploy    = flag.Bool("auto-deploy", true, "automatically deploy hasher-server to ASIC devices")
	cleanupOnExit = flag.Bool("cleanup", true, "clean up deployed hasher-server on exit")
	forceRedeploy = flag.Bool("force-redeploy", false, "force redeployment of hasher-server even if it's already running")

	// Server log monitoring configuration
	monitorServerLogs = flag.Bool("monitor-server-logs", true, "enable automatic monitoring of server logs for auto-recovery")
	serverLogPath     = flag.String("server-log", "/tmp/hasher-server.log", "path to server log file on ASIC device")
	serverDeviceIP    = flag.String("server-device-ip", "", "IP address of ASIC device (for log monitoring, auto-detected if empty)")
	serverSSHPassword = flag.String("server-ssh-password", "", "SSH password for ASIC device (for log monitoring). Defaults to DEVICE_PASSWORD env var or .env file")

	// Knowledge base
	framesDir = flag.String("frames-dir", "", "directory of training frame JSON files to load into the FlashSearcher at startup (e.g. /data/hasher/frames)")

	// Direct ASIC mode: skip CGMiner and drive the ASIC directly over raw USB
	directMode = flag.Bool("direct", false, "skip CGMiner and drive the ASIC directly over raw USB (no stratum pool, no CGMiner); mutually exclusive in effect with CGMiner mode")

	// CGMiner API (direct ASIC integration)
	cgminerHost    = flag.String("cgminer-host", "", "CGMiner API host (e.g. 192.168.12.151). If set, the ASIC mines the final golden nonce for each inference token")
	cgminerPort    = flag.Int("cgminer-port", 4028, "CGMiner API port (default 4028)")
	cgminerTimeout = flag.Duration("cgminer-timeout", 10*time.Second, "timeout per CGMiner mining request")

	// Schema-driven domain selection: makes hasher-host behave as HASHER or MATHASHER
	// at runtime depending on which schema YAML is loaded.
	schemaPath = flag.String("schema", "", "Path to slot-schema YAML (e.g. pipeline/2_DATA_ENCODER/config/math_schema.yaml). When domain is MATHASHER, mounts POST /v1/verify/math on the API server.")
	subdomain  = flag.Uint("subdomain", 0, "Math subdomain override when --schema names MATHASHER: 0=Arithmetic 1=Algebra 2=Calculus 3=Statistics 4=Logic")

	// Formal verification configuration
	formalVerifierEnabled = flag.Bool("formal-verifier", false, "enable formal proof verification via Lean worker")
	leanBinary            = flag.String("lean-binary", "lean", "path to Lean binary for formal verification")
	leanWorkDir           = flag.String("lean-work-dir", "", "working directory for Lean verification jobs (default: system temp dir)")
	leanMaxSeconds        = flag.Int("lean-max-seconds", 15, "max CPU seconds per Lean verification job")
	proofLedgerPath       = flag.String("proof-ledger", "", "path to proof_writes.jsonl (default: <data-path>/frames/proof_writes.jsonl)")
	grpcPort              = flag.Int("grpc-port", 50051, "gRPC server port for FormalVerificationService")
	grpcSocket            = flag.String("grpc-socket", "", "Unix socket path for FormalVerificationService gRPC (default: empty = TCP when not under KNIRVSERVER; /tmp/knirvhasher/formal-verifier.sock when --knirvserver-deployed)")
	knirvserverDeployed   = flag.Bool("knirvserver-deployed", false, "indicates KNIRVHASHER is initialized by KNIRVSERVER; defaults gRPC to Unix socket and disables direct port binding")
)

// Orchestrator manages the recursive inference process
type Orchestrator struct {
	asicClient      *asic.ASICClient
	engine          *inference.RecursiveEngine
	network         *neural.HashNetwork
	cryptoModel     *transformer.HasherTransformer
	miningNeuron    *neural.MiningNeuron         // Mining neuron for proof-of-work attestation
	jitterEngine    *jitterpkg.JitterEngine      // Core inference engine: 21-pass associative loop
	cgminerMethod   *devicepkg.CGMinerHashMethod // CGMiner ASIC for final golden-nonce mining
	directMode      bool                        // Direct ASIC mode (skip CGMiner, drive via raw USB)
	discoveryResult *discovery.DiscoveryResult
	startTime       time.Time
	mu              sync.RWMutex
	deployer        *host.Deployer // For auto-deployment

	// Domain-specific plugin: non-nil when --schema names a MATHASHER schema.
	// Routes POST /v1/verify/math and GET /v1/verify/math/health are mounted
	// on the gin router only when this is set.
	mathServer *hasherapi.MathVerifierServer

	// Formal verification: attached when --formal-verifier is enabled.
	formalVerifierEnabled bool
	leanWorker            interface{} // *lean.Worker — kept as interface{} to avoid import cycle; cast in runAPIServer
	proofLedgerPath       string
	formalVerifierGRPCServer *lean.GRPCServer
	grpcServer            *grpc.Server
	grpcPort              int
	grpcSocket            string
	knirvserverDeployed   bool

	// Connection monitoring
	connectionHealthy bool          // Current connection health status
	lastHealthCheck   time.Time     // Last successful health check
	reconnectAttempts int           // Number of reconnection attempts since last success
	stopMonitor       chan struct{} // Signal to stop the connection monitor

	// Server log monitoring
	stopLogMonitorChan chan struct{} // Signal to stop the log monitor
	serverDeviceIPAddr string        // IP of the ASIC device running server
	isRebooting        bool          // Flag indicating if we're handling a server reboot

	// Metrics
	totalInferences  uint64
	totalLatencyNs   uint64
	successfulInfers uint64
	failedInfers     uint64
}

// InferRequest is the API request for inference
type InferRequest struct {
	Data string `json:"data"` // Base64-encoded input data
}

// InferResponse is the API response for inference
type InferResponse struct {
	Prediction        int     `json:"prediction"`
	Confidence        float64 `json:"confidence"`
	AverageConfidence float64 `json:"average_confidence"`
	Passes            int     `json:"passes"`
	ValidPasses       int     `json:"valid_passes"`
	LatencyMs         float64 `json:"latency_ms"`
	UsingASIC         bool    `json:"using_asic"`
}

// HealthResponse is the API response for health check
type HealthResponse struct {
	Status            string `json:"status"`
	UsingASIC         bool   `json:"using_asic"`
	ChipCount         int    `json:"chip_count"`
	Uptime            string `json:"uptime"`
	ConnectionHealthy bool   `json:"connection_healthy"`
	LastHealthCheck   string `json:"last_health_check,omitempty"`
	DirectMode        bool   `json:"direct_mode"`
}

// MetricsResponse is the API response for metrics
type MetricsResponse struct {
	TotalInferences  uint64  `json:"total_inferences"`
	SuccessfulInfers uint64  `json:"successful_inferences"`
	FailedInfers     uint64  `json:"failed_inferences"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
	UsingASIC        bool    `json:"using_asic"`
	ChipCount        int     `json:"chip_count"`
	Uptime           string  `json:"uptime"`
}

// BatchInferRequest is the API request for batch inference
type BatchInferRequest struct {
	Data []string `json:"data"` // Array of base64-encoded inputs
}

// BatchInferResponse is the API response for batch inference
type BatchInferResponse struct {
	Results   []InferResponse `json:"results"`
	TotalMs   float64         `json:"total_ms"`
	UsingASIC bool            `json:"using_asic"`
}

// ChatRequest is the API request for crypto-transformer chat
type ChatRequest struct {
	Message     string  `json:"message"`
	Context     []int   `json:"context,omitempty"`
	Temperature float32 `json:"temperature,omitempty"`
}

// ChatResponse is the API response for crypto-transformer chat
type ChatResponse struct {
	Response   string  `json:"response"`
	TokenID    int     `json:"token_id"`
	Confidence float32 `json:"confidence"`
	LatencyMs  float64 `json:"latency_ms"`
	UsingASIC  bool    `json:"using_asic"`
}

// RuleRequest represents a request to add a logical rule
type RuleRequest struct {
	Domain      string   `json:"domain"`
	RuleType    string   `json:"rule_type"` // 'subsumption', 'disjoint', 'constraint'
	Premises    []string `json:"premises"`
	Conclusion  string   `json:"conclusion"`
	Description string   `json:"description"`
}

// RuleResponse represents a logical rule response
type RuleResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	RuleID  int    `json:"rule_id,omitempty"`
}

func main() {
	flag.Parse()

	log.Printf("Hasher Host Orchestrator starting...")

	// Initialize deployment module if auto-deployment is enabled
	if *directMode {
		log.Printf("Direct ASIC mode: %v", *directMode)
	}

	var deployer *host.Deployer
	if *autoDeploy {
		deployConfig := &host.DeploymentConfig{
			AutoDeploy:     *autoDeploy,
			CleanupOnExit:  *cleanupOnExit,
			ConnectTimeout: 30 * time.Second,
			DeployTimeout:  120 * time.Second,
			ForceRedeploy:  *forceRedeploy,
			DirectMode:     *directMode,
		}
		var err error
		deployer, err = host.NewDeployer(deployConfig)
		if err != nil {
			log.Printf("Warning: Failed to create deployer: %v", err)
		}
	}

	// Discover and connect to ASIC server
	var asicClient *asic.ASICClient
	var discoveryResult *discovery.DiscoveryResult
	var serverDeviceAddr string // Store the server device IP

	if *device != "" {
		// Use explicit device IP - deployment is required first
		log.Printf("Device specified: %s - deploying hasher-server first", *device)
		serverDeviceAddr = fmt.Sprintf("%s:8888", *device)

		// Deployment is mandatory - create deployer if not already created
		if deployer == nil {
			deployConfig := &host.DeploymentConfig{
				AutoDeploy:     true,
				CleanupOnExit:  *cleanupOnExit,
				ConnectTimeout: 30 * time.Second,
				DeployTimeout:  120 * time.Second,
				DirectMode:     *directMode,
			}
			var err error
			deployer, err = host.NewDeployer(deployConfig)
			if err != nil {
				log.Printf("Warning: Failed to create deployer for device %s: %v", *device, err)
			}
		}

		// Deploy hasher-server to the specified device and get connected client
		log.Printf("Deploying hasher-server to device %s...", *device)
		deployedClient, deployErr := deployer.EnsureServerDeployed(*device)
		if deployErr != nil {
			log.Printf("Warning: Failed to deploy hasher-server to device %s: %v", *device, deployErr)
			log.Printf("Falling back to software mode...")
			asicClient = nil
		} else {
			// Use the connected client from deployment (deployment already SSH'd to port 22, deployed, and connected to port 8888)
			asicClient = deployedClient
			log.Printf("Successfully deployed and connected to hasher-server on device %s", *device)
		}

	} else if *asicAddr != "" {
		// Extract IP from asicAddr for deployment
		deviceIP := extractIPFromAddress(*asicAddr)
		if deviceIP == "" {
			log.Printf("Warning: Invalid ASIC server address: %s - cannot extract device IP for deployment", *asicAddr)
			log.Printf("Falling back to software mode...")
			asicClient = nil
		} else {
			log.Printf("ASIC server specified: %s - deploying hasher-server to device %s first", *asicAddr, deviceIP)
			serverDeviceAddr = *asicAddr

			// Deployment is mandatory - create deployer if not already created
			if deployer == nil {
					deployConfig := &host.DeploymentConfig{
					AutoDeploy:     true,
					CleanupOnExit:  *cleanupOnExit,
					ConnectTimeout: 30 * time.Second,
					DeployTimeout:  120 * time.Second,
					DirectMode:     *directMode,
				}
				var createErr error
				deployer, createErr = host.NewDeployer(deployConfig)
				if createErr != nil {
					log.Printf("Warning: Failed to create deployer for device %s: %v", deviceIP, createErr)
				}
			}

			// Deploy hasher-server to device and get connected client
			log.Printf("Deploying hasher-server to device %s...", deviceIP)
			deployedClient, deployErr := deployer.EnsureServerDeployed(deviceIP)
			if deployErr != nil {
				log.Printf("Warning: Failed to deploy hasher-server to device %s: %v", deviceIP, deployErr)
				log.Printf("Falling back to software mode...")
				asicClient = nil
			} else {
				// Use the connected client from deployment (deployment already SSH'd to port 22, deployed, and connected to port 8888)
				asicClient = deployedClient
				log.Printf("Successfully deployed and connected to hasher-server on device %s", deviceIP)
			}
		}
	} else if *discoverNetwork {
		// Perform network discovery with auto-deployment
		log.Printf("Discovering hasher-server instances on network...")

		if deployer != nil {
			// Use auto-deployment with discovery
			log.Printf("Auto-deployment enabled, looking for devices to deploy hasher-server...")
			var err error
			asicClient, err = deployer.DeployWithDiscovery()
			if err != nil {
				log.Printf("Warning: Auto-deployment failed: %v", err)
				log.Printf("Falling back to localhost discovery...")

				// Quick localhost check before full network discovery
				asicClient, discoveryResult, err = discovery.DiscoverAndConnect(discovery.DiscoveryConfig{
					Port:          *discoveryPort,
					Timeout:       5 * time.Second, // Short timeout for localhost
					Subnet:        "127.0.0.0/30",  // Very limited range
					SkipLocalhost: false,
				})
				if err != nil {
					log.Printf("Warning: Localhost discovery failed: %v", err)
					log.Printf("Creating ASIC client for direct connection attempt...")
					var clientErr error
					asicClient, clientErr = asic.NewASICClient("") // Create client for direct connection attempt
					if clientErr != nil {
						log.Printf("Warning: ASIC client creation failed: %v", clientErr)
						asicClient = nil
					}
				} else {
					log.Printf("Connected to discovered hasher-server at %s", discoveryResult.Address)
					log.Printf("Server info: %d chips, %s, latency: %dms",
						discoveryResult.ChipCount, discoveryResult.Version, discoveryResult.LatencyMs)
					serverDeviceAddr = discoveryResult.Address
				}
			} else {
				log.Printf("Auto-deployment successful: hasher-server running on discovered device")
				// GetDeployedDevice returns the bare device IP, not IP:port -
				// unlike every other assignment to serverDeviceAddr in this
				// function, which all include the :8888 gRPC port. Without
				// it, the later ASICMethod dial (which uses serverDeviceAddr
				// verbatim) fails with "context deadline exceeded" trying to
				// connect to a portless address.
				serverDeviceAddr = fmt.Sprintf("%s:8888", deployer.GetDeployedDevice())
			}
		} else {
			// Quick localhost discovery only when no deployer
			log.Printf("Checking for local hasher-server...")
			config := discovery.DiscoveryConfig{
				Port:          *discoveryPort,
				Timeout:       5 * time.Second, // Short timeout for localhost
				Subnet:        "127.0.0.0/30",  // Very limited range
				SkipLocalhost: false,
			}
			if *discoverySubnet != "" {
				config.Subnet = *discoverySubnet
			}

			var err error
			asicClient, discoveryResult, err = discovery.DiscoverAndConnect(config)
			if err != nil {
				log.Printf("Warning: Localhost discovery failed: %v", err)
				log.Printf("Creating ASIC client for direct connection attempt...")
				var clientErr error
				asicClient, clientErr = asic.NewASICClient("") // Create client for direct connection attempt
				if clientErr != nil {
					log.Printf("Warning: ASIC client creation failed: %v", clientErr)
					asicClient = nil
				}
			} else {
				log.Printf("Connected to discovered hasher-server at %s", discoveryResult.Address)
				log.Printf("Server info: %d chips, %s, latency: %dms",
					discoveryResult.ChipCount, discoveryResult.Version, discoveryResult.LatencyMs)
				serverDeviceAddr = discoveryResult.Address
			}
		}
	} else {
		// Try localhost only
		log.Printf("Trying localhost hasher-server...")
		serverDeviceAddr = "localhost:8888"
		var err error
		asicClient, err = asic.NewASICClient("localhost:8888")
		if err != nil {
			log.Printf("Warning: Could not connect to localhost hasher-server: %v", err)
			asicClient = nil
		}
	}

	// Default to software fallback if no ASIC device is available or specified
	if asicClient == nil {
		log.Printf("No ASIC device available - enabling software fallback mode")
		// Create a fallback ASIC client that will use software SHA-256
		var clientErr error
		asicClient, clientErr = asic.NewASICClient("")
		if clientErr != nil {
			log.Printf("Warning: Failed to create software fallback client: %v", clientErr)
			asicClient = nil
		} else {
			log.Printf("Software fallback mode initialized successfully")
		}
	}

	// Override with explicit flags if provided.
	//
	// IMPORTANT: serverDeviceAddr is a host:port gRPC dial target (used
	// directly by asic.NewASICMethod below) everywhere else in this
	// function - every other assignment to it appends :8888. These two
	// branches used to assign the bare flag value with no port, silently
	// undoing whatever correct host:port the deployment logic above had
	// just set (--device is set on essentially every real run, so this
	// ran unconditionally and always won) - producing exactly the
	// "context deadline exceeded" dialing a portless address that this
	// comment used to not warn about.
	if *serverDeviceIP != "" {
		serverDeviceAddr = fmt.Sprintf("%s:8888", *serverDeviceIP)
	} else if *device != "" {
		// Device flag takes precedence for IP extraction
		serverDeviceAddr = fmt.Sprintf("%s:8888", *device)
	}

	if asicClient != nil {
		if asicClient.IsUsingFallback() {
			log.Printf("Running in SOFTWARE FALLBACK mode (ASIC server not available)")
		} else {
			chipCount := asicClient.GetChipCount()
			if discoveryResult != nil {
				log.Printf("Connected to ASIC server with %d chips at %s", chipCount, discoveryResult.Address)
			} else {
				log.Printf("Connected to ASIC server with %d chips", chipCount)
			}
		}
	}

	// Create hash network
	network, err := neural.NewHashNetwork(*inputSize, *hidden1, *hidden2, *outputSize)
	if err != nil {
		log.Fatalf("Failed to create hash network: %v", err)
	}
	log.Printf("Hash network created: [%d, %d, %d, %d]", *inputSize, *hidden1, *hidden2, *outputSize)

	// Create ASICMethod wrapper for HashMethod interface
	var hashMethod *asic.ASICMethod
	if asicClient != nil && !asicClient.IsUsingFallback() {
		// Create ASICMethod from the server device address
		address := serverDeviceAddr
		if address == "" {
			address = "localhost:8888"
		}
		if *directMode {
			hashMethod = asic.NewDirectASICMethod(address)
		} else {
			hashMethod = asic.NewASICMethod(address)
		}
		if err := hashMethod.Initialize(); err != nil {
			log.Printf("Warning: Failed to initialize ASICMethod: %v", err)
			hashMethod = nil
		} else if *directMode {
			// Monitor the same direct client used by inference. Previously the
			// monitor watched a separate connection, so it could not switch the
			// active direct simulator to fallback mode after a disconnect.
			asicClient = hashMethod.GetClient()
		}
	}

	// Create recursive engine with ASIC support
	engine, err := inference.NewRecursiveEngineWithHashMethod(network, hashMethod, *passes, *jitter, *seedRotation)
	if err != nil {
		log.Fatalf("Failed to create recursive engine: %v", err)
	}
	log.Printf("Recursive engine created: %d passes, jitter=%.3f, seed_rotation=%v", *passes, *jitter, *seedRotation)

	// Create crypto-transformer if enabled
	var cryptoModel *transformer.HasherTransformer
	var miningNeuron *neural.MiningNeuron
	if *enableCrypto {
		log.Printf("Initializing crypto-transformer...")
		transformerConfig := &transformer.HasherTransformerConfig{
			VocabSize:    *vocabSize,
			EmbedDim:     *embedDim,
			NumLayers:    *numLayers,
			NumHeads:     *numHeads,
			ContextLen:   128,
			DropoutRate:  0.1,
			FFNHiddenDim: *ffnHiddenDim,
			Activation:   *cryptoActivation,
		}
		cryptoModel = transformer.NewHasherTransformer(transformerConfig, hashMethod)
		log.Printf("Crypto-transformer created: vocab=%d, embed=%d, layers=%d, heads=%d",
			*vocabSize, *embedDim, *numLayers, *numHeads)

		// Initialize MiningNeuron for converting transformer output to nonces
		// Use much smaller nonce range for software fallback to avoid long delays
		nonceEnd := uint32(1000000) // Large range for ASIC mode
		if asicClient != nil && asicClient.IsUsingFallback() {
			nonceEnd = 1 // Single nonce for software fallback (instant)
		}

		miningNeuronConfig := neural.MiningNeuronConfig{
			InputDim:   *vocabSize, // Input to MiningNeuron is the tokenScores (logits)
			OutputDim:  8,          // Generate 8 projections for the mining header
			Salt:       0xDEADBEEF, // Fixed salt for now
			NonceStart: 0,
			NonceEnd:   nonceEnd, // Dynamic search range based on hardware mode
		}
		miningNeuron = neural.NewMiningNeuron(miningNeuronConfig)
		if hashMethod != nil {
			miningNeuron.SetHashMethod(hashMethod)
		}
		log.Printf("MiningNeuron created with InputDim=%d, OutputDim=%d", miningNeuronConfig.InputDim, miningNeuronConfig.OutputDim)
	}

	// Initialize JitterEngine — the core inference engine for the 21-pass temporal loop.
	// This is the "brain" of the model: it navigates the 12-slot Semantic Maze using
	// SHA-256 hashing to produce a Golden Nonce that IS the next token address.
	// Software mode uses pure-Go SHA-256 (identical logic to the ASIC hardware path).
	jitterCfg := jitterpkg.DefaultJitterConfig()
	jitterCfg.Verbose = false
	inferenceJitterEngine := jitterpkg.NewJitterEngine(jitterCfg)
	log.Printf("JitterEngine initialized: %d-pass temporal loop (software SHA-256)", jitterCfg.PassCount)

	// Load training frames into the FlashSearcher knowledge base.
	// This enables the 3-zone associative lookup and Reverse Lookup (golden nonce → token).
	actualFramesDir := *framesDir
	if actualFramesDir == "" {
		// Try default location
		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			defaultFramesDir := filepath.Join(homeDir, ".local", "share", "hasher", "data", "frames")
			if _, err := os.Stat(defaultFramesDir); err == nil {
				actualFramesDir = defaultFramesDir
			}
		}
	}

	if actualFramesDir != "" {
		n, err := jitterpkg.LoadFromDirectory(inferenceJitterEngine.GetSearcher(), actualFramesDir)
		if err != nil {
			log.Printf("Warning: knowledge base load error: %v", err)
		} else {
			log.Printf("Knowledge base: loaded %d training frames from %s", n, actualFramesDir)
		}
	} else {
		log.Printf("Knowledge base: no --frames-dir specified or found; FlashSearcher starts empty")
	}

	// Connect to the CGMiner API for ASIC-backed final nonce mining.
	// When available, the ASIC mines the semantically refined header at ~500 GH/s,
	// making the golden nonce a genuine hardware output rather than a software hash.
	// In direct mode (-direct), CGMiner is skipped entirely; the ASIC is driven
	// directly over raw USB via hasher-server.
	var cgminerMethod *devicepkg.CGMinerHashMethod
	if *directMode {
		log.Printf("Direct ASIC mode: CGMiner skipped, ASIC driven directly over raw USB")
	} else if *cgminerHost != "" {
		var cgminerErr error
		cgminerMethod, cgminerErr = devicepkg.NewCGMinerHashMethod(*cgminerHost, *cgminerPort, *cgminerTimeout)
		if cgminerErr != nil {
			log.Printf("Warning: CGMiner unavailable at %s:%d: %v — using software golden nonce", *cgminerHost, *cgminerPort, cgminerErr)
			cgminerMethod = nil
		} else {
			log.Printf("CGMiner connected at %s:%d — ASIC will mine the final golden nonce", *cgminerHost, *cgminerPort)
		}
	}

	// Create orchestrator
	orch := &Orchestrator{
		asicClient:           asicClient,
		engine:               engine,
		network:              network,
		cryptoModel:          cryptoModel,
		miningNeuron:         miningNeuron,
		jitterEngine:         inferenceJitterEngine,
		cgminerMethod:        cgminerMethod,
		directMode:           *directMode,
		discoveryResult:      discoveryResult,
		startTime:            time.Now(),
		deployer:             deployer,
		connectionHealthy:    asicClient != nil && !asicClient.IsUsingFallback(),
		lastHealthCheck:      time.Now(),
		stopMonitor:          make(chan struct{}),
		stopLogMonitorChan:   make(chan struct{}),
		serverDeviceIPAddr:   extractIPFromAddress(serverDeviceAddr),
		grpcSocket:           resolveGRPCSocket(*grpcSocket, *knirvserverDeployed),
		knirvserverDeployed:  *knirvserverDeployed,
	}

	// Initialize formal verifier worker if enabled.
	if *formalVerifierEnabled {
		leanCfg := lean.DefaultConfig()
		leanCfg.LeanBinary = *leanBinary
		if *leanWorkDir != "" {
			leanCfg.WorkDir = *leanWorkDir
		}
		leanCfg.MaxCheckerSeconds = *leanMaxSeconds
		orch.leanWorker = lean.NewWorker(leanCfg, nil)
		orch.grpcPort = *grpcPort
		log.Printf("Formal verifier worker initialized (lean_binary=%s work_dir=%s)", leanCfg.LeanBinary, leanCfg.WorkDir)

		workerPtr, ok := orch.leanWorker.(*lean.Worker)
		if ok {
			proofStoreDir := filepath.Join(os.TempDir(), "knirv-proof-assets")
			proofStore, err := lean.NewFileSystemProofAssetStore(proofStoreDir)
			if err != nil {
				log.Printf("WARNING: failed to create proof asset store: %v", err)
			} else {
				orch.formalVerifierGRPCServer = lean.NewGRPCServer(workerPtr, proofStore)
				if orch.knirvserverDeployed || orch.grpcSocket != "" {
					log.Printf("Formal verification gRPC server will use Unix socket: %s", orch.grpcSocket)
				} else {
					log.Printf("Formal verification gRPC server initialized on port %d", *grpcPort)
				}
			}
		}
	}

	// Load schema and activate domain-specific plugin if --schema is provided.
	// The same binary becomes a General HASHER or a MATHASHER at runtime:
	//   ./hasher-host --schema pipeline/2_DATA_ENCODER/config/prose_schema.yaml  → HASHER
	//   ./hasher-host --schema pipeline/2_DATA_ENCODER/config/math_schema.yaml   → MATHASHER
	if *schemaPath != "" {
		if _, statErr := os.Stat(*schemaPath); statErr != nil {
			log.Fatalf("Schema file not found: %s", *schemaPath)
		}
		domain, domErr := hashermath.LoadDomainFromSchema(*schemaPath)
		if domErr != nil {
			log.Fatalf("Failed to read schema domain from %s: %v", *schemaPath, domErr)
		}
		log.Printf("Schema loaded: domain=%s file=%s", domain, *schemaPath)

		if domain == "MATHASHER" {
			subdomainVal := uint32(*subdomain)
			rules, rulesErr := hashermath.LoadWatchdogRulesFromYAML(*schemaPath)
			if rulesErr != nil {
				log.Printf("Warning: could not load watchdog rules from schema: %v — using hard-coded fallback", rulesErr)
				orch.mathServer = hasherapi.NewMathVerifierServer(subdomainVal)
			} else {
				log.Printf("MATHASHER mode: loaded %d validation rules", len(rules))
				orch.mathServer = hasherapi.NewMathVerifierServerWithRules(subdomainVal, rules)
			}

			// Configure formal verifier if enabled.
			if *formalVerifierEnabled {
				orch.formalVerifierEnabled = true
				allowedImports := []string{
					"Mathlib.Algebra.Group.Basic",
					"Mathlib.Data.Real.Basic",
				}
				if *proofLedgerPath != "" {
					orch.proofLedgerPath = *proofLedgerPath
				} else if *framesDir != "" {
					orch.proofLedgerPath = filepath.Join(*framesDir, "proof_writes.jsonl")
				}

				orch.mathServer = orch.mathServer.
					WithAllowedImports(allowedImports).
					WithProofLedgerPath(orch.proofLedgerPath)

				log.Printf("Formal verification enabled: lean_binary=%s max_seconds=%d ledger=%s",
					*leanBinary, *leanMaxSeconds, orch.proofLedgerPath)
			}

			log.Printf("MATHASHER routes will be available at POST /v1/verify/math and GET /v1/verify/math/health")
		}
	}

	// Start connection monitor for ASIC connection health
	if asicClient != nil && !asicClient.IsUsingFallback() {
		go orch.runConnectionMonitor()
	}

	// Start server log monitoring if enabled and we have a server device IP
	if *monitorServerLogs && orch.serverDeviceIPAddr != "" {
		go orch.monitorServerLogs()
	} else if *monitorServerLogs {
		log.Printf("Warning: Server log monitoring enabled but no server device IP available")
	}

	// Find available port for API mode
	var apiPort int
	if *mode == "api" {
		var err error
		apiPort, err = findOpenPort(*port)
		if err != nil {
			log.Fatalf("Failed to find available port: %v", err)
		}
		// Update the port variable for the rest of the code
		*port = apiPort

		// Write port to file for CLI discovery
		if err := writePortFile(apiPort); err != nil {
			log.Printf("Warning: failed to write port file: %v", err)
		}
	}

	// Run based on mode
	switch *mode {
	case "api":
		if *enableAPI {
			runAPIServer(orch)
		} else {
			log.Println("API server disabled by flag --api=false")
			log.Println("Running in single mode instead")
			runSingleMode(orch)
		}
	case "single":
		runSingleMode(orch)
	case "batch":
		runBatchMode(orch)
	case "stream":
		runStreamMode(orch)
	case "metrics":
		showMetrics(orch)
	case "info":
		showInfo(orch)
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

// extractIPFromAddress extracts just the IP address from an address string (e.g., "192.168.1.1:8888" -> "192.168.1.1")
func extractIPFromAddress(addr string) string {
	if addr == "" {
		return ""
	}
	parts := strings.Split(addr, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return addr
}

// resolveGRPCSocket returns the Unix socket path for gRPC when running under KNIRVSERVER,
// or the explicit --grpc-socket value if provided. Returns empty string for TCP mode.
func resolveGRPCSocket(explicitSocket string, knirvserverDeployed bool) string {
	if explicitSocket != "" {
		return explicitSocket
	}
	if knirvserverDeployed {
		return filepath.Join(socketDir, "formal-verifier.sock")
	}
	return ""
}

// monitorServerLogs monitors the server logs via SSH and handles auto-reboot scenarios
func (o *Orchestrator) monitorServerLogs() {
	log.Printf("Starting server log monitor for device %s", o.serverDeviceIPAddr)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-o.stopLogMonitorChan:
			log.Printf("Server log monitor stopped")
			return
		case <-ticker.C:
			if o.isRebooting {
				// Skip monitoring while we're handling a reboot
				continue
			}

			// Check for auto-reboot marker in server logs
			if err := o.checkServerLogs(); err != nil {
				// Only log errors occasionally to avoid spam
				if o.reconnectAttempts%10 == 0 {
					log.Printf("Warning: Failed to check server logs: %v", err)
				}
			}
		}
	}
}

// checkServerLogs checks the server logs via SSH for auto-reboot markers
func (o *Orchestrator) checkServerLogs() error {
	// SSH into device and tail the log file for recent entries
	sshCmd := fmt.Sprintf("sshpass -p '%s' ssh -o KexAlgorithms=+diffie-hellman-group14-sha1 -o HostKeyAlgorithms=+ssh-rsa -o StrictHostKeyChecking=no -o ConnectTimeout=5 root@%s "+
		"\"tail -n 50 %s 2>/dev/null || echo 'LOG_FILE_NOT_FOUND'\"",
		*serverSSHPassword, o.serverDeviceIPAddr, *serverLogPath)

	cmd := exec.Command("sh", "-c", sshCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh command failed: %v (output: %s)", err, string(output))
	}

	logOutput := string(output)

	// Check for auto-reboot marker
	if strings.Contains(logOutput, "AUTO_REBOOT_TRIGGERED") {
		log.Printf("AUTO_REBOOT_TRIGGERED marker detected in server logs - server is rebooting")
		go o.handleServerReboot()
	}

	return nil
}

// handleServerReboot handles the full server reboot recovery sequence
func (o *Orchestrator) handleServerReboot() {
	o.mu.Lock()
	if o.isRebooting {
		o.mu.Unlock()
		return // Already handling a reboot
	}
	o.isRebooting = true
	o.mu.Unlock()

	log.Printf("=== Starting server reboot recovery sequence ===")

	// Step 1: Log that reboot is in progress
	log.Printf("Server reboot in progress - waiting for connection to drop...")

	// Step 2: Wait for connection to drop (indicating server is rebooting)
	connectionDropped := o.waitForConnectionDrop(60 * time.Second)
	if !connectionDropped {
		log.Printf("Warning: Connection did not drop within timeout, assuming reboot is complete")
	} else {
		log.Printf("Connection dropped - server is rebooting")
	}

	// Step 3: Poll for server to come back online (retry every 5 seconds for up to 2 minutes)
	log.Printf("Waiting for server to come back online...")
	backOnline := o.pollForServerRecovery(120*time.Second, 5*time.Second)
	if !backOnline {
		log.Printf("ERROR: Server did not come back online within timeout - manual intervention required")
		o.mu.Lock()
		o.isRebooting = false
		o.mu.Unlock()
		return
	}
	log.Printf("Server is back online")

	// Step 4: Download the server logs with timestamp
	log.Printf("Downloading server logs...")
	logFile, err := o.downloadServerLogs()
	if err != nil {
		log.Printf("Warning: Failed to download server logs: %v", err)
	} else {
		log.Printf("Server logs saved to: %s", logFile)
	}

	// Step 5: Cleanup old server process
	log.Printf("Cleaning up old server process...")
	if err := o.cleanupOldServerProcess(); err != nil {
		log.Printf("Warning: Failed to cleanup old server process: %v", err)
	}

	// Step 6: Redeploy the server binary
	log.Printf("Redeploying server binary...")
	if err := o.redeployServerBinary(); err != nil {
		log.Printf("ERROR: Failed to redeploy server binary: %v", err)
		o.mu.Lock()
		o.isRebooting = false
		o.mu.Unlock()
		return
	}
	log.Printf("Server binary redeployed successfully")

	// Step 7: Restart the server
	log.Printf("Restarting hasher-server...")
	if err := o.restartServer(); err != nil {
		log.Printf("ERROR: Failed to restart server: %v", err)
		o.mu.Lock()
		o.isRebooting = false
		o.mu.Unlock()
		return
	}
	log.Printf("Server restarted successfully")

	// Step 8: Reconnect the ASIC client
	log.Printf("Reconnecting ASIC client...")
	if err := o.reconnectASICClient(); err != nil {
		log.Printf("ERROR: Failed to reconnect ASIC client: %v", err)
		o.mu.Lock()
		o.isRebooting = false
		o.mu.Unlock()
		return
	}
	log.Printf("ASIC client reconnected successfully")

	// Step 9: Resume normal operations
	o.mu.Lock()
	o.isRebooting = false
	o.connectionHealthy = true
	o.lastHealthCheck = time.Now()
	o.reconnectAttempts = 0
	o.mu.Unlock()

	log.Printf("=== Server reboot recovery sequence complete - resuming normal operations ===")
}

// waitForConnectionDrop waits for the SSH connection to drop (indicating server reboot)
func (o *Orchestrator) waitForConnectionDrop(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	pollInterval := 1 * time.Second

	for time.Now().Before(deadline) {
		// Try to connect via SSH - if it fails, connection has dropped
		sshCmd := fmt.Sprintf("sshpass -p '%s' ssh -o KexAlgorithms=+diffie-hellman-group14-sha1 -o HostKeyAlgorithms=+ssh-rsa -o StrictHostKeyChecking=no -o ConnectTimeout=3 root@%s echo 'OK'",
			*serverSSHPassword, o.serverDeviceIPAddr)
		cmd := exec.Command("sh", "-c", sshCmd)
		if err := cmd.Run(); err != nil {
			// Connection dropped
			return true
		}
		time.Sleep(pollInterval)
	}

	return false // Timeout reached
}

// pollForServerRecovery polls for the server to come back online
func (o *Orchestrator) pollForServerRecovery(timeout time.Duration, interval time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Try to SSH into the device
		sshCmd := fmt.Sprintf("sshpass -p '%s' ssh -o KexAlgorithms=+diffie-hellman-group14-sha1 -o HostKeyAlgorithms=+ssh-rsa -o StrictHostKeyChecking=no -o ConnectTimeout=5 root@%s echo 'READY'",
			*serverSSHPassword, o.serverDeviceIPAddr)
		cmd := exec.Command("sh", "-c", sshCmd)
		if err := cmd.Run(); err == nil {
			// Server is back online
			return true
		}

		log.Printf("Waiting for server to come back online... (retry in %v)", interval)
		time.Sleep(interval)
	}

	return false // Timeout reached
}

// downloadServerLogs downloads the server logs and saves them with a timestamp
func (o *Orchestrator) downloadServerLogs() (string, error) {
	// Create logs directory if it doesn't exist
	logsDir := "./logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Generate timestamped filename
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("hasher-server_%s_%s.log", o.serverDeviceIPAddr, timestamp)
	filepath := filepath.Join(logsDir, filename)

	// Download logs via SSH
	sshCmd := fmt.Sprintf("sshpass -p '%s' ssh -o KexAlgorithms=+diffie-hellman-group14-sha1 -o HostKeyAlgorithms=+ssh-rsa -o StrictHostKeyChecking=no -o ConnectTimeout=10 root@%s "+
		"\"cat %s 2>/dev/null || echo 'LOG_FILE_NOT_FOUND'\"",
		*serverSSHPassword, o.serverDeviceIPAddr, *serverLogPath)

	cmd := exec.Command("sh", "-c", sshCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ssh command failed: %v (output: %s)", err, string(output))
	}

	// Write logs to file
	if err := os.WriteFile(filepath, output, 0644); err != nil {
		return "", fmt.Errorf("failed to write log file: %w", err)
	}

	log.Printf("Successfully downloaded server logs to %s (size: %d bytes)", filepath, len(output))
	return filepath, nil
}

// cleanupOldServerProcess kills any old hasher-server processes on the device
func (o *Orchestrator) cleanupOldServerProcess() error {
	sshCmd := fmt.Sprintf("sshpass -p '%s' ssh -o KexAlgorithms=+diffie-hellman-group14-sha1 -o HostKeyAlgorithms=+ssh-rsa -o StrictHostKeyChecking=no -o ConnectTimeout=10 root@%s "+
		"\"pkill -9 hasher-server 2>/dev/null; echo 'DONE'\"",
		*serverSSHPassword, o.serverDeviceIPAddr)

	cmd := exec.Command("sh", "-c", sshCmd)
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "DONE") {
		// pkill may fail if process doesn't exist, which is OK
		if !strings.Contains(err.Error(), "exit status 1") {
			return fmt.Errorf("ssh command failed: %v (output: %s)", err, string(output))
		}
	}

	return nil
}

// redeployServerBinary redeploys the hasher-server binary to the device
func (o *Orchestrator) redeployServerBinary() error {
	// If we have a deployer, use it
	if o.deployer != nil && o.serverDeviceIPAddr != "" {
		_, err := o.deployer.EnsureServerDeployed(o.serverDeviceIPAddr)
		if err != nil {
			return fmt.Errorf("deployer redeployment failed: %w", err)
		}
		return nil
	}

	// Otherwise, use direct SSH to copy and setup the binary
	// This assumes the binary is available locally
	return fmt.Errorf("no deployer available for redeployment")
}

// restartServer restarts the hasher-server on the device
func (o *Orchestrator) restartServer() error {
	sshCmd := fmt.Sprintf("sshpass -p '%s' ssh -o KexAlgorithms=+diffie-hellman-group14-sha1 -o HostKeyAlgorithms=+ssh-rsa -o StrictHostKeyChecking=no -o ConnectTimeout=10 root@%s "+
		"\"nohup /tmp/hasher-server --auto-reboot > %s 2>&1 & sleep 2; pgrep hasher-server > /dev/null && echo 'STARTED' || echo 'FAILED'\"",
		*serverSSHPassword, o.serverDeviceIPAddr, *serverLogPath)

	cmd := exec.Command("sh", "-c", sshCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh command failed: %v (output: %s)", err, string(output))
	}

	if !strings.Contains(string(output), "STARTED") {
		return fmt.Errorf("server failed to start (output: %s)", string(output))
	}

	// Wait for server to be ready
	time.Sleep(3 * time.Second)

	return nil
}

// reconnectASICClient reconnects the ASIC client after server restart
func (o *Orchestrator) reconnectASICClient() error {
	// Close existing client
	if o.asicClient != nil {
		o.asicClient.Close()
	}

	// Build server address
	serverAddr := fmt.Sprintf("%s:8888", o.serverDeviceIPAddr)

	// Try to reconnect with retries
	backoff := 1 * time.Second
	maxBackoff := 10 * time.Second
	timeout := 60 * time.Second
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		client, err := asic.NewASICClient(serverAddr)
		if err == nil {
			// Test the connection
			_, err := client.GetDeviceInfo()
			if err == nil {
				o.mu.Lock()
				o.asicClient = client
				o.mu.Unlock()
				return nil
			}
			client.Close()
		}

		log.Printf("Reconnection attempt failed, retrying in %v...", backoff)
		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	return fmt.Errorf("failed to reconnect within timeout (%v)", timeout)
}

// signalLogMonitorStop signals the log monitor to stop
func (o *Orchestrator) signalLogMonitorStop() {
	if o.stopLogMonitorChan != nil {
		close(o.stopLogMonitorChan)
	}
}

// runAPIServer starts the REST API server
func runAPIServer(orch *Orchestrator) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// API routes
	api := router.Group("/api/v1")
	{
		// Core inference endpoints
		api.POST("/infer", orch.handleInfer)
		api.POST("/batch", orch.handleBatchInfer)
		api.GET("/health", orch.handleHealth)
		api.GET("/metrics", orch.handleMetrics)
		api.GET("/device", orch.handleDeviceInfo)
		api.GET("/discovery", orch.handleDiscovery)
		api.POST("/discovery/scan", orch.handleDiscoveryScan)

		// Crypto-transformer endpoints
		api.POST("/chat", orch.handleChat)
		api.GET("/crypto/status", orch.handleCryptoStatus)

		// Logical rules endpoints
		api.GET("/rules", orch.handleListRules)
		api.POST("/rules", orch.handleAddRule)
		api.DELETE("/rules/:id", orch.handleDeleteRule)
		api.GET("/domains", orch.handleListDomains)

		// Shutdown endpoint
		api.POST("/shutdown", orch.handleShutdown)
	}

	// Domain-specific plugin routes — only mounted when a MATHASHER schema is loaded.
	if orch.mathServer != nil {
		v1 := router.Group("/v1")
		v1.POST("/verify/math", gin.WrapF(orch.mathServer.VerifyHandler()))
		v1.GET("/verify/math/health", gin.WrapF(orch.mathServer.MathHealthHandler()))
		log.Printf("MATHASHER routes mounted: POST /v1/verify/math  |  GET /v1/verify/math/health")

		// Formal verification routes — only mounted when formal verifier is enabled.
		if orch.formalVerifierEnabled {
			v1.POST("/verify/proof", orch.handleSubmitProof)
			v1.GET("/verify/proof/:id", orch.handleGetProofStatus)
			v1.GET("/verify/proof/:id/asset", orch.handleGetProofAsset)
			v1.POST("/verify/proof/:id/replay", orch.handleReplayProof)
			v1.POST("/verify/proof/:id/attest", orch.handleAttestProof)
			v1.GET("/verify/proof/metrics", orch.handleFormalVerificationMetrics)
			log.Printf("Formal verification routes mounted: POST /v1/verify/proof | GET /v1/verify/proof/:id | GET /v1/verify/proof/:id/asset | POST /v1/verify/proof/:id/replay | POST /v1/verify/proof/:id/attest | GET /v1/verify/proof/metrics")
		}
	}

	// Set up graceful shutdown
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: router,
	}

	go func() {
		log.Printf("API server listening on :%d", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("API server error: %v", err)
		}
	}()

	if orch.formalVerifierGRPCServer != nil {
		var grpcLis net.Listener
		var grpcAddr string
		var err error

		if orch.grpcSocket != "" {
			socketPath := orch.grpcSocket
			_ = os.Remove(socketPath)
			grpcLis, err = net.Listen("unix", socketPath)
			if err != nil {
				log.Printf("WARNING: failed to start gRPC Unix socket at %s: %v", socketPath, err)
			} else {
				if err := writeSocketFile(socketPath); err != nil {
					log.Printf("WARNING: failed to write socket discovery file: %v", err)
				}
				grpcAddr = "unix://" + socketPath
				grpcSrv := grpc.NewServer()
				pb.RegisterFormalVerificationServiceServer(grpcSrv, orch.formalVerifierGRPCServer)
				go func() {
					log.Printf("FormalVerificationService gRPC server listening on Unix socket %s", socketPath)
					if err := grpcSrv.Serve(grpcLis); err != nil {
						log.Printf("gRPC server error: %v", err)
					}
				}()
				orch.grpcServer = grpcSrv
			}
		} else {
			grpcAddr = fmt.Sprintf(":%d", orch.grpcPort)
			grpcLis, err = net.Listen("tcp", grpcAddr)
			if err != nil {
				log.Printf("WARNING: failed to start gRPC server on %s: %v", grpcAddr, err)
			} else {
				grpcSrv := grpc.NewServer()
				pb.RegisterFormalVerificationServiceServer(grpcSrv, orch.formalVerifierGRPCServer)
				go func() {
					log.Printf("FormalVerificationService gRPC server listening on %s", grpcAddr)
					if err := grpcSrv.Serve(grpcLis); err != nil {
						log.Printf("gRPC server error: %v", err)
					}
				}()
				orch.grpcServer = grpcSrv
			}
		}

		_ = grpcAddr
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Stop monitors
	orch.stopConnectionMonitor()
	orch.signalLogMonitorStop()

	// Clean up port file right away
	cleanupPortFile()

	// Clean up Unix socket if used
	if orch.grpcSocket != "" {
		cleanupSocketFile(orch.grpcSocket)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	if orch.asicClient != nil {
		orch.asicClient.Close()
	}

	if orch.cgminerMethod != nil {
		orch.cgminerMethod.Close()
	}

	if orch.grpcServer != nil {
		orch.grpcServer.GracefulStop()
	}

	// Cleanup deployed hasher-server if auto-deployment was used
	if orch.deployer != nil {
		orch.deployer.Cleanup()
	}

	log.Println("Server stopped")
}

// handleInfer handles single inference requests
func (o *Orchestrator) handleInfer(c *gin.Context) {
	var req InferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Decode base64 input
	data, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 data"})
		return
	}

	// Run inference
	start := time.Now()
	result, err := o.engine.Infer(data)
	latency := time.Since(start)

	o.mu.Lock()
	o.totalInferences++
	o.totalLatencyNs += uint64(latency.Nanoseconds())
	if err != nil {
		o.failedInfers++
	} else {
		o.successfulInfers++
	}
	o.mu.Unlock()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, InferResponse{
		Prediction:        result.Consensus.Prediction,
		Confidence:        result.Consensus.Confidence,
		AverageConfidence: result.Consensus.AverageConfidence,
		Passes:            result.TotalPasses,
		ValidPasses:       result.ValidPasses,
		LatencyMs:         float64(latency.Milliseconds()),
		UsingASIC:         o.safeIsUsingHardware(),
	})
}

// handleBatchInfer handles batch inference requests
func (o *Orchestrator) handleBatchInfer(c *gin.Context) {
	var req BatchInferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if len(req.Data) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty batch"})
		return
	}

	start := time.Now()
	results := make([]InferResponse, len(req.Data))

	for i, dataStr := range req.Data {
		data, err := base64.StdEncoding.DecodeString(dataStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid base64 at index %d", i)})
			return
		}

		inferStart := time.Now()
		result, err := o.engine.Infer(data)
		inferLatency := time.Since(inferStart)

		o.mu.Lock()
		o.totalInferences++
		o.totalLatencyNs += uint64(inferLatency.Nanoseconds())
		if err != nil {
			o.failedInfers++
		} else {
			o.successfulInfers++
		}
		o.mu.Unlock()

		if err != nil {
			results[i] = InferResponse{
				Prediction: -1,
				LatencyMs:  float64(inferLatency.Milliseconds()),
				UsingASIC:  o.safeIsUsingHardware(),
			}
			continue
		}

		results[i] = InferResponse{
			Prediction:        result.Consensus.Prediction,
			Confidence:        result.Consensus.Confidence,
			AverageConfidence: result.Consensus.AverageConfidence,
			Passes:            result.TotalPasses,
			ValidPasses:       result.ValidPasses,
			LatencyMs:         float64(inferLatency.Milliseconds()),
			UsingASIC:         o.safeIsUsingHardware(),
		}
	}

	c.JSON(http.StatusOK, BatchInferResponse{
		Results:   results,
		TotalMs:   float64(time.Since(start).Milliseconds()),
		UsingASIC: o.safeIsUsingHardware(),
	})
}

// safeIsUsingHardware safely checks if using hardware, recovering from any panics
func (o *Orchestrator) safeIsUsingHardware() bool {
	if o == nil || o.engine == nil {
		return false
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Warning: IsUsingHardware panic recovered: %v", r)
		}
	}()
	return o.engine.IsUsingHardware()
}

// handleHealth handles health check requests
func (o *Orchestrator) handleHealth(c *gin.Context) {
	chipCount := 0
	if o.asicClient != nil {
		chipCount = o.asicClient.GetChipCount()
	}

	o.mu.RLock()
	connectionHealthy := o.connectionHealthy
	lastHealthCheck := o.lastHealthCheck
	isRebooting := o.isRebooting
	o.mu.RUnlock()

	// Determine overall status
	status := "healthy"
	if o.asicClient != nil && !connectionHealthy {
		status = "degraded"
	}
	if isRebooting {
		status = "rebooting"
	}

	// Safely check if using hardware
	usingASIC := o.safeIsUsingHardware()

	c.JSON(http.StatusOK, HealthResponse{
		Status:            status,
		UsingASIC:         usingASIC,
		ChipCount:         chipCount,
		Uptime:            time.Since(o.startTime).String(),
		ConnectionHealthy: connectionHealthy,
		LastHealthCheck:   lastHealthCheck.Format(time.RFC3339),
		DirectMode:        o.directMode,
	})
}

// handleMetrics handles metrics requests
func (o *Orchestrator) handleMetrics(c *gin.Context) {
	o.mu.RLock()
	totalInferences := o.totalInferences
	successfulInfers := o.successfulInfers
	failedInfers := o.failedInfers
	totalLatencyNs := o.totalLatencyNs
	o.mu.RUnlock()

	avgLatencyMs := float64(0)
	if totalInferences > 0 {
		avgLatencyMs = float64(totalLatencyNs) / float64(totalInferences) / 1e6
	}

	chipCount := 0
	if o.asicClient != nil {
		chipCount = o.asicClient.GetChipCount()
	}

	c.JSON(http.StatusOK, MetricsResponse{
		TotalInferences:  totalInferences,
		SuccessfulInfers: successfulInfers,
		FailedInfers:     failedInfers,
		AverageLatencyMs: avgLatencyMs,
		UsingASIC:        o.safeIsUsingHardware(),
		ChipCount:        chipCount,
		Uptime:           time.Since(o.startTime).String(),
	})
}

// runConnectionMonitor periodically checks ASIC connection health and attempts reconnection
func (o *Orchestrator) runConnectionMonitor() {
	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()

	log.Printf("Connection monitor started for ASIC server")

	for {
		select {
		case <-o.stopMonitor:
			log.Printf("Connection monitor stopped")
			return
		case <-ticker.C:
			if o.isRebooting {
				// Skip health checks during reboot handling
				continue
			}
			o.checkConnectionHealth()
		}
	}
}

// checkConnectionHealth verifies ASIC connection and attempts reconnection if needed
func (o *Orchestrator) checkConnectionHealth() {
	if o.asicClient == nil {
		return
	}

	// A direct client switches itself to software simulation as soon as a hash
	// RPC loses its transport. Do not treat its synthetic software device info
	// as a recovered ASIC connection or attempt to reconnect it in a loop.
	if o.directMode && o.asicClient.IsUsingFallback() {
		o.mu.Lock()
		if o.connectionHealthy {
			log.Printf("ASIC connection lost; direct mode is now using SOFTWARE SIMULATION")
		}
		o.connectionHealthy = false
		o.mu.Unlock()
		return
	}

	// Try to get device info as a health check
	_, err := o.asicClient.GetDeviceInfo()

	o.mu.Lock()
	defer o.mu.Unlock()

	if err != nil {
		// Connection is unhealthy
		if o.connectionHealthy {
			log.Printf("ASIC connection lost: %v", err)
			o.connectionHealthy = false
		}
		if o.directMode {
			o.asicClient.SetFallbackOnConnectionLoss(true)
			o.asicClient.FallbackToSoftware("health check", err)
			log.Printf("Direct ASIC connection failed; switching to SOFTWARE SIMULATION mode")
			return
		}

		// Attempt reconnection with exponential backoff
		if o.reconnectAttempts < 5 { // Max 5 attempts before giving up temporarily
			o.reconnectAttempts++
			log.Printf("Attempting ASIC reconnection (attempt %d/5)...", o.reconnectAttempts)

			if err := o.asicClient.Reconnect(); err != nil {
				log.Printf("Reconnection failed: %v", err)
			} else {
				log.Printf("Successfully reconnected to ASIC server")
				o.connectionHealthy = true
				o.lastHealthCheck = time.Now()
				o.reconnectAttempts = 0
			}
		} else if o.reconnectAttempts == 5 {
			log.Printf("Max reconnection attempts reached. Will retry in 60 seconds.")
			o.reconnectAttempts++ // Increment to avoid repeated log message
		} else if o.reconnectAttempts >= 11 { // Reset after ~60 seconds (6 more ticks)
			o.reconnectAttempts = 0
		} else {
			o.reconnectAttempts++
		}
	} else {
		// Connection is healthy
		if !o.connectionHealthy {
			log.Printf("ASIC connection restored")
		}
		o.connectionHealthy = true
		o.lastHealthCheck = time.Now()
		o.reconnectAttempts = 0
	}
}

// stopConnectionMonitor signals the connection monitor to stop
func (o *Orchestrator) stopConnectionMonitor() {
	if o.stopMonitor != nil {
		close(o.stopMonitor)
	}
}

// handleDeviceInfo handles device info requests
func (o *Orchestrator) handleDeviceInfo(c *gin.Context) {
	if o.asicClient == nil {
		response := gin.H{
			"device_path":      "software",
			"chip_count":       0,
			"firmware_version": "software-fallback",
			"is_operational":   true,
			"uptime_seconds":   uint64(time.Since(o.startTime).Seconds()),
		}
		// Log the response in pretty JSON for debugging
		log.Printf("Device info (software fallback): %s", prettyJSON(response))
		c.JSON(http.StatusOK, response)
		return
	}

	info, err := o.asicClient.GetDeviceInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"device_path":      info.DevicePath,
		"chip_count":       info.ChipCount,
		"firmware_version": info.FirmwareVersion,
		"is_operational":   info.IsOperational,
		"uptime_seconds":   info.UptimeSeconds,
	}
	// Log the response in pretty JSON for debugging
	log.Printf("Device info: %s", prettyJSON(response))
	c.JSON(http.StatusOK, response)
}

// handleShutdown handles a request to gracefully shut down the server.
func (o *Orchestrator) handleShutdown(c *gin.Context) {
	log.Println("Received shutdown request via API...")
	c.JSON(http.StatusOK, gin.H{"message": "shutdown sequence initiated"})

	// Use a goroutine to send the signal after the response has been sent
	go func() {
		// A small delay to allow the HTTP response to flush
		time.Sleep(100 * time.Millisecond)
		p, err := os.FindProcess(os.Getpid())
		if err != nil {
			log.Printf("Error finding process to signal shutdown: %v", err)
			return
		}
		log.Println("Sending SIGTERM to self to trigger graceful shutdown...")
		if err := p.Signal(syscall.SIGTERM); err != nil {
			log.Printf("Error sending SIGTERM to self: %v", err)
		}
	}()
}

// buildContextFrame constructs a 12-slot TrainingFrame from the current inference context.
// The slots encode the semantic state that guides the JitterEngine's 21-pass search.
//
// Slot layout (per HASHER architecture):
//
//	Slot 0:  Anchor  — topic fingerprint (XOR-rotate over all context tokens)
//	Slot 1:  Subject — first token in the context
//	Slot 2:  Action  — most recent token in the context
//	Slot 3:  Entropy — flat XOR of all context tokens
//	Slot 4:  POS     — part-of-speech hint (0 = unknown/NOUN)
//	Slots 5-9:       — sliding window of the last 5 tokens
//	Slot 10: Domain  — semantic domain (e.g. DOMAIN_PROSE)
//	Slot 11: Length  — context window size
func buildContextFrame(contextTokens []int, domain uint32) jitterpkg.TrainingFrame {
	var slots [12]uint32

	// Slot 0: Anchor — rotating XOR gives a unique fingerprint per context sequence
	for _, t := range contextTokens {
		slots[0] ^= uint32(t)
		slots[0] = (slots[0] << 7) | (slots[0] >> 25) // left-rotate 7 bits
	}

	// Slot 1: Subject — first token
	if len(contextTokens) > 0 {
		slots[1] = uint32(contextTokens[0])
	}

	// Slot 2: Action — most recent token
	if len(contextTokens) > 0 {
		slots[2] = uint32(contextTokens[len(contextTokens)-1])
	}

	// Slot 3: Entropy — plain XOR of all tokens
	for _, t := range contextTokens {
		slots[3] ^= uint32(t)
	}

	// Slot 4: POS — 0 = NOUN (default when no tagger is available)
	slots[4] = 0

	// Slots 5-9: last 5 tokens from the context window
	contextStart := 0
	if len(contextTokens) > 5 {
		contextStart = len(contextTokens) - 5
	}
	for i, t := range contextTokens[contextStart:] {
		if i < 5 {
			slots[5+i] = uint32(t)
		}
	}

	// Slot 10: Domain
	slots[10] = domain

	// Slot 11: Context length
	slots[11] = uint32(len(contextTokens))

	return jitterpkg.TrainingFrame{
		AsicSlots:     slots,
		TargetTokenID: 0, // Unknown during inference; the JitterEngine determines this
	}
}

// handleChat handles chat inference requests using the JitterEngine.
//
// Architecture (per HASHER specification):
//   - Input text is tokenized into character-level token IDs.
//   - For each output token a 12-slot TrainingFrame is built from the current context.
//   - The frame is converted to an 80-byte Bitcoin-style header via ToBitcoinHeader().
//   - The JitterEngine runs the 21-pass temporal associative search (pure-Go SHA-256 in
//     software mode; ASIC hardware in hardware mode) to produce a GoldenNonceResult.
//   - The golden nonce (first 4 bytes of the final hash) IS the token address in the
//     Arrow Knowledge Base. If the FlashSearcher finds a match, it returns the token ID
//     directly. Otherwise the nonce is projected into the vocabulary range as a fallback.
//   - The full sequence of token IDs is detokenized to produce the response string.
func (o *Orchestrator) handleChat(c *gin.Context) {
	if o.jitterEngine == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "JitterEngine not initialized"})
		return
	}
	if o.cryptoModel == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "crypto-transformer config not available"})
		return
	}

	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	start := time.Now()

	// Tokenize the input message using Tiktoken (consistent with pipeline).
	inputTokenIDs := tokenizer.Tokenize(req.Message, o.cryptoModel.Config.VocabSize)
	currentContext := req.Context
	if len(currentContext) == 0 {
		currentContext = inputTokenIDs
	}

	usingASIC := o.asicClient != nil && !o.asicClient.IsUsingFallback()
	vocabSize := o.cryptoModel.Config.VocabSize

	// Generation loop: one JitterEngine pass per output token.
	const maxGenerationLen = 20
	generatedTokens := make([]int, 0, maxGenerationLen)
	consecutiveMisses := 0

	// Check if knowledge base is empty before starting generation
	if o.jitterEngine.GetSearcher().Size() == 0 {
		c.JSON(http.StatusOK, ChatResponse{
			Response:   "I am ready, but my knowledge base is currently empty. Please run the Data Pipeline (menu option 1) to train me on some data first.",
			TokenID:    0,
			Confidence: 0.0,
			LatencyMs:  float64(time.Since(start).Milliseconds()),
			UsingASIC:  usingASIC,
		})
		return
	}

	for i := 0; i < maxGenerationLen; i++ {
		// First, check if the current context matches a known pattern in the knowledge base.
		// This provides a deterministic "Exact Match" path for high-fidelity responses.
		nextTokenID, found := o.jitterEngine.GetSearcher().LookupByContext(currentContext)

		if found {
			consecutiveMisses = 0
		} else {
			// If no exact match is found, use the probabilistic path via the JitterEngine.
			// This represents the "Discovery" mode where the system searches for associations.

			// Build the 12-slot semantic context frame from the current window.
			frame := buildContextFrame(currentContext, jitterpkg.DOMAIN_PROSE)
			header := frame.ToBitcoinHeader()

			// Run the 21-pass temporal associative search.
			result, err := o.jitterEngine.Execute21PassLoop(header, 0)
			if err != nil {
				log.Printf("Warning: JitterEngine failed at step %d: %v", i, err)
				break
			}

			// Determine the golden nonce for this step.
			goldenNonce := result.Nonce
			if o.cgminerMethod != nil && o.cgminerMethod.IsAvailable() {
				asicNonce, _, mineErr := o.cgminerMethod.MineForGoldenNonce(header)
				if mineErr != nil {
					log.Printf("Warning: CGMiner mining failed at step %d: %v (using software nonce)", i, mineErr)
				} else {
					goldenNonce = asicNonce
					usingASIC = true
				}
			}

			// The golden nonce IS the token address in the knowledge base (Reverse Lookup).
			nextTokenID, found = o.jitterEngine.GetSearcher().LookupByNonce(goldenNonce, vocabSize)
			if !found {
				// Knowledge base miss: project the nonce into the vocabulary range.
				nextTokenID = int(goldenNonce % uint32(vocabSize))
				consecutiveMisses++
			} else {
				consecutiveMisses = 0
			}
		}

		// Stop if we have too many consecutive misses (preventing long gibberish)
		if consecutiveMisses > 3 {
			break
		}

		generatedTokens = append(generatedTokens, nextTokenID)

		currentContext = append(currentContext, nextTokenID)
		if len(currentContext) > o.cryptoModel.Config.ContextLen {
			currentContext = currentContext[len(currentContext)-o.cryptoModel.Config.ContextLen:]
		}

		// Stop at sentence-ending punctuation (Tiktoken IDs: . = 13, ! = 0, ? = 30)
		// Only break if we've generated at least 2 tokens to avoid premature cut-off
		if len(generatedTokens) >= 2 && (nextTokenID == 13 || nextTokenID == 0 || nextTokenID == 30) {
			break
		}
	}

	// Final detokenization: convert the full token sequence back into text.
	// If no tokens were generated (e.g. all nonces missed the knowledge base),
	// return a diagnostic message rather than an empty string.
	var response string
	if len(generatedTokens) == 0 {
		response = "[No match found — input context not yet in the knowledge base. Run the Data Pipeline (menu option 1) on more data.]"
	} else {
		response = tokenizer.Detokenize(generatedTokens)
	}

	lastTokenID := 0
	if len(generatedTokens) > 0 {
		lastTokenID = generatedTokens[len(generatedTokens)-1]
	}

	latency := time.Since(start)

	c.JSON(http.StatusOK, ChatResponse{
		Response:   response,
		TokenID:    lastTokenID,
		Confidence: 0.8,
		LatencyMs:  float64(latency.Milliseconds()),
		UsingASIC:  usingASIC,
	})
}

// handleCryptoStatus handles crypto-transformer status requests
func (o *Orchestrator) handleCryptoStatus(c *gin.Context) {
	if o.cryptoModel == nil {
		c.JSON(http.StatusOK, gin.H{
			"enabled": false,
			"reason":  "crypto-transformer not enabled via --crypto flag",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled":        true,
		"vocab_size":     o.cryptoModel.Config.VocabSize,
		"embedding_dim":  o.cryptoModel.Config.EmbedDim,
		"num_layers":     o.cryptoModel.Config.NumLayers,
		"num_heads":      o.cryptoModel.Config.NumHeads,
		"ffn_hidden_dim": o.cryptoModel.Config.FFNHiddenDim,
		"activation":     o.cryptoModel.Config.Activation,
		"using_asic":     o.safeIsUsingHardware(),
	})
}

// handleListRules handles GET /rules requests
func (o *Orchestrator) handleListRules(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"rules": []interface{}{}, // Empty for now - could be extended to store rules
		"total": 0,
	})
}

// handleAddRule handles POST /rules requests
func (o *Orchestrator) handleAddRule(c *gin.Context) {
	var req RuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate rule type
	if req.RuleType != "constraint" && req.RuleType != "subsumption" && req.RuleType != "disjoint" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule type. Must be: constraint, subsumption, or disjoint"})
		return
	}

	// For now, just acknowledge the rule (in production, would store in database)
	c.JSON(http.StatusOK, RuleResponse{
		Success: true,
		Message: fmt.Sprintf("Rule added successfully to domain '%s'", req.Domain),
		RuleID:  1, // Placeholder ID
	})
}

// handleDeleteRule handles DELETE /rules/:id requests
func (o *Orchestrator) handleDeleteRule(c *gin.Context) {
	ruleID := c.Param("id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rule ID required"})
		return
	}

	// For now, just acknowledge deletion (in production, would remove from database)
	c.JSON(http.StatusOK, RuleResponse{
		Success: true,
		Message: fmt.Sprintf("Rule '%s' deleted successfully", ruleID),
	})
}

// handleListDomains handles GET /domains requests
func (o *Orchestrator) handleListDomains(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"domains": []interface{}{}, // Empty for now - could be extended to store domains
		"total":   0,
	})
}

// handleSubmitProof handles POST /v1/verify/proof for formal proof submission.
func (o *Orchestrator) handleSubmitProof(c *gin.Context) {
	if !o.formalVerifierEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "formal verifier not enabled"})
		return
	}

	var req struct {
		CanonicalProofAsset     []byte `json:"canonical_proof_asset"`
		RequestHardwareAttestation bool `json:"request_hardware_attestation"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}
	if len(req.CanonicalProofAsset) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "canonical_proof_asset is required"})
		return
	}

	worker, ok := o.leanWorker.(*lean.Worker)
	if !ok || worker == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lean worker not initialized"})
		return
	}

	// Deserialize the proof asset.
	var asset proofasset.ProofAsset
	if err := json.Unmarshal(req.CanonicalProofAsset, &asset); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proof asset JSON: " + err.Error()})
		return
	}

	// Run precheck before submitting to the expensive checker.
	if err := proofasset.ValidateProofAsset(&asset, []string{
		"Mathlib.Algebra.Group.Basic",
		"Mathlib.Data.Real.Basic",
	}); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"proof_asset_id": "",
			"status":         proofasset.StatusStructurallyRejected,
			"diagnostic":     fmt.Sprintf("precheck failed: %v", err),
		})
		return
	}

	result, err := worker.SubmitProof(&asset)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"proof_asset_id": "",
			"status":         proofasset.StatusCheckerUnavailable,
			"diagnostic":     fmt.Sprintf("checker error: %v", err),
		})
		return
	}

	proofAssetID, _ := proofasset.ComputeProofAssetID(&asset)
	response := gin.H{
		"proof_asset_id": proofAssetID,
		"status":         result.Receipt.Status,
		"diagnostic":     result.Diagnostic,
	}

	if result.Receipt != nil {
		response["receipt"] = result.Receipt
	}

	if req.RequestHardwareAttestation && result.Receipt.Status == proofasset.StatusFormallyVerified {
		response["hardware_attestation"] = gin.H{
			"status": proofasset.StatusAttestationPending,
			"note":   "hardware attestation queued for verified proof asset",
		}
	}

	c.JSON(http.StatusOK, response)
}

// handleGetProofStatus handles GET /v1/verify/proof/:id for proof status polling.
func (o *Orchestrator) handleGetProofStatus(c *gin.Context) {
	proofAssetID := c.Param("id")
	if proofAssetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "proof asset ID is required"})
		return
	}

	if o.proofLedgerPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "proof ledger not configured"})
		return
	}

	ledger := proofasset.NewProofLedger(o.proofLedgerPath)
	entries, err := ledger.ReadAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read proof ledger: " + err.Error()})
		return
	}

	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].ProofAssetID == proofAssetID {
			c.JSON(http.StatusOK, gin.H{
				"proof_asset_id": entries[i].ProofAssetID,
				"status":         entries[i].FinalStatus,
				"receipt":        entries[i].Receipt,
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "proof asset not found"})
}

// handleGetProofAsset handles GET /v1/verify/proof/:id/asset for retrieving stored proof assets.
func (o *Orchestrator) handleGetProofAsset(c *gin.Context) {
	proofAssetID := c.Param("id")
	if proofAssetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "proof asset ID is required"})
		return
	}

	if o.proofLedgerPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "proof ledger not configured"})
		return
	}

	ledger := proofasset.NewProofLedger(o.proofLedgerPath)
	entries, err := ledger.ReadAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read proof ledger: " + err.Error()})
		return
	}

	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].ProofAssetID == proofAssetID {
			c.JSON(http.StatusOK, gin.H{
				"proof_asset_id":   entries[i].ProofAssetID,
				"canonical_asset":  entries[i].CanonicalAssetPath,
				"canonical_digest": entries[i].CanonicalAssetDigest,
				"status":           entries[i].FinalStatus,
				"receipt":          entries[i].Receipt,
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "proof asset not found"})
}

// handleReplayProof handles POST /v1/verify/proof/:id/replay for independent
// replay of formal verification on a clean node.
func (o *Orchestrator) handleReplayProof(c *gin.Context) {
	proofAssetID := c.Param("id")
	if proofAssetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "proof asset ID is required"})
		return
	}

	if !o.formalVerifierEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "formal verifier not enabled"})
		return
	}

	ledger := proofasset.NewProofLedger(o.proofLedgerPath)
	entries, err := ledger.ReadAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read proof ledger: " + err.Error()})
		return
	}

	var targetEntry *proofasset.ProofLedgerEntry
	for i := range entries {
		if entries[i].ProofAssetID == proofAssetID {
			cp := entries[i]
			targetEntry = &cp
			break
		}
	}

	if targetEntry == nil || targetEntry.Receipt == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "proof asset or receipt not found"})
		return
	}

	worker, ok := o.leanWorker.(*lean.Worker)
	if !ok || worker == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lean worker not initialized"})
		return
	}

	var asset proofasset.ProofAsset
	canonicalPath := targetEntry.CanonicalAssetPath
	if canonicalPath != "" {
		data, err := os.ReadFile(canonicalPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read canonical asset: " + err.Error()})
			return
		}
		if err := json.Unmarshal(data, &asset); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unmarshal canonical asset: " + err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "canonical asset path not recorded in ledger"})
		return
	}

	result, err := worker.SubmitProof(&asset)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"proof_asset_id": proofAssetID,
			"status":         proofasset.StatusCheckerUnavailable,
			"diagnostic":     fmt.Sprintf("replay checker error: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"proof_asset_id": proofAssetID,
		"original_status": targetEntry.Receipt.Status,
		"replay_status":   result.Receipt.Status,
		"replay_diagnostic": result.Diagnostic,
		"match":           targetEntry.Receipt.Status == result.Receipt.Status,
	})
}

// handleAttestProof handles POST /v1/verify/proof/:id/attest for ASIC
// attestation of verified proof assets.
func (o *Orchestrator) handleAttestProof(c *gin.Context) {
	proofAssetID := c.Param("id")
	if proofAssetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "proof asset ID is required"})
		return
	}

	if !o.formalVerifierEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "formal verifier not enabled"})
		return
	}

	ledger := proofasset.NewProofLedger(o.proofLedgerPath)
	entries, err := ledger.ReadAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read proof ledger: " + err.Error()})
		return
	}

	var targetEntry *proofasset.ProofLedgerEntry
	for i := range entries {
		if entries[i].ProofAssetID == proofAssetID {
			cp := entries[i]
			targetEntry = &cp
			break
		}
	}

	if targetEntry == nil || targetEntry.Receipt == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "proof asset or receipt not found"})
		return
	}

	if targetEntry.Receipt.Status != proofasset.StatusFormallyVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only formally verified assets can be attested"})
		return
	}

	attestSvc := proofasset.NewAsicAttestationService(true)
	result := attestSvc.ComputeAttestation(proofasset.AttestationRequest{
		ProofAssetID:     proofAssetID,
		Receipt:          targetEntry.Receipt,
		HeaderVersion:    1,
		Target:           0x10000,
		DeviceFirmware:   "software-fallback",
		AsicSlots:        [12]uint32{},
		NonceStart:       0,
		NonceEnd:         0xFFFF,
	})

	if !result.Attested {
		c.JSON(http.StatusOK, gin.H{
			"proof_asset_id": proofAssetID,
			"status":         proofasset.StatusAttestationPending,
			"diagnostic":     result.Diagnostic,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"proof_asset_id": proofAssetID,
		"status":         proofasset.StatusHardwareAttested,
		"attestation":    result.Record,
		"diagnostic":     result.Diagnostic,
	})
}

// handleFormalVerificationMetrics handles GET /v1/verify/proof/metrics for
// formal verification queue and execution metrics.
func (o *Orchestrator) handleFormalVerificationMetrics(c *gin.Context) {
	if !o.formalVerifierEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "formal verifier not enabled"})
		return
	}

	grpcSrv := o.formalVerifierGRPCServer
	if grpcSrv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "gRPC formal verification server not initialized"})
		return
	}

	resp, err := grpcSrv.Metrics(c, &pb.MetricsRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("metrics request failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_submissions":     resp.TotalSubmissions,
		"verified_count":        resp.VerifiedCount,
		"rejected_count":        resp.RejectedCount,
		"error_count":           resp.ErrorCount,
		"queue_depth":           resp.QueueDepth,
		"median_check_time_ms":  resp.MedianCheckTimeMs,
	})
}

// handleDiscovery handles GET /discovery requests
func (o *Orchestrator) handleDiscovery(c *gin.Context) {
	if o.discoveryResult == nil {
		c.JSON(http.StatusOK, gin.H{
			"discovered": false,
			"message":    "No network discovery performed or no servers found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"discovered": true,
		"server":     o.discoveryResult,
	})
}

// handleDiscoveryScan handles POST /discovery/scan requests
func (o *Orchestrator) handleDiscoveryScan(c *gin.Context) {
	// Parse request parameters
	type DiscoveryRequest struct {
		Subnet    string `json:"subnet,omitempty"`
		Port      int    `json:"port,omitempty"`
		TimeoutMs int64  `json:"timeout_ms,omitempty"`
		SkipLocal bool   `json:"skip_localhost,omitempty"`
	}

	var req DiscoveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Create discovery config
	config := discovery.NewDiscoveryConfig()
	if req.Subnet != "" {
		config.Subnet = req.Subnet
	}
	if req.Port > 0 {
		config.Port = req.Port
	}
	if req.TimeoutMs > 0 {
		config.Timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}
	config.SkipLocalhost = req.SkipLocal

	// Perform discovery
	discoveries, err := discovery.DiscoverServers(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Find best server
	best := discovery.FindBestServer(discoveries)

	c.JSON(http.StatusOK, gin.H{
		"discoveries": discoveries,
		"best_server": best,
		"total_found": len(discoveries),
		"responding": func() int {
			count := 0
			for _, d := range discoveries {
				if d.Responding {
					count++
				}
			}
			return count
		}(),
	})
}

// CLI Modes (kept from original implementation)

func runSingleMode(orch *Orchestrator) {
	log.Printf("Computing %d single hashes...", *count)

	if orch.asicClient == nil {
		log.Printf("Warning: ASIC client not available - cannot run single mode")
		log.Printf("Please start hasher-host with --api flag or ensure an ASIC device is connected")
		return
	}

	totalLatency := time.Duration(0)

	for i := 0; i < *count; i++ {
		data := randomData(*dataSize)

		start := time.Now()
		hash, err := orch.asicClient.ComputeHash(data)
		latency := time.Since(start)

		if err != nil {
			log.Fatalf("ComputeHash failed: %v", err)
		}

		totalLatency += latency

		if i == 0 {
			log.Printf("Hash #%d: %x (latency: %v)", i+1, hash[:8], latency)
		}
	}

	avgLatency := totalLatency / time.Duration(*count)
	log.Printf("Completed %d hashes, average latency: %v", *count, avgLatency)
}

func runBatchMode(orch *Orchestrator) {
	log.Printf("Computing %d hashes with batch size %d...", *count, *batchSize)

	if orch.asicClient == nil {
		log.Printf("Warning: ASIC client not available - cannot run batch mode")
		log.Printf("Please start hasher-host with --api flag or ensure an ASIC device is connected")
		return
	}

	// Prepare all data
	data := make([][]byte, *count)
	for i := 0; i < *count; i++ {
		data[i] = randomData(*dataSize)
	}

	start := time.Now()
	var totalHashes [][32]byte

	// Process in batches if batchSize is specified and smaller than total count
	if *batchSize > 0 && *batchSize < *count {
		for i := 0; i < *count; i += *batchSize {
			end := i + *batchSize
			if end > *count {
				end = *count
			}
			batch := data[i:end]

			log.Printf("Processing batch %d-%d...", i, end-1)
			hashes, err := orch.asicClient.ComputeBatch(batch)
			if err != nil {
				log.Fatalf("ComputeBatch failed at batch %d-%d: %v", i, end-1, err)
			}
			totalHashes = append(totalHashes, hashes...)
		}
	} else {
		// Single batch
		hashes, err := orch.asicClient.ComputeBatch(data)
		if err != nil {
			log.Fatalf("ComputeBatch failed: %v", err)
		}
		totalHashes = hashes
	}

	elapsed := time.Since(start)

	log.Printf("Computed %d hashes in %v", len(totalHashes), elapsed)
	log.Printf("Throughput: %.2f hashes/sec", float64(len(totalHashes))/elapsed.Seconds())

	if len(totalHashes) > 0 {
		log.Printf("First hash: %x", totalHashes[0][:8])
	}
}

func runStreamMode(orch *Orchestrator) {
	log.Printf("Streaming %d hashes...", *count)

	if orch.asicClient == nil {
		log.Printf("Warning: ASIC client not available - cannot run stream mode")
		log.Printf("Please start hasher-host with --api flag or ensure an ASIC device is connected")
		return
	}

	// Prepare data
	data := make([][]byte, *count)
	for i := 0; i < *count; i++ {
		data[i] = randomData(*dataSize)
	}

	var received int
	var totalLatency time.Duration
	start := time.Now()

	// Define callback for streaming results
	callback := func(requestID uint64, hash [32]byte, latencyUs uint64) {
		received++
		totalLatency += time.Duration(latencyUs) * time.Microsecond

		if received == 1 || received%100 == 0 {
			log.Printf("Received hash #%d: %x (latency: %dus)",
				requestID, hash[:8], latencyUs)
		}
	}

	// Perform streaming computation
	err := orch.asicClient.StreamCompute(data, callback)
	if err != nil {
		log.Fatalf("StreamCompute failed: %v", err)
	}

	elapsed := time.Since(start)
	avgLatency := time.Duration(0)
	if received > 0 {
		avgLatency = totalLatency / time.Duration(received)
	}

	log.Printf("Streamed %d hashes in %v", received, elapsed)
	log.Printf("Average latency: %v", avgLatency)
	log.Printf("Throughput: %.2f hashes/sec", float64(received)/elapsed.Seconds())
}

func showMetrics(orch *Orchestrator) {
	if orch.asicClient == nil {
		log.Printf("Warning: ASIC client not available")
		fmt.Println("\n=== Hasher Metrics ===")
		fmt.Println("ASIC client not available - running in software fallback mode")
		fmt.Println("Metrics are limited when running without ASIC hardware")
		return
	}

	resp, err := orch.asicClient.GetMetrics()
	if err != nil {
		log.Printf("GetMetrics failed (may be in software fallback mode): %v", err)
		fmt.Println("\n=== Hasher Metrics ===")
		fmt.Println("Metrics not available in software fallback mode")
		return
	}

	fmt.Println("\n=== Hasher Metrics ===")
	fmt.Printf("Total Requests:       %d\n", resp.TotalRequests)
	fmt.Printf("Total Bytes Processed: %d (%.2f MB)\n",
		resp.TotalBytesProcessed,
		float64(resp.TotalBytesProcessed)/1024/1024)
	fmt.Printf("Average Latency:      %d µs\n", resp.AverageLatencyUs)
	fmt.Printf("Peak Latency:         %d µs\n", resp.PeakLatencyUs)
	fmt.Printf("Total Errors:         %d\n", resp.TotalErrors)
	fmt.Printf("Cache Hits:           %d\n", resp.CacheHits)
	fmt.Printf("Cache Misses:         %d\n", resp.CacheMisses)

	if len(resp.DeviceStats) > 0 {
		fmt.Println("\nDevice Stats:")
		for k, v := range resp.DeviceStats {
			fmt.Printf("  %s: %d\n", k, v)
		}
	}
}

func showInfo(orch *Orchestrator) {
	if orch.asicClient == nil {
		log.Printf("Warning: ASIC client not available")
		fmt.Println("\n=== Device Info ===")
		fmt.Println("Device Path:      software")
		fmt.Println("Chip Count:       0")
		fmt.Println("Firmware Version: software-fallback")
		fmt.Println("Operational:      true")
		fmt.Println("Uptime:           N/A")
		fmt.Println("\n=== Orchestrator Info ===")
		fmt.Printf("Using ASIC:       %v\n", orch.engine.IsUsingHardware())
		fmt.Printf("Network:          [%d, %d, %d, %d]\n", *inputSize, *hidden1, *hidden2, *outputSize)
		fmt.Printf("Passes:           %d\n", *passes)
		fmt.Printf("Jitter:           %.3f\n", *jitter)
	fmt.Printf("Seed Rotation:    %v\n", *seedRotation)
	fmt.Printf("Direct ASIC Mode: %v\n", *directMode)
	fmt.Println("\nNote: Running in software fallback mode without ASIC hardware")
		return
	}

	resp, err := orch.asicClient.GetDeviceInfo()
	if err != nil {
		log.Fatalf("GetDeviceInfo failed: %v", err)
	}

	fmt.Println("\n=== Device Info ===")
	fmt.Printf("Device Path:      %s\n", resp.DevicePath)
	fmt.Printf("Chip Count:       %d\n", resp.ChipCount)
	fmt.Printf("Firmware Version: %s\n", resp.FirmwareVersion)
	fmt.Printf("Operational:      %v\n", resp.IsOperational)
	fmt.Printf("Uptime:           %d seconds (%.1f hours)\n",
		resp.UptimeSeconds,
		float64(resp.UptimeSeconds)/3600)

	// Also show orchestrator info
	fmt.Println("\n=== Orchestrator Info ===")
	fmt.Printf("Using ASIC:       %v\n", orch.engine.IsUsingHardware())
	fmt.Printf("Network:          [%d, %d, %d, %d]\n", *inputSize, *hidden1, *hidden2, *outputSize)
	fmt.Printf("Passes:           %d\n", *passes)
	fmt.Printf("Jitter:           %.3f\n", *jitter)
	fmt.Printf("Seed Rotation:    %v\n", *seedRotation)
	fmt.Printf("Direct ASIC Mode: %v\n", *directMode)
}

// findOpenPort finds an available port starting from the given port
func findOpenPort(startPort int) (int, error) {
	if startPort > 0 {
		// Check if the specified port is available
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", startPort))
		if err == nil {
			listener.Close()
			return startPort, nil
		}
		log.Printf("Port %d not available: %v", startPort, err)
	}

	// Find an available port starting from 8080
	for port := 8080; port <= 9090; port++ {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			listener.Close()
			log.Printf("Found available port: %d", port)
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports found in range 8080-9090")
}

func randomData(size int) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}

// Utility function to pretty print JSON
func prettyJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
