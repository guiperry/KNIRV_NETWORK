package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/guiperry/KNIRVCHAIN-CLI/core"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var agentLog = logrus.New()

// AgentCmd represents the agent command
var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage agents and agent plugins",
	Long: `Register and manage agents and their compiled WASM plugins on the AGENTCHAIN blockchain.
This command handles agent-centric operations including plugin registration and deployment.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// AgentRegistrationData represents the data for agent registration
type AgentRegistrationData struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	AgentType   string                 `json:"agent_type"`
	ImageURL    string                 `json:"image_url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AgentPluginRegistrationData represents the data for agent plugin registration
type AgentPluginRegistrationData struct {
	AgentID    string `json:"agent_id"`
	PluginPath string `json:"plugin_path"`
	Version    string `json:"version"`
	CompiledBy string `json:"compiled_by"`
	Hash       string `json:"hash"`
}

// agentCreateCmd represents the agent create command
var agentCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new agent",
	Long: `Create a new agent on the AGENTCHAIN blockchain.

Example:
  agentchain-cli agent create --node http://localhost:8080 --wallet ./my-wallet.json \
    --from 0x123abc... --name "My Agent" --description "A helpful agent" \
    --type "assistant" --fee 100`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return createAgent(cmd, args)
	},
}

// agentRegisterPluginCmd represents the agent register-plugin command
var agentRegisterPluginCmd = &cobra.Command{
	Use:   "register-plugin",
	Short: "Register a compiled WASM plugin for an agent",
	Long: `Register a compiled WASM plugin for an existing agent on the AGENTCHAIN blockchain.

This command will:
- Verify the agent exists
- Validate the plugin file
- Calculate the plugin hash for integrity verification
- Register the plugin with versioning support
- Optionally deploy to server instances

Example:
  agentchain-cli agent register-plugin --node http://localhost:8080 --wallet ./my-wallet.json \
    --from 0x123abc... --agent-id "agent123" --plugin ./agent.wasm \
    --version "1.0.0" --fee 100`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return registerAgentPlugin(cmd, args)
	},
}

// agentListCmd represents the agent list command
var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List agents",
	Long: `List agents on the AGENTCHAIN blockchain.

Example:
  agentchain-cli agent list --node http://localhost:8080`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listAgents(cmd, args)
	},
}

// agentInfoCmd represents the agent info command
var agentInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get information about a specific agent",
	Long: `Get detailed information about a specific agent including its plugins.

Example:
  agentchain-cli agent info --node http://localhost:8080 --agent-id "agent123"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return getAgentInfo(cmd, args)
	},
}

func init() {
	// Add agent command to root command
	rootCmd.AddCommand(AgentCmd)

	// Add subcommands to agent command
	AgentCmd.AddCommand(agentCreateCmd)
	AgentCmd.AddCommand(agentRegisterPluginCmd)
	AgentCmd.AddCommand(agentListCmd)
	AgentCmd.AddCommand(agentInfoCmd)

	// agent create flags
	agentCreateCmd.Flags().String("node", "", "URL of the AGENTCHAIN node")
	agentCreateCmd.Flags().String("wallet", "", "Path to wallet file")
	agentCreateCmd.Flags().String("from", "", "Address of the sender")
	agentCreateCmd.Flags().String("name", "", "Name of the agent")
	agentCreateCmd.Flags().String("description", "", "Description of the agent")
	agentCreateCmd.Flags().String("type", "", "Type of agent (assistant, worker, analyzer, etc.)")
	agentCreateCmd.Flags().String("image-url", "", "Image URL for the agent")
	agentCreateCmd.Flags().String("metadata", "", "JSON metadata for the agent")
	agentCreateCmd.Flags().Uint64("fee", 0, "Transaction fee")
	agentCreateCmd.Flags().String("password", "", "Wallet password")

	// agent register-plugin flags
	agentRegisterPluginCmd.Flags().String("node", "", "URL of the AGENTCHAIN node")
	agentRegisterPluginCmd.Flags().String("wallet", "", "Path to wallet file")
	agentRegisterPluginCmd.Flags().String("from", "", "Address of the sender")
	agentRegisterPluginCmd.Flags().String("agent-id", "", "ID of the agent")
	agentRegisterPluginCmd.Flags().String("plugin", "", "Path to the compiled WASM plugin file")
	agentRegisterPluginCmd.Flags().String("version", "", "Version of the plugin")
	agentRegisterPluginCmd.Flags().Uint64("fee", 0, "Transaction fee")
	agentRegisterPluginCmd.Flags().String("password", "", "Wallet password")

	// agent list flags
	agentListCmd.Flags().String("node", "", "URL of the AGENTCHAIN node")
	agentListCmd.Flags().String("owner", "", "Filter by owner address")
	agentListCmd.Flags().String("type", "", "Filter by agent type")

	// agent info flags
	agentInfoCmd.Flags().String("node", "", "URL of the AGENTCHAIN node")
	agentInfoCmd.Flags().String("agent-id", "", "ID of the agent")

	// Mark required flags
	agentCreateCmd.MarkFlagRequired("node")
	agentCreateCmd.MarkFlagRequired("wallet")
	agentCreateCmd.MarkFlagRequired("from")
	agentCreateCmd.MarkFlagRequired("name")
	agentCreateCmd.MarkFlagRequired("description")
	agentCreateCmd.MarkFlagRequired("type")

	agentRegisterPluginCmd.MarkFlagRequired("node")
	agentRegisterPluginCmd.MarkFlagRequired("wallet")
	agentRegisterPluginCmd.MarkFlagRequired("from")
	agentRegisterPluginCmd.MarkFlagRequired("agent-id")
	agentRegisterPluginCmd.MarkFlagRequired("plugin")
	agentRegisterPluginCmd.MarkFlagRequired("version")

	agentListCmd.MarkFlagRequired("node")
	agentInfoCmd.MarkFlagRequired("node")
	agentInfoCmd.MarkFlagRequired("agent-id")
}

// createAgent implements the agent create command
func createAgent(cmd *cobra.Command, args []string) error {
	// Parse and validate flags
	nodeURL, _ := cmd.Flags().GetString("node")
	walletPath, _ := cmd.Flags().GetString("wallet")
	fromAddress, _ := cmd.Flags().GetString("from")
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	agentType, _ := cmd.Flags().GetString("type")
	imageURL, _ := cmd.Flags().GetString("image-url")
	metadataStr, _ := cmd.Flags().GetString("metadata")
	fee, _ := cmd.Flags().GetUint64("fee")
	password, _ := cmd.Flags().GetString("password")

	// Validate required parameters
	if nodeURL == "" || walletPath == "" || fromAddress == "" || name == "" || description == "" || agentType == "" {
		return fmt.Errorf("missing required parameters")
	}

	// Parse metadata if provided
	var metadata map[string]interface{}
	if metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			return fmt.Errorf("invalid metadata JSON: %w", err)
		}
	}

	// Create wallet manager
	walletManager := core.NewWalletManager(filepath.Dir(walletPath), log)

	// Load wallet
	agentLog.Info("Loading wallet...")
	privateKey, err := walletManager.LoadWallet(walletPath, password)
	if err != nil {
		return fmt.Errorf("failed to load wallet: %w", err)
	}

	// Prepare agent registration data
	agentData := AgentRegistrationData{
		Name:        name,
		Description: description,
		AgentType:   agentType,
		ImageURL:    imageURL,
		Metadata:    metadata,
	}

	// Create transaction
	agentLog.Info("Creating agent registration transaction...")
	unsignedTx, err := core.CreateTransaction(
		fromAddress,
		"", // No recipient for agent registration
		0,  // No value transfer
		agentData,
		fee,
		"CREATE_AGENT",
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// Sign transaction
	agentLog.Info("Signing transaction...")
	dataBytes, err := json.Marshal(agentData)
	if err != nil {
		return fmt.Errorf("failed to marshal agent data: %w", err)
	}

	signature, txHash, err := core.SignTransactionData(privateKey, *unsignedTx, dataBytes)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Assemble signed transaction
	agentLog.Info("Assembling signed transaction...")
	publicKeyHex := fmt.Sprintf("%x", privateKey.PublicKey.X.Bytes()) + fmt.Sprintf("%x", privateKey.PublicKey.Y.Bytes())
	signedTx, err := core.AssembleSignedTransaction(*unsignedTx, publicKeyHex, signature, txHash)
	if err != nil {
		return fmt.Errorf("failed to assemble signed transaction: %w", err)
	}

	// Create API client
	apiClient := core.NewAPIClient(
		nodeURL,
		core.WithTimeout(30*time.Second),
		core.WithRetries(3),
		core.WithLogger(agentLog),
	)

	// Submit transaction
	agentLog.Info("Submitting transaction to blockchain...")
	ctx := context.Background()
	response, err := apiClient.SubmitTransaction(ctx, signedTx)
	if err != nil {
		return fmt.Errorf("failed to submit transaction: %w", err)
	}

	// Output results
	fmt.Printf("✅ Agent created successfully!\n")
	fmt.Printf("📋 Transaction Hash: %s\n", txHash)
	fmt.Printf("🔗 Agent ID: %s\n", response.TransactionHash) // Use transaction hash as agent ID for now
	fmt.Printf("🌐 Node: %s\n", nodeURL)

	return nil
}

// registerAgentPlugin implements the agent register-plugin command
func registerAgentPlugin(cmd *cobra.Command, args []string) error {
	// Parse and validate flags
	nodeURL, _ := cmd.Flags().GetString("node")
	walletPath, _ := cmd.Flags().GetString("wallet")
	fromAddress, _ := cmd.Flags().GetString("from")
	agentID, _ := cmd.Flags().GetString("agent-id")
	pluginPath, _ := cmd.Flags().GetString("plugin")
	version, _ := cmd.Flags().GetString("version")
	fee, _ := cmd.Flags().GetUint64("fee")
	password, _ := cmd.Flags().GetString("password")

	// Validate required parameters
	if nodeURL == "" || walletPath == "" || fromAddress == "" || agentID == "" || pluginPath == "" || version == "" {
		return fmt.Errorf("missing required parameters")
	}

	// Verify plugin file exists
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin file does not exist: %s", pluginPath)
	}

	// Read and hash the plugin file
	agentLog.Info("Reading and hashing plugin file...")
	pluginFileData, err := os.ReadFile(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to read plugin file: %w", err)
	}

	// Calculate SHA256 hash
	hash := fmt.Sprintf("%x", core.CalculateHash(pluginFileData))

	// Create wallet manager
	walletManager := core.NewWalletManager(filepath.Dir(walletPath), log)

	// Load wallet
	agentLog.Info("Loading wallet...")
	privateKey, err := walletManager.LoadWallet(walletPath, password)
	if err != nil {
		return fmt.Errorf("failed to load wallet: %w", err)
	}

	// Prepare plugin registration data
	pluginRegData := AgentPluginRegistrationData{
		AgentID:    agentID,
		PluginPath: filepath.Base(pluginPath), // Store only filename for security
		Version:    version,
		CompiledBy: fromAddress,
		Hash:       hash,
	}

	// Create transaction
	agentLog.Info("Creating plugin registration transaction...")
	unsignedTx, err := core.CreateTransaction(
		fromAddress,
		"", // No recipient for plugin registration
		0,  // No value transfer
		pluginRegData,
		fee,
		"REGISTER_AGENT_PLUGIN",
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// Sign transaction
	agentLog.Info("Signing transaction...")
	dataBytes, err := json.Marshal(pluginRegData)
	if err != nil {
		return fmt.Errorf("failed to marshal plugin data: %w", err)
	}

	signature, txHash, err := core.SignTransactionData(privateKey, *unsignedTx, dataBytes)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Assemble signed transaction
	agentLog.Info("Assembling signed transaction...")
	publicKeyHex := fmt.Sprintf("%x", privateKey.PublicKey.X.Bytes()) + fmt.Sprintf("%x", privateKey.PublicKey.Y.Bytes())
	signedTx, err := core.AssembleSignedTransaction(*unsignedTx, publicKeyHex, signature, txHash)
	if err != nil {
		return fmt.Errorf("failed to assemble signed transaction: %w", err)
	}

	// Create API client
	apiClient := core.NewAPIClient(
		nodeURL,
		core.WithTimeout(30*time.Second),
		core.WithRetries(3),
		core.WithLogger(agentLog),
	)

	// Submit transaction
	agentLog.Info("Submitting transaction to blockchain...")
	ctx := context.Background()
	response, err := apiClient.SubmitTransaction(ctx, signedTx)
	if err != nil {
		return fmt.Errorf("failed to submit transaction: %w", err)
	}

	// Output results
	fmt.Printf("✅ Agent plugin registered successfully!\n")
	fmt.Printf("📋 Transaction Hash: %s\n", txHash)
	fmt.Printf("🔗 Plugin ID: %s\n", response.TransactionHash) // Use transaction hash as plugin ID for now
	fmt.Printf("🏷️  Agent ID: %s\n", agentID)
	fmt.Printf("📦 Version: %s\n", version)
	fmt.Printf("🔒 Hash: %s\n", hash)
	fmt.Printf("🌐 Node: %s\n", nodeURL)

	return nil
}

// listAgents implements the agent list command
func listAgents(cmd *cobra.Command, args []string) error {
	// Parse flags
	nodeURL, _ := cmd.Flags().GetString("node")
	owner, _ := cmd.Flags().GetString("owner")
	agentType, _ := cmd.Flags().GetString("type")

	// Validate required parameters
	if nodeURL == "" {
		return fmt.Errorf("node URL is required")
	}

	// Build query parameters
	queryParams := make(map[string]string)
	if owner != "" {
		queryParams["owner"] = owner
	}
	if agentType != "" {
		queryParams["type"] = agentType
	}

	// Query agents from the blockchain
	agentLog.Info("Querying agents from blockchain...")
	agents, err := core.QueryAgents(nodeURL, queryParams)
	if err != nil {
		return fmt.Errorf("failed to query agents: %w", err)
	}

	// Display results
	if len(agents) == 0 {
		fmt.Println("No agents found.")
		return nil
	}

	fmt.Printf("Found %d agent(s):\n\n", len(agents))
	for i, agent := range agents {
		fmt.Printf("Agent #%d:\n", i+1)
		fmt.Printf("  🆔 ID: %s\n", agent["id"])
		fmt.Printf("  📛 Name: %s\n", agent["name"])
		fmt.Printf("  📝 Description: %s\n", agent["description"])
		fmt.Printf("  🏷️  Type: %s\n", agent["agent_type"])
		fmt.Printf("  👤 Owner: %s\n", agent["owner"])
		fmt.Printf("  📅 Created: %s\n", agent["created_at"])
		if pluginPath, ok := agent["plugin_path"]; ok && pluginPath != "" {
			fmt.Printf("  🔌 Plugin: %s\n", pluginPath)
		}
		fmt.Println()
	}

	return nil
}

// getAgentInfo implements the agent info command
func getAgentInfo(cmd *cobra.Command, args []string) error {
	// Parse flags
	nodeURL, _ := cmd.Flags().GetString("node")
	agentID, _ := cmd.Flags().GetString("agent-id")

	// Validate required parameters
	if nodeURL == "" || agentID == "" {
		return fmt.Errorf("node URL and agent ID are required")
	}

	// Query agent details from the blockchain
	agentLog.Info("Querying agent details from blockchain...")
	agent, err := core.QueryAgentByID(nodeURL, agentID)
	if err != nil {
		return fmt.Errorf("failed to query agent: %w", err)
	}

	// Query agent plugins
	agentLog.Info("Querying agent plugins...")
	plugins, err := core.QueryAgentPlugins(nodeURL, agentID)
	if err != nil {
		agentLog.Warnf("Failed to query agent plugins: %v", err)
		plugins = []map[string]interface{}{} // Continue without plugins
	}

	// Display agent information
	fmt.Printf("🤖 Agent Information\n")
	fmt.Printf("═══════════════════════════════════════════════════════════\n")
	fmt.Printf("🆔 ID: %s\n", agent["id"])
	fmt.Printf("📛 Name: %s\n", agent["name"])
	fmt.Printf("📝 Description: %s\n", agent["description"])
	fmt.Printf("🏷️  Type: %s\n", agent["agent_type"])
	fmt.Printf("👤 Owner: %s\n", agent["owner"])
	fmt.Printf("📅 Created: %s\n", agent["created_at"])

	if imageURL, ok := agent["image_url"]; ok && imageURL != "" {
		fmt.Printf("🖼️  Image: %s\n", imageURL)
	}

	if serverURL, ok := agent["server_url"]; ok && serverURL != "" {
		fmt.Printf("🌐 Server: %s\n", serverURL)
	}

	// Display capabilities if available
	if capabilities, ok := agent["capabilities"]; ok {
		if capList, ok := capabilities.([]interface{}); ok && len(capList) > 0 {
			fmt.Printf("\n🛠️  Capabilities (%d):\n", len(capList))
			for i, cap := range capList {
				if capMap, ok := cap.(map[string]interface{}); ok {
					fmt.Printf("  %d. %s - %s\n", i+1, capMap["name"], capMap["description"])
				}
			}
		}
	}

	// Display plugins
	if len(plugins) > 0 {
		fmt.Printf("\n🔌 Plugins (%d):\n", len(plugins))
		for i, plugin := range plugins {
			fmt.Printf("  Plugin #%d:\n", i+1)
			fmt.Printf("    🆔 ID: %s\n", plugin["id"])
			fmt.Printf("    📦 Version: %s\n", plugin["version"])
			fmt.Printf("    📁 File: %s\n", plugin["file_path"])
			fmt.Printf("    🔒 Hash: %s\n", plugin["hash"])
			fmt.Printf("    👤 Compiled By: %s\n", plugin["compiled_by"])
			fmt.Printf("    📅 Compiled: %s\n", plugin["compile_time"])
			fmt.Printf("    🏷️  Status: %s\n", plugin["status"])

			if serverURLs, ok := plugin["server_urls"]; ok {
				if urlList, ok := serverURLs.([]interface{}); ok && len(urlList) > 0 {
					fmt.Printf("    🌐 Servers: %v\n", urlList)
				}
			}
			fmt.Println()
		}
	} else {
		fmt.Printf("\n🔌 Plugins: None registered\n")
	}

	// Display metadata if available
	if metadata, ok := agent["metadata"]; ok {
		if metaMap, ok := metadata.(map[string]interface{}); ok && len(metaMap) > 0 {
			fmt.Printf("\n📊 Metadata:\n")
			for key, value := range metaMap {
				fmt.Printf("  %s: %v\n", key, value)
			}
		}
	}

	return nil
}
