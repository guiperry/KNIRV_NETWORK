package endpoints

import (
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/objects"
)

// EndpointRegistry manages TEE endpoint information for DVE rentals
type EndpointRegistry struct {
	endpoints map[string]*objects.TEEEndpoint // endpointID -> endpoint
	rentalEndpoints map[string][]*objects.TEEEndpoint // rentalID -> endpoints
	mutex     sync.RWMutex
}

// NewEndpointRegistry creates a new endpoint registry
func NewEndpointRegistry() *EndpointRegistry {
	er := &EndpointRegistry{
		endpoints:       make(map[string]*objects.TEEEndpoint),
		rentalEndpoints: make(map[string][]*objects.TEEEndpoint),
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

	// Store endpoint
	er.endpoints[endpoint.ID] = endpoint

	// Add to rental's endpoint list
	if er.rentalEndpoints[rentalID] == nil {
		er.rentalEndpoints[rentalID] = make([]*objects.TEEEndpoint, 0)
	}
	er.rentalEndpoints[rentalID] = append(er.rentalEndpoints[rentalID], endpoint)

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
			// Remove from endpoints map
			delete(er.endpoints, endpoint.ID)

			// Remove from rental's endpoint list
			er.rentalEndpoints[rentalID] = append(endpoints[:i], endpoints[i+1:]...)

			log.Printf("Unregistered %s endpoint %s for rental %s",
				endpointType, endpoint.ID, rentalID)
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

// HealthCheck performs a health check on an endpoint
func (er *EndpointRegistry) HealthCheck(endpointID string) (bool, error) {
	endpoint, err := er.GetEndpoint(endpointID)
	if err != nil {
		return false, err
	}

	// Perform basic health check using endpoint information
	// This is a simplified implementation - in production, this would include:
	// 1. Network connectivity test
	// 2. Service availability check
	// 3. Response time validation

	log.Printf("HealthCheck: endpoint %s (%s:%d) is active and within expiration time",
		endpoint.ID, endpoint.Host, endpoint.Port)

	// For now, consider endpoint healthy if it's not expired (already checked in GetEndpoint)
	return true, nil
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
