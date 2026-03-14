package mcp

import (
	"github.com/spf13/cobra"
)

// InvokeCmd represents the invoke command
var InvokeCmd = &cobra.Command{
	Use:   "invoke",
	Short: "Invoke a capability",
	Long: `Invoke a registered capability on the KNIRVCHAIN blockchain.
This command sends a transaction to invoke a capability with specified parameters.`,
	Run: func(cmd *cobra.Command, args []string) {
		// This will be implemented in a future phase
		log.Info("Capability invocation will be implemented in a future phase")
	},
}

func init() {
	// invoke flags
	InvokeCmd.Flags().String("id", "", "Capability ID to invoke")
	InvokeCmd.Flags().String("params", "", "Parameters in JSON format")
	InvokeCmd.Flags().String("wallet", "", "Wallet to use for invocation")
	InvokeCmd.Flags().Bool("async", false, "Invoke asynchronously")
	InvokeCmd.Flags().Bool("wait", false, "Wait for transaction confirmation")
	InvokeCmd.Flags().Int("timeout", 60, "Timeout in seconds for waiting")
}
