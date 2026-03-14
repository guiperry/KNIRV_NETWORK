package installation

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"KNIRVCHAIN/config"
	"KNIRVCHAIN/internal/uri"
)

func Uninstall() {
	// Try to determine role from config location
	role := determineInstalledRole()

	fmt.Printf("=== KNIRVCHAIN %s Node Uninstallation ===\n", role.String())
	fmt.Println("This uninstaller will:")
	fmt.Println("1. Stop any running node processes")
	fmt.Println("2. Remove URI handler for knirv:// protocol")
	fmt.Println("3. Clean up configuration and wallet files")
	fmt.Println("4. Remove system service")
	fmt.Println()

	// Step 1: Stop running processes
	fmt.Println("Stopping any running node processes...")
	stopRunningProcesses(role)

	// Step 2: Unregister URI handlers
	fmt.Println("Removing URI handlers...")
	err := UnregisterURIHandlers()
	if err != nil {
		log.Printf("Warning: Failed to unregister URI handlers: %v", err)
		fmt.Println("You may need to run this uninstaller with administrator/root privileges.")
	} else {
		fmt.Println("URI handlers removed successfully.")
	}

	// Step 3: Clean up configuration and wallets
	fmt.Println("Cleaning up configuration and wallet files...")
	cleanupConfiguration(role)

	// Step 4: Remove system service
	fmt.Printf("Removing %s service...\n", getServiceName(role))
	err = removeSystemService(role)
	if err != nil {
		log.Printf("Warning: Failed to remove system service: %v", err)
		fmt.Println("You may need to remove the service manually.")
	} else {
		fmt.Println("Service removed successfully.")
	}

	fmt.Println("\n=== Uninstallation Complete ===")
	fmt.Printf("KNIRVCHAIN %s Node has been uninstalled.\n", role.String())
}

func stopRunningProcesses(role config.Role) {
	// Implementation will vary by OS
	serviceName := getServiceName(role)
	switch runtime.GOOS {
	case "windows":
		// Windows implementation
		cmd := exec.Command("taskkill", "/F", "/IM", serviceName+".exe")
		cmd.Run()
	default:
		// Unix-like implementation
		cmd := exec.Command("pkill", "-f", serviceName)
		cmd.Run()
	}
}

func UnregisterURIHandlers() error {
	// Define the URI scheme to unregister
	schemes := []uri.URIScheme{
		{
			Name:        "agent",
			Description: "KNIRVCHAIN Decentralized Protocol",
		},
	}

	// Unregister the URI scheme based on the operating system
	return unregisterURISchemes(schemes)
}

func cleanupConfiguration(role config.Role) {
	// Remove config file
	configPath, err := config.GetConfigPath(role)
	if err == nil {
		if _, err := os.Stat(configPath); err == nil {
			err := os.Remove(configPath)
			if err != nil {
				log.Printf("Warning: Failed to remove config file: %v", err)
			} else {
				fmt.Printf("Removed config file (%s)\n", configPath)
			}
		}
	}

	// Remove wallet files if they exist
	if role == config.RoleBootnode || role == config.RolePeer {
		walletPath, err := config.GetPeerWalletPath(role)
		if err == nil {
			if _, err := os.Stat(walletPath); err == nil {
				os.Remove(walletPath)
				fmt.Printf("Removed wallet file (%s)\n", walletPath)
			}
		}
	}

	if role == config.RoleBootnode || role == config.Root {
		masterPath, err := config.GetMasterWalletPath(role)
		if err == nil {
			if _, err := os.Stat(masterPath); err == nil {
				os.Remove(masterPath)
				fmt.Printf("Removed master wallet file (%s)\n", masterPath)
			}
		}
	}
}

// unregisterURISchemes handles OS-specific URI scheme unregistration
func unregisterURISchemes(schemes []uri.URIScheme) error {
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

func unregisterURISchemesWindows(schemes []uri.URIScheme) error {
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

func unregisterURISchemesMacOS(schemes []uri.URIScheme) error {
	// macOS implementation would modify Info.plist or use defaults commands
	// This is a placeholder
	for _, scheme := range schemes {
		fmt.Printf("Would unregister %s protocol on macOS\n", scheme.Name)
	}
	return nil
}

func unregisterURISchemesLinux(schemes []uri.URIScheme) error {
	// Linux implementation would modify .desktop files or mime types
	// This is a placeholder
	for _, scheme := range schemes {
		fmt.Printf("Would unregister %s protocol on Linux\n", scheme.Name)
	}
	return nil
}

// determineInstalledRole tries to determine the installed role by checking config location
func determineInstalledRole() config.Role {
	// First check if config exists in app data dir (non-root roles)
	appDataPath, err := config.GetAppDataDir()
	if err == nil {
		configPath := filepath.Join(appDataPath, "config.json")
		if _, err := os.Stat(configPath); err == nil {
			// Config exists in app data dir - likely non-root role
			cfg, _, err := config.LoadConfig(configPath, config.RoleClient)
			if err == nil {
				if cfg.IsRoot {
					return config.Root
				} else if cfg.IsBootnode {
					return config.RoleBootnode
				} else if cfg.ClientOnly {
					return config.RoleClient
				}
				return config.RolePeer
			}
		}
	}

	// Check executable directory for root config
	exeDir, err := os.Getwd()
	if err == nil {
		configPath := filepath.Join(exeDir, "config.json")
		if _, err := os.Stat(configPath); err == nil {
			return config.Root
		}
	}

	// Default to client role if we can't determine
	return config.RoleClient
}

// getServiceName returns the service name based on role
func getServiceName(role config.Role) string {
	return fmt.Sprintf("KNIRVCHAIN-%s", strings.ToLower(role.String()))
}

// removeSystemService removes the system service based on role
func removeSystemService(role config.Role) error {
	serviceName := getServiceName(role)
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("sc", "delete", serviceName)
		return cmd.Run()
	case "linux":
		cmd := exec.Command("systemctl", "disable", "--now", serviceName)
		return cmd.Run()
	case "darwin":
		cmd := exec.Command("launchctl", "unload", "-w", fmt.Sprintf("/Library/LaunchDaemons/%s.plist", serviceName))
		return cmd.Run()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
