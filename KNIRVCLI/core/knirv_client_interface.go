package core

import (
	"context"
	"fmt"
)

// KNIRVServiceClient defines the common interface for all KNIRV service clients
type KNIRVServiceClient interface {
	// Connect establishes connection to the service
	Connect(ctx context.Context) error

	// Disconnect closes connection to the service
	Disconnect() error

	// HealthCheck performs a health check on the service
	HealthCheck(ctx context.Context) error

	// GetCapabilities returns the capabilities of the service
	GetCapabilities() []string

	// Subscribe subscribes to events from the service
	Subscribe(events []string, handler EventHandler) error

	// GetServiceName returns the name of the service
	GetServiceName() string

	// GetServiceURL returns the URL of the service
	GetServiceURL() string

	// IsConnected returns whether the client is connected
	IsConnected() bool
}

// EventHandler defines the interface for handling events
type EventHandler interface {
	HandleEvent(event *Event) error
}

// Event represents an event from a KNIRV service
type Event struct {
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
	Timestamp string                 `json:"timestamp"`
}

// KNIRVClientManager manages all KNIRV service clients
type KNIRVClientManager struct {
	registry      *ServiceRegistry
	clients       map[string]KNIRVServiceClient
	eventBus      *EventBus
	healthMonitor *HealthMonitor
}

// NewKNIRVClientManager creates a new client manager
func NewKNIRVClientManager(registry *ServiceRegistry, eventBus *EventBus, healthMonitor *HealthMonitor) *KNIRVClientManager {
	return &KNIRVClientManager{
		registry:      registry,
		clients:       make(map[string]KNIRVServiceClient),
		eventBus:      eventBus,
		healthMonitor: healthMonitor,
	}
}

// RegisterClient registers a service client
func (cm *KNIRVClientManager) RegisterClient(name string, client KNIRVServiceClient) {
	cm.clients[name] = client
}

// GetClient returns a service client by name
func (cm *KNIRVClientManager) GetClient(name string) (KNIRVServiceClient, bool) {
	client, exists := cm.clients[name]
	return client, exists
}

// ConnectAll connects to all registered services
func (cm *KNIRVClientManager) ConnectAll(ctx context.Context) error {
	for name, client := range cm.clients {
		if err := client.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect to %s: %w", name, err)
		}
	}
	return nil
}

// DisconnectAll disconnects from all services
func (cm *KNIRVClientManager) DisconnectAll() error {
	var lastErr error
	for name, client := range cm.clients {
		if err := client.Disconnect(); err != nil {
			lastErr = fmt.Errorf("failed to disconnect from %s: %w", name, err)
		}
	}
	return lastErr
}

// GetConnectedClients returns all connected clients
func (cm *KNIRVClientManager) GetConnectedClients() map[string]KNIRVServiceClient {
	connected := make(map[string]KNIRVServiceClient)
	for name, client := range cm.clients {
		if client.IsConnected() {
			connected[name] = client
		}
	}
	return connected
}
