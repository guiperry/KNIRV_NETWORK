package registry

import (
	"path/filepath"
	"testing"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/runner"
)

func TestUpsertAndGet(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "agents.yaml"))
	agent := AgentRecord{
		Name: "unit-agent",
		Type: runner.AgentTypeLocal,
		Path: "/bin/echo",
		Args: []string{"hello"},
	}

	if err := store.Upsert(agent); err != nil {
		t.Fatalf("upsert agent: %v", err)
	}

	got, err := store.Get("unit-agent")
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}

	if got.Name != agent.Name {
		t.Fatalf("got name %q, want %q", got.Name, agent.Name)
	}
	if got.Path != agent.Path {
		t.Fatalf("got path %q, want %q", got.Path, agent.Path)
	}
	if got.UpdatedAt.IsZero() {
		t.Fatalf("expected updated_at to be set")
	}
}
