package governance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

var ReliabilityCmd = &cobra.Command{
	Use:   "reliability",
	Short: "Manage circuit breakers, error budgets, kill switches, SLOs",
}

var listBreakersCmd = &cobra.Command{
	Use:   "list-breakers",
	Short: "List circuit breakers",
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("%s/api/v1/governance/reliability/breakers", serverURL)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var recordSuccessCmd = &cobra.Command{
	Use:   "record-success <breaker-id>",
	Short: "Record a success for a circuit breaker",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("%s/api/v1/governance/reliability/breakers/%s/success", serverURL, args[0])
		resp, err := http.Post(url, "application/json", nil)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var recordFailureCmd = &cobra.Command{
	Use:   "record-failure <breaker-id>",
	Short: "Record a failure for a circuit breaker",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("%s/api/v1/governance/reliability/breakers/%s/failure", serverURL, args[0])
		resp, err := http.Post(url, "application/json", nil)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var defineSLO = &cobra.Command{
	Use:   "define-slo",
	Short: "Define an SLO",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		target, _ := cmd.Flags().GetFloat64("target")
		metric, _ := cmd.Flags().GetString("metric")
		body := map[string]interface{}{"name": name, "target": target, "metric_name": metric}
		data, _ := json.Marshal(body)
		url := fmt.Sprintf("%s/api/v1/governance/reliability/slos/define", serverURL)
		resp, err := http.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var listSLOsCmd = &cobra.Command{
	Use:   "list-slos",
	Short: "List SLOs",
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("%s/api/v1/governance/reliability/slos", serverURL)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var armKillSwitchCmd = &cobra.Command{
	Use:   "arm-kill-switch",
	Short: "Arm a kill switch for an agent",
	Run: func(cmd *cobra.Command, args []string) {
		agentID, _ := cmd.Flags().GetString("agent-id")
		nodeID, _ := cmd.Flags().GetString("node-id")
		body := map[string]string{"agent_id": agentID, "node_id": nodeID}
		data, _ := json.Marshal(body)
		url := fmt.Sprintf("%s/api/v1/governance/reliability/kill-switches", serverURL)
		resp, err := http.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var listKillSwitchesCmd = &cobra.Command{
	Use:   "list-kill-switches",
	Short: "List kill switches",
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("%s/api/v1/governance/reliability/kill-switches/list", serverURL)
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
	ReliabilityCmd.AddCommand(listBreakersCmd)
	ReliabilityCmd.AddCommand(recordSuccessCmd)
	ReliabilityCmd.AddCommand(recordFailureCmd)
	ReliabilityCmd.AddCommand(defineSLO)
	ReliabilityCmd.AddCommand(listSLOsCmd)
	ReliabilityCmd.AddCommand(armKillSwitchCmd)
	ReliabilityCmd.AddCommand(listKillSwitchesCmd)

	defineSLO.Flags().String("name", "", "SLO name")
	defineSLO.Flags().Float64("target", 0.99, "SLO target (0-1)")
	defineSLO.Flags().String("metric", "", "Metric name")
	defineSLO.MarkFlagRequired("name")

	armKillSwitchCmd.Flags().String("agent-id", "", "Agent ID")
	armKillSwitchCmd.Flags().String("node-id", "", "Node ID")
	armKillSwitchCmd.MarkFlagRequired("agent-id")
	armKillSwitchCmd.MarkFlagRequired("node-id")
}
