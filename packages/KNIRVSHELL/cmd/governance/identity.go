package governance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

var serverURL string

var IdentityCmd = &cobra.Command{
	Use:   "identity",
	Short: "Manage identity envelopes and trust mappings",
}

var createEnvelopeCmd = &cobra.Command{
	Use:   "create-envelope",
	Short: "Create a trust envelope for a node",
	Run: func(cmd *cobra.Command, args []string) {
		nodeID, _ := cmd.Flags().GetString("node-id")
		agentID, _ := cmd.Flags().GetString("agent-id")
		source, _ := cmd.Flags().GetString("source")
		body := map[string]string{"node_id": nodeID, "agent_id": agentID, "source": source}
		data, _ := json.Marshal(body)
		url := fmt.Sprintf("%s/api/v1/governance/identity/envelopes", serverURL)
		resp, err := http.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var revokeNodeCmd = &cobra.Command{
	Use:   "revoke-node",
	Short: "Revoke a node identity",
	Run: func(cmd *cobra.Command, args []string) {
		identityID, _ := cmd.Flags().GetString("identity-id")
		nodeID, _ := cmd.Flags().GetString("node-id")
		reason, _ := cmd.Flags().GetString("reason")
		revokedBy, _ := cmd.Flags().GetString("revoked-by")
		body := map[string]string{
			"identity_id": identityID,
			"node_id":     nodeID,
			"reason":      reason,
			"revoked_by":  revokedBy,
		}
		data, _ := json.Marshal(body)
		url := fmt.Sprintf("%s/api/v1/governance/identity/revoke", serverURL)
		resp, err := http.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var checkRevokedCmd = &cobra.Command{
	Use:   "check-revoked <node-id>",
	Short: "Check if a node is revoked",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("%s/api/v1/governance/identity/revoked/%s", serverURL, args[0])
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
	IdentityCmd.AddCommand(createEnvelopeCmd)
	IdentityCmd.AddCommand(revokeNodeCmd)
	IdentityCmd.AddCommand(checkRevokedCmd)

	createEnvelopeCmd.Flags().String("node-id", "", "Node ID")
	createEnvelopeCmd.Flags().String("agent-id", "", "Agent ID")
	createEnvelopeCmd.Flags().String("source", "knirv", "Identity source (knirv/did/oidc/x509)")
	createEnvelopeCmd.MarkFlagRequired("node-id")

	revokeNodeCmd.Flags().String("identity-id", "", "Identity ID")
	revokeNodeCmd.Flags().String("node-id", "", "Node ID")
	revokeNodeCmd.Flags().String("reason", "", "Revocation reason")
	revokeNodeCmd.Flags().String("revoked-by", "", "Revoking authority")

	IdentityCmd.PersistentFlags().StringVar(&serverURL, "server-url", "http://localhost:8084", "KNIRVSERVER API URL")
}
