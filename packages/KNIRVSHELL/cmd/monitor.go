package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/config"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/core"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/registry"
	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Show KNIRVSHELL registry and network health information",
	RunE:  runMonitor,
}

func init() {
	rootCmd.AddCommand(monitorCmd)

	monitorCmd.Flags().StringVar(&registryPath, "registry-file", "", "Path to the agent registry file")
}

func runMonitor(cmd *cobra.Command, args []string) error {
	store := registry.NewStore(registryPath)
	agents, err := store.List()
	if err != nil {
		return err
	}

	fmt.Printf("Agent Registry: %s\n", store.Path())
	if len(agents) == 0 {
		fmt.Println("  No registered agents")
	} else {
		for _, agent := range agents {
			fmt.Printf("  - %s (%s)\n", agent.Name, agent.Type)
		}
	}

	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return err
	}

	registrySvc := coreRegistrySnapshot(cfg)
	fmt.Printf("\nConfigured Services: %d\n", len(registrySvc))
	for name, url := range registrySvc {
		status := "unverified"
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := quickHealthCheck(ctx, url); err == nil {
			status = "reachable"
		}
		cancel()
		fmt.Printf("  - %s: %s (%s)\n", name, url, status)
	}

	return nil
}

func coreRegistrySnapshot(cfg *config.Config) map[string]string {
	return map[string]string{
		"knirvoracle":  cfg.KNIRV.Services.KNIRVRoot.URL,
		"knirvgateway": cfg.KNIRV.Services.KNIRVGateway.URL,
		"knirvserver":  cfg.KNIRV.Services.KNIRVNexus.URL,
		"knirvgraph":   cfg.KNIRV.Services.KNIRVGraph.URL,
	}
}

func quickHealthCheck(ctx context.Context, baseURL string) error {
	client := core.NewAPIClient(baseURL, core.WithTimeout(5*time.Second))
	return client.Get(ctx, "/health", nil)
}
