package starter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime" // Import runtime
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/pion/stun"
)

// URIResponse represents the response from the KNIRVCHAIN Root's /uriGenerator endpoint
type URIResponse struct {
	TxnHash string `json:"txn_hash"`
	URI     string `json:"uri"`
}

// ConfigData represents the configuration data to be saved to .env file
type ConfigData map[string]string

func Install() {
	fmt.Println("=== KNIRVCHAIN Verifier Node Installation ===")
	fmt.Println("This installer will:")
	fmt.Println("1. Connect to the KNIRVCHAIN Root")
	fmt.Println("2. Generate a unique chain URI for this verifier")
	fmt.Println("3. Register with KNIRVCHAIN Node Registry")
	fmt.Println("4. Detect host operating system")
	fmt.Println("5. Register URI handler for knirv:// protocol")
	fmt.Println("6. Set up the verifier node to start on system boot")
	fmt.Println("7. Update the application configuration")
	fmt.Println("8. Start the verifier node")
	fmt.Println()

	// Step 1: Connect to KNIRVCHAIN Root
	rootEndpoint := promptForRootEndpoint()

	// Prompt for desired URI (optional)
	desiredURI := promptForDesiredURI()

	// Step 2: Generate unique chain URI
	chainURI, hashID, err := GenerateChainURI(rootEndpoint, desiredURI)
	if err != nil {
		log.Fatalf("Failed to generate chain URI: %v", err)
	}
	fmt.Printf("Successfully generated chain URI: %s\n", chainURI)
	fmt.Printf("Transaction Hash ID: %s\n", hashID)

	// Step 3: Register with KNIRVCHAIN Node Registry
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}
	// First discover our public IP/port via STUN
	publicIP, publicPort, err := DiscoverPublicAddress()
	if err != nil {
		log.Printf("Warning: Failed to discover public address via STUN: %v", err)
		// Fall back to local port if STUN fails
		publicIP = "0.0.0.0"
		publicPort = port
	} else {
		fmt.Printf("Discovered public address: %s:%s\n", publicIP, publicPort)
	}

	// Register with node registry using public address
	err = RegisterWithNodeRegistry(chainURI, publicIP, publicPort)
	if err != nil {
		log.Printf("Warning: Failed to register with node registry: %v", err)
		fmt.Println("You can register manually later using the registry API.")
	} else {
		fmt.Println("Successfully registered with KNIRVCHAIN Node Registry")
	}

	// Step 4: Detect host operating system
	fmt.Printf("Detected operating system: %s\n", runtime.GOOS)

	// Step 5: Register URI handlers
	if runtime.GOOS == "windows" && !CheckAdminPrivileges() {
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

	// Step 6: Set up system service (placeholder)
	fmt.Println("Setting up system service (Placeholder)...")

	// Step 7: Update configuration
	// Extract the chain ID from the new URI format (knirv://<ID>.chain/)
	var serviceAddress string
	if strings.HasPrefix(chainURI, "knirv://") {
		// Extract the ID part from knirv://<ID>.chain/
		parts := strings.Split(strings.TrimPrefix(chainURI, "knirv://"), ".")
		if len(parts) > 0 {
			serviceAddress = parts[0]
		}
	} else if strings.HasPrefix(chainURI, "chain://") {
		// Handle legacy format for backward compatibility
		serviceAddress = strings.TrimPrefix(chainURI, "chain://")
	} else {
		// If the format is unexpected, use the whole URI
		serviceAddress = chainURI
	}

	err = UpdateConfiguration(serviceAddress, rootEndpoint)
	if err != nil {
		log.Fatalf("Failed to update configuration: %v", err)
	}
	fmt.Println("Configuration updated successfully.")

	fmt.Println("\n=== Installation Complete ===")
	fmt.Println("Your KNIRVCHAIN Verifier Node is now configured with a unique chain URI.")
	fmt.Println("Launching the Verifier Node Manager...")

	// Launch the application in a new process (will start Fyne GUI by default)
	cmd := exec.Command(os.Args[0])
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the process without waiting (so we can exit)
	err = cmd.Start()
	if err != nil {
		log.Printf("Warning: Failed to launch application: %v", err)
		fmt.Println("You can manually start the verifier node using the appropriate command for your system.")
		return
	}

	// Exit the installer successfully
	os.Exit(0)
}

// LaunchAfterInstall launches the main application after installation
func LaunchAfterInstall() error {
	envPath := ".env"

	// Load existing .env file if it exists
	envVars := make(map[string]string)
	if _, err := os.Stat(envPath); err == nil {
		file, err := os.ReadFile(envPath)
		if err != nil {
			log.Printf("Warning: could not read .env file: %v", err)
		} else {
			// Parse existing key=value pairs
			for _, line := range strings.Split(string(file), "\n") {
				if strings.Contains(line, "=") {
					parts := strings.SplitN(line, "=", 2)
					envVars[parts[0]] = parts[1]
				}
			}
		}
	}

	// Set INSTALL_COMPLETE flag
	envVars["INSTALL_COMPLETE"] = "true"

	// Write back all environment variables
	var buffer bytes.Buffer
	for key, value := range envVars {
		buffer.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	}

	// Write the .env file
	if err := os.WriteFile(envPath, buffer.Bytes(), 0644); err != nil {
		log.Printf("Warning: could not write .env file: %v", err)
	}

	// Launch the application with environment variable set
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "INSTALL_COMPLETE=true")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start detached process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to launch application: %w", err)
	}
	return nil
}

// promptForRootEndpoint prompts the user for the KNIRVCHAIN Root endpoint
func promptForRootEndpoint() string {
	defaultEndpoint := "http://localhost:5000"

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
func promptForDesiredURI() string {
	fmt.Println("\nYou can request a specific URI (optional).")
	fmt.Println("This will be used to request a specific URI from the server.")
	fmt.Println("Leave empty for a randomly generated URI.")
	fmt.Print("Enter desired URI: ")

	var input string
	fmt.Scanln(&input)

	return strings.TrimSpace(input)
}

// generateChainURI connects to the KNIRVCHAIN Root and generates a unique chain URI
// If desiredURI is not empty, it will be passed to the server as a request parameter
func GenerateChainURI(rootEndpoint string, desiredURI string) (string, string, error) {
	// Construct the URI generator endpoint
	uriGeneratorURL := fmt.Sprintf("%s/uriGenerator", rootEndpoint)

	fmt.Printf("Connecting to %s...\n", uriGeneratorURL)

	// Create a client with a timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Prepare request body (empty if no desiredURI)
	var requestBody io.Reader = nil
	if desiredURI != "" {
		fmt.Printf("Requesting specific URI: %s\n", desiredURI)
		requestData := map[string]string{"desired_id": desiredURI}
		jsonData, err := json.Marshal(requestData)
		if err != nil {
			return "", "", fmt.Errorf("failed to marshal request data: %w", err)
		}
		requestBody = bytes.NewBuffer(jsonData)
	}

	// Create the request with POST method
	req, err := http.NewRequest("POST", uriGeneratorURL, requestBody)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "KNIRVCHAIN-Verifier-Installer/1.0")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP request failed: %v", err)
		return "", "", fmt.Errorf("failed to connect to KNIRVCHAIN Root: %w", err)
	}
	defer resp.Body.Close()

	// Log HTTP status code
	log.Printf("Received HTTP status code: %d", resp.StatusCode)

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	// Log raw response body
	log.Printf("Raw response body: %s", string(body))

	// Check the status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("Non-OK response received: %d - %s", resp.StatusCode, string(body))
		return "", "", fmt.Errorf("received non-OK response from server: %d - %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var uriResponse URIResponse
	err = json.Unmarshal(body, &uriResponse)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse response: %w", err)
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

	// Return the URI and transaction hash
	return uri, uriResponse.TxnHash, nil
}

// updateConfiguration updates the configuration with the new service address
func UpdateConfiguration(serviceAddress, rootEndpoint string) error {
	// Define the path to the .env file
	envPath := ".env"

	// Check if the .env file exists
	_, err := os.Stat(envPath)
	if os.IsNotExist(err) {
		// If it doesn't exist, create it from test.env if available
		testEnvPath := "test.env"
		_, err = os.Stat(testEnvPath)
		if os.IsNotExist(err) {
			// If test.env doesn't exist either, create a new .env file with default values
			return createDefaultEnvFile(envPath, serviceAddress, rootEndpoint)
		}

		// Copy test.env to .env
		testEnvContent, err := os.ReadFile(testEnvPath)
		if err != nil {
			return fmt.Errorf("failed to read test.env: %w", err)
		}

		err = os.WriteFile(envPath, testEnvContent, 0644)
		if err != nil {
			return fmt.Errorf("failed to create .env from test.env: %w", err)
		}
	}

	// Load the current environment variables from .env
	envVars, err := godotenv.Read(envPath)
	if err != nil {
		return fmt.Errorf("failed to read .env file: %w", err)
	}

	// Update the SERVICE_ADDRESS and RPC_ENDPOINT
	envVars["SERVICE_ADDRESS"] = serviceAddress
	envVars["RPC_ENDPOINT"] = rootEndpoint

	// Ensure other required variables are set
	if _, exists := envVars["PORT"]; !exists {
		envVars["PORT"] = "3001"
	}
	if _, exists := envVars["TURN_ADDRESS"]; !exists {
		envVars["TURN_ADDRESS"] = "0.0.0.0"
	}
	if _, exists := envVars["TURN_PORT"]; !exists {
		envVars["TURN_PORT"] = "3478"
	}

	// Write the updated environment variables back to .env
	var buffer bytes.Buffer
	for key, value := range envVars {
		buffer.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	}

	// Add installation complete flag
	buffer.WriteString("INSTALL_COMPLETE=true\n")

	err = os.WriteFile(envPath, buffer.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	return nil
}

// createDefaultEnvFile creates a new .env file with default values
func createDefaultEnvFile(envPath, serviceAddress, rootEndpoint string) error {
	// Define default configuration values
	defaultConfig := ConfigData{
		"PORT":            "3001",
		"SERVICE_ADDRESS": serviceAddress,
		"RPC_ENDPOINT":    rootEndpoint,
		"TURN_ADDRESS":    "0.0.0.0",
		"TURN_PORT":       "3478",
		// Add other default values as needed
		"MINERS_ADDRESS":    "knirvchain3dd025e8fec7eda7cdd012ddde9c8e978ee7fa33",
		"DATABASE_PATH":     "database/knirv.db",
		"BLOCKCHAIN_NAME":   "KNIRVCHAIN",
		"MINING_DIFFICULTY": "5",
		"MINING_REWARD":     "1200",
		"CURRENCY_NAME":     "nrn",
	}

	// Create the .env file content
	var buffer bytes.Buffer
	for key, value := range defaultConfig {
		buffer.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	}

	// Write the .env file
	err := os.WriteFile(envPath, buffer.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to create default .env file: %w", err)
	}

	return nil
}

// registerURIHandlers registers the URI handler for knirv:// protocol
// which now includes resource types like .chain and .nrn in the URI structure
func RegisterURIHandlers(chainURI string) error {
	// Define the URI scheme to register
	schemes := []URIScheme{
		{
			Name:        "knirv",
			Description: "KNIRVCHAIN Decentralized Protocol",
		},
	}

	// Register the URI scheme based on the operating system
	return registerURISchemes(schemes)
}

// DiscoverPublicAddress uses STUN to discover the node's public IP and port
func DiscoverPublicAddress() (string, string, error) {
	// Create a UDP connection for STUN
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create UDP connection: %w", err)
	}
	defer conn.Close()

	// Get the local address before STUN
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	log.Printf("Local UDP address: %s", localAddr.String())

	// Create STUN client
	c, err := stun.NewClient(conn)
	if err != nil {
		return "", "", fmt.Errorf("failed to create STUN client: %w", err)
	}
	defer c.Close()

	// Configure STUN server address
	stunServerAddr, err := net.ResolveUDPAddr("udp", "registry.knirv.com:3478")
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve STUN server address: %w", err)
	}
	log.Printf("STUN server address: %s", stunServerAddr.String())

	// Build STUN message
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	// Send request to STUN server
	var publicAddr *stun.XORMappedAddress
	err = c.Do(message, func(res stun.Event) {
		if res.Error != nil {
			err = res.Error
			return
		}

		// Decode the XOR-MAPPED-ADDRESS attribute
		var xorAddr stun.XORMappedAddress
		if err := xorAddr.GetFrom(res.Message); err != nil {
			err = fmt.Errorf("failed to get XOR-MAPPED-ADDRESS: %w", err)
			return
		}
		publicAddr = &xorAddr
	})
	if err != nil {
		return "", "", fmt.Errorf("STUN request failed: %w", err)
	}
	if publicAddr == nil {
		return "", "", fmt.Errorf("no public address received from STUN server")
	}

	return publicAddr.IP.String(), fmt.Sprintf("%d", publicAddr.Port), nil
}

// RegisterWithNodeRegistry registers this node with the KNIRVCHAIN Node Registry
func RegisterWithNodeRegistry(chainURI, ip, port string) error {
	// TODO: Implement actual registration with KNIRVCHAIN Node Registry API
	// This is a placeholder implementation
	fmt.Printf("Registering node %s at %s:%s with KNIRVCHAIN Node Registry\n", chainURI, ip, port)
	return nil
}
