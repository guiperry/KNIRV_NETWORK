package governance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

var MCPCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage MCP tool hardening, schemas, and poisoning detection",
}

var registerToolCmd = &cobra.Command{
	Use:   "register-tool",
	Short: "Register an MCP tool definition",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		desc, _ := cmd.Flags().GetString("description")
		body := map[string]string{"name": name, "description": desc}
		data, _ := json.Marshal(body)
		url := fmt.Sprintf("%s/api/v1/governance/mcp/tools", serverURL)
		resp, err := http.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var listToolsCmd = &cobra.Command{
	Use:   "list-tools",
	Short: "List registered MCP tools",
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("%s/api/v1/governance/mcp/tools", serverURL)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var validateSchemaCmd = &cobra.Command{
	Use:   "validate-schema",
	Short: "Validate tool arguments against schema",
	Run: func(cmd *cobra.Command, args []string) {
		toolName, _ := cmd.Flags().GetString("tool")
		argsJSON, _ := cmd.Flags().GetString("args")
		var argsMap map[string]interface{}
		json.Unmarshal([]byte(argsJSON), &argsMap)
		body := map[string]interface{}{"tool_name": toolName, "args": argsMap}
		data, _ := json.Marshal(body)
		url := fmt.Sprintf("%s/api/v1/governance/mcp/schema/validate", serverURL)
		resp, err := http.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var scanInjectionCmd = &cobra.Command{
	Use:   "scan-injection",
	Short: "Scan arguments for injection patterns",
	Run: func(cmd *cobra.Command, args []string) {
		argsJSON, _ := cmd.Flags().GetString("args")
		var argsMap map[string]interface{}
		json.Unmarshal([]byte(argsJSON), &argsMap)
		body := map[string]interface{}{"args": argsMap}
		data, _ := json.Marshal(body)
		url := fmt.Sprintf("%s/api/v1/governance/mcp/injection/scan", serverURL)
		resp, err := http.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

var sanitizeResponseCmd = &cobra.Command{
	Use:   "sanitize-response",
	Short: "Sanitize an MCP response for secrets",
	Run: func(cmd *cobra.Command, args []string) {
		content, _ := cmd.Flags().GetString("content")
		body := map[string]string{"content": content}
		data, _ := json.Marshal(body)
		url := fmt.Sprintf("%s/api/v1/governance/mcp/response/sanitize", serverURL)
		resp, err := http.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		io.Copy(cmd.OutOrStdout(), resp.Body)
	},
}

func init() {
	MCPCmd.AddCommand(registerToolCmd)
	MCPCmd.AddCommand(listToolsCmd)
	MCPCmd.AddCommand(validateSchemaCmd)
	MCPCmd.AddCommand(scanInjectionCmd)
	MCPCmd.AddCommand(sanitizeResponseCmd)

	registerToolCmd.Flags().String("name", "", "Tool name")
	registerToolCmd.Flags().String("description", "", "Tool description")
	registerToolCmd.MarkFlagRequired("name")

	validateSchemaCmd.Flags().String("tool", "", "Tool name")
	validateSchemaCmd.Flags().String("args", "{}", "JSON arguments")
	validateSchemaCmd.MarkFlagRequired("tool")

	scanInjectionCmd.Flags().String("args", "{}", "JSON arguments to scan")

	sanitizeResponseCmd.Flags().String("content", "", "Response content to sanitize")
	sanitizeResponseCmd.MarkFlagRequired("content")
}
