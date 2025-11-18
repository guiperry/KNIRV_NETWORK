package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"time"
)

func PluginServer() {
	// Start the plugin-server as a separate process
	// Pass the port number as a command-line argument
	port := "8081"
	cmd := exec.Command("./plugin-server", "--port", port)

	// You can capture its output for logging if needed
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()

	err := cmd.Start()
	if err != nil {
		log.Fatalf("Failed to start plugin-server: %v", err)
	}

	// Ensure the child process is killed when the main app exits
	defer cmd.Process.Kill()

	log.Println("Plugin server started on port", port)

	// Give the server a moment to start up
	time.Sleep(1 * time.Second)

	// Now, interact with its HTTP endpoint
	serverURL := fmt.Sprintf("http://localhost:%s", port)
	resp, err := http.Get(serverURL + "/list")
	if err != nil {
		log.Fatalf("Failed to call /list endpoint: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	fmt.Printf("Response from plugin-server: %s\n", string(body))
}
