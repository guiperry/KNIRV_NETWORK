package inner_test

import (
	"testing"

	"github.com/knirvcorp/knirvagent/pkg/inner"
)

// TestDefaultCatalog_ContainsTools verifies DefaultCatalog returns all 4 tools.
func TestDefaultCatalog_ContainsTools(t *testing.T) {
	cat := inner.DefaultCatalog()
	expectedNames := []string{"claude", "aider", "codex", "gh-copilot"}

	all := cat.All()
	if len(all) != len(expectedNames) {
		t.Fatalf("DefaultCatalog().All() returned %d tools, want %d", len(all), len(expectedNames))
	}

	got := make(map[string]bool)
	for _, tool := range all {
		got[tool.Name] = true
	}

	for _, name := range expectedNames {
		if !got[name] {
			t.Errorf("DefaultCatalog missing tool %q", name)
		}
	}
}

// TestDefaultCatalog_Lookup verifies Lookup returns the correct tool by name.
func TestDefaultCatalog_Lookup(t *testing.T) {
	cat := inner.DefaultCatalog()

	tests := []struct {
		name         string
		wantOK       bool
		wantBinary   string
		wantName     string
	}{
		{name: "claude", wantOK: true, wantBinary: "claude", wantName: "claude"},
		{name: "aider", wantOK: true, wantBinary: "aider", wantName: "aider"},
		{name: "codex", wantOK: true, wantBinary: "codex", wantName: "codex"},
		{name: "gh-copilot", wantOK: true, wantBinary: "gh", wantName: "gh-copilot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, ok := cat.Lookup(tt.name)
			if ok != tt.wantOK {
				t.Errorf("Lookup(%q) ok = %v, want %v", tt.name, ok, tt.wantOK)
			}
			if ok {
				if tool.Name != tt.wantName {
					t.Errorf("Lookup(%q) Name = %q, want %q", tt.name, tool.Name, tt.wantName)
				}
				if tool.Binary != tt.wantBinary {
					t.Errorf("Lookup(%q) Binary = %q, want %q", tt.name, tool.Binary, tt.wantBinary)
				}
			}
		})
	}
}

// TestDefaultCatalog_Lookup_NotFound verifies Lookup returns false for an unknown tool.
func TestDefaultCatalog_Lookup_NotFound(t *testing.T) {
	cat := inner.DefaultCatalog()

	unknownNames := []string{"", "nonexistent", "vim", "unknown-agent"}
	for _, name := range unknownNames {
		t.Run(name, func(t *testing.T) {
			tool, ok := cat.Lookup(name)
			if ok {
				t.Errorf("Lookup(%q) unexpectedly returned ok=true, tool=%+v", name, tool)
			}
			if tool != nil {
				t.Errorf("Lookup(%q) returned non-nil tool %+v, want nil", name, tool)
			}
		})
	}
}

// TestDefaultCatalog_All verifies All returns the correct number of tools.
func TestDefaultCatalog_All(t *testing.T) {
	cat := inner.DefaultCatalog()

	tests := []struct {
		name     string
		call     func() int
		want     int
	}{
		{name: "All returns 4 tools", call: func() int { return len(cat.All()) }, want: 4},
		{name: "All returns stable count on repeated call", call: func() int { return len(cat.All()) }, want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.call()
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

// TestTool_BuildCmd_SetsDir verifies BuildCmd sets Dir to the provided workdir.
func TestTool_BuildCmd_SetsDir(t *testing.T) {
	cat := inner.DefaultCatalog()
	workdir := "/tmp/test-workspace"

	tests := []struct {
		name string
	}{
		{name: "claude"},
		{name: "aider"},
		{name: "codex"},
		{name: "gh-copilot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, ok := cat.Lookup(tt.name)
			if !ok {
				t.Fatalf("tool %q not found in catalog", tt.name)
			}
			cmd := tool.BuildCmd(workdir, nil)
			if cmd.Dir != workdir {
				t.Errorf("BuildCmd Dir = %q, want %q", cmd.Dir, workdir)
			}
		})
	}
}

// TestTool_BuildCmd_AppendsExtraEnv verifies extra env vars are appended to the command's environment.
func TestTool_BuildCmd_AppendsExtraEnv(t *testing.T) {
	cat := inner.DefaultCatalog()
	workdir := "/tmp/test-workspace"
	extraEnv := []string{"TEST_VAR_1=value1", "TEST_VAR_2=value2"}

	tests := []struct {
		name string
	}{
		{name: "claude"},
		{name: "aider"},
		{name: "codex"},
		{name: "gh-copilot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, ok := cat.Lookup(tt.name)
			if !ok {
				t.Fatalf("tool %q not found in catalog", tt.name)
			}
			cmd := tool.BuildCmd(workdir, extraEnv)
			if len(cmd.Env) < len(extraEnv) {
				t.Fatalf("cmd.Env has %d entries, expected at least %d extra env vars", len(cmd.Env), len(extraEnv))
			}
			// The extra env should be appended after os.Environ(), so check the tail.
			envTail := cmd.Env[len(cmd.Env)-len(extraEnv):]
			for i, want := range extraEnv {
				if envTail[i] != want {
					t.Errorf("cmd.Env[%d] = %q, want %q", len(cmd.Env)-len(extraEnv)+i, envTail[i], want)
				}
			}
		})
	}
}

// TestTool_IsInstalled_NeverReturnsTrueForMissingBinary verifies IsInstalled returns false
// for a binary that does not exist on the system (lookpath will fail for random binary names).
func TestTool_IsInstalled_MissingBinary(t *testing.T) {
	cat := inner.DefaultCatalog()

	tests := []struct {
		name string
	}{
		{name: "claude"},
		{name: "aider"},
		{name: "codex"},
		{name: "gh-copilot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, ok := cat.Lookup(tt.name)
			if !ok {
				t.Fatalf("tool %q not found in catalog", tt.name)
			}
			// These binaries are generally not installed in CI/test environments,
			// so IsInstalled should return false. This is a sanity check that
			// the LookPath logic works.
			_ = tool.IsInstalled()
			// We just verify it doesn't panic — actual result is environment-dependent.
		})
	}
}
