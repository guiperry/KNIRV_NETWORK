package dataengine

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuntDBManager(t *testing.T) {
	// Create temporary database file
	tmpFile, err := os.CreateTemp("", "test_*.db")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewBuntDBManager(tmpFile.Name())
	require.NoError(t, err)
	require.NotNil(t, manager)

	err = manager.Close()
	assert.NoError(t, err)
}

func TestBuntDBManagerMetrics(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.db")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewBuntDBManager(tmpFile.Name())
	require.NoError(t, err)
	defer manager.Close()

	// Test storing metrics
	metric := &MetricEntry{
		ID:        "test-metric-1",
		Source:    "test-source",
		Type:      "cpu_usage",
		Value:     75.5,
		Unit:      "percent",
		Timestamp: time.Now(),
		Tags:      map[string]string{"host": "test-host"},
		Metadata:  map[string]interface{}{},
	}

	err = manager.StoreMetric(metric)
	assert.NoError(t, err)

	// Test retrieving metrics
	metrics, err := manager.GetMetrics("test-source", "", time.Now().Add(-1*time.Hour), 100)
	assert.NoError(t, err)
	assert.Len(t, metrics, 1)
	assert.Equal(t, metric.ID, metrics[0].ID)
	assert.Equal(t, metric.Value, metrics[0].Value)
}

func TestBuntDBManagerAlerts(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.db")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewBuntDBManager(tmpFile.Name())
	require.NoError(t, err)
	defer manager.Close()

	// Test storing alerts
	alert := &AlertEntry{
		ID:        "test-alert-1",
		Source:    "test-source",
		Type:      "warning",
		Severity:  "medium",
		Message:   "Test alert message",
		Timestamp: time.Now(),
		Resolved:  false,
		Metadata:  map[string]interface{}{"test": "value"},
	}

	err = manager.StoreAlert(alert)
	assert.NoError(t, err)

	// Test retrieving alerts
	alerts, err := manager.GetAlerts("", "", time.Now().Add(-1*time.Hour), 100)
	assert.NoError(t, err)
	assert.Len(t, alerts, 1)
	assert.Equal(t, alert.ID, alerts[0].ID)
	assert.Equal(t, alert.Message, alerts[0].Message)
	assert.False(t, alerts[0].Resolved)

	// Test resolving alert
	err = manager.ResolveAlert(alert.ID)
	assert.NoError(t, err)

	// Verify alert is resolved
	alerts, err = manager.GetAlerts("", "", time.Now().Add(-1*time.Hour), 100)
	assert.NoError(t, err)
	assert.Len(t, alerts, 1)
	assert.True(t, alerts[0].Resolved)
}

func TestBuntDBManagerUsers(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.db")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewBuntDBManager(tmpFile.Name())
	require.NoError(t, err)
	defer manager.Close()

	// Test creating user
	user := &UserEntry{
		ID:           "test-user-1",
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashed-password",
		FirstName:    "Test",
		LastName:     "User",
		Role:         "user",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Metadata:     map[string]interface{}{"test": "value"},
	}

	err = manager.CreateUser(user)
	assert.NoError(t, err)

	// Test retrieving user by ID
	retrievedUser, err := manager.GetUser(user.ID)
	assert.NoError(t, err)
	assert.Equal(t, user.Username, retrievedUser.Username)
	assert.Equal(t, user.Email, retrievedUser.Email)

	// Test retrieving user by username
	retrievedUser, err = manager.GetUserByUsername(user.Username)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, retrievedUser.ID)

	// Test updating user
	user.FirstName = "Updated"
	err = manager.UpdateUser(user)
	assert.NoError(t, err)

	retrievedUser, err = manager.GetUser(user.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated", retrievedUser.FirstName)

	// Test listing users
	users, err := manager.ListUsers(0, 10)
	assert.NoError(t, err)
	assert.Len(t, users, 1)

	// Test deleting user
	err = manager.DeleteUser(user.ID)
	assert.NoError(t, err)

	_, err = manager.GetUser(user.ID)
	assert.Error(t, err)
}

func TestBuntDBManagerAuth(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.db")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewBuntDBManager(tmpFile.Name())
	require.NoError(t, err)
	defer manager.Close()

	// Test creating auth entry
	auth := &AuthEntry{
		ID:        "test-auth-1",
		UserID:    "test-user-1",
		TokenHash: "hashed-token",
		TokenType: "jwt",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
		Revoked:   false,
		Metadata:  map[string]interface{}{"test": "value"},
	}

	err = manager.CreateAuth(auth)
	assert.NoError(t, err)

	// Test retrieving auth by ID
	retrievedAuth, err := manager.GetAuth(auth.ID)
	assert.NoError(t, err)
	assert.Equal(t, auth.TokenHash, retrievedAuth.TokenHash)

	// Test retrieving auth by token
	retrievedAuth, err = manager.GetAuthByToken(auth.TokenHash)
	assert.NoError(t, err)
	assert.Equal(t, auth.ID, retrievedAuth.ID)

	// Test revoking auth
	err = manager.RevokeAuth(auth.ID)
	assert.NoError(t, err)

	// Should not be able to get revoked token
	_, err = manager.GetAuthByToken(auth.TokenHash)
	assert.Error(t, err)
}

func TestBuntDBManagerReports(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.db")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewBuntDBManager(tmpFile.Name())
	require.NoError(t, err)
	defer manager.Close()

	// Test storing user report
	userReport := &UserReportEntry{
		ID:        "test-user-report-1",
		UserID:    "test-user-1",
		Type:      "activity",
		Data:      map[string]interface{}{"actions": 10},
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{"test": "value"},
	}

	err = manager.StoreUserReport(userReport)
	assert.NoError(t, err)

	// Test storing system report
	systemReport := &SystemReportEntry{
		ID:        "test-system-report-1",
		Type:      "performance",
		Data:      map[string]interface{}{"cpu_avg": 45.5},
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{"test": "value"},
	}

	err = manager.StoreSystemReport(systemReport)
	assert.NoError(t, err)

	// Test retrieving reports
	userReports, err := manager.GetUserReports("test-user-1", time.Now().Add(-1*time.Hour), time.Now().Add(1*time.Hour))
	assert.NoError(t, err)
	assert.Len(t, userReports, 1)

	systemReports, err := manager.GetSystemReports(time.Now().Add(-1*time.Hour), time.Now().Add(1*time.Hour))
	assert.NoError(t, err)
	assert.Len(t, systemReports, 1)
}

func TestBuntDBManagerEvents(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.db")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewBuntDBManager(tmpFile.Name())
	require.NoError(t, err)
	defer manager.Close()

	// Test storing event
	event := &EventEntry{
		ID:        "test-event-1",
		Timestamp: time.Now(),
		Type:      "user_action",
		Source:    "web-ui",
		Data:      map[string]interface{}{"action": "login"},
		Tags:      map[string]string{"user": "test-user"},
	}

	err = manager.StoreEvent(event)
	assert.NoError(t, err)

	// Test retrieving events
	events, err := manager.GetEvents("user_action", "web-ui", time.Now().Add(-1*time.Hour), 100)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, event.ID, events[0].ID)
}

func TestBuntDBManagerConcurrency(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.db")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewBuntDBManager(tmpFile.Name())
	require.NoError(t, err)
	defer manager.Close()

	// Test concurrent metric storage
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			metric := &MetricEntry{
				ID:        fmt.Sprintf("test-metric-%d", id),
				Source:    "test-source",
				Name:      "test_metric",
				Value:     float64(id),
				Unit:      "count",
				Timestamp: time.Now(),
				Tags:      map[string]string{"id": fmt.Sprintf("%d", id)},
			}

			err := manager.StoreMetric(metric)
			assert.NoError(t, err)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all metrics were stored
	metrics, err := manager.GetMetrics("test-source", "", time.Now().Add(-1*time.Hour), 100)
	assert.NoError(t, err)
	assert.Len(t, metrics, 10)
}

func TestBuntDBManagerCleanup(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.db")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewBuntDBManager(tmpFile.Name())
	require.NoError(t, err)
	defer manager.Close()

	// Store old metric
	oldMetric := &MetricEntry{
		ID:        "old-metric",
		Source:    "test-source",
		Name:      "old_metric",
		Value:     100.0,
		Unit:      "count",
		Timestamp: time.Now().Add(-48 * time.Hour), // 2 days old
		Tags:      map[string]string{},
	}

	err = manager.StoreMetric(oldMetric)
	assert.NoError(t, err)

	// Store recent metric
	recentMetric := &MetricEntry{
		ID:        "recent-metric",
		Source:    "test-source",
		Name:      "recent_metric",
		Value:     200.0,
		Unit:      "count",
		Timestamp: time.Now(),
		Tags:      map[string]string{},
	}

	err = manager.StoreMetric(recentMetric)
	assert.NoError(t, err)

	// Test cleanup (remove metrics older than 24 hours)
	err = manager.CleanupOldMetrics(24 * time.Hour)
	assert.NoError(t, err)

	// Verify only recent metric remains
	metrics, err := manager.GetMetrics("test-source", "", time.Now().Add(-72*time.Hour), 100)
	assert.NoError(t, err)
	assert.Len(t, metrics, 1)
	assert.Equal(t, recentMetric.ID, metrics[0].ID)
}

func TestBuntDBManagerStats(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.db")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	manager, err := NewBuntDBManager(tmpFile.Name())
	require.NoError(t, err)
	defer manager.Close()

	// Add some test data
	metric := &MetricEntry{
		ID:        "test-metric",
		Source:    "test-source",
		Name:      "test_metric",
		Value:     100.0,
		Unit:      "count",
		Timestamp: time.Now(),
		Tags:      map[string]string{},
	}
	manager.StoreMetric(metric)

	user := &UserEntry{
		ID:        "test-user",
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  map[string]interface{}{},
	}
	manager.CreateUser(user)

	// Test getting stats
	stats, err := manager.GetStats()
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "total_keys")
	assert.Contains(t, stats, "collections")
}
