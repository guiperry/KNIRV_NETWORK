package utils

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	// AppName is the name of the application used for directory naming
	AppName = "KNIRV-Engine"
)

// GetAppDataDir returns the OS-specific application data directory
// Linux: ~/.config/KNIRV-Engine
// Windows: %APPDATA%\KNIRV-Engine
// macOS: ~/Library/Application Support/KNIRV-Engine
func GetAppDataDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		// Use APPDATA environment variable
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", &AppDataError{Op: "GetAppDataDir", OS: "windows", Err: "APPDATA environment variable not set"}
		}
		return filepath.Join(appData, AppName), nil
	case "darwin":
		// Use ~/Library/Application Support/AppName
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", &AppDataError{Op: "GetAppDataDir", OS: "darwin", Err: err.Error()}
		}
		return filepath.Join(homeDir, "Library", "Application Support", AppName), nil
	default:
		// Linux and other Unix-like systems: use XDG config directory
		configDir, err := os.UserConfigDir()
		if err != nil {
			return "", &AppDataError{Op: "GetAppDataDir", OS: runtime.GOOS, Err: err.Error()}
		}
		return filepath.Join(configDir, AppName), nil
	}
}

// GetDatabaseDir returns the directory for storing database files
func GetDatabaseDir() (string, error) {
	appDataDir, err := GetAppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDataDir, "data"), nil
}

// GetConfigDir returns the directory for storing configuration files
func GetConfigDir() (string, error) {
	appDataDir, err := GetAppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDataDir, "config"), nil
}

// GetAgentsDBPath returns the path to the centralized agents database
func GetAgentsDBPath() (string, error) {
	dbDir, err := GetDatabaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dbDir, "agents.db"), nil
}

// GetMCPDir returns the base directory for all MCP-related data
func GetMCPDir() (string, error) {
	appDataDir, err := GetAppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDataDir, "mcp"), nil
}

// GetMCPConfigDir returns the directory for MCP configurations
func GetMCPConfigDir() (string, error) {
	mcpDir, err := GetMCPDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(mcpDir, "config"), nil
}

// GetMCPServersDir returns the directory for MCP server installations
func GetMCPServersDir() (string, error) {
	mcpDir, err := GetMCPDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(mcpDir, "servers"), nil
}

// GetPluginsDir returns the directory for storing plugins
func GetPluginsDir() (string, error) {
	appDataDir, err := GetAppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDataDir, "plugins"), nil
}

// GetCacheDir returns the cache directory inside the application's data root.
// Keeping cache alongside all other engine state makes the application's
// persistent footprint portable and keeps Linux installs in one App Data tree.
func GetCacheDir() (string, error) {
	appDataDir, err := GetAppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDataDir, "cache"), nil
}

// GetTempWorkspaceDir returns a temporary workspace directory for agent operations
func GetTempWorkspaceDir() (string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "workspace"), nil
}

// GetToolsDir returns the managed verification-tools directory. The Linux
// installer populates this from the published tools.tar.gz archive.
func GetToolsDir() (string, error) {
	databaseDir, err := GetDatabaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(databaseDir, "bin"), nil
}

// GetLogsDir returns the directory for storing application logs
func GetLogsDir() (string, error) {
	appDataDir, err := GetAppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDataDir, "logs"), nil
}

// GetSandboxScriptsDir returns the directory that project-owned sandbox
// helper scripts (e.g. the Frida RPC bridge) are extracted to at startup.
// These are embedded in the compiled binary via go:embed rather than shipped
// as loose files next to it, so this directory is populated at runtime, not
// by the build. Third-party sandbox binaries (bwrap, frida-server, etc.) are
// intentionally NOT extracted here — see sandboxToolsDir in api/sandbox_tools.go.
func GetSandboxScriptsDir() (string, error) {
	appDataDir, err := GetAppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDataDir, "sandbox-scripts"), nil
}

// GetPluginDataDir returns the directory for storing plugin-specific data
func GetPluginDataDir() (string, error) {
	pluginsDir, err := GetPluginsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(pluginsDir, "data"), nil
}

// GetMCPDataDir returns the directory for MCP-specific data
func GetMCPDataDir() (string, error) {
	mcpDir, err := GetMCPDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(mcpDir, "data"), nil
}

// GetMCPLogsDir returns the directory for MCP server logs
func GetMCPLogsDir() (string, error) {
	mcpDir, err := GetMCPDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(mcpDir, "logs"), nil
}

// GetMCPMonitoringDir returns the directory for MCP monitoring data
func GetMCPMonitoringDir() (string, error) {
	mcpDir, err := GetMCPDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(mcpDir, "monitoring"), nil
}

// GetQuarantineDir returns the directory for quarantined files
func GetQuarantineDir() (string, error) {
	appDataDir, err := GetAppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDataDir, "quarantine"), nil
}

// GetBackupDir returns the directory for storing backups
func GetBackupDir() (string, error) {
	appDataDir, err := GetAppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDataDir, "backups"), nil
}

// GetSecurityDir returns the directory for storing security-related files (keys, certificates)
func GetSecurityDir() (string, error) {
	appDataDir, err := GetAppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDataDir, "security"), nil
}

// EnsureDir creates the directory if it doesn't exist
func EnsureDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &AppDataError{Op: "EnsureDir", Path: dir, Err: err.Error()}
	}
	return nil
}

// EnsureAppDataDirs creates all necessary application data directories
func EnsureAppDataDirs() error {
	dirs := []func() (string, error){
		GetAppDataDir,
		GetDatabaseDir,
		GetConfigDir,
		GetMCPDir,
		GetMCPConfigDir,
		GetMCPServersDir,
		GetMCPDataDir,
		GetMCPLogsDir,
		GetMCPMonitoringDir,
		GetSandboxScriptsDir,
		GetPluginsDir,
		GetPluginDataDir,
		GetCacheDir,
		GetTempWorkspaceDir,
		GetToolsDir,
		GetLogsDir,
		GetQuarantineDir,
		GetBackupDir,
		GetSecurityDir,
	}

	for _, getDirFunc := range dirs {
		dir, err := getDirFunc()
		if err != nil {
			return err
		}
		if err := EnsureDir(dir); err != nil {
			return err
		}
	}

	return nil
}

// AppDataError represents an error related to application data directory operations
type AppDataError struct {
	Op   string // operation that failed
	OS   string // operating system
	Path string // path involved (if any)
	Err  string // error description
}

func (e *AppDataError) Error() string {
	if e.Path != "" {
		return "appdata " + e.Op + " " + e.Path + ": " + e.Err
	}
	if e.OS != "" {
		return "appdata " + e.Op + " on " + e.OS + ": " + e.Err
	}
	return "appdata " + e.Op + ": " + e.Err
}
