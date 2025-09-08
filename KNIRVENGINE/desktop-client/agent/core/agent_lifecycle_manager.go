package core

import (
	"sync"
)

// AgentLifecycleManagerImpl implements the AgentLifecycleManager interface
type AgentLifecycleManagerImpl struct {
	createdHooks []AgentLifecycleHook
	updatedHooks []AgentLifecycleHook
	deletedHooks []func(id string)
	mutex        sync.RWMutex
}

// NewAgentLifecycleManager creates a new agent lifecycle manager
func NewAgentLifecycleManager() *AgentLifecycleManagerImpl {
	return &AgentLifecycleManagerImpl{
		createdHooks: make([]AgentLifecycleHook, 0),
		updatedHooks: make([]AgentLifecycleHook, 0),
		deletedHooks: make([]func(id string), 0),
	}
}

// RegisterCreatedHook registers a hook for agent creation events
func (m *AgentLifecycleManagerImpl) RegisterCreatedHook(hook AgentLifecycleHook) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.createdHooks = append(m.createdHooks, hook)
}

// RegisterUpdatedHook registers a hook for agent update events
func (m *AgentLifecycleManagerImpl) RegisterUpdatedHook(hook AgentLifecycleHook) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.updatedHooks = append(m.updatedHooks, hook)
}

// RegisterDeletedHook registers a hook for agent deletion events
func (m *AgentLifecycleManagerImpl) RegisterDeletedHook(hook func(id string)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.deletedHooks = append(m.deletedHooks, hook)
}

// TriggerCreated triggers all registered creation hooks
func (m *AgentLifecycleManagerImpl) TriggerCreated(agent *UnifiedAgent) {
	m.mutex.RLock()
	hooks := make([]AgentLifecycleHook, len(m.createdHooks))
	copy(hooks, m.createdHooks)
	m.mutex.RUnlock()

	// Execute hooks in separate goroutines to avoid blocking
	for _, hook := range hooks {
		go func(h AgentLifecycleHook) {
			defer func() {
				if r := recover(); r != nil {
					// Log the panic but don't crash the application
					// In a real implementation, this would use a proper logger
				}
			}()
			h(agent)
		}(hook)
	}
}

// TriggerUpdated triggers all registered update hooks
func (m *AgentLifecycleManagerImpl) TriggerUpdated(agent *UnifiedAgent) {
	m.mutex.RLock()
	hooks := make([]AgentLifecycleHook, len(m.updatedHooks))
	copy(hooks, m.updatedHooks)
	m.mutex.RUnlock()

	// Execute hooks in separate goroutines to avoid blocking
	for _, hook := range hooks {
		go func(h AgentLifecycleHook) {
			defer func() {
				if r := recover(); r != nil {
					// Log the panic but don't crash the application
					// In a real implementation, this would use a proper logger
				}
			}()
			h(agent)
		}(hook)
	}
}

// TriggerDeleted triggers all registered deletion hooks
func (m *AgentLifecycleManagerImpl) TriggerDeleted(id string) {
	m.mutex.RLock()
	hooks := make([]func(id string), len(m.deletedHooks))
	copy(hooks, m.deletedHooks)
	m.mutex.RUnlock()

	// Execute hooks in separate goroutines to avoid blocking
	for _, hook := range hooks {
		go func(h func(id string)) {
			defer func() {
				if r := recover(); r != nil {
					// Log the panic but don't crash the application
					// In a real implementation, this would use a proper logger
				}
			}()
			h(id)
		}(hook)
	}
}
