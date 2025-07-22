package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	parser "github.com/cloud-equities/KNIRVCHAIN/sdk/go/transmission/parser"
)

func main() {
	// Parse command line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: example <knirv-uri>")
		fmt.Println("Example: example knirv://mychain-alpha.chain/block?number=123")
		os.Exit(1)
	}
	uriString := os.Args[1]

	// Parse the URI to validate it
	parsedURI, err := parser.Parse(uriString)
	if err != nil {
		log.Fatalf("Failed to parse URI: %v", err)
	}

	fmt.Println("Parsed URI:")
	fmt.Printf("  ID:           %s\n", parsedURI.ID)
	fmt.Printf("  ResourceType: %s\n", parsedURI.ResourceType)
	fmt.Printf("  Path:         %s\n", parsedURI.Path)
	fmt.Printf("  Raw URI:      %s\n", parsedURI.Raw)
	fmt.Println("  Query Parameters:")
	for key, values := range parsedURI.Query {
		for _, value := range values {
			fmt.Printf("    %s: %s\n", key, value)
		}
	}

	// Create a client with default configuration
	config := DefaultConfig()
	config.LogEnabled = true
	knirvClient, err := New(config, nil)
	if err != nil {
		log.Fatalf("Failed to create KNIRV client: %v", err)
	}
	defer knirvClient.Close()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		knirvClient.Close()
		os.Exit(0)
	}()

	// Bootstrap the client
	fmt.Println("Bootstrapping client...")
	if err := knirvClient.Bootstrap(); err != nil {
		log.Fatalf("Failed to bootstrap client: %v", err)
	}

	// Fetch the resource
	fmt.Printf("Fetching resource: %s\n", uriString)
	resource, err := knirvClient.FetchResource(uriString)
	if err != nil {
		log.Fatalf("Failed to fetch resource: %v", err)
	}

	// Print the resource data
	fmt.Println("Resource data:")
	fmt.Println(resource.String())
}
