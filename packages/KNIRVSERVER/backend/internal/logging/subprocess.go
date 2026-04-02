package logging

import (
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type SubprocessWriter struct {
	mu       sync.Mutex
	module   string
	original io.Writer
	closed   bool
}

func NewSubprocessWriter(module string, original io.Writer) *SubprocessWriter {
	return &SubprocessWriter{
		module:   module,
		original: original,
	}
}

func (sw *SubprocessWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if sw.closed {
		return 0, io.EOF
	}

	line := string(p)
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}

	if len(line) > 0 {
		EmitModuleLog(sw.module, detectLevel(line), line)
	}

	if sw.original != nil {
		return sw.original.Write(p)
	}
	return len(p), nil
}

func (sw *SubprocessWriter) Close() error {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.closed = true
	return nil
}

func detectLevel(line string) string {
	lower := strings.ToLower(line)
	if strings.Contains(lower, "error") || strings.Contains(lower, "failed") || strings.Contains(lower, "fatal") {
		return "error"
	}
	if strings.Contains(lower, "warn") {
		return "warn"
	}
	if strings.Contains(lower, "debug") || strings.Contains(lower, "trace") {
		return "debug"
	}
	return "info"
}

func CreatePipeWriters(module string) (io.Writer, io.Writer) {
	stdoutWriter := NewSubprocessWriter(module, os.Stdout)
	stderrWriter := NewSubprocessWriter(module, os.Stderr)
	return stdoutWriter, stderrWriter
}

type SubprocessMonitor struct {
	mu         sync.Mutex
	module     string
	running    bool
	interval   time.Duration
	done       chan bool
	logHandler func(module, level, message string)
}

func NewSubprocessMonitor(module string, interval time.Duration, logHandler func(module, level, message string)) *SubprocessMonitor {
	return &SubprocessMonitor{
		module:     module,
		interval:   interval,
		done:       make(chan bool),
		logHandler: logHandler,
	}
}

func (sm *SubprocessMonitor) Start() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.running {
		return
	}
	sm.running = true

	go func() {
		ticker := time.NewTicker(sm.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if sm.logHandler != nil {
					sm.logHandler(sm.module, "debug", "Health check: running")
				}
			case <-sm.done:
				return
			}
		}
	}()
}

func (sm *SubprocessMonitor) Stop() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if !sm.running {
		return
	}
	sm.running = false
	close(sm.done)
}

func (sm *SubprocessMonitor) EmitLog(level, message string) {
	if sm.logHandler != nil {
		sm.logHandler(sm.module, level, message)
	}
}
