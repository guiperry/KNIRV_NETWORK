package cmd

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/config"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/core"
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
		Short: "Request NRV tokens from testnet faucet",
		Long:  `Request NRV tokens from the testnet faucet for testing purposes.`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  runEconomicsFaucet,
	}

	economicsFaucetStatusCmd = &cobra.Command{
		Use:   "faucet-status",
		Short: "Get testnet faucet status",
		Long:  `Display the current status of the testnet faucet including balance, limits, and availability.`,
		RunE:  runEconomicsFaucetStatus,
	}

	economicsFaucetHistoryCmd = &cobra.Command{
		Use:   "faucet-history [address] [limit]",
		Short: "Get faucet request history",
		Long:  `Display the faucet request history for a specific address.`,
		Args:  cobra.MaximumNArgs(2),
		RunE:  runEconomicsFaucetHistory,
	}

	economicsFaucetHealthCmd = &cobra.Command{
		Use:   "faucet-health",
		Short: "Check testnet faucet health",
		Long:  `Check the health status of the testnet faucet service.`,
		RunE:  runEconomicsFaucetHealth,
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
	faucetReason     string
	maxRetries       int
	historyLimit     int
)

func init() {
	rootCmd.AddCommand(economicsCmd)

	// Add subcommands
	economicsCmd.AddCommand(economicsBalanceCmd)
	economicsCmd.AddCommand(economicsTransferCmd)
	economicsCmd.AddCommand(economicsFaucetCmd)
	economicsCmd.AddCommand(economicsFaucetStatusCmd)
	economicsCmd.AddCommand(economicsFaucetHistoryCmd)
	economicsCmd.AddCommand(economicsFaucetHealthCmd)
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
	economicsFaucetCmd.Flags().StringVar(&faucetReason, "reason", "", "Reason for faucet request")
	economicsFaucetCmd.Flags().IntVar(&maxRetries, "max-retries", 3, "Maximum retry attempts")

	// Faucet history command flags
	economicsFaucetHistoryCmd.Flags().IntVar(&historyLimit, "limit", 10, "Maximum number of history entries to show")

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

	// Create KNIRVORACLE client
	knirvRootClient := core.NewKNIRVRootClient(&cfg.KNIRV.Services.KNIRVRoot, log)

	// Connect to KNIRVORACLE
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := knirvRootClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to KNIRVORACLE: %w", err)
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

	// Create KNIRVORACLE client and NRN token manager
	knirvRootClient := core.NewKNIRVRootClient(&cfg.KNIRV.Services.KNIRVRoot, log)
	nrnManager := core.NewNRNTokenManager(&cfg.KNIRV.Wallet, knirvRootClient, log)

	// Connect to KNIRVORACLE
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := knirvRootClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to KNIRVORACLE: %w", err)
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

	// Create KNIRVORACLE client and NRN token manager
	knirvRootClient := core.NewKNIRVRootClient(&cfg.KNIRV.Services.KNIRVRoot, log)
	nrnManager := core.NewNRNTokenManager(&cfg.KNIRV.Wallet, knirvRootClient, log)

	// Connect to KNIRVORACLE
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := knirvRootClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to KNIRVORACLE: %w", err)
	}
	defer knirvRootClient.Disconnect()

	// Use enhanced faucet request with retry logic
	reason := faucetReason
	if reason == "" {
		reason = "CLI request"
	}

	fmt.Printf("Requesting %s NRV tokens from testnet faucet...\n", amount)
	fmt.Printf("Address: %s\n", wallet.Address)
	if reason != "CLI request" {
		fmt.Printf("Reason: %s\n", reason)
	}
	fmt.Printf("Max retries: %d\n", maxRetries)
	fmt.Println()

	tx, err := nrnManager.RequestFromFaucetWithRetry(ctx, wallet.Address, amount, reason, maxRetries)
	if err != nil {
		return fmt.Errorf("faucet request failed: %w", err)
	}

	fmt.Printf("✅ Testnet faucet request successful!\n")
	fmt.Printf("Transaction Hash: %s\n", tx.Hash)
	fmt.Printf("Request ID: %s\n", tx.ID)
	fmt.Printf("Address: %s\n", tx.To)
	fmt.Printf("Amount: %s NRV\n", tx.Amount.String())
	fmt.Printf("Status: %s\n", tx.Status)
	fmt.Printf("Timestamp: %s\n", tx.Timestamp.Format(time.RFC3339))

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

// Enhanced Testnet Faucet Commands

func runEconomicsFaucetStatus(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create KNIRVORACLE client and NRN token manager
	knirvRootClient := core.NewKNIRVRootClient(&cfg.KNIRV.Services.KNIRVRoot, log)
	nrnManager := core.NewNRNTokenManager(&cfg.KNIRV.Wallet, knirvRootClient, log)

	// Connect to KNIRVORACLE
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := knirvRootClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to KNIRVORACLE: %w", err)
	}
	defer knirvRootClient.Disconnect()

	// Get faucet status
	status, err := nrnManager.GetFaucetStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get faucet status: %w", err)
	}

	fmt.Printf("🚰 Testnet Faucet Status\n")
	fmt.Printf("========================\n")
	fmt.Printf("Enabled: %v\n", status.FaucetEnabled)
	fmt.Printf("Current Balance: %d NRV\n", status.CurrentBalance)
	fmt.Printf("Daily Limit: %d NRV\n", status.DailyLimit)
	fmt.Printf("Remaining Today: %d NRV\n", status.RemainingToday)
	fmt.Printf("Queue Size: %d requests\n", status.CurrentQueueSize)
	fmt.Printf("Success Rate Today: %.1f%%\n", status.SuccessRateToday*100)

	if status.LastFunding != "" {
		fmt.Printf("Last Funding: %s\n", status.LastFunding)
	}
	if status.NextFundingEst != "" {
		fmt.Printf("Next Funding Est: %s\n", status.NextFundingEst)
	}

	fmt.Printf("\nRate Limits:\n")
	for key, value := range status.RateLimits {
		fmt.Printf("  %s: %v\n", key, value)
	}

	fmt.Printf("\nSupported Amounts:\n")
	for key, value := range status.SupportedAmounts {
		fmt.Printf("  %s: %v\n", key, value)
	}

	return nil
}

func runEconomicsFaucetHistory(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get address
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

	// Parse limit
	limit := historyLimit
	if len(args) > 1 {
		if parsedLimit, err := fmt.Sscanf(args[1], "%d", &limit); err != nil || parsedLimit != 1 {
			return fmt.Errorf("invalid limit: %s", args[1])
		}
	}

	// Create KNIRVORACLE client and NRN token manager
	knirvRootClient := core.NewKNIRVRootClient(&cfg.KNIRV.Services.KNIRVRoot, log)
	nrnManager := core.NewNRNTokenManager(&cfg.KNIRV.Wallet, knirvRootClient, log)

	// Connect to KNIRVORACLE
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := knirvRootClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to KNIRVORACLE: %w", err)
	}
	defer knirvRootClient.Disconnect()

	// Get faucet history
	history, err := nrnManager.GetFaucetHistory(ctx, address, limit)
	if err != nil {
		return fmt.Errorf("failed to get faucet history: %w", err)
	}

	fmt.Printf("📜 Faucet History for %s\n", truncateAddress(address))
	fmt.Printf("=====================================\n")
	fmt.Printf("Total Requests: %d\n", history.TotalRequests)
	fmt.Printf("Total Amount: %d NRV\n", history.TotalAmount)
	fmt.Printf("Showing last %d entries:\n\n", len(history.History))

	if len(history.History) == 0 {
		fmt.Println("No faucet requests found.")
		return nil
	}

	for i, entry := range history.History {
		fmt.Printf("%d. Request ID: %s\n", i+1, entry.RequestID)
		fmt.Printf("   Amount: %d NRV\n", entry.Amount)
		fmt.Printf("   Status: %s\n", entry.Status)
		fmt.Printf("   Timestamp: %s\n", entry.Timestamp)
		if entry.TxHash != "" {
			fmt.Printf("   TX Hash: %s\n", entry.TxHash)
		}
		if entry.Error != "" {
			fmt.Printf("   Error: %s\n", entry.Error)
		}
		if entry.Reason != "" {
			fmt.Printf("   Reason: %s\n", entry.Reason)
		}
		fmt.Println()
	}

	return nil
}

func runEconomicsFaucetHealth(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create KNIRVORACLE client and NRN token manager
	knirvRootClient := core.NewKNIRVRootClient(&cfg.KNIRV.Services.KNIRVRoot, log)
	nrnManager := core.NewNRNTokenManager(&cfg.KNIRV.Wallet, knirvRootClient, log)

	// Connect to KNIRVORACLE
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := knirvRootClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to KNIRVORACLE: %w", err)
	}
	defer knirvRootClient.Disconnect()

	// Check faucet health
	health, err := nrnManager.CheckFaucetHealth(ctx)
	if err != nil {
		return fmt.Errorf("failed to check faucet health: %w", err)
	}

	fmt.Printf("🏥 Testnet Faucet Health Check\n")
	fmt.Printf("==============================\n")

	for key, value := range health {
		fmt.Printf("%s: %v\n", key, value)
	}

	// Determine overall health status
	status, ok := health["status"].(string)
	if ok && status == "healthy" {
		fmt.Printf("\n✅ Faucet is healthy and operational\n")
	} else {
		fmt.Printf("\n⚠️  Faucet may have issues\n")
	}

	return nil
}

// Helper function to truncate addresses for display
func truncateAddress(address string) string {
	if len(address) <= 42 {
		return address
	}
	return address[:6] + "..." + address[len(address)-4:]
}
