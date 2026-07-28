package app

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetAppDataDir returns the OS-specific application data directory.
// Privileged launches use a system location so sudo does not split data
// between /root and the invoking user's home directory.
func GetAppDataDir() (string, error) {
	var basePath string

	switch runtime.GOOS {
	case "windows":
		// %APPDATA% on Windows
		if appData := os.Getenv("APPDATA"); appData != "" {
			basePath = appData
		} else {
			// Fallback to user profile
			if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
				basePath = filepath.Join(userProfile, "AppData", "Roaming")
			} else {
				// Ultimate fallback
				basePath = filepath.Join(os.TempDir(), "knirvserver", "data")
			}
		}
	case "darwin":
		// ~/Library/Application Support on macOS
		if home := os.Getenv("HOME"); home != "" {
			basePath = filepath.Join(home, "Library", "Application Support")
		} else {
			basePath = os.TempDir()
		}
	default: // linux, unix, etc.
		// KNIRV_APP_DATA_DIR takes highest precedence
		if explicit := os.Getenv("KNIRV_APP_DATA_DIR"); explicit != "" {
			if err := os.MkdirAll(explicit, 0755); err == nil {
				return filepath.Join(explicit, "knirvhasher", "data"), nil
			}
		}
		// System location
		if err := os.MkdirAll("/var/lib/knirvserver/knirvhasher/data", 0755); err == nil {
			return "/var/lib/knirvserver/knirvhasher/data", nil
		}
		// ~/.local/share on XDG Base Directory Specification
		if home := os.Getenv("HOME"); home != "" {
			if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
				basePath = xdgData
			} else {
				basePath = filepath.Join(home, ".local", "share")
			}
		} else {
			basePath = os.TempDir()
		}
	}

	appDir := filepath.Join(basePath, "knirvserver", "knirvhasher", "data")

	// Ensure the directory exists
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", err
	}

	return appDir, nil
}

// SetupDataDirectories creates all necessary data directories within the app data directory
func SetupDataDirectories(appDataDir string) (map[string]string, error) {
	dirs := map[string]string{
		"checkpoints": filepath.Join(appDataDir, "checkpoints"),
		"papers":      filepath.Join(appDataDir, "papers"),
		"json":        filepath.Join(appDataDir, "json"),
		"documents":   filepath.Join(appDataDir, "documents"),
		"temp":        filepath.Join(appDataDir, "temp"),
	}

	// Create all directories
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	return dirs, nil
}
