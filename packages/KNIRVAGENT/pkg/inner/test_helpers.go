package inner

import "io"

// NewTestSession creates an InnerAgentSession with minimal fields for testing
// subscribe/unsubscribe/broadcast channel operations. PTY and Cmd are left nil
// since PumpOutput is not intended to be called on this session.
func NewTestSession(id string) *InnerAgentSession {
	return &InnerAgentSession{
		ID:            id,
		subscribers:   make([]chan []byte, 0),
		supervisorLog: io.Discard,
	}
}

// BroadcastToSession calls the unexported broadcast method on the session.
// This is exposed as a test helper for package inner_test.
func BroadcastToSession(s *InnerAgentSession, data []byte) {
	s.broadcast(data)
}
