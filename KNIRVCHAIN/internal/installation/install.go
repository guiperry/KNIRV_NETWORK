package installation

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/pion/stun"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"KNIRVCHAIN/config"
	"KNIRVCHAIN/internal/uri"
	"KNIRVCHAIN/internal/utils"
	"KNIRVCHAIN/internal/wallet"
)

var walletManager *wallet.WalletManagerImpl

// BootnodeRegistryURL is the default registry URL for bootnode registration
const BootnodeRegistryURL = "https://registry.knirv.com/api/bootnodes"

// DiscoveryManager is a minimal discovery manager for installation
type DiscoveryManager struct {
	host interface{} // libp2p host
}

// ID returns a mock node ID for installation purposes
func (dm *DiscoveryManager) ID() string {
	if dm.host != nil {
		// Try to get ID from libp2p host if available
		if h, ok := dm.host.(interface {
			ID() interface{ String() string }
		}); ok {
			return h.ID().String()
		}
	}
	return "mock-node-id-for-installation"
}

// RegisterWithRegistry registers a bootnode with the KNIRV registry
func RegisterWithRegistry(registryURL, chainID string, port int, ip, nodeID string) error {
	// Create registration payload
	payload := map[string]interface{}{
		"chain_id": chainID,
		"port":     port,
		"ip":       ip,
		"node_id":  nodeID,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal registration payload: %w", err)
	}

	// Make HTTP request
	resp, err := http.Post(registryURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to make registration request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed with status: %d", resp.StatusCode)
	}

	return nil
}

// GetDiscoveryManager creates a minimal DiscoveryManager for installation purposes
func GetDiscoveryManager() *DiscoveryManager {
	// Create basic libp2p host
	h, err := libp2p.New()
	if err != nil {
		log.Printf("Failed to create libp2p host: %v", err)
		return nil
	}

	return &DiscoveryManager{
		host: h,
	}
}

func init() {
	var err error
	// Use the deterministic encryption key for consistency
	encryptionKey := utils.DeriveEncryptionKey() // Defined in crypto_utils.go
	walletManager, err = wallet.NewWalletManager(encryptionKey)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize wallet manager: %v", err))
	}
}

// URIResponse represents the response from the KNIRVCHAIN Root's /uriGenerator endpoint
type URIResponse struct {
	TxnHash string `json:"txn_hash"`
	URI     string `json:"uri"`
}

// StunInfoResponse matches the JSON structure from the /stun endpoint
type StunInfoResponse struct {
	Protocols struct {
		UDP struct {
			Enabled bool   `json:"enabled"`
			Port    int    `json:"port"`
			Address string `json:"address"`
		} `json:"udp"`
		TCP struct {
			Enabled bool   `json:"enabled"`
			Port    int    `json:"port"`
			Address string `json:"address"`
		} `json:"tcp"`
	} `json:"protocols"`
	ConnectionStrings struct {
		UDP     string `json:"udp"`
		TCP     string `json:"tcp"`
		TurnUDP string `json:"turn_udp"`
		TurnTCP string `json:"turn_tcp"`
	} `json:"connectionStrings"`
	ServerSoftware string `json:"serverSoftware"`
}

// ConfigData represents the configuration data to be saved to .env file
type ConfigData map[string]string

func Install(configPath string, IsBootnode bool, role config.Role, nonInteractive bool, walletPath string) (*config.Config, error) {
	// Load existing config first
	currentCfg, _, err := config.LoadConfig(configPath, role)
	if err != nil {
		log.Printf("Warning: Could not load existing config.json: %v. Using default ports for prompts.", err)
		currentCfg = config.DefaultConfig() // Use defaults if load fails
	}

	// Use the provided role parameter directly
	// Role is now determined at compile time or via command-line flags
	fmt.Printf("Using role %s (bootnode=%v)\n", role.String(), IsBootnode)

	// Check if config exists and has InstallComplete=true
	if currentCfg != nil && currentCfg.InstallComplete {
		fmt.Println("Configuration already marked as complete - skipping installation")
		return currentCfg, nil
	}

	// Set bootnode flag in config
	currentCfg.IsBootnode = IsBootnode

	// Set client-only flag based on role
	currentCfg.ClientOnly = (role == config.RoleClient)

	// Set root flag based on role
	currentCfg.IsRoot = (role == config.Root)

	// If role is Root, many installation steps are skipped or simplified.
	if role == config.Root {
		fmt.Println("=== KNIRVCHAIN Root Node Setup ===")
		fmt.Println("Root uses hardcoded identity and minimal configuration.")
		fmt.Println("Root node setup: IP information will be fetched at runtime and stored in root_config.json.")
		// env.local creation for HOST_URL is removed. IP info will be fetched and stored in config.
		// Most steps below will be skipped or no-oped for Root.
	} else {
		fmt.Println("=== KNIRVCHAIN Node Installation ===")
		fmt.Printf("Installing as a %s node\n\n", role.String())
		fmt.Println("This installer will:")

		if role != config.Root {
			fmt.Println("1. Connect to the KNIRVCHAIN Bootnode")
			fmt.Println("2. Generate a unique chain URI for this node")
		}

		if role == config.RoleBootnode {
			fmt.Println("3. Register with KNIRVCHAIN Bootnode Registry")
		}

		if role == config.RoleBootnode || role == config.RolePeer {
			walletMsg := "4. Generate dev wallet"
			if role == config.RoleBootnode {
				walletMsg += " and master wallet"
			}
			fmt.Println(walletMsg)
		}

		fmt.Println("5. Detect host operating system")
		fmt.Println("6. Register URI handler for knirv:// protocol")
		fmt.Println("7. Find next available ports for node")
		fmt.Println("8. Update the application configuration")
		fmt.Println("9. Start the node")
	}
	fmt.Println()

	// Variables for chain identification
	var chainID string  // This will be the node's unique identifier (UUID or Wallet Address)
	var chainURI string // This is the full knirv://<ID>.chain/, only for non-client/non-root
	var hashID string   // Transaction hash from URI generation, only for non-client/non-root

	// Skip URI generation for Root nodes
	if role != config.Root && role != config.RoleClient { // Skip URI generation for Root AND Client roles
		// Step 1: Connect to KNIRVCHAIN Root
		rootEndpoint := promptForRootEndpoint(nonInteractive, currentCfg)

		// Prompt for desired URI (optional)
		desiredURI := promptForDesiredURI(nonInteractive)

		// Step 2: Generate unique chain URI with retry for conflicts
		var uri, hash string
		maxRetries := 5
		retryCount := 0

		for retryCount < maxRetries {
			_, uri, hash, err = GenerateChainURI(rootEndpoint, desiredURI)
			if err != nil {
				// Check if this is a 409 conflict for desired URI
				if strings.Contains(err.Error(), "409") && desiredURI != "" {
					fmt.Printf("\nError: %v\n", err)
					if nonInteractive {
						// In non-interactive mode, just use a random URI instead
						desiredURI = ""
						fmt.Println("Non-interactive mode: Switching to random URI generation")
					} else {
						fmt.Println("The requested URI is not available. Please try a different one.")
						desiredURI = promptForDesiredURI(nonInteractive)
					}
					retryCount++
					continue
				}
				return currentCfg, fmt.Errorf("failed to generate chain URI: %w", err)
			}
			break
		}

		if retryCount >= maxRetries {
			return currentCfg, fmt.Errorf("failed to generate chain URI after %d attempts", maxRetries)
		}

		// Store the full URI and the extracted ID part
		chainURI = uri // full knirv://... URI
		hashID = hash  // txn hash from URI generation

		// Extract the actual ID part from the generated URI
		if strings.HasPrefix(chainURI, "knirv://") {
			parts := strings.Split(strings.TrimPrefix(chainURI, "knirv://"), ".")
			if len(parts) > 0 {
				chainID = parts[0] // This is the <uuid> part
			}
		}
		fmt.Printf("Successfully generated Chain ID: %s (from URI: %s)\n", chainID, chainURI)
		fmt.Printf("Transaction Hash ID for URI: %s\n", hashID)
	} else {
		// For Root or Client roles, URI generation is skipped, chainID is handled differently below
		// For Root nodes, use a predefined URI format
		// The ChainID will be derived from constants or default config in main.go
		// For display purposes here, we can use a placeholder or the default port.
		chainURI = fmt.Sprintf("knirv://KNIRVCHAIN-BOOT:%d.chain/", currentCfg.Port) // Port might be default here
		fmt.Printf("Using Bootnode URI: %s\n", chainURI)
	}
	switch role {
	case config.Root:
		chainID = currentCfg.ChainID // Use existing or default from loaded config
		if chainID == "" {           // Fallback if not set in currentCfg
			chainID = fmt.Sprintf("KNIRVCHAIN-ROOT:%d", currentCfg.Port) // Default for root
		}
		fmt.Printf("Root Node: Using ChainID: %s\n", chainID)
	case config.RoleClient:
		fmt.Println("Client Node: ChainID will be derived from the generated wallet address.")
	default:
		// No special handling needed for other roles
	}

	// Variables for wallets
	var devWallet, masterWallet *wallet.WalletImpl

	if role == config.RoleBootnode || role == config.RolePeer || role == config.RoleClient { // Client now generates a wallet
		fmt.Println("Generating wallet for this node...")
		devWallet, err = wallet.NewWallet()
		if err != nil {
			return currentCfg, fmt.Errorf("failed to generate dev wallet: %w", err)
		}

		// Save dev wallet using wallet manager
		err = walletManager.SaveWallet(devWallet, role)
		if err != nil {
			return currentCfg, fmt.Errorf("failed to save dev wallet: %w", err)
		}

		if role == config.RoleClient {
			chainID = devWallet.GetAddress() // Client's chainID is its wallet address
			fmt.Printf("Client node: Using Wallet Address as ChainID: %s\n", chainID)
		}
		fmt.Printf("Generated Wallet Address: %s\n", devWallet.GetAddress())
		walletPath, err := config.GetPeerWalletPath(role)
		if err == nil {
			fmt.Printf("IMPORTANT: Securely back up the wallet file: %s\n", walletPath)
		}
	}

	// Generate master wallet for Bootnode role ONLY
	if role == config.RoleBootnode {
		fmt.Println("\nGenerating Bootnode Master wallet...")
		masterWallet, err = wallet.NewWallet()
		if err != nil {
			return currentCfg, fmt.Errorf("failed to generate master wallet: %w", err)
		}

		// Save master wallet using wallet manager
		err = walletManager.SaveMasterWallet(masterWallet, role)
		if err != nil {
			return currentCfg, fmt.Errorf("failed to save master wallet: %w", err)
		}

		fmt.Printf("Generated Bootnode Master Wallet Address: %s\n", masterWallet.GetAddress())
		masterPath, err := config.GetMasterWalletPath(role)
		if err == nil {
			fmt.Printf("IMPORTANT: Securely back up the bootnode master wallet file: %s\n", masterPath)
		}
	}

	// For Root role, wallet generation is handled differently (often hardcoded or specific logic in main.go)
	if role == config.Root {
		fmt.Println("Root wallet handling is managed by main application logic, not installer generation.")
	}

	// Step 3: Prompt for Reflection Ports (Suggest defaults offset from root)
	// Suggest ports different from root/default ports
	suggestedReflectionHTTPPort := currentCfg.Port + 1000
	if suggestedReflectionHTTPPort == currentCfg.Port { // Avoid suggesting the same port
		suggestedReflectionHTTPPort++
	}
	suggestedReflectionP2PPort := currentCfg.P2PPort + 1000
	if suggestedReflectionP2PPort == currentCfg.P2PPort { // Avoid suggesting the same port
		suggestedReflectionP2PPort++
	}

	var reflectionHTTPPort, reflectionP2PPort uint64
	if role == config.RolePeer || role == config.RoleBootnode { // Only prompt for these roles
		if nonInteractive {
			reflectionHTTPPort = 5050 // Default HTTP port
			reflectionP2PPort = 6060  // Default P2P port
			fmt.Printf("Non-interactive mode: Using default ports (HTTP: %d, P2P: %d)\n", reflectionHTTPPort, reflectionP2PPort)
		} else {
			fmt.Println("\nConfigure ports for this node:")
			reflectionHTTPPort = promptForUint("Enter Reflection HTTP Port", suggestedReflectionHTTPPort, nonInteractive)
			for !utils.IsPortAvailable(reflectionHTTPPort) {
				fmt.Printf("Port %d is currently in use.\n", reflectionHTTPPort)
				if nonInteractive {
					reflectionHTTPPort++
					fmt.Printf("Non-interactive: Trying next port %d for HTTP.\n", reflectionHTTPPort)
				} else {
					reflectionHTTPPort = promptForUint("Enter a different Reflection HTTP Port", reflectionHTTPPort+1, nonInteractive)
				}
			}

			reflectionP2PPort = promptForUint("Enter Reflection P2P (Libp2p) Port", suggestedReflectionP2PPort, nonInteractive)
			for !utils.IsPortAvailable(reflectionP2PPort) {
				fmt.Printf("Port %d is currently in use.\n", reflectionP2PPort)
				if nonInteractive {
					reflectionP2PPort++
					fmt.Printf("Non-interactive: Trying next port %d for P2P.\n", reflectionP2PPort)
				} else {
					reflectionP2PPort = promptForUint("Enter a different Reflection P2P Port", reflectionP2PPort+1, nonInteractive)
				}
			}
		}
	} else {
		// Root/Client will use defaults from config or constants
	}

	// Step 4: Register with KNIRVCHAIN Node Registry (only for Bootnode role)
	if role == config.RoleBootnode {
		// Use the P2P port for registration, as that's what other nodes will connect to
		p2pPort := fmt.Sprintf("%d", currentCfg.P2PPort)

		// First try to get our public IP via GetHostIP()
		publicIP, err := utils.GetHostIP()
		time.Sleep(5 * time.Second)
		if err != nil || publicIP == "" || publicIP == "localhost" || strings.HasPrefix(publicIP, "127.") {
			log.Printf("Warning: GetHostIP() failed or returned local address (%v). Trying STUN discovery...", err)
			logger := logrus.New()
			stunServer := "registry.knirv.com" // Default STUN server
			publicIP, _, err = DiscoverPublicAddress(stunServer, logger)
			if err != nil || publicIP == "" || publicIP == "localhost" || strings.HasPrefix(publicIP, "127.") {
				log.Printf("Warning: Failed to discover public address via STUN: %v. Falling back to local IP.", err)
				publicIP, err = getLocalIP()
				if err != nil || publicIP == "" {
					log.Printf("Could not determine host IP during installation: %v. Using placeholder or manual input.", err)
					publicIP = "localhost"
				} else {
					log.Printf("Determined local IP for installation: %s", publicIP)
				}
			} else {
				log.Printf("Discovered public IP address via STUN: %s", publicIP)
			}
		} else {
			log.Printf("Determined host IP for installation: %s", publicIP)
		}

		// Get nodeID from DiscoveryManager
		dm := GetDiscoveryManager()
		if dm == nil {
			return nil, fmt.Errorf("discovery manager not initialized")
		}
		nodeID := dm.ID()

		// Register with node registry using public IP and P2P port
		fmt.Println("Registering bootnode with KNIRVCHAIN Node Registry...")
		fmt.Printf("Registering node %s at %s:%s (PeerID: %s) with KNIRVCHAIN Node Registry\n",
			chainID, publicIP, p2pPort, nodeID)
		err = RegisterWithNodeRegistry(chainURI, publicIP, p2pPort, nodeID)
		if err != nil {
			log.Printf("Warning: Failed to register with node registry: %v", err)
			fmt.Println("You can register manually later using the registry API.")
			fmt.Println("The registry status page should open in your browser.")
			fmt.Println("If it doesn't, please visit: https://registry.knirv.com/status")
		} else {
			fmt.Println("Successfully registered with KNIRVCHAIN Node Registry")
		}
	} else if role != config.Root { // Skip for Root as well
		fmt.Println("Skipping registry registration (only required for bootnodes)")
	}

	// Step 4: Detect host operating system
	fmt.Printf("Detected operating system: %s\n", runtime.GOOS)

	// Step 5: Register URI handlers
	// URI handlers might not be relevant for a pure Root node.
	// If they are, the logic can remain. If not, this can be skipped foRoot Root.
	// For now, let's assume it's still desired for all roles that are not Root.
	if role != config.Root {
		if runtime.GOOS == "windows" && !uri.CheckAdminPrivileges() {
			log.Println("--------------------------------------------------------------------")
			log.Println("WARNING: Administrator privileges are required on Windows to register")
			log.Println("         URI handlers automatically. Registration will likely fail.")
			log.Println("         Please re-run the installer as an Administrator, or register")
			log.Println("         the handlers manually after installation.")
			log.Println("--------------------------------------------------------------------")
		}

		err = RegisterURIHandlers(chainURI)
		if err != nil {
			log.Printf("Warning: Failed to register URI handlers: %v", err)
			fmt.Println("You may need to run this installer with administrator/root privileges.")
			fmt.Println("Alternatively, you can manually register the URI handlers later.")
		} else {
			fmt.Println("URI handlers registered successfully.")
		}
	}

	// Step 6: Set up system service
	// System service might not be relevant for a pure Root node.
	// If it is, the logic can remain. If not, this can be skipped for Root.
	if role != config.Root {
		fmt.Println("Setting up system service...")
		_ = fmt.Sprintf("KNIRVCHAIN-%s", strings.ToLower(role.String())) // Used in log messages
		fmt.Printf("Configuring as a %s service...\n", role.String())

		if err := ConfigureSystemService(configPath, currentCfg, role); err != nil {
			log.Printf("Warning: Failed to set up system service: %v", err)
			fmt.Println("You can manually configure the service later.")
		} else {
			fmt.Printf("%s service configured successfully.\n", role.String())
		}
	}

	// Step 7: Update configuration
	// The 'chainID' variable now holds the correct identifier for the node (UUID or wallet address)
	// serviceAddress := chainID // This can be replaced by using chainID directly

	// Prepare config based on role
	configToSave := currentCfg
	configToSave.InstallComplete = true

	// Set role-specific config values
	switch role {
	case config.Root:
		// Root-specific settings
		configToSave.IsRoot = true
		configToSave.IsBootnode = false
		configToSave.InstallComplete = true // Root install is considered complete by this point.
		configToSave.ClientOnly = false
		configToSave.ChainID = chainID // This is KNIRVCHAIN-ROOT<port> or similar
		// MinersAddress for Root is set by DefaultRootConfig or constants.

		// Enable payment processor for Root nodes
		configToSave.PaymentProcessor.Enabled = true
		// MasterAddress for Root is BLOCKCHAIN_ADDRESS, set by DefaultRootConfig.

		// Set database paths
		dbPath, err := config.GetBlockchainDatabasePath(nil, "agent.db", role)
		if err != nil {
			return currentCfg, fmt.Errorf("failed to determine database path: %w", err)
		}
		configToSave.BlockchainDatabasePath = dbPath

	case config.RoleBootnode:
		// Bootnode-specific settings
		configToSave.IsBootnode = true
		configToSave.IsRoot = false
		configToSave.ClientOnly = false
		configToSave.ChainID = chainID // This is the <uuid> from URI

		// Set wallet addresses
		if devWallet != nil {
			configToSave.MinersAddress = devWallet.GetAddress()
		}
		if masterWallet != nil {
			configToSave.MasterAddress = masterWallet.GetAddress()
		}

		// Set database paths for Bootnode (it's a specialized Peer)
		// BlockchainDatabasePath will be its primary blockchain DB
		sharedDbPath, err := config.GetBlockchainDatabasePath(nil, fmt.Sprintf("%s_shared.db", chainID), role)
		if err != nil {
			return currentCfg, fmt.Errorf("failed to determine bootnode shared database path: %w", err)
		}
		configToSave.BlockchainDatabasePath = sharedDbPath

		// SearchableDatabasePath for other local data
		localDbPath, err := config.GetSearchableDatabasePath(nil, chainID, role)
		if err != nil {
			return currentCfg, fmt.Errorf("failed to determine dev database path: %w", err)
		}
		configToSave.SearchableDatabasePath = localDbPath

	case config.RolePeer:
		// Peer-specific settings
		configToSave.IsBootnode = false
		configToSave.IsRoot = false
		configToSave.ClientOnly = false
		configToSave.ChainID = chainID // This is the <uuid> from URI

		// Set wallet address
		if devWallet != nil {
			configToSave.MinersAddress = devWallet.GetAddress()
		}

		// Set database paths for Peer
		// BlockchainDatabasePath will be its primary blockchain DB
		sharedDbPath, err := config.GetBlockchainDatabasePath(nil, fmt.Sprintf("%s_shared.db", chainID), role)
		if err != nil {
			return currentCfg, fmt.Errorf("failed to determine dev shared database path: %w", err)
		}
		configToSave.BlockchainDatabasePath = sharedDbPath

		// SearchableDatabasePath for other local data
		localDbPath, err := config.GetSearchableDatabasePath(nil, chainID, role)
		if err != nil {
			return currentCfg, fmt.Errorf("failed to determine dev database path: %w", err)
		}
		configToSave.SearchableDatabasePath = localDbPath

	case config.RoleClient:
		// Client-specific settings
		configToSave.IsBootnode = false
		configToSave.IsRoot = false
		configToSave.ClientOnly = true
		configToSave.ChainID = chainID       // This is the client's wallet address
		configToSave.MinersAddress = chainID // Client's own wallet address

		// Set database paths
		dbPath, err := config.GetBlockchainDatabasePath(nil, "client.db", role)
		if err != nil {
			return currentCfg, fmt.Errorf("failed to determine database path: %w", err)
		}
		configToSave.BlockchainDatabasePath = dbPath
	}

	// Ensure InstallComplete flag is set to true for all non-Root roles
	if role != config.Root {
		log.Printf("Setting InstallComplete=true for %s role", role.String())
		configToSave.InstallComplete = true
	}

	// Ensure generic Port and P2PPort settings are explicitly set in the configuration for Peer and Bootnode roles
	if role == config.RolePeer || role == config.RoleBootnode {
		log.Printf("Setting generic Port and P2PPort in configuration for %s: HTTP=%d, P2P=%d", role.String(), reflectionHTTPPort, reflectionP2PPort)
		configToSave.Port = uint64(reflectionHTTPPort)
		configToSave.P2PPort = uint64(reflectionP2PPort)

		// Peers and bootnodes don't use reflection ports
		// Reflection ports are only used by root nodes in network mode

		// Set IsPeer for dev role
		if role == config.RolePeer {
			configToSave.IsPeer = true
		}
	}

	// For Root nodes, we'll set the reflection ports for network mode
	// These are only used when running in network mode with reflection nodes
	if role == config.Root {
		// Set reflection ports to be different from the main ports
		configToSave.ReflectionHTTPPort = configToSave.Port + 1000
		configToSave.ReflectionP2PPort = configToSave.P2PPort + 1000

		// Ensure they don't conflict with the main ports
		if configToSave.ReflectionHTTPPort == configToSave.Port {
			configToSave.ReflectionHTTPPort++
		}
		if configToSave.ReflectionP2PPort == configToSave.P2PPort {
			configToSave.ReflectionP2PPort++
		}

		log.Printf("Setting reflection ports for network mode: HTTP=%d, P2P=%d",
			configToSave.ReflectionHTTPPort, configToSave.ReflectionP2PPort)
	}

	// Log the configuration being saved
	log.Printf("Saving configuration for %s node with ChainID: %s, InstallComplete: %v",
		role.String(), configToSave.ChainID, configToSave.InstallComplete)

	// Save the configuration using the SaveConfigToUserDir function
	// This will save to the role-specific data directory
	log.Printf("Installer: Saving configuration for %s role", role.String())
	config.SaveConfigToUserDir(configToSave, role)

	// Get the path for logging purposes
	roleDataDir, err := config.GetDataDir(role)
	if err != nil {
		log.Printf("Warning: Could not determine role-specific data directory: %v", err)
	} else {
		roleSpecificFilename := fmt.Sprintf("%s_config.json", strings.ToLower(role.String()))
		configPath := filepath.Join(roleDataDir, roleSpecificFilename)
		fmt.Printf("Configuration updated and saved to %s\n", configPath)

		// Verify that the configuration file exists and contains the expected data
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			log.Printf("WARNING: Configuration file %s was not found after save attempt", configPath)
		} else {
			log.Printf("Verified: Configuration file %s exists", configPath)
		}
	}

	fmt.Println("\n=== Installation Complete ===")

	// Role-specific completion message
	switch role {
	case config.Root:
		fmt.Println("Your KNIRVCHAIN ROOT Node is now configured.")
		fmt.Println("This node will be able to create and manage chains.")
	case config.RoleBootnode:
		fmt.Println("Your KNIRVCHAIN Bootnode is now configured with a unique chain URI.")
		fmt.Println("This node will help with dev discovery and network coordination.")
	case config.RolePeer:
		fmt.Println("Your KNIRVCHAIN Peer Node is now configured with a unique chain URI.")
		fmt.Println("This node will participate in the network and process transactions.")
	case config.RoleClient:
		fmt.Println("Your KNIRVCHAIN Client Node is now configured.")
		fmt.Println("This node will connect to the network with reduced resource usage.")
	}

	fmt.Println("Finalizing installation and launching KNIRVCHAIN Node...")

	// Set installation complete flag
	configToSave.InstallComplete = true

	// Save config with installation complete flag using the SaveConfigToUserDir function
	log.Printf("Saving final configuration with InstallComplete=true for %s role", role.String())
	config.SaveConfigToUserDir(configToSave, role)

	// We've already saved the configuration using SaveConfigToUserDir, which handles
	// saving to the role-specific data directory. No need for additional saves.
	/*
		roleConfigFilename := fmt.Sprintf("%s_config.json", strings.ToLower(nodeRole.String()))

		// Only save in user config directory
		userConfigDir, err := os.UserConfigDir()
		if err == nil {
			appConfigDir := filepath.Join(userConfigDir, config.AppName)
			if err := os.MkdirAll(appConfigDir, 0755); err == nil {
				userRoleConfigPath := filepath.Join(appConfigDir, roleConfigFilename)
				_, err = config.SaveConfig(userRoleConfigPath, configToSave)
				if err != nil {
					log.Printf("Warning: Failed to save role config to user config dir %s: %v", userRoleConfigPath, err)
				} else {
					log.Printf("Saved role config to user config dir: %s", userRoleConfigPath)
				}
			}
		} else {
			log.Printf("Warning: Could not determine user config directory: %v", err)
		}
	*/

	// Add delay to ensure config is saved
	time.Sleep(500 * time.Millisecond)

	// Try multiple restart methods in sequence
	fmt.Println("Installation complete. Attempting to restart application...")

	// Create a new args slice with the --skip-install flag to prevent installer from running again
	newArgs := []string{}
	skipInstallFlag := "--skip-install"
	hasSkipInstallFlag := false

	// Check if --skip-install flag is already present
	for _, arg := range os.Args[1:] {
		newArgs = append(newArgs, arg)
		if arg == skipInstallFlag {
			hasSkipInstallFlag = true
		}
	}

	// Add --skip-install flag if not already present
	if !hasSkipInstallFlag {
		newArgs = append(newArgs, skipInstallFlag)
	}

	log.Printf("Restart will use args: %v", newArgs)

	// Method 1: Try to restart using executable path
	exePath, err := os.Executable()
	if err == nil {
		log.Printf("Attempting restart using executable path: %s", exePath)
		cmd := exec.Command(exePath, newArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err == nil {
			log.Printf("Successfully restarted application using executable path")
			os.Exit(0)
			return configToSave, nil
		} else {
			log.Printf("Failed to restart using executable path: %v", err)
		}
	} else {
		log.Printf("Warning: Failed to get executable path: %v", err)
	}

	// Method 2: Try to restart using "go run ." with new flags
	log.Printf("Attempting restart using 'go run .'")
	goArgs := append([]string{"run", "."}, newArgs...)
	cmd := exec.Command("go", goArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err == nil {
		log.Printf("Successfully restarted application using 'go run .'")
		os.Exit(0)
		return configToSave, nil
	} else {
		log.Printf("Failed to restart using 'go run .': %v", err)
	}

	// Method 3: Try to restart using current working directory executable
	cwd, err := os.Getwd()
	if err == nil {
		possibleExe := filepath.Join(cwd, filepath.Base(os.Args[0]))
		log.Printf("Attempting restart using CWD executable: %s", possibleExe)
		if _, err := os.Stat(possibleExe); err == nil {
			cmd := exec.Command(possibleExe, newArgs...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Start(); err == nil {
				log.Printf("Successfully restarted application using CWD executable")
				os.Exit(0)
				return configToSave, nil
			} else {
				log.Printf("Failed to restart using CWD executable: %v", err)
			}
		}
	}

	// Method 4: Fallback - return without exiting to continue in current process
	log.Printf("All restart methods failed. Continuing in current process...")
	fmt.Println("Installation complete. Continuing in current process...")

	return configToSave, nil
}

// LaunchAfterInstall launches the main application after installation
func LaunchAfterInstall(role config.Role) error {
	// Load existing config
	configPath := "config.json"
	cfg, _, err := config.LoadConfig(configPath, role)
	if err != nil {
		log.Printf("Warning: could not load config: %v", err)
		cfg = config.DefaultConfig()
	}

	// Set InstallComplete flag
	cfg.InstallComplete = true

	// Check if we need to load the master wallet
	if cfg.MasterAddress == "" {
		// Try to load master wallet if it exists
		masterWallet, err := walletManager.LoadMasterWallet("", role)
		if err == nil {
			cfg.MasterAddress = masterWallet.GetAddress()
			log.Printf("Loaded Master wallet address: %s", cfg.MasterAddress)
		}
	}

	// Write updated config back to file using SaveConfigToUserDir to role-specific _config.json dir for compatibility
	config.SaveConfigToUserDir(cfg, role)

	// Launch the application
	cmd := exec.Command(os.Args[0])
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start detached process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to launch application: %w", err)
	}
	return nil
}

// promptForRootEndpoint prompts the user for the KNIRVCHAIN Root endpoint
// If nonInteractive is true, it returns the default endpoint without prompting
func promptForRootEndpoint(nonInteractive bool, currentCfg *config.Config) string {
	defaultEndpoint := "http://localhost:5050" // Changed default to match test port

	// Check config for existing root endpoint
	if currentCfg != nil && currentCfg.P2P.RootNodeURI != "" {
		return currentCfg.P2P.RootNodeURI
	}

	// For test environments, check if we have a test root node URI
	if os.Getenv("agent_TEST_ROOT_URI") != "" {
		return os.Getenv("agent_TEST_ROOT_URI")
	}

	if nonInteractive {
		fmt.Printf("Non-interactive mode: Using root endpoint: %s\n", defaultEndpoint)
		return defaultEndpoint
	}

	fmt.Printf("Enter the KNIRVCHAIN Root endpoint [default: %s]: ", defaultEndpoint)
	var endpoint string
	fmt.Scanln(&endpoint)

	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	// Ensure the endpoint doesn't end with a slash
	endpoint = strings.TrimSuffix(endpoint, "/")

	return endpoint
}

// promptForDesiredURI prompts the user for an optional desired URI
// If nonInteractive is true, it returns an empty string for random URI generation
func promptForDesiredURI(nonInteractive bool) string {
	if nonInteractive {
		fmt.Println("Non-interactive mode: Using randomly generated URI")
		return ""
	}

	fmt.Println("\nYou can request a specific URI (optional).")
	fmt.Println("This will be used to request a specific URI from the server.")
	fmt.Println("Leave empty for a randomly generated URI.")
	fmt.Print("Enter desired URI: ")

	var input string
	fmt.Scanln(&input)

	return strings.TrimSpace(input)
}

// Role selection is now determined at compile time or via command-line flags

// GenerateChainURI connects to the KNIRVCHAIN Root and generates a unique chain URI
// If desiredURI is not empty, it will be passed to the server as a request parameter
// Returns: extractedID, fullURI, txnHash, error
func GenerateChainURI(rootEndpoint string, desiredURI string) (string, string, string, error) {
	// Construct the URI generator endpoint
	uriGeneratorURL := fmt.Sprintf("%s/uriGenerator", rootEndpoint)

	fmt.Printf("Connecting to %s...\n", uriGeneratorURL)

	// Create a client with a timeout
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Prepare request body (empty if no desiredURI)
	var requestBody io.Reader = nil
	if desiredURI != "" {
		fmt.Printf("Requesting specific URI: %s\n", desiredURI)
		requestData := map[string]string{"desired_id": desiredURI}
		jsonData, err := json.Marshal(requestData)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to marshal request data: %w", err)
		}
		requestBody = bytes.NewBuffer(jsonData)
	}

	// Create the request with POST method
	req, err := http.NewRequest("POST", uriGeneratorURL, requestBody)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "KNIRVCHAIN-Node-Installer/1.0")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP request failed: %v", err)
		return "", "", "", fmt.Errorf("failed to connect to KNIRVCHAIN Root: %w", err)
	}
	defer resp.Body.Close()

	// Log HTTP status code
	log.Printf("Received HTTP status code: %d", resp.StatusCode)

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return "", "", "", fmt.Errorf("failed to read response: %w", err)
	}

	// Log raw response body
	log.Printf("Raw response body: %s", string(body))

	// Check the status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Printf("Non-OK response received: %d - %s", resp.StatusCode, string(body))
		return "", "", "", fmt.Errorf("received non-OK response from server: %d - %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var uriResponse URIResponse
	err = json.Unmarshal(body, &uriResponse)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert the URI to the new format if it's in the old format
	// Old format: chain://<ID>
	// New format: knirv://<ID>.chain/
	uri := uriResponse.URI
	if strings.HasPrefix(uri, "chain://") {
		chainID := strings.TrimPrefix(uri, "chain://")
		uri = fmt.Sprintf("knirv://%s.chain/", chainID)
		log.Printf("Converted URI from old format to new format: %s", uri)
	}

	// Extract the chain ID from the URI
	var extractedID string
	if strings.HasPrefix(uri, "knirv://") {
		parts := strings.Split(strings.TrimPrefix(uri, "knirv://"), ".")
		if len(parts) > 0 {
			extractedID = parts[0]
		}
	}
	if extractedID == "" {
		return "", "", "", fmt.Errorf("could not extract chain ID from URI: %s", uri)
	}

	// Return extracted ID, full URI, and transaction hash
	return extractedID, uri, uriResponse.TxnHash, nil
}

// promptForUint prompts for a uint64 value with a default
// If nonInteractive is true, it returns the default value without prompting
func promptForUint(prompt string, defaultValue uint64, nonInteractive bool) uint64 {
	if nonInteractive {
		return defaultValue
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [default: %d]: ", prompt, defaultValue)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	value, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		fmt.Printf("Invalid input. Using default value: %d\n", defaultValue)
		return defaultValue
	}
	return value
}

// registerURIHandlers registers the URI handler for knirv:// protocol
// which now includes resource types like .chain and .nrn in the URI structure
func RegisterURIHandlers(chainURI string) error {
	// Define the URI scheme to register
	schemes := []uri.URIScheme{
		{
			Name:        "agent",
			Description: "KNIRVCHAIN Decentralized MCP Network",
		},
	}

	// Register the URI scheme based on the operating system
	return uri.RegisterURISchemes(schemes)
}

// DiscoverPublicAddress uses STUN to discover the node's public IP and port
func DiscoverPublicAddress(registryHTTPBaseURL string, logger *logrus.Logger) (string, int, error) {
	if logger == nil {
		logger = logrus.New()
		logger.SetOutput(os.Stdout)
		logger.SetLevel(logrus.InfoLevel)
	}

	stunInfoURL := fmt.Sprintf("%s/stun", registryHTTPBaseURL)
	logger.Infof("Fetching STUN server details from: %s", stunInfoURL)

	httpClient := &http.Client{Timeout: 45 * time.Second}
	resp, err := httpClient.Get(stunInfoURL)
	if err != nil {
		logger.Errorf("Failed to fetch /stun endpoint: %v", err)
		return getLocalIPOrFallback(logger, fmt.Errorf("failed to fetch /stun endpoint: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logger.Errorf("/stun endpoint returned non-OK status: %d - %s", resp.StatusCode, string(bodyBytes))
		return getLocalIPOrFallback(logger, fmt.Errorf("/stun endpoint returned status %d", resp.StatusCode))
	}

	var stunInfo StunInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&stunInfo); err != nil {
		logger.Errorf("Failed to decode /stun response: %v", err)
		return getLocalIPOrFallback(logger, fmt.Errorf("failed to decode /stun response: %w", err))
	}

	// Try UDP STUN if enabled
	if stunInfo.Protocols.UDP.Enabled && stunInfo.ConnectionStrings.UDP != "" {
		udpStunTarget := parseStunConnectionString(stunInfo.ConnectionStrings.UDP)
		if udpStunTarget != "" {
			logger.Infof("Attempting STUN discovery via UDP: %s", udpStunTarget)
			ip, port, err := performStunDiscovery("udp", udpStunTarget, logger)
			if err == nil {
				logger.Infof("Successfully discovered public address via UDP STUN: %s:%d", ip, port)
				return ip, port, nil
			}
			logger.Warnf("UDP STUN discovery failed for %s: %v", udpStunTarget, err)
			time.Sleep(5 * time.Second)
		}
	}

	// Try TCP STUN if enabled and UDP failed
	if stunInfo.Protocols.TCP.Enabled && stunInfo.ConnectionStrings.TCP != "" {
		tcpStunTarget := parseStunConnectionString(stunInfo.ConnectionStrings.TCP)
		if tcpStunTarget != "" {
			logger.Infof("Attempting STUN discovery via TCP: %s", tcpStunTarget)
			ip, port, err := performStunDiscovery("tcp", tcpStunTarget, logger)
			if err == nil {
				logger.Infof("Successfully discovered public address via TCP STUN: %s:%d", ip, port)
				return ip, port, nil
			}
			logger.Warnf("TCP STUN discovery failed for %s: %v", tcpStunTarget, err)
			time.Sleep(5 * time.Second)
		}
	}

	// Final fallback to local IP
	logger.Warnf("All STUN discovery methods failed or were not available.")
	return getLocalIPOrFallback(logger, errors.New("all STUN methods failed"))
}

func getLocalIPOrFallback(logger *logrus.Logger, originalError error) (string, int, error) {
	localIP, err := getLocalIP()
	if err != nil {
		logger.Errorf("Failed to get local IP as fallback: %v", err)
		return "", 0, fmt.Errorf("all STUN methods and local IP discovery failed. Original STUN error: %w; Local IP error: %v", originalError, err)
	}
	logger.Infof("Falling back to local IP: %s (port will be 0, actual P2P port used by node)", localIP)
	return localIP, 0, nil // Port 0 indicates it's just the IP, actual port is node's P2P port
}

func parseStunConnectionString(connStr string) string {
	// Remove "stun:" prefix
	s := strings.TrimPrefix(connStr, "stun:")
	// Remove "turn:" prefix (though less likely for STUN strings)
	s = strings.TrimPrefix(s, "turn:")
	// Remove any query parameters like "?transport=tcp"
	if queryIdx := strings.Index(s, "?"); queryIdx != -1 {
		s = s[:queryIdx]
	}
	return s // Should be "host:port"
}

// performStunDiscovery attempts STUN using the pion/stun library for UDP or TCP.
func performStunDiscovery(networkType string, stunServerAddress string, logger *logrus.Logger) (string, int, error) {
	if networkType != "udp" && networkType != "tcp" {
		return "", 0, fmt.Errorf("invalid network type for STUN: %s", networkType)
	}

	logger.Infof("Performing STUN discovery. Network: %s, Server: %s", networkType, stunServerAddress)

	// Create a STUN client
	var network string
	if networkType == "udp" {
		network = "udp"
	} else { // tcp
		network = "tcp"
	}

	client, err := stun.Dial(network, stunServerAddress)
	if err != nil {
		logger.Errorf("Failed to dial STUN server %s (%s): %v", stunServerAddress, networkType, err)
		return "", 0, fmt.Errorf("failed to dial STUN server: %w", err)
	}
	defer client.Close()

	// Build STUN message
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	var publicIP string
	var publicPort int

	// Send the request and handle the response
	if err := client.Do(message, func(res stun.Event) {
		if res.Error != nil {
			logger.Errorf("STUN request failed: %v", res.Error)
			err = res.Error
			return
		}

		var xorAddr stun.XORMappedAddress
		if err := xorAddr.GetFrom(res.Message); err != nil {
			logger.Errorf("Failed to get XOR-MAPPED-ADDRESS: %v", err)
			return
		}

		publicIP = xorAddr.IP.String()
		publicPort = xorAddr.Port
		logger.Infof("Discovered public address via %s STUN: %s:%d", strings.ToUpper(networkType), publicIP, publicPort)
	}); err != nil {
		logger.Errorf("STUN client.Do error for %s (%s): %v", stunServerAddress, networkType, err)
		return "", 0, fmt.Errorf("stun request execution failed: %w", err)
	}

	if publicIP == "" || publicPort == 0 {
		return "", 0, fmt.Errorf("stun discovery via %s did not return a valid address from %s", networkType, stunServerAddress)
	}

	return publicIP, publicPort, nil
}

func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no non-loopback IPv4 address found")
}

// RegisterWithNodeRegistry registers this node with the KNIRVCHAIN Node Registry
func RegisterWithNodeRegistry(chainURI, ip, portStr string, nodeID string) error {
	// Extract the chain ID from the URI
	var chainID string
	if strings.HasPrefix(chainURI, "knirv://") {
		parts := strings.Split(strings.TrimPrefix(chainURI, "knirv://"), ".")
		if len(parts) > 0 {
			chainID = parts[0]
		}
	} else if strings.HasPrefix(chainURI, "chain://") {
		chainID = strings.TrimPrefix(chainURI, "chain://")
	} else {
		chainID = chainURI // Use as-is if not in expected format
	}

	// Convert port string to integer
	portNum, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port number: %w", err)
	}

	// Use the enhanced registration function
	// BootnodeRegistryURL is defined in discovery_manager.go and should be used if available, or a default.
	err = RegisterWithRegistry(BootnodeRegistryURL, chainID, portNum, ip, nodeID)
	if err != nil {
		return fmt.Errorf("failed to register with node registry: %w", err)
	}

	return nil
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		// Try to detect the desktop environment
		if os.Getenv("DISPLAY") != "" {
			// Try xdg-open first (most common)
			if _, err := exec.LookPath("xdg-open"); err == nil {
				cmd = exec.Command("xdg-open", url)
			} else if _, err := exec.LookPath("sensible-browser"); err == nil {
				cmd = exec.Command("sensible-browser", url)
			} else if _, err := exec.LookPath("gnome-open"); err == nil {
				cmd = exec.Command("gnome-open", url)
			} else if _, err := exec.LookPath("kde-open"); err == nil {
				cmd = exec.Command("kde-open", url)
			} else {
				return fmt.Errorf("could not find a browser to open URL")
			}
		} else {
			return fmt.Errorf("no display detected for opening browser")
		}
	default:
		return fmt.Errorf("unsupported platform for opening browser")
	}

	return cmd.Start()
}

// ConfigureSystemService sets up the application as a background service based on OS
func ConfigureSystemService(configPath string, cfg *config.Config, role config.Role) error {
	os := runtime.GOOS
	switch os {
	case "linux":
		return configureLinuxService(configPath, cfg, role)
	case "windows":
		return configureWindowsService(configPath, cfg, role)
	case "darwin":
		return configureMacOSService(configPath, cfg, role)
	default:
		return fmt.Errorf("unsupported operating system for background service: %s", os)
	}
}

// configureLinuxService sets up a systemd service for Linux
func configureLinuxService(configPath string, cfg *config.Config, role config.Role) error {
	// Set default values
	serviceUser := "KNIRVCHAIN"
	workingDir := "/var/lib/KNIRVCHAIN"
	logPath := "/var/log/KNIRVCHAIN.log"

	// Determine service name and command line based on service type
	serviceName := fmt.Sprintf("KNIRVCHAIN-%s", strings.ToLower(role.String()))
	var description string
	var execStart string

	// Get the absolute path to the executable
	executablePath, err := os.Executable()
	if err != nil {
		executablePath = "/usr/local/bin/KNIRVCHAIN" // Fallback if we can't determine the path
	}

	if role == config.RoleBootnode {
		description = fmt.Sprintf("KNIRVCHAIN Bootnode Service for chain %s", cfg.ChainID)
		execStart = fmt.Sprintf("%s -bootnode -config %s", executablePath, configPath)

		// Use bootnode-specific settings if available
		if cfg.Bootnode.ServiceUser != "" {
			serviceUser = cfg.Bootnode.ServiceUser
		}
		if cfg.Bootnode.ServiceWorkingDir != "" {
			workingDir = cfg.Bootnode.ServiceWorkingDir
		}
		if cfg.Bootnode.ServiceLogPath != "" {
			logPath = cfg.Bootnode.ServiceLogPath
		}
	} else {
		description = fmt.Sprintf("KNIRVCHAIN Peer Node Service for chain %s", cfg.ChainID)
		execStart = fmt.Sprintf("%s -config %s", executablePath, configPath)
	}

	// Create systemd service file
	serviceContent := fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
Type=simple
User=%s
ExecStart=%s
Restart=always
RestartSec=10
WorkingDirectory=%s
StandardOutput=append:%s
StandardError=append:%s

[Install]
WantedBy=multi-user.target
`, description, serviceUser, execStart, workingDir, logPath, logPath)

	// Check if running as root
	if os.Geteuid() != 0 {
		// Not running as root, create a service file template for manual installation
		// Get the role-specific data directory
		dataDir, err := config.GetDataDir(role)
		if err != nil {
			// Fallback to home directory if we can't get the role data dir
			homeDir, err := os.UserHomeDir()
			if err != nil {
				homeDir = "." // Fallback to current directory if we can't get home dir
			}
			dataDir = filepath.Join(homeDir, ".config", "KNIRVCHAIN")
		}

		// Create a directory for the service file if it doesn't exist
		serviceDir := filepath.Join(dataDir, "services")
		if err := os.MkdirAll(serviceDir, 0755); err != nil {
			return fmt.Errorf("failed to create service directory: %w", err)
		}

		// Write the service file template
		serviceFilePath := filepath.Join(serviceDir, fmt.Sprintf("%s.service", serviceName))
		if err := os.WriteFile(serviceFilePath, []byte(serviceContent), 0644); err != nil {
			return fmt.Errorf("failed to write systemd service file template: %w", err)
		}

		// Print instructions for manual installation
		fmt.Printf("\n=== Manual Service Installation Instructions ===\n")
		fmt.Printf("A service file template has been created at: %s\n\n", serviceFilePath)
		fmt.Printf("To install the service manually, run the following commands as root:\n")
		fmt.Printf("  sudo cp %s /etc/systemd/system/\n", serviceFilePath)
		fmt.Printf("  sudo systemctl daemon-reload\n")
		fmt.Printf("  sudo systemctl enable %s\n", serviceName)
		fmt.Printf("  sudo systemctl start %s\n\n", serviceName)
		fmt.Printf("To check the service status:\n")
		fmt.Printf("  sudo systemctl status %s\n\n", serviceName)

		return fmt.Errorf("must run as root to install system service")
	}

	// Ensure log directory exists
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Ensure working directory exists
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}

	// Write service file
	serviceFilePath := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
	if err := os.WriteFile(serviceFilePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write systemd service file: %w", err)
	}

	// Reload systemd to recognize the new service
	cmd := exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Enable and start service
	cmd = exec.Command("systemctl", "enable", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable systemd service: %w", err)
	}

	cmd = exec.Command("systemctl", "start", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start systemd service: %w", err)
	}

	return nil
}

// configureWindowsService sets up a Windows service
func configureWindowsService(configPath string, cfg *config.Config, role config.Role) error {
	// Set default values
	displayName := "KNIRVCHAIN Node"
	description := "KNIRVCHAIN Node Service"
	logPath := "C:\\ProgramData\\KNIRVCHAIN\\logs\\KNIRVCHAIN.log"

	// Determine service name and command line based on service type
	serviceName := fmt.Sprintf("KNIRVCHAIN-%s", strings.ToLower(role.String()))

	// Get the executable path
	executablePath, err := os.Executable()
	if err != nil {
		executablePath = os.Getenv("ProgramFiles") + "\\KNIRVCHAIN\\KNIRVCHAIN.exe" // Fallback
	}

	var binPath string

	if role == config.RoleBootnode {
		displayName = "KNIRVCHAIN Bootnode"
		description = fmt.Sprintf("KNIRVCHAIN Bootnode Service for chain %s", cfg.ChainID)
		binPath = fmt.Sprintf("\"%s\" -bootnode -config \"%s\"", executablePath, configPath)

		// Use bootnode-specific settings if available
		if cfg.Bootnode.ServiceDisplayName != "" {
			displayName = cfg.Bootnode.ServiceDisplayName
		}
		if cfg.Bootnode.ServiceDescription != "" {
			description = cfg.Bootnode.ServiceDescription
		}
		if cfg.Bootnode.ServiceLogPath != "" {
			logPath = cfg.Bootnode.ServiceLogPath
		}
	} else {
		description = fmt.Sprintf("KNIRVCHAIN Peer Node Service for chain %s", cfg.ChainID)
		binPath = fmt.Sprintf("\"%s\" -config \"%s\"", executablePath, configPath)
	}

	// Ensure log directory exists
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Check if we have admin privileges
	isAdmin, err := checkWindowsAdminPrivileges()
	if err != nil || !isAdmin {
		// Create a batch file for manual installation
		// Get the role-specific data directory
		dataDir, err := config.GetDataDir(role)
		if err != nil {
			// Fallback to AppData directory if we can't get the role data dir
			appDataDir, err := os.UserHomeDir()
			if err != nil {
				appDataDir = "." // Fallback to current directory
			}
			dataDir = filepath.Join(appDataDir, "AppData", "Local", "KNIRVCHAIN")
		}

		serviceDir := filepath.Join(dataDir, "services")
		if err := os.MkdirAll(serviceDir, 0755); err != nil {
			return fmt.Errorf("failed to create service directory: %w", err)
		}

		// Create batch file for service installation
		batchContent := fmt.Sprintf(`@echo off
echo Installing KNIRVCHAIN %s Service
sc create %s binPath= "%s" start= auto DisplayName= "%s"
sc description %s "%s"
sc failure %s reset= 86400 actions= restart/60000/restart/60000/restart/60000
sc start %s
echo Service installation complete.
pause
`,
			cases.Title(language.English).String(strings.ToLower(role.String())),
			serviceName,
			binPath,
			displayName,
			serviceName,
			description,
			serviceName,
			serviceName)

		batchFilePath := filepath.Join(serviceDir, fmt.Sprintf("install_%s_service.bat", serviceName))
		if err := os.WriteFile(batchFilePath, []byte(batchContent), 0644); err != nil {
			return fmt.Errorf("failed to write service installation batch file: %w", err)
		}

		// Print instructions
		fmt.Printf("\n=== Manual Service Installation Instructions ===\n")
		fmt.Printf("A service installation batch file has been created at:\n%s\n\n", batchFilePath)
		fmt.Printf("To install the service manually:\n")
		fmt.Printf("1. Right-click on the batch file and select 'Run as administrator'\n")
		fmt.Printf("2. Follow the prompts to complete the installation\n\n")
		fmt.Printf("To check the service status, open Command Prompt as administrator and run:\n")
		fmt.Printf("  sc query %s\n\n", serviceName)

		return fmt.Errorf("must run as administrator to install Windows service")
	}

	// Create the service using sc.exe
	createCmd := exec.Command("sc", "create", serviceName, "binPath=", binPath, "start=", "auto", "DisplayName=", displayName)
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create Windows service: %w", err)
	}

	// Set the description
	descCmd := exec.Command("sc", "description", serviceName, description)
	if err := descCmd.Run(); err != nil {
		return fmt.Errorf("failed to set service description: %w", err)
	}

	// Configure failure recovery
	failureCmd := exec.Command("sc", "failure", serviceName, "reset=", "86400", "actions=", "restart/60000/restart/60000/restart/60000")
	if err := failureCmd.Run(); err != nil {
		return fmt.Errorf("failed to configure service failure recovery: %w", err)
	}

	// Start the service
	startCmd := exec.Command("sc", "start", serviceName)
	if err := startCmd.Run(); err != nil {
		return fmt.Errorf("failed to start Windows service: %w", err)
	}

	return nil
}

// checkWindowsAdminPrivileges checks if the current process has administrator privileges on Windows
func checkWindowsAdminPrivileges() (bool, error) {
	if runtime.GOOS != "windows" {
		return false, fmt.Errorf("not running on Windows")
	}

	// Create a test file in a protected directory
	testFile := "C:\\Windows\\Temp\\KNIRVORACLE_admin_test.tmp"
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err == nil {
		// Clean up the test file
		os.Remove(testFile)
		return true, nil
	}

	return false, nil
}

// configureMacOSService sets up a launchd service for macOS
func configureMacOSService(configPath string, cfg *config.Config, role config.Role) error {
	// Set default values
	workingDir := "/Library/KNIRVCHAIN"
	logPath := "/Library/Logs/KNIRVCHAIN/KNIRVCHAIN.log"

	// Determine service name and command line based on service type
	serviceName := fmt.Sprintf("com.KNIRVCHAIN.%s", strings.ToLower(role.String()))

	// Get the executable path
	executablePath, err := os.Executable()
	if err != nil {
		executablePath = "/usr/local/bin/KNIRVCHAIN" // Fallback
	}

	var programArgs []string

	if role == config.RoleBootnode {
		programArgs = []string{executablePath, "-bootnode", "-config", configPath}

		// Use bootnode-specific settings if available
		if cfg.Bootnode.ServiceWorkingDir != "" {
			workingDir = cfg.Bootnode.ServiceWorkingDir
		}
		if cfg.Bootnode.ServiceLogPath != "" {
			logPath = cfg.Bootnode.ServiceLogPath
		}
	} else {
		programArgs = []string{executablePath, "-config", configPath}
	}

	// Create plist content
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>%s</string>
		<string>%s</string>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>ThrottleInterval</key>
	<integer>10</integer>
	<key>WorkingDirectory</key>
	<string>%s</string>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
</dict>
</plist>`, serviceName, programArgs[0], programArgs[1], programArgs[2], programArgs[3], workingDir, logPath, logPath)

	// Check if running as root
	if os.Geteuid() != 0 {
		// Not running as root, create a plist file template for manual installation
		// Get the role-specific data directory
		dataDir, err := config.GetDataDir(role)
		if err != nil {
			// Fallback to home directory if we can't get the role data dir
			homeDir, err := os.UserHomeDir()
			if err != nil {
				homeDir = "." // Fallback to current directory
			}
			dataDir = filepath.Join(homeDir, "Library", "Application Support", "KNIRVCHAIN")
		}

		// Create a directory for the plist file if it doesn't exist
		serviceDir := filepath.Join(dataDir, "services")
		if err := os.MkdirAll(serviceDir, 0755); err != nil {
			return fmt.Errorf("failed to create service directory: %w", err)
		}

		// Write the plist file template
		plistFilePath := filepath.Join(serviceDir, fmt.Sprintf("%s.plist", serviceName))
		if err := os.WriteFile(plistFilePath, []byte(plistContent), 0644); err != nil {
			return fmt.Errorf("failed to write launchd plist file template: %w", err)
		}

		// Create a shell script for installation
		scriptContent := fmt.Sprintf(`#!/bin/bash
echo "Installing KNIRVCHAIN %s Service"
sudo mkdir -p %s
sudo mkdir -p %s
sudo cp "%s" /Library/LaunchDaemons/
sudo launchctl load -w /Library/LaunchDaemons/%s.plist
echo "Service installation complete."
`,
			cases.Title(language.English).String(strings.ToLower(role.String())),
			workingDir,
			filepath.Dir(logPath),
			plistFilePath,
			serviceName)

		scriptPath := filepath.Join(serviceDir, fmt.Sprintf("install_%s_service.sh", serviceName))
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
			return fmt.Errorf("failed to write service installation script: %w", err)
		}

		// Print instructions
		fmt.Printf("\n=== Manual Service Installation Instructions ===\n")
		fmt.Printf("A service plist file has been created at: %s\n", plistFilePath)
		fmt.Printf("An installation script has been created at: %s\n\n", scriptPath)
		fmt.Printf("To install the service manually, run the following commands in Terminal:\n")
		fmt.Printf("  chmod +x %s\n", scriptPath)
		fmt.Printf("  %s\n\n", scriptPath)
		fmt.Printf("You will be prompted for your administrator password.\n\n")
		fmt.Printf("To check the service status:\n")
		fmt.Printf("  sudo launchctl list | grep %s\n\n", serviceName)

		return fmt.Errorf("must run as root to install system service")
	}

	// Ensure log directory exists
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Ensure working directory exists
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}

	// Write plist file
	plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", serviceName)
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write launchd plist file: %w", err)
	}

	// Load and start the service
	loadCmd := exec.Command("launchctl", "load", "-w", plistPath)
	if err := loadCmd.Run(); err != nil {
		return fmt.Errorf("failed to load launchd service: %w", err)
	}

	return nil
}
