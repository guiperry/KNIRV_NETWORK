package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/core"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var log = logrus.New()

// CapabilityCmd represents the capability command
var CapabilityCmd = &cobra.Command{
	Use:   "capability",
	Short: "Manage direct plugin capability registration",
	Long: `Register and manage direct plugin capabilities on the KNIRVCHAIN blockchain.
This command handles the direct plugin integration route.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// MCPRegisterCapabilityData represents the data for registering a capability
type MCPRegisterCapabilityData struct {
	CapabilityType string                 `json:"capabilityType"`
	Descriptor     map[string]interface{} `json:"descriptor"`
	FileReferences []core.FileReference   `json:"fileReferences,omitempty"`
}

// MCPPrepareCapabilityRegistrationRequest represents the request for preparing a capability registration
type MCPPrepareCapabilityRegistrationRequest struct {
	FromAddress    string                 `json:"fromAddress"`
	CapabilityType string                 `json:"capabilityType"`
	Descriptor     map[string]interface{} `json:"descriptor"`
	Fee            uint64                 `json:"fee"`
}

// MCPPrepareCapabilityRegistrationResponse represents the response from preparing a capability registration
type MCPPrepareCapabilityRegistrationResponse struct {
	CapabilityID                 string                          `json:"capabilityId"`
	TransactionDetailsForSigning core.UnsignedTransactionDetails `json:"transactionDetailsForSigning"`
}

// capabilityRegisterCmd represents the capability register command
var capabilityRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new capability",
	Long: `Register a new capability on the KNIRVCHAIN blockchain.
This command validates the plugin file and registers it on the blockchain.

For plugin resources, this command will validate the plugin .so file and opSchema file,
create references to these files in the descriptor, and add location hints for peer access.

Example:
  knirv mcp capability register --node http://localhost:8080 --wallet ./my-wallet.json \
    --from 0x123abc... --type RESOURCE --descriptor ./descriptor.json \
    --plugin ./plugin.so --opschema ./opschema.yaml --fee 100`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return registerCapability(cmd, args)
	},
}

// capabilityListCmd represents the capability list command
var capabilityListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered capabilities",
	Long: `List all registered capabilities on the KNIRVCHAIN blockchain.
Displays capability IDs, names, and descriptions.`,
	Run: func(cmd *cobra.Command, args []string) {
		// This will be implemented in a future phase
		log.Info("Capability listing will be implemented in a future phase")
	},
}

// capabilityInfoCmd represents the capability info command
var capabilityInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get information about a capability",
	Long: `Get detailed information about a registered capability.
Displays capability metadata, parameters, and usage information.`,
	Run: func(cmd *cobra.Command, args []string) {
		// This will be implemented in a future phase
		log.Info("Capability info will be implemented in a future phase")
	},
}

func init() {
	CapabilityCmd.AddCommand(capabilityRegisterCmd)
	CapabilityCmd.AddCommand(capabilityListCmd)
	CapabilityCmd.AddCommand(capabilityInfoCmd)

	// capability register flags
	capabilityRegisterCmd.Flags().String("node", "", "URL of the KNIRVCHAIN node")
	capabilityRegisterCmd.Flags().String("wallet", "", "Path to wallet file")
	capabilityRegisterCmd.Flags().String("from", "", "Address of the sender")
	capabilityRegisterCmd.Flags().String("type", "", "Capability type (RESOURCE, TOOL, etc.)")
	capabilityRegisterCmd.Flags().String("descriptor", "", "JSON string or path to descriptor file")
	capabilityRegisterCmd.Flags().Uint64("fee", 0, "Transaction fee")

	// Plugin-specific flags
	capabilityRegisterCmd.Flags().String("plugin", "", "Path to plugin .so file (required for RESOURCE type)")
	capabilityRegisterCmd.Flags().String("opschema", "", "Path to opSchema file (required for RESOURCE type)")

	// Optional flags
	capabilityRegisterCmd.Flags().String("password", "", "Wallet password (will prompt if not provided)")
	capabilityRegisterCmd.Flags().StringSlice("location-hint", nil, "Additional location hints for file access")
	capabilityRegisterCmd.Flags().String("relative-path-base", "", "Base directory for relative paths")
	capabilityRegisterCmd.Flags().Bool("start-file-server", false, "Start a local file server for hosting plugin files")
	capabilityRegisterCmd.Flags().Int("file-server-port", 8090, "Port for the local file server")

	// Mark required flags
	capabilityRegisterCmd.MarkFlagRequired("node")
	capabilityRegisterCmd.MarkFlagRequired("wallet")
	capabilityRegisterCmd.MarkFlagRequired("from")
	capabilityRegisterCmd.MarkFlagRequired("type")
	capabilityRegisterCmd.MarkFlagRequired("descriptor")
	capabilityRegisterCmd.MarkFlagRequired("fee")

	// capability list flags
	capabilityListCmd.Flags().String("owner", "", "Filter by owner address")
	capabilityListCmd.Flags().String("type", "", "Filter by capability type")
	capabilityListCmd.Flags().String("format", "table", "Output format (table, json, csv)")

	// capability info flags
	capabilityInfoCmd.Flags().String("id", "", "Capability ID")
	capabilityInfoCmd.Flags().String("format", "table", "Output format (table, json, csv)")
}

// registerCapability implements the capability register command
func registerCapability(cmd *cobra.Command, args []string) error {
	// 1. Parse and validate flags
	nodeURL, _ := cmd.Flags().GetString("node")
	walletPath, _ := cmd.Flags().GetString("wallet")
	fromAddress, _ := cmd.Flags().GetString("from")
	capabilityType, _ := cmd.Flags().GetString("type")
	descriptorInput, _ := cmd.Flags().GetString("descriptor")
	pluginPath, _ := cmd.Flags().GetString("plugin")
	opschemaPath, _ := cmd.Flags().GetString("opschema")
	fee, _ := cmd.Flags().GetUint64("fee")
	password, _ := cmd.Flags().GetString("password")
	startFileServer, _ := cmd.Flags().GetBool("start-file-server")
	fileServerPort, _ := cmd.Flags().GetInt("file-server-port")
	relativePathBase, _ := cmd.Flags().GetString("relative-path-base")
	locationHints, _ := cmd.Flags().GetStringSlice("location-hint")

	// 2. Validate inputs based on capability type
	if capabilityType == "RESOURCE" {
		if pluginPath == "" || opschemaPath == "" {
			return fmt.Errorf("plugin and opschema files are required for RESOURCE capability type")
		}
	}

	// 3. Load and parse descriptor
	descriptor, err := loadDescriptor(descriptorInput)
	if err != nil {
		return fmt.Errorf("failed to load descriptor: %w", err)
	}

	// 4. Set up file reference strategy
	baseDir := relativePathBase
	if baseDir == "" {
		// Use current directory as default
		baseDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Create file manager
	fileManager, err := core.NewFileManager(baseDir)
	if err != nil {
		return fmt.Errorf("failed to create file manager: %w", err)
	}

	// Set up file reference strategy
	if startFileServer {
		// Use local file server strategy
		config := core.FileReferenceConfig{
			Strategy:        "local",
			BaseDir:         baseDir,
			ServerPort:      fileServerPort,
			AdditionalHints: locationHints,
			ValidateAccess:  true,
		}
		if err := fileManager.SetFileReferenceStrategy(config); err != nil {
			return fmt.Errorf("failed to set up file reference strategy: %w", err)
		}
		log.Infof("Started local file server on port %d", fileServerPort)
	} else if len(locationHints) > 0 {
		// Use web server strategy with the first location hint as base URL
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
		log.Infof("Using web server strategy with base URL: %s", locationHints[0])
	}

	// 5. Validate plugin and opschema files
	var fileRefs []core.FileReference
	if capabilityType == "RESOURCE" {
		// Validate plugin file
		log.Info("Validating plugin file...")
		if err := fileManager.ValidatePluginFile(pluginPath); err != nil {
			return fmt.Errorf("invalid plugin file: %w", err)
		}

		// Validate opschema file
		log.Info("Validating opschema file...")
		if err := fileManager.ValidateManifestFile(opschemaPath); err != nil {
			return fmt.Errorf("invalid opschema file: %w", err)
		}

		// Generate file references
		log.Info("Generating file references...")
		pluginRef, err := fileManager.GenerateFileReference(pluginPath)
		if err != nil {
			return fmt.Errorf("failed to generate plugin file reference: %w", err)
		}

		opschemaRef, err := fileManager.GenerateFileReference(opschemaPath)
		if err != nil {
			return fmt.Errorf("failed to generate opschema file reference: %w", err)
		}

		// Add file references to descriptor
		descriptor["pluginFile"] = pluginRef.LocationHint
		descriptor["pluginFileHash"] = pluginRef.ContentHash
		descriptor["opSchemaFile"] = opschemaRef.LocationHint
		descriptor["opSchemaFileHash"] = opschemaRef.ContentHash

		// Store file references for transaction data
		fileRefs = append(fileRefs, *pluginRef, *opschemaRef)

		log.Infof("Plugin file reference: %s (hash: %s)", pluginRef.LocationHint, pluginRef.ContentHash[:16]+"...")
		log.Infof("OpSchema file reference: %s (hash: %s)", opschemaRef.LocationHint, opschemaRef.ContentHash[:16]+"...")
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

	// 9. Create capability data
	capabilityData := MCPRegisterCapabilityData{
		CapabilityType: capabilityType,
		Descriptor:     descriptor,
		FileReferences: fileRefs,
	}

	// 10. Create transaction
	log.Info("Creating capability registration transaction...")
	unsignedTx, err := core.CreateTransaction(
		fromAddress,
		"", // No recipient for capability registration
		0,  // No value transfer
		capabilityData,
		fee,
		"REGISTER_CAPABILITY",
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// 11. Sign transaction
	log.Info("Signing transaction...")
	dataBytes, err := json.Marshal(capabilityData)
	if err != nil {
		return fmt.Errorf("failed to marshal capability data: %w", err)
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
	fmt.Println("✅ Capability registered successfully!")
	fmt.Printf("Transaction Hash: %s\n", response.TransactionHash)
	fmt.Printf("Status: %s\n", response.Status)

	if capabilityType == "RESOURCE" {
		fmt.Printf("Plugin File: %s\n", pluginPath)
		fmt.Printf("OpSchema File: %s\n", opschemaPath)
		fmt.Printf("Plugin Reference: %s\n", descriptor["pluginFile"])
		fmt.Printf("OpSchema Reference: %s\n", descriptor["opSchemaFile"])
	}

	// Generate capability URI
	capabilityURI := generateCapabilityURI(response.TransactionHash, capabilityType)
	fmt.Printf("Capability URI: %s\n", capabilityURI)

	return nil
}

// loadDescriptor loads a descriptor from a string or file
func loadDescriptor(input string) (map[string]interface{}, error) {
	var descriptor map[string]interface{}

	// Check if input is a file path
	if _, err := os.Stat(input); err == nil {
		// Read file
		data, err := os.ReadFile(input)
		if err != nil {
			return nil, fmt.Errorf("failed to read descriptor file: %w", err)
		}

		// Parse JSON
		if err := json.Unmarshal(data, &descriptor); err != nil {
			return nil, fmt.Errorf("failed to parse descriptor JSON: %w", err)
		}
	} else {
		// Parse JSON string
		if err := json.Unmarshal([]byte(input), &descriptor); err != nil {
			return nil, fmt.Errorf("failed to parse descriptor JSON: %w", err)
		}
	}

	return descriptor, nil
}

// generateCapabilityURI generates a URI for a capability
func generateCapabilityURI(capabilityID, capabilityType string) string {
	return fmt.Sprintf("knirv://%s/%s", strings.ToLower(capabilityType), capabilityID)
}
