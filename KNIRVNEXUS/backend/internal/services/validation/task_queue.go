package validation

import (
	"sort"

	"nexus-backend/internal/models"
)

// AddTask adds a task to the queue
func (tq *TaskQueue) AddTask(task *models.ValidationTask) {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	tq.tasks[task.ID] = task
}

// RemoveTask removes a task from the queue
func (tq *TaskQueue) RemoveTask(taskID string) {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	delete(tq.tasks, taskID)
}

// GetTask retrieves a specific task
func (tq *TaskQueue) GetTask(taskID string) (*models.ValidationTask, bool) {
	tq.mu.RLock()
	defer tq.mu.RUnlock()
	task, exists := tq.tasks[taskID]
	return task, exists
}

// GetPendingTasks returns all pending tasks sorted by priority
func (tq *TaskQueue) GetPendingTasks() []*models.ValidationTask {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	var pendingTasks []*models.ValidationTask
	for _, task := range tq.tasks {
		if task.Status == "pending" {
			pendingTasks = append(pendingTasks, task)
		}
	}

	// Sort by priority (higher priority first)
	sort.Slice(pendingTasks, func(i, j int) bool {
		return pendingTasks[i].Priority > pendingTasks[j].Priority
	})

	return pendingTasks
}

// GetAllTasks returns all tasks in the queue
func (tq *TaskQueue) GetAllTasks() []*models.ValidationTask {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	tasks := make([]*models.ValidationTask, 0, len(tq.tasks))
	for _, task := range tq.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// GetTasksByStatus returns tasks filtered by status
func (tq *TaskQueue) GetTasksByStatus(status string) []*models.ValidationTask {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	var filteredTasks []*models.ValidationTask
	for _, task := range tq.tasks {
		if task.Status == status {
			filteredTasks = append(filteredTasks, task)
		}
	}
	return filteredTasks
}

// GetTasksByType returns tasks filtered by type
func (tq *TaskQueue) GetTasksByType(taskType string) []*models.ValidationTask {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	var filteredTasks []*models.ValidationTask
	for _, task := range tq.tasks {
		if task.Type == taskType {
			filteredTasks = append(filteredTasks, task)
		}
	}
	return filteredTasks
}

// GetTasksByPriority returns tasks with priority >= minPriority
func (tq *TaskQueue) GetTasksByPriority(minPriority int) []*models.ValidationTask {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	var filteredTasks []*models.ValidationTask
	for _, task := range tq.tasks {
		if task.Priority >= minPriority {
			filteredTasks = append(filteredTasks, task)
		}
	}

	// Sort by priority (higher priority first)
	sort.Slice(filteredTasks, func(i, j int) bool {
		return filteredTasks[i].Priority > filteredTasks[j].Priority
	})

	return filteredTasks
}

// UpdateTaskStatus updates the status of a task
func (tq *TaskQueue) UpdateTaskStatus(taskID, status string) bool {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if task, exists := tq.tasks[taskID]; exists {
		task.Status = status
		return true
	}
	return false
}

// GetQueueStats returns statistics about the task queue
func (tq *TaskQueue) GetQueueStats() map[string]int {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	stats := map[string]int{
		"total":     0,
		"pending":   0,
		"assigned":  0,
		"running":   0,
		"completed": 0,
		"failed":    0,
	}

	for _, task := range tq.tasks {
		stats["total"]++
		stats[task.Status]++
	}

	return stats
}

// GetTaskTypeDistribution returns distribution of task types
func (tq *TaskQueue) GetTaskTypeDistribution() map[string]int {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	distribution := make(map[string]int)
	for _, task := range tq.tasks {
		distribution[task.Type]++
	}

	return distribution
}

// GetPriorityDistribution returns distribution of task priorities
func (tq *TaskQueue) GetPriorityDistribution() map[int]int {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	distribution := make(map[int]int)
	for _, task := range tq.tasks {
		distribution[task.Priority]++
	}

	return distribution
}

// GetNextTask returns the next task to be processed (highest priority pending)
func (tq *TaskQueue) GetNextTask() *models.ValidationTask {
	pendingTasks := tq.GetPendingTasks()
	if len(pendingTasks) > 0 {
		return pendingTasks[0]
	}
	return nil
}

// GetTaskCount returns the total number of tasks in the queue
func (tq *TaskQueue) GetTaskCount() int {
	tq.mu.RLock()
	defer tq.mu.RUnlock()
	return len(tq.tasks)
}

// GetPendingTaskCount returns the number of pending tasks
func (tq *TaskQueue) GetPendingTaskCount() int {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	count := 0
	for _, task := range tq.tasks {
		if task.Status == "pending" {
			count++
		}
	}
	return count
}

// GetRunningTaskCount returns the number of running tasks
func (tq *TaskQueue) GetRunningTaskCount() int {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	count := 0
	for _, task := range tq.tasks {
		if task.Status == "running" {
			count++
		}
	}
	return count
}

// Clear removes all tasks from the queue
func (tq *TaskQueue) Clear() {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	tq.tasks = make(map[string]*models.ValidationTask)
}

// HasTask checks if a task exists in the queue
func (tq *TaskQueue) HasTask(taskID string) bool {
	tq.mu.RLock()
	defer tq.mu.RUnlock()
	_, exists := tq.tasks[taskID]
	return exists
}

// GetTasksRequiringTEE returns tasks that require specific TEE types
func (tq *TaskQueue) GetTasksRequiringTEE(teeType string) []*models.ValidationTask {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	var filteredTasks []*models.ValidationTask
	for _, task := range tq.tasks {
		if task.RequiredTEEType == teeType {
			filteredTasks = append(filteredTasks, task)
		}
	}
	return filteredTasks
}

// GetTasksByRequestor returns tasks filtered by requestor
func (tq *TaskQueue) GetTasksByRequestor(requestor string) []*models.ValidationTask {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	var filteredTasks []*models.ValidationTask
	for _, task := range tq.tasks {
		if task.RequestedBy == requestor {
			filteredTasks = append(filteredTasks, task)
		}
	}
	return filteredTasks
}
