package main

import (
	"KNIRVROOT/config"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LaunchDeveloperPortal starts the Developer Portal Node.js service
// This function should be called only from the Root node
func LaunchDeveloperPortal(cfg *config.Config) error {
	if !cfg.IsRoot {
		return fmt.Errorf("developer portal can only be launched from a Root node")
	}

	if !cfg.NodeJSServices.Enabled || !cfg.NodeJSServices.DeveloperPortal.Enabled {
		log.Println("Developer Portal is disabled in configuration")
		return nil
	}

	scriptPath := cfg.NodeJSServices.DeveloperPortal.ScriptPath
	if scriptPath == "" {
		scriptPath = "agent-developer-portal/server.js"
	}

	// Resolve the script path relative to the executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	execDir := filepath.Dir(execPath)
	scriptFullPath := filepath.Join(execDir, scriptPath)

	// Check if the script exists
	if _, err := os.Stat(scriptFullPath); os.IsNotExist(err) {
		return fmt.Errorf("developer portal script not found at %s", scriptFullPath)
	}

	// Prepare environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("HTTP_API_PORT=%d", cfg.NodeJSServices.DeveloperPortal.HTTPPort))
	env = append(env, fmt.Sprintf("API_KEY=%s", cfg.NodeJSServices.DeveloperPortal.APIKey))
	env = append(env, fmt.Sprintf("CHAIN_ID=%s", cfg.ChainID))
	env = append(env, fmt.Sprintf("NODE_ENV=%s", "production"))

	// Start the Node.js process
	cmd := exec.Command("node", scriptFullPath)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the process in the background
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Developer Portal: %v", err)
	}

	log.Printf("Developer Portal started with PID %d on port %d", cmd.Process.Pid, cfg.NodeJSServices.DeveloperPortal.HTTPPort)

	// Don't wait for the process to complete
	go func() {
		if err := cmd.Wait(); err != nil {
			if strings.Contains(err.Error(), "signal: killed") {
				log.Println("Developer Portal process was terminated")
			} else {
				log.Printf("Developer Portal process exited with error: %v", err)
			}
		} else {
			log.Println("Developer Portal process exited normally")
		}
	}()

	return nil
}
