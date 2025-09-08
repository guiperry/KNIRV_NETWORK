package cmd

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/guiperry/KNIRVCHAIN-CLI/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// walletCmd represents the wallet command
var walletCmd = &cobra.Command{
	Use:   "wallet",
	Short: "Manage KNIRVCHAIN wallets",
	Long: `Manage KNIRVCHAIN wallets including creation, import, export, and listing.
These commands provide secure wallet management with encryption and key derivation.`,
}

// walletNewCmd represents the wallet new command
var walletNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Generate a new wallet",
	Long: `Generate a new ECDSA key pair and store it securely.
The private key is encrypted with a password using AES-256-GCM.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get wallet directory from config
		walletDir := viper.GetString("wallet_directory")
		if walletDir == "" {
			log.Fatal("Wallet directory not configured")
		}

		// Create wallet manager
		walletManager := core.NewWalletManager(walletDir, log)

		// Generate new key pair
		log.Info("Generating new ECDSA key pair...")
		privateKey, err := walletManager.GenerateKeyPair()
		if err != nil {
			log.Fatalf("Failed to generate key pair: %v", err)
		}

		// Get wallet address
		address := core.GetAddressFromPrivateKey(privateKey)
		log.Infof("Generated new wallet with address: %s", address)

		// Check if wallet already exists
		if walletManager.WalletExists(address) {
			overwrite, _ := cmd.Flags().GetBool("overwrite")
			if !overwrite {
				log.Fatalf("Wallet with address %s already exists. Use --overwrite to overwrite", address)
			}
			log.Warnf("Overwriting existing wallet with address: %s", address)
		}

		// Get password
		noPassword, _ := cmd.Flags().GetBool("no-password")
		password, _ := cmd.Flags().GetString("password")

		if !noPassword && password == "" {
			log.Warn("No password provided. Use --password to specify a password or --no-password for unencrypted wallet")
			log.Fatal("Password required for wallet encryption")
		}

		if noPassword {
			log.Warn("Creating unencrypted wallet. This is not recommended for production use.")
			password = ""
		}

		// Save wallet
		filePath, _ := cmd.Flags().GetString("file")
		if filePath == "" {
			filePath = walletManager.GetWalletPath(address)
		}

		log.Infof("Saving wallet to %s...", filePath)
		if err := walletManager.SaveWallet(privateKey, password, filePath); err != nil {
			log.Fatalf("Failed to save wallet: %v", err)
		}

		log.Infof("Wallet created successfully with address: %s", address)
		log.Infof("Public key: %x", crypto.FromECDSAPub(&privateKey.PublicKey))
	},
}

// walletImportCmd represents the wallet import command
var walletImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import an existing private key",
	Long: `Import an existing private key and store it securely.
The private key is encrypted with a password using AES-256-GCM.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get wallet directory from config
		walletDir := viper.GetString("wallet_directory")
		if walletDir == "" {
			log.Fatal("Wallet directory not configured")
		}

		// Create wallet manager
		walletManager := core.NewWalletManager(walletDir, log)

		// Get private key
		privateKeyHex, _ := cmd.Flags().GetString("private-key")
		if privateKeyHex == "" {
			log.Fatal("Private key is required. Use --private-key to specify a private key")
		}

		// Parse private key
		log.Info("Parsing private key...")
		privateKey, err := core.ParsePrivateKey(privateKeyHex)
		if err != nil {
			log.Fatalf("Failed to parse private key: %v", err)
		}

		// Get wallet address
		address := core.GetAddressFromPrivateKey(privateKey)
		log.Infof("Imported wallet with address: %s", address)

		// Check if wallet already exists
		if walletManager.WalletExists(address) {
			log.Warnf("Wallet with address %s already exists", address)
			overwrite, _ := cmd.Flags().GetBool("overwrite")
			if !overwrite {
				log.Fatal("Use --overwrite to overwrite existing wallet")
			}
			log.Warnf("Overwriting existing wallet with address: %s", address)
		}

		// Get password
		noPassword, _ := cmd.Flags().GetBool("no-password")
		password, _ := cmd.Flags().GetString("password")

		if !noPassword && password == "" {
			log.Warn("No password provided. Use --password to specify a password or --no-password for unencrypted wallet")
			log.Fatal("Password required for wallet encryption")
		}

		if noPassword {
			log.Warn("Creating unencrypted wallet. This is not recommended for production use.")
			password = ""
		}

		// Save wallet
		filePath, _ := cmd.Flags().GetString("file")
		if filePath == "" {
			filePath = walletManager.GetWalletPath(address)
		}

		log.Infof("Saving wallet to %s...", filePath)
		if err := walletManager.SaveWallet(privateKey, password, filePath); err != nil {
			log.Fatalf("Failed to save wallet: %v", err)
		}

		log.Infof("Wallet imported successfully with address: %s", address)
		log.Infof("Public key: %x", crypto.FromECDSAPub(&privateKey.PublicKey))
	},
}

// walletListCmd represents the wallet list command
var walletListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available wallets",
	Long: `List all available wallets in the wallet directory.
Displays wallet addresses and file paths.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get wallet directory from config
		walletDir := viper.GetString("wallet_directory")
		customDir, _ := cmd.Flags().GetString("directory")
		if customDir != "" {
			walletDir = customDir
		}

		if walletDir == "" {
			log.Fatal("Wallet directory not configured")
		}

		// Create wallet manager
		walletManager := core.NewWalletManager(walletDir, log)

		// List wallets
		wallets, err := walletManager.ListWallets()
		if err != nil {
			log.Fatalf("Failed to list wallets: %v", err)
		}

		if len(wallets) == 0 {
			log.Info("No wallets found")
			return
		}

		// Get output format
		format, _ := cmd.Flags().GetString("format")
		showPaths, _ := cmd.Flags().GetBool("show-paths")

		switch format {
		case "table":
			log.Infof("Found %d wallets:", len(wallets))
			for i, address := range wallets {
				if showPaths {
					log.Infof("%d. %s (%s)", i+1, address, walletManager.GetWalletPath(address))
				} else {
					log.Infof("%d. %s", i+1, address)
				}
			}
		case "json":
			type WalletInfo struct {
				Address  string `json:"address"`
				FilePath string `json:"file_path,omitempty"`
			}

			var walletInfos []WalletInfo
			for _, address := range wallets {
				info := WalletInfo{Address: address}
				if showPaths {
					info.FilePath = walletManager.GetWalletPath(address)
				}
				walletInfos = append(walletInfos, info)
			}

			jsonData, err := json.MarshalIndent(walletInfos, "", "  ")
			if err != nil {
				log.Fatalf("Failed to marshal wallet info: %v", err)
			}
			fmt.Println(string(jsonData))
		case "csv":
			fmt.Println("address,file_path")
			for _, address := range wallets {
				if showPaths {
					fmt.Printf("%s,%s\n", address, walletManager.GetWalletPath(address))
				} else {
					fmt.Printf("%s,\n", address)
				}
			}
		default:
			log.Fatalf("Unsupported format: %s", format)
		}
	},
}

// walletExportCmd represents the wallet export command
var walletExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export wallet private key",
	Long: `Export a wallet's private key with proper security warnings.
Requires password confirmation.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get wallet directory from config
		walletDir := viper.GetString("wallet_directory")
		if walletDir == "" {
			log.Fatal("Wallet directory not configured")
		}

		// Create wallet manager
		walletManager := core.NewWalletManager(walletDir, log)

		// Get wallet file
		address, _ := cmd.Flags().GetString("address")
		filePath, _ := cmd.Flags().GetString("file")

		if address == "" && filePath == "" {
			log.Fatal("Either --address or --file must be specified")
		}

		if address != "" && filePath == "" {
			filePath = walletManager.GetWalletPath(address)
		}

		// Check if wallet file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Fatalf("Wallet file not found: %s", filePath)
		}

		// Get password
		password, _ := cmd.Flags().GetString("password")
		if password == "" {
			log.Fatal("Password is required. Use --password to specify a password")
		}

		// Display security warning
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			log.Warn("SECURITY WARNING: Exporting your private key is a security risk.")
			log.Warn("Anyone with your private key can access your wallet and steal your funds.")
			log.Warn("Only export your private key in a secure environment.")
			log.Warn("Use --force to skip this warning.")
			log.Fatal("Export aborted for security reasons. Use --force to proceed.")
		}

		// Load wallet
		log.Info("Decrypting wallet...")
		privateKey, err := walletManager.LoadWallet(filePath, password)
		if err != nil {
			log.Fatalf("Failed to load wallet: %v", err)
		}

		// Get export format
		format, _ := cmd.Flags().GetString("format")
		toFile, _ := cmd.Flags().GetBool("to-file")
		outputFile, _ := cmd.Flags().GetString("output")

		// Export private key
		var output string
		switch format {
		case "hex":
			output = "0x" + hex.EncodeToString(crypto.FromECDSA(privateKey))
		case "json":
			type PrivateKeyExport struct {
				PrivateKey string `json:"private_key"`
				Address    string `json:"address"`
				PublicKey  string `json:"public_key"`
			}

			export := PrivateKeyExport{
				PrivateKey: "0x" + hex.EncodeToString(crypto.FromECDSA(privateKey)),
				Address:    core.GetAddressFromPrivateKey(privateKey),
				PublicKey:  "0x" + hex.EncodeToString(crypto.FromECDSAPub(&privateKey.PublicKey)),
			}

			jsonData, err := json.MarshalIndent(export, "", "  ")
			if err != nil {
				log.Fatalf("Failed to marshal private key: %v", err)
			}
			output = string(jsonData)
		case "pem":
			log.Fatal("PEM format not implemented yet")
		default:
			log.Fatalf("Unsupported format: %s", format)
		}

		// Output private key
		if toFile {
			if outputFile == "" {
				log.Fatal("Output file is required when using --to-file")
			}

			if err := os.WriteFile(outputFile, []byte(output), 0600); err != nil {
				log.Fatalf("Failed to write output file: %v", err)
			}

			log.Infof("Private key exported to %s", outputFile)
		} else {
			log.Warn("SECURITY WARNING: Your private key is being displayed on the screen.")
			log.Warn("Make sure no one is watching your screen.")
			fmt.Println(output)
		}
	},
}

func init() {
	rootCmd.AddCommand(walletCmd)
	walletCmd.AddCommand(walletNewCmd)
	walletCmd.AddCommand(walletImportCmd)
	walletCmd.AddCommand(walletListCmd)
	walletCmd.AddCommand(walletExportCmd)

	// wallet new flags
	walletNewCmd.Flags().String("password", "", "Wallet encryption password (not recommended for production)")
	walletNewCmd.Flags().String("file", "", "Output file path")
	walletNewCmd.Flags().Bool("no-password", false, "Create unencrypted wallet (development only)")
	walletNewCmd.Flags().Bool("overwrite", false, "Overwrite existing wallet file")

	// wallet import flags
	walletImportCmd.Flags().String("private-key", "", "Private key in hex format")
	walletImportCmd.Flags().String("password", "", "Wallet encryption password (not recommended for production)")
	walletImportCmd.Flags().String("file", "", "Output file path")
	walletImportCmd.Flags().Bool("no-password", false, "Create unencrypted wallet (development only)")
	walletImportCmd.Flags().Bool("overwrite", false, "Overwrite existing wallet file")

	// wallet list flags
	walletListCmd.Flags().Bool("show-paths", false, "Show file paths")
	walletListCmd.Flags().String("directory", "", "Specify custom directory to scan")
	walletListCmd.Flags().String("format", "table", "Output format (table, json, csv)")

	// wallet export flags
	walletExportCmd.Flags().String("address", "", "Wallet address to export")
	walletExportCmd.Flags().String("file", "", "Wallet file to export")
	walletExportCmd.Flags().String("password", "", "Wallet password")
	walletExportCmd.Flags().Bool("to-file", false, "Export to file instead of stdout")
	walletExportCmd.Flags().String("output", "", "Output file path for exported key")
	walletExportCmd.Flags().String("format", "hex", "Output format (hex, json, pem)")
	walletExportCmd.Flags().Bool("force", false, "Skip security warnings")
}
