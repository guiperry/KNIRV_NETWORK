package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"KNIRVCHAIN/config"
)

// RunExample demonstrates how to use the tunnel client
func RunTunnelExample() {
	// Create a tunnel client configuration
	tunnelConfig := &config.TunnelClientConfig{
		Enabled:        true,
		ServerAddress:  getEnvOrDefault("TUNNEL_SERVER_HOST", "ROOTCHAIN_URL"),
		ControlPort:    uint(getEnvAsIntOrDefault("TUNNEL_SERVER_CONTROL_PORT", 4001)),
		PingInterval:   uint(getEnvAsIntOrDefault("PING_INTERVAL", 30)),
		ReconnectDelay: uint(getEnvAsIntOrDefault("RECONNECT_DELAY", 5)),
	}

	// Create a new tunnel client
	client := NewTunnelClient(
		tunnelConfig,
		getEnvOrDefault("PEER_ID", "QmExamplePeerID"),
		getEnvOrDefault("CHAIN_ID", "agent-default"),
		getEnvOrDefault("INTERNAL_IP", "192.168.1.100"),
		uint(getEnvAsIntOrDefault("INTERNAL_P2P_PORT", 5050)),
		getEnvOrDefault("NODE_TYPE", "dev"),
	)

	// Connect to the tunnel server
	err := client.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to tunnel server: %v", err)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	<-sigChan
	log.Println("Received termination signal. Shutting down...")

	// Disconnect from the tunnel server
	client.Disconnect()
	log.Println("Shutdown complete")
}

// Helper function to get environment variable with default value
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Helper function to get environment variable as integer with default value
func getEnvAsIntOrDefault(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Warning: Invalid value for %s: %s. Using default: %d", key, valueStr, defaultValue)
		return defaultValue
	}
	return value
}
