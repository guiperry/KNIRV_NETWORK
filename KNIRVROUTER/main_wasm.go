//go:build wasmloader
// +build wasmloader

// WASM-enabled main for KNIRVROUTER (GUI disabled)
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"KNIRVROUTER_GO_Verifyer/starter"

	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("🚀 KNIRVROUTER with Revolutionary WASM Support")
	fmt.Println("ℹ️  GUI disabled in WASM build")

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: Could not load .env file")
	}

	installComplete := os.Getenv("INSTALL_COMPLETE")
	if installComplete != "true" {
		// Console installation only for WASM builds
		fmt.Println("=== Running Console Installation ===")
		starter.Install()
		fmt.Println("Installation complete. Please restart the application.")
		return
	}

	// Define command-line flags
	chainFlag := flag.Bool("chain", false, "Start blockchain node")
	walletFlag := flag.Bool("wallet", false, "Start wallet server")
	installCompleteFlag := flag.Bool("install-complete", false, "Flag indicating installation just completed")
	testnetFlag := flag.Bool("testnet", false, "Run in testnet mode with local network")
	localNetworkFlag := flag.Bool("local-network", false, "Enable local network mode")
	mockNRNFlag := flag.Bool("mock-nrn", false, "Enable mock NRN minting")

	// Parse command-line flags
	flag.Parse()

	// Set environment variables based on flags
	if *installCompleteFlag {
		os.Setenv("INSTALL_COMPLETE", "true")
	}

	if *testnetFlag {
		os.Setenv("TESTNET_MODE", "true")
		fmt.Println("🧪 Testnet mode enabled")
	}

	if *localNetworkFlag {
		os.Setenv("LOCAL_NETWORK", "true")
		fmt.Println("🏠 Local network mode enabled")
	}

	if *mockNRNFlag {
		os.Setenv("MOCK_NRN", "true")
		fmt.Println("🪙 Mock NRN minting enabled")
	}

	// Handle different startup modes
	if *chainFlag {
		fmt.Println("Starting KNIRVROUTER blockchain node with WASM support...")
		starter.StartCommandLine()
		return
	}

	if *walletFlag {
		fmt.Println("Starting KNIRVROUTER wallet server...")
		starter.StartCommandLine()
		return
	}

	// WASM builds default to command-line mode
	fmt.Println("Starting KNIRVROUTER in command-line mode with WASM support...")
	fmt.Println("🔗 WASM endpoints will be available when KNIRV_ENABLE_WASM=true")
	starter.StartCommandLine()
}
