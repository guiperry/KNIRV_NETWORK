package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/core"
	"github.com/spf13/cobra"
)

// ServerCmd represents the server command
var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage MCP server registration (extrapolation)",
	Long: `Register and manage MCP servers on the KNIRVCHAIN blockchain.
This command handles the extrapolation integration route.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// MCPRegisterServerData represents the data for registering a server
type MCPRegisterServerData struct {
	ServerSchema   map[string]interface{} `json:"serverSchema"`
	Endpoint       string                 `json:"endpoint"`
	AutoRegister   bool                   `json:"autoRegister"`
	FileReferences []core.FileReference   `json:"fileReferences,omitempty"`
}

// serverRegisterCmd represents the server register command
var serverRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new MCP server",
	Long: `Register a new MCP server on the KNIRVCHAIN blockchain using the extrapolation route.

This command will register the server and optionally auto-register all capabilities
provided by the server.

Example:
  knirv mcp server register --node http://localhost:8080 --wallet ./my-wallet.json \
    --from 0x123abc... --server-schema ./server-schema.yaml --endpoint https://my-server.com/api \
    --fee 100 --auto-register-capabilities`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return registerServer(cmd, args)
	},
}

// serverListCmd represents the server list command
var serverListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered MCP servers",
	Long: `List all registered MCP servers on the KNIRVCHAIN blockchain.
Displays server IDs, endpoints, and capabilities.`,
	Run: func(cmd *cobra.Command, args []string) {
		// This will be implemented in a future phase
		log.Info("Server listing will be implemented in a future phase")
	},
}

// serverInfoCmd represents the server info command
var serverInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get information about an MCP server",
	Long: `Get detailed information about a registered MCP server.
Displays server metadata, capabilities, and status.`,
	Run: func(cmd *cobra.Command, args []string) {
		// This will be implemented in a future phase
		log.Info("Server info will be implemented in a future phase")
	},
}

func init() {
	ServerCmd.AddCommand(serverRegisterCmd)
	ServerCmd.AddCommand(serverListCmd)
	ServerCmd.AddCommand(serverInfoCmd)

	// server register flags
	serverRegisterCmd.Flags().String("node", "", "URL of the KNIRVCHAIN node")
	serverRegisterCmd.Flags().String("wallet", "", "Path to wallet file")
	serverRegisterCmd.Flags().String("from", "", "Address of the sender")
	serverRegisterCmd.Flags().String("server-schema", "", "Path to server schema file")
	serverRegisterCmd.Flags().String("endpoint", "", "Server endpoint URL")
	serverRegisterCmd.Flags().Uint64("fee", 0, "Transaction fee")

	// Optional flags
	serverRegisterCmd.Flags().String("password", "", "Wallet password (will prompt if not provided)")
	serverRegisterCmd.Flags().Bool("auto-register-capabilities", false, "Automatically register all capabilities provided by the server")
	serverRegisterCmd.Flags().Bool("deploy-server", false, "Deploy the server if it's not already running")
	serverRegisterCmd.Flags().String("deployment-config", "", "Path to deployment configuration file")
	serverRegisterCmd.Flags().Bool("test-connection", true, "Test server connection before registration")

	// Mark required flags
	serverRegisterCmd.MarkFlagRequired("node")
	serverRegisterCmd.MarkFlagRequired("wallet")
	serverRegisterCmd.MarkFlagRequired("from")
	serverRegisterCmd.MarkFlagRequired("server-schema")
	serverRegisterCmd.MarkFlagRequired("endpoint")
	serverRegisterCmd.MarkFlagRequired("fee")

	// server list flags
	serverListCmd.Flags().String("owner", "", "Filter by owner address")
	serverListCmd.Flags().String("format", "table", "Output format (table, json, csv)")

	// server info flags
	serverInfoCmd.Flags().String("id", "", "Server ID")
	serverInfoCmd.Flags().String("format", "table", "Output format (table, json, csv)")
}

// registerServer implements the server register command
func registerServer(cmd *cobra.Command, args []string) error {
	// 1. Parse and validate flags
	nodeURL, _ := cmd.Flags().GetString("node")
	walletPath, _ := cmd.Flags().GetString("wallet")
	fromAddress, _ := cmd.Flags().GetString("from")
	serverSchemaPath, _ := cmd.Flags().GetString("server-schema")
	endpoint, _ := cmd.Flags().GetString("endpoint")
	fee, _ := cmd.Flags().GetUint64("fee")
	password, _ := cmd.Flags().GetString("password")
	autoRegisterCapabilities, _ := cmd.Flags().GetBool("auto-register-capabilities")
	deployServer, _ := cmd.Flags().GetBool("deploy-server")
	deploymentConfigPath, _ := cmd.Flags().GetString("deployment-config")
	testConnection, _ := cmd.Flags().GetBool("test-connection")

	// 2. Load and parse server schema
	log.Info("Loading server schema...")
	serverSchema, err := loadServerSchema(serverSchemaPath)
	if err != nil {
		return fmt.Errorf("failed to load server schema: %w", err)
	}

	// 3. Validate server endpoint
	log.Info("Validating server endpoint...")
	if err := validateServerEndpoint(endpoint); err != nil {
		return fmt.Errorf("invalid server endpoint: %w", err)
	}

	// 4. Test server connection if requested
	if testConnection {
		log.Info("Testing server connection...")
		if err := testServerConnection(endpoint); err != nil {
			return fmt.Errorf("server connection test failed: %w", err)
		}
		log.Info("✅ Server connection test passed")
	}

	// 5. Deploy server if requested
	if deployServer {
		if deploymentConfigPath == "" {
			return fmt.Errorf("deployment configuration is required when deploy-server is enabled")
		}

		log.Info("Deploying MCP server...")
		if err := deployMCPServer(deploymentConfigPath, endpoint); err != nil {
			return fmt.Errorf("failed to deploy server: %w", err)
		}
		log.Info("✅ Server deployed successfully")
	}

	// 6. Load wallet and get private key
	if password == "" {
		// Prompt for password
		fmt.Print("Enter wallet password: ")
		var passwordBytes [64]byte
		n, err := os.Stdin.Read(passwordBytes[:])
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		password = strings.TrimSpace(string(passwordBytes[:n]))
	}

	// Create wallet manager
	walletManager := core.NewWalletManager(filepath.Dir(walletPath), log)

	// Load wallet
	log.Info("Loading wallet...")
	privateKey, err := walletManager.LoadWallet(walletPath, password)
	if err != nil {
		return fmt.Errorf("failed to load wallet: %w", err)
	}

	// 7. Create API client
	apiClient := core.NewAPIClient(
		nodeURL,
		core.WithTimeout(30*time.Second),
		core.WithRetries(3),
		core.WithLogger(log),
	)

	// 8. Test API connection
	log.Info("Testing API connection...")
	ctx := context.Background()
	if err := apiClient.HealthCheck(ctx); err != nil {
		return fmt.Errorf("failed to connect to KNIRVCHAIN node: %w", err)
	}

	// 9. Prepare server registration data
	serverData := MCPRegisterServerData{
		ServerSchema: serverSchema,
		Endpoint:     endpoint,
		AutoRegister: autoRegisterCapabilities,
	}

	// 10. Create transaction
	log.Info("Creating server registration transaction...")
	unsignedTx, err := core.CreateTransaction(
		fromAddress,
		"", // No recipient for server registration
		0,  // No value transfer
		serverData,
		fee,
		"REGISTER_SERVER",
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// 11. Sign transaction
	log.Info("Signing transaction...")
	dataBytes, err := json.Marshal(serverData)
	if err != nil {
		return fmt.Errorf("failed to marshal server data: %w", err)
	}

	signature, txHash, err := core.SignTransactionData(privateKey, *unsignedTx, dataBytes)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// 12. Assemble signed transaction
	publicKeyHex := core.GetPublicKeyHex(&privateKey.PublicKey)
	signedTx, err := core.AssembleSignedTransaction(*unsignedTx, publicKeyHex, signature, txHash)
	if err != nil {
		return fmt.Errorf("failed to assemble signed transaction: %w", err)
	}

	// 13. Submit transaction
	log.Info("Submitting transaction to KNIRVCHAIN...")
	response, err := apiClient.SubmitTransaction(ctx, signedTx)
	if err != nil {
		return fmt.Errorf("failed to submit transaction: %w", err)
	}

	// 14. Display result
	fmt.Println("✅ Server registered successfully!")
	fmt.Printf("Transaction Hash: %s\n", response.TransactionHash)
	fmt.Printf("Status: %s\n", response.Status)
	fmt.Printf("Server Endpoint: %s\n", endpoint)
	fmt.Printf("Auto-Register Capabilities: %t\n", autoRegisterCapabilities)

	// Generate server URI
	serverURI := fmt.Sprintf("knirv://server/%s", response.TransactionHash)
	fmt.Printf("Server URI: %s\n", serverURI)

	// 15. Register capabilities if requested
	if autoRegisterCapabilities {
		log.Info("Auto-registering server capabilities...")
		// This would be implemented in a future phase
		log.Info("Auto-registration of capabilities will be implemented in a future phase")
	}

	return nil
}

// loadServerSchema loads a server schema from a file
func loadServerSchema(filePath string) (map[string]interface{}, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read server schema file: %w", err)
	}

	// Parse JSON
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse server schema JSON: %w", err)
	}

	// Validate required fields
	requiredFields := []string{"name", "version", "description"}
	for _, field := range requiredFields {
		if _, ok := schema[field]; !ok {
			return nil, fmt.Errorf("missing required field in server schema: %s", field)
		}
	}

	return schema, nil
}

// validateServerEndpoint validates a server endpoint
func validateServerEndpoint(endpoint string) error {
	// Basic validation
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		return fmt.Errorf("endpoint must start with http:// or https://")
	}

	// Check for valid URL format
	if strings.Contains(endpoint, " ") {
		return fmt.Errorf("endpoint URL cannot contain spaces")
	}

	// Ensure endpoint ends with a path
	if strings.HasSuffix(endpoint, "/") {
		return fmt.Errorf("endpoint should not end with a trailing slash")
	}

	return nil
}

// testServerConnection tests the connection to a server
func testServerConnection(endpoint string) error {
	// Create a simple HTTP client to test the connection
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Try to connect to the server
	resp, err := client.Get(endpoint + "/health")
	if err != nil {
		// Try the root endpoint if /health fails
		resp, err = client.Get(endpoint)
		if err != nil {
			return fmt.Errorf("failed to connect to server: %w", err)
		}
	}
	defer resp.Body.Close()

	// Check if we got a reasonable response
	if resp.StatusCode >= 500 {
		return fmt.Errorf("server returned error status: %d", resp.StatusCode)
	}

	return nil
}

// deployMCPServer deploys an MCP server
func deployMCPServer(configPath, endpoint string) error {
	// This would be implemented in a future phase
	// For now, we'll just validate that the config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("deployment configuration file not found: %s", configPath)
	}

	log.Info("Server deployment will be implemented in a future phase")
	return nil
}
