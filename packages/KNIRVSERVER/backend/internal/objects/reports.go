package objects

import (
	"time"
)

// ReportRecord represents a generated report
type ReportRecord struct {
	ID            string                 `json:"id"`
	Title         string                 `json:"title"`
	Type          string                 `json:"type"`   // "system_health", "node_performance", "validation_summary", "tee_status", "custom"
	Format        string                 `json:"format"` // "pdf", "csv", "json", "html", "xlsx"
	Parameters    map[string]interface{} `json:"parameters"`
	GeneratedBy   string                 `json:"generated_by"`
	FilePath      string                 `json:"file_path"`
	FileSize      int64                  `json:"file_size"`
	SharedWith    []string               `json:"shared_with"` // user IDs or "public"
	Scheduled     bool                   `json:"scheduled"`
	ScheduleCron  string                 `json:"schedule_cron,omitempty"`
	Status        string                 `json:"status"` // "generating", "completed", "failed", "expired"
	CreatedAt     time.Time              `json:"created_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	ExpiresAt     *time.Time             `json:"expires_at,omitempty"`
	DownloadCount int                    `json:"download_count"`
	LastAccessed  *time.Time             `json:"last_accessed,omitempty"`
}

// ReportTemplateRecord represents a report template
type ReportTemplateRecord struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Type           string                 `json:"type"`
	Category       string                 `json:"category"` // "system", "performance", "security", "custom"
	TemplateConfig map[string]interface{} `json:"template_config"`
	DefaultParams  map[string]interface{} `json:"default_params"`
	CreatedBy      string                 `json:"created_by"`
	IsPublic       bool                   `json:"is_public"`
	IsSystem       bool                   `json:"is_system"` // System templates cannot be deleted
	Version        string                 `json:"version"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	UsageCount     int                    `json:"usage_count"`
}

// ReportShareRecord represents a report sharing record
type ReportShareRecord struct {
	ReportID     string     `json:"report_id"`
	UserID       string     `json:"user_id"`
	ShareToken   string     `json:"share_token"`
	Permissions  []string   `json:"permissions"` // "view", "download", "edit"
	ExpiresAt    time.Time  `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	AccessCount  int        `json:"access_count"`
	LastAccessed *time.Time `json:"last_accessed,omitempty"`
}

// ReportSchedule represents a scheduled report
type ReportSchedule struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	TemplateID   string                 `json:"template_id"`
	UserID       string                 `json:"user_id"`
	CronSchedule string                 `json:"cron_schedule"`
	Parameters   map[string]interface{} `json:"parameters"`
	Format       string                 `json:"format"`
	Recipients   []string               `json:"recipients"` // Email addresses or user IDs
	IsActive     bool                   `json:"is_active"`
	LastRun      *time.Time             `json:"last_run,omitempty"`
	NextRun      *time.Time             `json:"next_run,omitempty"`
	RunCount     int                    `json:"run_count"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// MetricsSnapshot represents a snapshot of system metrics
type MetricsSnapshot struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"` // "node_metrics", "system_health", "validation_stats", "tee_metrics"
	Data      map[string]interface{} `json:"data"`
	NodeID    string                 `json:"node_id,omitempty"`
	Source    string                 `json:"source"` // Component that generated the metrics
}

// ReportData represents the data structure for report generation
type ReportData struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	GeneratedAt time.Time              `json:"generated_at"`
	GeneratedBy string                 `json:"generated_by"`
	Period      ReportPeriod           `json:"period"`
	Summary     map[string]interface{} `json:"summary"`
	Sections    []ReportSection        `json:"sections"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ReportPeriod represents the time period for a report
type ReportPeriod struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  string    `json:"duration"`
}

// ReportSection represents a section within a report
type ReportSection struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Type        string       `json:"type"` // "table", "chart", "text", "metrics"
	Data        interface{}  `json:"data"`
	Charts      []ChartData  `json:"charts,omitempty"`
	Tables      []TableData  `json:"tables,omitempty"`
	Metrics     []MetricData `json:"metrics,omitempty"`
	Description string       `json:"description,omitempty"`
}

// ChartData represents chart data for reports
type ChartData struct {
	ID     string                 `json:"id"`
	Title  string                 `json:"title"`
	Type   string                 `json:"type"` // "line", "bar", "pie", "area"
	Data   map[string]interface{} `json:"data"`
	Config map[string]interface{} `json:"config"`
}

// TableData represents table data for reports
type TableData struct {
	ID      string                   `json:"id"`
	Title   string                   `json:"title"`
	Headers []string                 `json:"headers"`
	Rows    []map[string]interface{} `json:"rows"`
	Summary map[string]interface{}   `json:"summary,omitempty"`
}

// MetricData represents metric data for reports
type MetricData struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Value       interface{} `json:"value"`
	Unit        string      `json:"unit,omitempty"`
	Description string      `json:"description,omitempty"`
	Trend       string      `json:"trend,omitempty"`  // "up", "down", "stable"
	Change      float64     `json:"change,omitempty"` // Percentage change
}

// ReportGenerationRequest represents a request to generate a report
type ReportGenerationRequest struct {
	Title      string                 `json:"title"`
	Type       string                 `json:"type"`
	Format     string                 `json:"format"`
	Template   string                 `json:"template,omitempty"`
	Parameters map[string]interface{} `json:"parameters"`
	UserID     string                 `json:"user_id"`
	ShareWith  []string               `json:"share_with,omitempty"`
	ExpiresIn  *time.Duration         `json:"expires_in,omitempty"`
}

// ReportFilter represents filters for report queries
type ReportFilter struct {
	UserID     string     `json:"user_id,omitempty"`
	Type       string     `json:"type,omitempty"`
	Format     string     `json:"format,omitempty"`
	Status     string     `json:"status,omitempty"`
	CreatedBy  string     `json:"created_by,omitempty"`
	StartDate  *time.Time `json:"start_date,omitempty"`
	EndDate    *time.Time `json:"end_date,omitempty"`
	Scheduled  *bool      `json:"scheduled,omitempty"`
	SharedWith string     `json:"shared_with,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
	SortBy     string     `json:"sort_by,omitempty"`
	SortOrder  string     `json:"sort_order,omitempty"` // "asc", "desc"
}

// ReportStats represents statistics about reports
type ReportStats struct {
	TotalReports     int               `json:"total_reports"`
	ReportsByType    map[string]int    `json:"reports_by_type"`
	ReportsByFormat  map[string]int    `json:"reports_by_format"`
	ReportsByStatus  map[string]int    `json:"reports_by_status"`
	TotalDownloads   int               `json:"total_downloads"`
	TotalShares      int               `json:"total_shares"`
	AverageFileSize  int64             `json:"average_file_size"`
	StorageUsed      int64             `json:"storage_used"`
	MostUsedTemplate string            `json:"most_used_template"`
	TopUsers         []UserReportStats `json:"top_users"`
	RecentActivity   []ReportActivity  `json:"recent_activity"`
}

// UserReportStats represents report statistics for a user
type UserReportStats struct {
	UserID        string `json:"user_id"`
	Username      string `json:"username"`
	ReportCount   int    `json:"report_count"`
	DownloadCount int    `json:"download_count"`
	ShareCount    int    `json:"share_count"`
}

// ReportActivity represents recent report activity
type ReportActivity struct {
	Type      string    `json:"type"` // "generated", "downloaded", "shared", "expired"
	ReportID  string    `json:"report_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Timestamp time.Time `json:"timestamp"`
}
