package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

//go:embed bin/*
var binFS embed.FS

//go:embed config/*
var configFS embed.FS

//go:embed data/*
var dataFS embed.FS

//go:embed scripts/*
var scriptsFS embed.FS

//go:embed server/*
var serverFS embed.FS

//go:embed .env
var envFS embed.FS

// Service represents a KNIRV service to be managed by the orchestrator
type Service struct {
	Name            string
	Path            string
	Args            []string
	LogFile         string
	PidFile         string
	HealthCheck     string
	HealthCheckPort string
}

func loadEnv() {
	fmt.Println("--- Loading environment variables...")
	envData, err := envFS.ReadFile(".env")
	if err != nil {
		log.Fatalf("Error reading .env file: %v", err)
	}

	lines := strings.Split(string(envData), "\n")
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		os.Setenv(parts[0], parts[1])
	}
	fmt.Println("+++ Environment variables loaded.")
}

func main() {
	fmt.Println("🚀 KNIRV Native Orchestrator starting...")

	// Load environment variables
	loadEnv()

	// Define the base directory for deployment
	deployDir := "/opt/knirv-testnet"

	// Extract and deploy embedded files
	extractAndDeploy("bin", binFS, deployDir)
	extractAndDeploy("config", configFS, deployDir)
	extractAndDeploy("data", dataFS, deployDir)
	extractAndDeploy("scripts", scriptsFS, deployDir)
	extractAndDeploy("server", serverFS, deployDir)

	// Define the services to be orchestrated
	services := []Service{
		{
			Name:            "KNIRV-ORACLE",
			Path:            filepath.Join(deployDir, "bin", "knirvoracle"),
			Args:            []string{"--testnet", "--disable-p2p", "--config", filepath.Join(deployDir, "config", "knirvoracle-testnet-config.json"), "--port", "1317", "--p2p.port", "26656", "--shared_database_path", filepath.Join(deployDir, "data", "testnet", "blockchain.db"), "--miners_address", "KNIRVORACLE_Faucet", "--root", "--non-interactive", "--skip-install"},
			LogFile:         filepath.Join(deployDir, "logs", "knirvoracle.log"),
			PidFile:         filepath.Join(deployDir, "data", "knirvoracle.pid"),
			HealthCheck:     "/health",
			HealthCheckPort: "1317",
		},
		{
			Name:            "KNIRVCHAIN",
			Path:            filepath.Join(deployDir, "bin", "knirvchain"),
			Args:            []string{"--testnet", "--disable-mining"},
			LogFile:         filepath.Join(deployDir, "logs", "knirvchain.log"),
			PidFile:         filepath.Join(deployDir, "data", "knirvchain.pid"),
			HealthCheck:     "/health",
			HealthCheckPort: "8090",
		},
		{
			Name:            "KNIRVGRAPH",
			Path:            filepath.Join(deployDir, "bin", "knirvgraph"),
			Args:            []string{"--testnet", "--memory", "--populate", "--max-nodes", "250", "--rpc-port", "8082", "--home", filepath.Join(deployDir, "data", "knirvgraph")},
			LogFile:         filepath.Join(deployDir, "logs", "knirvgraph.log"),
			PidFile:         filepath.Join(deployDir, "data", "knirvgraph.pid"),
			HealthCheck:     "/height",
			HealthCheckPort: "8082",
		},
		{
			Name:            "KNIRV-NEXUS",
			Path:            filepath.Join(deployDir, "bin", "knirvnexus"),
			Args:            []string{"--testnet", "--port", "8084", "--host", "0.0.0.0", "--config", filepath.Join(deployDir, "config", "nexus-testnet.yaml")},
			LogFile:         filepath.Join(deployDir, "logs", "knirvnexus.log"),
			PidFile:         filepath.Join(deployDir, "data", "knirvnexus.pid"),
			HealthCheck:     "/",
			HealthCheckPort: "8084",
		},
		{
			Name:            "KNIRV-ROUTER",
			Path:            filepath.Join(deployDir, "bin", "knirvrouter"),
			Args:            []string{"-testnet", "-local-network", "-mock-nrn", "-port", "8086", "-miners_address", "KNIRVROUTER_Testnet_Miner"},
			LogFile:         filepath.Join(deployDir, "logs", "knirvrouter.log"),
			PidFile:         filepath.Join(deployDir, "data", "knirvrouter.pid"),
			HealthCheck:     "/status",
			HealthCheckPort: "8086",
		},
		// Node.js services will be handled separately
	}

	// Create necessary directories
	createDirectories(deployDir)

	// Start Go services sequentially
	for _, s := range services {
		startService(s)
	}

	// Start Node.js services
	startNodeServices(deployDir)

	fmt.Println("🎉 All services started successfully!")

	// Wait for shutdown signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("🛑 Shutting down KNIRV Orchestrator...")
	// In a production environment, we would gracefully stop all services here
}

func extractAndDeploy(fsName string, embedFS embed.FS, deployDir string) {
	fmt.Printf("- Extracting and deploying %s...\n", fsName)
	err := fs.WalkDir(embedFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// The path in embed.FS is relative to the embedded directory.
		// We need to construct the full destination path.
		destPath := filepath.Join(deployDir, path)

		// Ensure the destination directory exists.
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		// Read the embedded file.
		data, err := embedFS.ReadFile(path)
		if err != nil {
			return err
		}

		// Write the file to the destination.
		if err := os.WriteFile(destPath, data, 0755); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Error extracting and deploying %s: %v", fsName, err)
	}
}

func createDirectories(deployDir string) {
	dirs := []string{"logs", "data", "config", "bin"}
	for _, dir := range dirs {
		fullPath := filepath.Join(deployDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			log.Printf("Error creating directory %s: %v", fullPath, err)
		} else {
			log.Printf("✅ Created directory: %s", fullPath)
		}
	}
}

func startService(s Service) {
	fmt.Printf("🚀 Starting service: %s\n", s.Name)

	logFile, err := os.Create(s.LogFile)
	if err != nil {
		log.Printf("Error creating log file for %s: %v", s.Name, err)
		return
	}
	defer logFile.Close()

	cmd := exec.Command(s.Path, s.Args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		log.Printf("❌ Failed to start service %s: %v", s.Name, err)
		return
	}

	// Write PID file
	pidFile, err := os.Create(s.PidFile)
	if err != nil {
		log.Printf("Error creating pid file for %s: %v", s.Name, err)
		return
	}
	defer pidFile.Close()
	fmt.Fprintf(pidFile, "%d", cmd.Process.Pid)

	log.Printf("✅ Service %s started with PID %d. Logs at %s", s.Name, cmd.Process.Pid, s.LogFile)

	// Wait for service to be healthy
	waitForService(s)

	// Wait for the service to complete (in a separate goroutine to not block)
	go func() {
		err := cmd.Wait()
		log.Printf("Service %s exited. Error: %v", s.Name, err)
	}()
}

func waitForService(s Service) {
	if s.HealthCheck == "" {
		return
	}

	fmt.Printf("⏳ Waiting for %s to be healthy...\n", s.Name)
	healthURL := fmt.Sprintf("http://localhost:%s%s", s.HealthCheckPort, s.HealthCheck)

	for i := 0; i < 30; i++ {
		resp, err := http.Get(healthURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			fmt.Printf("✅ %s is healthy!\n", s.Name)
			return
		}
		time.Sleep(2 * time.Second)
	}

	log.Printf("⚠️ %s failed to become healthy after 60 seconds.", s.Name)
}

func startNodeServices(deployDir string) {
	fmt.Println("🚀 Starting Node.js services...")

	// Start KNIRVCONTROLLER
	startNodeService("KNIRVCONTROLLER", filepath.Join(deployDir, "scripts", "start-knirvcontroller.sh"), "auto")

	// Start KNIRV-GATEWAY
	startNodeService("KNIRV-GATEWAY", filepath.Join(deployDir, "scripts", "start-knirvgateway.sh"))

	// Start KNIRVTESTNET Server
	startNodeService("KNIRVTESTNET Server", "node", filepath.Join(deployDir, "server", "app.js"))
}

func startNodeService(name string, command string, args ...string) {
	fmt.Printf("🚀 Starting Node.js service: %s\n", name)

	logFile, err := os.Create(filepath.Join("/opt/knirv-testnet", "logs", name+".log"))
	if err != nil {
		log.Printf("Error creating log file for %s: %v", name, err)
		return
	}
	defer logFile.Close()

	cmd := exec.Command(command, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		log.Printf("❌ Failed to start Node.js service %s: %v", name, err)
		return
	}

	// Write PID file
	pidFile, err := os.Create(filepath.Join("/opt/knirv-testnet", "data", name+".pid"))
	if err != nil {
		log.Printf("Error creating pid file for %s: %v", name, err)
		return
	}
	defer pidFile.Close()
	fmt.Fprintf(pidFile, "%d", cmd.Process.Pid)

	log.Printf("✅ Node.js service %s started with PID %d. Logs at %s", name, cmd.Process.Pid, logFile.Name())

	// Wait for the service to complete (in a separate goroutine to not block)
	go func() {
		err := cmd.Wait()
		log.Printf("Node.js service %s exited. Error: %v", name, err)
	}()
}
