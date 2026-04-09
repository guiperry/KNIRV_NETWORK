package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/config"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/core"
	"github.com/spf13/cobra"
)

// NRVCmd represents the NRV (Network Resolution Vector) command
var NRVCmd = &cobra.Command{
	Use:   "nrv",
	Short: "Network Resolution Vector operations with KNIRV Graph integration",
	Long: `NRV command provides operations for submitting ErrorNodes and SkillNodes to the KNIRV Graph,
querying the NRV system for error resolution, and managing skill-based problem solving.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var (
	nrvSubmitErrorCmd = &cobra.Command{
		Use:   "submit-error <error-type> <description>",
		Short: "Submit an ErrorNode to the NRV system",
		Long:  `Submit an ErrorNode to the KNIRV Graph for resolution tracking and skill matching.`,
		Args:  cobra.ExactArgs(2),
		RunE:  runNRVSubmitError,
	}

	nrvSubmitSkillCmd = &cobra.Command{
		Use:   "submit-skill <skill-type> <capabilities>",
		Short: "Submit a SkillNode to the NRV system",
		Long:  `Submit a SkillNode to the KNIRV Graph for error resolution and capability registration.`,
		Args:  cobra.ExactArgs(2),
		RunE:  runNRVSubmitSkill,
	}

	nrvQuerySkillsCmd = &cobra.Command{
		Use:   "query-skills [error-type]",
		Short: "Query skills for error resolution",
		Long:  `Query the NRV system for skills that can resolve specific errors or list all available skills.`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  runNRVQuerySkills,
	}

	nrvQueryErrorsCmd = &cobra.Command{
		Use:   "query-errors [status]",
		Short: "Query errors in the NRV system",
		Long:  `Query the NRV system for errors by status or list all errors.`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  runNRVQueryErrors,
	}

	nrvStatsCmd = &cobra.Command{
		Use:   "stats",
		Short: "Show NRV system statistics",
		Long:  `Display statistics about the NRV system including error resolution rates and skill performance.`,
		RunE:  runNRVStats,
	}

	// Flags
	severity     int
	errorContext string
	capabilities string
	autoResolve  bool
	networkWide  bool
)

func init() {
	// Add subcommands
	NRVCmd.AddCommand(nrvSubmitErrorCmd)
	NRVCmd.AddCommand(nrvSubmitSkillCmd)
	NRVCmd.AddCommand(nrvQuerySkillsCmd)
	NRVCmd.AddCommand(nrvQueryErrorsCmd)
	NRVCmd.AddCommand(nrvStatsCmd)

	// Submit error flags
	nrvSubmitErrorCmd.Flags().IntVar(&severity, "severity", 5, "Error severity (1-10)")
	nrvSubmitErrorCmd.Flags().StringVar(&errorContext, "context", "", "Error context as JSON")
	nrvSubmitErrorCmd.Flags().BoolVar(&autoResolve, "auto-resolution", false, "Enable automatic resolution")

	// Submit skill flags
	nrvSubmitSkillCmd.Flags().StringVar(&capabilities, "capabilities", "", "Skill capabilities as comma-separated list")

	// Query skills flags
	nrvQuerySkillsCmd.Flags().StringVar(&capabilities, "capability-type", "", "Filter by capability type")
	nrvQuerySkillsCmd.Flags().BoolVar(&networkWide, "network-wide", false, "Query across entire network")
}

func runNRVSubmitError(cmd *cobra.Command, args []string) error {
	errorType := args[0]
	description := args[1]

	// Load configuration
	cfg, err := config.LoadConfig("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create KNIRVGRAPH client
	graphClient := core.NewKNIRVGraphClient(&cfg.KNIRV.Services.KNIRVGraph, log)

	// Connect to KNIRVGRAPH
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := graphClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to KNIRVGRAPH: %w", err)
	}
	defer graphClient.Disconnect()

	// Create ErrorNode
	errorNode := &core.ErrorNode{
		ID:               generateNodeID("error"),
		ErrorType:        errorType,
		Description:      description,
		Context:          parseContext(errorContext),
		Severity:         severity,
		Timestamp:        time.Now().Format(time.RFC3339),
		ResolutionStatus: "pending",
	}

	// Submit ErrorNode
	if err := graphClient.SubmitErrorNode(ctx, errorNode); err != nil {
		return fmt.Errorf("failed to submit error node: %w", err)
	}

	fmt.Printf("ErrorNode submitted successfully!\n")
	fmt.Printf("ID: %s\n", errorNode.ID)
	fmt.Printf("Type: %s\n", errorNode.ErrorType)
	fmt.Printf("Description: %s\n", errorNode.Description)
	fmt.Printf("Severity: %d\n", errorNode.Severity)

	// Auto-resolution if requested
	if autoResolve {
		fmt.Println("\nSearching for resolution skills...")
		skills, err := graphClient.QuerySkillsForError(ctx, errorType, errorNode.Context)
		if err != nil {
			log.Errorf("Failed to query skills for auto-resolution: %v", err)
		} else if len(skills) > 0 {
			fmt.Printf("Found %d potential resolution skills:\n", len(skills))
			for i, skill := range skills {
				if i >= 3 { // Limit to top 3
					break
				}
				fmt.Printf("  - %s: %s\n", skill.ID, skill.SkillType)
			}
		} else {
			fmt.Println("No resolution skills found")
		}
	}

	return nil
}

func runNRVSubmitSkill(cmd *cobra.Command, args []string) error {
	skillType := args[0]
	capabilitiesStr := args[1]

	// Load configuration
	cfg, err := config.LoadConfig("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create KNIRVGRAPH client
	graphClient := core.NewKNIRVGraphClient(&cfg.KNIRV.Services.KNIRVGraph, log)

	// Connect to KNIRVGRAPH
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := graphClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to KNIRVGRAPH: %w", err)
	}
	defer graphClient.Disconnect()

	// Parse capabilities
	capsList := parseCapabilities(capabilitiesStr)

	// Create SkillNode
	skillNode := &core.SkillNode{
		ID:           generateNodeID("skill"),
		SkillType:    skillType,
		Capabilities: capsList,
		Requirements: make(map[string]interface{}),
		Timestamp:    time.Now().Format(time.RFC3339),
	}

	// Submit SkillNode
	if err := graphClient.SubmitSkillNode(ctx, skillNode); err != nil {
		return fmt.Errorf("failed to submit skill node: %w", err)
	}

	fmt.Printf("SkillNode submitted successfully!\n")
	fmt.Printf("ID: %s\n", skillNode.ID)
	fmt.Printf("Type: %s\n", skillNode.SkillType)
	fmt.Printf("Capabilities: %v\n", skillNode.Capabilities)

	return nil
}

func runNRVQuerySkills(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create KNIRVGRAPH client
	graphClient := core.NewKNIRVGraphClient(&cfg.KNIRV.Services.KNIRVGraph, log)

	// Connect to KNIRVGRAPH
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := graphClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to KNIRVGRAPH: %w", err)
	}
	defer graphClient.Disconnect()

	// Query skills
	skills, err := graphClient.GetAllSkills(ctx)
	if err != nil {
		return fmt.Errorf("failed to query skills: %w", err)
	}

	if len(skills) == 0 {
		fmt.Println("No skills found in the NRV system")
		return nil
	}

	fmt.Printf("Found %d skills in the NRV system:\n\n", len(skills))

	// Display skills
	for _, skill := range skills {
		fmt.Printf("Skill ID: %s\n", skill.ID)
		fmt.Printf("  Type: %s\n", skill.SkillType)
		fmt.Printf("  Capabilities: %v\n", skill.Capabilities)
		if skill.Performance != nil {
			fmt.Printf("  Success Rate: %.2f%%\n", skill.Performance.SuccessRate*100)
			fmt.Printf("  Avg Resolution Time: %.2f seconds\n", skill.Performance.AvgResolutionTime)
		}
		if skill.Validation != nil {
			fmt.Printf("  Validated: %t\n", skill.Validation.IsValidated)
			if skill.Validation.IsValidated {
				fmt.Printf("  Validation Score: %.2f\n", skill.Validation.ValidationScore)
			}
		}
		fmt.Println()
	}

	return nil
}

func runNRVQueryErrors(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create KNIRVGRAPH client
	graphClient := core.NewKNIRVGraphClient(&cfg.KNIRV.Services.KNIRVGraph, log)

	// Connect to KNIRVGRAPH
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := graphClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to KNIRVGRAPH: %w", err)
	}
	defer graphClient.Disconnect()

	// Query errors
	errors, err := graphClient.GetAllErrors(ctx)
	if err != nil {
		return fmt.Errorf("failed to query errors: %w", err)
	}

	if len(errors) == 0 {
		fmt.Println("No errors found in the NRV system")
		return nil
	}

	fmt.Printf("Found %d errors in the NRV system:\n\n", len(errors))

	// Display errors
	for _, errorNode := range errors {
		fmt.Printf("Error ID: %s\n", errorNode.ID)
		fmt.Printf("  Type: %s\n", errorNode.ErrorType)
		fmt.Printf("  Description: %s\n", errorNode.Description)
		fmt.Printf("  Severity: %d\n", errorNode.Severity)
		fmt.Printf("  Status: %s\n", errorNode.ResolutionStatus)
		if len(errorNode.ResolvedBy) > 0 {
			fmt.Printf("  Resolved By: %v\n", errorNode.ResolvedBy)
		}
		fmt.Println()
	}

	return nil
}

func runNRVStats(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create KNIRVGRAPH client
	graphClient := core.NewKNIRVGraphClient(&cfg.KNIRV.Services.KNIRVGraph, log)

	// Connect to KNIRVGRAPH
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := graphClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to KNIRVGRAPH: %w", err)
	}
	defer graphClient.Disconnect()

	// Get graph statistics
	stats, err := graphClient.GetGraphStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get graph stats: %w", err)
	}

	fmt.Println("NRV System Statistics:")
	fmt.Println("======================")
	fmt.Printf("Graph Height: %d\n", stats.Height)
	fmt.Printf("Total Nodes: %d\n", stats.TotalNodes)
	fmt.Printf("Total Edges: %d\n", stats.TotalEdges)
	fmt.Printf("Skill Nodes: %d\n", stats.TotalSkillNodes)
	fmt.Printf("Error Nodes: %d\n", stats.TotalErrorNodes)
	fmt.Printf("NRV Vectors: %d\n", stats.TotalVectors)
	fmt.Printf("Avg Resolution Time: %.2f seconds\n", stats.AvgResolutionTime)

	return nil
}

// Helper functions

func generateNodeID(nodeType string) string {
	return fmt.Sprintf("%s_%d", nodeType, time.Now().UnixNano())
}

func parseContext(contextStr string) map[string]interface{} {
	if contextStr == "" {
		return make(map[string]interface{})
	}
	// TODO: Parse JSON context
	return map[string]interface{}{"raw": contextStr}
}

func parseCapabilities(capabilitiesStr string) []string {
	if capabilitiesStr == "" {
		return []string{}
	}
	// Split by comma and trim spaces
	caps := make([]string, 0)
	for _, cap := range splitAndTrim(capabilitiesStr, ",") {
		if cap != "" {
			caps = append(caps, cap)
		}
	}
	return caps
}

func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, part := range splitString(s, sep) {
		trimmed := trimString(part)
		parts = append(parts, trimmed)
	}
	return parts
}

func splitString(s, sep string) []string {
	// Simple string split implementation
	if s == "" {
		return []string{}
	}
	// TODO: Implement proper string splitting
	return []string{s} // Simplified for now
}

func trimString(s string) string {
	// Simple string trim implementation
	return s // Simplified for now
}
