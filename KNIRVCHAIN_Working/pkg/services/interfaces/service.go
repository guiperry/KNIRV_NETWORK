package interfaces

import (
	"context"
	"time"
)

// Service represents a generic service interface
type Service interface {
	// Start starts the service
	Start(ctx context.Context) error
	
	// Stop stops the service
	Stop(ctx context.Context) error
	
	// IsRunning returns true if the service is currently running
	IsRunning() bool
	
	// Name returns the service name
	Name() string
	
	// Status returns the current service status
	Status() ServiceStatus
}

// EmbeddedService represents an embedded service (Node.js or binary)
type EmbeddedService interface {
	Service
	
	// GetPort returns the port the service is running on
	GetPort() uint64
	
	// GetPID returns the process ID if applicable
	GetPID() int
	
	// Restart restarts the service
	Restart(ctx context.Context) error
}

// ServiceManager manages multiple services
type ServiceManager interface {
	// RegisterService registers a service with the manager
	RegisterService(service Service) error
	
	// UnregisterService unregisters a service from the manager
	UnregisterService(name string) error
	
	// StartService starts a specific service
	StartService(name string) error
	
	// StopService stops a specific service
	StopService(name string) error
	
	// StartAllServices starts all registered services
	StartAllServices() error
	
	// StopAllServices stops all registered services
	StopAllServices() error
	
	// GetService returns a service by name
	GetService(name string) (Service, error)
	
	// ListServices returns all registered services
	ListServices() []Service
	
	// GetRunningServices returns all currently running services
	GetRunningServices() []Service
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name      string    `json:"name"`
	Running   bool      `json:"running"`
	StartTime time.Time `json:"start_time,omitempty"`
	Port      uint64    `json:"port,omitempty"`
	PID       int       `json:"pid,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// ServiceConfig represents configuration for a service
type ServiceConfig struct {
	Name        string            `json:"name"`
	Enabled     bool              `json:"enabled"`
	Port        uint64            `json:"port"`
	ScriptPath  string            `json:"script_path,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Args        []string          `json:"args,omitempty"`
}

// NodeJSServiceConfig represents configuration for Node.js services
type NodeJSServiceConfig struct {
	ServiceConfig
	NodePath string `json:"node_path,omitempty"`
}

// BinaryServiceConfig represents configuration for binary services
type BinaryServiceConfig struct {
	ServiceConfig
	BinaryPath string `json:"binary_path"`
	WorkingDir string `json:"working_dir,omitempty"`
}
