package dverental

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"nexus-backend/internal/models"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

// DVERentalService manages DVE rental operations
type DVERentalService struct {
	db      *buntdb.DB
	mu      sync.RWMutex
	running bool

	// Service references
	dveManager interface{} // DVE manager for node allocation
	cdeService interface{} // CDE service for environment provisioning

	// Rental data
	activeRentals map[string]*models.DVERental
	rentalPlans   map[string]*models.RentalPlan

	// Configuration
	cleanupInterval time.Duration
	defaultPlans    []*models.RentalPlan
}

// NewDVERentalService creates a new DVE rental service
func NewDVERentalService(db *buntdb.DB) (*DVERentalService, error) {
	service := &DVERentalService{
		db:              db,
		activeRentals:   make(map[string]*models.DVERental),
		rentalPlans:     make(map[string]*models.RentalPlan),
		cleanupInterval: 5 * time.Minute,
		defaultPlans:    createDefaultRentalPlans(),
	}

	// Load existing data from database
	if err := service.loadFromDatabase(); err != nil {
		log.Printf("Warning: Failed to load rental data from database: %v", err)
	}

	// Initialize default rental plans
	service.initializeDefaultPlans()

	return service, nil
}

// SetServiceReferences sets references to other services
func (drs *DVERentalService) SetServiceReferences(dveManager, cdeService interface{}) {
	drs.mu.Lock()
	defer drs.mu.Unlock()

	drs.dveManager = dveManager
	drs.cdeService = cdeService
}

// Start starts the DVE rental service
func (drs *DVERentalService) Start() error {
	drs.mu.Lock()
	defer drs.mu.Unlock()

	if drs.running {
		return fmt.Errorf("DVE rental service is already running")
	}

	log.Println("Starting DVE rental service...")

	// Start cleanup routine
	go drs.cleanupRoutine()

	drs.running = true
	log.Println("DVE rental service started successfully")
	return nil
}

// Stop stops the DVE rental service
func (drs *DVERentalService) Stop() error {
	drs.mu.Lock()
	defer drs.mu.Unlock()

	if !drs.running {
		return nil
	}

	log.Println("Stopping DVE rental service...")

	// Save current state to database
	if err := drs.saveToDatabase(); err != nil {
		log.Printf("Warning: Failed to save rental data to database: %v", err)
	}

	drs.running = false
	log.Println("DVE rental service stopped")
	return nil
}

// IsRunning returns whether the service is running
func (drs *DVERentalService) IsRunning() bool {
	drs.mu.RLock()
	defer drs.mu.RUnlock()
	return drs.running
}

// CreateRental creates a new DVE rental
func (drs *DVERentalService) CreateRental(req *models.RentalRequest) (*models.RentalResponse, error) {
	drs.mu.Lock()
	defer drs.mu.Unlock()

	// Validate rental plan
	plan, exists := drs.rentalPlans[req.PlanID]
	if !exists {
		return &models.RentalResponse{
			Success: false,
			Error:   "Invalid rental plan ID",
		}, nil
	}

	// Validate duration
	if req.Duration < plan.MinDuration || req.Duration > plan.MaxDuration {
		return &models.RentalResponse{
			Success: false,
			Error:   fmt.Sprintf("Duration must be between %d and %d seconds", plan.MinDuration, plan.MaxDuration),
		}, nil
	}

	// Calculate total cost
	hours := float64(req.Duration) / 3600.0
	totalCost := int64(hours * float64(plan.PricePerHour))

	// TODO: Verify NRN payment transaction
	// For now, we'll assume the payment is valid

	// Find available DVE node
	dveNodeID := drs.findAvailableDVENode(req.PreferredDVE)
	if dveNodeID == "" {
		return &models.RentalResponse{
			Success: false,
			Error:   "No available DVE nodes",
		}, nil
	}

	// Create rental record
	rental := &models.DVERental{
		ID:             uuid.New().String(),
		UserID:         req.UserID,
		DVENodeID:      dveNodeID,
		NRNAmount:      totalCost,
		RentalDuration: req.Duration,
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(time.Duration(req.Duration) * time.Second),
		Status:         "active",
		PaymentTxHash:  req.PaymentTxHash,
		ResourceLimits: plan.ResourceLimits,
		UsageMetrics:   models.UsageMetrics{LastUpdated: time.Now()},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Provision CDE environment
	cdeEnvID, cdeURL, credentials, err := drs.provisionCDEEnvironment(rental)
	if err != nil {
		return &models.RentalResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to provision CDE environment: %v", err),
		}, nil
	}

	rental.CDEEnvironmentID = cdeEnvID

	// Store rental
	drs.activeRentals[rental.ID] = rental

	// Save to database
	if err := drs.saveRentalToDatabase(rental); err != nil {
		log.Printf("Warning: Failed to save rental to database: %v", err)
	}

	return &models.RentalResponse{
		Success:        true,
		RentalID:       rental.ID,
		DVENodeID:      dveNodeID,
		CDEAccessURL:   cdeURL,
		CDECredentials: credentials,
		ExpiresAt:      rental.EndTime,
		Message:        "DVE rental created successfully",
	}, nil
}

// GetActiveRentals returns all active rentals for a user
func (drs *DVERentalService) GetActiveRentals(userID string) ([]*models.DVERental, error) {
	drs.mu.RLock()
	defer drs.mu.RUnlock()

	var userRentals []*models.DVERental
	for _, rental := range drs.activeRentals {
		if rental.UserID == userID && rental.Status == "active" {
			userRentals = append(userRentals, rental)
		}
	}

	return userRentals, nil
}

// GetAllUserRentals returns all rentals (active, expired, cancelled) for a user from database
func (drs *DVERentalService) GetAllUserRentals(userID string) ([]*models.DVERental, error) {
	drs.mu.RLock()
	defer drs.mu.RUnlock()

	var userRentals []*models.DVERental

	// Query database for all rentals belonging to this user
	err := drs.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			if len(key) > 8 && key[:8] == "rental:" {
				var rental models.DVERental
				if err := json.Unmarshal([]byte(value), &rental); err == nil {
					if rental.UserID == userID {
						userRentals = append(userRentals, &rental)
					}
				}
			}
			return true
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query user rentals: %w", err)
	}

	return userRentals, nil
}

// GetUserRentedNodeIDs returns a list of DVE node IDs that a user has rented (all statuses)
func (drs *DVERentalService) GetUserRentedNodeIDs(userID string) ([]string, error) {
	rentals, err := drs.GetAllUserRentals(userID)
	if err != nil {
		return nil, err
	}

	nodeIDMap := make(map[string]bool)
	for _, rental := range rentals {
		nodeIDMap[rental.DVENodeID] = true
	}

	nodeIDs := make([]string, 0, len(nodeIDMap))
	for nodeID := range nodeIDMap {
		nodeIDs = append(nodeIDs, nodeID)
	}

	return nodeIDs, nil
}

// GetRentalPlans returns all available rental plans
func (drs *DVERentalService) GetRentalPlans() ([]*models.RentalPlan, error) {
	drs.mu.RLock()
	defer drs.mu.RUnlock()

	var plans []*models.RentalPlan
	for _, plan := range drs.rentalPlans {
		if plan.IsActive {
			plans = append(plans, plan)
		}
	}

	return plans, nil
}

// GetRentalStats returns rental system statistics
func (drs *DVERentalService) GetRentalStats() (*models.DVERentalStats, error) {
	drs.mu.RLock()
	defer drs.mu.RUnlock()

	stats := &models.DVERentalStats{
		TotalRentals:  int64(len(drs.activeRentals)),
		ActiveRentals: drs.countActiveRentals(),
		Timestamp:     time.Now(),
	}

	// Calculate revenue and other metrics
	stats.TotalNRNCollected = drs.calculateTotalRevenue()
	stats.RevenueToday = drs.calculateRevenueForPeriod(24 * time.Hour)
	stats.Revenue7Days = drs.calculateRevenueForPeriod(7 * 24 * time.Hour)
	stats.Revenue30Days = drs.calculateRevenueForPeriod(30 * 24 * time.Hour)

	return stats, nil
}

// ExtendRental extends an existing rental
func (drs *DVERentalService) ExtendRental(rentalID string, additionalDuration int64, paymentTxHash string) error {
	drs.mu.Lock()
	defer drs.mu.Unlock()

	rental, exists := drs.activeRentals[rentalID]
	if !exists {
		return fmt.Errorf("rental not found")
	}

	if rental.Status != "active" {
		return fmt.Errorf("rental is not active")
	}

	// TODO: Verify additional payment

	// Extend the rental
	rental.EndTime = rental.EndTime.Add(time.Duration(additionalDuration) * time.Second)
	rental.RentalDuration += additionalDuration
	rental.UpdatedAt = time.Now()

	// Save to database
	if err := drs.saveRentalToDatabase(rental); err != nil {
		log.Printf("Warning: Failed to save extended rental to database: %v", err)
	}

	return nil
}

// CancelRental cancels an active rental
func (drs *DVERentalService) CancelRental(rentalID string, userID string) error {
	drs.mu.Lock()
	defer drs.mu.Unlock()

	rental, exists := drs.activeRentals[rentalID]
	if !exists {
		return fmt.Errorf("rental not found")
	}

	if rental.UserID != userID {
		return fmt.Errorf("unauthorized: rental belongs to different user")
	}

	if rental.Status != "active" {
		return fmt.Errorf("rental is not active")
	}

	// Cancel the rental
	rental.Status = "cancelled"
	rental.UpdatedAt = time.Now()

	// Clean up CDE environment
	if err := drs.cleanupCDEEnvironment(rental.CDEEnvironmentID); err != nil {
		log.Printf("Warning: Failed to cleanup CDE environment: %v", err)
	}

	// Remove from active rentals
	delete(drs.activeRentals, rentalID)

	// Save to database
	if err := drs.saveRentalToDatabase(rental); err != nil {
		log.Printf("Warning: Failed to save cancelled rental to database: %v", err)
	}

	return nil
}

// Helper methods

// findAvailableDVENode finds an available DVE node for rental
func (drs *DVERentalService) findAvailableDVENode(preferredDVE string) string {
	// TODO: Implement DVE node availability checking
	// For now, return a mock DVE node ID
	if preferredDVE != "" {
		return preferredDVE
	}
	return "dve-node-" + uuid.New().String()[:8]
}

// provisionCDEEnvironment provisions a CDE environment for the rental
func (drs *DVERentalService) provisionCDEEnvironment(rental *models.DVERental) (string, string, models.CDECredentials, error) {
	// Check if CDE service is available
	if drs.cdeService == nil {
		// Fallback to mock data if CDE service is not available
		envID := "cde-env-" + uuid.New().String()[:8]
		accessURL := fmt.Sprintf("https://cde.knirv.com/env/%s", envID)
		credentials := models.CDECredentials{
			Username:    "user-" + rental.UserID[:8],
			Password:    "temp-" + uuid.New().String()[:12],
			AccessToken: "token-" + uuid.New().String(),
		}
		return envID, accessURL, credentials, nil
	}

	// Use actual CDE service
	type CDEServiceInterface interface {
		CreateEnvironment(userID, name string, envType interface{}, config map[string]interface{}) (interface{}, error)
		CreateSession(userID, envID string, connectionType string) (interface{}, error)
	}

	cdeService, ok := drs.cdeService.(CDEServiceInterface)
	if !ok {
		// Fallback to mock if interface doesn't match
		envID := "cde-env-" + uuid.New().String()[:8]
		accessURL := fmt.Sprintf("https://cde.knirv.com/env/%s", envID)
		credentials := models.CDECredentials{
			Username:    "user-" + rental.UserID[:8],
			Password:    "temp-" + uuid.New().String()[:12],
			AccessToken: "token-" + uuid.New().String(),
		}
		return envID, accessURL, credentials, nil
	}

	// Create CDE environment
	envName := fmt.Sprintf("rental-%s", rental.ID[:8])
	envConfig := map[string]interface{}{
		"rental_id":       rental.ID,
		"resource_limits": rental.ResourceLimits,
	}

	env, err := cdeService.CreateEnvironment(rental.UserID, envName, "development", envConfig)
	if err != nil {
		return "", "", models.CDECredentials{}, fmt.Errorf("failed to create CDE environment: %w", err)
	}

	// Extract environment ID from the created environment
	envID := fmt.Sprintf("cde-env-%s", rental.ID[:8])
	if envInterface, ok := env.(interface{ GetID() string }); ok {
		envID = envInterface.GetID()
	}

	// Create a session for the environment
	session, err := cdeService.CreateSession(rental.UserID, envID, "websocket")
	if err != nil {
		log.Printf("Warning: Failed to create CDE session: %v", err)
		// Continue without session - user can create one later
	}

	// Generate access URL and credentials
	accessURL := fmt.Sprintf("https://cde.knirv.com/env/%s", envID)
	credentials := models.CDECredentials{
		Username:    "user-" + rental.UserID[:8],
		Password:    "temp-" + uuid.New().String()[:12],
		AccessToken: "token-" + uuid.New().String(),
	}

	// If session was created successfully, add session info
	if session != nil {
		if sessionInterface, ok := session.(interface{ GetID() string }); ok {
			credentials.AccessToken = sessionInterface.GetID()
		}
	}

	return envID, accessURL, credentials, nil
}

// cleanupCDEEnvironment cleans up a CDE environment
func (drs *DVERentalService) cleanupCDEEnvironment(envID string) error {
	// Check if CDE service is available
	if drs.cdeService == nil {
		log.Printf("Cleaning up CDE environment (mock): %s", envID)
		return nil
	}

	// Use actual CDE service
	type CDEServiceInterface interface {
		DeleteEnvironment(envID string) error
	}

	cdeService, ok := drs.cdeService.(CDEServiceInterface)
	if !ok {
		log.Printf("Cleaning up CDE environment (fallback): %s", envID)
		return nil
	}

	// Delete the CDE environment
	if err := cdeService.DeleteEnvironment(envID); err != nil {
		log.Printf("Warning: Failed to delete CDE environment %s: %v", envID, err)
		return err
	}

	log.Printf("Successfully cleaned up CDE environment: %s", envID)
	return nil
}
