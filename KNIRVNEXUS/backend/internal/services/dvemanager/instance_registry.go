package dvemanager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/config"
	"backend_server/internal/objects"

	"github.com/tidwall/buntdb"
)

// InstanceRegistry manages remote DVE instance connections and health
type InstanceRegistry struct {
	db        *buntdb.DB
	config    *config.DVEDiscoveryConfig
	clients   map[string]*DVEClient // URL -> client mapping
	instances map[string]*RemoteInstance
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// RemoteInstance represents a remote DVE instance
type RemoteInstance struct {
	URL             string
	LastHealthCheck time.Time
	IsHealthy       bool
	ConnectionCount int
	FailureCount    int
	LastError       string
	AvailableNodes  int
	Client          *DVEClient
	LastErrorTime   time.Time
	BackoffUntil    time.Time
}

// NewInstanceRegistry creates a new instance registry
func NewInstanceRegistry(db *buntdb.DB, cfg *config.DVEDiscoveryConfig) (*InstanceRegistry, error) {
	ctx, cancel := context.WithCancel(context.Background())

	registry := &InstanceRegistry{
		db:        db,
		config:    cfg,
		clients:   make(map[string]*DVEClient),
		instances: make(map[string]*RemoteInstance),
		ctx:       ctx,
		cancel:    cancel,
	}

	return registry, nil
}

// Start starts the instance registry service
func (ir *InstanceRegistry) Start() error {
	log.Println("Starting DVE Instance Registry...")

	// Set defaults for config values if not set
	if ir.config.HealthCheckInterval == 0 {
		ir.config.HealthCheckInterval = 60 * time.Second
	}
	if ir.config.DiscoveryInterval == 0 {
		ir.config.DiscoveryInterval = 30 * time.Second
	}
	if ir.config.ConnectionTimeout == 0 {
		ir.config.ConnectionTimeout = 5 * time.Second
	}

	// Initialize bootstrap instances
	for _, bootstrapURL := range ir.config.BootstrapNodes {
		if err := ir.addInstance(bootstrapURL); err != nil {
			log.Printf("Warning: Failed to add bootstrap instance %s: %v", bootstrapURL, err)
		}
	}

	// Start discovery goroutine
	ir.wg.Add(1)
	go ir.discoverInstances()

	// Start health check goroutine
	ir.wg.Add(1)
	go ir.checkInstanceHealth()

	log.Println("DVE Instance Registry started successfully")
	return nil
}

// Stop stops the instance registry service
func (ir *InstanceRegistry) Stop() error {
	log.Println("Stopping DVE Instance Registry...")

	ir.cancel()
	ir.wg.Wait()

	// Close all clients
	ir.mu.Lock()
	for _, client := range ir.clients {
		client.Close()
	}
	ir.clients = make(map[string]*DVEClient)
	ir.instances = make(map[string]*RemoteInstance)
	ir.mu.Unlock()

	log.Println("DVE Instance Registry stopped")
	return nil
}

// AddInstance adds a new remote instance to the registry (public wrapper)
func (ir *InstanceRegistry) AddInstance(url string) error {
	return ir.addInstance(url)
}

// addInstance adds a new remote instance to the registry
func (ir *InstanceRegistry) addInstance(url string) error {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	if _, exists := ir.instances[url]; exists {
		return fmt.Errorf("instance %s already exists", url)
	}

	// Create client
	client := NewDVEClient(
		url,
		ir.config.ConnectionTimeout,
		ir.config.MaxRetries,
		ir.config.RetryBackoffDuration,
	)

	instance := &RemoteInstance{
		URL:             url,
		Client:          client,
		LastHealthCheck: time.Now(),
		IsHealthy:       false,
	}

	ir.instances[url] = instance
	ir.clients[url] = client

	log.Printf("Added remote DVE instance: %s", url)
	return nil
}

// RemoveInstance removes a remote instance from the registry
func (ir *InstanceRegistry) RemoveInstance(url string) error {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	instance, exists := ir.instances[url]
	if !exists {
		return fmt.Errorf("instance %s not found", url)
	}

	if instance.Client != nil {
		instance.Client.Close()
	}

	delete(ir.instances, url)
	delete(ir.clients, url)

	log.Printf("Removed remote DVE instance: %s", url)
	return nil
}

// GetHealthyInstances returns all healthy instances
func (ir *InstanceRegistry) GetHealthyInstances() []string {
	ir.mu.RLock()
	defer ir.mu.RUnlock()

	var healthy []string
	for url, instance := range ir.instances {
		if instance.IsHealthy && time.Now().Before(instance.BackoffUntil) == false {
			healthy = append(healthy, url)
		}
	}

	return healthy
}

// GetInstance gets a specific instance
func (ir *InstanceRegistry) GetInstance(url string) *RemoteInstance {
	ir.mu.RLock()
	defer ir.mu.RUnlock()

	return ir.instances[url]
}

// GetAllInstances returns all instances
func (ir *InstanceRegistry) GetAllInstances() map[string]*RemoteInstance {
	ir.mu.RLock()
	defer ir.mu.RUnlock()

	// Return a copy to avoid concurrent map access
	instances := make(map[string]*RemoteInstance)
	for url, instance := range ir.instances {
		instances[url] = instance
	}

	return instances
}

// GetInstanceNodes fetches nodes from a specific remote instance
func (ir *InstanceRegistry) GetInstanceNodes(url string) ([]*objects.DVENode, error) {
	ir.mu.RLock()
	_, exists := ir.instances[url]
	client := ir.clients[url]
	ir.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("instance %s not found", url)
	}

	ctx, cancel := context.WithTimeout(ir.ctx, ir.config.ConnectionTimeout)
	defer cancel()

	nodes, err := client.GetNodes(ctx, "online", ir.config.ConnectionPoolSize)
	if err != nil {
		ir.recordInstanceFailure(url, err)
		return nil, err
	}

	// Mark nodes as remote and convert to pointers
	nodePointers := make([]*objects.DVENode, len(nodes))
	for i := range nodes {
		nodes[i].IsRemote = true
		nodes[i].RemoteInstanceURL = url
		nodes[i].Connected = true
		nodePointers[i] = &nodes[i]
	}

	ir.recordInstanceSuccess(url, len(nodes))
	return nodePointers, nil
}

// discoverInstances periodically discovers new instances
func (ir *InstanceRegistry) discoverInstances() {
	defer ir.wg.Done()

	ticker := time.NewTicker(ir.config.DiscoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ir.ctx.Done():
			return
		case <-ticker.C:
			ir.performDiscovery()
		}
	}
}

// performDiscovery discovers nodes from all instances and persists them to DB
func (ir *InstanceRegistry) performDiscovery() {
	healthyInstances := ir.GetHealthyInstances()
	if len(healthyInstances) == 0 {
		log.Println("No healthy DVE instances available for discovery")
		return
	}

	for _, instanceURL := range healthyInstances {
		nodes, err := ir.GetInstanceNodes(instanceURL)
		if err != nil {
			log.Printf("Failed to discover nodes from %s: %v", instanceURL, err)
			continue
		}

		// Persist discovered nodes to database
		for _, node := range nodes {
			if err := ir.persistNodeToDatabase(node); err != nil {
				log.Printf("Failed to persist discovered node %s: %v", node.ID, err)
			}
		}

		log.Printf("Discovered %d nodes from instance %s", len(nodes), instanceURL)
	}
}

// persistNodeToDatabase persists a discovered node to the database
func (ir *InstanceRegistry) persistNodeToDatabase(node *objects.DVENode) error {
	if node.ID == "" {
		return fmt.Errorf("node ID is empty")
	}

	nodeJSON, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal node: %w", err)
	}

	key := fmt.Sprintf("dve:nodes:%s", node.ID)
	return ir.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, string(nodeJSON), nil)
		return err
	})
}

// checkInstanceHealth periodically checks health of all instances
func (ir *InstanceRegistry) checkInstanceHealth() {
	defer ir.wg.Done()

	ticker := time.NewTicker(ir.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ir.ctx.Done():
			return
		case <-ticker.C:
			ir.performHealthChecks()
		}
	}
}

// performHealthChecks checks health of all instances
func (ir *InstanceRegistry) performHealthChecks() {
	instances := ir.GetAllInstances()

	ctx, cancel := context.WithTimeout(ir.ctx, ir.config.ConnectionTimeout)
	defer cancel()

	for url, instance := range instances {
		isHealthy, err := instance.Client.HealthCheck(ctx)

		ir.mu.Lock()
		if isHealthy {
			instance.IsHealthy = true
			instance.FailureCount = 0
			instance.LastError = ""
			instance.BackoffUntil = time.Time{} // Clear backoff
		} else {
			instance.IsHealthy = false
			instance.FailureCount++
			instance.LastError = err.Error()
			instance.LastErrorTime = time.Now()

			// Implement exponential backoff
			backoffDuration := time.Duration(instance.FailureCount*instance.FailureCount) * time.Second
			if backoffDuration > 5*time.Minute {
				backoffDuration = 5 * time.Minute
			}
			instance.BackoffUntil = time.Now().Add(backoffDuration)
		}

		instance.LastHealthCheck = time.Now()
		ir.instances[url] = instance
		ir.mu.Unlock()

		if isHealthy {
			log.Printf("Instance %s is healthy", url)
		} else {
			log.Printf("Instance %s health check failed: %v (backoff: %v)", url, err, time.Until(instance.BackoffUntil))
		}
	}
}

// recordInstanceSuccess records a successful operation
func (ir *InstanceRegistry) recordInstanceSuccess(url string, nodesCount int) {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	if instance, exists := ir.instances[url]; exists {
		instance.ConnectionCount++
		instance.AvailableNodes = nodesCount
		instance.FailureCount = 0
		instance.LastError = ""
	}
}

// recordInstanceFailure records a failed operation
func (ir *InstanceRegistry) recordInstanceFailure(url string, err error) {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	if instance, exists := ir.instances[url]; exists {
		instance.FailureCount++
		instance.LastError = err.Error()
		instance.LastErrorTime = time.Now()

		// Implement exponential backoff
		backoffDuration := time.Duration(instance.FailureCount*instance.FailureCount) * time.Second
		if backoffDuration > 5*time.Minute {
			backoffDuration = 5 * time.Minute
		}
		instance.BackoffUntil = time.Now().Add(backoffDuration)
	}
}

// GetInstanceStats returns statistics for an instance
func (ir *InstanceRegistry) GetInstanceStats(url string) map[string]interface{} {
	ir.mu.RLock()
	defer ir.mu.RUnlock()

	instance, exists := ir.instances[url]
	if !exists {
		return map[string]interface{}{
			"error": "instance not found",
		}
	}

	return map[string]interface{}{
		"url":               url,
		"is_healthy":        instance.IsHealthy,
		"connection_count":  instance.ConnectionCount,
		"failure_count":     instance.FailureCount,
		"available_nodes":   instance.AvailableNodes,
		"last_health_check": instance.LastHealthCheck,
		"last_error":        instance.LastError,
		"last_error_time":   instance.LastErrorTime,
		"backoff_until":     instance.BackoffUntil,
		"is_backoff_active": time.Now().Before(instance.BackoffUntil),
	}
}

// GetAllInstancesStats returns statistics for all instances
func (ir *InstanceRegistry) GetAllInstancesStats() map[string]map[string]interface{} {
	ir.mu.RLock()
	defer ir.mu.RUnlock()

	stats := make(map[string]map[string]interface{})

	for url, instance := range ir.instances {
		stats[url] = map[string]interface{}{
			"url":               url,
			"is_healthy":        instance.IsHealthy,
			"connection_count":  instance.ConnectionCount,
			"failure_count":     instance.FailureCount,
			"available_nodes":   instance.AvailableNodes,
			"last_health_check": instance.LastHealthCheck,
			"last_error":        instance.LastError,
			"last_error_time":   instance.LastErrorTime,
			"backoff_until":     instance.BackoffUntil,
			"is_backoff_active": time.Now().Before(instance.BackoffUntil),
		}
	}

	return stats
}
