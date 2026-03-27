package tunnel

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Service manages the tunnel registry functionality
type Service struct {
	config              *Config
	registryManager     *RegistryManager
	tunnelManager       *TunnelManager
	handler             *Handler
	controlListener     *ControlListener
	publicRelayListener *PublicRelayListener
	stunServer          *STUNServer
	logger              *zap.Logger
	pruneTicker         *time.Ticker
	pruneDone           chan struct{}
	mu                  sync.Mutex
}

// NewService creates a new tunnel service
func NewService(config *Config, logger *zap.Logger) *Service {
	registryManager := NewRegistryManager(logger)
	tunnelManager := NewTunnelManager(logger)

	handler := NewHandler(registryManager, tunnelManager, config, logger)
	controlListener := NewControlListener(config.ControlListenerPort, registryManager, tunnelManager, config, logger)
	publicRelayListener := NewPublicRelayListener(config.PublicRelayPort, tunnelManager, config, logger)
	stunServer := NewSTUNServer(config.STUNPort, logger)

	return &Service{
		config:              config,
		registryManager:     registryManager,
		tunnelManager:       tunnelManager,
		handler:             handler,
		controlListener:     controlListener,
		publicRelayListener: publicRelayListener,
		stunServer:          stunServer,
		logger:              logger,
		pruneDone:           make(chan struct{}),
	}
}

// Start starts all tunnel registry services
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Starting tunnel registry service")

	// Start control listener
	if err := s.controlListener.Start(); err != nil {
		return err
	}

	// Start public relay listener
	if err := s.publicRelayListener.Start(); err != nil {
		s.controlListener.Stop()
		return err
	}

	// Start STUN server
	if err := s.stunServer.Start(); err != nil {
		s.controlListener.Stop()
		s.publicRelayListener.Stop()
		return err
	}

	// Start connection pruning
	s.startPruning()

	s.logger.Info("Tunnel registry service started successfully")
	return nil
}

// Stop stops all tunnel registry services
func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Stopping tunnel registry service")

	// Stop pruning
	s.stopPruning()

	// Stop services
	s.stunServer.Stop()
	s.publicRelayListener.Stop()
	s.controlListener.Stop()

	s.logger.Info("Tunnel registry service stopped")
	return nil
}

// GetHandler returns the HTTP handler for tunnel registry routes
func (s *Service) GetHandler() *Handler {
	return s.handler
}

// startPruning starts the periodic connection pruning
func (s *Service) startPruning() {
	s.pruneTicker = time.NewTicker(15 * time.Minute) // Prune every 15 minutes
	go func() {
		defer s.pruneTicker.Stop()
		for {
			select {
			case <-s.pruneTicker.C:
				s.pruneInactiveConnections()
			case <-s.pruneDone:
				return
			}
		}
	}()
}

// stopPruning stops the periodic connection pruning
func (s *Service) stopPruning() {
	if s.pruneTicker != nil {
		s.pruneTicker.Stop()
		close(s.pruneDone)
	}
}

// pruneInactiveConnections removes inactive connections
func (s *Service) pruneInactiveConnections() {
	now := time.Now()
	maxInactiveTime := 60 * time.Minute // 1 hour
	prunedCount := 0
	skippedCount := 0
	activeCount := 0

	s.logger.Info("Running inactive connection pruning")

	connectionRegistryMu.Lock()
	defer connectionRegistryMu.Unlock()

	// Identify connections to prune
	var connectionsToPrune []string
	for connectionID, info := range connectionRegistry {
		timeSinceLastSeen := now.Sub(info.LastSeen)

		if timeSinceLastSeen > maxInactiveTime {
			// Check if this is a critical connection that should be preserved
			if info.Type == "bootnode" || info.Type == "root" {
				s.logger.Info("Skipping pruning of critical connection",
					zap.String("connectionId", connectionID),
					zap.String("type", info.Type),
					zap.Duration("inactiveTime", timeSinceLastSeen))

				skippedCount++

				// Try to ping the connection to see if it's still alive
				if info.Socket != nil && !isSocketClosed(info.Socket) {
					// Send PING_CHECK if possible (this would need to be implemented per connection type)
					s.logger.Info("Critical connection still has active socket",
						zap.String("connectionId", connectionID))
				} else {
					s.logger.Info("Critical connection socket is closed, marking for pruning",
						zap.String("connectionId", connectionID))
					connectionsToPrune = append(connectionsToPrune, connectionID)
				}
			} else {
				// Regular connection that can be pruned
				connectionsToPrune = append(connectionsToPrune, connectionID)
			}
		} else {
			activeCount++
		}
	}

	// Prune the identified connections
	for _, connectionID := range connectionsToPrune {
		info := connectionRegistry[connectionID]
		s.logger.Info("Pruning inactive connection",
			zap.String("connectionId", connectionID),
			zap.String("type", info.Type),
			zap.Time("lastSeen", info.LastSeen))

		// Close any active sockets
		if info.Socket != nil && !isSocketClosed(info.Socket) {
			info.Socket.Close()
		}

		// Remove from tunnel manager if applicable
		if info.Type == "dev" {
			s.tunnelManager.RemoveControlSocket(connectionID, "")
		}

		// Remove from registry
		delete(connectionRegistry, connectionID)
		prunedCount++
	}

	s.logger.Info("Connection pruning completed",
		zap.Int("active", activeCount),
		zap.Int("pruned", prunedCount),
		zap.Int("skipped", skippedCount))
}
