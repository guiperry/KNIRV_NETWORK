package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/config"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/core"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/incident"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/staging"
	"github.com/spf13/cobra"
)

var stageCmd = &cobra.Command{
	Use:   "stage [file]",
	Short: "Stage an incident artifact and optionally publish it to KNIRVGRAPH",
	Args:  cobra.ExactArgs(1),
	RunE:  runStage,
}

func init() {
	rootCmd.AddCommand(stageCmd)

	stageCmd.Flags().String("staging-dir", "", "Directory to copy staged artifacts into")
	stageCmd.Flags().Bool("publish", false, "Publish the staged artifact as an incident node to KNIRVGRAPH")
	stageCmd.Flags().String("agent", "manual-stage", "Agent name associated with the staged artifact")
	stageCmd.Flags().String("signature", "STAGED_ARTIFACT", "Incident signature to publish")
	stageCmd.Flags().String("trigger", "manual stage operation", "Trigger line for the incident")
	stageCmd.Flags().Int("exit-code", 0, "Exit code associated with the incident")
}

func runStage(cmd *cobra.Command, args []string) error {
	stagingDir, _ := cmd.Flags().GetString("staging-dir")
	result, err := staging.StageFile(args[0], stagingDir)
	if err != nil {
		return err
	}

	fmt.Printf("Staged %s\n", result.SourcePath)
	fmt.Printf("  copy: %s\n", result.StagedPath)
	fmt.Printf("  sha256: %s\n", result.Digest)
	fmt.Printf("  size: %d bytes\n", result.Size)

	publish, _ := cmd.Flags().GetBool("publish")
	if !publish {
		return nil
	}

	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return err
	}

	client := core.NewKNIRVGraphClient(&cfg.KNIRV.Services.KNIRVGraph, log)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		return err
	}
	defer client.Disconnect()

	agentName, _ := cmd.Flags().GetString("agent")
	signature, _ := cmd.Flags().GetString("signature")
	trigger, _ := cmd.Flags().GetString("trigger")
	exitCode, _ := cmd.Flags().GetInt("exit-code")

	inc := &incident.Incident{
		ID:             fmt.Sprintf("INC-STAGE-%d", time.Now().UTC().Unix()),
		AgentName:      agentName,
		AgentType:      "staged",
		Status:         "STAGED",
		ExitCode:       exitCode,
		Timestamp:      time.Now().UTC(),
		Environment:    "stage",
		ErrorSignature: signature,
		TriggerLine:    trigger,
		LogSnapshot:    []string{fmt.Sprintf("artifact=%s", filepath.Base(result.StagedPath))},
	}

	nodeID, err := client.CreateIncidentNode(ctx, inc, result.StagedPath, result.Digest)
	if err != nil {
		return err
	}

	fmt.Printf("Published incident node %s\n", nodeID)
	return nil
}
