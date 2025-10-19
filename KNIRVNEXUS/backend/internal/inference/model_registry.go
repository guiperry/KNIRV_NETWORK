package inference

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ModelRegistry manages model instances and their lifecycle
type ModelRegistry struct {
	objects   map[string]*RegisteredModel
	cache     *ModelCache
	maxModels int

	mu sync.RWMutex
}

// RegisteredModel represents a registered model in the registry
type RegisteredModel struct {
	Name         string      `json:"name"`
	Provider     string      `json:"provider"`
	Version      string      `json:"version"`
	Status       ModelStatus `json:"status"`
	RegisteredAt time.Time   `json:"registered_at"`
	LastUsed     time.Time   `json:"last_used"`
	UsageCount   int64       `json:"usage_count"`

	// Performance metrics
	AvgLatency    time.Duration `json:"avg_latency"`
	ErrorRate     float64       `json:"error_rate"`
	ThroughputRPS float64       `json:"throughput_rps"`

	// Resource usage
	MemoryUsage uint64  `json:"memory_usage"`
	CPUUsage    float64 `json:"cpu_usage"`

	// Configuration
	Config       map[string]interface{} `json:"config"`
	Capabilities []string               `json:"capabilities"`

	// Health status
	HealthScore     float64   `json:"health_score"`
	LastHealthCheck time.Time `json:"last_health_check"`
}

// ModelStatus represents the status of a model
type ModelStatus string

const (
	ModelStatusRegistered ModelStatus = "registered"
	ModelStatusLoading    ModelStatus = "loading"
	ModelStatusReady      ModelStatus = "ready"
	ModelStatusBusy       ModelStatus = "busy"
	ModelStatusError      ModelStatus = "error"
	ModelStatusUnloading  ModelStatus = "unloading"
	ModelStatusUnloaded   ModelStatus = "unloaded"
)

// ModelCache manages model caching and eviction
type ModelCache struct {
	entries     map[string]*CacheEntry
	maxSize     int
	currentSize int

	// LRU tracking
	head *CacheEntry
	tail *CacheEntry

	mu sync.RWMutex
}

// CacheEntry represents a cached model entry
type CacheEntry struct {
	ModelName   string
	LoadedAt    time.Time
	LastAccess  time.Time
	AccessCount int64
	Size        int // Memory size in MB

	// LRU pointers
	prev *CacheEntry
	next *CacheEntry
}

// NewModelRegistry creates a new model registry
func NewModelRegistry(maxModels int) *ModelRegistry {
	return &ModelRegistry{
		objects:   make(map[string]*RegisteredModel),
		cache:     NewModelCache(maxModels),
		maxModels: maxModels,
	}
}

// NewModelCache creates a new model cache
func NewModelCache(maxSize int) *ModelCache {
	cache := &ModelCache{
		entries: make(map[string]*CacheEntry),
		maxSize: maxSize,
	}

	// Initialize LRU list
	cache.head = &CacheEntry{}
	cache.tail = &CacheEntry{}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head

	return cache
}

// RegisterModel registers a new model in the registry
func (mr *ModelRegistry) RegisterModel(name, provider, version string, config map[string]interface{}) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	// Check if model already exists
	if _, exists := mr.objects[name]; exists {
		return fmt.Errorf("model %s is already registered", name)
	}

	// Check model limit
	if len(mr.objects) >= mr.maxModels {
		// Try to evict least used model
		if err := mr.evictLeastUsedModel(); err != nil {
			return fmt.Errorf("cannot register model: registry is full and eviction failed: %w", err)
		}
	}

	// Create registered model
	model := &RegisteredModel{
		Name:            name,
		Provider:        provider,
		Version:         version,
		Status:          ModelStatusRegistered,
		RegisteredAt:    time.Now(),
		LastUsed:        time.Now(),
		Config:          config,
		Capabilities:    []string{},
		HealthScore:     1.0,
		LastHealthCheck: time.Now(),
	}

	// Determine capabilities based on provider
	model.Capabilities = mr.determineCapabilities(provider)

	mr.objects[name] = model

	log.Printf("ModelRegistry: Registered model %s (provider: %s, version: %s)", name, provider, version)

	return nil
}

// UnregisterModel removes a model from the registry
func (mr *ModelRegistry) UnregisterModel(name string) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	model, exists := mr.objects[name]
	if !exists {
		return fmt.Errorf("model %s not found", name)
	}

	// Update status
	model.Status = ModelStatusUnloading

	// Remove from cache
	mr.cache.Remove(name)

	// Remove from registry
	delete(mr.objects, name)

	log.Printf("ModelRegistry: Unregistered model %s", name)

	return nil
}

// GetModel returns a model by name
func (mr *ModelRegistry) GetModel(name string) (*RegisteredModel, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	model, exists := mr.objects[name]
	if !exists {
		return nil, fmt.Errorf("model %s not found", name)
	}

	// Update last used time
	model.LastUsed = time.Now()
	model.UsageCount++

	// Update cache access
	mr.cache.Access(name)

	// Return a copy
	modelCopy := *model
	return &modelCopy, nil
}

// ListModels returns all registered objects
func (mr *ModelRegistry) ListModels() []*RegisteredModel {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	objects := make([]*RegisteredModel, 0, len(mr.objects))
	for _, model := range mr.objects {
		modelCopy := *model
		objects = append(objects, &modelCopy)
	}

	return objects
}

// UpdateModelStatus updates the status of a model
func (mr *ModelRegistry) UpdateModelStatus(name string, status ModelStatus) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	model, exists := mr.objects[name]
	if !exists {
		return fmt.Errorf("model %s not found", name)
	}

	model.Status = status

	return nil
}

// UpdateModelMetrics updates performance metrics for a model
func (mr *ModelRegistry) UpdateModelMetrics(name string, latency time.Duration, errorOccurred bool) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	model, exists := mr.objects[name]
	if !exists {
		return fmt.Errorf("model %s not found", name)
	}

	// Update average latency (simple moving average)
	if model.AvgLatency == 0 {
		model.AvgLatency = latency
	} else {
		model.AvgLatency = (model.AvgLatency + latency) / 2
	}

	// Update error rate
	if errorOccurred {
		model.ErrorRate = (model.ErrorRate*float64(model.UsageCount-1) + 1.0) / float64(model.UsageCount)
	} else {
		model.ErrorRate = (model.ErrorRate * float64(model.UsageCount-1)) / float64(model.UsageCount)
	}

	// Update health score based on performance
	mr.updateHealthScore(model)

	return nil
}

// GetBestModel returns the best performing model for a given capability
func (mr *ModelRegistry) GetBestModel(capability string) (*RegisteredModel, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	var bestModel *RegisteredModel
	var bestScore float64

	for _, model := range mr.objects {
		if model.Status != ModelStatusReady {
			continue
		}

		// Check if model has the required capability
		hasCapability := false
		for _, cap := range model.Capabilities {
			if cap == capability {
				hasCapability = true
				break
			}
		}

		if !hasCapability {
			continue
		}

		// Calculate performance score
		score := mr.calculatePerformanceScore(model)

		if bestModel == nil || score > bestScore {
			bestModel = model
			bestScore = score
		}
	}

	if bestModel == nil {
		return nil, fmt.Errorf("no suitable model found for capability %s", capability)
	}

	// Return a copy
	modelCopy := *bestModel
	return &modelCopy, nil
}

// evictLeastUsedModel evicts the least recently used model
func (mr *ModelRegistry) evictLeastUsedModel() error {
	var lruModel *RegisteredModel
	var lruTime time.Time

	for _, model := range mr.objects {
		if model.Status == ModelStatusReady || model.Status == ModelStatusRegistered {
			if lruModel == nil || model.LastUsed.Before(lruTime) {
				lruModel = model
				lruTime = model.LastUsed
			}
		}
	}

	if lruModel == nil {
		return fmt.Errorf("no objects available for eviction")
	}

	return mr.UnregisterModel(lruModel.Name)
}

// determineCapabilities determines model capabilities based on provider
func (mr *ModelRegistry) determineCapabilities(provider string) []string {
	switch provider {
	case "cerebras":
		return []string{"text-generation", "code-generation", "reasoning"}
	case "gemini":
		return []string{"text-generation", "multimodal", "long-context"}
	case "deepseek":
		return []string{"text-generation", "code-generation", "reasoning", "math"}
	default:
		return []string{"text-generation"}
	}
}

// updateHealthScore updates the health score of a model
func (mr *ModelRegistry) updateHealthScore(model *RegisteredModel) {
	// Calculate health score based on error rate and latency
	errorPenalty := model.ErrorRate * 0.5
	latencyPenalty := 0.0

	// Penalize high latency (>5 seconds)
	if model.AvgLatency > 5*time.Second {
		latencyPenalty = 0.3
	} else if model.AvgLatency > 2*time.Second {
		latencyPenalty = 0.1
	}

	model.HealthScore = 1.0 - errorPenalty - latencyPenalty
	if model.HealthScore < 0 {
		model.HealthScore = 0
	}

	model.LastHealthCheck = time.Now()
}

// calculatePerformanceScore calculates a performance score for model selection
func (mr *ModelRegistry) calculatePerformanceScore(model *RegisteredModel) float64 {
	// Base score from health
	score := model.HealthScore

	// Bonus for recent usage
	timeSinceLastUse := time.Since(model.LastUsed)
	if timeSinceLastUse < time.Hour {
		score += 0.2
	}

	// Bonus for low latency
	if model.AvgLatency < time.Second {
		score += 0.1
	}

	// Penalty for high error rate
	score -= model.ErrorRate * 0.3

	return score
}

// Cache methods

// Add adds a model to the cache
func (mc *ModelCache) Add(modelName string, size int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Check if already exists
	if entry, exists := mc.entries[modelName]; exists {
		mc.moveToHead(entry)
		entry.LastAccess = time.Now()
		entry.AccessCount++
		return
	}

	// Evict if necessary
	for mc.currentSize+size > mc.maxSize && len(mc.entries) > 0 {
		mc.evictLRU()
	}

	// Create new entry
	entry := &CacheEntry{
		ModelName:   modelName,
		LoadedAt:    time.Now(),
		LastAccess:  time.Now(),
		AccessCount: 1,
		Size:        size,
	}

	mc.entries[modelName] = entry
	mc.addToHead(entry)
	mc.currentSize += size
}

// Access marks a model as accessed
func (mc *ModelCache) Access(modelName string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if entry, exists := mc.entries[modelName]; exists {
		mc.moveToHead(entry)
		entry.LastAccess = time.Now()
		entry.AccessCount++
	}
}

// Remove removes a model from the cache
func (mc *ModelCache) Remove(modelName string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if entry, exists := mc.entries[modelName]; exists {
		mc.removeEntry(entry)
		delete(mc.entries, modelName)
		mc.currentSize -= entry.Size
	}
}

// evictLRU evicts the least recently used entry
func (mc *ModelCache) evictLRU() {
	if mc.tail.prev != mc.head {
		lru := mc.tail.prev
		mc.removeEntry(lru)
		delete(mc.entries, lru.ModelName)
		mc.currentSize -= lru.Size
	}
}

// addToHead adds entry to head of LRU list
func (mc *ModelCache) addToHead(entry *CacheEntry) {
	entry.prev = mc.head
	entry.next = mc.head.next
	mc.head.next.prev = entry
	mc.head.next = entry
}

// removeEntry removes entry from LRU list
func (mc *ModelCache) removeEntry(entry *CacheEntry) {
	entry.prev.next = entry.next
	entry.next.prev = entry.prev
}

// moveToHead moves entry to head of LRU list
func (mc *ModelCache) moveToHead(entry *CacheEntry) {
	mc.removeEntry(entry)
	mc.addToHead(entry)
}
