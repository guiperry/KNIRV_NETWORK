package dverental

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"nexus-backend/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/buntdb"
)

func setupTestDVERentalService(t *testing.T) (*DVERentalService, *buntdb.DB) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_dve_rental.db")

	db, err := buntdb.Open(dbPath)
	require.NoError(t, err)

	service, err := NewDVERentalService(db)
	require.NoError(t, err)

	t.Cleanup(func() {
		if service.running {
			service.Stop()
		}
		db.Close()
		os.Remove(dbPath)
	})

	return service, db
}

func TestNewDVERentalService(t *testing.T) {
	service, _ := setupTestDVERentalService(t)

	assert.NotNil(t, service)
	assert.NotNil(t, service.db)
	assert.False(t, service.running)
	assert.NotNil(t, service.activeRentals)
	assert.NotNil(t, service.rentalPlans)
	assert.Equal(t, 5*time.Minute, service.cleanupInterval)
	assert.NotEmpty(t, service.defaultPlans)

	// Check that default plans are loaded
	plans, err := service.GetRentalPlans()
	assert.NoError(t, err)
	assert.NotEmpty(t, plans)
}

func TestDVERentalService_Start(t *testing.T) {
	service, _ := setupTestDVERentalService(t)

	err := service.Start()
	assert.NoError(t, err)
	assert.True(t, service.running)

	// Test starting already running service
	err = service.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Cleanup
	err = service.Stop()
	assert.NoError(t, err)
}

func TestDVERentalService_Stop(t *testing.T) {
	service, _ := setupTestDVERentalService(t)

	// Test stopping non-running service (should be no-op, no error)
	err := service.Stop()
	assert.NoError(t, err)

	// Start and then stop
	err = service.Start()
	require.NoError(t, err)

	err = service.Stop()
	assert.NoError(t, err)
	assert.False(t, service.running)
}

func TestDVERentalService_CreateRental(t *testing.T) {
	service, _ := setupTestDVERentalService(t)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// Get a plan to use
	plans, err := service.GetRentalPlans()
	require.NoError(t, err)
	require.NotEmpty(t, plans)
	plan := plans[0]

	request := &models.RentalRequest{
		PlanID:        plan.ID,
		Duration:      24 * 3600, // 24 hours in seconds
		PaymentTxHash: "0xtest123456789",
		UserID:        "test-user",
	}

	rental, err := service.CreateRental(request)
	assert.NoError(t, err)
	assert.NotNil(t, rental)
	assert.True(t, rental.Success)
	assert.NotEmpty(t, rental.RentalID)
	assert.NotEmpty(t, rental.DVENodeID)
	assert.NotEmpty(t, rental.CDEAccessURL)
}

func TestDVERentalService_CreateRentalInvalidPlan(t *testing.T) {
	service, _ := setupTestDVERentalService(t)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	request := &models.RentalRequest{
		PlanID:        "invalid-plan-id",
		Duration:      24 * 3600,
		PaymentTxHash: "0xtest123456789",
		UserID:        "test-user",
	}

	rental, err := service.CreateRental(request)
	assert.NoError(t, err)
	assert.NotNil(t, rental)
	assert.False(t, rental.Success)
	assert.Contains(t, rental.Error, "Invalid rental plan ID")
}
