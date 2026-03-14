package data_engine

import (
	"fmt"
	"testing"
	"time"
)

func TestNewWindowedAggregator(t *testing.T) {
	aggregator := NewWindowedAggregator(TumblingWindow, time.Minute)
	if aggregator == nil {
		t.Fatal("NewWindowedAggregator() returned nil")
	}
	if aggregator.windowType != TumblingWindow {
		t.Errorf("windowType = %v, want %v", aggregator.windowType, TumblingWindow)
	}
	if aggregator.windowSize != time.Minute {
		t.Errorf("windowSize = %v, want %v", aggregator.windowSize, time.Minute)
	}
	if aggregator.windows == nil {
		t.Error("windows map not initialized")
	}
	if aggregator.aggregators == nil {
		t.Error("aggregators map not initialized")
	}
}

func TestWindowedAggregator_SetSlideSize(t *testing.T) {
	aggregator := NewWindowedAggregator(SlidingWindow, time.Minute)
	newSlideSize := time.Second * 30

	aggregator.SetSlideSize(newSlideSize)
	if aggregator.slideSize != newSlideSize {
		t.Errorf("slideSize = %v, want %v", aggregator.slideSize, newSlideSize)
	}
}

func TestWindowedAggregator_SetSessionGap(t *testing.T) {
	aggregator := NewWindowedAggregator(SessionWindow, time.Minute)
	newSessionGap := time.Minute * 10

	aggregator.SetSessionGap(newSessionGap)
	if aggregator.sessionGap != newSessionGap {
		t.Errorf("sessionGap = %v, want %v", aggregator.sessionGap, newSessionGap)
	}
}

func TestWindowedAggregator_RegisterAggregator(t *testing.T) {
	aggregator := NewWindowedAggregator(TumblingWindow, time.Minute)

	testAggregator := func(window *WindowData, event Event) error {
		return nil
	}

	aggregator.RegisterAggregator("test", testAggregator)

	if fn, exists := aggregator.aggregators["test"]; !exists {
		t.Error("Aggregator not registered")
	} else if fn == nil {
		t.Error("Aggregator function is nil")
	}
}

func TestWindowedAggregator_ProcessEvent_TumblingWindow(t *testing.T) {
	aggregator := NewWindowedAggregator(TumblingWindow, time.Minute)

	event := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Source:    "test",
		Data: map[string]interface{}{
			"value": 42.0,
		},
	}

	err := aggregator.ProcessEvent(event)
	if err != nil {
		t.Fatalf("ProcessEvent() failed: %v", err)
	}

	windows := aggregator.GetWindows()
	if len(windows) != 1 {
		t.Errorf("Expected 1 window, got %d", len(windows))
	}

	window := windows[0]
	if window.EventCount != 1 {
		t.Errorf("EventCount = %v, want %v", window.EventCount, 1)
	}
}

func TestWindowedAggregator_ProcessEvent_SlidingWindow(t *testing.T) {
	aggregator := NewWindowedAggregator(SlidingWindow, time.Minute)
	aggregator.SetSlideSize(time.Second * 30)

	event := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Source:    "test",
		Data: map[string]interface{}{
			"value": 42.0,
		},
	}

	err := aggregator.ProcessEvent(event)
	if err != nil {
		t.Fatalf("ProcessEvent() failed: %v", err)
	}

	windows := aggregator.GetWindows()
	if len(windows) == 0 {
		t.Error("Expected at least 1 window")
	}
}

func TestWindowedAggregator_ProcessEvent_SessionWindow(t *testing.T) {
	aggregator := NewWindowedAggregator(SessionWindow, time.Minute)
	aggregator.SetSessionGap(time.Minute * 5)

	event := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Source:    "test",
		Data: map[string]interface{}{
			"value": 42.0,
		},
	}

	err := aggregator.ProcessEvent(event)
	if err != nil {
		t.Fatalf("ProcessEvent() failed: %v", err)
	}

	windows := aggregator.GetWindows()
	if len(windows) != 1 {
		t.Errorf("Expected 1 window, got %d", len(windows))
	}

	window := windows[0]
	if window.Window.Type != SessionWindow {
		t.Errorf("Window type = %v, want %v", window.Window.Type, SessionWindow)
	}
}

func TestWindowedAggregator_GetWindows(t *testing.T) {
	aggregator := NewWindowedAggregator(TumblingWindow, time.Minute)

	event := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Source:    "test",
		Data: map[string]interface{}{
			"value": 42.0,
		},
	}

	aggregator.ProcessEvent(event)

	windows := aggregator.GetWindows()
	if len(windows) != 1 {
		t.Errorf("Expected 1 window, got %d", len(windows))
	}
}

func TestWindowedAggregator_GetWindowsInRange(t *testing.T) {
	aggregator := NewWindowedAggregator(TumblingWindow, time.Minute)

	now := time.Now()
	event := Event{
		Type:      "test-event",
		Timestamp: now,
		Source:    "test",
		Data: map[string]interface{}{
			"value": 42.0,
		},
	}

	aggregator.ProcessEvent(event)

	start := now.Add(-time.Minute)
	end := now.Add(time.Minute)

	windows := aggregator.GetWindowsInRange(start, end)
	if len(windows) != 1 {
		t.Errorf("Expected 1 window, got %d", len(windows))
	}

	// Test range that doesn't include the window
	start = now.Add(-2 * time.Minute)
	end = now.Add(-time.Minute)

	windows = aggregator.GetWindowsInRange(start, end)
	if len(windows) != 0 {
		t.Errorf("Expected 0 windows, got %d", len(windows))
	}
}

func TestWindowedAggregator_GetActiveUsers(t *testing.T) {
	aggregator := NewWindowedAggregator(SessionWindow, time.Minute)

	now := time.Now()
	event := Event{
		Type:      "test-event",
		Timestamp: now,
		Source:    "test",
		Data: map[string]interface{}{
			"value": 42.0,
		},
	}

	aggregator.ProcessEvent(event)

	start := now.Add(-time.Minute)
	end := now.Add(time.Minute)

	activeUsers := aggregator.GetActiveUsers(start, end)
	// Session windows create a user_id based on timestamp, so there should be 1 active user
	if activeUsers != 1 {
		t.Errorf("Expected 1 active user, got %d", activeUsers)
	}
}

func TestWindowedAggregator_GetEventRate(t *testing.T) {
	aggregator := NewWindowedAggregator(TumblingWindow, time.Minute)

	now := time.Now()
	event := Event{
		Type:      "test-event",
		Timestamp: now,
		Source:    "test",
		Data: map[string]interface{}{
			"value": 42.0,
		},
	}

	aggregator.ProcessEvent(event)

	start := now.Add(-time.Minute)
	end := now.Add(time.Minute)

	rate := aggregator.GetEventRate(start, end)
	if rate <= 0 {
		t.Errorf("Expected positive event rate, got %v", rate)
	}
}

func TestCountAggregator(t *testing.T) {
	window := &WindowData{
		Data: make(map[string]interface{}),
	}

	event := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"value": 42.0,
		},
	}

	err := countAggregator(window, event)
	if err != nil {
		t.Fatalf("countAggregator() failed: %v", err)
	}

	if count, ok := window.Data["count"].(int64); !ok || count != 1 {
		t.Errorf("count = %v, want %v", count, 1)
	}

	// Test multiple events
	err = countAggregator(window, event)
	if err != nil {
		t.Fatalf("countAggregator() failed: %v", err)
	}

	if count, ok := window.Data["count"].(int64); !ok || count != 2 {
		t.Errorf("count = %v, want %v", count, 2)
	}
}

func TestSumAggregator(t *testing.T) {
	window := &WindowData{
		Data: make(map[string]interface{}),
	}

	event := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"value": 42.0,
			"other": 10.0,
		},
	}

	err := sumAggregator(window, event)
	if err != nil {
		t.Fatalf("sumAggregator() failed: %v", err)
	}

	if sum, ok := window.Data["sum_value"].(float64); !ok || sum != 42.0 {
		t.Errorf("sum_value = %v, want %v", sum, 42.0)
	}

	if sum, ok := window.Data["sum_other"].(float64); !ok || sum != 10.0 {
		t.Errorf("sum_other = %v, want %v", sum, 10.0)
	}

	// Test multiple events
	event2 := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"value": 8.0,
		},
	}

	err = sumAggregator(window, event2)
	if err != nil {
		t.Fatalf("sumAggregator() failed: %v", err)
	}

	if sum, ok := window.Data["sum_value"].(float64); !ok || sum != 50.0 {
		t.Errorf("sum_value = %v, want %v", sum, 50.0)
	}
}

func TestAvgAggregator(t *testing.T) {
	window := &WindowData{
		Data: make(map[string]interface{}),
	}

	event := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"value": 40.0,
		},
	}

	err := avgAggregator(window, event)
	if err != nil {
		t.Fatalf("avgAggregator() failed: %v", err)
	}

	if avg, ok := window.Data["avg_value"].(float64); !ok || avg != 40.0 {
		t.Errorf("avg_value = %v, want %v", avg, 40.0)
	}

	// Test multiple events
	event2 := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"value": 60.0,
		},
	}

	err = avgAggregator(window, event2)
	if err != nil {
		t.Fatalf("avgAggregator() failed: %v", err)
	}

	if avg, ok := window.Data["avg_value"].(float64); !ok || avg != 50.0 {
		t.Errorf("avg_value = %v, want %v", avg, 50.0)
	}
}

func TestMinAggregator(t *testing.T) {
	window := &WindowData{
		Data: make(map[string]interface{}),
	}

	event := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"value": 50.0,
		},
	}

	err := minAggregator(window, event)
	if err != nil {
		t.Fatalf("minAggregator() failed: %v", err)
	}

	if min, ok := window.Data["min_value"].(float64); !ok || min != 50.0 {
		t.Errorf("min_value = %v, want %v", min, 50.0)
	}

	// Test smaller value
	event2 := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"value": 30.0,
		},
	}

	err = minAggregator(window, event2)
	if err != nil {
		t.Fatalf("minAggregator() failed: %v", err)
	}

	if min, ok := window.Data["min_value"].(float64); !ok || min != 30.0 {
		t.Errorf("min_value = %v, want %v", min, 30.0)
	}

	// Test larger value (should not change min)
	event3 := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"value": 70.0,
		},
	}

	err = minAggregator(window, event3)
	if err != nil {
		t.Fatalf("minAggregator() failed: %v", err)
	}

	if min, ok := window.Data["min_value"].(float64); !ok || min != 30.0 {
		t.Errorf("min_value = %v, want %v", min, 30.0)
	}
}

func TestMaxAggregator(t *testing.T) {
	window := &WindowData{
		Data: make(map[string]interface{}),
	}

	event := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"value": 50.0,
		},
	}

	err := maxAggregator(window, event)
	if err != nil {
		t.Fatalf("maxAggregator() failed: %v", err)
	}

	if max, ok := window.Data["max_value"].(float64); !ok || max != 50.0 {
		t.Errorf("max_value = %v, want %v", max, 50.0)
	}

	// Test larger value
	event2 := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"value": 70.0,
		},
	}

	err = maxAggregator(window, event2)
	if err != nil {
		t.Fatalf("maxAggregator() failed: %v", err)
	}

	if max, ok := window.Data["max_value"].(float64); !ok || max != 70.0 {
		t.Errorf("max_value = %v, want %v", max, 70.0)
	}

	// Test smaller value (should not change max)
	event3 := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"value": 30.0,
		},
	}

	err = maxAggregator(window, event3)
	if err != nil {
		t.Fatalf("maxAggregator() failed: %v", err)
	}

	if max, ok := window.Data["max_value"].(float64); !ok || max != 70.0 {
		t.Errorf("max_value = %v, want %v", max, 70.0)
	}
}

func TestWindowedAggregator_CleanupWindows(t *testing.T) {
	aggregator := NewWindowedAggregator(TumblingWindow, time.Minute)

	// Create an old window manually
	oldTime := time.Now().Add(-2 * time.Hour)
	windowKey := fmt.Sprintf("%s-%s", oldTime.Format(time.RFC3339), oldTime.Add(time.Minute).Format(time.RFC3339))

	aggregator.windows[windowKey] = &WindowData{
		Window: Window{
			StartTime: oldTime,
			EndTime:   oldTime.Add(time.Minute),
			Type:      TumblingWindow,
			Size:      time.Minute,
		},
		Data:       make(map[string]interface{}),
		EventCount: 1,
		LastUpdate: oldTime,
	}

	// Process a new event to trigger cleanup
	event := Event{
		Type:      "test-event",
		Timestamp: time.Now(),
		Source:    "test",
		Data: map[string]interface{}{
			"value": 42.0,
		},
	}

	err := aggregator.ProcessEvent(event)
	if err != nil {
		t.Fatalf("ProcessEvent() failed: %v", err)
	}

	// Check that old window was cleaned up
	if _, exists := aggregator.windows[windowKey]; exists {
		t.Error("Old window was not cleaned up")
	}

	// Check that new window exists
	windows := aggregator.GetWindows()
	if len(windows) != 1 {
		t.Errorf("Expected 1 window, got %d", len(windows))
	}
}
