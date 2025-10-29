package container

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// PortAllocator manages port allocation for DVE containers
type PortAllocator struct {
	config       PortAllocationConfig
	allocations  map[string]*PortAllocation // rentalID -> allocation
	usedPorts    map[int]string             // port -> rentalID
	mutex        sync.RWMutex
	cleanupTimer *time.Ticker
}

// NewPortAllocator creates a new port allocator
func NewPortAllocator(config PortAllocationConfig) *PortAllocator {
	allocator := &PortAllocator{
		config:      config,
		allocations: make(map[string]*PortAllocation),
		usedPorts:   make(map[int]string),
	}

	// Start cleanup routine
	allocator.cleanupTimer = time.NewTicker(5 * time.Minute)
	go allocator.cleanupExpiredAllocations()

	return allocator
}

// AllocatePorts allocates a set of ports for a rental
func (pa *PortAllocator) AllocatePorts(rentalID string) (*Endpoints, error) {
	pa.mutex.Lock()
	defer pa.mutex.Unlock()

	// Check if already allocated
	if allocation, exists := pa.allocations[rentalID]; exists {
		return &Endpoints{
			SSHPort:        allocation.SSHPort,
			ValidationPort: allocation.ValidationPort,
			ErrorResPort:   allocation.ErrorResPort,
			Host:           "localhost", // Default host
		}, nil
	}

	// Find available ports
	sshPort, err := pa.findAvailablePort(pa.config.SSHRangeStart, pa.config.SSHRangeEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate SSH port: %w", err)
	}

	validationPort, err := pa.findAvailablePort(pa.config.ValidationRangeStart, pa.config.ValidationRangeEnd)
	if err != nil {
		pa.releasePort(sshPort) // Cleanup
		return nil, fmt.Errorf("failed to allocate validation port: %w", err)
	}

	errorResPort, err := pa.findAvailablePort(pa.config.ErrorResRangeStart, pa.config.ErrorResRangeEnd)
	if err != nil {
		pa.releasePort(sshPort)        // Cleanup
		pa.releasePort(validationPort) // Cleanup
		return nil, fmt.Errorf("failed to allocate error resolution port: %w", err)
	}

	// Create allocation record
	allocation := &PortAllocation{
		RentalID:       rentalID,
		SSHPort:        sshPort,
		ValidationPort: validationPort,
		ErrorResPort:   errorResPort,
		AllocatedAt:    time.Now(),
		ExpiresAt:      time.Now().Add(25 * time.Hour), // Default 25 hours (1 hour grace period)
	}

	// Record allocation
	pa.allocations[rentalID] = allocation
	pa.usedPorts[sshPort] = rentalID
	pa.usedPorts[validationPort] = rentalID
	pa.usedPorts[errorResPort] = rentalID

	log.Printf("Allocated ports for rental %s: SSH=%d, Validation=%d, ErrorRes=%d",
		rentalID, sshPort, validationPort, errorResPort)

	return &Endpoints{
		SSHPort:        sshPort,
		ValidationPort: validationPort,
		ErrorResPort:   errorResPort,
		Host:           "localhost",
	}, nil
}

// ReleasePorts releases all ports allocated for a rental
func (pa *PortAllocator) ReleasePorts(rentalID string) {
	pa.mutex.Lock()
	defer pa.mutex.Unlock()

	allocation, exists := pa.allocations[rentalID]
	if !exists {
		return
	}

	// Release individual ports
	pa.releasePort(allocation.SSHPort)
	pa.releasePort(allocation.ValidationPort)
	pa.releasePort(allocation.ErrorResPort)

	// Remove allocation record
	delete(pa.allocations, rentalID)

	log.Printf("Released ports for rental %s", rentalID)
}

// GetAllocation returns the port allocation for a rental
func (pa *PortAllocator) GetAllocation(rentalID string) (*PortAllocation, bool) {
	pa.mutex.RLock()
	defer pa.mutex.RUnlock()

	allocation, exists := pa.allocations[rentalID]
	return allocation, exists
}

// ExtendAllocation extends the expiration time of a port allocation
func (pa *PortAllocator) ExtendAllocation(rentalID string, duration time.Duration) error {
	pa.mutex.Lock()
	defer pa.mutex.Unlock()

	allocation, exists := pa.allocations[rentalID]
	if !exists {
		return fmt.Errorf("no allocation found for rental %s", rentalID)
	}

	allocation.ExpiresAt = time.Now().Add(duration)
	log.Printf("Extended allocation for rental %s until %s", rentalID, allocation.ExpiresAt)
	return nil
}

// ListAllocations returns all current port allocations
func (pa *PortAllocator) ListAllocations() []*PortAllocation {
	pa.mutex.RLock()
	defer pa.mutex.RUnlock()

	allocations := make([]*PortAllocation, 0, len(pa.allocations))
	for _, allocation := range pa.allocations {
		allocations = append(allocations, allocation)
	}
	return allocations
}

// findAvailablePort finds an available port in the given range
func (pa *PortAllocator) findAvailablePort(start, end int) (int, error) {
	for port := start; port <= end; port++ {
		if _, used := pa.usedPorts[port]; !used {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports in range %d-%d", start, end)
}

// releasePort releases a specific port
func (pa *PortAllocator) releasePort(port int) {
	if rentalID, exists := pa.usedPorts[port]; exists {
		delete(pa.usedPorts, port)
		log.Printf("Released port %d (was allocated to %s)", port, rentalID)
	}
}

// cleanupExpiredAllocations periodically cleans up expired port allocations
func (pa *PortAllocator) cleanupExpiredAllocations() {
	for range pa.cleanupTimer.C {
		pa.mutex.Lock()
		now := time.Now()
		expiredRentals := make([]string, 0)

		for rentalID, allocation := range pa.allocations {
			if now.After(allocation.ExpiresAt) {
				expiredRentals = append(expiredRentals, rentalID)
			}
		}

		// Release expired allocations
		for _, rentalID := range expiredRentals {
			log.Printf("Cleaning up expired allocation for rental %s", rentalID)
			pa.ReleasePorts(rentalID)
		}

		pa.mutex.Unlock()

		if len(expiredRentals) > 0 {
			log.Printf("Cleaned up %d expired port allocations", len(expiredRentals))
		}
	}
}

// Stop stops the port allocator and cleanup routine
func (pa *PortAllocator) Stop() {
	if pa.cleanupTimer != nil {
		pa.cleanupTimer.Stop()
	}
}
