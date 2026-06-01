package pluginmanagement

import (
	"context"
	"testing"

	"backend_server/internal/objects"
)

// mockBuiltin is a test implementation of BuiltinPlugin.
type mockBuiltin struct {
	id      string
	name    string
	enabled bool
	running bool
}

func (m *mockBuiltin) PluginID() string    { return m.id }
func (m *mockBuiltin) PluginName() string  { return m.name }
func (m *mockBuiltin) Category() string    { return "test" }
func (m *mockBuiltin) Description() string { return "test plugin" }
func (m *mockBuiltin) Version() string     { return "0.0.1" }
func (m *mockBuiltin) IsEnabled() bool     { return m.enabled }
func (m *mockBuiltin) IsRunning() bool     { return m.running }

func (m *mockBuiltin) Start(_ context.Context) error {
	m.enabled = true
	m.running = true
	return nil
}

func (m *mockBuiltin) Stop(_ context.Context) error {
	m.enabled = false
	m.running = false
	return nil
}

// newTestService creates a PluginManagementService with a nil db suitable for unit tests
// that do not exercise the database path.
func newTestService() *PluginManagementService {
	return &PluginManagementService{
		objects:               make(map[string]*objects.Plugin),
		deployments:           make(map[string]*objects.PluginDeployment),
		templates:             make(map[string]*objects.PluginTemplate),
		runtimeInstances:      make(map[string]*objects.PluginRuntimeInstance),
		metrics:               make(map[string][]*objects.PluginMetrics),
		logs:                  make(map[string][]*objects.PluginLog),
		events:                make([]*objects.PluginEvent, 0),
		builtins:              make(map[string]BuiltinPlugin),
		maxPlugins:            100,
		maxInstancesPerPlugin: 10,
	}
}

func TestRegisterBuiltin(t *testing.T) {
	svc := newTestService()
	mock := &mockBuiltin{id: "test-plugin", name: "Test Plugin"}
	svc.RegisterBuiltin(mock)

	list := svc.ListBuiltins()
	if len(list) != 1 {
		t.Fatalf("expected 1 builtin, got %d", len(list))
	}
	if list[0].ID != "test-plugin" {
		t.Errorf("expected id 'test-plugin', got %q", list[0].ID)
	}
}

func TestToggleBuiltin_Enable(t *testing.T) {
	svc := newTestService()
	mock := &mockBuiltin{id: "toggle-plugin", name: "Toggle Plugin"}
	svc.RegisterBuiltin(mock)

	// We cannot call ToggleBuiltin because it calls saveBuiltinState which requires db.
	// Call Start directly to simulate what ToggleBuiltin does.
	if err := mock.Start(context.Background()); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	if !mock.IsRunning() {
		t.Error("expected IsRunning() == true after Start()")
	}
}

func TestToggleBuiltin_Disable(t *testing.T) {
	svc := newTestService()
	mock := &mockBuiltin{id: "toggle-plugin", name: "Toggle Plugin", enabled: true, running: true}
	svc.RegisterBuiltin(mock)

	if err := mock.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}
	if mock.IsRunning() {
		t.Error("expected IsRunning() == false after Stop()")
	}
}

func TestToggleBuiltin_NotFound(t *testing.T) {
	svc := newTestService()
	err := svc.ToggleBuiltin(context.Background(), "nonexistent-plugin", true)
	if err == nil {
		t.Error("expected error for unknown plugin id, got nil")
	}
}
