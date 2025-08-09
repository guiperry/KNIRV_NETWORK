package cmd

import (
	"github.com/guiperry/KNIRVCHAIN-CLI/cmd/mcp"
	"github.com/spf13/cobra"
)

// mcpCmd represents the mcp command
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage MCP capabilities and servers",
	Long: `Manage MCP (Multi-Capability Protocol) capabilities, servers, and procedures.
This command provides access to all MCP-related functionality.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)

	// Add MCP subcommands
	mcpCmd.AddCommand(mcp.CapabilityCmd)
	mcpCmd.AddCommand(mcp.ServerCmd)
	mcpCmd.AddCommand(mcp.ProcedureCmd)
	mcpCmd.AddCommand(mcp.GenerateCmd)
	mcpCmd.AddCommand(mcp.InvokeCmd)
	mcpCmd.AddCommand(mcp.NRVCmd) // Enhanced NRV integration
}
