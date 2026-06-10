package governance

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

var oracleURL string

var DIDCmd = &cobra.Command{
	Use:   "did",
	Short: "Manage DIDs (Decentralized Identifiers)",
}

var didRegisterCmd = &cobra.Command{
	Use:   "register <did-json>",
	Short: "Register a DID document",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("%s/oracle/v3/did/register", oracleURL)
		resp, err := http.Post(url, "application/json", strings.NewReader(args[0]))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
	},
}

var didResolveCmd = &cobra.Command{
	Use:   "resolve <did>",
	Short: "Resolve a DID to a DID document",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("%s/oracle/v3/did/%s", oracleURL, args[0])
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		var doc map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&doc)
		b, _ := json.MarshalIndent(doc, "", "  ")
		fmt.Println(string(b))
	},
}

var didDeactivateCmd = &cobra.Command{
	Use:   "deactivate <did>",
	Short: "Deactivate a DID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("%s/oracle/v3/did/%s/deactivate", oracleURL, args[0])
		resp, err := http.Post(url, "application/json", nil)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
	},
}

func init() {
	DIDCmd.AddCommand(didRegisterCmd)
	DIDCmd.AddCommand(didResolveCmd)
	DIDCmd.AddCommand(didDeactivateCmd)

	DIDCmd.PersistentFlags().StringVar(&oracleURL, "oracle-url", "http://localhost:1317", "KNIRVORACLE API URL")
}
