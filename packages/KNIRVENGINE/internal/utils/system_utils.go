package utils

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// SystemInfo represents system information
type SystemInfo struct {
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
	Version      string `json:"version"`
}

// ProcessInfo represents information about a running process
type ProcessInfo struct {
	PID     int    `json:"pid"`
	Name    string `json:"name"`
	Command string `json:"command"`
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Available bool   `json:"available"`
}

// MountedDrive represents a mounted drive/filesystem
type MountedDrive struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Type       string `json:"type"`
	Size       string `json:"size"`
	Available  string `json:"available"`
	Accessible bool   `json:"accessible"`
}

// IsProcessRunning checks if a process is running with enhanced cross-platform detection
func IsProcessRunning(processName string) (bool, error) {
	switch runtime.GOOS {
	case "windows":
		return isProcessRunningWindows(processName)
	case "darwin":
		return isProcessRunningMacOS(processName)
	case "linux":
		return isProcessRunningLinux(processName)
	default:
		return false, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// isProcessRunningWindows checks for running processes on Windows
func isProcessRunningWindows(processName string) (bool, error) {
	cmd := exec.Command("tasklist", "/NH", "/FO", "CSV")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(output), "\n")
	lowerProcessName := strings.ToLower(processName)

	for _, line := range lines {
		if strings.Contains(line, ",") {
			parts := strings.Split(line, ",")
			if len(parts) > 0 {
				imageName := strings.Trim(parts[0], "\"")
				cleanImageName := strings.ToLower(imageName)
				if cleanImageName == lowerProcessName+".exe" ||
					cleanImageName == lowerProcessName ||
					strings.Contains(cleanImageName, lowerProcessName) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// isProcessRunningMacOS checks for running processes on macOS
func isProcessRunningMacOS(processName string) (bool, error) {
	// First try exact match with pgrep
	cmd := exec.Command("pgrep", "-x", processName)
	output, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) != "" {
		return true, nil
	}

	// Then check with ps for broader matching
	cmd = exec.Command("ps", "-A", "-o", "comm,command")
	output, err = cmd.Output()
	if err != nil {
		return false, err
	}

	processes := strings.Split(string(output), "\n")
	lowerProcessName := strings.ToLower(processName)

	for _, proc := range processes {
		lowerProc := strings.ToLower(strings.TrimSpace(proc))
		if strings.Contains(lowerProc, lowerProcessName) ||
			strings.Contains(lowerProc, lowerProcessName+".app") {
			return true, nil
		}
	}
	return false, nil
}

// isProcessRunningLinux checks for running processes on Linux
func isProcessRunningLinux(processName string) (bool, error) {
	cmd := exec.Command("pgrep", "-f", processName)
	output, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) != "" {
		return true, nil
	}

	// Fallback to ps
	cmd = exec.Command("ps", "-A", "-o", "comm,command")
	output, err = cmd.Output()
	if err != nil {
		return false, err
	}

	processes := strings.Split(string(output), "\n")
	lowerProcessName := strings.ToLower(processName)

	for _, proc := range processes {
		lowerProc := strings.ToLower(strings.TrimSpace(proc))
		if strings.Contains(lowerProc, lowerProcessName) {
			return true, nil
		}
	}
	return false, nil
}

// IsApplicationInstalled checks if an application is installed
func IsApplicationInstalled(appName string) (bool, error) {
	switch runtime.GOOS {
	case "windows":
		return isApplicationInstalledWindows(appName)
	case "darwin":
		return isApplicationInstalledMacOS(appName)
	case "linux":
		return isApplicationInstalledLinux(appName)
	default:
		return false, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// isApplicationInstalledWindows checks if an application is installed on Windows
func isApplicationInstalledWindows(appName string) (bool, error) {
	// Check common installation paths
	commonPaths := []string{
		"C:\\Program Files\\",
		"C:\\Program Files (x86)\\",
		os.Getenv("LOCALAPPDATA") + "\\Programs\\",
	}

	for _, basePath := range commonPaths {
		if basePath == "" {
			continue
		}
		appPath := basePath + appName
		if _, err := os.Stat(appPath); err == nil {
			return true, nil
		}
	}

	// Check if it's in PATH
	cmd := exec.Command("where", appName)
	err := cmd.Run()
	return err == nil, nil
}

// isApplicationInstalledMacOS checks if an application is installed on macOS
func isApplicationInstalledMacOS(appName string) (bool, error) {
	// Check /Applications directory
	appPath := "/Applications/" + appName + ".app"
	if _, err := os.Stat(appPath); err == nil {
		return true, nil
	}

	// Check user Applications directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userAppPath := homeDir + "/Applications/" + appName + ".app"
		if _, err := os.Stat(userAppPath); err == nil {
			return true, nil
		}
	}

	// Check if it's in PATH
	cmd := exec.Command("which", appName)
	err = cmd.Run()
	return err == nil, nil
}

// isApplicationInstalledLinux checks if an application is installed on Linux
func isApplicationInstalledLinux(appName string) (bool, error) {
	// Check if it's in PATH
	cmd := exec.Command("which", appName)
	err := cmd.Run()
	if err == nil {
		return true, nil
	}

	// Check common installation directories
	commonPaths := []string{
		"/usr/bin/",
		"/usr/local/bin/",
		"/opt/",
		"/snap/bin/",
	}

	for _, basePath := range commonPaths {
		appPath := basePath + appName
		if _, err := os.Stat(appPath); err == nil {
			return true, nil
		}
	}

	return false, nil
}

// GetPlatformInfo returns detailed platform information
func GetPlatformInfo() SystemInfo {
	return SystemInfo{
		Platform:     runtime.GOOS,
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
		Version:      runtime.Version(),
	}
}

// IsServiceRunning checks if a system service is running
func IsServiceRunning(serviceName string) (bool, error) {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("sc", "query", serviceName)
		output, err := cmd.Output()
		if err != nil {
			return false, err
		}
		return strings.Contains(string(output), "RUNNING"), nil
	case "darwin":
		cmd := exec.Command("launchctl", "list", serviceName)
		err := cmd.Run()
		return err == nil, nil
	case "linux":
		cmd := exec.Command("systemctl", "is-active", serviceName)
		output, err := cmd.Output()
		if err != nil {
			return false, err
		}
		return strings.TrimSpace(string(output)) == "active", nil
	default:
		return false, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// IsDatabaseRunning checks if a database service is running
func IsDatabaseRunning(dbType string) (bool, error) {
	switch strings.ToLower(dbType) {
	case "mysql":
		return IsServiceRunning("mysql")
	case "postgresql", "postgres":
		return IsServiceRunning("postgresql")
	case "mongodb", "mongo":
		return IsServiceRunning("mongodb")
	case "redis":
		return IsServiceRunning("redis")
	default:
		return false, fmt.Errorf("unsupported database type: %s", dbType)
	}
}
