package inner_test

import (
	"testing"

	"github.com/knirvcorp/knirvagent/pkg/inner"
)

// TestNewInnerAgentManager verifies initialization creates an empty session map
// with the default catalog.
func TestNewInnerAgentManager(t *testing.T) {
	mgr := inner.NewInnerAgentManager("dve-test", "/tmp/test-workspace")

	if mgr == nil {
		t.Fatal("NewInnerAgentManager returned nil")
	}

	sessions := mgr.ListSessions()
	if sessions == nil {
		t.Fatal("ListSessions returned nil")
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}

	tools := mgr.ListTools()
	if tools == nil {
		t.Fatal("ListTools returned nil")
	}
	if len(tools) != 4 {
		t.Errorf("expected 4 tools, got %d", len(tools))
	}
}

// TestManager_GetSession_NotFound verifies that GetSession returns an error for
// an unknown session ID.
func TestManager_GetSession_NotFound(t *testing.T) {
	mgr := inner.NewInnerAgentManager("dve-test", "/tmp/test-workspace")

	tests := []struct {
		name string
		id   string
	}{
		{name: "empty string", id: ""},
		{name: "non-existent ID", id: "00000000-0000-0000-0000-000000000000"},
		{name: "random string", id: "i-do-not-exist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sess, err := mgr.GetSession(tt.id)
			if err == nil {
				t.Fatal("expected error for unknown session, got nil")
			}
			if sess != nil {
				t.Errorf("expected nil session, got %+v", sess)
			}
		})
	}
}

// TestManager_ListSessions_Empty verifies ListSessions returns an empty slice
// when no sessions have been created.
func TestManager_ListSessions_Empty(t *testing.T) {
	mgr := inner.NewInnerAgentManager("dve-test", "/tmp/test-workspace")

	sessions := mgr.ListSessions()
	if sessions == nil {
		t.Fatal("ListSessions returned nil, expected empty slice")
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

// TestManager_ListTools verifies ListTools returns the DefaultCatalog tools.
func TestManager_ListTools(t *testing.T) {
	mgr := inner.NewInnerAgentManager("dve-test", "/tmp/test-workspace")

	tools := mgr.ListTools()
	expectedNames := []string{"claude", "aider", "codex", "gh-copilot"}

	if len(tools) != len(expectedNames) {
		t.Fatalf("ListTools returned %d tools, want %d", len(tools), len(expectedNames))
	}

	got := make(map[string]bool)
	for _, tool := range tools {
		got[tool.Name] = true
	}

	for _, name := range expectedNames {
		if !got[name] {
			t.Errorf("ListTools missing tool %q", name)
		}
	}
}

// TestManager_Spawn_UnknownTool verifies Spawn returns an error when given an
// unknown tool name.
func TestManager_Spawn_UnknownTool(t *testing.T) {
	mgr := inner.NewInnerAgentManager("dve-test", "/tmp/test-workspace")

	tests := []struct {
		name     string
		toolName string
	}{
		{name: "empty name", toolName: ""},
		{name: "non-existent tool", toolName: "vim"},
		{name: "typo tool", toolName: "claud"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sess, err := mgr.Spawn(tt.toolName, nil)
			if err == nil {
				t.Fatalf("expected error for unknown tool %q, got nil session", tt.toolName)
			}
			if sess != nil {
				t.Errorf("expected nil session, got %+v", sess)
			}
		})
	}
}

// TestManager_Spawn_ToolNotInstalled verifies Spawn returns an error when the
// tool binary is not installed on the system.
func TestManager_Spawn_ToolNotInstalled(t *testing.T) {
	mgr := inner.NewInnerAgentManager("dve-test", "/tmp/test-workspace")

	tests := []struct {
		name     string
		toolName string
	}{
		{name: "claude not installed", toolName: "claude"},
		{name: "aider not installed", toolName: "aider"},
		{name: "codex not installed", toolName: "codex"},
		{name: "gh-copilot not installed", toolName: "gh-copilot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sess, err := mgr.Spawn(tt.toolName, nil)
			if err == nil {
				t.Fatalf("expected error for uninstalled tool %q, got nil", tt.toolName)
			}
			if sess != nil {
				t.Errorf("expected nil session, got %+v", sess)
			}
		})
	}
}

// TestManager_Install_UnknownTool verifies Install returns an error for an
// unknown tool name.
func TestManager_Install_UnknownTool(t *testing.T) {
	mgr := inner.NewInnerAgentManager("dve-test", "/tmp/test-workspace")

	tests := []struct {
		name     string
		toolName string
	}{
		{name: "empty name", toolName: ""},
		{name: "non-existent tool", toolName: "nonexistent-agent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.Install(tt.toolName)
			if err == nil {
				t.Errorf("expected error for Install(%q), got nil", tt.toolName)
			}
		})
	}
}

// TestManager_SendInput_NotFound verifies SendInput returns an error for a
// non-existent session.
func TestManager_SendInput_NotFound(t *testing.T) {
	mgr := inner.NewInnerAgentManager("dve-test", "/tmp/test-workspace")

	tests := []struct {
		name string
		id   string
	}{
		{name: "empty ID", id: ""},
		{name: "invalid session ID", id: "00000000-0000-0000-0000-000000000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.SendInput(tt.id, []byte("test input"))
			if err == nil {
				t.Errorf("expected error for SendInput(%q), got nil", tt.id)
			}
		})
	}
}

// TestManager_Terminate_NonExistent verifies Terminate returns nil (no error)
// for a non-existent session (the production code returns nil for unknown sessions).
func TestManager_Terminate_NonExistent(t *testing.T) {
	mgr := inner.NewInnerAgentManager("dve-test", "/tmp/test-workspace")

	tests := []struct {
		name string
		id   string
	}{
		{name: "empty ID", id: ""},
		{name: "non-existent session", id: "does-not-exist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.Terminate(tt.id)
			if err != nil {
				t.Errorf("expected nil error for Terminate(%q), got %v", tt.id, err)
			}
		})
	}
}
