package governance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

var ComplianceCmd = &cobra.Command{
	Use:   "compliance",
	Short: "Manage compliance events, frameworks, and chain verification",
}

var recordComplianceEventCmd = &cobra.Command{
	Use:   "record-event",
	Short: "Record a compliance event",
	Run: func(cmd *cobra.Command, args []string) {
		nodeID, _ := cmd.Flags().GetString("node-id")
		agentID, _ := cmd.Flags().GetString("agent-id")
		eventType, _ := cmd.Flags().GetString("event-type")
		severity, _ := cmd.Flags().GetString("severity")
		body := map[string]string{
			"node_id":    nodeID,
			"agent_id":   agentID,
			"event_type": eventType,
			"severity":   severity,
		}
		data, _ := json.Marshal(body)
		url := fmt.Sprintf("%s/api/v1/governance/compliance/events", serverURL)
		resp, err := http.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var listComplianceEventsCmd = &cobra.Command{
	Use:   "list-events",
	Short: "List compliance events",
	Run: func(cmd *cobra.Command, args []string) {
		nodeID, _ := cmd.Flags().GetString("node-id")
		url := fmt.Sprintf("%s/api/v1/governance/compliance/events?node_id=%s", serverURL, nodeID)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var listFrameworksCmd = &cobra.Command{
	Use:   "list-frameworks",
	Short: "List compliance frameworks",
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("%s/api/v1/governance/compliance/frameworks", serverURL)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var getComplianceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get compliance status for a node",
	Run: func(cmd *cobra.Command, args []string) {
		nodeID, _ := cmd.Flags().GetString("node-id")
		url := fmt.Sprintf("%s/api/v1/governance/compliance/status?node_id=%s", serverURL, nodeID)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var verifyComplianceChainCmd = &cobra.Command{
	Use:   "verify-chain",
	Short: "Verify compliance event chain integrity",
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("%s/api/v1/governance/compliance/chain/verify", serverURL)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

func init() {
	ComplianceCmd.AddCommand(recordComplianceEventCmd)
	ComplianceCmd.AddCommand(listComplianceEventsCmd)
	ComplianceCmd.AddCommand(listFrameworksCmd)
	ComplianceCmd.AddCommand(getComplianceStatusCmd)
	ComplianceCmd.AddCommand(verifyComplianceChainCmd)

	recordComplianceEventCmd.Flags().String("node-id", "", "Node ID")
	recordComplianceEventCmd.Flags().String("agent-id", "", "Agent ID")
	recordComplianceEventCmd.Flags().String("event-type", "", "Event type")
	recordComplianceEventCmd.Flags().String("severity", "", "Severity (low/medium/high/critical)")
	recordComplianceEventCmd.MarkFlagRequired("node-id")
	recordComplianceEventCmd.MarkFlagRequired("event-type")

	listComplianceEventsCmd.Flags().String("node-id", "", "Filter by node ID")

	getComplianceStatusCmd.Flags().String("node-id", "", "Node ID")
	getComplianceStatusCmd.MarkFlagRequired("node-id")
}
