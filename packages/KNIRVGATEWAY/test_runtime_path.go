package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/embedded"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/runtime"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	rt, err := runtime.NewRuntime(logger, embedded.WebGUIFS, embedded.NetworkWebsiteFS, nil)
	if err != nil {
		fmt.Printf("Error creating runtime: %v\n", err)
		os.Exit(1)
	}

	homeDir, _ := os.UserHomeDir()
	expectedPath := filepath.Join(homeDir, ".local", "share", "knirvserver", "knirvgateway", "runtime")

	fmt.Printf("Expected path: %s\n", expectedPath)
	fmt.Printf("Actual path: %s\n", rt.BaseDir)

	if rt.BaseDir == expectedPath {
		fmt.Println("✓ Path matches expected location")
	} else {
		fmt.Println("✗ Path does not match expected location")
		os.Exit(1)
	}
}
