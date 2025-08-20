package dataengine

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Agent Management Methods

// CreateAgent creates a new agent
func (m *BuntDBManager) CreateAgent(agent *AgentEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()
	
	data, err := json.Marshal(agent)
	if err != nil {
		return err
	}
	
	return m.db.Update(func(tx *buntdb.Tx) error {
		key := AgentsCollection + agent.ID
		_, _, err := tx.Set(key, string(data), nil)
		return err
	})
}

// GetAgent retrieves an agent by ID
func (m *BuntDBManager) GetAgent(agentID string) (*AgentEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var agent AgentEntry
	err := m.db.View(func(tx *buntdb.Tx) error {
		key := AgentsCollection + agentID
		value, err := tx.Get(key)
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(value), &agent)
	})
	
	if err != nil {
		return nil, err
	}
	
	return &agent, nil
}

// UpdateAgent updates an existing agent
func (m *BuntDBManager) UpdateAgent(agent *AgentEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	agent.UpdatedAt = time.Now()
	
	data, err := json.Marshal(agent)
	if err != nil {
		return err
	}
	
	return m.db.Update(func(tx *buntdb.Tx) error {
		key := AgentsCollection + agent.ID
		_, _, err := tx.Set(key, string(data), nil)
		return err
	})
}

// ListAgentsByOwner lists all agents for a specific owner
func (m *BuntDBManager) ListAgentsByOwner(ownerID string) ([]*AgentEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var agents []*AgentEntry
	
	err := m.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			if !strings.HasPrefix(key, AgentsCollection) {
				return true
			}
			
			var agent AgentEntry
			if err := json.Unmarshal([]byte(value), &agent); err != nil {
				return true
			}
			
			if agent.OwnerID == ownerID {
				agents = append(agents, &agent)
			}
			
			return true
		})
	})
	
	return agents, err
}

// DVE Node Management Methods

// CreateDVENode creates a new DVE node
func (m *BuntDBManager) CreateDVENode(node *DVENodeEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	node.CreatedAt = time.Now()
	node.UpdatedAt = time.Now()
	
	data, err := json.Marshal(node)
	if err != nil {
		return err
	}
	
	return m.db.Update(func(tx *buntdb.Tx) error {
		key := DVENodesCollection + node.ID
		_, _, err := tx.Set(key, string(data), nil)
		return err
	})
}

// GetDVENode retrieves a DVE node by ID
func (m *BuntDBManager) GetDVENode(nodeID string) (*DVENodeEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var node DVENodeEntry
	err := m.db.View(func(tx *buntdb.Tx) error {
		key := DVENodesCollection + nodeID
		value, err := tx.Get(key)
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(value), &node)
	})
	
	if err != nil {
		return nil, err
	}
	
	return &node, nil
}

// UpdateDVENode updates an existing DVE node
func (m *BuntDBManager) UpdateDVENode(node *DVENodeEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	node.UpdatedAt = time.Now()
	
	data, err := json.Marshal(node)
	if err != nil {
		return err
	}
	
	return m.db.Update(func(tx *buntdb.Tx) error {
		key := DVENodesCollection + node.ID
		_, _, err := tx.Set(key, string(data), nil)
		return err
	})
}

// ListDVENodes lists all DVE nodes with optional status filter
func (m *BuntDBManager) ListDVENodes(status string) ([]*DVENodeEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var nodes []*DVENodeEntry
	
	err := m.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			if !strings.HasPrefix(key, DVENodesCollection) {
				return true
			}
			
			var node DVENodeEntry
			if err := json.Unmarshal([]byte(value), &node); err != nil {
				return true
			}
			
			if status == "" || node.Status == status {
				nodes = append(nodes, &node)
			}
			
			return true
		})
	})
	
	return nodes, err
}

// Validation Task Management Methods

// CreateValidationTask creates a new validation task
func (m *BuntDBManager) CreateValidationTask(task *ValidationTaskEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}
	
	return m.db.Update(func(tx *buntdb.Tx) error {
		key := ValidationTasksCollection + task.ID
		_, _, err := tx.Set(key, string(data), nil)
		return err
	})
}

// GetValidationTask retrieves a validation task by ID
func (m *BuntDBManager) GetValidationTask(taskID string) (*ValidationTaskEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var task ValidationTaskEntry
	err := m.db.View(func(tx *buntdb.Tx) error {
		key := ValidationTasksCollection + taskID
		value, err := tx.Get(key)
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(value), &task)
	})
	
	if err != nil {
		return nil, err
	}
	
	return &task, nil
}

// UpdateValidationTask updates an existing validation task
func (m *BuntDBManager) UpdateValidationTask(task *ValidationTaskEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	task.UpdatedAt = time.Now()
	
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}
	
	return m.db.Update(func(tx *buntdb.Tx) error {
		key := ValidationTasksCollection + task.ID
		_, _, err := tx.Set(key, string(data), nil)
		return err
	})
}

// ListValidationTasksByStatus lists validation tasks by status
func (m *BuntDBManager) ListValidationTasksByStatus(status string) ([]*ValidationTaskEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var tasks []*ValidationTaskEntry
	
	err := m.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			if !strings.HasPrefix(key, ValidationTasksCollection) {
				return true
			}
			
			var task ValidationTaskEntry
			if err := json.Unmarshal([]byte(value), &task); err != nil {
				return true
			}
			
			if task.Status == status {
				tasks = append(tasks, &task)
			}
			
			return true
		})
	})
	
	return tasks, err
}

// CreateValidationResult creates a new validation result
func (m *BuntDBManager) CreateValidationResult(result *ValidationResultEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	result.CreatedAt = time.Now()
	
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	
	return m.db.Update(func(tx *buntdb.Tx) error {
		key := ValidationResultsCollection + result.ID
		_, _, err := tx.Set(key, string(data), nil)
		return err
	})
}

// GetValidationResult retrieves a validation result by ID
func (m *BuntDBManager) GetValidationResult(resultID string) (*ValidationResultEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var result ValidationResultEntry
	err := m.db.View(func(tx *buntdb.Tx) error {
		key := ValidationResultsCollection + resultID
		value, err := tx.Get(key)
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(value), &result)
	})
	
	if err != nil {
		return nil, err
	}
	
	return &result, nil
}

// GetValidationResultsByTask retrieves all validation results for a task
func (m *BuntDBManager) GetValidationResultsByTask(taskID string) ([]*ValidationResultEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var results []*ValidationResultEntry
	
	err := m.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			if !strings.HasPrefix(key, ValidationResultsCollection) {
				return true
			}
			
			var result ValidationResultEntry
			if err := json.Unmarshal([]byte(value), &result); err != nil {
				return true
			}
			
			if result.TaskID == taskID {
				results = append(results, &result)
			}
			
			return true
		})
	})
	
	return results, err
}
