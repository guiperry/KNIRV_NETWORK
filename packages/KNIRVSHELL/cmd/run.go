package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/incident"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/registry"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/runner"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/watchtower"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [agent]",
	Short: "Run a registered KNIRVSHELL agent under watchtower supervision",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgent,
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVar(&registryPath, "registry-file", "", "Path to the agent registry file")
	runCmd.Flags().Duration("timeout", 0, "Optional maximum runtime before the agent is stopped")
}

func runAgent(cmd *cobra.Command, args []string) error {
	store := registry.NewStore(registryPath)
	record, err := store.Get(args[0])
	if err != nil {
		return err
	}

	runTimeout, _ := cmd.Flags().GetDuration("timeout")
	ctx := context.Background()
	if runTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, runTimeout)
		defer cancel()
	}

	executor, err := runner.ForType(record.Type)
	if err != nil {
		return err
	}

	proc, err := executor.Start(ctx, record.RunnerConfig())
	if err != nil {
		return err
	}

	uiCh := make(chan watchtower.UILogLine, 128)
	incCh := make(chan *incident.Incident, 16)
	tower := watchtower.New(uiCh, incCh)
	resultCh := tower.Watch(ctx, proc)

	go func() {
		for line := range uiCh {
			stream := "stdout"
			if line.IsError {
				stream = "stderr"
			}
			fmt.Printf("[%s][%s] %s\n", line.AgentID[:8], stream, line.Line)
		}
	}()

	go func() {
		for inc := range incCh {
			fmt.Printf("[incident] %s signature=%s exit_code=%d\n", inc.ID, inc.ErrorSignature, inc.ExitCode)
		}
	}()

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case sig := <-sigCh:
		fmt.Printf("Received %s, stopping %s\n", sig, record.Name)
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := proc.StopFn(stopCtx); err != nil {
			return fmt.Errorf("stop agent: %w", err)
		}
		result := <-resultCh
		if result.Err != nil {
			return result.Err
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("agent exited with code %d", result.ExitCode)
		}
		return nil
	case result := <-resultCh:
		if result.Err != nil {
			return result.Err
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("agent exited with code %d", result.ExitCode)
		}
		return nil
	}
}
