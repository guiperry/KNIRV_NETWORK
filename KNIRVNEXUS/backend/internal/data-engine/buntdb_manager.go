package dataengine

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/buntdb"
)

// BuntDBManager manages the embedded BuntDB database
type BuntDBManager struct {
	db      *buntdb.DB
	dbPath  string
	indexes map[string]string
	mu      sync.RWMutex
}

// BuntDB collections as defined in the NEXUS_MERGE.md
const (
	// Core Data Collections
	MetricsCollection = "metrics:"
	AlertsCollection  = "alerts:"
	ReportsCollection = "reports:"
	ConfigCollection  = "config:"

	// User Management Collections
	UsersCollection    = "users:"
	AuthCollection     = "auth:"
	SessionsCollection = "sessions:"

	// Model Management Collections
	ModelsCollection        = "objects:"
	ModelBinariesCollection = "model_binaries:"
	ModelRuntimeCollection  = "model_runtime:"

	// DVE Node Collections
	DVENodesCollection          = "dve_nodes:"
	ValidationTasksCollection   = "validation_tasks:"
	ValidationResultsCollection = "validation_results:"

	// CDE (Cloud Development Environment) Collections
	CDEEnvironmentsCollection = "cde_environments:"
	CDESessionsCollection     = "cde_sessions:"
	CDEProjectsCollection     = "cde_projects:"

	// Report Collections (User vs System)
	UserReportsCollection   = "user_reports:"
	SystemReportsCollection = "system_reports:"

	// Cross-DVE Communication Collections
	DVEMessagesCollection = "dve_messages:"
	DVERoutingCollection  = "dve_routing:"
	DVERegistryCollection = "dve_registry:"
)

// MetricEntry represents a metric entry in BuntDB
type MetricEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Value     float64                `json:"value"`
	Unit      string                 `json:"unit"`
	Tags      map[string]string      `json:"tags"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// AlertEntry represents an alert entry in BuntDB
type AlertEntry struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Message     string                 `json:"message"`
	Severity    string                 `json:"severity"`
	Source      string                 `json:"source"`
	Status      string                 `json:"status"`
	Resolved    bool                   `json:"resolved"`
	Tags        map[string]string      `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// ReportEntry represents a report entry in BuntDB
type ReportEntry struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	GeneratedBy string                 `json:"generated_by"`
	Format      string                 `json:"format"`
	Status      string                 `json:"status"`
}

// EventEntry represents an event entry in BuntDB
type EventEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
	Tags      map[string]string      `json:"tags"`
}

// UserEntry represents a user in BuntDB
type UserEntry struct {
	ID           string                 `json:"id"`
	Username     string                 `json:"username"`
	Email        string                 `json:"email"`
	PasswordHash string                 `json:"password_hash"`
	FirstName    string                 `json:"first_name"`
	LastName     string                 `json:"last_name"`
	Role         string                 `json:"role"`
	Status       string                 `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	LastLogin    *time.Time             `json:"last_login,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// AuthEntry represents authentication data in BuntDB
type AuthEntry struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	TokenHash string                 `json:"token_hash"`
	TokenType string                 `json:"token_type"` // jwt, api_key, refresh
	ExpiresAt time.Time              `json:"expires_at"`
	CreatedAt time.Time              `json:"created_at"`
	LastUsed  *time.Time             `json:"last_used,omitempty"`
	IPAddress string                 `json:"ip_address"`
	UserModel string                 `json:"user_model"`
	Revoked   bool                   `json:"revoked"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// SessionEntry represents a user session in BuntDB
type SessionEntry struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	Status       string                 `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
	LastActivity time.Time              `json:"last_activity"`
	IPAddress    string                 `json:"ip_address"`
	UserModel    string                 `json:"user_model"`
	Data         map[string]interface{} `json:"data"`
}

// ModelEntry represents an model in BuntDB
type ModelEntry struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Status    string                 `json:"status"`
	Version   string                 `json:"version"`
	Binary    string                 `json:"binary"`
	Config    map[string]interface{} `json:"config"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	LastSeen  *time.Time             `json:"last_seen,omitempty"`
	OwnerID   string                 `json:"owner_id"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// DVENodeEntry represents a DVE node in BuntDB
type DVENodeEntry struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Status       string                 `json:"status"`
	IPAddress    string                 `json:"ip_address"`
	Port         int                    `json:"port"`
	Capabilities []string               `json:"capabilities"`
	Resources    map[string]interface{} `json:"resources"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	LastSeen     *time.Time             `json:"last_seen,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ValidationTaskEntry represents a validation task in BuntDB
type ValidationTaskEntry struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	Priority    int                    `json:"priority"`
	Data        map[string]interface{} `json:"data"`
	Result      map[string]interface{} `json:"result,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	AssignedTo  string                 `json:"assigned_to,omitempty"`
	CreatedBy   string                 `json:"created_by"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ValidationResultEntry represents a validation result in BuntDB
type ValidationResultEntry struct {
	ID          string                 `json:"id"`
	TaskID      string                 `json:"task_id"`
	Status      string                 `json:"status"`
	Score       float64                `json:"score"`
	Details     map[string]interface{} `json:"details"`
	Errors      []string               `json:"errors,omitempty"`
	Warnings    []string               `json:"warnings,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	ValidatedBy string                 `json:"validated_by"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewBuntDBManager creates a new BuntDB manager
func NewBuntDBManager(dbPath string) (*BuntDBManager, error) {
	db, err := buntdb.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open BuntDB: %w", err)
	}

	manager := &BuntDBManager{
		db:      db,
		dbPath:  dbPath,
		indexes: make(map[string]string),
	}

	// Create indexes for efficient querying
	if err := manager.createIndexes(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return manager, nil
}

// Close closes the BuntDB connection
func (m *BuntDBManager) Close() error {
	return m.db.Close()
}

// createIndexes creates indexes for efficient querying
func (m *BuntDBManager) createIndexes() error {
	return m.db.Update(func(tx *buntdb.Tx) error {
		// Metrics indexes
		tx.CreateIndex("metrics_time", MetricsCollection+"*", buntdb.IndexJSON("timestamp"))
		tx.CreateIndex("metrics_source", MetricsCollection+"*", buntdb.IndexJSON("source"))
		tx.CreateIndex("metrics_type", MetricsCollection+"*", buntdb.IndexJSON("type"))

		// Alerts indexes
		tx.CreateIndex("alerts_time", AlertsCollection+"*", buntdb.IndexJSON("timestamp"))
		tx.CreateIndex("alerts_severity", AlertsCollection+"*", buntdb.IndexJSON("severity"))
		tx.CreateIndex("alerts_status", AlertsCollection+"*", buntdb.IndexJSON("status"))
		tx.CreateIndex("alerts_source", AlertsCollection+"*", buntdb.IndexJSON("source"))

		// Reports indexes
		tx.CreateIndex("reports_time", ReportsCollection+"*", buntdb.IndexJSON("timestamp"))
		tx.CreateIndex("reports_type", ReportsCollection+"*", buntdb.IndexJSON("type"))
		tx.CreateIndex("reports_status", ReportsCollection+"*", buntdb.IndexJSON("status"))

		// User reports indexes
		tx.CreateIndex("user_reports_time", UserReportsCollection+"*", buntdb.IndexJSON("timestamp"))
		tx.CreateIndex("user_reports_user", UserReportsCollection+"*", buntdb.IndexJSON("generated_by"))

		// System reports indexes
		tx.CreateIndex("system_reports_time", SystemReportsCollection+"*", buntdb.IndexJSON("timestamp"))
		tx.CreateIndex("system_reports_type", SystemReportsCollection+"*", buntdb.IndexJSON("type"))

		// Model indexes
		tx.CreateIndex("objects_status", ModelsCollection+"*", buntdb.IndexJSON("status"))
		tx.CreateIndex("objects_type", ModelsCollection+"*", buntdb.IndexJSON("type"))

		// DVE Node indexes
		tx.CreateIndex("dve_nodes_status", DVENodesCollection+"*", buntdb.IndexJSON("status"))
		tx.CreateIndex("dve_nodes_type", DVENodesCollection+"*", buntdb.IndexJSON("type"))

		// Validation tasks indexes
		tx.CreateIndex("validation_tasks_status", ValidationTasksCollection+"*", buntdb.IndexJSON("status"))
		tx.CreateIndex("validation_tasks_type", ValidationTasksCollection+"*", buntdb.IndexJSON("type"))
		tx.CreateIndex("validation_tasks_time", ValidationTasksCollection+"*", buntdb.IndexJSON("timestamp"))

		return nil
	})
}

// StoreMetric stores a metric entry
func (m *BuntDBManager) StoreMetric(metric *MetricEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s%s", MetricsCollection, metric.ID)
	data, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	return m.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, string(data), nil)
		return err
	})
}

// GetMetrics retrieves metrics with optional filtering
func (m *BuntDBManager) GetMetrics(source, metricType string, since time.Time, limit int) ([]*MetricEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var metrics []*MetricEntry
	count := 0

	err := m.db.View(func(tx *buntdb.Tx) error {
		// Use appropriate index based on filters
		var indexName string
		var pattern string

		if source != "" {
			indexName = "metrics_source"
			pattern = source
		} else if metricType != "" {
			indexName = "metrics_type"
			pattern = metricType
		} else {
			indexName = "metrics_time"
			pattern = ""
		}
		_ = pattern // TODO: Use pattern for filtering if needed

		return tx.Ascend(indexName, func(key, value string) bool {
			if limit > 0 && count >= limit {
				return false
			}

			var metric MetricEntry
			if err := json.Unmarshal([]byte(value), &metric); err != nil {
				return true // Continue iteration
			}

			// Apply filters
			if source != "" && metric.Source != source {
				return true
			}
			if metricType != "" && metric.Type != metricType {
				return true
			}
			if !since.IsZero() && metric.Timestamp.Before(since) {
				return true
			}

			metrics = append(metrics, &metric)
			count++
			return true
		})
	})

	return metrics, err
}

// StoreAlert stores an alert entry
func (m *BuntDBManager) StoreAlert(alert *AlertEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s%s", AlertsCollection, alert.ID)
	data, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	return m.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, string(data), nil)
		return err
	})
}

// GetAlerts retrieves alerts with optional filtering
func (m *BuntDBManager) GetAlerts(severity, status string, since time.Time, limit int) ([]*AlertEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var alerts []*AlertEntry
	count := 0

	err := m.db.View(func(tx *buntdb.Tx) error {
		return tx.Descend("alerts_time", func(key, value string) bool {
			if limit > 0 && count >= limit {
				return false
			}

			var alert AlertEntry
			if err := json.Unmarshal([]byte(value), &alert); err != nil {
				return true // Continue iteration
			}

			// Apply filters
			if severity != "" && alert.Severity != severity {
				return true
			}
			if status != "" && alert.Status != status {
				return true
			}
			if !since.IsZero() && alert.Timestamp.Before(since) {
				return true
			}

			alerts = append(alerts, &alert)
			count++
			return true
		})
	})

	return alerts, err
}

// ResolveAlert marks an alert as resolved
func (m *BuntDBManager) ResolveAlert(alertID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s%s", AlertsCollection, alertID)
	return m.db.Update(func(tx *buntdb.Tx) error {
		val, err := tx.Get(key)
		if err != nil {
			return err
		}

		var alert AlertEntry
		if err := json.Unmarshal([]byte(val), &alert); err != nil {
			return err
		}

		// Mark as resolved
		now := time.Now()
		alert.Resolved = true
		alert.ResolvedAt = &now

		data, err := json.Marshal(alert)
		if err != nil {
			return err
		}

		_, _, err = tx.Set(key, string(data), nil)
		return err
	})
}

// StoreReport stores a report entry
func (m *BuntDBManager) StoreReport(report *ReportEntry, isUserReport bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var key string
	if isUserReport {
		key = fmt.Sprintf("%s%s", UserReportsCollection, report.ID)
	} else {
		key = fmt.Sprintf("%s%s", SystemReportsCollection, report.ID)
	}

	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	return m.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, string(data), nil)
		return err
	})
}

// GetReports retrieves reports with optional filtering
func (m *BuntDBManager) GetReports(reportType string, isUserReport bool, since time.Time, limit int) ([]*ReportEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var reports []*ReportEntry
	count := 0

	collection := SystemReportsCollection
	indexName := "system_reports_time"
	if isUserReport {
		collection = UserReportsCollection
		indexName = "user_reports_time"
	}

	err := m.db.View(func(tx *buntdb.Tx) error {
		return tx.Descend(indexName, func(key, value string) bool {
			if !strings.HasPrefix(key, collection) {
				return true
			}

			if limit > 0 && count >= limit {
				return false
			}

			var report ReportEntry
			if err := json.Unmarshal([]byte(value), &report); err != nil {
				return true // Continue iteration
			}

			// Apply filters
			if reportType != "" && report.Type != reportType {
				return true
			}
			if !since.IsZero() && report.Timestamp.Before(since) {
				return true
			}

			reports = append(reports, &report)
			count++
			return true
		})
	})

	return reports, err
}

// UserReportEntry represents a user-specific report entry
type UserReportEntry struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// SystemReportEntry represents a system-wide report entry
type SystemReportEntry struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// StoreUserReport stores a user-specific report entry
func (m *BuntDBManager) StoreUserReport(report *UserReportEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%suser:%s", UserReportsCollection, report.ID)
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}

	return m.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, string(data), nil)
		return err
	})
}

// StoreSystemReport stores a system-wide report entry
func (m *BuntDBManager) StoreSystemReport(report *SystemReportEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%ssystem:%s", SystemReportsCollection, report.ID)
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}

	return m.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, string(data), nil)
		return err
	})
}

// GetUserReports retrieves user reports with filtering
func (m *BuntDBManager) GetUserReports(userID string, since time.Time, until time.Time) ([]*UserReportEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var reports []*UserReportEntry
	err := m.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			// Only process user report keys
			if !strings.HasPrefix(key, UserReportsCollection+"user:") {
				return true
			}

			var report UserReportEntry
			if err := json.Unmarshal([]byte(value), &report); err != nil {
				return true // Continue iteration
			}

			// Filter by user ID and time range
			if report.UserID == userID &&
				report.Timestamp.After(since) &&
				report.Timestamp.Before(until) {
				reports = append(reports, &report)
			}
			return true
		})
	})

	return reports, err
}

// GetSystemReports retrieves system reports with filtering
func (m *BuntDBManager) GetSystemReports(since time.Time, until time.Time) ([]*SystemReportEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var reports []*SystemReportEntry
	err := m.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			// Only process system report keys
			if !strings.HasPrefix(key, SystemReportsCollection+"system:") {
				return true
			}

			var report SystemReportEntry
			if err := json.Unmarshal([]byte(value), &report); err != nil {
				return true // Continue iteration
			}

			// Filter by time range
			if report.Timestamp.After(since) && report.Timestamp.Before(until) {
				reports = append(reports, &report)
			}
			return true
		})
	})

	return reports, err
}

// StoreEvent stores an event entry
func (m *BuntDBManager) StoreEvent(event *EventEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store in metrics collection with event prefix
	key := fmt.Sprintf("%sevents:%s", MetricsCollection, event.ID)
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return m.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, string(data), nil)
		return err
	})
}

// GetEvents retrieves events with optional filtering
func (m *BuntDBManager) GetEvents(eventType, source string, since time.Time, limit int) ([]*EventEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var events []*EventEntry
	count := 0

	err := m.db.View(func(tx *buntdb.Tx) error {
		pattern := MetricsCollection + "events:*"
		_ = pattern // TODO: Use pattern for filtering if needed
		return tx.Ascend("", func(key, value string) bool {
			if !strings.HasPrefix(key, MetricsCollection+"events:") {
				return true
			}

			if limit > 0 && count >= limit {
				return false
			}

			var event EventEntry
			if err := json.Unmarshal([]byte(value), &event); err != nil {
				return true // Continue iteration
			}

			// Apply filters
			if eventType != "" && event.Type != eventType {
				return true
			}
			if source != "" && event.Source != source {
				return true
			}
			if !since.IsZero() && event.Timestamp.Before(since) {
				return true
			}

			events = append(events, &event)
			count++
			return true
		})
	})

	return events, err
}

// CleanupOldMetrics removes metrics older than the specified duration
func (m *BuntDBManager) CleanupOldMetrics(maxAge time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	var keysToDelete []string

	err := m.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			// Only process metric keys
			if !strings.HasPrefix(key, MetricsCollection) {
				return true
			}

			var metric MetricEntry
			if err := json.Unmarshal([]byte(value), &metric); err != nil {
				return true // Continue iteration
			}

			if metric.Timestamp.Before(cutoff) {
				keysToDelete = append(keysToDelete, key)
			}
			return true
		})
	})

	if err != nil {
		return err
	}

	// Delete old metrics
	return m.db.Update(func(tx *buntdb.Tx) error {
		for _, key := range keysToDelete {
			_, err := tx.Delete(key)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// GetStats returns database statistics
func (m *BuntDBManager) GetStats() (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})

	// Count total keys
	totalKeys := 0
	err := m.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			totalKeys++
			return true
		})
	})

	if err != nil {
		return nil, err
	}

	stats["total_keys"] = totalKeys
	stats["database_path"] = m.dbPath
	stats["collections"] = map[string]int{
		"metrics":   m.countKeysWithPrefix(MetricsCollection),
		"alerts":    m.countKeysWithPrefix(AlertsCollection),
		"users":     m.countKeysWithPrefix(UsersCollection),
		"objects":   m.countKeysWithPrefix(ModelsCollection),
		"dve_nodes": m.countKeysWithPrefix(DVENodesCollection),
	}

	// Count all keys in each collection
	collections := map[string]string{
		"metrics":            MetricsCollection,
		"alerts":             AlertsCollection,
		"reports":            ReportsCollection,
		"user_reports":       UserReportsCollection,
		"system_reports":     SystemReportsCollection,
		"objects":            ModelsCollection,
		"dve_nodes":          DVENodesCollection,
		"validation_tasks":   ValidationTasksCollection,
		"validation_results": ValidationResultsCollection,
		"cde_environments":   CDEEnvironmentsCollection,
		"cde_sessions":       CDESessionsCollection,
		"cde_projects":       CDEProjectsCollection,
		"dve_messages":       DVEMessagesCollection,
		"dve_routing":        DVERoutingCollection,
		"dve_registry":       DVERegistryCollection,
		"model_binaries":     ModelBinariesCollection,
		"model_runtime":      ModelRuntimeCollection,
		"config":             ConfigCollection,
		"auth":               AuthCollection,
		"sessions":           SessionsCollection,
	}

	for name, prefix := range collections {
		count := 0
		m.db.View(func(tx *buntdb.Tx) error {
			return tx.Ascend("", func(key, value string) bool {
				if strings.HasPrefix(key, prefix) {
					count++
				}
				return true
			})
		})
		stats[name+"_count"] = count
	}

	// The counts are already set in the loop above, no need to duplicate

	return stats, nil
}

// countKeysWithPrefix counts keys with a specific prefix
func (m *BuntDBManager) countKeysWithPrefix(prefix string) int {
	count := 0
	err := m.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend(prefix+"*", func(key, value string) bool {
			count++
			return true
		})
	})
	if err != nil {
		return 0
	}
	return count
}


// User Management Methods

// CreateUser creates a new user
func (m *BuntDBManager) CreateUser(user *UserEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return m.db.Update(func(tx *buntdb.Tx) error {
		key := UsersCollection + user.ID
		_, _, err := tx.Set(key, string(data), nil)
		return err
	})
}

// GetUser retrieves a user by ID
func (m *BuntDBManager) GetUser(userID string) (*UserEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var user UserEntry
	err := m.db.View(func(tx *buntdb.Tx) error {
		key := UsersCollection + userID
		value, err := tx.Get(key)
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(value), &user)
	})

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (m *BuntDBManager) GetUserByUsername(username string) (*UserEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var foundUser *UserEntry
	err := m.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			if !strings.HasPrefix(key, UsersCollection) {
				return true
			}

			var user UserEntry
			if err := json.Unmarshal([]byte(value), &user); err != nil {
				return true
			}

			if user.Username == username {
				foundUser = &user
				return false
			}

			return true
		})
	})

	if err != nil {
		return nil, err
	}

	if foundUser == nil {
		return nil, fmt.Errorf("user not found")
	}

	return foundUser, nil
}

// UpdateUser updates an existing user
func (m *BuntDBManager) UpdateUser(user *UserEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user.UpdatedAt = time.Now()

	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return m.db.Update(func(tx *buntdb.Tx) error {
		key := UsersCollection + user.ID
		_, _, err := tx.Set(key, string(data), nil)
		return err
	})
}

// DeleteUser deletes a user
func (m *BuntDBManager) DeleteUser(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.db.Update(func(tx *buntdb.Tx) error {
		key := UsersCollection + userID
		_, err := tx.Delete(key)
		return err
	})
}

// ListUsers lists all users with pagination
func (m *BuntDBManager) ListUsers(offset, limit int) ([]*UserEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var users []*UserEntry
	count := 0

	err := m.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			if !strings.HasPrefix(key, UsersCollection) {
				return true
			}

			if count < offset {
				count++
				return true
			}

			if len(users) >= limit {
				return false
			}

			var user UserEntry
			if err := json.Unmarshal([]byte(value), &user); err != nil {
				return true
			}

			users = append(users, &user)
			count++
			return true
		})
	})

	return users, err
}

// Authentication Methods

// CreateAuth creates a new authentication entry
func (m *BuntDBManager) CreateAuth(auth *AuthEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	auth.CreatedAt = time.Now()

	data, err := json.Marshal(auth)
	if err != nil {
		return err
	}

	return m.db.Update(func(tx *buntdb.Tx) error {
		key := AuthCollection + auth.ID
		_, _, err := tx.Set(key, string(data), nil)
		return err
	})
}

// GetAuth retrieves an authentication entry by ID
func (m *BuntDBManager) GetAuth(authID string) (*AuthEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var auth AuthEntry
	err := m.db.View(func(tx *buntdb.Tx) error {
		key := AuthCollection + authID
		value, err := tx.Get(key)
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(value), &auth)
	})

	if err != nil {
		return nil, err
	}

	return &auth, nil
}

// GetAuthByToken retrieves an authentication entry by token hash
func (m *BuntDBManager) GetAuthByToken(tokenHash string) (*AuthEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var foundAuth *AuthEntry
	err := m.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			if !strings.HasPrefix(key, AuthCollection) {
				return true
			}

			var auth AuthEntry
			if err := json.Unmarshal([]byte(value), &auth); err != nil {
				return true
			}

			if auth.TokenHash == tokenHash && !auth.Revoked && time.Now().Before(auth.ExpiresAt) {
				foundAuth = &auth
				return false
			}

			return true
		})
	})

	if err != nil {
		return nil, err
	}

	if foundAuth == nil {
		return nil, fmt.Errorf("auth token not found or expired")
	}

	return foundAuth, nil
}

// RevokeAuth revokes an authentication entry
func (m *BuntDBManager) RevokeAuth(authID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.db.Update(func(tx *buntdb.Tx) error {
		key := AuthCollection + authID
		value, err := tx.Get(key)
		if err != nil {
			return err
		}

		var auth AuthEntry
		if err := json.Unmarshal([]byte(value), &auth); err != nil {
			return err
		}

		auth.Revoked = true

		data, err := json.Marshal(auth)
		if err != nil {
			return err
		}

		_, _, err = tx.Set(key, string(data), nil)
		return err
	})
}

// CleanupExpiredAuth removes expired authentication entries
func (m *BuntDBManager) CleanupExpiredAuth() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var expiredKeys []string

	err := m.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			if !strings.HasPrefix(key, AuthCollection) {
				return true
			}

			var auth AuthEntry
			if err := json.Unmarshal([]byte(value), &auth); err != nil {
				return true
			}

			if now.After(auth.ExpiresAt) {
				expiredKeys = append(expiredKeys, key)
			}

			return true
		})
	})

	if err != nil {
		return err
	}

	// Delete expired entries
	return m.db.Update(func(tx *buntdb.Tx) error {
		for _, key := range expiredKeys {
			tx.Delete(key)
		}
		return nil
	})
}

// Session Management Methods

// CreateSession creates a new session
func (m *BuntDBManager) CreateSession(session *SessionEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session.CreatedAt = time.Now()
	session.LastActivity = time.Now()

	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return m.db.Update(func(tx *buntdb.Tx) error {
		key := SessionsCollection + session.ID
		_, _, err := tx.Set(key, string(data), nil)
		return err
	})
}

// GetSession retrieves a session by ID
func (m *BuntDBManager) GetSession(sessionID string) (*SessionEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var session SessionEntry
	err := m.db.View(func(tx *buntdb.Tx) error {
		key := SessionsCollection + sessionID
		value, err := tx.Get(key)
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(value), &session)
	})

	if err != nil {
		return nil, err
	}

	return &session, nil
}
