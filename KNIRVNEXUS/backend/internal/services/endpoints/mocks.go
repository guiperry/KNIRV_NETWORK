package endpoints

import (
	"context"
	"fmt"
	"time"

	"backend_server/internal/objects"
)

// MockEndpointRegistry is a mock implementation of EndpointRegistry for testing
type MockEndpointRegistry struct {
	endpoints map[string]*objects.TEEEndpoint
	rentalEndpoints map[string][]*objects.TEEEndpoint
}

func NewMockEndpointRegistry() *MockEndpointRegistry {
	return &MockEndpointRegistry{
		endpoints: make(map[string]*objects.TEEEndpoint),
		rentalEndpoints: make(map[string][]*objects.TEEEndpoint),
	}
}

func (m *MockEndpointRegistry) RegisterEndpoint(ctx context.Context, rentalID string, endpointType string, endpoint *objects.TEEEndpoint) error {
	endpointID := fmt.Sprintf("endpoint-%s-%s", endpointType, rentalID)
	endpoint.ID = endpointID
	endpoint.RentalID = rentalID
	endpoint.EndpointType = endpointType
	endpoint.Status = "active"
	endpoint.CreatedAt = time.Now()
	endpoint.ExpiresAt = time.Now().Add(24 * time.Hour)

	m.endpoints[endpointID] = endpoint

	if m.rentalEndpoints[rentalID] == nil {
		m.rentalEndpoints[rentalID] = make([]*objects.TEEEndpoint, 0)
	}
	m.rentalEndpoints[rentalID] = append(m.rentalEndpoints[rentalID], endpoint)

	return nil
}

func (m *MockEndpointRegistry) GetEndpoint(ctx context.Context, endpointID string) (*objects.TEEEndpoint, error) {
	endpoint, exists := m.endpoints[endpointID]
	if !exists {
		return nil, fmt.Errorf("endpoint not found")
	}
	return endpoint, nil
}

func (m *MockEndpointRegistry) GetEndpointByRentalAndType(ctx context.Context, rentalID, endpointType string) (*objects.TEEEndpoint, error) {
	endpoints, exists := m.rentalEndpoints[rentalID]
	if !exists {
		return nil, fmt.Errorf("no endpoints found for rental")
	}

	for _, endpoint := range endpoints {
		if endpoint.EndpointType == endpointType {
			return endpoint, nil
		}
	}

	return nil, fmt.Errorf("endpoint type not found")
}

func (m *MockEndpointRegistry) ListEndpoints(ctx context.Context, rentalID string) ([]*objects.TEEEndpoint, error) {
	endpoints, exists := m.rentalEndpoints[rentalID]
	if !exists {
		return []*objects.TEEEndpoint{}, nil
	}
	return endpoints, nil
}

func (m *MockEndpointRegistry) UpdateEndpointStatus(ctx context.Context, endpointID, status string) error {
	if endpoint, exists := m.endpoints[endpointID]; exists {
		endpoint.Status = status
	}
	return nil
}

func (m *MockEndpointRegistry) ExtendEndpointExpiration(ctx context.Context, endpointID string, duration time.Duration) error {
	if endpoint, exists := m.endpoints[endpointID]; exists {
		endpoint.ExpiresAt = time.Now().Add(duration)
	}
	return nil
}

func (m *MockEndpointRegistry) UnregisterEndpoint(ctx context.Context, rentalID, endpointType string) error {
	endpoints, exists := m.rentalEndpoints[rentalID]
	if !exists {
		return fmt.Errorf("no endpoints found for rental")
	}

	for i, endpoint := range endpoints {
		if endpoint.EndpointType == endpointType {
			// Remove from endpoints map
			delete(m.endpoints, endpoint.ID)
			// Remove from rental's endpoint list
			m.rentalEndpoints[rentalID] = append(endpoints[:i], endpoints[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("endpoint type not found")
}

func (m *MockEndpointRegistry) UnregisterAllEndpointsForRental(ctx context.Context, rentalID string) error {
	if endpoints, exists := m.rentalEndpoints[rentalID]; exists {
		for _, endpoint := range endpoints {
			delete(m.endpoints, endpoint.ID)
		}
		delete(m.rentalEndpoints, rentalID)
	}
	return nil
}

func (m *MockEndpointRegistry) GetAllEndpoints(ctx context.Context) []*objects.TEEEndpoint {
	endpoints := make([]*objects.TEEEndpoint, 0, len(m.endpoints))
	for _, endpoint := range m.endpoints {
		endpoints = append(endpoints, endpoint)
	}
	return endpoints
}

func (m *MockEndpointRegistry) GetEndpointsByType(ctx context.Context, endpointType string) []*objects.TEEEndpoint {
	endpoints := make([]*objects.TEEEndpoint, 0)
	for _, endpoint := range m.endpoints {
		if endpoint.EndpointType == endpointType {
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints
}

func (m *MockEndpointRegistry) HealthCheck(ctx context.Context, endpointID string) (bool, error) {
	_, exists := m.endpoints[endpointID]
	return exists, nil
}
