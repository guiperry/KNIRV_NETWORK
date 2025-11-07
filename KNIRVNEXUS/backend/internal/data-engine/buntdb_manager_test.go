package dataengine

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupExpiredAuth(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	// Create an expired auth entry
	expiredAuth := &AuthEntry{
		ID:        "expired-auth",
		UserID:    "user1",
		TokenHash: "expired-hash",
		TokenType: "jwt",
		ExpiresAt: time.Now().Add(-time.Hour), // Already expired
		CreatedAt: time.Now().Add(-time.Hour * 2),
		Revoked:   false,
	}

	err = manager.CreateAuth(expiredAuth)
	require.NoError(t, err)

	// Create a valid auth entry
	validAuth := &AuthEntry{
		ID:        "valid-auth",
		UserID:    "user2",
		TokenHash: "valid-hash",
		TokenType: "jwt",
		ExpiresAt: time.Now().Add(time.Hour), // Still valid
		CreatedAt: time.Now(),
		Revoked:   false,
	}

	err = manager.CreateAuth(validAuth)
	require.NoError(t, err)

	// Run cleanup
	err = manager.CleanupExpiredAuth()
	assert.NoError(t, err)

	// Verify expired auth is gone
	_, err = manager.GetAuth("expired-auth")
	assert.Error(t, err) // Should not find expired auth

	// Verify valid auth still exists
	retrievedAuth, err := manager.GetAuth("valid-auth")
	assert.NoError(t, err)
	assert.Equal(t, "valid-auth", retrievedAuth.ID)
}

func TestCreateSession(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	session := &SessionEntry{
		ID:           "test-session",
		UserID:       "user1",
		Status:       "active",
		ExpiresAt:    time.Now().Add(time.Hour),
		IPAddress:    "127.0.0.1",
		UserModel:    "default",
		Data:         map[string]interface{}{"key": "value"},
	}

	err = manager.CreateSession(session)
	assert.NoError(t, err)

	// Verify session was created
	retrievedSession, err := manager.GetSession("test-session")
	assert.NoError(t, err)
	assert.Equal(t, "test-session", retrievedSession.ID)
	assert.Equal(t, "user1", retrievedSession.UserID)
	assert.Equal(t, "active", retrievedSession.Status)
	assert.NotZero(t, retrievedSession.CreatedAt)
	assert.NotZero(t, retrievedSession.LastActivity)
}

func TestGetSession(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	session := &SessionEntry{
		ID:           "test-session",
		UserID:       "user1",
		Status:       "active",
		ExpiresAt:    time.Now().Add(time.Hour),
		IPAddress:    "127.0.0.1",
		UserModel:    "default",
		Data:         map[string]interface{}{"key": "value"},
	}

	err = manager.CreateSession(session)
	require.NoError(t, err)

	// Test successful retrieval
	retrievedSession, err := manager.GetSession("test-session")
	assert.NoError(t, err)
	assert.Equal(t, "test-session", retrievedSession.ID)

	// Test retrieval of non-existent session
	_, err = manager.GetSession("non-existent")
	assert.Error(t, err)
}

func TestGetMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	// Create test metrics
	metric1 := &MetricEntry{
		ID:        "metric1",
		Timestamp: time.Now(),
		Source:    "test-source",
		Name:      "cpu_usage",
		Type:      "gauge",
		Value:     85.5,
		Unit:      "percent",
		Tags:      map[string]string{"host": "server1"},
	}

	metric2 := &MetricEntry{
		ID:        "metric2",
		Timestamp: time.Now().Add(-time.Hour),
		Source:    "test-source",
		Name:      "memory_usage",
		Type:      "gauge",
		Value:     70.2,
		Unit:      "percent",
		Tags:      map[string]string{"host": "server1"},
	}

	err = manager.StoreMetric(metric1)
	require.NoError(t, err)
	err = manager.StoreMetric(metric2)
	require.NoError(t, err)

	// Test GetMetrics with source filter
	metrics, err := manager.GetMetrics("test-source", "", time.Time{}, 10)
	assert.NoError(t, err)
	assert.Len(t, metrics, 2)

	// Test GetMetrics with limit
	metrics, err = manager.GetMetrics("", "", time.Time{}, 1)
	assert.NoError(t, err)
	assert.Len(t, metrics, 1)
}

func TestGetAlerts(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	// Create test alerts
	alert1 := &AlertEntry{
		ID:          "alert1",
		Timestamp:   time.Now(),
		Type:        "system",
		Title:       "High CPU",
		Description: "CPU usage is high",
		Severity:    "warning",
		Source:      "monitor",
		Status:      "active",
		Resolved:    false,
	}

	alert2 := &AlertEntry{
		ID:          "alert2",
		Timestamp:   time.Now().Add(-time.Hour),
		Type:        "system",
		Title:       "Low Memory",
		Description: "Memory usage is low",
		Severity:    "info",
		Source:      "monitor",
		Status:      "resolved",
		Resolved:    true,
	}

	err = manager.StoreAlert(alert1)
	require.NoError(t, err)
	err = manager.StoreAlert(alert2)
	require.NoError(t, err)

	// Test GetAlerts with severity filter
	alerts, err := manager.GetAlerts("warning", "", time.Time{}, 10)
	assert.NoError(t, err)
	assert.Len(t, alerts, 1)
	assert.Equal(t, "alert1", alerts[0].ID)

	// Test GetAlerts with status filter
	alerts, err = manager.GetAlerts("", "resolved", time.Time{}, 10)
	assert.NoError(t, err)
	assert.Len(t, alerts, 1)
	assert.Equal(t, "alert2", alerts[0].ID)
}

func TestGetReports(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	// Create test reports
	report1 := &ReportEntry{
		ID:          "report1",
		Timestamp:   time.Now(),
		Type:        "performance",
		Title:       "Monthly Report",
		Description: "Performance metrics for the month",
		Data:        map[string]interface{}{"metric": "value"},
		GeneratedBy: "system",
		Format:      "json",
		Status:      "completed",
	}

	report2 := &ReportEntry{
		ID:          "report2",
		Timestamp:   time.Now().Add(-time.Hour),
		Type:        "security",
		Title:       "Security Audit",
		Description: "Security audit results",
		Data:        map[string]interface{}{"audit": "passed"},
		GeneratedBy: "system",
		Format:      "pdf",
		Status:      "completed",
	}

	err = manager.StoreReport(report1, false)
	require.NoError(t, err)
	err = manager.StoreReport(report2, false)
	require.NoError(t, err)

	// Test GetReports with type filter
	reports, err := manager.GetReports("performance", false, time.Time{}, 10)
	assert.NoError(t, err)
	assert.Len(t, reports, 1)
	assert.Equal(t, "report1", reports[0].ID)

	// Test GetReports with limit
	reports, err = manager.GetReports("", false, time.Time{}, 1)
	assert.NoError(t, err)
	assert.Len(t, reports, 1)
}

func TestGetEvents(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	// Create test events
	event1 := &EventEntry{
		ID:        "event1",
		Timestamp: time.Now(),
		Type:      "user_action",
		Source:    "web_app",
		Data:      map[string]interface{}{"action": "login"},
		Tags:      map[string]string{"user": "user1"},
	}

	event2 := &EventEntry{
		ID:        "event2",
		Timestamp: time.Now().Add(-time.Hour),
		Type:      "system_event",
		Source:    "web_app",
		Data:      map[string]interface{}{"event": "startup"},
		Tags:      map[string]string{"component": "server"},
	}
	err = manager.StoreEvent(event1)
	require.NoError(t, err)
	err = manager.StoreEvent(event2)
	require.NoError(t, err)

	// Test GetEvents with type filter
	events, err := manager.GetEvents("user_action", "", time.Time{}, 10)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "event1", events[0].ID)

	// Test GetEvents with source filter
	events, err = manager.GetEvents("", "web_app", time.Time{}, 10)
	assert.NoError(t, err)
	assert.Len(t, events, 2)

	// Test GetEvents with limit
	events, err = manager.GetEvents("", "", time.Time{}, 1)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestResolveAlert(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	// Create test alert
	alert := &AlertEntry{
		ID:          "alert1",
		Timestamp:   time.Now(),
		Type:        "system",
		Title:       "Test Alert",
		Description: "Test alert description",
		Severity:    "warning",
		Source:      "test",
		Status:      "active",
		Resolved:    false,
	}

	err = manager.StoreAlert(alert)
	require.NoError(t, err)

	// Resolve the alert
	err = manager.ResolveAlert("alert1")
	assert.NoError(t, err)

	// Verify alert is resolved
	alerts, err := manager.GetAlerts("", "", time.Time{}, 10)
	assert.NoError(t, err)
	assert.Len(t, alerts, 1)
	assert.True(t, alerts[0].Resolved)
	assert.NotNil(t, alerts[0].ResolvedAt)
}

func TestStoreUserReport(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	report := &UserReportEntry{
		ID:        "user-report1",
		UserID:    "user1",
		Type:      "performance",
		Data:      map[string]interface{}{"metric": "value"},
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{"source": "test"},
	}

	err = manager.StoreUserReport(report)
	assert.NoError(t, err)

	// Verify report was stored
	reports, err := manager.GetUserReports("user1", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	assert.NoError(t, err)
	assert.Len(t, reports, 1)
	assert.Equal(t, "user-report1", reports[0].ID)
}

func TestStoreSystemReport(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	report := &SystemReportEntry{
		ID:        "system-report1",
		Type:      "health",
		Data:      map[string]interface{}{"status": "ok"},
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{"source": "test"},
	}

	err = manager.StoreSystemReport(report)
	assert.NoError(t, err)

	// Verify report was stored
	reports, err := manager.GetSystemReports(time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	assert.NoError(t, err)
	assert.Len(t, reports, 1)
	assert.Equal(t, "system-report1", reports[0].ID)
}

func TestGetUserReports(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	// Create test reports
	report1 := &UserReportEntry{
		ID:        "report1",
		UserID:    "user1",
		Type:      "performance",
		Data:      map[string]interface{}{"metric": "value1"},
		Timestamp: time.Now(),
	}

	report2 := &UserReportEntry{
		ID:        "report2",
		UserID:    "user2",
		Type:      "security",
		Data:      map[string]interface{}{"metric": "value2"},
		Timestamp: time.Now().Add(-time.Hour),
	}

	err = manager.StoreUserReport(report1)
	require.NoError(t, err)
	err = manager.StoreUserReport(report2)
	require.NoError(t, err)

	// Test filtering by user
	reports, err := manager.GetUserReports("user1", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	assert.NoError(t, err)
	assert.Len(t, reports, 1)
	assert.Equal(t, "report1", reports[0].ID)

	// Test filtering by time range
	reports, err = manager.GetUserReports("user2", time.Now().Add(-2*time.Hour), time.Now().Add(-30*time.Minute))
	assert.NoError(t, err)
	assert.Len(t, reports, 1)
	assert.Equal(t, "report2", reports[0].ID)
}

func TestGetSystemReports(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	// Create test reports
	report1 := &SystemReportEntry{
		ID:        "report1",
		Type:      "health",
		Data:      map[string]interface{}{"status": "ok"},
		Timestamp: time.Now(),
	}

	report2 := &SystemReportEntry{
		ID:        "report2",
		Type:      "performance",
		Data:      map[string]interface{}{"metric": "value"},
		Timestamp: time.Now().Add(-time.Hour),
	}

	err = manager.StoreSystemReport(report1)
	require.NoError(t, err)
	err = manager.StoreSystemReport(report2)
	require.NoError(t, err)

	// Test time range filtering
	reports, err := manager.GetSystemReports(time.Now().Add(-2*time.Hour), time.Now().Add(-30*time.Minute))
	assert.NoError(t, err)
	assert.Len(t, reports, 1)
	assert.Equal(t, "report2", reports[0].ID)
}

func TestCleanupOldMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	// Create old and new metrics
	oldMetric := &MetricEntry{
		ID:        "old-metric",
		Timestamp: time.Now().Add(-48 * time.Hour), // 2 days ago
		Source:    "test",
		Name:      "old_metric",
		Value:     100,
	}

	newMetric := &MetricEntry{
		ID:        "new-metric",
		Timestamp: time.Now(), // Now
		Source:    "test",
		Name:      "new_metric",
		Value:     200,
	}

	err = manager.StoreMetric(oldMetric)
	require.NoError(t, err)
	err = manager.StoreMetric(newMetric)
	require.NoError(t, err)

	// Cleanup metrics older than 24 hours
	err = manager.CleanupOldMetrics(24 * time.Hour)
	assert.NoError(t, err)

	// Verify old metric is gone, new metric remains
	metrics, err := manager.GetMetrics("", "", time.Time{}, 10)
	assert.NoError(t, err)
	assert.Len(t, metrics, 1)
	assert.Equal(t, "new-metric", metrics[0].ID)
}

func TestGetStats(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	// Create some test data
	metric := &MetricEntry{
		ID:        "metric1",
		Timestamp: time.Now(),
		Source:    "test",
		Name:      "test_metric",
		Value:     42,
	}

	alert := &AlertEntry{
		ID:          "alert1",
		Timestamp:   time.Now(),
		Type:        "test",
		Title:       "Test Alert",
		Severity:    "info",
		Source:      "test",
		Status:      "active",
		Resolved:    false,
	}

	user := &UserEntry{
		ID:       "user1",
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
		Status:   "active",
	}

	err = manager.StoreMetric(metric)
	require.NoError(t, err)
	err = manager.StoreAlert(alert)
	require.NoError(t, err)
	err = manager.CreateUser(user)
	require.NoError(t, err)

	// Get stats
	stats, err := manager.GetStats()
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Contains(t, stats["database_path"], "test.db")
	assert.Equal(t, 1, stats["collections"].(map[string]int)["metrics"])
	assert.Equal(t, 1, stats["collections"].(map[string]int)["alerts"])
	assert.Equal(t, 0, stats["collections"].(map[string]int)["users"])
}

func TestGetDatabaseStats(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	// Create test data
	metric := &MetricEntry{
		ID:        "metric1",
		Timestamp: time.Now(),
		Source:    "test",
		Name:      "test_metric",
		Value:     42,
	}

	alert := &AlertEntry{
		ID:          "alert1",
		Timestamp:   time.Now(),
		Type:        "test",
		Title:       "Test Alert",
		Severity:    "info",
		Source:      "test",
		Status:      "active",
		Resolved:    false,
	}

	err = manager.StoreMetric(metric)
	require.NoError(t, err)
	err = manager.StoreAlert(alert)
	require.NoError(t, err)

	// Get database stats
	stats, err := manager.GetDatabaseStats()
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Contains(t, stats["db_path"], "test.db")
	assert.Equal(t, 1, stats["metrics_count"])
	assert.Equal(t, 1, stats["alerts_count"])
}

func TestCreateUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	user := &UserEntry{
		ID:       "user1",
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
		Status:   "active",
	}

	err = manager.CreateUser(user)
	assert.NoError(t, err)

	// Verify user was created
	retrievedUser, err := manager.GetUser("user1")
	assert.NoError(t, err)
	assert.Equal(t, "user1", retrievedUser.ID)
	assert.Equal(t, "testuser", retrievedUser.Username)
	assert.NotZero(t, retrievedUser.CreatedAt)
	assert.NotZero(t, retrievedUser.UpdatedAt)
}

func TestGetUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	user := &UserEntry{
		ID:       "user1",
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
		Status:   "active",
	}

	err = manager.CreateUser(user)
	require.NoError(t, err)

	// Test successful retrieval
	retrievedUser, err := manager.GetUser("user1")
	assert.NoError(t, err)
	assert.Equal(t, "user1", retrievedUser.ID)

	// Test retrieval of non-existent user
	_, err = manager.GetUser("non-existent")
	assert.Error(t, err)
}

func TestGetUserByUsername(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	user := &UserEntry{
		ID:       "user1",
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
		Status:   "active",
	}

	err = manager.CreateUser(user)
	require.NoError(t, err)

	// Test successful retrieval
	retrievedUser, err := manager.GetUserByUsername("testuser")
	assert.NoError(t, err)
	assert.Equal(t, "user1", retrievedUser.ID)

	// Test retrieval of non-existent username
	_, err = manager.GetUserByUsername("non-existent")
	assert.Error(t, err)
}

func TestUpdateUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	user := &UserEntry{
		ID:       "user1",
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
		Status:   "active",
	}

	err = manager.CreateUser(user)
	require.NoError(t, err)

	// Update user
	user.Email = "updated@example.com"
	user.Status = "inactive"
	err = manager.UpdateUser(user)
	assert.NoError(t, err)

	// Verify update
	retrievedUser, err := manager.GetUser("user1")
	assert.NoError(t, err)
	assert.Equal(t, "updated@example.com", retrievedUser.Email)
	assert.Equal(t, "inactive", retrievedUser.Status)
	assert.True(t, retrievedUser.UpdatedAt.After(retrievedUser.CreatedAt))
}

func TestDeleteUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	user := &UserEntry{
		ID:       "user1",
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
		Status:   "active",
	}

	err = manager.CreateUser(user)
	require.NoError(t, err)

	// Delete user
	err = manager.DeleteUser("user1")
	assert.NoError(t, err)

	// Verify user is gone
	_, err = manager.GetUser("user1")
	assert.Error(t, err)
}

func TestListUsers(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	// Create multiple users
	for i := 1; i <= 5; i++ {
		user := &UserEntry{
			ID:       fmt.Sprintf("user%d", i),
			Username: fmt.Sprintf("testuser%d", i),
			Email:    fmt.Sprintf("test%d@example.com", i),
			Role:     "user",
			Status:   "active",
		}
		err = manager.CreateUser(user)
		require.NoError(t, err)
	}

	// Test listing all users
	users, err := manager.ListUsers(0, 10)
	assert.NoError(t, err)
	assert.Len(t, users, 5)

	// Test pagination
	users, err = manager.ListUsers(0, 2)
	assert.NoError(t, err)
	assert.Len(t, users, 2)

	users, err = manager.ListUsers(2, 2)
	assert.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestGetAuthByToken(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	// Create valid auth entry
	validAuth := &AuthEntry{
		ID:        "auth1",
		UserID:    "user1",
		TokenHash: "valid-token-hash",
		TokenType: "jwt",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
		Revoked:   false,
	}

	// Create expired auth entry
	expiredAuth := &AuthEntry{
		ID:        "auth2",
		UserID:    "user2",
		TokenHash: "expired-token-hash",
		TokenType: "jwt",
		ExpiresAt: time.Now().Add(-time.Hour),
		CreatedAt: time.Now().Add(-2 * time.Hour),
		Revoked:   false,
	}

	// Create revoked auth entry
	revokedAuth := &AuthEntry{
		ID:        "auth3",
		UserID:    "user3",
		TokenHash: "revoked-token-hash",
		TokenType: "jwt",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
		Revoked:   true,
	}

	err = manager.CreateAuth(validAuth)
	require.NoError(t, err)
	err = manager.CreateAuth(expiredAuth)
	require.NoError(t, err)
	err = manager.CreateAuth(revokedAuth)
	require.NoError(t, err)

	// Test valid token retrieval
	auth, err := manager.GetAuthByToken("valid-token-hash")
	assert.NoError(t, err)
	assert.Equal(t, "auth1", auth.ID)

	// Test expired token retrieval
	_, err = manager.GetAuthByToken("expired-token-hash")
	assert.Error(t, err)

	// Test revoked token retrieval
	_, err = manager.GetAuthByToken("revoked-token-hash")
	assert.Error(t, err)

	// Test non-existent token
	_, err = manager.GetAuthByToken("non-existent")
	assert.Error(t, err)
}

func TestRevokeAuth(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	manager, err := NewBuntDBManager(dbPath)
	require.NoError(t, err)
	defer manager.Close()

	auth := &AuthEntry{
		ID:        "auth1",
		UserID:    "user1",
		TokenHash: "token-hash",
		TokenType: "jwt",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
		Revoked:   false,
	}

	err = manager.CreateAuth(auth)
	require.NoError(t, err)

	// Revoke auth
	err = manager.RevokeAuth("auth1")
	assert.NoError(t, err)

	// Verify auth is revoked
	retrievedAuth, err := manager.GetAuth("auth1")
	assert.NoError(t, err)
	assert.True(t, retrievedAuth.Revoked)
}
