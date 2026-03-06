package fabricserver

import (
	"fmt"
	"sort"
	"time"
)

// NewFabricScheduler creates a new fabric scheduler
func NewFabricScheduler() (*FabricScheduler, error) {
	return &FabricScheduler{
		schedulingPolicy: "resource-aware",
		queue:            make([]*FabricScheduleRequest, 0),
		running:          false,
	}, nil
}

// Start starts the scheduler
func (as *FabricScheduler) Start() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.running {
		return fmt.Errorf("scheduler is already running")
	}

	as.running = true
	return nil
}

// Stop stops the scheduler
func (as *FabricScheduler) Stop() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	as.running = false
	return nil
}

// ScheduleFabric adds a fabric item to the scheduling queue
func (as *FabricScheduler) ScheduleFabric(request *FabricScheduleRequest) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if !as.running {
		return fmt.Errorf("scheduler is not running")
	}

	// Add to queue
	as.queue = append(as.queue, request)

	// Sort queue based on scheduling policy
	as.sortQueue()

	return nil
}

// ProcessQueue processes the scheduling queue
func (as *FabricScheduler) ProcessQueue() {
	as.mu.Lock()
	defer as.mu.Unlock()

	if !as.running || len(as.queue) == 0 {
		return
	}

	// Process requests based on scheduling policy
	switch as.schedulingPolicy {
	case "round-robin":
		as.processRoundRobin()
	case "resource-aware":
		as.processResourceAware()
	case "priority":
		as.processPriority()
	default:
		as.processResourceAware() // Default to resource-aware
	}
}

// sortQueue sorts the queue based on the scheduling policy
func (as *FabricScheduler) sortQueue() {
	switch as.schedulingPolicy {
	case "priority":
		// Sort by priority (higher priority first)
		sort.Slice(as.queue, func(i, j int) bool {
			return as.queue[i].Priority > as.queue[j].Priority
		})
	case "resource-aware":
		// Sort by resource requirements (smaller requirements first)
		sort.Slice(as.queue, func(i, j int) bool {
			reqI := as.calculateResourceScore(as.queue[i].Resources)
			reqJ := as.calculateResourceScore(as.queue[j].Resources)
			return reqI < reqJ
		})
	case "round-robin":
		// Sort by request time (FIFO)
		sort.Slice(as.queue, func(i, j int) bool {
			return as.queue[i].RequestTime.Before(as.queue[j].RequestTime)
		})
	}
}

// calculateResourceScore calculates a score for resource requirements
func (as *FabricScheduler) calculateResourceScore(resources *ResourceAllocation) float64 {
	if resources == nil {
		return 0
	}

	// Simple scoring: normalize and sum all resource requirements
	// This is a simplified approach - real implementations would be more sophisticated
	cpuScore := resources.CPUCores / 8.0                                   // Assume max 8 cores
	memScore := float64(resources.MemoryBytes) / (8 * 1024 * 1024 * 1024)  // Assume max 8GB
	diskScore := float64(resources.DiskBytes) / (100 * 1024 * 1024 * 1024) // Assume max 100GB

	return cpuScore + memScore + diskScore
}

// processRoundRobin processes queue in round-robin fashion
func (as *FabricScheduler) processRoundRobin() {
	// In round-robin, we just process in FIFO order
	// This is a simplified implementation
	if len(as.queue) > 0 {
		// Remove the first request (would be processed by runtime manager)
		as.queue = as.queue[1:]
	}
}

// processResourceAware processes queue based on resource availability
func (as *FabricScheduler) processResourceAware() {
	// This would integrate with the resource pool to check availability
	// For now, just process the first request that can fit

	for i, request := range as.queue {
		// Check if resources are available (simplified)
		if as.canSchedule(request) {
			// Remove from queue (would be processed by runtime manager)
			as.queue = append(as.queue[:i], as.queue[i+1:]...)
			break
		}
	}
}

// processPriority processes queue based on priority
func (as *FabricScheduler) processPriority() {
	// Process highest priority first
	if len(as.queue) > 0 {
		// Remove the first request (highest priority due to sorting)
		as.queue = as.queue[1:]
	}
}

// canSchedule checks if a request can be scheduled
func (as *FabricScheduler) canSchedule(request *FabricScheduleRequest) bool {
	// This is a simplified check
	// In a real implementation, this would check with the resource pool
	return request.Resources != nil &&
		request.Resources.CPUCores > 0 &&
		request.Resources.MemoryBytes > 0
}

// GetQueueStatus returns the current queue status
func (as *FabricScheduler) GetQueueStatus() map[string]interface{} {
	as.mu.RLock()
	defer as.mu.RUnlock()

	queueInfo := make([]map[string]interface{}, len(as.queue))
	for i, request := range as.queue {
		queueInfo[i] = map[string]interface{}{
			"fabric_name":  request.FabricName,
			"binary":       request.Binary,
			"priority":     request.Priority,
			"request_time": request.RequestTime,
			"resources": map[string]interface{}{
				"cpu_cores":    request.Resources.CPUCores,
				"memory_bytes": request.Resources.MemoryBytes,
				"disk_bytes":   request.Resources.DiskBytes,
			},
		}
	}

	return map[string]interface{}{
		"policy":     as.schedulingPolicy,
		"queue_size": len(as.queue),
		"running":    as.running,
		"queue":      queueInfo,
	}
}

// SetSchedulingPolicy sets the scheduling policy
func (as *FabricScheduler) SetSchedulingPolicy(policy string) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	validPolicies := []string{"round-robin", "resource-aware", "priority"}
	for _, validPolicy := range validPolicies {
		if policy == validPolicy {
			as.schedulingPolicy = policy
			as.sortQueue() // Re-sort queue with new policy
			return nil
		}
	}

	return fmt.Errorf("invalid scheduling policy: %s", policy)
}

// GetSchedulingPolicy returns the current scheduling policy
func (as *FabricScheduler) GetSchedulingPolicy() string {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.schedulingPolicy
}

// ClearQueue clears the scheduling queue
func (as *FabricScheduler) ClearQueue() {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.queue = make([]*FabricScheduleRequest, 0)
}

// RemoveFromQueue removes a specific request from the queue
func (as *FabricScheduler) RemoveFromQueue(fabricName string) bool {
	as.mu.Lock()
	defer as.mu.Unlock()

	for i, request := range as.queue {
		if request.FabricName == fabricName {
			as.queue = append(as.queue[:i], as.queue[i+1:]...)
			return true
		}
	}

	return false
}

// GetQueueSize returns the current queue size
func (as *FabricScheduler) GetQueueSize() int {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return len(as.queue)
}

// IsRunning returns whether the scheduler is running
func (as *FabricScheduler) IsRunning() bool {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.running
}

// NewFabricScheduleRequest creates a new schedule request
func NewFabricScheduleRequest(fabricName, binary string, resources *ResourceAllocation, config map[string]interface{}, priority int) *FabricScheduleRequest {
	return &FabricScheduleRequest{
		FabricName:  fabricName,
		Binary:      binary,
		Resources:   resources,
		Config:      config,
		Priority:    priority,
		RequestTime: time.Now(),
	}
}

// ScheduleFabricWithDefaults schedules a fabric unit with default resource allocation
func (as *FabricScheduler) ScheduleFabricWithDefaults(fabricName, binary string, config map[string]interface{}) error {
	defaultResources := &ResourceAllocation{
		CPUCores:    0.5,
		MemoryBytes: 256 * 1024 * 1024,  // 256MB
		DiskBytes:   1024 * 1024 * 1024, // 1GB
	}

	request := NewFabricScheduleRequest(fabricName, binary, defaultResources, config, 0)
	return as.ScheduleFabric(request)
}

// ScheduleFabricWithPriority schedules a fabric item with specific priority
func (as *FabricScheduler) ScheduleFabricWithPriority(fabricName, binary string, resources *ResourceAllocation, config map[string]interface{}, priority int) error {
	request := NewFabricScheduleRequest(fabricName, binary, resources, config, priority)
	return as.ScheduleFabric(request)
}
