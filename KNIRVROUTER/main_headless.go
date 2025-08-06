//go:build headless
// +build headless

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"KNIRVCHAIN_GO_Verifyer/starter"

	"github.com/joho/godotenv"
)

func main() {
	// Set up panic recovery
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Application panic: %v\n", r)
			os.Exit(1)
		}
	}()

	// Check if installation is complete
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Could not load .env file")
	}

	installComplete := os.Getenv("INSTALL_COMPLETE")
	if installComplete != "true" {
		// Run headless installation
		fmt.Println("=== Running Headless Installation ===")
		starter.Install()
		fmt.Println("Installation complete. Please restart the application.")
		return
	}

	// Define command-line flags
	testnetFlag := flag.Bool("testnet", false, "Run in testnet mode")
	localNetworkFlag := flag.Bool("local-network", false, "Enable local network mode")
	mockNRNFlag := flag.Bool("mock-nrn", false, "Enable mock NRN minting")
	portFlag := flag.Int("port", 5001, "Port to run on")
	minersAddressFlag := flag.String("miners_address", "KNIRVROUTER_Miner", "Miners address")
	helpFlag := flag.Bool("help", false, "Show help")

	flag.Parse()

	if *helpFlag {
		fmt.Println("KNIRVROUTER - Headless Mode")
		fmt.Println("Usage:")
		flag.PrintDefaults()
		return
	}

	// Set environment variables based on flags
	if *testnetFlag {
		os.Setenv("TESTNET_MODE", "true")
		fmt.Println("🧪 Testnet mode enabled")
	}
	if *localNetworkFlag {
		os.Setenv("LOCAL_NETWORK_MODE", "true")
		fmt.Println("🌐 Local network mode enabled")
	}
	if *mockNRNFlag {
		os.Setenv("MOCK_NRN_MINTING", "true")
		fmt.Println("💰 Mock NRN minting enabled")
	}

	// Set port
	os.Setenv("PORT", fmt.Sprintf("%d", *portFlag))
	os.Setenv("MINERS_ADDRESS", *minersAddressFlag)

	// Always start in chain mode for headless
	fmt.Printf("🚀 Starting KNIRVROUTER in headless mode on port %d\n", *portFlag)

	// Start the blockchain node
	starter.StartRootBlockchain(uint64(*portFlag), *minersAddressFlag)
}
