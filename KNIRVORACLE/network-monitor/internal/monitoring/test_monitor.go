package monitoring

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// TestMonitor handles real-time test monitoring
type TestMonitor struct {
	logger       *logrus.Logger
	testSessions map[string]*TestSession
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	httpServer   *http.Server
}

// TestSession represents an active test session
type TestSession struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Status      string                 `json:"status"` // "running", "passed", "failed", "cancelled"
	TestCases   map[string]*TestCase   `json:"test_cases"`
	Metrics     map[string]interface{} `json:"metrics"`
	Environment string                 `json:"environment"`
	Component   string                 `json:"component"`
}

// TestCase represents an individual test case
type TestCase struct {
	Name       string                 `json:"name"`
	Status     string                 `json:"status"` // "pending", "running", "passed", "failed", "skipped"
	StartTime  time.Time              `json:"start_time"`
	EndTime    *time.Time             `json:"end_time,omitempty"`
	Duration   time.Duration          `json:"duration"`
	Error      string                 `json:"error,omitempty"`
	Assertions int                    `json:"assertions"`
	Coverage   float64                `json:"coverage"`
	Metrics    map[string]interface{} `json:"metrics"`
}

// TestMetrics represents aggregated test metrics
type TestMetrics struct {
	TotalTests      int           `json:"total_tests"`
	PassedTests     int           `json:"passed_tests"`
	FailedTests     int           `json:"failed_tests"`
	SkippedTests    int           `json:"skipped_tests"`
	TotalDuration   time.Duration `json:"total_duration"`
	AverageDuration time.Duration `json:"average_duration"`
	Coverage        float64       `json:"coverage"`
	LastUpdate      time.Time     `json:"last_update"`
}

// NewTestMonitor creates a new test monitor instance
func NewTestMonitor(logger *logrus.Logger) *TestMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	tm := &TestMonitor{
		logger:       logger,
		testSessions: make(map[string]*TestSession),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Setup HTTP server for test monitoring API
	tm.setupHTTPServer()

	return tm
}

// Start begins the test monitoring service
func (tm *TestMonitor) Start() error {
	tm.logger.Info("Starting test monitor...")

	// Start HTTP server in a goroutine
	go func() {
		if err := tm.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			tm.logger.WithError(err).Error("Test monitor HTTP server error")
		}
	}()

	// Start cleanup routine
	go tm.cleanupRoutine()

	tm.logger.Info("Test monitor started on :8082")
	return nil
}

// Stop stops the test monitoring service
func (tm *TestMonitor) Stop() error {
	tm.logger.Info("Stopping test monitor...")

	tm.cancel()

	if tm.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return tm.httpServer.Shutdown(ctx)
	}

	return nil
}

// StartTestSession starts a new test session
func (tm *TestMonitor) StartTestSession(id, name, component, environment string) *TestSession {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	session := &TestSession{
		ID:          id,
		Name:        name,
		StartTime:   time.Now(),
		Status:      "running",
		TestCases:   make(map[string]*TestCase),
		Metrics:     make(map[string]interface{}),
		Environment: environment,
		Component:   component,
	}

	tm.testSessions[id] = session
	tm.logger.WithFields(logrus.Fields{
		"session_id": id,
		"name":       name,
		"component":  component,
	}).Info("Started test session")

	return session
}

// EndTestSession ends a test session
func (tm *TestMonitor) EndTestSession(id string, status string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if session, exists := tm.testSessions[id]; exists {
		now := time.Now()
		session.EndTime = &now
		session.Status = status

		tm.logger.WithFields(logrus.Fields{
			"session_id": id,
			"status":     status,
			"duration":   now.Sub(session.StartTime),
		}).Info("Ended test session")
	}
}

// AddTestCase adds a test case to a session
func (tm *TestMonitor) AddTestCase(sessionID, testName string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if session, exists := tm.testSessions[sessionID]; exists {
		testCase := &TestCase{
			Name:      testName,
			Status:    "pending",
			StartTime: time.Now(),
			Metrics:   make(map[string]interface{}),
		}
		session.TestCases[testName] = testCase
	}
}

// UpdateTestCase updates a test case status
func (tm *TestMonitor) UpdateTestCase(sessionID, testName, status string, err error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if session, exists := tm.testSessions[sessionID]; exists {
		if testCase, exists := session.TestCases[testName]; exists {
			now := time.Now()
			testCase.Status = status
			testCase.EndTime = &now
			testCase.Duration = now.Sub(testCase.StartTime)

			if err != nil {
				testCase.Error = err.Error()
			}
		}
	}
}

// GetTestMetrics returns aggregated test metrics
func (tm *TestMonitor) GetTestMetrics() TestMetrics {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var totalTests, passedTests, failedTests, skippedTests int
	var totalDuration time.Duration
	var lastUpdate time.Time

	for _, session := range tm.testSessions {
		for _, testCase := range session.TestCases {
			totalTests++
			totalDuration += testCase.Duration

			switch testCase.Status {
			case "passed":
				passedTests++
			case "failed":
				failedTests++
			case "skipped":
				skippedTests++
			}

			if testCase.EndTime != nil && testCase.EndTime.After(lastUpdate) {
				lastUpdate = *testCase.EndTime
			}
		}
	}

	var avgDuration time.Duration
	if totalTests > 0 {
		avgDuration = totalDuration / time.Duration(totalTests)
	}

	return TestMetrics{
		TotalTests:      totalTests,
		PassedTests:     passedTests,
		FailedTests:     failedTests,
		SkippedTests:    skippedTests,
		TotalDuration:   totalDuration,
		AverageDuration: avgDuration,
		LastUpdate:      lastUpdate,
	}
}

// setupHTTPServer sets up the HTTP server for test monitoring API
func (tm *TestMonitor) setupHTTPServer() {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/test-sessions", tm.handleTestSessions)
	mux.HandleFunc("/api/test-metrics", tm.handleTestMetrics)
	mux.HandleFunc("/api/test-session/", tm.handleTestSession)

	// Static files for test monitoring dashboard
	mux.HandleFunc("/", tm.handleDashboard)

	tm.httpServer = &http.Server{
		Addr:    ":8082",
		Handler: mux,
	}
}

// HTTP handlers
func (tm *TestMonitor) handleTestSessions(w http.ResponseWriter, r *http.Request) {
	tm.mu.RLock()
	sessions := make([]*TestSession, 0, len(tm.testSessions))
	for _, session := range tm.testSessions {
		sessions = append(sessions, session)
	}
	tm.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (tm *TestMonitor) handleTestMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := tm.GetTestMetrics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (tm *TestMonitor) handleTestSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Path[len("/api/test-session/"):]

	tm.mu.RLock()
	session, exists := tm.testSessions[sessionID]
	tm.mu.RUnlock()

	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (tm *TestMonitor) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Simple dashboard HTML
	html := `<!DOCTYPE html>
<html>
<head>
    <title>KNIRV Test Monitor</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .metric { display: inline-block; margin: 10px; padding: 10px; border: 1px solid #ccc; }
        .session { margin: 10px 0; padding: 10px; border: 1px solid #ddd; }
        .passed { color: green; }
        .failed { color: red; }
        .running { color: orange; }
    </style>
</head>
<body>
    <h1>KNIRV Test Monitor</h1>
    <div id="metrics"></div>
    <div id="sessions"></div>
    <script>
        function updateDashboard() {
            fetch('/api/test-metrics')
                .then(r => r.json())
                .then(data => {
                    document.getElementById('metrics').innerHTML = 
                        '<div class="metric">Total: ' + data.total_tests + '</div>' +
                        '<div class="metric passed">Passed: ' + data.passed_tests + '</div>' +
                        '<div class="metric failed">Failed: ' + data.failed_tests + '</div>' +
                        '<div class="metric">Skipped: ' + data.skipped_tests + '</div>';
                });
            
            fetch('/api/test-sessions')
                .then(r => r.json())
                .then(data => {
                    let html = '<h2>Test Sessions</h2>';
                    data.forEach(session => {
                        html += '<div class="session ' + session.status + '">' +
                               '<h3>' + session.name + ' (' + session.status + ')</h3>' +
                               '<p>Component: ' + session.component + '</p>' +
                               '<p>Started: ' + new Date(session.start_time).toLocaleString() + '</p>' +
                               '</div>';
                    });
                    document.getElementById('sessions').innerHTML = html;
                });
        }
        
        updateDashboard();
        setInterval(updateDashboard, 5000);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// cleanupRoutine periodically cleans up old test sessions
func (tm *TestMonitor) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-tm.ctx.Done():
			return
		case <-ticker.C:
			tm.cleanupOldSessions()
		}
	}
}

// cleanupOldSessions removes test sessions older than 24 hours
func (tm *TestMonitor) cleanupOldSessions() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	for id, session := range tm.testSessions {
		if session.StartTime.Before(cutoff) {
			delete(tm.testSessions, id)
			tm.logger.WithField("session_id", id).Debug("Cleaned up old test session")
		}
	}
}
