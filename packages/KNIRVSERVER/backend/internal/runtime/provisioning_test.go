// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultProvisioningPortConfig(t *testing.T) {
	config := DefaultProvisioningPortConfig()

	assert.Equal(t, 2200, config.SSHRangeStart)
	assert.Equal(t, 2299, config.SSHRangeEnd)
	assert.Equal(t, 9100, config.ValidationRangeStart)
	assert.Equal(t, 9199, config.ValidationRangeEnd)
	assert.Equal(t, 9200, config.ErrorResRangeStart)
	assert.Equal(t, 9299, config.ErrorResRangeEnd)
}

func TestNewPortAllocator(t *testing.T) {
	config := DefaultProvisioningPortConfig()
	allocator := NewPortAllocator(config)

	assert.NotNil(t, allocator)
	assert.NotNil(t, allocator.cleanupTimer)
	assert.Empty(t, allocator.allocations)
	assert.Empty(t, allocator.usedPorts)
}

func TestPortAllocator_AllocatePorts(t *testing.T) {
	config := DefaultProvisioningPortConfig()
	allocator := NewPortAllocator(config)
	defer allocator.Stop()

	endpoints, err := allocator.AllocatePorts("rental-1")
	assert.NoError(t, err)
	assert.NotNil(t, endpoints)

	// Verify ports are within expected ranges
	assert.GreaterOrEqual(t, endpoints.SSHPort, config.SSHRangeStart)
	assert.LessOrEqual(t, endpoints.SSHPort, config.SSHRangeEnd)
	assert.GreaterOrEqual(t, endpoints.ValidationPort, config.ValidationRangeStart)
	assert.LessOrEqual(t, endpoints.ValidationPort, config.ValidationRangeEnd)
	assert.GreaterOrEqual(t, endpoints.ErrorResPort, config.ErrorResRangeStart)
	assert.LessOrEqual(t, endpoints.ErrorResPort, config.ErrorResRangeEnd)

	// Verify all ports are unique
	assert.NotEqual(t, endpoints.SSHPort, endpoints.ValidationPort)
	assert.NotEqual(t, endpoints.SSHPort, endpoints.ErrorResPort)
	assert.NotEqual(t, endpoints.ValidationPort, endpoints.ErrorResPort)
}

func TestPortAllocator_AllocatePorts_Idempotent(t *testing.T) {
	config := DefaultProvisioningPortConfig()
	allocator := NewPortAllocator(config)
	defer allocator.Stop()

	// Allocate twice for the same rental ID
	endpoints1, err := allocator.AllocatePorts("rental-1")
	assert.NoError(t, err)

	endpoints2, err := allocator.AllocatePorts("rental-1")
	assert.NoError(t, err)

	// Should return the same endpoints
	assert.Equal(t, endpoints1.SSHPort, endpoints2.SSHPort)
	assert.Equal(t, endpoints1.ValidationPort, endpoints2.ValidationPort)
	assert.Equal(t, endpoints1.ErrorResPort, endpoints2.ErrorResPort)
}

func TestPortAllocator_ReleasePorts(t *testing.T) {
	config := DefaultProvisioningPortConfig()
	allocator := NewPortAllocator(config)
	defer allocator.Stop()

	// Allocate ports
	endpoints, err := allocator.AllocatePorts("rental-1")
	assert.NoError(t, err)

	// Release ports
	allocator.ReleasePorts("rental-1")

	// Verify allocation is removed
	alloc, exists := allocator.GetAllocation("rental-1")
	assert.False(t, exists)
	assert.Nil(t, alloc)

	// Verify ports are freed (can be reallocated)
	newEndpoints, err := allocator.AllocatePorts("rental-2")
	assert.NoError(t, err)
	assert.Equal(t, endpoints.SSHPort, newEndpoints.SSHPort)
	assert.Equal(t, endpoints.ValidationPort, newEndpoints.ValidationPort)
	assert.Equal(t, endpoints.ErrorResPort, newEndpoints.ErrorResPort)
}

func TestPortAllocator_GetAllocation(t *testing.T) {
	config := DefaultProvisioningPortConfig()
	allocator := NewPortAllocator(config)
	defer allocator.Stop()

	// Allocate ports
	_, err := allocator.AllocatePorts("rental-1")
	assert.NoError(t, err)

	// Get allocation
	alloc, exists := allocator.GetAllocation("rental-1")
	assert.True(t, exists)
	assert.NotNil(t, alloc)
	assert.Equal(t, "rental-1", alloc.RentalID)
	assert.True(t, alloc.ExpiresAt.After(time.Now()))
}

func TestPortAllocator_ExtendAllocation(t *testing.T) {
	config := DefaultProvisioningPortConfig()
	allocator := NewPortAllocator(config)
	defer allocator.Stop()

	// Allocate ports - AllocatePorts returns ProvisioningEndpoints, not ProvisioningPortAllocation
	_, err := allocator.AllocatePorts("rental-1")
	assert.NoError(t, err)

	// Get original allocation to capture expiry
	originalAlloc, exists := allocator.GetAllocation("rental-1")
	assert.True(t, exists)
	originalExpiry := originalAlloc.ExpiresAt

	// Extend allocation
	err = allocator.ExtendAllocation("rental-1", 1*time.Hour)
	assert.NoError(t, err)

	// Verify expiry has been extended
	newAlloc, exists := allocator.GetAllocation("rental-1")
	assert.True(t, exists)
	assert.True(t, newAlloc.ExpiresAt.After(originalExpiry))
}

func TestPortAllocator_ExtendAllocation_NotFound(t *testing.T) {
	config := DefaultProvisioningPortConfig()
	allocator := NewPortAllocator(config)
	defer allocator.Stop()

	err := allocator.ExtendAllocation("non-existent", 1*time.Hour)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no allocation found")
}

func TestPortAllocator_ListAllocations(t *testing.T) {
	config := DefaultProvisioningPortConfig()
	allocator := NewPortAllocator(config)
	defer allocator.Stop()

	// Allocate multiple ports
	_, err := allocator.AllocatePorts("rental-1")
	assert.NoError(t, err)
	_, err = allocator.AllocatePorts("rental-2")
	assert.NoError(t, err)
	_, err = allocator.AllocatePorts("rental-3")
	assert.NoError(t, err)

	allocs := allocator.ListAllocations()
	assert.Len(t, allocs, 3)
}

func TestNewSSHProvisioner(t *testing.T) {
	provisioner := NewSSHProvisioner()
	assert.NotNil(t, provisioner)
}

func TestSSHProvisioner_GenerateSSHKeypair(t *testing.T) {
	provisioner := NewSSHProvisioner()
	keypair, err := provisioner.GenerateSSHKeypair()

	assert.NoError(t, err)
	assert.NotNil(t, keypair)
	assert.NotEmpty(t, keypair.PublicKey)
	assert.NotEmpty(t, keypair.PrivateKey)
	assert.NotEmpty(t, keypair.KeyFingerprint)
	assert.Contains(t, keypair.PublicKey, "ssh-rsa")
	assert.Contains(t, keypair.PrivateKey, "-----BEGIN RSA PRIVATE KEY-----")
}

func TestSSHProvisioner_ValidateSSHKey(t *testing.T) {
	provisioner := NewSSHProvisioner()

	// Generate a valid key
	keypair, err := provisioner.GenerateSSHKeypair()
	assert.NoError(t, err)

	// Validate the key
	err = provisioner.ValidateSSHKey(keypair.PublicKey)
	assert.NoError(t, err)
}

func TestSSHProvisioner_ValidateSSHKey_Invalid(t *testing.T) {
	provisioner := NewSSHProvisioner()

	err := provisioner.ValidateSSHKey("invalid-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid SSH public key")
}

func TestSSHProvisioner_GetSSHConfig(t *testing.T) {
	provisioner := NewSSHProvisioner()
	keypair, err := provisioner.GenerateSSHKeypair()
	assert.NoError(t, err)

	config, err := provisioner.GetSSHConfig(keypair.PrivateKey, "testuser")
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "testuser", config.User)
	assert.Len(t, config.Auth, 1)
}

func TestSSHProvisioner_GetSSHConfig_InvalidKey(t *testing.T) {
	provisioner := NewSSHProvisioner()

	_, err := provisioner.GetSSHConfig("invalid-private-key", "testuser")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse private key")
}

func TestSSHProvisioner_InjectSSHKey(t *testing.T) {
	provisioner := NewSSHProvisioner()
	keypair, _ := provisioner.GenerateSSHKeypair()

	err := provisioner.InjectSSHKey("container-123", keypair.PublicKey, "testuser")
	assert.NoError(t, err)
}

func TestSSHProvisioner_RevokeSSHKey(t *testing.T) {
	provisioner := NewSSHProvisioner()
	keypair, _ := provisioner.GenerateSSHKeypair()

	err := provisioner.RevokeSSHKey("container-123", "testuser", keypair.KeyFingerprint)
	assert.NoError(t, err)
}

func TestSSHProvisioner_ListSSHKeys(t *testing.T) {
	provisioner := NewSSHProvisioner()

	keys, err := provisioner.ListSSHKeys("container-123", "testuser")
	assert.NoError(t, err)
	assert.Empty(t, keys)
}

func TestNewProvisioner(t *testing.T) {
	config := DefaultProvisioningPortConfig()
	provisioner := NewProvisioner(config)

	assert.NotNil(t, provisioner)
	assert.NotNil(t, provisioner.Ports)
	assert.NotNil(t, provisioner.SSH)
}

func TestProvisioner_ProvisionContainer(t *testing.T) {
	config := DefaultProvisioningPortConfig()
	provisioner := NewProvisioner(config)
	defer provisioner.Stop()

	endpoints, keypair, err := provisioner.ProvisionContainer("rental-1")

	assert.NoError(t, err)
	assert.NotNil(t, endpoints)
	assert.NotNil(t, keypair)
	assert.NotEmpty(t, keypair.PublicKey)
	assert.NotEmpty(t, keypair.PrivateKey)
}

func TestProvisioner_Stop(t *testing.T) {
	config := DefaultProvisioningPortConfig()
	provisioner := NewProvisioner(config)

	// Should not panic
	provisioner.Stop()
}

func TestNewUnifiedContainerManagerWithProvisioning(t *testing.T) {
	config := DefaultProvisioningPortConfig()
	manager := NewUnifiedContainerManagerWithProvisioning(nil, nil, nil, config)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.UnifiedContainerManager)
	assert.NotNil(t, manager.Provisioner)
	assert.NotNil(t, manager.Provisioner.Ports)
	assert.NotNil(t, manager.Provisioner.SSH)
}

func TestUnifiedContainerManagerWithProvisioning_Stop(t *testing.T) {
	config := DefaultProvisioningPortConfig()
	manager := NewUnifiedContainerManagerWithProvisioning(nil, nil, nil, config)

	// Should not panic
	manager.Stop()
}

func TestProvisioningEndpoints_Fields(t *testing.T) {
	endpoints := &ProvisioningEndpoints{
		SSHPort:        2200,
		ValidationPort: 9100,
		ErrorResPort:   9200,
		Host:           "localhost",
	}

	assert.Equal(t, 2200, endpoints.SSHPort)
	assert.Equal(t, 9100, endpoints.ValidationPort)
	assert.Equal(t, 9200, endpoints.ErrorResPort)
	assert.Equal(t, "localhost", endpoints.Host)
}

func TestProvisioningPortAllocation_Fields(t *testing.T) {
	now := time.Now()
	alloc := &ProvisioningPortAllocation{
		RentalID:       "rental-1",
		SSHPort:        2200,
		ValidationPort: 9100,
		ErrorResPort:   9200,
		AllocatedAt:    now,
		ExpiresAt:      now.Add(24 * time.Hour),
	}

	assert.Equal(t, "rental-1", alloc.RentalID)
	assert.Equal(t, 2200, alloc.SSHPort)
	assert.Equal(t, 9100, alloc.ValidationPort)
	assert.Equal(t, 9200, alloc.ErrorResPort)
	assert.Equal(t, now, alloc.AllocatedAt)
	assert.True(t, alloc.ExpiresAt.After(now))
}
