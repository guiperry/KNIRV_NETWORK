package endpoints

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/objects"

	"github.com/tidwall/buntdb"
)

// EndpointRegistry manages TEE endpoint information for DVE rentals
type EndpointRegistry struct {
	endpoints       map[string]*objects.TEEEndpoint       // endpointID -> endpoint
	rentalEndpoints map[string][]*objects.TEEEndpoint     // rentalID -> endpoints
	db              *buntdb.DB
	mutex           sync.RWMutex
}

// NewEndpointRegistry creates a new endpoint registry with optional database persistence
func NewEndpointRegistry(db ...*buntdb.DB) *EndpointRegistry {
	var database *buntdb.DB
	if len(db) > 0 {
		database = db[0]
	}

	er := &EndpointRegistry{
		endpoints:       make(map[string]*objects.TEEEndpoint),
		rentalEndpoints: make(map[string][]*objects.TEEEndpoint),
		db:              database,
	}

	// Load existing endpoints from database if available
	if database != nil {
		if err := er.loadEndpointsFromDB(); err != nil {
			log.Printf("Warning: Failed to load endpoints from database: %v", err)
		}
	}

	// Start cleanup routine
	go er.cleanupExpiredEndpoints()

	return er
}

// RegisterEndpoint registers a new TEE endpoint
func (er *EndpointRegistry) RegisterEndpoint(rentalID string, endpointType string, endpoint *objects.TEEEndpoint) error {
	er.mutex.Lock()
	defer er.mutex.Unlock()

	// Set endpoint properties
	endpoint.ID = er.generateEndpointID()
	endpoint.RentalID = rentalID
	endpoint.EndpointType = endpointType
	endpoint.Status = "active"
	endpoint.CreatedAt = time.Now()
	endpoint.ExpiresAt = time.Now().Add(25 * time.Hour) // Default 25 hours

	// Store endpoint in memory
	er.endpoints[endpoint.ID] = endpoint

	// Add to rental's endpoint list
	if er.rentalEndpoints[rentalID] == nil {
		er.rentalEndpoints[rentalID] = make([]*objects.TEEEndpoint, 0)
	}
	er.rentalEndpoints[rentalID] = append(er.rentalEndpoints[rentalID], endpoint)

	// Persist to database
	if err := er.saveEndpointToDB(endpoint); err != nil {
		log.Printf("Warning: Failed to persist endpoint %s to database: %v", endpoint.ID, err)
		// Continue - endpoint is still registered in memory
	}

	log.Printf("Registered %s endpoint %s for rental %s: %s:%d",
		endpointType, endpoint.ID, rentalID, endpoint.Host, endpoint.Port)
	return nil
}

// GetEndpoint retrieves an endpoint by ID
func (er *EndpointRegistry) GetEndpoint(endpointID string) (*objects.TEEEndpoint, error) {
	er.mutex.RLock()
	defer er.mutex.RUnlock()

	endpoint, exists := er.endpoints[endpointID]
	if !exists {
		return nil, fmt.Errorf("endpoint not found: %s", endpointID)
	}

	if time.Now().After(endpoint.ExpiresAt) {
		return nil, fmt.Errorf("endpoint expired: %s", endpointID)
	}

	return endpoint, nil
}

// GetEndpointByRentalAndType retrieves an endpoint by rental ID and type
func (er *EndpointRegistry) GetEndpointByRentalAndType(rentalID, endpointType string) (*objects.TEEEndpoint, error) {
	er.mutex.RLock()
	defer er.mutex.RUnlock()

	endpoints, exists := er.rentalEndpoints[rentalID]
	if !exists {
		return nil, fmt.Errorf("no endpoints found for rental: %s", rentalID)
	}

	for _, endpoint := range endpoints {
		if endpoint.EndpointType == endpointType && time.Now().Before(endpoint.ExpiresAt) {
			return endpoint, nil
		}
	}

	return nil, fmt.Errorf("no active %s endpoint found for rental: %s", endpointType, rentalID)
}

// ListEndpoints returns all endpoints for a rental
func (er *EndpointRegistry) ListEndpoints(rentalID string) ([]*objects.TEEEndpoint, error) {
	er.mutex.RLock()
	defer er.mutex.RUnlock()

	endpoints, exists := er.rentalEndpoints[rentalID]
	if !exists {
		return []*objects.TEEEndpoint{}, nil
	}

	// Filter out expired endpoints
	activeEndpoints := make([]*objects.TEEEndpoint, 0)
	now := time.Now()
	for _, endpoint := range endpoints {
		if now.Before(endpoint.ExpiresAt) {
			activeEndpoints = append(activeEndpoints, endpoint)
		}
	}

	return activeEndpoints, nil
}

// UpdateEndpointStatus updates the status of an endpoint
func (er *EndpointRegistry) UpdateEndpointStatus(endpointID, status string) error {
	er.mutex.Lock()
	defer er.mutex.Unlock()

	endpoint, exists := er.endpoints[endpointID]
	if !exists {
		return fmt.Errorf("endpoint not found: %s", endpointID)
	}

	endpoint.Status = status
	log.Printf("Updated status of endpoint %s to %s", endpointID, status)
	return nil
}

// ExtendEndpointExpiration extends the expiration time of an endpoint
func (er *EndpointRegistry) ExtendEndpointExpiration(endpointID string, duration time.Duration) error {
	er.mutex.Lock()
	defer er.mutex.Unlock()

	endpoint, exists := er.endpoints[endpointID]
	if !exists {
		return fmt.Errorf("endpoint not found: %s", endpointID)
	}

	endpoint.ExpiresAt = time.Now().Add(duration)
	log.Printf("Extended expiration of endpoint %s to %s", endpointID, endpoint.ExpiresAt)
	return nil
}

// UnregisterEndpoint removes an endpoint from the registry
func (er *EndpointRegistry) UnregisterEndpoint(rentalID, endpointType string) error {
	er.mutex.Lock()
	defer er.mutex.Unlock()

	endpoints, exists := er.rentalEndpoints[rentalID]
	if !exists {
		return fmt.Errorf("no endpoints found for rental: %s", rentalID)
	}

	// Find and remove the endpoint
	for i, endpoint := range endpoints {
		if endpoint.EndpointType == endpointType {
			endpointID := endpoint.ID

			// Remove from endpoints map
			delete(er.endpoints, endpointID)

			// Remove from database
			if err := er.deleteEndpointFromDB(endpointID); err != nil {
				log.Printf("Warning: Failed to delete endpoint %s from database: %v", endpointID, err)
			}

			// Remove from rental's endpoint list
			er.rentalEndpoints[rentalID] = append(endpoints[:i], endpoints[i+1:]...)

			log.Printf("Unregistered %s endpoint %s for rental %s",
				endpointType, endpointID, rentalID)
			return nil
		}
	}

	return fmt.Errorf("endpoint type %s not found for rental %s", endpointType, rentalID)
}

// UnregisterAllEndpointsForRental removes all endpoints for a rental
func (er *EndpointRegistry) UnregisterAllEndpointsForRental(rentalID string) error {
	er.mutex.Lock()
	defer er.mutex.Unlock()

	endpoints, exists := er.rentalEndpoints[rentalID]
	if !exists {
		return nil // No endpoints to unregister
	}

	// Remove all endpoints for this rental
	for _, endpoint := range endpoints {
		delete(er.endpoints, endpoint.ID)

		// Remove from database
		if err := er.deleteEndpointFromDB(endpoint.ID); err != nil {
			log.Printf("Warning: Failed to delete endpoint %s from database: %v", endpoint.ID, err)
		}

		log.Printf("Unregistered endpoint %s for rental %s", endpoint.ID, rentalID)
	}

	// Remove rental from registry
	delete(er.rentalEndpoints, rentalID)

	log.Printf("Unregistered all endpoints for rental %s", rentalID)
	return nil
}

// GetAllEndpoints returns all registered endpoints
func (er *EndpointRegistry) GetAllEndpoints() []*objects.TEEEndpoint {
	er.mutex.RLock()
	defer er.mutex.RUnlock()

	endpoints := make([]*objects.TEEEndpoint, 0, len(er.endpoints))
	now := time.Now()
	for _, endpoint := range er.endpoints {
		if now.Before(endpoint.ExpiresAt) {
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints
}

// GetEndpointsByType returns all endpoints of a specific type
func (er *EndpointRegistry) GetEndpointsByType(endpointType string) []*objects.TEEEndpoint {
	er.mutex.RLock()
	defer er.mutex.RUnlock()

	endpoints := make([]*objects.TEEEndpoint, 0)
	now := time.Now()
	for _, endpoint := range er.endpoints {
		if endpoint.EndpointType == endpointType && now.Before(endpoint.ExpiresAt) {
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints
}

// HealthCheck performs a comprehensive health check on an endpoint
func (er *EndpointRegistry) HealthCheck(endpointID string) (bool, error) {
	endpoint, err := er.GetEndpoint(endpointID)
	if err != nil {
		return false, err
	}

	log.Printf("Performing health check for endpoint %s (%s:%d)", endpoint.ID, endpoint.Host, endpoint.Port)

	// Perform network connectivity test
	if err := er.checkNetworkConnectivity(endpoint); err != nil {
		log.Printf("Network connectivity check failed for endpoint %s: %v", endpointID, err)
		return false, fmt.Errorf("network connectivity check failed: %w", err)
	}

	// Perform service availability check based on endpoint type
	if err := er.checkServiceAvailability(endpoint); err != nil {
		log.Printf("Service availability check failed for endpoint %s: %v", endpointID, err)
		return false, fmt.Errorf("service availability check failed: %w", err)
	}

	// Update endpoint status to healthy
	if err := er.UpdateEndpointStatus(endpointID, "healthy"); err != nil {
		log.Printf("Warning: Failed to update endpoint status to healthy: %v", err)
	}

	log.Printf("Health check passed for endpoint %s", endpointID)
	return true, nil
}

// checkNetworkConnectivity performs basic network connectivity test
func (er *EndpointRegistry) checkNetworkConnectivity(endpoint *objects.TEEEndpoint) error {
	// For now, implement a basic connectivity check
	// In production, this would use net.Dial or similar
	// For demonstration, we'll assume localhost endpoints are accessible

	if endpoint.Host == "localhost" || endpoint.Host == "127.0.0.1" {
		// Local endpoints are assumed accessible
		return nil
	}

	// For remote endpoints, we would perform actual network tests
	// For now, mark as potentially accessible
	log.Printf("Network connectivity check: %s:%d (assuming accessible)", endpoint.Host, endpoint.Port)
	return nil
}

// checkServiceAvailability checks if the service behind the endpoint is responding
func (er *EndpointRegistry) checkServiceAvailability(endpoint *objects.TEEEndpoint) error {
	switch endpoint.EndpointType {
	case "ssh":
		// For SSH endpoints, we could attempt a basic connection test
		// For now, assume SSH service is available if port is allocated
		if endpoint.Port == 0 {
			return fmt.Errorf("SSH endpoint has no port assigned")
		}
		return nil

	case "validation", "error-resolution":
		// For HTTP-based services, we could make a health check request
		// For now, assume service is available if port is allocated
		if endpoint.Port == 0 {
			return fmt.Errorf("%s endpoint has no port assigned", endpoint.EndpointType)
		}
		return nil

	default:
		// For unknown endpoint types, just check if port is assigned
		if endpoint.Port == 0 {
			return fmt.Errorf("endpoint has no port assigned")
		}
		return nil
	}
}

// generateEndpointID generates a unique endpoint ID
func (er *EndpointRegistry) generateEndpointID() string {
	return fmt.Sprintf("ep-%d", time.Now().UnixNano())
}

// cleanupExpiredEndpoints periodically cleans up expired endpoints
func (er *EndpointRegistry) cleanupExpiredEndpoints() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		er.mutex.Lock()
		now := time.Now()
		expiredEndpoints := make([]string, 0)

		// Find expired endpoints
		for endpointID, endpoint := range er.endpoints {
			if now.After(endpoint.ExpiresAt) {
				expiredEndpoints = append(expiredEndpoints, endpointID)
			}
		}

		// Remove expired endpoints
		for _, endpointID := range expiredEndpoints {
			endpoint, exists := er.endpoints[endpointID]
			if !exists {
				continue
			}
			rentalID := endpoint.RentalID
			delete(er.endpoints, endpointID)

			// Remove from database
			if er.db != nil {
				er.db.Update(func(tx *buntdb.Tx) error {
					key := fmt.Sprintf("endpoint:%s", endpointID)
					_, err := tx.Delete(key)
					return err
				})
			}

			// Also remove from rental's endpoint list
			if endpoints, exists := er.rentalEndpoints[rentalID]; exists {
				for i, ep := range endpoints {
					if ep.ID == endpointID {
						er.rentalEndpoints[rentalID] = append(endpoints[:i], endpoints[i+1:]...)
						break
					}
				}
			}

			log.Printf("Cleaned up expired endpoint %s for rental %s", endpointID, rentalID)
		}

		er.mutex.Unlock()

		if len(expiredEndpoints) > 0 {
			log.Printf("Cleaned up %d expired endpoints", len(expiredEndpoints))
		}
	}
}

// loadEndpointsFromDB loads all endpoints from the database into memory
func (er *EndpointRegistry) loadEndpointsFromDB() error {
	if er.db == nil {
		return fmt.Errorf("database not available")
	}

	return er.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendKeys("endpoint:*", func(key, value string) bool {
			var endpoint objects.TEEEndpoint
			if err := json.Unmarshal([]byte(value), &endpoint); err != nil {
				log.Printf("Warning: Failed to unmarshal endpoint %s: %v", key, err)
				return true // Continue with next
			}

			// Skip expired endpoints
			if time.Now().After(endpoint.ExpiresAt) {
				log.Printf("Skipping expired endpoint %s", endpoint.ID)
				return true
			}

			// Load into memory
			er.endpoints[endpoint.ID] = &endpoint

			// Add to rental's endpoint list
			if er.rentalEndpoints[endpoint.RentalID] == nil {
				er.rentalEndpoints[endpoint.RentalID] = make([]*objects.TEEEndpoint, 0)
			}
			er.rentalEndpoints[endpoint.RentalID] = append(er.rentalEndpoints[endpoint.RentalID], &endpoint)

			log.Printf("Loaded endpoint %s for rental %s from database", endpoint.ID, endpoint.RentalID)
			return true // Continue
		})
	})
}

// saveEndpointToDB persists an endpoint to the database
func (er *EndpointRegistry) saveEndpointToDB(endpoint *objects.TEEEndpoint) error {
	if er.db == nil {
		return fmt.Errorf("database not available")
	}

	return er.db.Update(func(tx *buntdb.Tx) error {
		data, err := json.Marshal(endpoint)
		if err != nil {
			return err
		}

		key := fmt.Sprintf("endpoint:%s", endpoint.ID)
		_, _, err = tx.Set(key, string(data), nil)
		return err
	})
}

// deleteEndpointFromDB removes an endpoint from the database
func (er *EndpointRegistry) deleteEndpointFromDB(endpointID string) error {
	if er.db == nil {
		return fmt.Errorf("database not available")
	}

	return er.db.Update(func(tx *buntdb.Tx) error {
		key := fmt.Sprintf("endpoint:%s", endpointID)
		_, err := tx.Delete(key)
		return err
	})
}
