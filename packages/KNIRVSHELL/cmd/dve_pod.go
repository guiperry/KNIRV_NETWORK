package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/dvepod"
	"github.com/spf13/cobra"
)

func init() {
	dveCmd.AddCommand(dvePodCmd)
	dvePodCmd.AddCommand(dvePodNewCmd)
	dvePodCmd.AddCommand(dvePodRunCmd)
	dvePodCmd.AddCommand(dvePodListCmd)
	dvePodCmd.AddCommand(dvePodDockCmd)
	dvePodCmd.AddCommand(dvePodBundleCmd)
	dvePodCmd.AddCommand(dvePodStatusCmd)

	dvePodDockCmd.Flags().String("server", "http://localhost:8084", "KNIRVSERVER URL")
	dvePodDockCmd.Flags().String("auth-token", "", "JWT bearer token")
	dvePodBundleCmd.Flags().String("output", "./dvepod.html", "output HTML path")
	dvePodBundleCmd.Flags().String("dock-url", "", "pre-seed dock URL in bundle")
	dvePodRunCmd.Flags().String("pod-id", "", "existing pod ID to resume")
	dvePodListCmd.Flags().String("data-dir", "", "custom data directory")
	dvePodStatusCmd.Flags().String("data-dir", "", "custom data directory")
}

var dvePodCmd = &cobra.Command{
	Use:   "pod",
	Short: "Manage portable DVE Pods",
	Long:  `Create, run, and manage portable DVE Pods — self-contained WASM environments with embedded KNIRVAGENT, TEE simulation, and BusyBox tools.`,
}

var dvePodNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create and launch a new DVE Pod",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := dvepod.NewManager()
		if err != nil {
			return fmt.Errorf("failed to initialize DVE Pod manager: %w", err)
		}
		fmt.Println("knirv: launching new DVE Pod...")
		return mgr.New(cmd.Context())
	},
}

var dvePodRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Resume an existing DVE Pod",
	RunE: func(cmd *cobra.Command, args []string) error {
		podID, _ := cmd.Flags().GetString("pod-id")
		mgr, err := dvepod.NewManager()
		if err != nil {
			return fmt.Errorf("failed to initialize DVE Pod manager: %w", err)
		}
		return mgr.Run(cmd.Context(), podID)
	},
}

var dvePodListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all local DVE Pods",
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir, _ := cmd.Flags().GetString("data-dir")
		mgr, err := dvepod.NewManager()
		if err != nil {
			return fmt.Errorf("failed to initialize DVE Pod manager: %w", err)
		}
		if dataDir != "" {
			mgr.SetDataDir(dataDir)
		}
		return mgr.ListPods()
	},
}

var dvePodBundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Export a DVE Pod as a self-contained HTML file",
	RunE: func(cmd *cobra.Command, args []string) error {
		out, _ := cmd.Flags().GetString("output")
		dockURL, _ := cmd.Flags().GetString("dock-url")

		out = filepath.Clean(out)
		parentDir := filepath.Dir(out)
		if parentDir != "." {
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}

		mgr, err := dvepod.NewManager()
		if err != nil {
			return fmt.Errorf("failed to initialize DVE Pod manager: %w", err)
		}
		return mgr.Bundle(out, dockURL)
	},
}

var dvePodDockCmd = &cobra.Command{
	Use:   "dock",
	Short: "Dock a running DVE Pod to a KNIRVSERVER",
	RunE: func(cmd *cobra.Command, args []string) error {
		server, _ := cmd.Flags().GetString("server")
		token, _ := cmd.Flags().GetString("auth-token")
		mgr, err := dvepod.NewManager()
		if err != nil {
			return fmt.Errorf("failed to initialize DVE Pod manager: %w", err)
		}
		return mgr.Dock(cmd.Context(), server, token)
	},
}

var dvePodStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show DVE Pod status information",
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir, _ := cmd.Flags().GetString("data-dir")
		mgr, err := dvepod.NewManager()
		if err != nil {
			return fmt.Errorf("failed to initialize DVE Pod manager: %w", err)
		}
		if dataDir != "" {
			mgr.SetDataDir(dataDir)
		}
		return mgr.Status()
	},
}
