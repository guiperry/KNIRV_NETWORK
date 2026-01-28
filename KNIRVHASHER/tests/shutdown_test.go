package tests

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func testShutdown() bool {
	fmt.Println("=== Tiny-LLM Shutdown Test ===")

	// Cleanup any existing running processes
	exec.Command("pkill", "-9", "-f", "llama-server").Run()
	time.Sleep(1 * time.Second)

	// Check initial server state
	if serverRunning() {
		fmt.Println("❌ Server already running before test")
		return false
	}
	fmt.Println("✅ Server is not running initially")

	// Start the application
	fmt.Println("\nStarting application...")
	cmd := exec.Command("./tinyllm")
	err := cmd.Start()
	if err != nil {
		fmt.Printf("❌ Error starting application: %v\n", err)
		return false
	}

	// Give server time to start
	fmt.Println("Waiting for server to start...")
	for i := 0; i < 20; i++ {
		if serverRunning() {
			fmt.Println("✅ Server started successfully")
			break
		}
		time.Sleep(1 * time.Second)
		if i == 19 {
			fmt.Println("❌ Server failed to start within 20 seconds")
			cmd.Process.Kill()
			return false
		}
	}

	// Test shutdown by sending SIGINT
	fmt.Println("\nTesting shutdown with SIGINT...")
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		fmt.Printf("❌ Error sending SIGINT: %v\n", err)
		cmd.Process.Kill()
		return false
	}

	// Give server time to shutdown
	time.Sleep(3 * time.Second)

	// Check if server is still running
	if serverRunning() {
		fmt.Println("❌ Server failed to shut down")
		// Force kill the server
		exec.Command("pkill", "-9", "-f", "llama-server").Run()
		time.Sleep(1 * time.Second)
		if serverRunning() {
			fmt.Println("❌ Failed to force kill server")
		} else {
			fmt.Println("⚠️ Server force killed")
		}
		return false
	} else {
		fmt.Println("✅ Server shut down successfully")
		return true
	}
}

func serverRunning() bool {
	// Use curl to check if server responds
	cmd := exec.Command("curl", "-s", "http://localhost:8000/health")
	err := cmd.Run()
	return err == nil
}
