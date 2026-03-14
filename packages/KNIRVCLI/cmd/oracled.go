package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVCLI/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	oracledRPCURL string
)

// oracledCmd represents the oracled command
var oracledCmd = &cobra.Command{
	Use:   "oracled",
	Short: "Interact with the KNIRV Oracle Daemon (knirv-oracled)",
	Long: `Provides commands to interact with the KNIRV Oracle Daemon (knirv-oracled),
including fetching its current status, querying blockchain data, and
submitting governance proposals.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Ensure global flags are processed first
		rootCmd.PersistentPreRun(cmd, args)

		// Set oracledRPCURL from flag or config
		// Directly get flag value first, then fallback to viper config
		flagValue, err := cmd.Flags().GetString("oracled-url")
		log.Debugf("Flag 'oracled-url' value: %s, error: %v", flagValue, err)

		if err == nil && flagValue != "" {
			oracledRPCURL = flagValue
		} else {
			oracledRPCURL = viper.GetString("knirv.services.knirvoracled.rpc_url")
		}

		log.Debugf("Final oracledRPCURL: %s", oracledRPCURL)

		if oracledRPCURL == "" {
			log.Fatal("Oracle Daemon RPC URL not provided. Use --oracled-url flag or configure 'knirv.services.knirvoracled.rpc_url'.")
		}
	},
}

// statusCmd represents the status subcommand for oracled
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get the current status of the knirv-oracled node",
	Long:  `Fetches and displays the current operational status of the KNIRV Oracle Daemon.`, 
	Run: func(cmd *cobra.Command, args []string) {
		client := core.NewOracledClient(oracledRPCURL)
		
	// Add a timeout for the client request
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		status, err := client.GetStatus(ctx)
		if err != nil {
			log.Fatalf("Failed to get knirv-oracled status: %v", err)
		}

		fmt.Printf("KNIRV Oracle Daemon Status:\n")
		fmt.Printf("  Chain ID: %s\n", status.ChainID)
		fmt.Printf("  Network ID: %s\n", status.NetworkID)
		fmt.Printf("  Token:\n")
		fmt.Printf("    Name: %s\n", status.Token.Name)
		fmt.Printf("    Symbol: %s\n", status.Token.Symbol)
		fmt.Printf("    Total Supply: %s\n", status.Token.TotalSupply)
		fmt.Printf("    Max Supply: %s\n", status.Token.MaxSupply)
		fmt.Printf("    Contract Address: %s\n", status.Token.ContractAddress)
		fmt.Printf("  Consensus:\n")
		fmt.Printf("    Latest Block Height: %d\n", status.Consensus.LatestBlockHeight)
		fmt.Printf("    Validator Count: %d\n", status.Consensus.ValidatorCount)
		fmt.Printf("  IBC Enabled: %t\n", status.IBC.Enabled)
		// More detailed output for Governance, Economics, P2P could be added here if needed
	},
}

func init() {
	rootCmd.AddCommand(oracledCmd) // Add oracled command to root
	oracledCmd.AddCommand(statusCmd) // Add status subcommand to oracled

	// Local flag for oracledCmd to override default RPC URL
	oracledCmd.PersistentFlags().StringVar(&oracledRPCURL, "oracled-url", "", "URL of the knirv-oracled RPC endpoint")
	viper.BindPFlag("oracled.rpc_url", oracledCmd.PersistentFlags().Lookup("oracled-url"))
}
