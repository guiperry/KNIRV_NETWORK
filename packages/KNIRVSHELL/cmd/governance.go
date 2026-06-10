package cmd

import (
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/cmd/governance"
	"github.com/spf13/cobra"
)

var governanceCmd = &cobra.Command{
	Use:   "governance",
	Short: "Manage governance, DID, identity, policy, reliability, and compliance",
	Long:  `Governance commands for identity management, policy enforcement, reliability controls, MCP hardening, and compliance.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(governanceCmd)

	governanceCmd.AddCommand(governance.DIDCmd)
	governanceCmd.AddCommand(governance.IdentityCmd)
	governanceCmd.AddCommand(governance.PolicyCmd)
	governanceCmd.AddCommand(governance.ReliabilityCmd)
	governanceCmd.AddCommand(governance.MCPCmd)
	governanceCmd.AddCommand(governance.RevokeCmd)
	governanceCmd.AddCommand(governance.ComplianceCmd)
}
