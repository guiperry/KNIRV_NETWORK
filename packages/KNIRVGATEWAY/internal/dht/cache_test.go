package dht

import (
	"sync"
	"testing"
	"time"
)

func TestNewResourceCache(t *testing.T) {
	rc := NewResourceCache()
	if rc == nil {
		t.Fatal("NewResourceCache returned nil")
	}
	if rc.GetResourceCount() != 0 {
		t.Errorf("expected 0 resources, got %d", rc.GetResourceCount())
	}
}

func TestAddResource(t *testing.T) {
	rc := NewResourceCache()

	resource := ResourceEntry{
		ID:          "skill-123",
		Type:        "skill",
		Multiaddress: "/ip4/127.0.0.1/tcp/4001",
		Source:      "knirvgraph",
	}

	rc.AddResource(resource)

	if rc.GetResourceCount() != 1 {
		t.Errorf("expected 1 resource, got %d", rc.GetResourceCount())
	}

	resources := rc.GetAllResources()
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource in slice, got %d", len(resources))
	}

	if resources[0].ID != "skill-123" {
		t.Errorf("expected ID 'skill-123', got %s", resources[0].ID)
	}

	if resources[0].Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}

func TestAddResource_Update(t *testing.T) {
	rc := NewResourceCache()

	r1 := ResourceEntry{
		ID:          "cap-1",
		Type:        "capability",
		Multiaddress: "/ip4/1.2.3.4/tcp/4001",
		Source:      "knirvchain",
	}
	rc.AddResource(r1)

	r2 := ResourceEntry{
		ID:          "cap-1",
		Type:        "capability",
		Multiaddress: "/ip4/5.6.7.8/tcp/4002",
		Source:      "knirvchain",
	}
	rc.AddResource(r2)

	if rc.GetResourceCount() != 1 {
		t.Errorf("expected 1 resource after update, got %d", rc.GetResourceCount())
	}

	resources := rc.GetAllResources()
	if resources[0].Multiaddress != "/ip4/5.6.7.8/tcp/4002" {
		t.Errorf("expected updated multiaddress, got %s", resources[0].Multiaddress)
	}
}

func TestGetAllResources(t *testing.T) {
	rc := NewResourceCache()

	resources := []ResourceEntry{
		{ID: "skill-1", Type: "skill", Multiaddress: "/ip4/1.2.3.4/tcp/4001", Source: "knirvgraph"},
		{ID: "cap-1", Type: "capability", Multiaddress: "/ip4/1.2.3.4/tcp/4001", Source: "knirvchain"},
		{ID: "prop-1", Type: "property", Multiaddress: "/ip4/1.2.3.4/tcp/4001", Source: "knirvchain"},
	}

	for _, r := range resources {
		rc.AddResource(r)
	}

	all := rc.GetAllResources()
	if len(all) != 3 {
		t.Errorf("expected 3 resources, got %d", len(all))
	}
}

func TestClearCache(t *testing.T) {
	rc := NewResourceCache()

	rc.AddResource(ResourceEntry{ID: "s1", Type: "skill", Multiaddress: "/ip4/1.2.3.4/tcp/4001"})
	rc.AddResource(ResourceEntry{ID: "s2", Type: "skill", Multiaddress: "/ip4/1.2.3.4/tcp/4001"})

	rc.ClearCache()

	if rc.GetResourceCount() != 0 {
		t.Errorf("expected 0 after clear, got %d", rc.GetResourceCount())
	}
}

func TestConcurrentAccess(t *testing.T) {
	rc := NewResourceCache()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			rc.AddResource(ResourceEntry{
				ID:          string(rune('a' + idx)),
				Type:        "skill",
				Multiaddress: "/ip4/127.0.0.1/tcp/4001",
				Source:      "knirvgraph",
			})
		}(i)
	}

	wg.Wait()

	if rc.GetResourceCount() != 100 {
		t.Errorf("expected 100 resources, got %d", rc.GetResourceCount())
	}
}

func TestResourceTimestamp(t *testing.T) {
	rc := NewResourceCache()

	before := time.Now().Add(-time.Millisecond)
	rc.AddResource(ResourceEntry{ID: "t1", Type: "skill", Multiaddress: "/ip4/127.0.0.1/tcp/4001"})
	after := time.Now().Add(time.Millisecond)

	resources := rc.GetAllResources()
	ts := resources[0].Timestamp

	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp %v not within expected range", ts)
	}
}
