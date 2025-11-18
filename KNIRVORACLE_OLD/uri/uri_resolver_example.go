package uri

import (
	"fmt"
	"log"
)

// RunExample demonstrates how to use the URI resolver
func RunURIExample() {
	// Create a new URI resolver
	resolver := NewURIResolver()

	// Example URIs to resolve
	uris := []string{
		"agent://tunnel.agent.com/abc123.dev/some/path?query=value",
		"agent://tunnel.agent.com/agent-default.chain",
		"agent://192.168.1.100/QmExamplePeerID.dev",
	}

	for _, uri := range uris {
		fmt.Printf("\nResolving URI: %s\n", uri)

		// Parse the URI
		authority, identifier, resourceType, subPath, err := resolver.ParseURI(uri)
		if err != nil {
			log.Printf("Failed to parse URI: %v", err)
			continue
		}

		fmt.Printf("Parsed URI:\n")
		fmt.Printf("  Authority: %s\n", authority)
		fmt.Printf("  Identifier: %s\n", identifier)
		fmt.Printf("  Resource Type: %s\n", resourceType)
		fmt.Printf("  Sub Path: %s\n", subPath)

		// In a real application, you would resolve the URI and connect to it
		// For this example, we'll just show how it would be done
		fmt.Printf("To resolve and connect to this URI, you would use:\n")
		fmt.Printf("  resolved, err := resolver.ResolveURI(uri)\n")
		fmt.Printf("  if err != nil { handle error }\n")
		fmt.Printf("  conn, err := resolver.ConnectToURI(uri)\n")
		fmt.Printf("  if err != nil { handle error }\n")
		fmt.Printf("  // Use conn to communicate with the target\n")
	}
}
