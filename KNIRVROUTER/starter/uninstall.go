package starter

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
)

func Uninstall() {
	fmt.Println("=== KNIRVROUTER Verifier Node Uninstallation ===")
	fmt.Println("This uninstaller will:")
	fmt.Println("1. Stop any running verifier node processes")
	fmt.Println("2. Remove URI handler for knirv:// protocol")
	fmt.Println("3. Clean up configuration files")
	fmt.Println("4. Remove system service (if installed)")
	fmt.Println()

	// Step 1: Stop running processes
	fmt.Println("Stopping any running verifier node processes...")
	stopRunningProcesses()

	// Step 2: Unregister URI handlers
	fmt.Println("Removing URI handlers...")
	err := UnregisterURIHandlers()
	if err != nil {
		log.Printf("Warning: Failed to unregister URI handlers: %v", err)
		fmt.Println("You may need to run this uninstaller with administrator/root privileges.")
	} else {
		fmt.Println("URI handlers removed successfully.")
	}

	// Step 3: Clean up configuration
	fmt.Println("Cleaning up configuration...")
	cleanupConfiguration()

	// Step 4: Remove system service (placeholder)
	fmt.Println("Removing system service (Placeholder)...")

	fmt.Println("\n=== Uninstallation Complete ===")
	fmt.Println("KNIRVROUTER Verifier Node has been uninstalled.")
}

func stopRunningProcesses() {
	// Implementation will vary by OS
	switch runtime.GOOS {
	case "windows":
		// Windows implementation
		cmd := exec.Command("taskkill", "/F", "/IM", "KNIRVROUTER_GO_Verifyer.exe")
		cmd.Run()
	default:
		// Unix-like implementation
		cmd := exec.Command("pkill", "-f", "KNIRVROUTER_GO_Verifyer")
		cmd.Run()
	}
}

func UnregisterURIHandlers() error {
	// Define the URI scheme to unregister
	schemes := []URIScheme{
		{
			Name:        "knirv",
			Description: "KNIRVROUTER Decentralized Protocol",
		},
	}

	// Unregister the URI scheme based on the operating system
	return unregisterURISchemes(schemes)
}

func cleanupConfiguration() {
	envPath := ".env"

	// Check if .env exists and remove it
	if _, err := os.Stat(envPath); err == nil {
		err := os.Remove(envPath)
		if err != nil {
			log.Printf("Warning: Failed to remove .env file: %v", err)
		} else {
			fmt.Println("Removed configuration file (.env)")
		}
	}

	// Check for other configuration files to clean up
	// (Add any additional config files that need removal here)
}

// unregisterURISchemes handles OS-specific URI scheme unregistration
func unregisterURISchemes(schemes []URIScheme) error {
	switch runtime.GOOS {
	case "windows":
		return unregisterURISchemesWindows(schemes)
	case "darwin":
		return unregisterURISchemesMacOS(schemes)
	case "linux":
		return unregisterURISchemesLinux(schemes)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func unregisterURISchemesWindows(schemes []URIScheme) error {
	// Windows implementation would use reg delete commands
	// This is a simplified example - actual implementation would need more details
	for _, scheme := range schemes {
		cmd := exec.Command("reg", "delete", fmt.Sprintf("HKCU\\Software\\Classes\\%s", scheme.Name), "/f")
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to unregister %s protocol: %w", scheme.Name, err)
		}
	}
	return nil
}

func unregisterURISchemesMacOS(schemes []URIScheme) error {
	// macOS implementation would modify Info.plist or use defaults commands
	// This is a placeholder
	for _, scheme := range schemes {
		fmt.Printf("Would unregister %s protocol on macOS\n", scheme.Name)
	}
	return nil
}

func unregisterURISchemesLinux(schemes []URIScheme) error {
	// Linux implementation would modify .desktop files or mime types
	// This is a placeholder
	for _, scheme := range schemes {
		fmt.Printf("Would unregister %s protocol on Linux\n", scheme.Name)
	}
	return nil
}
