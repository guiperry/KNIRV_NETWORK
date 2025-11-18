package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Reflection represents a network reflection node
type Reflection struct {
	ReflectionAddress string
	IsActive          bool
	LastSeen          time.Time
}

// ReflectionManager manages the list of reflection nodes
type ReflectionManager struct {
	mu          sync.Mutex
	reflections []Reflection
}

var (
	reflectionManagerInstance *ReflectionManager
	reflectionManagerOnce     sync.Once
)

// GetReflectionManager returns the singleton instance
func GetReflectionManager() *ReflectionManager {
	reflectionManagerOnce.Do(func() {
		reflectionManagerInstance = &ReflectionManager{}
	})
	return reflectionManagerInstance
}

// GetReflections returns all known reflections
func (rm *ReflectionManager) GetReflections() []Reflection {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return rm.reflections
}

// AddReflection adds a new reflection node
func (rm *ReflectionManager) AddReflection(address string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	newReflection := Reflection{
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
func (c *ReflectionClient) BroadcastTransaction(tx *Transaction) (bool, error) {
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
func (c *ReflectionClient) BroadcastBlock(block *Block) (bool, error) {
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
