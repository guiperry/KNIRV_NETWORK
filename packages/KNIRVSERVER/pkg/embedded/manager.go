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

// ValidationChainService wraps the embedded Validation Chain binary,
// spawned and monitored entirely within pkg/embedded/validationchain, bound
// to a Unix domain socket. KNIRVSERVER is the sole owner of this process's
// lifecycle; backend_server (KNIRV_CORP) only holds an HTTP client dialing
// the same socket — nothing exposes a TCP port for it.
type ValidationChainService struct {
	socketPath string
}

// Name returns service identifier
func (s *ValidationChainService) Name() string { return "validation_chain" }

// Init stores configuration; the socket path is applied on Start.
func (s *ValidationChainService) Init(config []byte) error { return nil }

// Start extracts and launches the validation chain subprocess.
func (s *ValidationChainService) Start(ctx context.Context) error {
	return validationchain.Get().Start(ctx, s.socketPath)
}

// Stop stops the validation chain subprocess.
func (s *ValidationChainService) Stop() error { return validationchain.Get().Stop() }

// Health checks validation chain health status.
func (s *ValidationChainService) Health() error { return validationchain.Get().HealthCheck() }

// TransactionChainService wraps the embedded Transaction Chain Node.js
// process, spawned and monitored entirely within
// pkg/embedded/transactionchain, bound to a Unix domain socket. KNIRVSERVER
// is the sole owner of this process's lifecycle; backend_server (KNIRV_CORP)
// only holds an HTTP client dialing the same socket — nothing exposes a TCP
// port for it.
type TransactionChainService struct {
	socketPath string
}

// Name returns service identifier
func (s *TransactionChainService) Name() string { return "transaction_chain" }

// Init stores configuration; the socket path is applied on Start.
func (s *TransactionChainService) Init(config []byte) error { return nil }

// Start launches transaction chain process
func (s *TransactionChainService) Start(ctx context.Context) error {
	return transactionchain.Get().Start(ctx, s.socketPath)
}

// Stop stops transaction chain process
func (s *TransactionChainService) Stop() error {
	return transactionchain.Get().Stop()
}

// Health checks transaction chain health status
func (s *TransactionChainService) Health() error {
	return transactionchain.Get().HealthCheck()
}

// Initialize sets up the always-on embedded services (currently GraphRag).
// Call StartValidationChain / StartTransactionChain separately — each is
// independently best-effort (a failure in one must not prevent the other,
// or graphrag, from running), so they are not folded into this call.
func (m *Manager) Initialize(ctx context.Context, graphRagConfig []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		return fmt.Errorf("embedded manager already initialized")
	}

	m.ctx, m.cancelFunc = context.WithCancel(ctx)

	graphSvc := &GraphRagService{}
	if err := graphSvc.Init(graphRagConfig); err != nil {
		return fmt.Errorf("failed to initialize graphrag: %w", err)
	}
	m.services = append(m.services, graphSvc)

	m.initialized = true
	return nil
}

// StartValidationChain starts the embedded Validation Chain subprocess
// bound to socketPath and registers it for Health()/Shutdown() tracking.
// Must be called after Initialize.
func (m *Manager) StartValidationChain(ctx context.Context, socketPath string) error {
	svc := &ValidationChainService{socketPath: socketPath}
	if err := svc.Start(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	m.services = append(m.services, svc)
	m.mu.Unlock()
	return nil
}

// StartTransactionChain starts the embedded Transaction Chain subprocess
// bound to socketPath and registers it for Health()/Shutdown() tracking.
// Must be called after Initialize.
func (m *Manager) StartTransactionChain(ctx context.Context, socketPath string) error {
	svc := &TransactionChainService{socketPath: socketPath}
	if err := svc.Start(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	m.services = append(m.services, svc)
	m.mu.Unlock()
	return nil
}

// Shutdown stops all registered services in reverse order, regardless of
// whether Initialize (graphrag) was ever called — StartValidationChain and
// StartTransactionChain register services independently of Initialize, so
// Shutdown must not skip them just because graphrag was never started.
func (m *Manager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancelFunc != nil {
		m.cancelFunc()
	}

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
