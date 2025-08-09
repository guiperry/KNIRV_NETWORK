package cmd

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/guiperry/KNIRVCHAIN-CLI/config"
	"github.com/guiperry/KNIRVCHAIN-CLI/core"
	"github.com/spf13/cobra"
)

var (
	economicsCmd = &cobra.Command{
		Use:   "economics",
		Short: "NRN token and economics management",
		Long:  `Economics command provides operations for managing NRN tokens, economics data, and financial operations within the KNIRV Network.`,
	}

	economicsBalanceCmd = &cobra.Command{
		Use:   "balance [address]",
		Short: "Get NRN token balance",
		Long:  `Retrieve the NRN token balance for a specific address.`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  runEconomicsBalance,
	}

	economicsTransferCmd = &cobra.Command{
		Use:   "transfer <to> <amount>",
		Short: "Transfer NRN tokens",
		Long:  `Transfer NRN tokens to another address.`,
		Args:  cobra.ExactArgs(2),
		RunE:  runEconomicsTransfer,
	}

	economicsFaucetCmd = &cobra.Command{
		Use:   "faucet [amount]",
		Short: "Request NRN tokens from faucet",
		Long:  `Request NRN tokens from the faucet for testing purposes.`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  runEconomicsFaucet,
	}

	economicsHistoryCmd = &cobra.Command{
		Use:   "history",
		Short: "Show NRN transaction history",
		Long:  `Display the transaction history for NRN tokens.`,
		RunE:  runEconomicsHistory,
	}

	economicsStatsCmd = &cobra.Command{
		Use:   "stats",
		Short: "Show NRN token statistics",
		Long:  `Display statistics about NRN token usage and transactions.`,
		RunE:  runEconomicsStats,
	}

	economicsSkillsCmd = &cobra.Command{
		Use:   "skills",
		Short: "Manage skills in the economics system",
		Long:  `List and manage skills registered in the KNIRV economics system.`,
		RunE:  runEconomicsSkills,
	}

	economicsFeesCmd = &cobra.Command{
		Use:   "fees <operation> [complexity]",
		Short: "Estimate fees for operations",
		Long:  `Estimate fees for various operations in the KNIRV Network.`,
		Args:  cobra.RangeArgs(1, 2),
		RunE:  runEconomicsFees,
	}

	// Flags
	walletName       string
	includePending   bool
	autoRefill       bool
	minBalance       string
	tokenType        string
	ownedByMe        bool
	availableForRent bool
	complexity       string
)

func init() {
	rootCmd.AddCommand(economicsCmd)

	// Add subcommands
	economicsCmd.AddCommand(economicsBalanceCmd)
	economicsCmd.AddCommand(economicsTransferCmd)
	economicsCmd.AddCommand(economicsFaucetCmd)
	economicsCmd.AddCommand(economicsHistoryCmd)
	economicsCmd.AddCommand(economicsStatsCmd)
	economicsCmd.AddCommand(economicsSkillsCmd)
	economicsCmd.AddCommand(economicsFeesCmd)

	// Global flags
	economicsCmd.PersistentFlags().StringVar(&walletName, "wallet", "", "Wallet name to use")

	// Balance command flags
	economicsBalanceCmd.Flags().BoolVar(&includePending, "include-pending", false, "Include pending transactions")

	// Transfer command flags
	economicsTransferCmd.Flags().StringVar(&tokenType, "token", "NRN", "Token type to transfer")

	// Faucet command flags
	economicsFaucetCmd.Flags().BoolVar(&autoRefill, "auto-refill", false, "Enable auto-refill")
	economicsFaucetCmd.Flags().StringVar(&minBalance, "min-balance", "1000", "Minimum balance for auto-refill")

	// Skills command flags
	economicsSkillsCmd.Flags().BoolVar(&ownedByMe, "owned-by-me", false, "Show only skills owned by me")
	economicsSkillsCmd.Flags().BoolVar(&availableForRent, "available-for-rent", false, "Show only skills available for rent")

	// Fees command flags
	economicsFeesCmd.Flags().StringVar(&complexity, "complexity", "medium", "Operation complexity (low, medium, high)")
}

func runEconomicsBalance(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get wallet address
	var address string
	if len(args) > 0 {
		address = args[0]
	} else {
		// Get address from wallet
		if walletName == "" {
			return fmt.Errorf("wallet name or address required")
		}

		walletManager := core.NewWalletManager(cfg.KNIRV.Wallet.Directory, log)
		wallet, err := walletManager.GetWallet(walletName)
		if err != nil {
			return fmt.Errorf("failed to get wallet: %w", err)
		}
		address = wallet.Address
	}

	// Create KNIRVROOT client
	knirvRootClient := core.NewKNIRVRootClient(&cfg.KNIRV.Services.KNIRVRoot, log)

	// Connect to KNIRVROOT
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := knirvRootClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to KNIRVROOT: %w", err)
	}
	defer knirvRootClient.Disconnect()

	// Get balance
	balance, err := knirvRootClient.GetNRNBalance(ctx, address)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	fmt.Printf("NRN Balance for %s: %s\n", address, balance)

	if includePending {
		fmt.Println("Note: Pending transactions are not yet implemented")
	}

	return nil
}

func runEconomicsTransfer(cmd *cobra.Command, args []string) error {
	to := args[0]
	amountStr := args[1]

	// Parse amount
	amount, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		return fmt.Errorf("invalid amount: %s", amountStr)
	}

	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get wallet
	if walletName == "" {
		return fmt.Errorf("wallet name required")
	}

	walletManager := core.NewWalletManager(cfg.KNIRV.Wallet.Directory, log)
	wallet, err := walletManager.GetWallet(walletName)
	if err != nil {
		return fmt.Errorf("failed to get wallet: %w", err)
	}

	// Create KNIRVROOT client and NRN token manager
	knirvRootClient := core.NewKNIRVRootClient(&cfg.KNIRV.Services.KNIRVRoot, log)
	nrnManager := core.NewNRNTokenManager(&cfg.KNIRV.Wallet, knirvRootClient, log)

	// Connect to KNIRVROOT
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := knirvRootClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to KNIRVROOT: %w", err)
	}
	defer knirvRootClient.Disconnect()

	// Update balance first
	if err := nrnManager.UpdateBalance(ctx, wallet.Address); err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	// Perform transfer
	tx, err := nrnManager.Transfer(ctx, wallet.Address, to, amount)
	if err != nil {
		return fmt.Errorf("transfer failed: %w", err)
	}

	fmt.Printf("Transfer successful!\n")
	fmt.Printf("Transaction Hash: %s\n", tx.Hash)
	fmt.Printf("From: %s\n", tx.From)
	fmt.Printf("To: %s\n", tx.To)
	fmt.Printf("Amount: %s NRN\n", tx.Amount.String())
	fmt.Printf("Status: %s\n", tx.Status)

	return nil
}

func runEconomicsFaucet(cmd *cobra.Command, args []string) error {
	// Default amount
	amount := "1000"
	if len(args) > 0 {
		amount = args[0]
	}

	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get wallet
	if walletName == "" {
		return fmt.Errorf("wallet name required")
	}

	walletManager := core.NewWalletManager(cfg.KNIRV.Wallet.Directory, log)
	wallet, err := walletManager.GetWallet(walletName)
	if err != nil {
		return fmt.Errorf("failed to get wallet: %w", err)
	}

	// Create KNIRVROOT client and NRN token manager
	knirvRootClient := core.NewKNIRVRootClient(&cfg.KNIRV.Services.KNIRVRoot, log)
	nrnManager := core.NewNRNTokenManager(&cfg.KNIRV.Wallet, knirvRootClient, log)

	// Connect to KNIRVROOT
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := knirvRootClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to KNIRVROOT: %w", err)
	}
	defer knirvRootClient.Disconnect()

	// Request from faucet
	tx, err := nrnManager.RequestFromFaucet(ctx, wallet.Address, amount)
	if err != nil {
		return fmt.Errorf("faucet request failed: %w", err)
	}

	fmt.Printf("Faucet request successful!\n")
	fmt.Printf("Transaction Hash: %s\n", tx.Hash)
	fmt.Printf("Address: %s\n", tx.To)
	fmt.Printf("Amount: %s NRN\n", tx.Amount.String())
	fmt.Printf("Status: %s\n", tx.Status)

	return nil
}

func runEconomicsHistory(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create NRN token manager
	knirvRootClient := core.NewKNIRVRootClient(&cfg.KNIRV.Services.KNIRVRoot, log)
	nrnManager := core.NewNRNTokenManager(&cfg.KNIRV.Wallet, knirvRootClient, log)

	// Get transaction history
	history := nrnManager.GetTransactionHistory()

	if len(history) == 0 {
		fmt.Println("No transactions found")
		return nil
	}

	fmt.Printf("NRN Transaction History (%d transactions):\n\n", len(history))

	// Table header
	fmt.Printf("%-20s %-10s %-42s %-42s %-15s %-10s\n",
		"TIMESTAMP", "TYPE", "FROM", "TO", "AMOUNT", "STATUS")
	fmt.Println("----------------------------------------------------------------------------------------------------")

	// Table rows
	for _, tx := range history {
		timestamp := tx.Timestamp.Format("2006-01-02 15:04:05")
		from := truncateAddress(tx.From)
		to := truncateAddress(tx.To)

		fmt.Printf("%-20s %-10s %-42s %-42s %-15s %-10s\n",
			timestamp, tx.Type, from, to, tx.Amount.String(), tx.Status)
	}

	return nil
}

func runEconomicsStats(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create NRN token manager
	knirvRootClient := core.NewKNIRVRootClient(&cfg.KNIRV.Services.KNIRVRoot, log)
	nrnManager := core.NewNRNTokenManager(&cfg.KNIRV.Wallet, knirvRootClient, log)

	// Get statistics
	stats := nrnManager.GetNRNStats()

	fmt.Println("NRN Token Statistics:")
	fmt.Println("=====================")
	fmt.Printf("Current Balance: %s NRN\n", stats["current_balance"])
	fmt.Printf("Total Transactions: %v\n", stats["total_transactions"])
	fmt.Printf("Total Transferred: %s NRN\n", stats["total_transferred"])
	fmt.Printf("Total Received: %s NRN\n", stats["total_received"])
	fmt.Printf("Total Burned: %s NRN\n", stats["total_burned"])
	fmt.Printf("Auto-refill Enabled: %v\n", stats["auto_refill_enabled"])
	fmt.Printf("Minimum Balance: %s NRN\n", stats["min_balance"])

	return nil
}

func runEconomicsSkills(cmd *cobra.Command, args []string) error {
	fmt.Println("Skills management:")
	fmt.Println("- List skills: Coming soon")
	fmt.Println("- Register skill: Coming soon")
	fmt.Println("- Rent skill: Coming soon")

	return nil
}

func runEconomicsFees(cmd *cobra.Command, args []string) error {
	operation := args[0]

	fmt.Printf("Fee estimation for operation: %s\n", operation)
	fmt.Printf("Complexity: %s\n", complexity)
	fmt.Println("Estimated fee: 0 NRN (gasless)")

	return nil
}

// Helper function to truncate addresses for display
func truncateAddress(address string) string {
	if len(address) <= 42 {
		return address
	}
	return address[:6] + "..." + address[len(address)-4:]
}
