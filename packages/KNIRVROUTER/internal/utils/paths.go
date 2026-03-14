package utils 

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
)

const AppName = "KNIRVROUTER" // Or your desired app name

// GetDefaultDataDir returns the default base directory for application data.
// It uses OS-specific standard locations.
func GetDefaultDataDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to current directory if user config dir is unavailable
		log.Printf("Warning: Could not find user config dir: %v. Using current directory.", err)
		// Use a subdirectory within the current directory as a fallback
		fallbackPath, _ := filepath.Abs(filepath.Join(".", "appdata", AppName, "data"))
		log.Printf("Using fallback data directory: %s", fallbackPath)
		return fallbackPath
	}
	// Construct path like ~/.config/KNIRVROUTER/data or %APPDATA%/KNIRVROUTER/data
	defaultPath := filepath.Join(configDir, AppName, "data")
	log.Printf("Using default data directory: %s", defaultPath)
	return defaultPath
}

// GetDefaultRootDBPath returns the default database path for the root node.
func GetDefaultRootDBPath() string {
	return filepath.Join(GetDefaultDataDir(), "root", "knirvdb")
}

// GetPeerDBPath returns the database path for a specific peer port.
func GetPeerDBPath(port int) string {
	portStr := strconv.Itoa(port)
	return filepath.Join(GetDefaultDataDir(), portStr, "knirvdb")
}



