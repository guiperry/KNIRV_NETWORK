// node_lifecycle_test.go
package main

import (
	"KNIRVROOT/config" // Adjust import path
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// WaitForNode is imported from test_helper.go

// startTestNodeProcess: Starts a node process (root or node) with specific flags.
// Returns the command and a cleanup function.
// The caller should determine the expected URL based on the node's configuration.
func startTestNodeProcess(t *testing.T, mode string, dbPath string, extraArgs ...string) (*exec.Cmd, func()) {
	t.Helper()

	args := []string{"run", "."} // Assuming running from project root

	// --- Mode Specific Flags ---
	IsPeerMode := false
	switch mode {
	case "network":
		args = append(args, "-network")
		// Network mode (root) might need port/p2p flags if not using config/defaults
		// Add them here if necessary, potentially via extraArgs or specific params
	case "agent":
		args = append(args, "-agent")
		IsPeerMode = true
		// Peer mode ignores -port, -p2p.port, -shared_database_path. Relies on -config.
	default:
		// single node mode potentially
		// Add flags as needed
	}

	// --- Add Common/Extra Args ---
	// Only add shared_database_path if NOT in node mode
	if !IsPeerMode {
		args = append(args, "--shared_database_path", dbPath)
	}
	// Always add --no-wallet-server for testing consistency
	args = append(args, "--no-wallet-server")

	// Add any other specific args passed in
	args = append(args, extraArgs...)

	// --- Determine Logged Ports (for info only, not necessarily used by process) ---
	// Try to find port/p2p port in extraArgs for logging purposes
	logPort := "default"
	logP2pPort := "default"
	for i, arg := range extraArgs {
		if (arg == "--port" || arg == "-port") && i+1 < len(extraArgs) {
			logPort = extraArgs[i+1]
		}
		if (arg == "--p2p.port" || arg == "-p2p.port") && i+1 < len(extraArgs) {
			logP2pPort = extraArgs[i+1]
		}
		// If node mode, override with port settings from config if possible (complex to parse here)
		// For simplicity, we'll rely on the test log for the *expected* node ports.
	}

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout // Pipe output for debugging
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start node process (mode: %s, args: %v): %v", mode, args, err)
	}
	t.Logf("Started node process (mode: %s, pid: %d) with args: %v", mode, cmd.Process.Pid, args)
	t.Logf("  (Note: Logged ports %s/%s might be ignored by process depending on mode)", logPort, logP2pPort)

	cleanup := func() {
		// ... (cleanup logic remains the same) ...
		pid := cmd.Process.Pid
		t.Logf("Cleaning up node process group (mode: %s, pid: %d)...", mode, pid)
		if pid > 0 {
			pgid := -pid
			if err := syscall.Kill(pgid, syscall.SIGKILL); err != nil {
				if !strings.Contains(err.Error(), "process already finished") && !strings.Contains(err.Error(), "no such process") {
					t.Logf("Warning: Failed to kill process group %d: %v", pgid, err)
				}
			} else {
				t.Logf("Sent SIGKILL to process group %d.", pgid)
			}
		}
		if cmd.Process != nil {
			_ = cmd.Wait()
		}
		t.Logf("Node process group cleanup attempt complete for pid %d.", pid)
	}

	// Return only cmd and cleanup, caller determines expected URL
	return cmd, cleanup
}

// simulateInstall performs the core installation steps without interactive prompts.
// The nodeHTTPPort and nodeP2PPort parameters are used to set the regular HTTP and P2P ports
// for the node node, not reflection-specific ports.
func simulateInstall(t *testing.T, nodeConfigPath, rootNodeURL, desiredURI string, nodeHTTPPort, nodeP2PPort uint64) (string, error) {
	t.Helper()
	t.Logf("Simulating installation for config: %s", nodeConfigPath)

	// 1. Load initial/default config (to preserve root settings if any were copied)
	currentCfg, _, err := config.LoadConfig(nodeConfigPath, config.RolePeer)
	if err != nil {
		// If it doesn't exist, LoadConfig should create a default one.
		// If that fails, we have a problem.
		return "", fmt.Errorf("failed to load/create initial node config at %s: %w", nodeConfigPath, err)
	}

	// 2. Generate Chain URI via Root Node
	t.Logf("Requesting Chain ID from root: %s (Desired: '%s')", rootNodeURL, desiredURI)
	// Note: GenerateChainURI now returns (extractedID, fullURI, txnHash, error)
	generatedChainID, _, _, err := GenerateChainURI(rootNodeURL, desiredURI)
	if err != nil {
		return "", fmt.Errorf("failed to generate chain URI from root: %w", err)
	}
	if generatedChainID == "" {
		// Fallback if GenerateChainURI doesn't return ID directly
		// This part depends heavily on your GenerateChainURI implementation and response format
		return "", fmt.Errorf("GenerateChainURI did not return a valid ChainID")
	}
	t.Logf("Successfully generated ChainID: %s", generatedChainID)

	// 3. Prepare the config to save
	configToSave := currentCfg
	configToSave.ChainID = generatedChainID
	// Set the regular HTTP and P2P ports for the node node
	configToSave.Port = nodeHTTPPort
	configToSave.P2PPort = nodeP2PPort
	configToSave.InstallComplete = true
	// Ensure other node-specific defaults are set if needed
	configToSave.NoWalletServer = true
	configToSave.IsPeer = true
	configToSave.ClientOnly = false

	// Set the node database paths (used by tests to isolate node DBs)
	tempDir := t.TempDir()
	configToSave.BlockchainDatabasePath = filepath.Join(tempDir, "blockchain.db")
	configToSave.SearchableDatabasePath = filepath.Join(tempDir, "searchable.db")

	// 4. Save the updated configuration
	t.Logf("Saving updated node configuration to: %s", nodeConfigPath)
	_, err = config.SaveConfig(nodeConfigPath, configToSave)
	if err != nil {
		return "", fmt.Errorf("failed to save updated node config: %w", err)
	}
	t.Log("Peer configuration saved successfully.")

	return generatedChainID, nil
}

// findPeerInDHT uses the root node's /nodes endpoint to check for the node.
func findPeerInDHT(t *testing.T, rootNodeURL, nodeChainID string, expectedPeerP2PPort int) (peer.ID, bool) {
	t.Helper()
	client := &http.Client{Timeout: 60 * time.Second}
	// Assuming /nodes endpoint implicitly uses the node's own ChainID from its config.
	// If /nodes needs the target ChainID, adjust the URL.
	nodesURL := fmt.Sprintf("%s/nodes?chainID=%s", rootNodeURL, nodeChainID)

	t.Logf("Querying root node %s for nodes providing resource '%s.chain'", rootNodeURL, nodeChainID)

	// Retry mechanism as DHT propagation takes time
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second) // Overall timeout
	defer cancel()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Logf("Timeout waiting to find node %s in DHT via root node %s", nodeChainID, rootNodeURL)
			return "", false
		case <-ticker.C:
			resp, err := client.Get(nodesURL)
			if err != nil {
				t.Logf("Error querying /nodes endpoint: %v", err)
				continue // Retry
			}

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				t.Logf("/nodes endpoint returned non-OK status: %d, body: %s", resp.StatusCode, string(bodyBytes))
				continue // Retry
			}

			var nodeAddrs []string
			if err := json.NewDecoder(resp.Body).Decode(&nodeAddrs); err != nil {
				resp.Body.Close()
				t.Logf("Error decoding /nodes response: %v", err)
				continue // Retry
			}
			resp.Body.Close()

			t.Logf("Received %d multiaddrs from /nodes endpoint", len(nodeAddrs))

			// Check if any of the returned multiaddrs belong to the node and have the correct port
			for _, addrStr := range nodeAddrs {
				ma, err := multiaddr.NewMultiaddr(addrStr)
				if err != nil {
					continue // Ignore invalid multiaddrs
				}

				nodeIDStr, err := ma.ValueForProtocol(multiaddr.P_P2P)
				if err != nil {
					continue // Not a p2p multiaddr
				}
				nodeID, err := peer.Decode(nodeIDStr)
				if err != nil {
					continue
				}

				// Check if the address contains the expected P2P port
				portStr, err := ma.ValueForProtocol(multiaddr.P_TCP)
				if err == nil && portStr == fmt.Sprintf("%d", expectedPeerP2PPort) {
					// Found the node with the correct port!
					t.Logf("Found node %s in DHT with expected P2P port %d at address %s", nodeID.String(), expectedPeerP2PPort, addrStr)
					return nodeID, true
				}
			}
			t.Logf("Peer %s not found with port %d in current node list, retrying...", nodeChainID, expectedPeerP2PPort)
		}
	}
}

// --- Main Test Function ---

func TestPeerLifecycle_Integration(t *testing.T) {
	// --- Test Setup ---
	t.Log("Setting up node lifecycle integration test...")

	// Root Node Configuration
	rootDir, err := os.MkdirTemp("", "agent-root-test-*")
	if err != nil {
		t.Fatalf("Failed to create root temp dir: %v", err)
	}
	defer os.RemoveAll(rootDir)
	rootConfigPath := filepath.Join(rootDir, "config.json")
	rootHttpPort := 5050 // Use distinct ports for tests
	rootP2pPort := 4050

	// Create root config with unique database paths
	rootCfg := config.DefaultConfig()
	rootCfg.InstallComplete = true
	rootCfg.Port = uint64(rootHttpPort)
	rootCfg.P2PPort = uint64(rootP2pPort)
	rootCfg.BlockchainDatabasePath = filepath.Join(rootDir, "blockchain.db")
	rootCfg.SearchableDatabasePath = filepath.Join(rootDir, "searchable.db")
	if _, err := config.SaveConfig(rootConfigPath, rootCfg); err != nil {
		t.Fatalf("Failed to create root config: %v", err)
	}

	// Peer Node Configuration
	nodeDir, err := os.MkdirTemp("", "agent-node-test-*")
	if err != nil {
		t.Fatalf("Failed to create node temp dir: %v", err)
	}
	defer os.RemoveAll(nodeDir)
	nodeConfigPath := filepath.Join(nodeDir, "config.json")
	nodeHttpPort := uint64(6050) // Ports the node *should* use
	nodeP2pPort := uint64(7050)

	// Create initial node config with install_complete: false
	initialPeerCfg := config.DefaultConfig()
	initialPeerCfg.InstallComplete = false
	if _, err := config.SaveConfig(nodeConfigPath, initialPeerCfg); err != nil {
		t.Fatalf("Failed to create initial node config: %v", err)
	}

	// Start Root Node (in network mode to have DHT running)
	t.Logf("Starting Root Node (Network Mode) on HTTP:%d, P2P:%d", rootHttpPort, rootP2pPort)
	// Pass root-specific flags including config path
	rootArgs := []string{
		"--config", rootConfigPath,
	}
	_, rootCleanup := startTestNodeProcess(t, "network", "", rootArgs...)
	defer rootCleanup()
	rootNodeURL := fmt.Sprintf("http://localhost:%d", rootHttpPort) // Construct root URL
	config.WaitForNode(t, rootNodeURL, 30*time.Second)              // Wait for root node
	t.Log("Root node is healthy.")

	// --- Step 1: Simulate Installation ---
	randSuffix := fmt.Sprintf("%d", rand.Intn(10000))
	desiredPeerID := "test-node-" + randSuffix
	nodeChainID, err := simulateInstall(t, nodeConfigPath, rootNodeURL, desiredPeerID, nodeHttpPort, nodeP2pPort)
	if err != nil {
		t.Fatalf("Peer installation simulation failed: %v", err)
	}
	// ... (config verification remains the same) ...

	// --- Step 2: Start Peer Node ---
	t.Logf("Starting Peer Node with ChainID '%s' (Expected HTTP:%d, P2P:%d)", nodeChainID, nodeHttpPort, nodeP2pPort)
	// Only pass -node and -config flags. main.go will read ports/db from config.
	nodeArgs := []string{"-config", nodeConfigPath}
	// Ensure the node database directory exists before starting
	if err := os.MkdirAll(filepath.Dir(initialPeerCfg.SearchableDatabasePath), 0750); err != nil {
		t.Fatalf("Failed to create node database directory: %v", err)
	}
	// The dbPath argument to startTestNodeProcess is ignored in node mode.
	_, nodeCleanup := startTestNodeProcess(t, "node", "" /* dbPath ignored */, nodeArgs...)
	defer nodeCleanup()

	// Construct the URL the node *should* be listening on based on its config
	expectedPeerURL := fmt.Sprintf("http://localhost:%d", nodeHttpPort)
	config.WaitForNode(t, expectedPeerURL, 30*time.Second) // Wait for node node on its correct port
	t.Logf("Peer node started and is healthy at %s", expectedPeerURL)

	// --- Step 3: Verify DHT Announcement ---
	t.Logf("Verifying node '%s' announcement in DHT via root node...", nodeChainID)
	foundPeerID, found := findPeerInDHT(t, rootNodeURL, nodeChainID, int(nodeP2pPort))
	if !found {
		t.Errorf("Failed to find node '%s' with P2P port %d announced in DHT", nodeChainID, nodeP2pPort)
	} else {
		t.Logf("Successfully verified node '%s' (PeerID: %s) announced in DHT with correct port.", nodeChainID, foundPeerID.String())
	}

	// --- Step 4: Cleanup (Deferred) ---
	t.Log("Peer lifecycle test completed successfully. Cleanup will occur.")
}
