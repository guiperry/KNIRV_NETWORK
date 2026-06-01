package tests

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"
)

func getExtractedBinDir() string {
	usr, err := user.Current()
	if err != nil {
		return ""
	}
	return filepath.Join(usr.HomeDir, ".local", "share", "knirvserver", "bin")
}

func killAllServices() {
	scriptPath := filepath.Join("..", "..", "scripts", "kill-all-services.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		scriptPath = "/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/scripts/kill-all-services.sh"
	}

	cmd := exec.Command("bash", scriptPath, "--force")
	cmd.Dir = filepath.Dir(scriptPath)
	_ = cmd.Run()
	time.Sleep(2 * time.Second)
}

func getBackendPathFromExtracted() string {
	extractedBinDir := getExtractedBinDir()
	backendPath := filepath.Join(extractedBinDir, "backend_server")
	if _, err := os.Stat(backendPath); err == nil {
		return backendPath
	}

	binDir := filepath.Join("..", "..", "bin")
	return filepath.Join(binDir, "backend_server")
}
