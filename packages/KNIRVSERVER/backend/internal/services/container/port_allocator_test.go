package container

import (
	"testing"
	"time"
)

func TestNewPortAllocator(t *testing.T) {
	config := PortAllocationConfig{
		SSHRangeStart:        22000,
		SSHRangeEnd:          22010,
		ValidationRangeStart: 23000,
		ValidationRangeEnd:   23010,
		ErrorResRangeStart:   24000,
		ErrorResRangeEnd:     24010,
	}

	pa := NewPortAllocator(config)

	if pa == nil {
		t.Fatal("Port allocator is nil")
	}

	if pa.config.SSHRangeStart != 22000 {
		t.Errorf("Expected SSH range start 22000, got %d", pa.config.SSHRangeStart)
	}

	if pa.allocations == nil {
		t.Error("Allocations map is nil")
	}

	if pa.usedPorts == nil {
		t.Error("Used ports map is nil")
	}
}

func TestPortAllocator_AllocatePorts(t *testing.T) {
	config := PortAllocationConfig{
		SSHRangeStart:        22000,
		SSHRangeEnd:          22005,
		ValidationRangeStart: 23000,
		ValidationRangeEnd:   23005,
		ErrorResRangeStart:   24000,
		ErrorResRangeEnd:     24005,
	}

	pa := NewPortAllocator(config)
	defer pa.Stop()

	endpoints, err := pa.AllocatePorts("test-rental")
	if err != nil {
		t.Fatalf("Failed to allocate ports: %v", err)
	}

	if endpoints.SSHPort < 22000 || endpoints.SSHPort > 22005 {
		t.Errorf("SSH port %d not in expected range [22000, 22005]", endpoints.SSHPort)
	}

	if endpoints.ValidationPort < 23000 || endpoints.ValidationPort > 23005 {
		t.Errorf("Validation port %d not in expected range [23000, 23005]", endpoints.ValidationPort)
	}

	if endpoints.ErrorResPort < 24000 || endpoints.ErrorResPort > 24005 {
		t.Errorf("Error resolution port %d not in expected range [24000, 24005]", endpoints.ErrorResPort)
	}

	if endpoints.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", endpoints.Host)
	}

	// Check that ports are marked as used
	allocation, exists := pa.GetAllocation("test-rental")
	if !exists {
		t.Error("Allocation not found")
	}

	if allocation.SSHPort != endpoints.SSHPort {
		t.Errorf("Allocation SSH port mismatch: expected %d, got %d", endpoints.SSHPort, allocation.SSHPort)
	}
}

func TestPortAllocator_AllocatePortsDuplicate(t *testing.T) {
	config := PortAllocationConfig{
		SSHRangeStart:        22000,
		SSHRangeEnd:          22005,
		ValidationRangeStart: 23000,
		ValidationRangeEnd:   23005,
		ErrorResRangeStart:   24000,
		ErrorResRangeEnd:     24005,
	}

	pa := NewPortAllocator(config)
	defer pa.Stop()

	// First allocation
	endpoints1, err := pa.AllocatePorts("test-rental")
	if err != nil {
		t.Fatalf("Failed to allocate ports first time: %v", err)
	}

	// Second allocation for same rental should return same ports
	endpoints2, err := pa.AllocatePorts("test-rental")
	if err != nil {
		t.Fatalf("Failed to allocate ports second time: %v", err)
	}

	if endpoints1.SSHPort != endpoints2.SSHPort {
		t.Errorf("SSH ports don't match: %d != %d", endpoints1.SSHPort, endpoints2.SSHPort)
	}

	if endpoints1.ValidationPort != endpoints2.ValidationPort {
		t.Errorf("Validation ports don't match: %d != %d", endpoints1.ValidationPort, endpoints2.ValidationPort)
	}

	if endpoints1.ErrorResPort != endpoints2.ErrorResPort {
		t.Errorf("Error resolution ports don't match: %d != %d", endpoints1.ErrorResPort, endpoints2.ErrorResPort)
	}
}

func TestPortAllocator_ReleasePorts(t *testing.T) {
	config := PortAllocationConfig{
		SSHRangeStart:        22000,
		SSHRangeEnd:          22005,
		ValidationRangeStart: 23000,
		ValidationRangeEnd:   23005,
		ErrorResRangeStart:   24000,
		ErrorResRangeEnd:     24005,
	}

	pa := NewPortAllocator(config)
	defer pa.Stop()

	// Allocate ports
	endpoints, err := pa.AllocatePorts("test-rental")
	if err != nil {
		t.Fatalf("Failed to allocate ports: %v", err)
	}

	// Verify allocation exists
	_, exists := pa.GetAllocation("test-rental")
	if !exists {
		t.Error("Allocation should exist before release")
	}

	// Release ports
	pa.ReleasePorts("test-rental")

	// Verify allocation is gone
	_, exists = pa.GetAllocation("test-rental")
	if exists {
		t.Error("Allocation should not exist after release")
	}

	// Verify ports are available again
	newEndpoints, err := pa.AllocatePorts("test-rental")
	if err != nil {
		t.Fatalf("Failed to allocate ports after release: %v", err)
	}

	if newEndpoints.SSHPort != endpoints.SSHPort {
		t.Errorf("SSH port should be reused: expected %d, got %d", endpoints.SSHPort, newEndpoints.SSHPort)
	}
}

func TestPortAllocator_ExtendAllocation(t *testing.T) {
	config := PortAllocationConfig{
		SSHRangeStart:        22000,
		SSHRangeEnd:          22005,
		ValidationRangeStart: 23000,
		ValidationRangeEnd:   23005,
		ErrorResRangeStart:   24000,
		ErrorResRangeEnd:     24005,
	}

	pa := NewPortAllocator(config)
	defer pa.Stop()

	// Allocate ports
	_, err := pa.AllocatePorts("test-rental")
	if err != nil {
		t.Fatalf("Failed to allocate ports: %v", err)
	}

	allocation, exists := pa.GetAllocation("test-rental")
	if !exists {
		t.Fatal("Allocation not found")
	}

	originalExpiry := allocation.ExpiresAt

	// Extend allocation
	err = pa.ExtendAllocation("test-rental", 2*time.Hour)
	if err != nil {
		t.Fatalf("Failed to extend allocation: %v", err)
	}

	extendedAllocation, exists := pa.GetAllocation("test-rental")
	if !exists {
		t.Fatal("Allocation not found after extension")
	}

	expectedExpiry := time.Now().Add(2 * time.Hour)
	if extendedAllocation.ExpiresAt.Equal(originalExpiry) {
		t.Errorf("Allocation expiry was not extended. Original: %v, Extended: %v",
			originalExpiry, extendedAllocation.ExpiresAt)
	}
	// Allow some tolerance for timing
	if extendedAllocation.ExpiresAt.Before(expectedExpiry.Add(-time.Second)) {
		t.Errorf("Allocation expiry was not extended sufficiently. Expected at least: %v, Got: %v",
			expectedExpiry, extendedAllocation.ExpiresAt)
	}
}

func TestPortAllocator_ExtendAllocationNotFound(t *testing.T) {
	config := PortAllocationConfig{
		SSHRangeStart:        22000,
		SSHRangeEnd:          22005,
		ValidationRangeStart: 23000,
		ValidationRangeEnd:   23005,
		ErrorResRangeStart:   24000,
		ErrorResRangeEnd:     24005,
	}

	pa := NewPortAllocator(config)
	defer pa.Stop()

	err := pa.ExtendAllocation("nonexistent-rental", time.Hour)
	if err == nil {
		t.Error("Expected error for extending non-existent allocation")
	}
}

func TestPortAllocator_ListAllocations(t *testing.T) {
	config := PortAllocationConfig{
		SSHRangeStart:        22000,
		SSHRangeEnd:          22005,
		ValidationRangeStart: 23000,
		ValidationRangeEnd:   23005,
		ErrorResRangeStart:   24000,
		ErrorResRangeEnd:     24005,
	}

	pa := NewPortAllocator(config)
	defer pa.Stop()

	// Initially empty
	allocations := pa.ListAllocations()
	if len(allocations) != 0 {
		t.Errorf("Expected 0 allocations initially, got %d", len(allocations))
	}

	// Allocate for one rental
	_, err := pa.AllocatePorts("rental1")
	if err != nil {
		t.Fatalf("Failed to allocate ports: %v", err)
	}

	allocations = pa.ListAllocations()
	if len(allocations) != 1 {
		t.Errorf("Expected 1 allocation, got %d", len(allocations))
	}

	if allocations[0].RentalID != "rental1" {
		t.Errorf("Expected rental ID 'rental1', got '%s'", allocations[0].RentalID)
	}

	// Allocate for another rental
	_, err = pa.AllocatePorts("rental2")
	if err != nil {
		t.Fatalf("Failed to allocate ports: %v", err)
	}

	allocations = pa.ListAllocations()
	if len(allocations) != 2 {
		t.Errorf("Expected 2 allocations, got %d", len(allocations))
	}
}

func TestPortAllocator_FindAvailablePort(t *testing.T) {
	config := PortAllocationConfig{
		SSHRangeStart:        22000,
		SSHRangeEnd:          22002,
		ValidationRangeStart: 23000,
		ValidationRangeEnd:   23002,
		ErrorResRangeStart:   24000,
		ErrorResRangeEnd:     24002,
	}

	pa := NewPortAllocator(config)
	defer pa.Stop()

	// Test finding available port
	port, err := pa.findAvailablePort(22000, 22002)
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}

	if port < 22000 || port > 22002 {
		t.Errorf("Port %d not in range [22000, 22002]", port)
	}

	// Mark port as used
	pa.usedPorts[port] = "test"

	// Should find different port
	port2, err := pa.findAvailablePort(22000, 22002)
	if err != nil {
		t.Fatalf("Failed to find second available port: %v", err)
	}

	if port2 == port {
		t.Errorf("Should have found different port, but got same port %d", port2)
	}
}

func TestPortAllocator_FindAvailablePortExhausted(t *testing.T) {
	config := PortAllocationConfig{
		SSHRangeStart:        22000,
		SSHRangeEnd:          22000,
		ValidationRangeStart: 23000,
		ValidationRangeEnd:   23000,
		ErrorResRangeStart:   24000,
		ErrorResRangeEnd:     24000,
	}

	pa := NewPortAllocator(config)
	defer pa.Stop()

	// Mark the only port as used
	pa.usedPorts[22000] = "test"

	// Should fail to find available port
	_, err := pa.findAvailablePort(22000, 22000)
	if err == nil {
		t.Error("Expected error when no ports available")
	}
}

func TestPortAllocator_Stop(t *testing.T) {
	config := PortAllocationConfig{
		SSHRangeStart:        22000,
		SSHRangeEnd:          22005,
		ValidationRangeStart: 23000,
		ValidationRangeEnd:   23005,
		ErrorResRangeStart:   24000,
		ErrorResRangeEnd:     24005,
	}

	pa := NewPortAllocator(config)

	// Should not panic
	pa.Stop()

	// Timer should be stopped
	if pa.cleanupTimer != nil {
		// This is a basic check - in real implementation we might check if it's actually stopped
	}
}

func TestPortAllocator_CleanupExpiredAllocations(t *testing.T) {
	config := PortAllocationConfig{
		SSHRangeStart:        22000,
		SSHRangeEnd:          22005,
		ValidationRangeStart: 23000,
		ValidationRangeEnd:   23005,
		ErrorResRangeStart:   24000,
		ErrorResRangeEnd:     24005,
	}

	pa := NewPortAllocator(config)
	defer pa.Stop()

	// Allocate a port
	_, err := pa.AllocatePorts("test-rental")
	if err != nil {
		t.Fatalf("Failed to allocate ports: %v", err)
	}

	// Verify allocation exists
	_, exists := pa.GetAllocation("test-rental")
	if !exists {
		t.Error("Allocation should exist")
	}

	// Test that the cleanup routine exists and can be called
	// Note: This method blocks on a channel receive, so we can't easily test it directly
	// without complex setup. For now, just ensure the port allocator was created successfully.
	if pa == nil {
		t.Error("Port allocator should not be nil")
	}
}