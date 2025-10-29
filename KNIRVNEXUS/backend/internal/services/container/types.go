package container

import (
	"time"

	"backend_server/internal/objects"
)

// Container represents a provisioned container
type Container struct {
	ID            string            `json:"id"`
	Status        ContainerStatus   `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	Spec          *ContainerSpec    `json:"spec"`
	Runtime       string            `json:"runtime"`
	SSHKeys       *SSHKeypair       `json:"ssh_keys,omitempty"`
	Endpoints     *Endpoints        `json:"endpoints,omitempty"`
	ResourceUsage *ContainerMetrics `json:"resource_usage,omitempty"`
}

// ContainerStatus represents the status of a container
type ContainerStatus string

const (
	ContainerStatusPending   ContainerStatus = "pending"
	ContainerStatusRunning   ContainerStatus = "running"
	ContainerStatusStopped   ContainerStatus = "stopped"
	ContainerStatusFailed    ContainerStatus = "failed"
	ContainerStatusTerminated ContainerStatus = "terminated"
)

// ContainerSpec defines the specification for creating a container
type ContainerSpec struct {
	ID              string                      `json:"id"`
	Image           string                      `json:"image"`
	SSHPort         int                         `json:"ssh_port"`
	ValidationPort  int                         `json:"validation_port"`
	ErrorResPort    int                         `json:"error_resolution_port"`
	SSHUsername     string                      `json:"ssh_username"`
	SSHPublicKey    string                      `json:"ssh_public_key"`
	TEEType         string                      `json:"tee_type"`
	ResourceLimits  objects.ResourceLimits      `json:"resource_limits"`
	Environment     map[string]string           `json:"environment,omitempty"`
	Volumes         map[string]string           `json:"volumes,omitempty"`
	NetworkMode     string                      `json:"network_mode,omitempty"`
	RestartPolicy   string                      `json:"restart_policy,omitempty"`
	Labels          map[string]string           `json:"labels,omitempty"`
}

// Endpoints represents the network endpoints allocated for a container
type Endpoints struct {
	SSHPort         int    `json:"ssh_port"`
	ValidationPort  int    `json:"validation_port"`
	ErrorResPort    int    `json:"error_resolution_port"`
	Host            string `json:"host"`
}

// SSHKeypair represents an SSH key pair
type SSHKeypair struct {
	PublicKey      string `json:"public_key"`
	PrivateKey     string `json:"private_key"`
	KeyFingerprint string `json:"key_fingerprint"`
}

// ContainerMetrics represents resource usage metrics for a container
type ContainerMetrics struct {
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsage    int64     `json:"memory_usage"`
	DiskUsage      int64     `json:"disk_usage"`
	NetworkRX      int64     `json:"network_rx"`
	NetworkTX      int64     `json:"network_tx"`
	LastUpdated    time.Time `json:"last_updated"`
}

// PortAllocation represents a port allocation record
type PortAllocation struct {
	RentalID       string    `json:"rental_id"`
	SSHPort        int       `json:"ssh_port"`
	ValidationPort int       `json:"validation_port"`
	ErrorResPort   int       `json:"error_resolution_port"`
	AllocatedAt    time.Time `json:"allocated_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// PortAllocationConfig holds configuration for port allocation
type PortAllocationConfig struct {
	SSHRangeStart        int
	SSHRangeEnd          int
	ValidationRangeStart int
	ValidationRangeEnd   int
	ErrorResRangeStart   int
	ErrorResRangeEnd     int
}

// ContainerEvent represents an event in the container lifecycle
type ContainerEvent struct {
	ID          string          `json:"id"`
	ContainerID string          `json:"container_id"`
	RentalID    string          `json:"rental_id"`
	Type        ContainerEventType `json:"type"`
	Message     string          `json:"message"`
	Timestamp   time.Time       `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ContainerEventType represents the type of container event
type ContainerEventType string

const (
	ContainerEventProvisioned ContainerEventType = "provisioned"
	ContainerEventStarted     ContainerEventType = "started"
	ContainerEventStopped     ContainerEventType = "stopped"
	ContainerEventFailed      ContainerEventType = "failed"
	ContainerEventTerminated  ContainerEventType = "terminated"
	ContainerEventCleaned     ContainerEventType = "cleaned"
)
