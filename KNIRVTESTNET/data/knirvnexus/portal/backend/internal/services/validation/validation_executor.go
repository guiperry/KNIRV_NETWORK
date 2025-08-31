package validation

import "time"

// AddExecution adds a running execution
func (ve *ValidationExecutor) AddExecution(taskID string, execution *ValidationExecution) {
	ve.mu.Lock()
	defer ve.mu.Unlock()
	ve.running[taskID] = execution
}

// RemoveExecution removes a running execution
func (ve *ValidationExecutor) RemoveExecution(taskID string) {
	ve.mu.Lock()
	defer ve.mu.Unlock()
	delete(ve.running, taskID)
}

// GetExecution retrieves a specific execution
func (ve *ValidationExecutor) GetExecution(taskID string) (*ValidationExecution, bool) {
	ve.mu.RLock()
	defer ve.mu.RUnlock()
	execution, exists := ve.running[taskID]
	return execution, exists
}

// IsRunning checks if a task is currently running
func (ve *ValidationExecutor) IsRunning(taskID string) bool {
	ve.mu.RLock()
	defer ve.mu.RUnlock()
	_, exists := ve.running[taskID]
	return exists
}

// GetRunningCount returns the number of currently running executions
func (ve *ValidationExecutor) GetRunningCount() int {
	ve.mu.RLock()
	defer ve.mu.RUnlock()
	return len(ve.running)
}

// GetRunningExecutions returns all currently running executions
func (ve *ValidationExecutor) GetRunningExecutions() []*ValidationExecution {
	ve.mu.RLock()
	defer ve.mu.RUnlock()

	executions := make([]*ValidationExecution, 0, len(ve.running))
	for _, execution := range ve.running {
		executions = append(executions, execution)
	}
	return executions
}

// GetRunningTaskIDs returns the IDs of all currently running tasks
func (ve *ValidationExecutor) GetRunningTaskIDs() []string {
	ve.mu.RLock()
	defer ve.mu.RUnlock()

	taskIDs := make([]string, 0, len(ve.running))
	for taskID := range ve.running {
		taskIDs = append(taskIDs, taskID)
	}
	return taskIDs
}

// CanExecuteMore checks if more executions can be started
func (ve *ValidationExecutor) CanExecuteMore() bool {
	ve.mu.RLock()
	defer ve.mu.RUnlock()
	return len(ve.running) < ve.maxConcurrent
}

// GetAvailableSlots returns the number of available execution slots
func (ve *ValidationExecutor) GetAvailableSlots() int {
	ve.mu.RLock()
	defer ve.mu.RUnlock()
	available := ve.maxConcurrent - len(ve.running)
	if available < 0 {
		return 0
	}
	return available
}

// CancelExecution cancels a specific execution
func (ve *ValidationExecutor) CancelExecution(taskID string) bool {
	ve.mu.Lock()
	defer ve.mu.Unlock()

	if execution, exists := ve.running[taskID]; exists {
		execution.Cancel()
		delete(ve.running, taskID)
		return true
	}
	return false
}

// CancelAll cancels all running executions
func (ve *ValidationExecutor) CancelAll() {
	ve.mu.Lock()
	defer ve.mu.Unlock()

	for taskID, execution := range ve.running {
		execution.Cancel()
		delete(ve.running, taskID)
	}
}

// SetMaxConcurrent sets the maximum number of concurrent executions
func (ve *ValidationExecutor) SetMaxConcurrent(max int) {
	ve.mu.Lock()
	defer ve.mu.Unlock()
	ve.maxConcurrent = max
}

// GetMaxConcurrent returns the maximum number of concurrent executions
func (ve *ValidationExecutor) GetMaxConcurrent() int {
	ve.mu.RLock()
	defer ve.mu.RUnlock()
	return ve.maxConcurrent
}

// GetExecutorStats returns statistics about the executor
func (ve *ValidationExecutor) GetExecutorStats() map[string]interface{} {
	ve.mu.RLock()
	defer ve.mu.RUnlock()

	return map[string]interface{}{
		"max_concurrent":    ve.maxConcurrent,
		"currently_running": len(ve.running),
		"available_slots":   ve.maxConcurrent - len(ve.running),
		"utilization_rate":  float64(len(ve.running)) / float64(ve.maxConcurrent),
	}
}

// GetExecutionsByType returns executions grouped by task type
func (ve *ValidationExecutor) GetExecutionsByType() map[string][]*ValidationExecution {
	ve.mu.RLock()
	defer ve.mu.RUnlock()

	byType := make(map[string][]*ValidationExecution)
	for _, execution := range ve.running {
		taskType := execution.Task.Type
		byType[taskType] = append(byType[taskType], execution)
	}
	return byType
}

// GetExecutionsByPriority returns executions grouped by task priority
func (ve *ValidationExecutor) GetExecutionsByPriority() map[int][]*ValidationExecution {
	ve.mu.RLock()
	defer ve.mu.RUnlock()

	byPriority := make(map[int][]*ValidationExecution)
	for _, execution := range ve.running {
		priority := execution.Task.Priority
		byPriority[priority] = append(byPriority[priority], execution)
	}
	return byPriority
}

// GetLongestRunningExecution returns the execution that has been running the longest
func (ve *ValidationExecutor) GetLongestRunningExecution() *ValidationExecution {
	ve.mu.RLock()
	defer ve.mu.RUnlock()

	var longest *ValidationExecution
	for _, execution := range ve.running {
		if longest == nil || execution.StartTime.Before(longest.StartTime) {
			longest = execution
		}
	}
	return longest
}

// GetExecutionsOlderThan returns executions that have been running longer than the specified duration
func (ve *ValidationExecutor) GetExecutionsOlderThan(duration int64) []*ValidationExecution {
	ve.mu.RLock()
	defer ve.mu.RUnlock()

	var oldExecutions []*ValidationExecution
	cutoff := ve.getCurrentTime() - duration

	for _, execution := range ve.running {
		if execution.StartTime.Unix() < cutoff {
			oldExecutions = append(oldExecutions, execution)
		}
	}
	return oldExecutions
}

// getCurrentTime returns the current Unix timestamp (helper for testing)
func (ve *ValidationExecutor) getCurrentTime() int64 {
	return time.Now().Unix()
}

// HasCapacityForPriority checks if there's capacity for a task with given priority
func (ve *ValidationExecutor) HasCapacityForPriority(priority int) bool {
	ve.mu.RLock()
	defer ve.mu.RUnlock()

	// If we have available slots, we can always execute
	if len(ve.running) < ve.maxConcurrent {
		return true
	}

	// If at capacity, check if we can preempt lower priority tasks
	for _, execution := range ve.running {
		if execution.Task.Priority < priority {
			return true // We can preempt this lower priority task
		}
	}

	return false
}

// PreemptLowerPriorityTask preempts the lowest priority running task
func (ve *ValidationExecutor) PreemptLowerPriorityTask(minPriority int) *ValidationExecution {
	ve.mu.Lock()
	defer ve.mu.Unlock()

	var lowestPriorityExecution *ValidationExecution
	var lowestPriorityTaskID string

	for taskID, execution := range ve.running {
		if execution.Task.Priority < minPriority {
			if lowestPriorityExecution == nil || execution.Task.Priority < lowestPriorityExecution.Task.Priority {
				lowestPriorityExecution = execution
				lowestPriorityTaskID = taskID
			}
		}
	}

	if lowestPriorityExecution != nil {
		lowestPriorityExecution.Cancel()
		delete(ve.running, lowestPriorityTaskID)
		return lowestPriorityExecution
	}

	return nil
}

// GetExecutionDurations returns the duration of all running executions
func (ve *ValidationExecutor) GetExecutionDurations() map[string]int64 {
	ve.mu.RLock()
	defer ve.mu.RUnlock()

	durations := make(map[string]int64)
	currentTime := ve.getCurrentTime()

	for taskID, execution := range ve.running {
		duration := currentTime - execution.StartTime.Unix()
		durations[taskID] = duration
	}

	return durations
}

// IsAtCapacity checks if the executor is at maximum capacity
func (ve *ValidationExecutor) IsAtCapacity() bool {
	ve.mu.RLock()
	defer ve.mu.RUnlock()
	return len(ve.running) >= ve.maxConcurrent
}
