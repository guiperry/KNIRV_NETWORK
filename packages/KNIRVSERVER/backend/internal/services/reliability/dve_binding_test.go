package reliability

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDVEBindingManager(t *testing.T) {
	dm := NewDVEBindingManager()
	assert.NotNil(t, dm)
	assert.Empty(t, dm.ListBindings())
}

func TestBind(t *testing.T) {
	dm := NewDVEBindingManager()
	binding := dm.Bind("dve-1", "node-1", "agent-1", "breaker-1")

	assert.Equal(t, "dve-1", binding.DVEID)
	assert.Equal(t, "node-1", binding.NodeID)
	assert.Equal(t, "agent-1", binding.AgentID)
	assert.Equal(t, "breaker-1", binding.BreakerID)
	assert.NotZero(t, binding.BoundAt)
}

func TestGetBinding(t *testing.T) {
	dm := NewDVEBindingManager()
	dm.Bind("dve-1", "node-1", "agent-1", "breaker-1")

	binding, ok := dm.GetBinding("breaker-1")
	assert.True(t, ok)
	assert.Equal(t, "dve-1", binding.DVEID)
}

func TestGetBindingNotFound(t *testing.T) {
	dm := NewDVEBindingManager()
	_, ok := dm.GetBinding("nonexistent")
	assert.False(t, ok)
}

func TestUnbind(t *testing.T) {
	dm := NewDVEBindingManager()
	dm.Bind("dve-1", "node-1", "agent-1", "breaker-1")
	assert.Len(t, dm.ListBindings(), 1)

	dm.Unbind("breaker-1")
	assert.Empty(t, dm.ListBindings())

	_, ok := dm.GetBinding("breaker-1")
	assert.False(t, ok)
}

func TestUnbindNonexistent(t *testing.T) {
	dm := NewDVEBindingManager()
	dm.Unbind("nonexistent")
	assert.Empty(t, dm.ListBindings())
}

func TestListBindings(t *testing.T) {
	dm := NewDVEBindingManager()
	dm.Bind("dve-1", "node-1", "agent-1", "breaker-1")
	dm.Bind("dve-2", "node-2", "agent-2", "breaker-2")

	bindings := dm.ListBindings()
	assert.Len(t, bindings, 2)

	ids := make(map[string]bool)
	for _, b := range bindings {
		ids[b.BreakerID] = true
	}
	assert.True(t, ids["breaker-1"])
	assert.True(t, ids["breaker-2"])
}
