package inference_engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockDatabaseAccessor implements DatabaseAccessor for testing
type MockDatabaseAccessor struct{}

func (m *MockDatabaseAccessor) GetValue(key string) (string, error) {
	return "mock value", nil
}

func (m *MockDatabaseAccessor) SetValue(key, value string) error {
	return nil
}

func TestNewInferenceService(t *testing.T) {
	mockDB := &MockDatabaseAccessor{}

	service, err := NewInferenceService(mockDB)

	assert.NoError(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, mockDB, service.db)
}
