package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(dveCmd)
	dveCmd.AddCommand(browserDVECmd)

	browserDVECmd.AddCommand(browserRegisterCmd)
	browserDVECmd.AddCommand(browserListCmd)
	browserDVECmd.AddCommand(browserStatusCmd)

	browserRegisterCmd.Flags().String("wallet", "", "Wallet address for the browser DVE")
	browserRegisterCmd.Flags().String("server", "http://localhost:8084", "KNIRVSERVER API URL")
	browserRegisterCmd.Flags().String("extension-id", "", "Browser extension ID")
	browserRegisterCmd.Flags().String("capabilities", "policy-check,signature-verify", "Comma-separated capabilities")
	browserRegisterCmd.MarkFlagRequired("wallet")

	browserListCmd.Flags().String("server", "http://localhost:8084", "KNIRVSERVER API URL")

	browserStatusCmd.Flags().String("dve-id", "", "DVE node ID")
	browserStatusCmd.Flags().String("server", "http://localhost:8084", "KNIRVSERVER API URL")
	browserStatusCmd.MarkFlagRequired("dve-id")
}

var dveCmd = &cobra.Command{
	Use:   "dve",
	Short: "Manage DVE nodes and operations",
	Long:  `Manage DVE (Deterministic Validation Environment) nodes including browser DVE registration, listing, and status checks.`,
}

var browserDVECmd = &cobra.Command{
	Use:   "browser",
	Short: "Manage browser DVE nodes",
	Long:  `Manage browser-extension-based DVE nodes. These are lightweight DVE nodes running in browser extensions.`,
}

var browserRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a browser DVE node",
	Long: `Register a browser DVE node with KNIRVSERVER.
	
Example:
  knirv dve browser register --wallet g1abc123... --server https://server:8084 --extension-id abcdef...`,
	RunE: func(cmd *cobra.Command, args []string) error {
		wallet, _ := cmd.Flags().GetString("wallet")
		server, _ := cmd.Flags().GetString("server")
		extensionID, _ := cmd.Flags().GetString("extension-id")
		capStr, _ := cmd.Flags().GetString("capabilities")

		capabilities := strings.Split(capStr, ",")

		// Build the registration payload
		payload := map[string]interface{}{
			"name":            fmt.Sprintf("Browser DVE - %s", wallet[:min(8, len(wallet))]),
			"tee_type":        "browser-extension",
			"wallet_address":  wallet,
			"extension_id":    extensionID,
			"capabilities":    capabilities,
			"is_remote":       true,
			"stake_amount":    5000,
		}

		body, _ := json.Marshal(payload)

		resp, err := http.Post(
			fmt.Sprintf("%s/api/v1/dve/nodes", strings.TrimRight(server, "/")),
			"application/json",
			strings.NewReader(string(body)),
		)
		if err != nil {
			return fmt.Errorf("failed to register browser DVE: %w", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		fmt.Printf("Response (%d): %s\n", resp.StatusCode, string(respBody))
		return nil
	},
}

var browserListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all browser DVE nodes",
	RunE: func(cmd *cobra.Command, args []string) error {
		server, _ := cmd.Flags().GetString("server")

		resp, err := http.Get(
			fmt.Sprintf("%s/api/v1/dve/nodes?tee_type=browser-extension", strings.TrimRight(server, "/")),
		)
		if err != nil {
			return fmt.Errorf("failed to list browser DVEs: %w", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		var result map[string]interface{}
		json.Unmarshal(body, &result)

		d, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(d))
		return nil
	},
}

var browserStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get status of a browser DVE node",
	RunE: func(cmd *cobra.Command, args []string) error {
		dveID, _ := cmd.Flags().GetString("dve-id")
		server, _ := cmd.Flags().GetString("server")

		resp, err := http.Get(
			fmt.Sprintf("%s/api/v1/dve/nodes/%s", strings.TrimRight(server, "/"), dveID),
		)
		if err != nil {
			return fmt.Errorf("failed to get DVE status: %w", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		var result map[string]interface{}
		json.Unmarshal(body, &result)

		d, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(d))
		return nil
	},
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
