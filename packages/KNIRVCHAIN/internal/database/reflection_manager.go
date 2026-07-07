package database

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// ReflectionNode represents a network reflection node
type ReflectionNode struct {
	ReflectionAddress string
	IsActive          bool
	LastSeen          time.Time
}

// NetworkReflectionManager manages the list of reflection nodes
type NetworkReflectionManager struct {
	mu          sync.Mutex
	reflections []ReflectionNode
}

var (
	reflectionManagerInstance *NetworkReflectionManager
	reflectionManagerOnce     sync.Once
)

// GetReflectionManager returns the singleton instance
func GetReflectionManager() *NetworkReflectionManager {
	reflectionManagerOnce.Do(func() {
		reflectionManagerInstance = &NetworkReflectionManager{}
	})
	return reflectionManagerInstance
}

// GetReflections returns all known reflections
func (rm *NetworkReflectionManager) GetReflections() []ReflectionNode {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return rm.reflections
}

// AddReflection adds a new reflection node
func (rm *NetworkReflectionManager) AddReflection(address string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	newReflection := ReflectionNode{
		ReflectionAddress: address,
		IsActive:          true,
		LastSeen:          time.Now(),
	}
	rm.reflections = append(rm.reflections, newReflection)
}

// ReflectionClient handles communication with reflection nodes
type ReflectionClient struct {
	ReflectionAddress string
	httpClient        *http.Client
}

// NewReflectionClient creates a new client for a reflection node
func NewReflectionClient(address string) *ReflectionClient {
	return &ReflectionClient{
		ReflectionAddress: address,
		httpClient:        &http.Client{Timeout: 5 * time.Second},
	}
}

// Ping checks if the reflection node is responsive
func (c *ReflectionClient) Ping() bool {
	resp, err := c.httpClient.Get(c.ReflectionAddress + "/ping")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// BroadcastTransaction sends a transaction to the reflection node
func (c *ReflectionClient) BroadcastTransaction(tx interface{}) (bool, error) {
	url := fmt.Sprintf("%s/transaction", c.ReflectionAddress)
	data, err := json.Marshal(tx)
	if err != nil {
		return false, fmt.Errorf("failed to marshal transaction: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return false, fmt.Errorf("failed to send POST request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("POST request failed: %s. Status Code: %d, Body: %s",
			url, resp.StatusCode, string(body))
	}

	return true, nil
}

// BroadcastBlock sends a block to the reflection node
func (c *ReflectionClient) BroadcastBlock(block interface{}) (bool, error) {
	url := fmt.Sprintf("%s/block", c.ReflectionAddress)
	data, err := json.Marshal(block)
	if err != nil {
		return false, fmt.Errorf("failed to marshal block: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return false, fmt.Errorf("failed to send POST request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("POST request failed: %s. Status Code: %d, Body: %s",
			url, resp.StatusCode, string(body))
	}

	return true, nil
}

// Close cleans up the client resources
func (c *ReflectionClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
