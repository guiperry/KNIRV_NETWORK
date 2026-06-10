package governance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

var PolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage policy adapter and Rego evaluation",
}

var normalizeInputCmd = &cobra.Command{
	Use:   "normalize-input",
	Short: "Normalize a policy input",
	Run: func(cmd *cobra.Command, args []string) {
		nodeID, _ := cmd.Flags().GetString("node-id")
		action, _ := cmd.Flags().GetString("action")
		actionType, _ := cmd.Flags().GetString("action-type")
		body := map[string]string{"node_id": nodeID, "action": action, "action_type": actionType}
		data, _ := json.Marshal(body)
		url := fmt.Sprintf("%s/api/v1/governance/policy/inputs", serverURL)
		resp, err := http.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var listInputsCmd = &cobra.Command{
	Use:   "list-inputs",
	Short: "List policy inputs",
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("%s/api/v1/governance/policy/inputs", serverURL)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var getContractCmd = &cobra.Command{
	Use:   "contract",
	Short: "Get the portability contract",
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("%s/api/v1/governance/policy/contract", serverURL)
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
	PolicyCmd.AddCommand(normalizeInputCmd)
	PolicyCmd.AddCommand(listInputsCmd)
	PolicyCmd.AddCommand(getContractCmd)

	normalizeInputCmd.Flags().String("node-id", "", "Node ID")
	normalizeInputCmd.Flags().String("action", "", "Action name")
	normalizeInputCmd.Flags().String("action-type", "", "Action type")
	normalizeInputCmd.MarkFlagRequired("node-id")
	normalizeInputCmd.MarkFlagRequired("action")
	normalizeInputCmd.MarkFlagRequired("action-type")
}
