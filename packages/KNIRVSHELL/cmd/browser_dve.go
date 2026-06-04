package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/core"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(dveCmd)
	dveCmd.AddCommand(browserDVECmd)

	browserDVECmd.AddCommand(browserRegisterCmd)
	browserDVECmd.AddCommand(browserListCmd)
	browserDVECmd.AddCommand(browserStatusCmd)
	browserDVECmd.AddCommand(browserConnectCmd)

	browserRegisterCmd.Flags().String("wallet", "", "Wallet address for the browser DVE")
	browserRegisterCmd.Flags().String("server", "http://localhost:8084", "KNIRVSERVER API URL")
	browserRegisterCmd.Flags().String("extension-id", "", "Browser extension ID")
	browserRegisterCmd.Flags().String("capabilities", "policy-check,signature-verify", "Comma-separated capabilities")
	browserRegisterCmd.Flags().String("badge-nft-ids", "", "Comma-separated badge NFT IDs")
	browserRegisterCmd.Flags().String("browser-version", "", "Browser or client version string")
	browserRegisterCmd.MarkFlagRequired("wallet")

	browserListCmd.Flags().String("server", "http://localhost:8084", "KNIRVSERVER API URL")

	browserStatusCmd.Flags().String("dve-id", "", "DVE node ID")
	browserStatusCmd.Flags().String("server", "http://localhost:8084", "KNIRVSERVER API URL")
	browserStatusCmd.MarkFlagRequired("dve-id")

	browserConnectCmd.Flags().String("wallet", "", "Wallet address for the browser DVE")
	browserConnectCmd.Flags().String("server", "http://localhost:8084", "KNIRVSERVER API URL")
	browserConnectCmd.Flags().String("auth-token", "", "JWT or bearer token for websocket authentication")
	browserConnectCmd.Flags().String("extension-id", "knirvshell", "Browser extension or CLI identity source")
	browserConnectCmd.Flags().String("capabilities", "policy-check,signature-verify", "Comma-separated capabilities")
	browserConnectCmd.Flags().String("badge-nft-ids", "", "Comma-separated badge NFT IDs")
	browserConnectCmd.Flags().String("browser-version", "", "Browser or client version string")
	browserConnectCmd.Flags().Duration("heartbeat-interval", 45*time.Second, "Heartbeat interval for the websocket connection")
	browserConnectCmd.MarkFlagRequired("auth-token")
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
		server, _ := cmd.Flags().GetString("server")
		wallet, _ := cmd.Flags().GetString("wallet")
		extensionID, _ := cmd.Flags().GetString("extension-id")
		capStr, _ := cmd.Flags().GetString("capabilities")
		badgeStr, _ := cmd.Flags().GetString("badge-nft-ids")
		browserVersion, _ := cmd.Flags().GetString("browser-version")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resolvedWallet, err := resolveWalletAddress(ctx, wallet)
		if err != nil {
			return err
		}

		identity := core.DeriveBrowserDVEIdentity(resolvedWallet, extensionID, browserVersion)
		capabilities := splitCSVFlag(capStr)
		badgeNFTIDs := splitCSVFlag(badgeStr)

		// Build the registration payload
		payload := map[string]interface{}{
			"name":            fmt.Sprintf("Browser DVE - %s", resolvedWallet[:min(8, len(resolvedWallet))]),
			"tee_type":        "browser-extension",
			"wallet_address":  resolvedWallet,
			"extension_id":    identity.ExtensionID,
			"browser_version": identity.BrowserVersion,
			"capabilities":    capabilities,
			"badge_nft_ids":   badgeNFTIDs,
			"node_id":         identity.NodeID,
			"dve_uri":         identity.DVEURI,
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

var browserConnectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to the browser DVE websocket",
	Long: `Connect to the browser DVE websocket, register the node identity,
and stream task/heartbeat messages until interrupted.

Example:
  knirv dve browser connect --wallet gno1abc123... --auth-token <jwt> --server http://localhost:8084`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server, _ := cmd.Flags().GetString("server")
		authToken, _ := cmd.Flags().GetString("auth-token")
		wallet, _ := cmd.Flags().GetString("wallet")
		extensionID, _ := cmd.Flags().GetString("extension-id")
		capStr, _ := cmd.Flags().GetString("capabilities")
		badgeStr, _ := cmd.Flags().GetString("badge-nft-ids")
		browserVersion, _ := cmd.Flags().GetString("browser-version")
		heartbeatInterval, _ := cmd.Flags().GetDuration("heartbeat-interval")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resolvedWallet, err := resolveWalletAddress(ctx, wallet)
		if err != nil {
			return err
		}

		identity := core.DeriveBrowserDVEIdentity(resolvedWallet, extensionID, browserVersion)
		capabilities := splitCSVFlag(capStr)
		badgeNFTIDs := splitCSVFlag(badgeStr)

		clientLogger := logrus.New()
		clientLogger.SetLevel(logrus.InfoLevel)

		client := core.NewBrowserDVEClient(
			server,
			authToken,
			identity,
			core.WithBrowserDVELogger(clientLogger),
			core.WithBrowserDVECapabilities(capabilities),
			core.WithBrowserDVEBadgeNFTIDs(badgeNFTIDs),
			core.WithBrowserDVEHeartbeatInterval(heartbeatInterval),
		)

		client.On("task_assigned", func(message core.BrowserDVEMessage) {
			fmt.Printf("[task_assigned] %s\n", string(message.Payload))
		})
		client.On("policy_sync", func(message core.BrowserDVEMessage) {
			fmt.Printf("[policy_sync] %s\n", string(message.Payload))
		})
		client.On("badge_refresh", func(message core.BrowserDVEMessage) {
			fmt.Printf("[badge_refresh] %s\n", string(message.Payload))
		})
		client.On("heartbeat_ack", func(message core.BrowserDVEMessage) {
			fmt.Printf("[heartbeat_ack] %s\n", string(message.Payload))
		})
		client.On("ws_register", func(message core.BrowserDVEMessage) {
			fmt.Printf("[ws_register] %s\n", string(message.Payload))
		})
		client.On("*", func(message core.BrowserDVEMessage) {
			if message.Type == "task_assigned" || message.Type == "policy_sync" || message.Type == "badge_refresh" || message.Type == "heartbeat_ack" || message.Type == "ws_register" {
				return
			}
			fmt.Printf("[%s] %s\n", message.Type, string(message.Payload))
		})

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		if err := client.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect browser DVE websocket: %w", err)
		}

		fmt.Printf(
			"Connected browser DVE node %s (%s) at %s\n",
			identity.NodeID,
			identity.DVEURI,
			server,
		)
		fmt.Println("Press Ctrl+C to disconnect.")

		<-ctx.Done()

		if err := client.Disconnect(); err != nil {
			return fmt.Errorf("failed to disconnect browser DVE websocket: %w", err)
		}

		fmt.Println("Disconnected browser DVE websocket.")
		return nil
	},
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func splitCSVFlag(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
