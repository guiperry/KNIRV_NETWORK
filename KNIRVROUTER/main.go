package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"KNIRVCHAIN_GO_Verifyer/gui"
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
		// First try GUI installation
		fmt.Println("=== Starting GUI Installation ===")
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("GUI installer failed: %v\n", r)
					fmt.Println("GUI installer failed, falling back to console...")
					fmt.Println("=== Running Console Installation ===")
					starter.Install()
					fmt.Println("Installation complete. Please restart the application.")
				}
			}()
			gui.StartInstallGUI()
		}()
		return
	}

	// Define command-line flags
	chainFlag := flag.Bool("chain", false, "Start blockchain node instead of GUI")
	walletFlag := flag.Bool("wallet", false, "Start wallet server instead of GUI")
	installCompleteFlag := flag.Bool("install-complete", false, "Flag indicating installation just completed")

	// Parse flags
	flag.Parse()

	// Handle install complete flag
	if *installCompleteFlag {
		// Set environment variable to prevent re-installation
		os.Setenv("INSTALL_COMPLETE", "true")
		// Write to .env file for persistence
		f, err := os.OpenFile(".env", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			defer f.Close()
			f.WriteString("INSTALL_COMPLETE=true\n")
		}
	}

	// Check if chain or wallet flags are set, or if subcommands are used
	if *chainFlag || *walletFlag || (len(os.Args) > 1 && (os.Args[1] == "chain" || os.Args[1] == "wallet")) {
		// If chain or wallet flags are set, proceed with normal startup
		fmt.Println("Starting KNIRVCHAIN in command-line mode...")
		starter.StartCommandLine() // Call the exported function from the starter package
		return
	}

	// Start the desktop GUI by default
	fmt.Println("Starting KNIRVCHAIN Desktop GUI...")
	// Use the updated GUI implementation
	gui.StartFyneGUI()
}
