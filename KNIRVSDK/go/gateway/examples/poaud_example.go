// PoAu-D SDK Usage Example
// This example demonstrates how to use the KNIRV SDK to interact with the PoAu-D consensus mechanism

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloud-equities/KNIRVGATEWAY/sdk/go/gateway"
	"github.com/cloud-equities/KNIRVGATEWAY/sdk/go/gateway/option"
)

func main() {
	// Create a new PoAu-D client
	client := gateway.NewPoAuDClient(
		option.WithBaseURL("http://localhost:8000"), // Gateway URL
		option.WithAPIKey("your-api-key-here"),      // Optional API key
	)

	ctx := context.Background()

	// Example 1: Check current PoAu-D status
	fmt.Println("=== PoAu-D Status Check ===")
	status, err := client.GetConsensusStatus(ctx)
	if err != nil {
		log.Printf("Error getting PoAu-D status: %v", err)
	} else {
		fmt.Printf("PoAu-D Enabled: %t\n", status.Enabled)
		fmt.Printf("Network Authors Count: %d\n", status.NetworkAuthorsCount)
		if status.Enabled {
			fmt.Printf("Main Pool Size: %d\n", status.MainPoolSize)
			fmt.Printf("PAS Pool Size: %d\n", status.PasPoolSize)
			fmt.Printf("Delegated Transactions: %d\n", status.DelegatedTransactions)
		}
	}

	// Example 2: Enable PoAu-D consensus
	fmt.Println("\n=== Enabling PoAu-D Consensus ===")
	enableResp, err := client.EnableConsensus(ctx)
	if err != nil {
		log.Printf("Error enabling PoAu-D: %v", err)
	} else {
		fmt.Printf("Enable Response: %s\n", enableResp.Message)
	}

	// Example 3: Add Network Authors
	fmt.Println("\n=== Adding Network Authors ===")
	networkAuthors := []string{
		"knirv1abc123def456ghi789",
		"knirv1xyz789uvw456rst123",
		"knirv1mno345pqr678stu901",
	}

	for _, author := range networkAuthors {
		addResp, err := client.AddNetworkAuthor(ctx, author)
		if err != nil {
			log.Printf("Error adding network author %s: %v", author, err)
		} else {
			fmt.Printf("Added Network Author: %s - %s\n", author, addResp.Message)
		}
	}

	// Example 4: List all Network Authors
	fmt.Println("\n=== Listing Network Authors ===")
	authors, err := client.ListNetworkAuthors(ctx)
	if err != nil {
		log.Printf("Error listing network authors: %v", err)
	} else {
		fmt.Printf("Total Network Authors: %d\n", authors.Count)
		for i, author := range authors.NetworkAuthors {
			fmt.Printf("  %d. %s\n", i+1, author)
		}
	}

	// Example 5: Check if specific address is a Network Author
	fmt.Println("\n=== Checking Network Author Status ===")
	testAddress := "knirv1abc123def456ghi789"
	isAuthor, err := client.IsNetworkAuthor(ctx, testAddress)
	if err != nil {
		log.Printf("Error checking network author status: %v", err)
	} else {
		fmt.Printf("Address %s is Network Author: %t\n", testAddress, isAuthor)
	}

	// Example 6: Get delegation statistics
	fmt.Println("\n=== Delegation Statistics ===")
	stats, err := client.GetDelegationStatistics(ctx)
	if err != nil {
		log.Printf("Error getting delegation statistics: %v", err)
	} else {
		fmt.Println("Delegation Statistics:")
		for key, value := range stats {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	// Example 7: Monitor PoAu-D status over time
	fmt.Println("\n=== Monitoring PoAu-D Status ===")
	monitorPoAuDStatus(client, ctx, 3) // Monitor for 3 iterations

	// Example 8: Remove a Network Author
	fmt.Println("\n=== Removing Network Author ===")
	removeAddress := "knirv1xyz789uvw456rst123"
	removeResp, err := client.RemoveNetworkAuthor(ctx, removeAddress)
	if err != nil {
		log.Printf("Error removing network author: %v", err)
	} else {
		fmt.Printf("Removed Network Author: %s - %s\n", removeAddress, removeResp.Message)
	}

	// Example 9: Disable PoAu-D consensus
	fmt.Println("\n=== Disabling PoAu-D Consensus ===")
	disableResp, err := client.DisableConsensus(ctx)
	if err != nil {
		log.Printf("Error disabling PoAu-D: %v", err)
	} else {
		fmt.Printf("Disable Response: %s\n", disableResp.Message)
	}

	// Example 10: Final status check
	fmt.Println("\n=== Final Status Check ===")
	finalStatus, err := client.GetConsensusStatus(ctx)
	if err != nil {
		log.Printf("Error getting final status: %v", err)
	} else {
		fmt.Printf("Final PoAu-D Enabled: %t\n", finalStatus.Enabled)
	}

	fmt.Println("\n=== PoAu-D SDK Example Complete ===")
}

// monitorPoAuDStatus demonstrates continuous monitoring of PoAu-D status
func monitorPoAuDStatus(client *gateway.PoAuDClient, ctx context.Context, iterations int) {
	for i := 0; i < iterations; i++ {
		status, err := client.GetConsensusStatus(ctx)
		if err != nil {
			log.Printf("Monitor iteration %d - Error: %v", i+1, err)
		} else {
			fmt.Printf("Monitor %d - Enabled: %t, NAPs: %d, Delegated: %d\n",
				i+1, status.Enabled, status.NetworkAuthorsCount, status.DelegatedTransactions)
		}
		
		if i < iterations-1 {
			time.Sleep(2 * time.Second)
		}
	}
}

// demonstrateErrorHandling shows how to handle common errors
func demonstrateErrorHandling() {
	client := gateway.NewPoAuDClient()
	ctx := context.Background()

	// Example of handling validation errors
	err := gateway.ValidateNetworkAuthor("")
	if err != nil {
		fmt.Printf("Validation error: %v\n", err)
	}

	err = gateway.ValidateNetworkAuthor("short")
	if err != nil {
		fmt.Printf("Validation error: %v\n", err)
	}

	// Example of handling API errors
	_, err = client.AddNetworkAuthor(ctx, "invalid-address")
	if err != nil {
		fmt.Printf("API error: %v\n", err)
	}
}

// demonstrateAdvancedUsage shows advanced PoAu-D SDK features
func demonstrateAdvancedUsage() {
	// Get default configuration
	config := gateway.GetDefaultPoAuDConfig()
	fmt.Printf("Default PoAu-D Config: %+v\n", config)

	// Create client with custom options
	client := gateway.NewPoAuDClient(
		option.WithBaseURL("https://api.knirv.network"),
		option.WithAPIKey("production-api-key"),
		// Add more options as needed
	)

	ctx := context.Background()

	// Check if PoAu-D is enabled before performing operations
	enabled, err := client.IsPoAuDEnabled(ctx)
	if err != nil {
		log.Printf("Error checking PoAu-D status: %v", err)
		return
	}

	if !enabled {
		fmt.Println("PoAu-D is not enabled, enabling it first...")
		_, err := client.EnableConsensus(ctx)
		if err != nil {
			log.Printf("Error enabling PoAu-D: %v", err)
			return
		}
	}

	// Get network author count
	count, err := client.GetNetworkAuthorCount(ctx)
	if err != nil {
		log.Printf("Error getting network author count: %v", err)
		return
	}

	fmt.Printf("Current Network Author Count: %d\n", count)

	// Perform operations based on current state
	if count == 0 {
		fmt.Println("No Network Authors found, adding initial authors...")
		// Add initial network authors
	} else {
		fmt.Printf("Found %d Network Authors, system is ready\n", count)
	}
}

// Example of batch operations
func demonstrateBatchOperations() {
	client := gateway.NewPoAuDClient()
	ctx := context.Background()

	// Batch add multiple network authors
	authors := []string{
		"knirv1batch001",
		"knirv1batch002", 
		"knirv1batch003",
	}

	fmt.Println("Adding multiple Network Authors...")
	for i, author := range authors {
		resp, err := client.AddNetworkAuthor(ctx, author)
		if err != nil {
			log.Printf("Failed to add author %d (%s): %v", i+1, author, err)
		} else {
			fmt.Printf("Added author %d: %s\n", i+1, resp.Message)
		}
	}

	// Verify all were added
	authorsList, err := client.ListNetworkAuthors(ctx)
	if err != nil {
		log.Printf("Error verifying authors: %v", err)
		return
	}

	fmt.Printf("Total authors after batch add: %d\n", authorsList.Count)
}
