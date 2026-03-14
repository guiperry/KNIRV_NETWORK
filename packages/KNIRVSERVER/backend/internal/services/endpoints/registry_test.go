package endpoints

import (
	"testing"
	"time"

	"backend_server/internal/objects"
)

func TestNewEndpointRegistry(t *testing.T) {
	registry := NewEndpointRegistry()

	if registry == nil {
		t.Fatal("NewEndpointRegistry returned nil")
	}

	if registry.endpoints == nil {
		t.Error("Endpoints map not initialized")
	}

	if registry.rentalEndpoints == nil {
		t.Error("Rental endpoints map not initialized")
	}
}

func TestEndpointRegistry_RegisterEndpoint(t *testing.T) {
	registry := NewEndpointRegistry()

	rentalID := "rental-123"
	endpointType := "api"
	endpoint := &objects.TEEEndpoint{
		Host: "localhost",
		Port: 8080,
	}

	err := registry.RegisterEndpoint(rentalID, endpointType, endpoint)
	if err != nil {
		t.Fatalf("Failed to register endpoint: %v", err)
	}

	if endpoint.ID == "" {
		t.Error("Endpoint ID not set")
	}

	if endpoint.RentalID != rentalID {
		t.Errorf("Expected RentalID %s, got %s", rentalID, endpoint.RentalID)
	}

	if endpoint.EndpointType != endpointType {
		t.Errorf("Expected EndpointType %s, got %s", endpointType, endpoint.EndpointType)
	}

	if endpoint.Status != "active" {
		t.Errorf("Expected status 'active', got %s", endpoint.Status)
	}

	if endpoint.CreatedAt.IsZero() {
		t.Error("CreatedAt not set")
	}

	if endpoint.ExpiresAt.IsZero() {
		t.Error("ExpiresAt not set")
	}
}

func TestEndpointRegistry_GetEndpoint(t *testing.T) {
	registry := NewEndpointRegistry()

	rentalID := "rental-123"
	endpointType := "api"
	endpoint := &objects.TEEEndpoint{
		Host: "localhost",
		Port: 8080,
	}

	err := registry.RegisterEndpoint(rentalID, endpointType, endpoint)
	if err != nil {
		t.Fatalf("Failed to register endpoint: %v", err)
	}

	// Get the endpoint
	retrievedEndpoint, err := registry.GetEndpoint(endpoint.ID)
	if err != nil {
		t.Fatalf("Failed to get endpoint: %v", err)
	}

	if retrievedEndpoint.ID != endpoint.ID {
		t.Errorf("Expected endpoint ID %s, got %s", endpoint.ID, retrievedEndpoint.ID)
	}

	if retrievedEndpoint.Host != endpoint.Host {
		t.Errorf("Expected host %s, got %s", endpoint.Host, retrievedEndpoint.Host)
	}

	if retrievedEndpoint.Port != endpoint.Port {
		t.Errorf("Expected port %d, got %d", endpoint.Port, retrievedEndpoint.Port)
	}
}

func TestEndpointRegistry_GetEndpoint_NotFound(t *testing.T) {
	registry := NewEndpointRegistry()

	_, err := registry.GetEndpoint("non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent endpoint")
	}
}

func TestEndpointRegistry_GetEndpoint_Expired(t *testing.T) {
	registry := NewEndpointRegistry()

	rentalID := "rental-123"
	endpointType := "api"
	endpoint := &objects.TEEEndpoint{
		Host: "localhost",
		Port: 8080,
	}

	err := registry.RegisterEndpoint(rentalID, endpointType, endpoint)
	if err != nil {
		t.Fatalf("Failed to register endpoint: %v", err)
	}

	// Manually expire the endpoint
	endpoint.ExpiresAt = time.Now().Add(-1 * time.Hour)

	_, err = registry.GetEndpoint(endpoint.ID)
	if err == nil {
		t.Error("Expected error for expired endpoint")
	}
}

func TestEndpointRegistry_GetEndpointByRentalAndType(t *testing.T) {
	registry := NewEndpointRegistry()

	rentalID := "rental-123"
	endpointType := "api"
	endpoint := &objects.TEEEndpoint{
		Host: "localhost",
		Port: 8080,
	}

	err := registry.RegisterEndpoint(rentalID, endpointType, endpoint)
	if err != nil {
		t.Fatalf("Failed to register endpoint: %v", err)
	}

	retrievedEndpoint, err := registry.GetEndpointByRentalAndType(rentalID, endpointType)
	if err != nil {
		t.Fatalf("Failed to get endpoint by rental and type: %v", err)
	}

	if retrievedEndpoint.ID != endpoint.ID {
		t.Errorf("Expected endpoint ID %s, got %s", endpoint.ID, retrievedEndpoint.ID)
	}
}

func TestEndpointRegistry_GetEndpointByRentalAndType_NotFound(t *testing.T) {
	registry := NewEndpointRegistry()

	_, err := registry.GetEndpointByRentalAndType("non-existent-rental", "api")
	if err == nil {
		t.Error("Expected error for non-existent rental")
	}

	rentalID := "rental-123"
	endpointType := "api"
	endpoint := &objects.TEEEndpoint{
		Host: "localhost",
		Port: 8080,
	}

	err = registry.RegisterEndpoint(rentalID, endpointType, endpoint)
	if err != nil {
		t.Fatalf("Failed to register endpoint: %v", err)
	}

	_, err = registry.GetEndpointByRentalAndType(rentalID, "non-existent-type")
	if err == nil {
		t.Error("Expected error for non-existent endpoint type")
	}
}

func TestEndpointRegistry_ListEndpoints(t *testing.T) {
	registry := NewEndpointRegistry()

	rentalID := "rental-123"

	// Register multiple endpoints
	for i := 0; i < 3; i++ {
		endpoint := &objects.TEEEndpoint{
			Host: "localhost",
			Port: 8080 + i,
		}
		err := registry.RegisterEndpoint(rentalID, "api", endpoint)
		if err != nil {
			t.Fatalf("Failed to register endpoint %d: %v", i, err)
		}
	}

	endpoints, err := registry.ListEndpoints(rentalID)
	if err != nil {
		t.Fatalf("Failed to list endpoints: %v", err)
	}

	if len(endpoints) != 3 {
		t.Errorf("Expected 3 endpoints, got %d", len(endpoints))
	}

	for _, endpoint := range endpoints {
		if endpoint.RentalID != rentalID {
			t.Errorf("Expected RentalID %s, got %s", rentalID, endpoint.RentalID)
		}
	}
}

func TestEndpointRegistry_ListEndpoints_EmptyRental(t *testing.T) {
	registry := NewEndpointRegistry()

	endpoints, err := registry.ListEndpoints("non-existent-rental")
	if err != nil {
		t.Fatalf("Failed to list endpoints: %v", err)
	}

	if len(endpoints) != 0 {
		t.Errorf("Expected 0 endpoints, got %d", len(endpoints))
	}
}

func TestEndpointRegistry_UpdateEndpointStatus(t *testing.T) {
	registry := NewEndpointRegistry()

	rentalID := "rental-123"
	endpointType := "api"
	endpoint := &objects.TEEEndpoint{
		Host: "localhost",
		Port: 8080,
	}

	err := registry.RegisterEndpoint(rentalID, endpointType, endpoint)
	if err != nil {
		t.Fatalf("Failed to register endpoint: %v", err)
	}

	newStatus := "inactive"
	err = registry.UpdateEndpointStatus(endpoint.ID, newStatus)
	if err != nil {
		t.Fatalf("Failed to update endpoint status: %v", err)
	}

	// Verify status was updated
	retrievedEndpoint, err := registry.GetEndpoint(endpoint.ID)
	if err != nil {
		t.Fatalf("Failed to get endpoint: %v", err)
	}

	if retrievedEndpoint.Status != newStatus {
		t.Errorf("Expected status %s, got %s", newStatus, retrievedEndpoint.Status)
	}
}

func TestEndpointRegistry_UpdateEndpointStatus_NotFound(t *testing.T) {
	registry := NewEndpointRegistry()

	err := registry.UpdateEndpointStatus("non-existent-id", "inactive")
	if err == nil {
		t.Error("Expected error for non-existent endpoint")
	}
}

func TestEndpointRegistry_ExtendEndpointExpiration(t *testing.T) {
	registry := NewEndpointRegistry()

	rentalID := "rental-123"
	endpointType := "api"
	endpoint := &objects.TEEEndpoint{
		Host: "localhost",
		Port: 8080,
	}

	err := registry.RegisterEndpoint(rentalID, endpointType, endpoint)
	if err != nil {
		t.Fatalf("Failed to register endpoint: %v", err)
	}

	originalExpiresAt := endpoint.ExpiresAt
	extensionDuration := 1 * time.Hour

	err = registry.ExtendEndpointExpiration(endpoint.ID, extensionDuration)
	if err != nil {
		t.Fatalf("Failed to extend endpoint expiration: %v", err)
	}

	// Check if the endpoint's expiration was extended
	// The ExtendEndpointExpiration method sets ExpiresAt = time.Now().Add(duration)
	// So it should be approximately extensionDuration from now
	expectedExpiresAt := time.Now().Add(extensionDuration)
	if endpoint.ExpiresAt.Before(expectedExpiresAt.Add(-2*time.Second)) || endpoint.ExpiresAt.After(expectedExpiresAt.Add(2*time.Second)) {
		t.Errorf("Endpoint expiration not set correctly. Expected around: %v, Got: %v", expectedExpiresAt, endpoint.ExpiresAt)
	}

	// Also verify it was actually changed from the original
	if endpoint.ExpiresAt.Equal(originalExpiresAt) {
		t.Error("Endpoint expiration was not changed")
	}
}

func TestEndpointRegistry_ExtendEndpointExpiration_NotFound(t *testing.T) {
	registry := NewEndpointRegistry()

	err := registry.ExtendEndpointExpiration("non-existent-id", 1*time.Hour)
	if err == nil {
		t.Error("Expected error for non-existent endpoint")
	}
}

func TestEndpointRegistry_UnregisterEndpoint(t *testing.T) {
	registry := NewEndpointRegistry()

	rentalID := "rental-123"
	endpointType := "api"
	endpoint := &objects.TEEEndpoint{
		Host: "localhost",
		Port: 8080,
	}

	err := registry.RegisterEndpoint(rentalID, endpointType, endpoint)
	if err != nil {
		t.Fatalf("Failed to register endpoint: %v", err)
	}

	err = registry.UnregisterEndpoint(rentalID, endpointType)
	if err != nil {
		t.Fatalf("Failed to unregister endpoint: %v", err)
	}

	// Verify endpoint was removed
	_, err = registry.GetEndpoint(endpoint.ID)
	if err == nil {
		t.Error("Expected error when getting unregistered endpoint")
	}
}

func TestEndpointRegistry_UnregisterEndpoint_NotFound(t *testing.T) {
	registry := NewEndpointRegistry()

	err := registry.UnregisterEndpoint("non-existent-rental", "api")
	if err == nil {
		t.Error("Expected error for non-existent rental")
	}

	rentalID := "rental-123"
	endpointType := "api"
	endpoint := &objects.TEEEndpoint{
		Host: "localhost",
		Port: 8080,
	}

	err = registry.RegisterEndpoint(rentalID, endpointType, endpoint)
	if err != nil {
		t.Fatalf("Failed to register endpoint: %v", err)
	}

	err = registry.UnregisterEndpoint(rentalID, "non-existent-type")
	if err == nil {
		t.Error("Expected error for non-existent endpoint type")
	}
}

func TestEndpointRegistry_UnregisterAllEndpointsForRental(t *testing.T) {
	registry := NewEndpointRegistry()

	rentalID := "rental-123"

	// Register multiple endpoints
	for i := 0; i < 3; i++ {
		endpoint := &objects.TEEEndpoint{
			Host: "localhost",
			Port: 8080 + i,
		}
		err := registry.RegisterEndpoint(rentalID, "api", endpoint)
		if err != nil {
			t.Fatalf("Failed to register endpoint %d: %v", i, err)
		}
	}

	err := registry.UnregisterAllEndpointsForRental(rentalID)
	if err != nil {
		t.Fatalf("Failed to unregister all endpoints: %v", err)
	}

	// Verify all endpoints were removed
	endpoints, err := registry.ListEndpoints(rentalID)
	if err != nil {
		t.Fatalf("Failed to list endpoints: %v", err)
	}

	if len(endpoints) != 0 {
		t.Errorf("Expected 0 endpoints after unregistering all, got %d", len(endpoints))
	}
}

func TestEndpointRegistry_GetAllEndpoints(t *testing.T) {
	registry := NewEndpointRegistry()

	// Register multiple endpoints for different rentals
	rentals := []string{"rental-1", "rental-2"}
	for i, rentalID := range rentals {
		endpoint := &objects.TEEEndpoint{
			Host: "localhost",
			Port: 8080 + i,
		}
		err := registry.RegisterEndpoint(rentalID, "api", endpoint)
		if err != nil {
			t.Fatalf("Failed to register endpoint for %s: %v", rentalID, err)
		}
	}

	allEndpoints := registry.GetAllEndpoints()

	if len(allEndpoints) != 2 {
		t.Errorf("Expected 2 endpoints, got %d", len(allEndpoints))
	}
}

func TestEndpointRegistry_GetEndpointsByType(t *testing.T) {
	registry := NewEndpointRegistry()

	// Register endpoints of different types
	endpointTypes := []string{"api", "websocket", "api"}
	for i, endpointType := range endpointTypes {
		endpoint := &objects.TEEEndpoint{
			Host: "localhost",
			Port: 8080 + i,
		}
		err := registry.RegisterEndpoint("rental-123", endpointType, endpoint)
		if err != nil {
			t.Fatalf("Failed to register endpoint %d: %v", i, err)
		}
	}

	apiEndpoints := registry.GetEndpointsByType("api")

	if len(apiEndpoints) != 2 {
		t.Errorf("Expected 2 API endpoints, got %d", len(apiEndpoints))
	}

	for _, endpoint := range apiEndpoints {
		if endpoint.EndpointType != "api" {
			t.Errorf("Expected endpoint type 'api', got %s", endpoint.EndpointType)
		}
	}
}

func TestEndpointRegistry_HealthCheck(t *testing.T) {
	registry := NewEndpointRegistry()

	rentalID := "rental-123"
	endpointType := "api"
	endpoint := &objects.TEEEndpoint{
		Host: "localhost",
		Port: 8080,
	}

	err := registry.RegisterEndpoint(rentalID, endpointType, endpoint)
	if err != nil {
		t.Fatalf("Failed to register endpoint: %v", err)
	}

	healthy, err := registry.HealthCheck(endpoint.ID)
	if err != nil {
		t.Fatalf("Failed to perform health check: %v", err)
	}

	if !healthy {
		t.Error("Expected endpoint to be healthy")
	}
}

func TestEndpointRegistry_HealthCheck_NotFound(t *testing.T) {
	registry := NewEndpointRegistry()

	_, err := registry.HealthCheck("non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent endpoint")
	}
}

func TestEndpointRegistry_HealthCheck_Expired(t *testing.T) {
	registry := NewEndpointRegistry()

	rentalID := "rental-123"
	endpointType := "api"
	endpoint := &objects.TEEEndpoint{
		Host: "localhost",
		Port: 8080,
	}

	err := registry.RegisterEndpoint(rentalID, endpointType, endpoint)
	if err != nil {
		t.Fatalf("Failed to register endpoint: %v", err)
	}

	// Manually expire the endpoint
	endpoint.ExpiresAt = time.Now().Add(-1 * time.Hour)

	_, err = registry.HealthCheck(endpoint.ID)
	if err == nil {
		t.Error("Expected error for expired endpoint")
	}
}