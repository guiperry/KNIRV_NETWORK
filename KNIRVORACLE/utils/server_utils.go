package utils

import (
	"fmt"
	"net"
	"sync"
)

// Global variable to store the server port
var (
	serverPort uint64
	portMutex  sync.Mutex
)

// SetServerPort sets the server port globally
func SetServerPort(port uint64) {
	portMutex.Lock()
	defer portMutex.Unlock()
	serverPort = port
}

// GetServerPort returns the current server port
func GetServerPort() uint64 {
	portMutex.Lock()
	defer portMutex.Unlock()
	return serverPort
}

func IsPortAvailable(port uint64) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// FindAvailablePort finds an available port starting from startPort
func FindAvailablePort(startPort uint64) uint64 {
	port := startPort
	for {
		if IsPortAvailable(port) {
			return port
		}
		if port > 65535 {
			port = 1024 // Reset to well-known ports if we hit max
		} else {
			port++
		}
	}
}
