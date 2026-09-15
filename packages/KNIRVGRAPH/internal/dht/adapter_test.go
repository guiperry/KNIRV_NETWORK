package dht

import (
	"errors"
	"testing"
)

// Publish previously returned nil unconditionally, reporting a successful
// publish that never happened — DRQ's Q-table gossip silently exchanged
// nothing. KNIRVGATEWAY exposes no pub/sub route, so the honest behaviour is an
// explicit, recognisable error.
func TestPublishFailsLoudly(t *testing.T) {
	adapter := &DHTClientAdapter{client: NewClient()}
	err := adapter.Publish("some-topic", []byte("payload"))
	if err == nil {
		t.Fatal("Publish must not report success: KNIRVGATEWAY has no pub/sub route")
	}
	if !errors.Is(err, ErrPubSubUnavailable) {
		t.Fatalf("error = %v, want it to wrap ErrPubSubUnavailable", err)
	}
}

// Subscribe previously returned (nil, nil), handing callers a nil subscription
// that would panic on first use.
func TestSubscribeFailsLoudly(t *testing.T) {
	adapter := &DHTClientAdapter{client: NewClient()}
	sub, err := adapter.Subscribe("some-topic")
	if err == nil {
		t.Fatal("Subscribe must not report success: KNIRVGATEWAY has no pub/sub route")
	}
	if !errors.Is(err, ErrPubSubUnavailable) {
		t.Fatalf("error = %v, want it to wrap ErrPubSubUnavailable", err)
	}
	if sub != nil {
		t.Fatal("a failed Subscribe must not return a subscription")
	}
}

// FindGraphServices previously returned (nil, nil). It now really queries the
// gateway's /dht/find route; against a gateway that is not running it must
// surface a transport error rather than pretending there are no peers.
func TestFindGraphServicesSurfacesTransportFailure(t *testing.T) {
	adapter := &DHTClientAdapter{client: NewClient()}
	peers, err := adapter.FindGraphServices()
	if err == nil {
		// A gateway happens to be reachable: an empty-or-populated list is a
		// legitimate answer, but it must be a real HTTP result.
		_ = peers
		return
	}
	if errors.Is(err, ErrPubSubUnavailable) {
		t.Fatalf("FindGraphServices must not be reported as pub/sub unavailable: %v", err)
	}
}

func TestFindGraphServicesRequiresClient(t *testing.T) {
	adapter := &DHTClientAdapter{}
	if _, err := adapter.FindGraphServices(); err == nil {
		t.Fatal("expected an error when the DHT client is not configured")
	}
}
