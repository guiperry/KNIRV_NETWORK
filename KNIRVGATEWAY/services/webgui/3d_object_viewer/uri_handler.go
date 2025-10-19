// KNIRV URI Handler
//
// This script provides functionality for handling custom URI schemes in the KNIRV application:
// - chain:// - For KNIRVCHAIN Verifiers
// - nrn:// - For privately shared NRN Assets
//
// The application can be launched via URI and will parse the URI to extract relevant information.

package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// AssetMetadata represents the metadata extracted from an NRN URI
type AssetMetadata struct {
	AssetID string
	Author  string
	Version int
	License string
	Network string
	Relay   string
}

// parseChainURI parses a chain:// URI and extracts parameters.
// This URI is used for KNIRVCHAIN Verifiers to:
// - Identify specific verifier nodes
// - Access verifier APIs
// - Establish peer-to-peer connections
// - Manage node reputation or staking
func parseChainURI(uriString string) (map[string]string, error) {
	u, err := url.Parse(uriString)
	if err != nil {
		return nil, fmt.Errorf("invalid chain URI: %w", err)
	}

	if u.Scheme != "chain" {
		return nil, fmt.Errorf("invalid scheme: expected 'chain'")
	}

	// Extract host and path information
	params := make(map[string]string)
	params["host"] = u.Host
	params["path"] = u.Path

	// Extract query parameters
	for key, values := range u.Query() {
		params[key] = strings.Join(values, ",") // Handle multiple values if needed
	}

	return params, nil
}

// parseNRNURI parses the nrn:// URI and extracts the metadata.
// This URI is used for privately shared NRN Assets to:
// - Identify and locate 3D assets
// - Specify asset metadata (author, version, license)
// - Facilitate access to the asset content
func parseNRNURI(uriString string) (AssetMetadata, error) {
	u, err := url.Parse(uriString)
	if err != nil {
		return AssetMetadata{}, fmt.Errorf("invalid URI: %w", err)
	}

	if u.Scheme != "nrn" {
		return AssetMetadata{}, fmt.Errorf("invalid scheme: expected 'nrn'")
	}

	assetID := u.Path
	if len(assetID) > 0 && assetID[0] == '/' {
		assetID = assetID[1:] // Remove leading slash
	}

	author := u.Query().Get("author")
	versionStr := u.Query().Get("version")
	license := u.Query().Get("license")
	network := u.Query().Get("network")
	relay := u.Query().Get("relay")

	// Default version to 1 if not specified
	version := 1
	if versionStr != "" {
		var err error
		version, err = strconv.Atoi(versionStr)
		if err != nil {
			return AssetMetadata{}, fmt.Errorf("invalid version: %w", err)
		}
	}

	metadata := AssetMetadata{
		AssetID: assetID,
		Author:  author,
		Version: version,
		License: license,
		Network: network,
		Relay:   relay,
	}

	return metadata, nil
}

// handleURI determines the URI type and processes it accordingly
func handleURI(uri string) error {
	// Check if it's a chain:// URI
	if strings.HasPrefix(uri, "chain://") {
		verifierInfo, err := parseChainURI(uri)
		if err != nil {
			return fmt.Errorf("error parsing chain URI: %w", err)
		}
		fmt.Printf("Verifier Info: %+v\n", verifierInfo)
		// Process verifier information here
		// ...
		return nil
	}

	// Check if it's an nrn:// URI
	if strings.HasPrefix(uri, "nrn://") {
		metadata, err := parseNRNURI(uri)
		if err != nil {
			return fmt.Errorf("error parsing NRN URI: %w", err)
		}
		fmt.Printf("Asset Metadata: %+v\n", metadata)
		// Retrieve content, check permissions, and display content
		// ...
		return nil
	}

	return fmt.Errorf("unsupported URI scheme: %s", uri)
}

func main() {
	// Check for command-line arguments
	if len(os.Args) > 1 {
		uri := os.Args[1] // The URI will be the first argument
		fmt.Println("Application launched with URI:", uri)

		// Handle the URI
		err := handleURI(uri)
		if err != nil {
			log.Println("Error handling URI:", err)
			// Display an error message to the user
		}
	} else {
		fmt.Println("Application launched without a URI.")
		// Normal application startup

		// For demonstration purposes, we'll test the URI parsers with examples
		testURIParsers()
	}

	// Your GUI or other application logic would go here
	// ...
}

// testURIParsers demonstrates the URI parsing functionality with example URIs
func testURIParsers() {
	fmt.Println("\n--- Testing URI Parsers ---")

	// Test chain:// URI
	chainURI := "chain://verifier1.example.com/status?nodeID=xyz123&region=US"
	fmt.Println("Testing chain URI:", chainURI)
	verifierInfo, err := parseChainURI(chainURI)
	if err != nil {
		fmt.Printf("Error parsing chain URI: %v\n", err)
	} else {
		fmt.Printf("Verifier Info: %+v\n", verifierInfo)
	}

	// Test nrn:// URI
	nrnURI := "nrn://asset123?author=creator&version=2&license=MIT&network=mainnet&relay=relay.example.com"
	fmt.Println("\nTesting NRN URI:", nrnURI)
	parsedMetadata, err := parseNRNURI(nrnURI)
	if err != nil {
		fmt.Printf("Error parsing NRN URI: %v\n", err)
	} else {
		fmt.Printf("Asset Metadata: %+v\n", parsedMetadata)
	}
}
