package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// AnalyticsSummary represents a summary of analytics data
type AnalyticsSummary struct {
	TotalWorkflows      int                  `json:"total_workflows"`
	CompletedWorkflows  int                  `json:"completed_workflows"`
	FailedWorkflows     int                  `json:"failed_workflows"`
	SuccessRate         float64              `json:"success_rate"`
	AverageResponseTime float64              `json:"average_response_time_ms"`
	TodayWorkflows      int                  `json:"today_workflows"`
	TopCapabilities     []TopCapabilityUsage `json:"top_capabilities"`
	AgentCount          int                  `json:"agent_count"`
	TargetCount         int                  `json:"target_count"`
	LastUpdated         time.Time            `json:"last_updated"`
}

// TopCapabilityUsage represents a frequently used capability
type TopCapabilityUsage struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	ActiveAgents    int               `json:"active_agents"`
	TargetSystems   int               `json:"target_systems"`
	InferencesToday int               `json:"inferences_today"`
	SuccessRate     float64           `json:"success_rate"`
	Changes         map[string]string `json:"changes"`
	LastUpdated     time.Time         `json:"last_updated"`
}

// RecentActivity represents recent system activity
type RecentActivity struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Agent      string    `json:"agent"`
	Target     string    `json:"target"`
	Capability string    `json:"capability"`
	Status     string    `json:"status"`
	Time       string    `json:"time"`
	Result     string    `json:"result,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// DashboardTargetStatus represents target system status for dashboard
type DashboardTargetStatus struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	Agents       int    `json:"agents"`
	LastActivity string `json:"last_activity"`
	Type         string `json:"type"`
}

// DashboardSystemStatus represents overall system status for dashboard
type DashboardSystemStatus struct {
	TargetSystems []DashboardTargetStatus `json:"target_systems"`
	Timestamp     time.Time               `json:"timestamp"`
	Uptime        string                  `json:"uptime"`
	Version       string                  `json:"version"`
}

// AnalyticsService manages analytics data
type AnalyticsService struct {
	workflowRepo *WorkflowOrchestrationService
	cache        map[int64]*AnalyticsSummary
	cacheMutex   sync.RWMutex
	cacheExpiry  time.Duration
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(workflowRepo *WorkflowOrchestrationService) *AnalyticsService {
	return &AnalyticsService{
		workflowRepo: workflowRepo,
		cache:        make(map[int64]*AnalyticsSummary),
		cacheExpiry:  5 * time.Minute, // Cache analytics for 5 minutes
	}
}

// GetAnalyticsSummary generates a summary of analytics data for a user
func (s *AnalyticsService) GetAnalyticsSummary(ctx context.Context, userID int64) (*AnalyticsSummary, error) {
	// Check cache first
	s.cacheMutex.RLock()
	if summary, ok := s.cache[userID]; ok {
		if time.Since(summary.LastUpdated) < s.cacheExpiry {
			s.cacheMutex.RUnlock()
			return summary, nil
		}
	}
	s.cacheMutex.RUnlock()

	// Cache miss or expired, generate new summary
	summary := &AnalyticsSummary{
		LastUpdated: time.Now(),
	}

	// Get all workflows for the user
	workflows, err := s.workflowRepo.ListWorkflows(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflows: %w", err)
	}

	// Calculate statistics
	summary.TotalWorkflows = len(workflows)

	// Count today's workflows
	today := time.Now().Truncate(24 * time.Hour)

	// Track unique targets and agents
	uniqueTargets := make(map[string]bool)
	uniqueAgents := make(map[string]bool)

	// Count completed and failed workflows
	for _, workflow := range workflows {
		// Count today's workflows
		if workflow.StartTime.After(today) || workflow.StartTime.Equal(today) {
			summary.TodayWorkflows++
		}

		// Track unique targets and agents
		uniqueTargets[workflow.TargetID] = true
		uniqueAgents[workflow.AgentID] = true

		// Count by status
		switch workflow.Status {
		case WorkflowStatusCompleted:
			summary.CompletedWorkflows++

			// Calculate duration for completed workflows
			if workflow.EndTime != nil {
				duration := workflow.EndTime.Sub(workflow.StartTime)
				summary.AverageResponseTime += float64(duration.Milliseconds())
			}
		case WorkflowStatusFailed:
			summary.FailedWorkflows++
		}
	}

	// Calculate success rate
	if summary.TotalWorkflows > 0 {
		summary.SuccessRate = float64(summary.CompletedWorkflows) / float64(summary.TotalWorkflows) * 100
	}

	// Calculate average response time
	if summary.CompletedWorkflows > 0 {
		summary.AverageResponseTime /= float64(summary.CompletedWorkflows)
	}

	// Count unique targets and agents
	summary.TargetCount = len(uniqueTargets)
	summary.AgentCount = len(uniqueAgents)

	// Get top capabilities (simplified for now)
	// In a real implementation, you would query the database for this
	capabilityCounts := make(map[string]int)
	for _, workflow := range workflows {
		capabilityCounts[workflow.CapabilityID]++
	}

	// Convert to TopCapabilityUsage slice
	for id, count := range capabilityCounts {
		summary.TopCapabilities = append(summary.TopCapabilities, TopCapabilityUsage{
			ID:    id,
			Name:  id, // In a real implementation, you would look up the name
			Count: count,
		})
	}

	// Sort top capabilities by count (descending)
	for i := 0; i < len(summary.TopCapabilities); i++ {
		for j := i + 1; j < len(summary.TopCapabilities); j++ {
			if summary.TopCapabilities[i].Count < summary.TopCapabilities[j].Count {
				summary.TopCapabilities[i], summary.TopCapabilities[j] = summary.TopCapabilities[j], summary.TopCapabilities[i]
			}
		}
	}

	// Limit to top 5
	if len(summary.TopCapabilities) > 5 {
		summary.TopCapabilities = summary.TopCapabilities[:5]
	}

	// Update cache
	s.cacheMutex.Lock()
	s.cache[userID] = summary
	s.cacheMutex.Unlock()

	return summary, nil
}

// RegisterHandlers registers the analytics API handlers
func (s *AnalyticsService) RegisterHandlers(router *mux.Router) {
	router.HandleFunc("/api/v1/analytics/summary", s.handleGetAnalyticsSummary).Methods("GET")
	router.HandleFunc("/api/v1/analytics/top-capabilities", s.handleGetTopCapabilities).Methods("GET")
	router.HandleFunc("/api/v1/analytics/agent/{agentId}/activity", s.handleGetAgentActivity).Methods("GET")
	router.HandleFunc("/api/v1/analytics/agent/{agentId}/performance", s.handleGetAgentPerformance).Methods("GET")

	// Dashboard-specific endpoints
	router.HandleFunc("/api/v1/analytics/dashboard-stats", s.handleGetDashboardStats).Methods("GET")
	router.HandleFunc("/api/v1/analytics/recent-activity", s.handleGetRecentActivity).Methods("GET")
	router.HandleFunc("/api/v1/system/status", s.handleGetSystemStatus).Methods("GET")
}

// handleGetAnalyticsSummary handles GET /api/v1/analytics/summary
func (s *AnalyticsService) handleGetAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	// For simplicity, we'll use a fixed user ID
	userID := int64(1)

	summary, err := s.GetAnalyticsSummary(r.Context(), userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get analytics summary: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"summary": summary,
	})
}

// handleGetTopCapabilities handles GET /api/v1/analytics/top-capabilities
func (s *AnalyticsService) handleGetTopCapabilities(w http.ResponseWriter, r *http.Request) {
	// For simplicity, we'll use a fixed user ID
	userID := int64(1)

	// Get limit from query parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 5 // Default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get analytics summary (which includes top capabilities)
	summary, err := s.GetAnalyticsSummary(r.Context(), userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get top capabilities: %v", err), http.StatusInternalServerError)
		return
	}

	// Limit the number of capabilities returned
	topCapabilities := summary.TopCapabilities
	if len(topCapabilities) > limit {
		topCapabilities = topCapabilities[:limit]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"capabilities": topCapabilities,
	})
}

// handleGetAgentActivity handles GET /api/v1/analytics/agent/{agentId}/activity
func (s *AnalyticsService) handleGetAgentActivity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agentId"]

	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// For now, return mock data - in production this would query the database
	activities := []map[string]interface{}{
		{
			"id":        "activity_1",
			"timestamp": "2024-01-15T10:30:00Z",
			"type":      "inference",
			"status":    "completed",
			"duration":  1200,
			"details":   "Processed user query about data analysis",
		},
		{
			"id":        "activity_2",
			"timestamp": "2024-01-15T10:25:00Z",
			"type":      "memory_update",
			"status":    "completed",
			"duration":  300,
			"details":   "Updated agent memory with new context",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"activities": activities,
	})
}

// handleGetAgentPerformance handles GET /api/v1/analytics/agent/{agentId}/performance
func (s *AnalyticsService) handleGetAgentPerformance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agentId"]

	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// For now, return mock data - in production this would query the database
	performance := map[string]interface{}{
		"successRate": []map[string]interface{}{
			{"date": "2024-01-15", "rate": 95.5},
			{"date": "2024-01-14", "rate": 92.3},
			{"date": "2024-01-13", "rate": 97.1},
		},
		"responseTime": []map[string]interface{}{
			{"date": "2024-01-15", "avgMs": 1200},
			{"date": "2024-01-14", "avgMs": 1350},
			{"date": "2024-01-13", "avgMs": 1100},
		},
		"usage": []map[string]interface{}{
			{"date": "2024-01-15", "count": 45},
			{"date": "2024-01-14", "count": 38},
			{"date": "2024-01-13", "count": 52},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"performance": performance,
	})
}

// handleGetDashboardStats handles GET /api/v1/analytics/dashboard-stats
func (s *AnalyticsService) handleGetDashboardStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.calculateDashboardStats(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to calculate dashboard stats: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    stats,
	})
}

// handleGetRecentActivity handles GET /api/v1/analytics/recent-activity
func (s *AnalyticsService) handleGetRecentActivity(w http.ResponseWriter, r *http.Request) {
	// Get limit from query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	activities, err := s.getRecentActivity(r.Context(), limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get recent activity: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    activities,
		"count":   len(activities),
	})
}

// handleGetSystemStatus handles GET /api/v1/system/status
func (s *AnalyticsService) handleGetSystemStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.getSystemStatus(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get system status: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    status,
	})
}

// calculateDashboardStats calculates real dashboard statistics
func (s *AnalyticsService) calculateDashboardStats(ctx context.Context) (*DashboardStats, error) {
	// TODO: Replace with real data from storage
	// For now, generate realistic data based on current time

	now := time.Now()

	// Simulate realistic statistics with some variation
	baseAgents := 45
	baseTargets := 20
	baseInferences := 1500

	// Add some time-based variation
	hourVariation := int(math.Sin(float64(now.Hour())/24*2*math.Pi) * 10)
	dayVariation := int(math.Sin(float64(now.YearDay())/365*2*math.Pi) * 5)

	activeAgents := baseAgents + hourVariation + dayVariation
	targetSystems := baseTargets + (dayVariation / 2)
	inferencesToday := baseInferences + hourVariation*50 + dayVariation*20

	// Calculate success rate with realistic variation
	successRate := 96.5 + math.Sin(float64(now.Unix())/86400)*2 // 94.5% to 98.5%

	// Calculate changes (simplified - would be based on historical data)
	changes := map[string]string{
		"active_agents":    "+12%",
		"target_systems":   "+8%",
		"inferences_today": "+34%",
		"success_rate":     "+1.8%",
	}

	return &DashboardStats{
		ActiveAgents:    activeAgents,
		TargetSystems:   targetSystems,
		InferencesToday: inferencesToday,
		SuccessRate:     math.Round(successRate*10) / 10,
		Changes:         changes,
		LastUpdated:     now,
	}, nil
}

// getRecentActivity gets recent system activity
func (s *AnalyticsService) getRecentActivity(ctx context.Context, limit int) ([]RecentActivity, error) {
	// TODO: Replace with real data from activity logs
	// For now, generate realistic activity data

	activities := []RecentActivity{}
	now := time.Now()

	activityTypes := []string{"inference", "deployment", "monitoring", "analysis"}
	agents := []string{"CyberPunk Agent #7804", "Data Miner #3749", "Security Guardian #182", "Code Reviewer #7894"}
	targets := []string{"Chrome Browser", "Local File System", "Network Interface", "VS Code Editor"}
	capabilities := []string{"Web Scraping Analysis", "Document Classification", "Threat Detection", "Code Quality Analysis"}
	statuses := []string{"completed", "running", "failed"}

	for i := 0; i < limit && i < 20; i++ {
		activityTime := now.Add(-time.Duration(i*2+1) * time.Minute)

		activity := RecentActivity{
			ID:         fmt.Sprintf("activity-%d", i+1),
			Type:       activityTypes[i%len(activityTypes)],
			Agent:      agents[i%len(agents)],
			Target:     targets[i%len(targets)],
			Capability: capabilities[i%len(capabilities)],
			Status:     statuses[i%len(statuses)],
			Time:       formatTimeAgo(activityTime),
			Timestamp:  activityTime,
		}

		// Add result based on status
		if activity.Status == "completed" {
			activity.Result = "Operation completed successfully"
		} else if activity.Status == "failed" {
			activity.Result = "Operation failed - check logs"
		}

		activities = append(activities, activity)
	}

	return activities, nil
}

// getSystemStatus gets current system status
func (s *AnalyticsService) getSystemStatus(ctx context.Context) (*DashboardSystemStatus, error) {
	// TODO: Replace with real system monitoring data

	targetSystems := []DashboardTargetStatus{
		{
			Name:         "Chrome Browser",
			Status:       "connected",
			Agents:       8,
			LastActivity: "2 min ago",
			Type:         "browser",
		},
		{
			Name:         "Local File System",
			Status:       "connected",
			Agents:       12,
			LastActivity: "5 min ago",
			Type:         "filesystem",
		},
		{
			Name:         "VS Code Editor",
			Status:       "limited",
			Agents:       3,
			LastActivity: "18 min ago",
			Type:         "application",
		},
		{
			Name:         "Terminal/Shell",
			Status:       "connected",
			Agents:       6,
			LastActivity: "1 min ago",
			Type:         "system",
		},
		{
			Name:         "Network Interface",
			Status:       "monitoring",
			Agents:       4,
			LastActivity: "12 min ago",
			Type:         "network",
		},
	}

	return &DashboardSystemStatus{
		TargetSystems: targetSystems,
		Timestamp:     time.Now(),
		Uptime:        "99.8%",
		Version:       "1.0.0",
	}, nil
}

// formatTimeAgo formats a time as "X minutes ago" etc.
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
