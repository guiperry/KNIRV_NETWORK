package objects

import (
	"testing"
	"time"
)

func TestReportRecord_StructFields(t *testing.T) {
	report := ReportRecord{
		ID:            "report-123",
		Type:          "system_health",
		Format:        "pdf",
		FileSize:      1024000,
		SharedWith:    []string{"user-456", "public"},
		Scheduled:     true,
		ScheduleCron:  "0 0 * * *",
		DownloadCount: 5,
	}

	if report.ID != "report-123" {
		t.Errorf("Expected ID 'report-123', got '%s'", report.ID)
	}
	if report.Type != "system_health" {
		t.Errorf("Expected Type 'system_health', got '%s'", report.Type)
	}
	if report.Format != "pdf" {
		t.Errorf("Expected Format 'pdf', got '%s'", report.Format)
	}
	if report.FileSize != 1024000 {
		t.Errorf("Expected FileSize 1024000, got %d", report.FileSize)
	}
	if len(report.SharedWith) != 2 {
		t.Errorf("Expected 2 shared users, got %d", len(report.SharedWith))
	}
	if !report.Scheduled {
		t.Error("Expected Scheduled to be true")
	}
	if report.ScheduleCron != "0 0 * * *" {
		t.Errorf("Expected ScheduleCron '0 0 * * *', got '%s'", report.ScheduleCron)
	}
	if report.DownloadCount != 5 {
		t.Errorf("Expected DownloadCount 5, got %d", report.DownloadCount)
	}
}

func TestReportTemplateRecord_StructFields(t *testing.T) {
	now := time.Now()
	updated := now.Add(time.Hour)

	template := ReportTemplateRecord{
		ID:             "template-123",
		Name:           "System Health Template",
		Description:    "Template for generating system health reports",
		Type:           "system_health",
		Category:       "system",
		TemplateConfig: map[string]interface{}{"layout": "standard", "charts": true},
		DefaultParams:  map[string]interface{}{"period": "24h", "include_charts": true},
		CreatedBy:      "admin",
		IsPublic:       true,
		IsSystem:       true,
		Version:        "1.2.0",
		CreatedAt:      now,
		UpdatedAt:      updated,
		UsageCount:     42,
	}

	if template.ID != "template-123" {
		t.Errorf("Expected ID 'template-123', got '%s'", template.ID)
	}
	if template.Category != "system" {
		t.Errorf("Expected Category 'system', got '%s'", template.Category)
	}
	if !template.IsPublic {
		t.Error("Expected IsPublic to be true")
	}
	if !template.IsSystem {
		t.Error("Expected IsSystem to be true")
	}
	if template.Version != "1.2.0" {
		t.Errorf("Expected Version '1.2.0', got '%s'", template.Version)
	}
	if template.UsageCount != 42 {
		t.Errorf("Expected UsageCount 42, got %d", template.UsageCount)
	}
}

func TestReportShareRecord_StructFields(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(7 * 24 * time.Hour)
	lastAccessed := now.Add(time.Hour)

	share := ReportShareRecord{
		ReportID:     "report-123",
		UserID:       "user-456",
		ShareToken:   "token-abc123",
		Permissions:  []string{"view", "download"},
		ExpiresAt:    expiresAt,
		CreatedAt:    now,
		AccessCount:  3,
		LastAccessed: &lastAccessed,
	}

	if share.ReportID != "report-123" {
		t.Errorf("Expected ReportID 'report-123', got '%s'", share.ReportID)
	}
	if share.ShareToken != "token-abc123" {
		t.Errorf("Expected ShareToken 'token-abc123', got '%s'", share.ShareToken)
	}
	if len(share.Permissions) != 2 {
		t.Errorf("Expected 2 permissions, got %d", len(share.Permissions))
	}
	if share.AccessCount != 3 {
		t.Errorf("Expected AccessCount 3, got %d", share.AccessCount)
	}
}

func TestReportSchedule_StructFields(t *testing.T) {
	now := time.Now()
	updated := now.Add(time.Hour)
	lastRun := now.Add(-time.Hour)
	nextRun := now.Add(time.Hour * 24)

	schedule := ReportSchedule{
		ID:           "schedule-123",
		Name:         "Daily Health Report",
		Description:  "Daily system health report generation",
		TemplateID:   "template-123",
		UserID:       "user-123",
		CronSchedule: "0 0 * * *",
		Parameters:   map[string]interface{}{"period": "24h"},
		Format:       "pdf",
		Recipients:   []string{"admin@example.com", "user-456"},
		IsActive:     true,
		LastRun:      &lastRun,
		NextRun:      &nextRun,
		RunCount:     15,
		CreatedAt:    now,
		UpdatedAt:    updated,
	}

	if schedule.ID != "schedule-123" {
		t.Errorf("Expected ID 'schedule-123', got '%s'", schedule.ID)
	}
	if schedule.CronSchedule != "0 0 * * *" {
		t.Errorf("Expected CronSchedule '0 0 * * *', got '%s'", schedule.CronSchedule)
	}
	if len(schedule.Recipients) != 2 {
		t.Errorf("Expected 2 recipients, got %d", len(schedule.Recipients))
	}
	if !schedule.IsActive {
		t.Error("Expected IsActive to be true")
	}
	if schedule.RunCount != 15 {
		t.Errorf("Expected RunCount 15, got %d", schedule.RunCount)
	}
}

func TestMetricsSnapshot_StructFields(t *testing.T) {
	now := time.Now()

	snapshot := MetricsSnapshot{
		ID:        "snapshot-123",
		Timestamp: now,
		Type:      "node_metrics",
		Data:      map[string]interface{}{"cpu_usage": 45.2, "memory_usage": 67.8},
		NodeID:    "node-123",
		Source:    "system-monitor",
	}

	if snapshot.ID != "snapshot-123" {
		t.Errorf("Expected ID 'snapshot-123', got '%s'", snapshot.ID)
	}
	if snapshot.Type != "node_metrics" {
		t.Errorf("Expected Type 'node_metrics', got '%s'", snapshot.Type)
	}
	if snapshot.NodeID != "node-123" {
		t.Errorf("Expected NodeID 'node-123', got '%s'", snapshot.NodeID)
	}
	if snapshot.Source != "system-monitor" {
		t.Errorf("Expected Source 'system-monitor', got '%s'", snapshot.Source)
	}
	if snapshot.Data["cpu_usage"] != 45.2 {
		t.Errorf("Expected cpu_usage 45.2, got %v", snapshot.Data["cpu_usage"])
	}
}

func TestReportData_StructFields(t *testing.T) {
	now := time.Now()
	period := ReportPeriod{
		StartTime: now.Add(-24 * time.Hour),
		EndTime:   now,
		Duration:  "24h",
	}

	section := ReportSection{
		ID:          "section-1",
		Title:       "System Overview",
		Type:        "metrics",
		Data:        map[string]interface{}{"total_nodes": 10},
		Description: "Overview of system metrics",
	}

	reportData := ReportData{
		Title:       "Daily System Report",
		Description: "Daily system health and performance report",
		GeneratedAt: now,
		GeneratedBy: "user-123",
		Period:      period,
		Summary:     map[string]interface{}{"status": "healthy", "alerts": 0},
		Sections:    []ReportSection{section},
		Metadata:    map[string]interface{}{"version": "1.0", "format": "pdf"},
	}

	if reportData.Title != "Daily System Report" {
		t.Errorf("Expected Title 'Daily System Report', got '%s'", reportData.Title)
	}
	if reportData.Period.Duration != "24h" {
		t.Errorf("Expected Period Duration '24h', got '%s'", reportData.Period.Duration)
	}
	if len(reportData.Sections) != 1 {
		t.Errorf("Expected 1 section, got %d", len(reportData.Sections))
	}
	if reportData.Summary["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", reportData.Summary["status"])
	}
}

func TestReportSection_StructFields(t *testing.T) {
	chartData := ChartData{
		ID:     "chart-1",
		Title:  "CPU Usage",
		Type:   "line",
		Data:   map[string]interface{}{"values": []float64{10.5, 20.3, 15.7}},
		Config: map[string]interface{}{"color": "blue"},
	}

	tableData := TableData{
		ID:      "table-1",
		Title:   "Node Status",
		Headers: []string{"Node ID", "Status", "CPU %"},
		Rows:    []map[string]interface{}{{"node_id": "node-1", "status": "healthy", "cpu": 45.2}},
		Summary: map[string]interface{}{"total_nodes": 1},
	}

	metricData := MetricData{
		ID:          "metric-1",
		Name:        "Total Nodes",
		Value:       10,
		Unit:        "count",
		Description: "Total number of active nodes",
		Trend:       "up",
		Change:      5.5,
	}

	section := ReportSection{
		ID:          "section-1",
		Title:       "Performance Metrics",
		Type:        "metrics",
		Data:        map[string]interface{}{"summary": "good"},
		Charts:      []ChartData{chartData},
		Tables:      []TableData{tableData},
		Metrics:     []MetricData{metricData},
		Description: "System performance overview",
	}

	if section.ID != "section-1" {
		t.Errorf("Expected ID 'section-1', got '%s'", section.ID)
	}
	if len(section.Charts) != 1 {
		t.Errorf("Expected 1 chart, got %d", len(section.Charts))
	}
	if len(section.Tables) != 1 {
		t.Errorf("Expected 1 table, got %d", len(section.Tables))
	}
	if len(section.Metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(section.Metrics))
	}
	if section.Charts[0].Type != "line" {
		t.Errorf("Expected chart type 'line', got '%s'", section.Charts[0].Type)
	}
	if section.Metrics[0].Trend != "up" {
		t.Errorf("Expected metric trend 'up', got '%s'", section.Metrics[0].Trend)
	}
}

func TestChartData_StructFields(t *testing.T) {
	chart := ChartData{
		ID:     "chart-123",
		Title:  "Memory Usage Over Time",
		Type:   "area",
		Data:   map[string]interface{}{"timestamps": []string{"2024-01-01", "2024-01-02"}, "values": []float64{45.2, 67.8}},
		Config: map[string]interface{}{"color": "green", "fill": true},
	}

	if chart.ID != "chart-123" {
		t.Errorf("Expected ID 'chart-123', got '%s'", chart.ID)
	}
	if chart.Type != "area" {
		t.Errorf("Expected Type 'area', got '%s'", chart.Type)
	}
	if chart.Config["color"] != "green" {
		t.Errorf("Expected color 'green', got %v", chart.Config["color"])
	}
}

func TestTableData_StructFields(t *testing.T) {
	table := TableData{
		ID:      "table-123",
		Title:   "Node Performance Summary",
		Headers: []string{"Node ID", "CPU %", "Memory %", "Status"},
		Rows: []map[string]interface{}{
			{"node_id": "node-1", "cpu": 45.2, "memory": 67.8, "status": "healthy"},
			{"node_id": "node-2", "cpu": 32.1, "memory": 54.3, "status": "healthy"},
		},
		Summary: map[string]interface{}{"avg_cpu": 38.65, "avg_memory": 61.05, "healthy_nodes": 2},
	}

	if table.ID != "table-123" {
		t.Errorf("Expected ID 'table-123', got '%s'", table.ID)
	}
	if len(table.Headers) != 4 {
		t.Errorf("Expected 4 headers, got %d", len(table.Headers))
	}
	if len(table.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(table.Rows))
	}
	if table.Summary["healthy_nodes"] != 2 {
		t.Errorf("Expected healthy_nodes 2, got %v", table.Summary["healthy_nodes"])
	}
}

func TestMetricData_StructFields(t *testing.T) {
	metric := MetricData{
		ID:          "metric-123",
		Name:        "Average Response Time",
		Value:       125.5,
		Unit:        "ms",
		Description: "Average API response time",
		Trend:       "down",
		Change:      -12.3,
	}

	if metric.ID != "metric-123" {
		t.Errorf("Expected ID 'metric-123', got '%s'", metric.ID)
	}
	if metric.Value != 125.5 {
		t.Errorf("Expected Value 125.5, got %v", metric.Value)
	}
	if metric.Unit != "ms" {
		t.Errorf("Expected Unit 'ms', got '%s'", metric.Unit)
	}
	if metric.Trend != "down" {
		t.Errorf("Expected Trend 'down', got '%s'", metric.Trend)
	}
	if metric.Change != -12.3 {
		t.Errorf("Expected Change -12.3, got %f", metric.Change)
	}
}

func TestReportGenerationRequest_StructFields(t *testing.T) {
	duration := time.Hour * 24 * 7 // 7 days

	request := ReportGenerationRequest{
		Title:      "Weekly Performance Report",
		Type:       "node_performance",
		Format:     "xlsx",
		Template:   "template-123",
		Parameters: map[string]interface{}{"period": "7d", "include_charts": true},
		UserID:     "user-123",
		ShareWith:  []string{"user-456", "user-789"},
		ExpiresIn:  &duration,
	}

	if request.Title != "Weekly Performance Report" {
		t.Errorf("Expected Title 'Weekly Performance Report', got '%s'", request.Title)
	}
	if request.Type != "node_performance" {
		t.Errorf("Expected Type 'node_performance', got '%s'", request.Type)
	}
	if request.Format != "xlsx" {
		t.Errorf("Expected Format 'xlsx', got '%s'", request.Format)
	}
	if len(request.ShareWith) != 2 {
		t.Errorf("Expected 2 share recipients, got %d", len(request.ShareWith))
	}
	if *request.ExpiresIn != duration {
		t.Errorf("Expected ExpiresIn %v, got %v", duration, *request.ExpiresIn)
	}
}

func TestReportFilter_StructFields(t *testing.T) {
	startDate := time.Now().Add(-7 * 24 * time.Hour)
	endDate := time.Now()
	scheduled := true

	filter := ReportFilter{
		UserID:     "user-123",
		Type:       "system_health",
		Format:     "pdf",
		Status:     "completed",
		CreatedBy:  "admin",
		StartDate:  &startDate,
		EndDate:    &endDate,
		Scheduled:  &scheduled,
		SharedWith: "user-456",
		Limit:      50,
		Offset:     0,
		SortBy:     "created_at",
		SortOrder:  "desc",
	}

	if filter.UserID != "user-123" {
		t.Errorf("Expected UserID 'user-123', got '%s'", filter.UserID)
	}
	if filter.Type != "system_health" {
		t.Errorf("Expected Type 'system_health', got '%s'", filter.Type)
	}
	if filter.Status != "completed" {
		t.Errorf("Expected Status 'completed', got '%s'", filter.Status)
	}
	if *filter.Scheduled != true {
		t.Errorf("Expected Scheduled true, got %v", *filter.Scheduled)
	}
	if filter.Limit != 50 {
		t.Errorf("Expected Limit 50, got %d", filter.Limit)
	}
	if filter.SortOrder != "desc" {
		t.Errorf("Expected SortOrder 'desc', got '%s'", filter.SortOrder)
	}
}

func TestReportStats_StructFields(t *testing.T) {
	userStats := UserReportStats{
		UserID:        "user-123",
		Username:      "admin",
		ReportCount:   25,
		DownloadCount: 150,
		ShareCount:    10,
	}

	activity := ReportActivity{
		Type:      "generated",
		ReportID:  "report-123",
		UserID:    "user-123",
		Username:  "admin",
		Timestamp: time.Now(),
	}

	stats := ReportStats{
		TotalReports:     100,
		ReportsByType:    map[string]int{"system_health": 40, "node_performance": 35, "validation_summary": 25},
		ReportsByFormat:  map[string]int{"pdf": 60, "xlsx": 25, "csv": 15},
		ReportsByStatus:  map[string]int{"completed": 85, "generating": 5, "failed": 10},
		TotalDownloads:   500,
		TotalShares:      75,
		AverageFileSize:  2048000,
		StorageUsed:      204800000,
		MostUsedTemplate: "template-123",
		TopUsers:         []UserReportStats{userStats},
		RecentActivity:   []ReportActivity{activity},
	}

	if stats.TotalReports != 100 {
		t.Errorf("Expected TotalReports 100, got %d", stats.TotalReports)
	}
	if stats.ReportsByType["system_health"] != 40 {
		t.Errorf("Expected system_health reports 40, got %d", stats.ReportsByType["system_health"])
	}
	if stats.TotalDownloads != 500 {
		t.Errorf("Expected TotalDownloads 500, got %d", stats.TotalDownloads)
	}
	if stats.AverageFileSize != 2048000 {
		t.Errorf("Expected AverageFileSize 2048000, got %d", stats.AverageFileSize)
	}
	if len(stats.TopUsers) != 1 {
		t.Errorf("Expected 1 top user, got %d", len(stats.TopUsers))
	}
	if stats.TopUsers[0].ReportCount != 25 {
		t.Errorf("Expected user report count 25, got %d", stats.TopUsers[0].ReportCount)
	}
}

func TestUserReportStats_StructFields(t *testing.T) {
	userStats := UserReportStats{
		UserID:        "user-456",
		Username:      "operator",
		ReportCount:   15,
		DownloadCount: 75,
		ShareCount:    5,
	}

	if userStats.UserID != "user-456" {
		t.Errorf("Expected UserID 'user-456', got '%s'", userStats.UserID)
	}
	if userStats.Username != "operator" {
		t.Errorf("Expected Username 'operator', got '%s'", userStats.Username)
	}
	if userStats.ReportCount != 15 {
		t.Errorf("Expected ReportCount 15, got %d", userStats.ReportCount)
	}
	if userStats.DownloadCount != 75 {
		t.Errorf("Expected DownloadCount 75, got %d", userStats.DownloadCount)
	}
	if userStats.ShareCount != 5 {
		t.Errorf("Expected ShareCount 5, got %d", userStats.ShareCount)
	}
}

func TestReportActivity_StructFields(t *testing.T) {
	now := time.Now()

	activity := ReportActivity{
		Type:      "downloaded",
		ReportID:  "report-456",
		UserID:    "user-789",
		Username:  "viewer",
		Timestamp: now,
	}

	if activity.Type != "downloaded" {
		t.Errorf("Expected Type 'downloaded', got '%s'", activity.Type)
	}
	if activity.ReportID != "report-456" {
		t.Errorf("Expected ReportID 'report-456', got '%s'", activity.ReportID)
	}
	if activity.UserID != "user-789" {
		t.Errorf("Expected UserID 'user-789', got '%s'", activity.UserID)
	}
	if activity.Username != "viewer" {
		t.Errorf("Expected Username 'viewer', got '%s'", activity.Username)
	}
	if !activity.Timestamp.Equal(now) {
		t.Errorf("Expected Timestamp %v, got %v", now, activity.Timestamp)
	}
}
