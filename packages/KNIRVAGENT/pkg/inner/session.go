package inner

import (
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

type SessionStatus string

const (
	StatusRunning SessionStatus = "running"
	StatusExited  SessionStatus = "exited"
)

type InnerAgentSession struct {
	ID           string
	ToolName     string
	Cmd          *exec.Cmd
	PTY          *os.File
	Status       SessionStatus
	StartedAt    time.Time
	WorkspaceDir string
	DVEID        string

	supervisorLog io.Writer

	mu          sync.RWMutex
	subscribers []chan []byte
	done        chan struct{}
}

func (s *InnerAgentSession) Subscribe() chan []byte {
	ch := make(chan []byte, 256)
	s.mu.Lock()
	s.subscribers = append(s.subscribers, ch)
	s.mu.Unlock()
	return ch
}

func (s *InnerAgentSession) Unsubscribe(ch chan []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, sub := range s.subscribers {
		if sub == ch {
			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
			return
		}
	}
}

func (s *InnerAgentSession) broadcast(chunk []byte) {
	s.supervisorLog.Write(chunk)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range s.subscribers {
		select {
		case ch <- append([]byte(nil), chunk...):
		default:
		}
	}
}

func (s *InnerAgentSession) PumpOutput() {
	buf := make([]byte, 4096)
	for {
		n, err := s.PTY.Read(buf)
		if n > 0 {
			s.broadcast(buf[:n])
		}
		if err != nil {
			break
		}
	}
	s.mu.Lock()
	s.Status = StatusExited
	for _, ch := range s.subscribers {
		close(ch)
	}
	s.subscribers = nil
	s.mu.Unlock()
	close(s.done)
}
