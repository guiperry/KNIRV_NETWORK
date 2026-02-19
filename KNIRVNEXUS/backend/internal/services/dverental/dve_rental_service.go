package dverental

import (
	"backend_server/internal/database"
	"backend_server/internal/objects"
	"backend_server/internal/services/blockchain"
	"backend_server/internal/services/cde"
	"backend_server/internal/services/container"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

// BlockchainClientInterface defines the interface for blockchain operations
type BlockchainClientInterface interface {
	VerifyPaymentTransaction(txHash string, expectedAmount int64, expectedRecipient string) (*objects.NRNPayment, error)
	GetTransactionPool() ([]*blockchain.Transaction, error)
	SubmitTransaction(tx *blockchain.Transaction) (string, error)
	GetAccountBalance(address string) (int64, error)
}

// DVERentalService manages DVE rental operations
type DVERentalService struct {
	db               *database.BuntDBManager
	mu               sync.RWMutex
	running          bool
	blockchainClient BlockchainClientInterface

	// Service references
	dveManager            interface{} // DVE manager for node allocation
	cdeService            interface{} // CDE service for environment provisioning
	containerOrchestrator interface{} // Container orchestrator for TEE containers

	// Rental data
	activeRentals map[string]*objects.DVERental
	rentalPlans   map[string]*objects.RentalPlan

	// Configuration
	cleanupInterval time.Duration
	defaultPlans    []*objects.RentalPlan
	testMode        bool // Enable mock fallbacks for testing
}

// NewDVERentalService creates a new DVE rental service
func NewDVERentalService(db *database.BuntDBManager) (*DVERentalService, error) {
	log.Printf("[DVE Rental] Creating new DVE rental service")
	service := &DVERentalService{
		db:              db,
		activeRentals:   make(map[string]*objects.DVERental),
		rentalPlans:     make(map[string]*objects.RentalPlan),
		cleanupInterval: 5 * time.Minute,
		defaultPlans:    createDefaultRentalPlans(),
	}

	log.Printf("[DVE Rental] Created %d default plans", len(service.defaultPlans))

	// Load existing data from database
	if err := service.loadFromDatabase(); err != nil {
		log.Printf("Warning: Failed to load rental data from database: %v", err)
	}

	// Initialize default rental plans
	service.initializeDefaultPlans()
	log.Printf("[DVE Rental] Initialized %d rental plans", len(service.rentalPlans))

	return service, nil
}

// SetServiceReferences sets references to other services
func (drs *DVERentalService) SetServiceReferences(dveManager, cdeService interface{}) {
	drs.mu.Lock()
	defer drs.mu.Unlock()

	drs.dveManager = dveManager
	drs.cdeService = cdeService
}

// SetContainerOrchestrator sets the container orchestrator reference
func (drs *DVERentalService) SetContainerOrchestrator(orchestrator interface{}) {
	drs.mu.Lock()
	defer drs.mu.Unlock()

	drs.containerOrchestrator = orchestrator
}

// SetBlockchainClient sets the blockchain client for payment verification
func (drs *DVERentalService) SetBlockchainClient(client BlockchainClientInterface) {
	drs.mu.Lock()
	defer drs.mu.Unlock()

	drs.blockchainClient = client
}

// SetTestMode enables or disables test mode for mock fallbacks
func (drs *DVERentalService) SetTestMode(enabled bool) {
	drs.mu.Lock()
	defer drs.mu.Unlock()

	drs.testMode = enabled
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

	// Start usage tracking routine
	go drs.usageTrackingRoutine()

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
func (drs *DVERentalService) CreateRental(req *objects.RentalRequest) (*objects.RentalResponse, error) {
	drs.mu.Lock()
	defer drs.mu.Unlock()

	// Validate rental plan
	plan, exists := drs.rentalPlans[req.PlanID]
	if !exists {
		return &objects.RentalResponse{
			Success: false,
			Error:   "Invalid rental plan ID",
		}, nil
	}

	// Validate duration
	if req.Duration < plan.MinDuration || req.Duration > plan.MaxDuration {
		return &objects.RentalResponse{
			Success: false,
			Error:   fmt.Sprintf("Duration must be between %d and %d seconds", plan.MinDuration, plan.MaxDuration),
		}, nil
	}

	// Calculate total cost
	hours := float64(req.Duration) / 3600.0
	totalCost := int64(hours * float64(plan.PricePerHour))

	// Verify NRN payment transaction on blockchain or use test mode fallback
	if drs.blockchainClient == nil {
		if drs.testMode {
			// In test mode, just require a payment tx hash as authorization proof
			if req.PaymentTxHash == "" {
				return &objects.RentalResponse{
					Success: false,
					Error:   "Payment transaction hash is required",
				}, nil
			}
			log.Printf("[Test Mode] Skipping blockchain verification, using payment authorization: %s", req.PaymentTxHash)
		} else {
			return &objects.RentalResponse{
				Success: false,
				Error:   "Blockchain client not configured",
			}, nil
		}
	} else {
		// Verify payment on blockchain
		systemWalletAddress := "knirv1system" // This should be configurable
		payment, err := drs.blockchainClient.VerifyPaymentTransaction(req.PaymentTxHash, totalCost, systemWalletAddress)
		if err != nil {
			log.Printf("Payment verification failed for tx %s: %v", req.PaymentTxHash, err)
			return &objects.RentalResponse{
				Success: false,
				Error:   "Payment verification failed: " + err.Error(),
			}, nil
		}

		if payment.Status != "confirmed" {
			return &objects.RentalResponse{
				Success: false,
				Error:   "Payment not confirmed on blockchain",
			}, nil
		}

		log.Printf("Payment verified: %d NRN received for rental", payment.Amount)
	}

	// Find available DVE node
	dveNodeID := drs.findAvailableDVENode(req.PreferredDVE)
	if dveNodeID == "" {
		return &objects.RentalResponse{
			Success: false,
			Error:   "No available DVE nodes",
		}, nil
	}

	// Create rental record
	rental := &objects.DVERental{
		ID:                 uuid.New().String(),
		UserID:             req.UserID,
		DVENodeID:          dveNodeID,
		NRNAmount:          totalCost,
		RentalDuration:     req.Duration,
		StartTime:          time.Now(),
		EndTime:            time.Now().Add(time.Duration(req.Duration) * time.Second),
		Status:             "active",
		PaymentTxHash:      req.PaymentTxHash,
		ResourceLimits:     plan.ResourceLimits,
		UsageMetrics:       objects.UsageMetrics{LastUpdated: time.Now()},
		AutoRenewalEnabled: req.AutoRenewalEnabled,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Provision CDE environment
	cdeEnvID, cdeURL, credentials, err := drs.provisionCDEEnvironment(rental)
	if err != nil {
		return &objects.RentalResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to provision CDE environment: %v", err),
		}, nil
	}

	rental.CDEEnvironmentID = cdeEnvID

	// ⭐ Provision TEE container for SSH access
	container, err := drs.provisionTEEContainer(rental)
	if err != nil {
		// Clean up CDE environment if container provisioning fails
		drs.cleanupCDEEnvironment(cdeEnvID)
		return &objects.RentalResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to provision TEE container: %v", err),
		}, nil
	}

	// Update rental with container information
	rental.ContainerID = container.ID
	rental.SSHUsername = container.Spec.SSHUsername
	rental.SSHPort = container.Spec.SSHPort
	rental.ProvisionedAt = time.Now()
	rental.ProvisioningStatus = "provisioned"

	// Store rental
	drs.activeRentals[rental.ID] = rental

	// Save to database
	if err := drs.saveRentalToDatabase(rental); err != nil {
		log.Printf("Warning: Failed to save rental to database: %v", err)
	}

	return &objects.RentalResponse{
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
func (drs *DVERentalService) GetActiveRentals(userID string) ([]*objects.DVERental, error) {
	drs.mu.RLock()
	defer drs.mu.RUnlock()

	var userRentals []*objects.DVERental
	for _, rental := range drs.activeRentals {
		if rental.UserID == userID && rental.Status == "active" {
			userRentals = append(userRentals, rental)
		}
	}

	return userRentals, nil
}

// GetAllUserRentals returns all rentals (active, expired, cancelled) for a user from database
func (drs *DVERentalService) GetAllUserRentals(userID string) ([]*objects.DVERental, error) {
	drs.mu.RLock()
	defer drs.mu.RUnlock()

	var userRentals []*objects.DVERental

	// Query database for all rentals belonging to this user
	err := drs.db.ViewTransaction(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			if len(key) > 8 && key[:8] == "rental:" {
				var rental objects.DVERental
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

// GetRentalByID returns a specific rental by ID
func (drs *DVERentalService) GetRentalByID(rentalID string) (*objects.DVERental, error) {
	drs.mu.RLock()
	defer drs.mu.RUnlock()

	// First check active rentals
	if rental, exists := drs.activeRentals[rentalID]; exists {
		return rental, nil
	}

	// If not in active rentals, check database
	var rental objects.DVERental
	err := drs.db.ViewTransaction(func(tx *buntdb.Tx) error {
		val, err := tx.Get("rental:" + rentalID)
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(val), &rental)
	})

	if err != nil {
		return nil, fmt.Errorf("rental not found: %w", err)
	}

	return &rental, nil
}

// GetRentalPlans returns all available rental plans
func (drs *DVERentalService) GetRentalPlans() ([]*objects.RentalPlan, error) {
	drs.mu.RLock()
	defer drs.mu.RUnlock()

	var plans []*objects.RentalPlan
	for _, plan := range drs.rentalPlans {
		if plan.IsActive {
			plans = append(plans, plan)
		}
	}

	return plans, nil
}

// GetRentalStats returns rental system statistics
func (drs *DVERentalService) GetRentalStats() (*objects.DVERentalStats, error) {
	drs.mu.RLock()
	defer drs.mu.RUnlock()

	stats := &objects.DVERentalStats{
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
	// Check if DVE manager is available
	if drs.dveManager == nil {
		log.Printf("DVE manager not available, using fallback node selection")
		if preferredDVE != "" {
			return preferredDVE
		}
		return "dve-node-" + uuid.New().String()[:8]
	}

	// Use DVE manager interface to find available nodes
	type DVEManagerInterface interface {
		GetAllNodes() []*objects.DVENode
		GetNode(nodeID string) (*objects.DVENode, error)
	}

	dveManager, ok := drs.dveManager.(DVEManagerInterface)
	if !ok {
		log.Printf("DVE manager interface mismatch, using fallback")
		if preferredDVE != "" {
			return preferredDVE
		}
		return "dve-node-" + uuid.New().String()[:8]
	}

	// If preferred node is specified, check if it's available
	if preferredDVE != "" {
		node, err := dveManager.GetNode(preferredDVE)
		if err == nil && node.Status == "online" {
			log.Printf("Using preferred DVE node: %s", preferredDVE)
			return preferredDVE
		}
		log.Printf("Preferred DVE node %s not available, selecting alternative", preferredDVE)
	}

	// Get all available nodes and select one
	allNodes := dveManager.GetAllNodes()
	var availableNodes []*objects.DVENode

	for _, node := range allNodes {
		if node.Status == "online" && node.ReputationScore >= 50 {
			availableNodes = append(availableNodes, node)
		}
	}

	if len(availableNodes) == 0 {
		log.Printf("No available DVE nodes found")
		return ""
	}

	// Simple selection: pick the first available node
	// In a real implementation, this could use load balancing
	selectedNode := availableNodes[0]
	log.Printf("Selected DVE node %s for rental", selectedNode.ID)

	return selectedNode.ID
}

// provisionCDEEnvironment provisions a CDE environment for the rental
func (drs *DVERentalService) provisionCDEEnvironment(rental *objects.DVERental) (string, string, objects.CDECredentials, error) {
	// Check if CDE service is available
	if drs.cdeService == nil {
		// Fallback to mock data if CDE service is not available
		envID := "cde-env-" + uuid.New().String()[:8]
		accessURL := fmt.Sprintf("https://cde.knirv.com/env/%s", envID)
		credentials := objects.CDECredentials{
			Username:    "user-" + rental.UserID[:8],
			Password:    "temp-" + uuid.New().String()[:12],
			AccessToken: "token-" + uuid.New().String(),
		}
		return envID, accessURL, credentials, nil
	}

	// Use actual CDE service
	type CDEServiceInterface interface {
		CreateEnvironment(userID, name string, envType interface{}, config map[string]interface{}) (*cde.CDEEnvironment, error)
		CreateSession(userID, envID string, connectionType string) (*cde.CDESession, error)
	}

	cdeService, ok := drs.cdeService.(CDEServiceInterface)
	if !ok {
		// Fallback to mock if interface doesn't match
		envID := "cde-env-" + uuid.New().String()[:8]
		accessURL := fmt.Sprintf("https://cde.knirv.com/env/%s", envID)
		credentials := objects.CDECredentials{
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
		return "", "", objects.CDECredentials{}, fmt.Errorf("failed to create CDE environment: %w", err)
	}

	// Extract environment ID from the created environment
	envID := env.ID

	// Create a session for the environment
	session, err := cdeService.CreateSession(rental.UserID, envID, "websocket")
	if err != nil {
		log.Printf("Warning: Failed to create CDE session: %v", err)
		// Continue without session - user can create one later
	}

	// Generate access URL and secure credentials
	accessURL := fmt.Sprintf("https://cde.knirv.com/env/%s", envID)
	credentials, err := drs.generateSecureCredentials(rental.UserID, envID)
	if err != nil {
		return "", "", objects.CDECredentials{}, fmt.Errorf("failed to generate secure credentials: %w", err)
	}

	// If session was created successfully, use session ID as access token
	if session != nil {
		credentials.AccessToken = session.ID
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

// generateSecureCredentials generates cryptographically secure credentials for CDE access
func (drs *DVERentalService) generateSecureCredentials(userID, envID string) (objects.CDECredentials, error) {
	// Generate secure password (32 bytes = 256 bits of entropy)
	passwordBytes := make([]byte, 32)
	if _, err := rand.Read(passwordBytes); err != nil {
		return objects.CDECredentials{}, fmt.Errorf("failed to generate secure password: %w", err)
	}
	password := base64.URLEncoding.EncodeToString(passwordBytes)

	// Generate secure access token (32 bytes)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return objects.CDECredentials{}, fmt.Errorf("failed to generate secure token: %w", err)
	}
	accessToken := base64.URLEncoding.EncodeToString(tokenBytes)

	// Create username based on user ID and environment
	username := fmt.Sprintf("user-%s-%s", userID[:8], envID[:8])

	credentials := objects.CDECredentials{
		Username:    username,
		Password:    password,
		AccessToken: accessToken,
	}

	log.Printf("Generated secure credentials for user %s in environment %s", userID, envID)
	return credentials, nil
}

// provisionTEEContainer provisions a TEE container for SSH access
func (drs *DVERentalService) provisionTEEContainer(rental *objects.DVERental) (*container.Container, error) {
	// Check if container orchestrator is available
	if drs.containerOrchestrator == nil {
		if drs.testMode {
			// Fallback to mock container only in test mode
			containerID := "container-" + uuid.New().String()[:8]
			mockContainer := &container.Container{
				ID:        containerID,
				Status:    container.ContainerStatusRunning,
				CreatedAt: time.Now(),
				Spec: &container.ContainerSpec{
					SSHPort:     22145, // Mock port
					SSHUsername: fmt.Sprintf("rental-user-%s", rental.ID[:8]),
				},
				Runtime: "mock",
			}
			log.Printf("Provisioned mock TEE container %s for rental %s (test mode)", containerID, rental.ID)
			return mockContainer, nil
		}
		return nil, fmt.Errorf("container orchestrator not available and not in test mode")
	}

	// Use actual container orchestrator
	type ContainerOrchestratorInterface interface {
		ProvisionContainer(rentalID string) (*container.Container, error)
	}

	orchestrator, ok := drs.containerOrchestrator.(ContainerOrchestratorInterface)
	if !ok {
		if drs.testMode {
			// Fallback to mock if interface doesn't match (only in test mode)
			containerID := "container-" + uuid.New().String()[:8]
			mockContainer := &container.Container{
				ID:        containerID,
				Status:    container.ContainerStatusRunning,
				CreatedAt: time.Now(),
				Spec: &container.ContainerSpec{
					SSHPort:     22145, // Mock port
					SSHUsername: fmt.Sprintf("rental-user-%s", rental.ID[:8]),
				},
				Runtime: "mock",
			}
			log.Printf("Provisioned fallback TEE container %s for rental %s (test mode)", containerID, rental.ID)
			return mockContainer, nil
		}
		return nil, fmt.Errorf("container orchestrator interface mismatch and not in test mode")
	}

	// Provision the container
	provisionedContainer, err := orchestrator.ProvisionContainer(rental.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to provision TEE container: %w", err)
	}

	log.Printf("Successfully provisioned TEE container %s for rental %s", provisionedContainer.ID, rental.ID)
	return provisionedContainer, nil
}
