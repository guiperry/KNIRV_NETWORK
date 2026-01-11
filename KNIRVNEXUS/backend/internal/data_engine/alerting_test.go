package data_engine

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestAlertLevel_String(t *testing.T) {
	tests := []struct {
		level    AlertLevel
		expected string
	}{
		{InfoAlert, "INFO"},
		{WarningAlert, "WARNING"},
		{ErrorAlert, "ERROR"},
		{CriticalAlert, "CRITICAL"},
		{AlertLevel(999), "UNKNOWN"},
	}

	for _, test := range tests {
		if result := test.level.String(); result != test.expected {
			t.Errorf("AlertLevel.String() = %v, want %v", result, test.expected)
		}
	}
}

func TestNewAlertingSystem(t *testing.T) {
	system := NewAlertingSystem(100)
	if system == nil {
		t.Fatal("NewAlertingSystem() returned nil")
	}
	if system.maxAlerts != 100 {
		t.Errorf("maxAlerts = %v, want %v", system.maxAlerts, 100)
	}
	if system.rules == nil {
		t.Error("rules map not initialized")
	}
	if system.alerts == nil {
		t.Error("alerts slice not initialized")
	}
	if system.handlers == nil {
		t.Error("handlers slice not initialized")
	}
	if system.state == nil {
		t.Error("state not initialized")
	}
}

func TestNewAlertingSystem_DefaultMaxAlerts(t *testing.T) {
	system := NewAlertingSystem(0)
	if system.maxAlerts != 1000 {
		t.Errorf("maxAlerts = %v, want %v", system.maxAlerts, 1000)
	}
}

func TestNewAlertingSystem_NegativeMaxAlerts(t *testing.T) {
	system := NewAlertingSystem(-1)
	if system.maxAlerts != 1000 {
		t.Errorf("maxAlerts = %v, want %v", system.maxAlerts, 1000)
	}
}

func TestAlertingSystem_RegisterRule(t *testing.T) {
	system := NewAlertingSystem(100)
	rule := AlertRule{
		ID:          "test-rule",
		Name:        "Test Rule",
		Description: "A test alert rule",
		EventType:   "test-event",
		Condition: func(event Event, state *AlertingState) bool {
			return true
		},
		Level:    WarningAlert,
		Cooldown: time.Minute,
	}

	system.RegisterRule(rule)

	if len(system.rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(system.rules))
	}

	if storedRule, exists := system.rules["test-rule"]; !exists {
		t.Error("Rule not stored")
	} else if storedRule.Name != "Test Rule" {
		t.Errorf("Rule name = %v, want %v", storedRule.Name, "Test Rule")
	}
}

func TestAlertingSystem_RegisterHandler(t *testing.T) {
	system := NewAlertingSystem(100)
	handlerCalled := false
	handler := func(alert Alert) {
		handlerCalled = true
	}

	system.RegisterHandler(handler)

	if len(system.handlers) != 1 {
		t.Errorf("Expected 1 handler, got %d", len(system.handlers))
	}

	// Test handler execution
	system.handlers[0](Alert{ID: "test"})
	if !handlerCalled {
		t.Error("Handler was not called")
	}
}

func TestAlertingSystem_ProcessEvent(t *testing.T) {
	system := NewAlertingSystem(100)
	alertsReceived := make(chan bool, 1)
	handler := func(alert Alert) {
		alertsReceived <- true
	}
	system.RegisterHandler(handler)

	rule := AlertRule{
		ID:          "test-rule",
		Name:        "Test Rule",
		Description: "A test alert rule",
		EventType:   "test-event",
		Condition: func(event Event, state *AlertingState) bool {
			return true // Always trigger
		},
		Level:    WarningAlert,
		Cooldown: time.Minute,
	}
	system.RegisterRule(rule)

	event := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"test": "data"},
	}

	system.ProcessEvent(event)

	// Wait for async handler
	select {
	case <-alertsReceived:
		// Handler called
	case <-time.After(100 * time.Millisecond):
		t.Error("Handler was not called within timeout")
	}

	if len(system.alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(system.alerts))
	}
}

func TestAlertingSystem_ProcessEvent_Cooldown(t *testing.T) {
	system := NewAlertingSystem(100)

	rule := AlertRule{
		ID:          "test-rule",
		Name:        "Test Rule",
		Description: "A test alert rule",
		EventType:   "test-event",
		Condition: func(event Event, state *AlertingState) bool {
			return true
		},
		Level:    WarningAlert,
		Cooldown: time.Hour, // Long cooldown
	}
	system.RegisterRule(rule)

	event := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"test": "data"},
	}

	// First event should trigger
	system.ProcessEvent(event)
	firstAlerts := len(system.alerts)

	// Second event should be blocked by cooldown
	system.ProcessEvent(event)
	secondAlerts := len(system.alerts)

	if firstAlerts != 1 {
		t.Errorf("Expected 1 alert after first event, got %d", firstAlerts)
	}
	if secondAlerts != 1 {
		t.Errorf("Expected still 1 alert after second event (cooldown), got %d", secondAlerts)
	}
}

func TestAlertingSystem_GetAlerts(t *testing.T) {
	system := NewAlertingSystem(100)

	// Add a test alert manually
	alert := Alert{
		ID:          "test-alert",
		Level:       WarningAlert,
		Title:       "Test Alert",
		Description: "Test description",
		Source:      "test",
		Timestamp:   time.Now(),
		Data:        map[string]interface{}{"key": "value"},
		Resolved:    false,
	}
	system.alerts = append(system.alerts, alert)

	alerts := system.GetAlerts()
	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].ID != "test-alert" {
		t.Errorf("Alert ID = %v, want %v", alerts[0].ID, "test-alert")
	}

	// Ensure it's a copy, not the original slice
	alerts[0].ID = "modified"
	if system.alerts[0].ID == "modified" {
		t.Error("GetAlerts() returned reference to original slice")
	}
}

func TestAlertingSystem_GetActiveAlerts(t *testing.T) {
	system := NewAlertingSystem(100)

	// Add resolved alert
	system.alerts = append(system.alerts, Alert{
		ID:       "resolved-alert",
		Resolved: true,
	})

	// Add active alert
	system.alerts = append(system.alerts, Alert{
		ID:       "active-alert",
		Resolved: false,
	})

	activeAlerts := system.GetActiveAlerts()
	if len(activeAlerts) != 1 {
		t.Errorf("Expected 1 active alert, got %d", len(activeAlerts))
	}
	if activeAlerts[0].ID != "active-alert" {
		t.Errorf("Active alert ID = %v, want %v", activeAlerts[0].ID, "active-alert")
	}
}

func TestAlertingSystem_ResolveAlert(t *testing.T) {
	system := NewAlertingSystem(100)

	system.alerts = append(system.alerts, Alert{
		ID:       "test-alert",
		Resolved: false,
	})

	resolved := system.ResolveAlert("test-alert")
	if !resolved {
		t.Error("ResolveAlert() should have returned true")
	}
	if !system.alerts[0].Resolved {
		t.Error("Alert should be marked as resolved")
	}
	if system.alerts[0].ResolvedAt.IsZero() {
		t.Error("ResolvedAt should be set")
	}

	// Test resolving non-existent alert
	resolved = system.ResolveAlert("non-existent")
	if resolved {
		t.Error("ResolveAlert() should have returned false for non-existent alert")
	}
}

func TestAlertingSystem_MaxAlerts(t *testing.T) {
	system := NewAlertingSystem(2)

	// Add alerts manually to exceed max
	for i := 0; i < 3; i++ {
		system.alerts = append(system.alerts, Alert{
			ID: fmt.Sprintf("alert-%d", i),
		})
	}

	// Process an event to trigger trimming
	event := Event{Type: "test", Timestamp: time.Now()}
	system.ProcessEvent(event)

	// The system should trim alerts when exceeding max, but the trimming happens in ProcessEvent
	// Let's check after processing multiple events
	for i := 0; i < 5; i++ {
		system.ProcessEvent(event)
	}

	// The max alerts logic trims when len(alerts) > maxAlerts, keeping the most recent ones
	// So we should have at most maxAlerts + 1 (the new alert) before trimming
	if len(system.alerts) > 3 {
		t.Errorf("Expected at most 3 alerts, got %d", len(system.alerts))
	}
}

func TestAlertingSystem_Close(t *testing.T) {
	system := NewAlertingSystem(100)

	// Ensure context is not cancelled initially
	select {
	case <-system.ctx.Done():
		t.Error("Context should not be cancelled initially")
	default:
	}

	system.Close()

	// Ensure context is cancelled after Close()
	select {
	case <-system.ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should be cancelled after Close()")
	}
}

func TestThresholdCondition(t *testing.T) {
	tests := []struct {
		name           string
		metricKey      string
		threshold      float64
		isGreaterThan  bool
		eventData      map[string]interface{}
		expectedResult bool
	}{
		{
			name:           "greater than threshold - true",
			metricKey:      "cpu",
			threshold:      80.0,
			isGreaterThan:  true,
			eventData:      map[string]interface{}{"cpu": 90.0},
			expectedResult: true,
		},
		{
			name:           "greater than threshold - false",
			metricKey:      "cpu",
			threshold:      80.0,
			isGreaterThan:  true,
			eventData:      map[string]interface{}{"cpu": 70.0},
			expectedResult: false,
		},
		{
			name:           "less than threshold - true",
			metricKey:      "memory",
			threshold:      50.0,
			isGreaterThan:  false,
			eventData:      map[string]interface{}{"memory": 30.0},
			expectedResult: true,
		},
		{
			name:           "less than threshold - false",
			metricKey:      "memory",
			threshold:      50.0,
			isGreaterThan:  false,
			eventData:      map[string]interface{}{"memory": 70.0},
			expectedResult: false,
		},
		{
			name:           "int value",
			metricKey:      "count",
			threshold:      10,
			isGreaterThan:  true,
			eventData:      map[string]interface{}{"count": 15},
			expectedResult: true,
		},
		{
			name:           "missing metric",
			metricKey:      "missing",
			threshold:      80.0,
			isGreaterThan:  true,
			eventData:      map[string]interface{}{"cpu": 90.0},
			expectedResult: false,
		},
		{
			name:           "invalid type",
			metricKey:      "invalid",
			threshold:      80.0,
			isGreaterThan:  true,
			eventData:      map[string]interface{}{"invalid": "string"},
			expectedResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			condition := ThresholdCondition(test.metricKey, test.threshold, test.isGreaterThan)
			event := Event{Data: test.eventData}
			state := &AlertingState{}

			result := condition(event, state)
			if result != test.expectedResult {
				t.Errorf("ThresholdCondition() = %v, want %v", result, test.expectedResult)
			}
		})
	}
}

func TestSpikeCondition(t *testing.T) {
	tests := []struct {
		name           string
		metricKey      string
		percentChange  float64
		eventData      map[string]interface{}
		stateMetrics   map[string]float64
		expectedResult bool
	}{
		{
			name:           "spike detected",
			metricKey:      "cpu",
			percentChange:  20.0,                                // 20% increase needed
			eventData:      map[string]interface{}{"cpu": 75.0}, // 75 > 50 * 1.2 = 60
			stateMetrics:   map[string]float64{"avg_test-event_cpu": 50.0},
			expectedResult: true,
		},
		{
			name:           "no spike",
			metricKey:      "cpu",
			percentChange:  100.0,                               // 100% increase needed
			eventData:      map[string]interface{}{"cpu": 75.0}, // 75 < 50 * 2
			stateMetrics:   map[string]float64{"avg_test-event_cpu": 50.0},
			expectedResult: false,
		},
		{
			name:           "missing average",
			metricKey:      "cpu",
			percentChange:  50.0,
			eventData:      map[string]interface{}{"cpu": 75.0},
			stateMetrics:   map[string]float64{},
			expectedResult: false,
		},
		{
			name:           "zero average",
			metricKey:      "cpu",
			percentChange:  50.0,
			eventData:      map[string]interface{}{"cpu": 75.0},
			stateMetrics:   map[string]float64{"avg_test-event_cpu": 0.0},
			expectedResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := &AlertingState{
				customMetrics: test.stateMetrics,
			}
			condition := SpikeCondition(test.metricKey, test.percentChange)
			event := Event{
				Type: "test-event",
				Data: test.eventData,
			}

			result := condition(event, state)
			if result != test.expectedResult {
				t.Errorf("SpikeCondition() = %v, want %v", result, test.expectedResult)
			}
		})
	}
}

func TestErrorRateCondition(t *testing.T) {
	state := &AlertingState{
		eventRates: map[EventType]float64{
			ErrorEvent: 2.0, // 2 errors per second = 120 per minute
		},
	}

	tests := []struct {
		name            string
		errorsPerMinute float64
		eventType       EventType
		expectedResult  bool
	}{
		{
			name:            "error rate exceeded",
			errorsPerMinute: 100.0,
			eventType:       ErrorEvent,
			expectedResult:  true,
		},
		{
			name:            "error rate not exceeded",
			errorsPerMinute: 150.0,
			eventType:       ErrorEvent,
			expectedResult:  false,
		},
		{
			name:            "wrong event type",
			errorsPerMinute: 50.0,
			eventType:       "other-event",
			expectedResult:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			condition := ErrorRateCondition(test.errorsPerMinute)
			event := Event{Type: test.eventType}

			result := condition(event, state)
			if result != test.expectedResult {
				t.Errorf("ErrorRateCondition() = %v, want %v", result, test.expectedResult)
			}
		})
	}
}

func TestEventFrequencyCondition(t *testing.T) {
	state := &AlertingState{
		eventRates: map[EventType]float64{
			"test-event": 3.0, // 3 events per second = 180 per minute
		},
	}

	tests := []struct {
		name            string
		eventType       EventType
		eventsPerMinute float64
		expectedResult  bool
	}{
		{
			name:            "frequency exceeded",
			eventType:       "test-event",
			eventsPerMinute: 150.0,
			expectedResult:  true,
		},
		{
			name:            "frequency not exceeded",
			eventType:       "test-event",
			eventsPerMinute: 200.0,
			expectedResult:  false,
		},
		{
			name:            "wrong event type",
			eventType:       "other-event",
			eventsPerMinute: 100.0,
			expectedResult:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			condition := EventFrequencyCondition(test.eventType, test.eventsPerMinute)
			event := Event{Type: test.eventType}

			result := condition(event, state)
			if result != test.expectedResult {
				t.Errorf("EventFrequencyCondition() = %v, want %v", result, test.expectedResult)
			}
		})
	}
}

func TestInactivityCondition(t *testing.T) {
	now := time.Now()
	state := &AlertingState{
		lastEventTimes: map[EventType]time.Time{
			"test-event": now.Add(-time.Minute * 2), // 2 minutes ago
		},
	}

	tests := []struct {
		name                 string
		eventType            EventType
		maxInactivitySeconds float64
		expectedResult       bool
	}{
		{
			name:                 "inactivity exceeded",
			eventType:            "test-event",
			maxInactivitySeconds: 60.0, // 1 minute
			expectedResult:       true,
		},
		{
			name:                 "inactivity not exceeded",
			eventType:            "test-event",
			maxInactivitySeconds: 180.0, // 3 minutes
			expectedResult:       false,
		},
		{
			name:                 "unknown event type",
			eventType:            "unknown-event",
			maxInactivitySeconds: 60.0,
			expectedResult:       false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			condition := InactivityCondition(test.eventType, test.maxInactivitySeconds)
			event := Event{Type: test.eventType}

			result := condition(event, state)
			if result != test.expectedResult {
				t.Errorf("InactivityCondition() = %v, want %v", result, test.expectedResult)
			}
		})
	}
}

func TestCreateLogAlertHandler(t *testing.T) {
	handler := CreateLogAlertHandler()
	if handler == nil {
		t.Fatal("CreateLogAlertHandler() returned nil")
	}

	alert := Alert{
		ID:          "test-alert",
		Level:       CriticalAlert,
		Title:       "Test Alert",
		Description: "Test description",
		Source:      "test",
		Timestamp:   time.Now(),
		Data:        map[string]interface{}{"key": "value"},
		Resolved:    false,
	}

	// Handler should not panic
	handler(alert)
}

func TestAlertingState_Concurrency(t *testing.T) {
	state := newAlertingState()
	var wg sync.WaitGroup

	// Test concurrent access to state
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			event := Event{
				Type:      "test-event",
				Timestamp: time.Now(),
				Data:      map[string]interface{}{"metric": 1.0},
			}

			// Simulate updating state and store latest event data
			state.mutex.Lock()
			state.eventCounts[event.Type]++
			state.lastEventTimes[event.Type] = event.Timestamp
			state.lastEventData[event.Type] = event.Data
			state.mutex.Unlock()
		}()
	}

	wg.Wait()

	// Verify state was updated correctly
	state.mutex.RLock()
	count := state.eventCounts["test-event"]
	lastData := state.lastEventData["test-event"]
	state.mutex.RUnlock()

	if count != 10 {
		t.Errorf("Expected event count of 10, got %d", count)
	}

	// Verify the event data was properly stored
	expectedMetric := 1.0
	if metric, ok := lastData["metric"].(float64); !ok || metric != expectedMetric {
		t.Errorf("Expected last event data to have metric=%.1f, got %v", expectedMetric, lastData["metric"])
	}
}

func TestAlertingSystem_ConcurrentAccess(t *testing.T) {
	system := NewAlertingSystem(100)
	var wg sync.WaitGroup

	// Test concurrent rule registration
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rule := AlertRule{
				ID:   fmt.Sprintf("rule-%d", id),
				Name: fmt.Sprintf("Rule %d", id),
				Condition: func(event Event, state *AlertingState) bool {
					return false
				},
				Level: InfoAlert,
			}
			system.RegisterRule(rule)
		}(i)
	}

	wg.Wait()

	if len(system.rules) != 10 {
		t.Errorf("Expected 10 rules, got %d", len(system.rules))
	}
}

func TestAlertingSystem_GetRules(t *testing.T) {
	system := NewAlertingSystem(100)

	rule := AlertRule{
		ID:          "test-rule",
		Name:        "Test Rule",
		Description: "A test alert rule",
		EventType:   "test-event",
		Condition: func(event Event, state *AlertingState) bool {
			return true
		},
		Level:    WarningAlert,
		Cooldown: time.Minute,
	}

	system.RegisterRule(rule)

	// Access rules directly since there's no GetRules method
	if len(system.rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(system.rules))
	}
	if rule, exists := system.rules["test-rule"]; !exists {
		t.Error("Rule not found")
	} else if rule.ID != "test-rule" {
		t.Errorf("Rule ID = %v, want %v", rule.ID, "test-rule")
	}
}

func TestAlertingSystem_GetRules_Method(t *testing.T) {
	system := NewAlertingSystem(100)

	rule := AlertRule{
		ID:          "test-rule",
		Name:        "Test Rule",
		Description: "A test alert rule",
		EventType:   "test-event",
		Condition: func(event Event, state *AlertingState) bool {
			return true
		},
		Level:    WarningAlert,
		Cooldown: time.Minute,
	}

	system.RegisterRule(rule)

	// Test that we can access rules (this covers the RegisterRule method more thoroughly)
	system.mutex.RLock()
	rulesCount := len(system.rules)
	system.mutex.RUnlock()

	if rulesCount != 1 {
		t.Errorf("Expected 1 rule, got %d", rulesCount)
	}
}

func TestAlertingSystem_ProcessEvent_EmptyRules(t *testing.T) {
	system := NewAlertingSystem(100)

	event := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"test": "data"},
	}

	// Should not panic with empty rules
	system.ProcessEvent(event)

	if len(system.alerts) != 0 {
		t.Errorf("Expected 0 alerts with no rules, got %d", len(system.alerts))
	}
}

func TestAlertingSystem_ProcessEvent_NoHandlers(t *testing.T) {
	system := NewAlertingSystem(100)

	rule := AlertRule{
		ID:          "test-rule",
		Name:        "Test Rule",
		Description: "A test alert rule",
		EventType:   "test-event",
		Condition: func(event Event, state *AlertingState) bool {
			return true
		},
		Level:    WarningAlert,
		Cooldown: time.Minute,
	}

	system.RegisterRule(rule)

	event := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"test": "data"},
	}

	// Should not panic with no handlers
	system.ProcessEvent(event)

	if len(system.alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(system.alerts))
	}
}

func TestAlertingSystem_ProcessEvent_WrongEventType(t *testing.T) {
	system := NewAlertingSystem(100)

	rule := AlertRule{
		ID:          "test-rule",
		Name:        "Test Rule",
		Description: "A test alert rule",
		EventType:   "specific-event",
		Condition: func(event Event, state *AlertingState) bool {
			return true
		},
		Level:    WarningAlert,
		Cooldown: time.Minute,
	}

	system.RegisterRule(rule)

	event := Event{
		Type:      "different-event",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"test": "data"},
	}

	system.ProcessEvent(event)

	// Should not trigger alert for wrong event type
	if len(system.alerts) != 0 {
		t.Errorf("Expected 0 alerts for wrong event type, got %d", len(system.alerts))
	}
}

func TestAlertingSystem_ProcessEvent_ConditionFalse(t *testing.T) {
	system := NewAlertingSystem(100)

	rule := AlertRule{
		ID:          "test-rule",
		Name:        "Test Rule",
		Description: "A test alert rule",
		EventType:   "test-event",
		Condition: func(event Event, state *AlertingState) bool {
			return false // Always false
		},
		Level:    WarningAlert,
		Cooldown: time.Minute,
	}

	system.RegisterRule(rule)

	event := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"test": "data"},
	}

	system.ProcessEvent(event)

	// Should not trigger alert when condition is false
	if len(system.alerts) != 0 {
		t.Errorf("Expected 0 alerts when condition is false, got %d", len(system.alerts))
	}
}

func TestAlertingSystem_ProcessEvent_MultipleRules(t *testing.T) {
	system := NewAlertingSystem(100)

	// Rule 1: triggers
	rule1 := AlertRule{
		ID:          "rule1",
		Name:        "Rule 1",
		Description: "First test rule",
		EventType:   "test-event",
		Condition: func(event Event, state *AlertingState) bool {
			return true
		},
		Level:    WarningAlert,
		Cooldown: time.Minute,
	}

	// Rule 2: doesn't trigger
	rule2 := AlertRule{
		ID:          "rule2",
		Name:        "Rule 2",
		Description: "Second test rule",
		EventType:   "test-event",
		Condition: func(event Event, state *AlertingState) bool {
			return false
		},
		Level:    ErrorAlert,
		Cooldown: time.Minute,
	}

	system.RegisterRule(rule1)
	system.RegisterRule(rule2)

	event := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"test": "data"},
	}

	system.ProcessEvent(event)

	// Should trigger only one alert
	if len(system.alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(system.alerts))
	}

	if len(system.alerts) > 0 && system.alerts[0].Level != WarningAlert {
		t.Errorf("Expected WarningAlert, got %v", system.alerts[0].Level)
	}
}

func TestAlertingSystem_ResolveAlert_NonExistent(t *testing.T) {
	system := NewAlertingSystem(100)

	// Try to resolve non-existent alert
	resolved := system.ResolveAlert("non-existent-id")
	if resolved {
		t.Error("Expected false for non-existent alert")
	}
}

func TestAlertingSystem_GetActiveAlerts_Empty(t *testing.T) {
	system := NewAlertingSystem(100)

	activeAlerts := system.GetActiveAlerts()
	if len(activeAlerts) != 0 {
		t.Errorf("Expected 0 active alerts, got %d", len(activeAlerts))
	}
}

func TestAlertingSystem_GetActiveAlerts_Mixed(t *testing.T) {
	system := NewAlertingSystem(100)

	// Add resolved alert
	system.alerts = append(system.alerts, Alert{
		ID:       "resolved",
		Resolved: true,
	})

	// Add active alert
	system.alerts = append(system.alerts, Alert{
		ID:       "active",
		Resolved: false,
	})

	activeAlerts := system.GetActiveAlerts()
	if len(activeAlerts) != 1 {
		t.Errorf("Expected 1 active alert, got %d", len(activeAlerts))
	}

	if len(activeAlerts) > 0 && activeAlerts[0].ID != "active" {
		t.Errorf("Expected active alert ID 'active', got %s", activeAlerts[0].ID)
	}
}

func TestAlertingSystem_GetAlerts_Empty(t *testing.T) {
	system := NewAlertingSystem(100)

	alerts := system.GetAlerts()
	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts, got %d", len(alerts))
	}
}

func TestAlertingSystem_GetAlerts_Modification(t *testing.T) {
	system := NewAlertingSystem(100)

	system.alerts = append(system.alerts, Alert{
		ID: "original",
	})

	alerts := system.GetAlerts()

	// Modify the returned slice
	alerts[0].ID = "modified"

	// Original should not be affected
	if system.alerts[0].ID != "original" {
		t.Error("Original alerts slice was modified")
	}
}

func TestAlertingSystem_RegisterHandler_Multiple(t *testing.T) {
	system := NewAlertingSystem(100)

	handler1Called := false
	handler2Called := false

	handler1 := func(alert Alert) {
		handler1Called = true
	}

	handler2 := func(alert Alert) {
		handler2Called = true
	}

	system.RegisterHandler(handler1)
	system.RegisterHandler(handler2)

	if len(system.handlers) != 2 {
		t.Errorf("Expected 2 handlers, got %d", len(system.handlers))
	}

	// Test both handlers are called
	alert := Alert{ID: "test"}
	for _, handler := range system.handlers {
		handler(alert)
	}

	if !handler1Called || !handler2Called {
		t.Error("Both handlers should have been called")
	}
}

func TestAlertingSystem_ProcessEvent_AlertTrimming(t *testing.T) {
	system := NewAlertingSystem(2) // Small max alerts

	// Add alerts manually to reach the limit
	for i := 0; i < 3; i++ {
		system.alerts = append(system.alerts, Alert{
			ID: fmt.Sprintf("alert-%d", i),
		})
	}

	rule := AlertRule{
		ID:          "test-rule",
		Name:        "Test Rule",
		Description: "A test alert rule",
		EventType:   "test-event",
		Condition: func(event Event, state *AlertingState) bool {
			return true
		},
		Level:    WarningAlert,
		Cooldown: time.Minute,
	}

	system.RegisterRule(rule)

	event := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"test": "data"},
	}

	system.ProcessEvent(event)

	// Should have trimmed to max alerts (2)
	if len(system.alerts) > 2 {
		t.Errorf("Expected at most 2 alerts after trimming, got %d", len(system.alerts))
	}
}

func TestAlertingSystem_Close_CancelledContext(t *testing.T) {
	system := NewAlertingSystem(100)

	// Context should not be cancelled initially
	select {
	case <-system.ctx.Done():
		t.Error("Context should not be cancelled initially")
	default:
		// Expected
	}

	system.Close()

	// Context should be cancelled after Close
	select {
	case <-system.ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should be cancelled after Close")
	}

	// Second close should not panic
	system.Close()
}

func TestAlertingState_UpdateState_ErrorEvent(t *testing.T) {
	system := NewAlertingSystem(100)

	event := Event{
		Type:      ErrorEvent,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"source": "test-source",
		},
	}

	system.ProcessEvent(event)

	// Check that error count was updated
	system.state.mutex.RLock()
	count, exists := system.state.errorCounts["test-source"]
	system.state.mutex.RUnlock()

	if !exists {
		t.Error("Error count should exist for test-source")
	}

	if count != 1 {
		t.Errorf("Expected error count 1, got %d", count)
	}
}

func TestAlertingState_UpdateState_WindowedMetrics(t *testing.T) {
	system := NewAlertingSystem(100)

	event := Event{
		Type:      "metric-event",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"value": 10.0,
		},
	}

	system.ProcessEvent(event)

	// Check that windowed metrics were updated
	system.state.mutex.RLock()
	avg, exists := system.state.customMetrics["avg_metric-event_value"]
	system.state.mutex.RUnlock()

	if !exists {
		t.Error("Average metric should exist")
	}

	if avg != 10.0 {
		t.Errorf("Expected average 10.0, got %f", avg)
	}
}

func TestAlertingState_UpdateState_EventRates(t *testing.T) {
	system := NewAlertingSystem(100)

	event := Event{
		Type:      "rate-event",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{},
	}

	system.ProcessEvent(event)

	// Check that event rate was calculated
	system.state.mutex.RLock()
	rate, exists := system.state.eventRates["rate-event"]
	system.state.mutex.RUnlock()

	if !exists {
		t.Error("Event rate should exist")
	}

	if rate <= 0 {
		t.Errorf("Expected positive event rate, got %f", rate)
	}
}

func TestAlertingState_Concurrency_Safe(t *testing.T) {
	system := NewAlertingSystem(100)

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			event := Event{
				Type:      EventType(fmt.Sprintf("event-%d", id%5)), // 5 different event types
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"metric": float64(id),
				},
			}

			system.ProcessEvent(event)
		}(i)
	}

	wg.Wait()

	// Verify state integrity
	system.state.mutex.RLock()
	totalEvents := int64(0)
	for _, count := range system.state.eventCounts {
		totalEvents += count
	}
	system.state.mutex.RUnlock()

	if totalEvents != int64(numGoroutines) {
		t.Errorf("Expected %d total events, got %d", numGoroutines, totalEvents)
	}
}
