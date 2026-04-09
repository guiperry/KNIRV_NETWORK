package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/core"
	"github.com/spf13/cobra"
)

// ProcedureCmd represents the procedure command
var ProcedureCmd = &cobra.Command{
	Use:   "procedure",
	Short: "Manage operational procedure registration (interpolation)",
	Long: `Register and manage operational procedures on the KNIRVCHAIN blockchain.
This command handles the interpolation integration route.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// MCPRegisterProcedureData represents the data for registering a procedure
type MCPRegisterProcedureData struct {
	OpSchema       map[string]interface{} `json:"opSchema"`
	DeployServers  bool                   `json:"deployServers"`
	FileReferences []core.FileReference   `json:"fileReferences,omitempty"`
}

// procedureRegisterCmd represents the procedure register command
var procedureRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new operational procedure",
	Long: `Register a new operational procedure on the KNIRVCHAIN blockchain using the interpolation route.

This command will register the procedure, deploy any required MCP servers, and register
the capability on the blockchain.

Example:
  knirv mcp procedure register --node http://localhost:8080 --wallet ./my-wallet.json \
    --from 0x123abc... --opschema ./opschema.yaml --plugin ./plugin.so --fee 100 --deploy-servers`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return registerProcedure(cmd, args)
	},
}

// procedureListCmd represents the procedure list command
var procedureListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered operational procedures",
	Long: `List all registered operational procedures on the KNIRVCHAIN blockchain.
Displays procedure IDs, names, and descriptions.`,
	Run: func(cmd *cobra.Command, args []string) {
		// This will be implemented in a future phase
		log.Info("Procedure listing will be implemented in a future phase")
	},
}

// procedureInfoCmd represents the procedure info command
var procedureInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get information about an operational procedure",
	Long: `Get detailed information about a registered operational procedure.
Displays procedure metadata, steps, and parameters.`,
	Run: func(cmd *cobra.Command, args []string) {
		// This will be implemented in a future phase
		log.Info("Procedure info will be implemented in a future phase")
	},
}

func init() {
	ProcedureCmd.AddCommand(procedureRegisterCmd)
	ProcedureCmd.AddCommand(procedureListCmd)
	ProcedureCmd.AddCommand(procedureInfoCmd)

	// procedure register flags
	procedureRegisterCmd.Flags().String("node", "", "URL of the KNIRVCHAIN node")
	procedureRegisterCmd.Flags().String("wallet", "", "Path to wallet file")
	procedureRegisterCmd.Flags().String("from", "", "Address of the sender")
	procedureRegisterCmd.Flags().String("opschema", "", "Path to opSchema file")
	procedureRegisterCmd.Flags().Uint64("fee", 0, "Transaction fee")

	// Optional flags
	procedureRegisterCmd.Flags().String("password", "", "Wallet password (will prompt if not provided)")
	procedureRegisterCmd.Flags().String("plugin", "", "Path to plugin .so file (if procedure includes plugin functions)")
	procedureRegisterCmd.Flags().Bool("deploy-servers", false, "Deploy any MCP servers required by the procedure")
	procedureRegisterCmd.Flags().StringSlice("location-hint", nil, "Additional location hints for file access")
	procedureRegisterCmd.Flags().Bool("validate-servers", true, "Validate that all referenced MCP servers are accessible")

	// Mark required flags
	procedureRegisterCmd.MarkFlagRequired("node")
	procedureRegisterCmd.MarkFlagRequired("wallet")
	procedureRegisterCmd.MarkFlagRequired("from")
	procedureRegisterCmd.MarkFlagRequired("opschema")
	procedureRegisterCmd.MarkFlagRequired("fee")

	// procedure list flags
	procedureListCmd.Flags().String("owner", "", "Filter by owner address")
	procedureListCmd.Flags().String("format", "table", "Output format (table, json, csv)")

	// procedure info flags
	procedureInfoCmd.Flags().String("id", "", "Procedure ID")
	procedureInfoCmd.Flags().String("format", "table", "Output format (table, json, csv)")
}

// registerProcedure implements the procedure register command
func registerProcedure(cmd *cobra.Command, args []string) error {
	// 1. Parse and validate flags
	nodeURL, _ := cmd.Flags().GetString("node")
	walletPath, _ := cmd.Flags().GetString("wallet")
	fromAddress, _ := cmd.Flags().GetString("from")
	opschemaPath, _ := cmd.Flags().GetString("opschema")
	pluginPath, _ := cmd.Flags().GetString("plugin")
	fee, _ := cmd.Flags().GetUint64("fee")
	password, _ := cmd.Flags().GetString("password")
	deployServers, _ := cmd.Flags().GetBool("deploy-servers")
	locationHints, _ := cmd.Flags().GetStringSlice("location-hint")
	validateServers, _ := cmd.Flags().GetBool("validate-servers")

	// 2. Load and parse opschema
	log.Info("Loading operational schema...")
	opSchema, err := loadOpSchema(opschemaPath)
	if err != nil {
		return fmt.Errorf("failed to load opschema: %w", err)
	}

	// 3. Validate plugin file if provided
	var fileRefs []core.FileReference
	if pluginPath != "" {
		// Create file manager
		baseDir := filepath.Dir(opschemaPath)
		fileManager, err := core.NewFileManager(baseDir)
		if err != nil {
			return fmt.Errorf("failed to create file manager: %w", err)
		}

		// Set up file reference strategy if location hints are provided
		if len(locationHints) > 0 {
			config := core.FileReferenceConfig{
				Strategy:        "web",
				BaseDir:         baseDir,
				BaseURL:         locationHints[0],
				AdditionalHints: locationHints[1:],
				ValidateAccess:  true,
			}
			if err := fileManager.SetFileReferenceStrategy(config); err != nil {
				return fmt.Errorf("failed to set up file reference strategy: %w", err)
			}
		}

		// Validate plugin file
		log.Info("Validating plugin file...")
		if err := fileManager.ValidatePluginFile(pluginPath); err != nil {
			return fmt.Errorf("invalid plugin file: %w", err)
		}

		// Generate file reference
		pluginRef, err := fileManager.GenerateFileReference(pluginPath)
		if err != nil {
			return fmt.Errorf("failed to generate plugin file reference: %w", err)
		}

		// Add file reference
		fileRefs = append(fileRefs, *pluginRef)

		// Add plugin file reference to opschema
		if opSchema["pluginFile"] == nil {
			opSchema["pluginFile"] = pluginRef.LocationHint
		}
		if opSchema["pluginFileHash"] == nil {
			opSchema["pluginFileHash"] = pluginRef.ContentHash
		}

		log.Infof("Plugin file reference: %s (hash: %s)", pluginRef.LocationHint, pluginRef.ContentHash[:16]+"...")
	}

	// 4. Validate servers if required
	if validateServers {
		log.Info("Validating MCP servers...")
		if err := validateMCPServers(opSchema); err != nil {
			return fmt.Errorf("server validation failed: %w", err)
		}
		log.Info("✅ Server validation passed")
	}

	// 5. Deploy servers if requested
	if deployServers {
		log.Info("Deploying required MCP servers...")
		if err := deployRequiredServers(opSchema); err != nil {
			return fmt.Errorf("server deployment failed: %w", err)
		}
		log.Info("✅ Servers deployed successfully")
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

	// 9. Prepare procedure registration data
	procedureData := MCPRegisterProcedureData{
		OpSchema:       opSchema,
		DeployServers:  deployServers,
		FileReferences: fileRefs,
	}

	// 10. Create transaction
	log.Info("Creating procedure registration transaction...")
	unsignedTx, err := core.CreateTransaction(
		fromAddress,
		"", // No recipient for procedure registration
		0,  // No value transfer
		procedureData,
		fee,
		"REGISTER_PROCEDURE",
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// 11. Sign transaction
	log.Info("Signing transaction...")
	dataBytes, err := json.Marshal(procedureData)
	if err != nil {
		return fmt.Errorf("failed to marshal procedure data: %w", err)
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
	fmt.Println("✅ Procedure registered successfully!")
	fmt.Printf("Transaction Hash: %s\n", response.TransactionHash)
	fmt.Printf("Status: %s\n", response.Status)
	fmt.Printf("OpSchema File: %s\n", opschemaPath)

	if pluginPath != "" {
		fmt.Printf("Plugin File: %s\n", pluginPath)
	}

	if deployServers {
		fmt.Printf("Servers Deployed: %t\n", deployServers)
	}

	// Generate procedure URI
	procedureURI := fmt.Sprintf("knirv://procedure/%s", response.TransactionHash)
	fmt.Printf("Procedure URI: %s\n", procedureURI)

	return nil
}

// loadOpSchema loads an opschema from a file
func loadOpSchema(filePath string) (map[string]interface{}, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read opschema file: %w", err)
	}

	// Parse JSON
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse opschema JSON: %w", err)
	}

	// Validate required fields
	requiredFields := []string{"name", "version", "description"}
	for _, field := range requiredFields {
		if _, ok := schema[field]; !ok {
			return nil, fmt.Errorf("missing required field in opschema: %s", field)
		}
	}

	return schema, nil
}

// validateMCPServers validates that all MCP servers referenced in the opschema are accessible
func validateMCPServers(opSchema map[string]interface{}) error {
	// Check if the opschema contains server references
	servers, ok := opSchema["servers"]
	if !ok {
		// No servers to validate
		return nil
	}

	serverList, ok := servers.([]interface{})
	if !ok {
		return fmt.Errorf("servers field must be an array")
	}

	// Validate each server
	for i, server := range serverList {
		serverMap, ok := server.(map[string]interface{})
		if !ok {
			return fmt.Errorf("server %d must be an object", i)
		}

		endpoint, ok := serverMap["endpoint"].(string)
		if !ok {
			return fmt.Errorf("server %d must have an endpoint field", i)
		}

		// Test server connection
		if err := testServerConnection(endpoint); err != nil {
			return fmt.Errorf("server %d (%s) validation failed: %w", i, endpoint, err)
		}

		log.Infof("✅ Server %d (%s) validation passed", i, endpoint)
	}

	return nil
}

// deployRequiredServers deploys all MCP servers required by the procedure
func deployRequiredServers(opSchema map[string]interface{}) error {
	// Check if the opschema contains deployment configurations
	deployments, ok := opSchema["deployments"]
	if !ok {
		// No deployments to perform
		log.Info("No server deployments specified in opschema")
		return nil
	}

	deploymentList, ok := deployments.([]interface{})
	if !ok {
		return fmt.Errorf("deployments field must be an array")
	}

	// Deploy each server
	for i, deployment := range deploymentList {
		deploymentMap, ok := deployment.(map[string]interface{})
		if !ok {
			return fmt.Errorf("deployment %d must be an object", i)
		}

		name, ok := deploymentMap["name"].(string)
		if !ok {
			return fmt.Errorf("deployment %d must have a name field", i)
		}

		config, ok := deploymentMap["config"].(string)
		if !ok {
			return fmt.Errorf("deployment %d must have a config field", i)
		}

		endpoint, ok := deploymentMap["endpoint"].(string)
		if !ok {
			return fmt.Errorf("deployment %d must have an endpoint field", i)
		}

		log.Infof("Deploying server: %s", name)
		if err := deployMCPServer(config, endpoint); err != nil {
			return fmt.Errorf("failed to deploy server %s: %w", name, err)
		}

		log.Infof("✅ Server %s deployed successfully", name)
	}

	return nil
}
