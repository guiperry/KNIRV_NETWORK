package main

import (
	"blockchain-app/internal/types"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	neturl "net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	rpcURL = "http://localhost:8080"
)

// makeHTTPRequest performs an HTTP GET request and returns the response body
func makeHTTPRequest(url string) ([]byte, error) {
	// Validate URL to prevent SSRF attacks
	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	resp, err := http.Get(url) // #nosec G107 - URL is validated above
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Error closing response body: %v", closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	return body, nil
}

// makeHTTPPostRequest performs an HTTP POST request and returns the response body
func makeHTTPPostRequest(url string, data []byte) ([]byte, error) {
	// Validate URL to prevent SSRF attacks
	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data)) // #nosec G107 - URL is validated above
	if err != nil {
		return nil, fmt.Errorf("HTTP POST request failed: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Error closing response body: %v", closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	return body, nil
}

// prettyPrintJSON formats and prints JSON data
func prettyPrintJSON(data []byte) error {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, data, "", "  ")
	if err != nil {
		return fmt.Errorf("error formatting JSON: %v", err)
	}
	fmt.Println(prettyJSON.String())
	return nil
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "graphchain-cli",
		Short: "GraphChain CLI tool",
		Long:  "Command line interface for interacting with the GraphChain",
	}

	rootCmd.PersistentFlags().StringVar(&rpcURL, "rpc", "http://localhost:8080", "RPC server URL")

	// Add commands
	rootCmd.AddCommand(
		getHeightCmd(),
		getNodeCmd(),
		getEdgeCmd(),
		getHeadsCmd(),
		getNeighborsCmd(),
		findPathCmd(),
		getAccountCmd(),
		createNodeCmd(),
		createEdgeCmd(),
		sendGraphTxCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getHeightCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "height",
		Short: "Get current GraphChain height",
		Run: func(cmd *cobra.Command, args []string) {
			body, err := makeHTTPRequest(rpcURL + "/height")
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}

			fmt.Println(string(body))
		},
	}
}

func getNodeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "node [nodeID]",
		Short: "Get node by ID",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			nodeID := args[0]
			body, err := makeHTTPRequest(rpcURL + "/node/" + nodeID)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}

			var node types.GraphNode
			if err := json.Unmarshal(body, &node); err != nil {
				fmt.Printf("Error parsing node: %v\n", err)
				return
			}

			prettyJSON, _ := json.MarshalIndent(node, "", "  ")
			fmt.Println(string(prettyJSON))
		},
	}
}

func getEdgeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edge [edgeID]",
		Short: "Get edge by ID",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			edgeID := args[0]
			body, err := makeHTTPRequest(rpcURL + "/edge/" + edgeID)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}

			var edge types.Edge
			if err := json.Unmarshal(body, &edge); err != nil {
				fmt.Printf("Error parsing edge: %v\n", err)
				return
			}

			prettyJSON, _ := json.MarshalIndent(edge, "", "  ")
			fmt.Println(string(prettyJSON))
		},
	}
}

func getHeadsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "heads",
		Short: "Get current graph heads",
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := http.Get(rpcURL + "/graph/heads")
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading response: %v\n", err)
				return
			}

			fmt.Println(string(body))
		},
	}
}

func getNeighborsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "neighbors [nodeID]",
		Short: "Get neighbors of a node",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			nodeID := args[0]
			resp, err := http.Get(rpcURL + "/graph/neighbors/" + nodeID)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading response: %v\n", err)
				return
			}

			fmt.Println(string(body))
		},
	}
}

func findPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path [from] [to]",
		Short: "Find path between two nodes",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			from := args[0]
			to := args[1]
			resp, err := http.Get(rpcURL + "/graph/path/" + from + "/" + to)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading response: %v\n", err)
				return
			}

			// Pretty print JSON response
			var prettyJSON bytes.Buffer
			err = json.Indent(&prettyJSON, body, "", "  ")
			if err != nil {
				fmt.Printf("Error formatting JSON: %v\n", err)
				fmt.Println(string(body))
				return
			}

			fmt.Println(prettyJSON.String())
		},
	}
}

func getAccountCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "account [address]",
		Short: "Get account information",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			address := args[0]
			resp, err := http.Get(rpcURL + "/account/" + address)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading response: %v\n", err)
				return
			}

			var account types.Account
			if err := json.Unmarshal(body, &account); err != nil {
				fmt.Printf("Error parsing account: %v\n", err)
				return
			}

			prettyJSON, _ := json.MarshalIndent(account, "", "  ")
			fmt.Println(string(prettyJSON))
		},
	}
}

func createNodeCmd() *cobra.Command {
	var (
		nodeID  string
		parents string
		weight  float64
	)

	cmd := &cobra.Command{
		Use:   "create-node",
		Short: "Create a new graph node",
		Run: func(cmd *cobra.Command, args []string) {
			var parentsList []string
			if parents != "" {
				parentsList = strings.Split(parents, ",")
			}

			node := types.NewGraphNode(nodeID, parentsList, types.GraphData{})
			node.Weight = weight

			nodeData, err := json.Marshal(node)
			if err != nil {
				fmt.Printf("Error marshaling node: %v\n", err)
				return
			}

			resp, err := http.Post(rpcURL+"/node", "application/json", bytes.NewBuffer(nodeData))
			if err != nil {
				fmt.Printf("Error creating node: %v\n", err)
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading response: %v\n", err)
				return
			}

			fmt.Println(string(body))
		},
	}

	cmd.Flags().StringVar(&nodeID, "id", "", "Node ID")
	cmd.Flags().StringVar(&parents, "parents", "", "Comma-separated parent node IDs")
	cmd.Flags().Float64Var(&weight, "weight", 1.0, "Node weight")

	if err := cmd.MarkFlagRequired("id"); err != nil {
		log.Printf("Error marking flag as required: %v", err)
	}

	return cmd
}

func createEdgeCmd() *cobra.Command {
	var (
		from     string
		to       string
		weight   float64
		edgeType int
	)

	cmd := &cobra.Command{
		Use:   "create-edge",
		Short: "Create a new graph edge",
		Run: func(cmd *cobra.Command, args []string) {
			edge := types.NewEdge(from, to, types.EdgeType(edgeType), weight)

			edgeData, err := json.Marshal(edge)
			if err != nil {
				fmt.Printf("Error marshaling edge: %v\n", err)
				return
			}

			resp, err := http.Post(rpcURL+"/edge", "application/json", bytes.NewBuffer(edgeData))
			if err != nil {
				fmt.Printf("Error creating edge: %v\n", err)
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading response: %v\n", err)
				return
			}

			fmt.Println(string(body))
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "From node ID")
	cmd.Flags().StringVar(&to, "to", "", "To node ID")
	cmd.Flags().Float64Var(&weight, "weight", 1.0, "Edge weight")
	cmd.Flags().IntVar(&edgeType, "type", 0, "Edge type (0=Transaction, 1=Validation, 2=Consensus, 3=State, 4=Custom)")

	if err := cmd.MarkFlagRequired("from"); err != nil {
		log.Printf("Error marking flag as required: %v", err)
	}
	if err := cmd.MarkFlagRequired("to"); err != nil {
		log.Printf("Error marking flag as required: %v", err)
	}

	return cmd
}

func sendGraphTxCmd() *cobra.Command {
	var (
		from   string
		to     string
		amount uint64
		fee    uint64
		txType int
	)

	cmd := &cobra.Command{
		Use:   "send-tx",
		Short: "Send a graph transaction",
		Run: func(cmd *cobra.Command, args []string) {
			tx := types.NewGraphTransaction(types.GraphTxType(txType), from, to, amount, fee, []byte{})

			txData, err := json.Marshal(tx)
			if err != nil {
				fmt.Printf("Error marshaling graph transaction: %v\n", err)
				return
			}

			resp, err := http.Post(rpcURL+"/transaction", "application/json", bytes.NewBuffer(txData))
			if err != nil {
				fmt.Printf("Error sending graph transaction: %v\n", err)
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading response: %v\n", err)
				return
			}

			fmt.Println(string(body))
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "From address")
	cmd.Flags().StringVar(&to, "to", "", "To address")
	cmd.Flags().Uint64Var(&amount, "amount", 0, "Amount to send")
	cmd.Flags().Uint64Var(&fee, "fee", 1, "Transaction fee")
	cmd.Flags().IntVar(&txType, "type", 5, "Transaction type (0=CreateNode, 1=CreateEdge, 2=UpdateNode, 3=DeleteNode, 4=DeleteEdge, 5=Transfer)")

	if err := cmd.MarkFlagRequired("from"); err != nil {
		log.Printf("Error marking flag as required: %v", err)
	}

	return cmd
}
