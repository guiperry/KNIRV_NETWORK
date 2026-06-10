package governance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

var RevokeCmd = &cobra.Command{
	Use:   "revocation",
	Short: "Manage identity revocation list",
}

var revokeIdentityCmd = &cobra.Command{
	Use:   "revoke",
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

var verifyRevocationChainCmd = &cobra.Command{
	Use:   "verify-chain",
	Short: "Verify the integrity of the revocation chain",
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("%s/api/v1/governance/identity/revocation/chain/verify", serverURL)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var listRevocationsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all revocation entries",
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("%s/api/v1/governance/identity/revocation/list", serverURL)
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
	RevokeCmd.AddCommand(revokeIdentityCmd)
	RevokeCmd.AddCommand(verifyRevocationChainCmd)
	RevokeCmd.AddCommand(listRevocationsCmd)

	revokeIdentityCmd.Flags().String("identity-id", "", "Identity ID")
	revokeIdentityCmd.Flags().String("node-id", "", "Node ID")
	revokeIdentityCmd.Flags().String("reason", "", "Revocation reason")
	revokeIdentityCmd.Flags().String("revoked-by", "", "Revoking authority")
	revokeIdentityCmd.MarkFlagRequired("identity-id")
	revokeIdentityCmd.MarkFlagRequired("node-id")
}
