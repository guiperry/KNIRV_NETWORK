package inner_test

import (
	"testing"
	"time"

	"github.com/knirvcorp/knirvagent/pkg/inner"
)

// newTestSession creates an InnerAgentSession with minimal fields for testing
// subscribe/unsubscribe/broadcast channel operations. PTY and Cmd are left nil
// since we are not testing subprocess output pumping.
func newTestSession(id string) *inner.InnerAgentSession {
	return inner.NewTestSession(id)
}

// TestSession_Subscribe verifies that subscribing returns a buffered channel.
func TestSession_Subscribe(t *testing.T) {
	sess := newTestSession("test-subscribe")
	ch := sess.Subscribe()

	if ch == nil {
		t.Fatal("Subscribe() returned nil channel")
	}

	// Verify the channel is usable — it should be buffered (non-blocking send).
	select {
	case ch <- []byte("hello"):
	default:
		t.Fatal("channel is not buffered; send blocked")
	}

	// Read back what we sent.
	select {
	case msg := <-ch:
		if string(msg) != "hello" {
			t.Errorf("received %q, want %q", string(msg), "hello")
		}
	default:
		t.Fatal("expected message in channel, got nothing")
	}
}

// TestSession_Subscribe_Broadcast verifies that broadcast sends data to all subscribers.
func TestSession_Subscribe_Broadcast(t *testing.T) {
	sess := newTestSession("test-broadcast")

	// Subscribe multiple listeners.
	ch1 := sess.Subscribe()
	ch2 := sess.Subscribe()
	ch3 := sess.Subscribe()

	// Use the package-level Broadcast helper to send data to all subscribers.
	msg := []byte("broadcast message")
	inner.BroadcastToSession(sess, msg)

	// Give the mutex a moment to release after broadcast.
	// (Broadcast acquires RLock, so it returns immediately after the loop.)
	time.Sleep(10 * time.Millisecond)

	// Verify all subscribers received the message.
	checkMsg := func(t *testing.T, ch chan []byte, label string) {
		t.Helper()
		select {
		case got := <-ch:
			if string(got) != string(msg) {
				t.Errorf("%s received %q, want %q", label, string(got), string(msg))
			}
		default:
			t.Errorf("%s received nothing from broadcast", label)
		}
	}

	checkMsg(t, ch1, "ch1")
	checkMsg(t, ch2, "ch2")
	checkMsg(t, ch3, "ch3")
}

// TestSession_Unsubscribe verifies that unsubscribing removes the channel from the list.
func TestSession_Unsubscribe(t *testing.T) {
	sess := newTestSession("test-unsubscribe")

	ch1 := sess.Subscribe()
	ch2 := sess.Subscribe()

	// Unsubscribe ch1.
	sess.Unsubscribe(ch1)

	// Broadcast should only reach ch2.
	msg := []byte("after unsubscribe")
	inner.BroadcastToSession(sess, msg)
	time.Sleep(10 * time.Millisecond)

	// ch2 should get it.
	select {
	case got := <-ch2:
		if string(got) != string(msg) {
			t.Errorf("ch2 received %q, want %q", string(got), string(msg))
		}
	default:
		t.Error("ch2 should have received broadcast after ch1 unsubscribe")
	}

	// ch1 should NOT get it — the channel is no longer in the subscriber list.
	// But there might be a stale message in the buffer from before unsubscription.
	// Drain any stale data first.
drain:
	for {
		select {
		case <-ch1:
		default:
			break drain
		}
	}

	// Now verify no more data arrives after unsubscribe.
	select {
	case got := <-ch1:
		t.Errorf("ch1 unexpectedly received %q after unsubscribe", string(got))
	default:
		// Expected — ch1 should not receive after being unsubscribed.
	}
}

// TestSession_Unsubscribe_NonExistent verifies unsubscribing a channel not in the
// subscriber list does not panic.
func TestSession_Unsubscribe_NonExistent(t *testing.T) {
	sess := newTestSession("test-unsubscribe-nonexistent")

	// Subscribe one channel, then try to unsubscribe a different one.
	_ = sess.Subscribe()

	// Create a channel that was never subscribed.
	unknownCh := make(chan []byte, 1)

	// This should not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Unsubscribe panicked with non-existent channel: %v", r)
		}
	}()

	sess.Unsubscribe(unknownCh)
}

// TestSession_MultipleSubscribeAndBroadcast verifies correct behavior with
// multiple subscribe/unsubscribe cycles and broadcasts.
func TestSession_MultipleSubscribeAndBroadcast(t *testing.T) {
	sess := newTestSession("test-multi-cycle")

	// Subscribe 3, unsubscribe 1, verify broadcast still reaches the remaining 2.
	chs := make([]chan []byte, 3)
	for i := range chs {
		chs[i] = sess.Subscribe()
	}

	// Unsubscribe the middle one.
	sess.Unsubscribe(chs[1])

	msg := []byte("multi-cycle test")
	inner.BroadcastToSession(sess, msg)
	time.Sleep(10 * time.Millisecond)

	// chs[0] and chs[2] should receive.
	select {
	case got := <-chs[0]:
		if string(got) != string(msg) {
			t.Errorf("chs[0] received %q, want %q", string(got), string(msg))
		}
	default:
		t.Error("chs[0] should have received broadcast")
	}

	// Drain chs[1] of any stale data.
drainCh1:
	for {
		select {
		case <-chs[1]:
		default:
			break drainCh1
		}
	}

	select {
	case got := <-chs[1]:
		t.Errorf("chs[1] (unsubscribed) unexpectedly received %q", string(got))
	default:
		// Expected.
	}

	select {
	case got := <-chs[2]:
		if string(got) != string(msg) {
			t.Errorf("chs[2] received %q, want %q", string(got), string(msg))
		}
	default:
		t.Error("chs[2] should have received broadcast")
	}

	// Now unsubscribe all remaining and verify no one receives.
	sess.Unsubscribe(chs[0])
	sess.Unsubscribe(chs[2])

	inner.BroadcastToSession(sess, []byte("should not reach anyone"))
	time.Sleep(10 * time.Millisecond)

	for i, ch := range chs {
		select {
		case got := <-ch:
			t.Errorf("chs[%d] (unsubscribed) received %q after full unsubscribe", i, string(got))
		default:
			// Expected.
		}
	}
}
