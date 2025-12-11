package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// HEARTService provides HTTP endpoints for KNIRVCORTEX to query HEART
type HEARTService struct {
	bridge    *CerebrasBridge
	processor *NetworkMetricsProcessor
	stats     *HEARTServiceStats
	mu        sync.RWMutex
}

type HEARTServiceStats struct {
	TotalInquiries       uint64
	SuccessfulResponses  uint64
	FailedResponses      uint64
	TotalResponseTimeMs  float64
	MinResponseTimeMs    float64
	MaxResponseTimeMs    float64
	ErrorTypeCounts      map[string]uint64
	HeuristicUsage       map[string]uint64
}

// HEARTErrorInquiry matches the proto definition
type HEARTErrorInquiry struct {
	ErrorID          string            `json:"error_id"`
	ErrorType        string            `json:"error_type"`
	ErrorMessage     string            `json:"error_message"`
	ErrorContext     string            `json:"error_context"`
	StackTrace       string            `json:"stack_trace,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	NetworkMetrics   *NetworkMetrics   `json:"network_metrics,omitempty"`
	Prompt           string            `json:"prompt,omitempty"`
	ModelResponse    string            `json:"model_response,omitempty"`
	ConfidenceScore  float32           `json:"confidence_score,omitempty"`
	Timestamp        uint64            `json:"timestamp"`
}

// NetworkMetrics for HEART analysis
type NetworkMetrics struct {
	NodeID         uint32    `json:"node_id"`
	TimeSlice      uint32    `json:"time_slice"`
	MeasurementVec []float32 `json:"measurement_vec"`
	TrafficIn      float32   `json:"traffic_in"`
	TrafficOut     float32   `json:"traffic_out"`
	LatencyMs      float32   `json:"latency_ms"`
	ErrorRate      float32   `json:"error_rate"`
	CPULoad        float32   `json:"cpu_load"`
}

// HEARTHeuristicResponse matches the proto definition
type HEARTHeuristicResponse struct {
	InquiryID           string          `json:"inquiry_id"`
	CommandVector       []float32       `json:"command_vector"`
	AlertLevel          uint32          `json:"alert_level"`
	HeuristicID         uint32          `json:"heuristic_id"`
	TargetNodeID        uint32          `json:"target_node_id"`
	ConfidenceScore     float32         `json:"confidence_score"`
	ActionParameters    []float32       `json:"action_parameters"`
	AnalysisSummary     string          `json:"analysis_summary"`
	RecommendedActions  []string        `json:"recommended_actions"`
	DebugInsights       []string        `json:"debug_insights"`
	IdentifiedPatterns  []ErrorPattern  `json:"identified_patterns"`
	SimilarErrors       []SimilarError  `json:"similar_errors"`
	ProcessingTimeMs    float32         `json:"processing_time_ms"`
	Timestamp           uint64          `json:"timestamp"`
}

type ErrorPattern struct {
	PatternID       string   `json:"pattern_id"`
	PatternName     string   `json:"pattern_name"`
	Description     string   `json:"description"`
	MatchConfidence float32  `json:"match_confidence"`
	Indicators      []string `json:"indicators"`
}

type SimilarError struct {
	ErrorID         string  `json:"error_id"`
	ErrorType       string  `json:"error_type"`
	SimilarityScore float32 `json:"similarity_score"`
	Resolution      string  `json:"resolution"`
	SkillID         string  `json:"skill_id,omitempty"`
}

// NewHEARTService creates a new HEART service
func NewHEARTService(bridge *CerebrasBridge, processor *NetworkMetricsProcessor) *HEARTService {
	return &HEARTService{
		bridge:    bridge,
		processor: processor,
		stats: &HEARTServiceStats{
			ErrorTypeCounts: make(map[string]uint64),
			HeuristicUsage:  make(map[string]uint64),
			MinResponseTimeMs: 999999.0,
		},
	}
}

// Start starts the HEART HTTP service
func (hs *HEARTService) Start(port int) error {
	http.HandleFunc("/heart/analyze", hs.handleAnalyze)
	http.HandleFunc("/heart/health", hs.handleHealth)
	http.HandleFunc("/heart/stats", hs.handleStats)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting HEART service on %s", addr)
	return http.ListenAndServe(addr, nil)
}

// handleAnalyze processes error inquiries from CORTEX
func (hs *HEARTService) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Enable CORS for WASM clients
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	startTime := time.Now()

	// Parse inquiry
	var inquiry HEARTErrorInquiry
	if err := json.NewDecoder(r.Body).Decode(&inquiry); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("HEART inquiry: %s (type: %s)", inquiry.ErrorID, inquiry.ErrorType)

	// Update stats
	hs.mu.Lock()
	hs.stats.TotalInquiries++
	hs.stats.ErrorTypeCounts[inquiry.ErrorType]++
	hs.mu.Unlock()

	// Process inquiry
	response, err := hs.processInquiry(&inquiry)
	if err != nil {
		hs.mu.Lock()
		hs.stats.FailedResponses++
		hs.mu.Unlock()

		http.Error(w, fmt.Sprintf("Processing failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Update stats
	elapsed := time.Since(startTime).Milliseconds()
	hs.mu.Lock()
	hs.stats.SuccessfulResponses++
	hs.stats.TotalResponseTimeMs += float64(elapsed)
	if float64(elapsed) < hs.stats.MinResponseTimeMs {
		hs.stats.MinResponseTimeMs = float64(elapsed)
	}
	if float64(elapsed) > hs.stats.MaxResponseTimeMs {
		hs.stats.MaxResponseTimeMs = float64(elapsed)
	}
	hs.stats.HeuristicUsage[fmt.Sprintf("h%d", response.HeuristicID)]++
	hs.mu.Unlock()

	response.ProcessingTimeMs = float32(elapsed)

	// Return response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// processInquiry analyzes the error using HEART transformer
func (hs *HEARTService) processInquiry(inquiry *HEARTErrorInquiry) (*HEARTHeuristicResponse, error) {
	// Prepare input for HEART
	var heartInput *HEARTInput

	if inquiry.NetworkMetrics != nil {
		// Use network metrics if provided
		heartInput = &HEARTInput{
			Measurements: [][]float32{inquiry.NetworkMetrics.MeasurementVec},
			NodeIDs:      []uint32{inquiry.NetworkMetrics.NodeID},
			TimeSlices:   []uint32{inquiry.NetworkMetrics.TimeSlice},
		}
	} else {
		// Synthesize metrics from error context
		heartInput = hs.synthesizeMetricsFromError(inquiry)
	}

	// Run inference on HEART (Cerebras WSE)
	output, err := hs.bridge.RunInference(heartInput)
	if err != nil {
		return nil, fmt.Errorf("HEART inference failed: %w", err)
	}

	// Interpret HEART output
	if len(output.Commands) == 0 || len(output.Commands[0]) < 8 {
		return nil, fmt.Errorf("invalid HEART output")
	}

	command := output.Commands[0]

	// Extract command components
	alertLevel := uint32(command[0])
	heuristicID := uint32(command[1])
	targetNodeID := uint32(command[2])
	confidence := command[3]
	actionParams := command[4:8]

	// Generate analysis
	analysis := hs.generateAnalysisSummary(inquiry, alertLevel, heuristicID)
	actions := hs.generateRecommendedActions(inquiry, heuristicID)
	insights := hs.generateDebugInsights(inquiry, command)

	// Identify patterns (simulated - would use KNIRVGRAPH in production)
	patterns := hs.identifyErrorPatterns(inquiry)

	// Find similar errors (simulated - would query KNIRVGRAPH)
	similarErrors := hs.findSimilarErrors(inquiry)

	response := &HEARTHeuristicResponse{
		InquiryID:          inquiry.ErrorID,
		CommandVector:      command,
		AlertLevel:         alertLevel,
		HeuristicID:        heuristicID,
		TargetNodeID:       targetNodeID,
		ConfidenceScore:    confidence,
		ActionParameters:   actionParams,
		AnalysisSummary:    analysis,
		RecommendedActions: actions,
		DebugInsights:      insights,
		IdentifiedPatterns: patterns,
		SimilarErrors:      similarErrors,
		Timestamp:          uint64(time.Now().Unix()),
	}

	return response, nil
}

// synthesizeMetricsFromError creates measurement vectors from error context
func (hs *HEARTService) synthesizeMetricsFromError(inquiry *HEARTErrorInquiry) *HEARTInput {
	// Create synthetic metrics based on error characteristics
	measurements := make([]float32, 16)

	// Encode error type (simple hash)
	typeHash := 0.0
	for _, c := range inquiry.ErrorType {
		typeHash += float64(c)
	}
	measurements[0] = float32(typeHash / 1000.0)

	// Encode error severity
	switch inquiry.ErrorType {
	case "InferenceError", "ModelError":
		measurements[1] = 5.0 // High severity
	case "NetworkError":
		measurements[1] = 4.0
	case "TypeError":
		measurements[1] = 3.0
	default:
		measurements[1] = 2.0
	}

	// Other synthetic features
	measurements[2] = float32(len(inquiry.ErrorMessage)) / 100.0
	measurements[3] = float32(len(inquiry.ErrorContext)) / 100.0

	if inquiry.ConfidenceScore > 0 {
		measurements[4] = inquiry.ConfidenceScore
	}

	// Fill rest with normalized metadata
	metaIdx := 5
	for k, v := range inquiry.Metadata {
		if metaIdx < 16 {
			valHash := 0.0
			for _, c := range k + v {
				valHash += float64(c)
			}
			measurements[metaIdx] = float32(valHash / 1000.0)
			metaIdx++
		}
	}

	return &HEARTInput{
		Measurements: [][]float32{measurements},
		NodeIDs:      []uint32{0},
		TimeSlices:   []uint32{uint32(inquiry.Timestamp % 64)},
	}
}

// generateAnalysisSummary creates human-readable analysis
func (hs *HEARTService) generateAnalysisSummary(inquiry *HEARTErrorInquiry, alertLevel, heuristicID uint32) string {
	severity := "medium"
	switch alertLevel {
	case 5:
		severity = "critical"
	case 4:
		severity = "high"
	case 3:
		severity = "medium"
	case 2:
		severity = "low"
	case 1:
		severity = "info"
	}

	return fmt.Sprintf("HEART Analysis: %s severity %s detected. Recommended heuristic: H%d. %s",
		severity, inquiry.ErrorType, heuristicID, inquiry.ErrorMessage)
}

// generateRecommendedActions creates actionable recommendations
func (hs *HEARTService) generateRecommendedActions(inquiry *HEARTErrorInquiry, heuristicID uint32) []string {
	// Heuristic-specific actions
	switch heuristicID {
	case 101: // Type errors
		return []string{
			"Validate input types before processing",
			"Add type guards to prevent undefined access",
			"Review variable initialization",
		}
	case 201: // Network errors
		return []string{
			"Implement exponential backoff for retries",
			"Check endpoint availability and CORS configuration",
			"Add fallback to cached data if available",
		}
	case 301: // Model errors
		return []string{
			"Verify model weights are loaded correctly",
			"Check input tensor shapes and formats",
			"Consider reloading model weights",
			"Apply relevant LoRA adapter if available",
		}
	default:
		return []string{
			"Review error context and stack trace",
			"Check system logs for related errors",
			"Consider retry with exponential backoff",
		}
	}
}

// generateDebugInsights provides debugging information
func (hs *HEARTService) generateDebugInsights(inquiry *HEARTErrorInquiry, command []float32) []string {
	insights := []string{
		fmt.Sprintf("Error type: %s", inquiry.ErrorType),
		fmt.Sprintf("Command vector magnitude: %.2f", vectorMagnitude(command)),
	}

	if len(inquiry.Metadata) > 0 {
		insights = append(insights, fmt.Sprintf("Metadata entries: %d", len(inquiry.Metadata)))
	}

	return insights
}

// identifyErrorPatterns finds known patterns in the error
func (hs *HEARTService) identifyErrorPatterns(inquiry *HEARTErrorInquiry) []ErrorPattern {
	patterns := []ErrorPattern{}

	// Simple pattern matching (would use KNIRVGRAPH in production)
	if contains(inquiry.ErrorMessage, "undefined") {
		patterns = append(patterns, ErrorPattern{
			PatternID:       "PTN001",
			PatternName:     "Undefined Reference",
			Description:     "Accessing undefined variable or property",
			MatchConfidence: 0.85,
			Indicators:      []string{"undefined", "null reference"},
		})
	}

	if contains(inquiry.ErrorType, "Network") {
		patterns = append(patterns, ErrorPattern{
			PatternID:       "PTN002",
			PatternName:     "Network Connectivity",
			Description:     "Network request failed or timed out",
			MatchConfidence: 0.90,
			Indicators:      []string{"fetch failed", "timeout", "CORS"},
		})
	}

	return patterns
}

// findSimilarErrors queries for similar past errors
func (hs *HEARTService) findSimilarErrors(inquiry *HEARTErrorInquiry) []SimilarError {
	// Simulated - would query KNIRVGRAPH in production
	return []SimilarError{
		{
			ErrorID:         "ERR-2024-001",
			ErrorType:       inquiry.ErrorType,
			SimilarityScore: 0.78,
			Resolution:      "Applied type validation LoRA adapter",
			SkillID:         "skill-typecheck-v1",
		},
	}
}

// handleHealth returns service health status
func (hs *HEARTService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	health := map[string]interface{}{
		"available":   true,
		"status":      "online",
		"avg_latency_ms": hs.getAvgLatency(),
		"total_queries":  hs.stats.TotalInquiries,
		"success_rate": hs.getSuccessRate(),
	}

	json.NewEncoder(w).Encode(health)
}

// handleStats returns detailed statistics
func (hs *HEARTService) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	hs.mu.RLock()
	defer hs.mu.RUnlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_inquiries":       hs.stats.TotalInquiries,
		"successful_responses":  hs.stats.SuccessfulResponses,
		"failed_responses":      hs.stats.FailedResponses,
		"avg_response_time_ms":  hs.getAvgLatency(),
		"min_response_time_ms":  hs.stats.MinResponseTimeMs,
		"max_response_time_ms":  hs.stats.MaxResponseTimeMs,
		"error_type_counts":     hs.stats.ErrorTypeCounts,
		"heuristic_usage":       hs.stats.HeuristicUsage,
	})
}

func (hs *HEARTService) getAvgLatency() float64 {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	if hs.stats.SuccessfulResponses > 0 {
		return hs.stats.TotalResponseTimeMs / float64(hs.stats.SuccessfulResponses)
	}
	return 0.0
}

func (hs *HEARTService) getSuccessRate() float64 {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	if hs.stats.TotalInquiries > 0 {
		return float64(hs.stats.SuccessfulResponses) / float64(hs.stats.TotalInquiries)
	}
	return 0.0
}

// Helper functions

func vectorMagnitude(vec []float32) float64 {
	sum := 0.0
	for _, v := range vec {
		sum += float64(v * v)
	}
	return sum
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
