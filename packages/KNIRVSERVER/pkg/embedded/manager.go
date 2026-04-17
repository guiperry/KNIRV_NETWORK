package embedded

import (
	"context"
	"fmt"
	"sync"

	"knirv-server/pkg/embedded/graphrag"
	"knirv-server/pkg/embedded/transactionchain"
	"knirv-server/pkg/embedded/validationchain"
)

// EmbeddedService defines the interface for all embedded services
type EmbeddedService interface {
	Name() string
	Init(config []byte) error
	Start(ctx context.Context) error
	Stop() error
	Health() error
}

// Manager handles lifecycle of all embedded services
type Manager struct {
	services    []EmbeddedService
	ctx         context.Context
	cancelFunc  context.CancelFunc
	initialized bool
	mu          sync.Mutex
}

var (
	globalManager *Manager
	managerOnce   sync.Once
)

// GetManager returns the global embedded service manager
func GetManager() *Manager {
	managerOnce.Do(func() {
		globalManager = &Manager{
			services: make([]EmbeddedService, 0, 3),
		}
	})
	return globalManager
}

// GraphRagService wraps graphrag CGo library
type GraphRagService struct {
	config []byte
}

// Name returns service identifier
func (s *GraphRagService) Name() string { return "graphrag" }

// Init initializes graphrag engine
func (s *GraphRagService) Init(config []byte) error {
	s.config = config
	return graphrag.Init(config)
}

// Start is no-op for static libraries
func (s *GraphRagService) Start(ctx context.Context) error { return nil }

// Stop shuts down graphrag engine
func (s *GraphRagService) Stop() error { return graphrag.Shutdown() }

// Health checks graphrag health status
func (s *GraphRagService) Health() error { return graphrag.HealthCheck() }

// ValidationChainService wraps validation_chain CGo library
type ValidationChainService struct {
	config []byte
}

// Name returns service identifier
func (s *ValidationChainService) Name() string { return "validation_chain" }

// Init initializes validation chain engine
func (s *ValidationChainService) Init(config []byte) error {
	s.config = config
	return validationchain.Init(config)
}

// Start is no-op for static libraries
func (s *ValidationChainService) Start(ctx context.Context) error { return nil }

// Stop shuts down validation chain
func (s *ValidationChainService) Stop() error { return validationchain.Shutdown() }

// Health checks validation chain health status
func (s *ValidationChainService) Health() error { return validationchain.HealthCheck() }

// TransactionChainService wraps transaction_chain Node.js process
type TransactionChainService struct {
	port int
}

// Name returns service identifier
func (s *TransactionChainService) Name() string { return "transaction_chain" }

// Init stores configuration
func (s *TransactionChainService) Init(config []byte) error {
	// For transaction chain port is passed via Start
	return nil
}

// Start launches transaction chain process
func (s *TransactionChainService) Start(ctx context.Context) error {
	return transactionchain.Get().Start(ctx, s.port)
}

// Stop stops transaction chain process
func (s *TransactionChainService) Stop() error {
	return transactionchain.Get().Stop()
}

// Health checks transaction chain health status
func (s *TransactionChainService) Health() error {
	return transactionchain.Get().HealthCheck()
}

// Initialize sets up all embedded services
func (m *Manager) Initialize(ctx context.Context, graphRagConfig, validationChainConfig []byte, txPort int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		return fmt.Errorf("embedded manager already initialized")
	}

	m.ctx, m.cancelFunc = context.WithCancel(ctx)

	// 1. Initialize GraphRag
	graphSvc := &GraphRagService{}
	if err := graphSvc.Init(graphRagConfig); err != nil {
		return fmt.Errorf("failed to initialize graphrag: %w", err)
	}
	m.services = append(m.services, graphSvc)

	// 2. Initialize Validation Chain
	validationSvc := &ValidationChainService{}
	if err := validationSvc.Init(validationChainConfig); err != nil {
		return fmt.Errorf("failed to initialize validation chain: %w", err)
	}
	m.services = append(m.services, validationSvc)

	// 3. Initialize Transaction Chain
	txSvc := &TransactionChainService{port: txPort}
	if err := txSvc.Init(nil); err != nil {
		return fmt.Errorf("failed to initialize transaction chain: %w", err)
	}
	if err := txSvc.Start(m.ctx); err != nil {
		return fmt.Errorf("failed to start transaction chain: %w", err)
	}
	m.services = append(m.services, txSvc)

	m.initialized = true
	return nil
}

// Shutdown stops all embedded services in reverse order
func (m *Manager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return nil
	}

	m.cancelFunc()

	var firstErr error
	for i := len(m.services) - 1; i >= 0; i-- {
		if err := m.services[i].Stop(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	m.initialized = false
	m.services = nil

	return firstErr
}

// Health returns status of all services
func (m *Manager) Health() map[string]error {
	m.mu.Lock()
	defer m.mu.Unlock()

	health := make(map[string]error, len(m.services))
	for _, svc := range m.services {
		health[svc.Name()] = svc.Health()
	}

	return health
}

// IsInitialized returns true if manager is properly initialized
func (m *Manager) IsInitialized() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.initialized
}
