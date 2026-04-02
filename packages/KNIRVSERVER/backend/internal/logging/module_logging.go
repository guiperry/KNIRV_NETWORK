package logging

import (
	"fmt"
	"io"
	"os"
	"sync"

	ws "backend_server/internal/services/websocket"
)

var logHandler *ws.LogStreamHandler

func SetLogStreamHandler(h *ws.LogStreamHandler) {
	logHandler = h
}

func getLogHandler() *ws.LogStreamHandler {
	return logHandler
}

func EmitModuleLog(module, level, message string) {
	if logHandler != nil {
		logHandler.EmitLog(module, level, message, "backend_server", nil)
		return
	}
	if wsHandler := ws.GetLogStreamHandler(); wsHandler != nil {
		wsHandler.EmitLog(module, level, message, "backend_server", nil)
	}
}

type ModuleLogger interface {
	EmitLog(level, msg string)
	GetModuleName() string
}

type LogEmitter struct {
	mu        sync.Mutex
	module    string
	logWriter io.Writer
	original  io.Writer
}

func NewLogEmitter(module string, writer io.Writer) *LogEmitter {
	return &LogEmitter{
		module:    module,
		logWriter: writer,
		original:  writer,
	}
}

func (le *LogEmitter) StartCapture() {
	le.mu.Lock()
	defer le.mu.Unlock()
	if le.logWriter == nil {
		return
	}
	le.original = le.logWriter
	le.logWriter = &logPrefixWriter{
		module: le.module,
		parent: le.original,
	}
}

func (le *LogEmitter) StopCapture() {
	le.mu.Lock()
	defer le.mu.Unlock()
	if le.logWriter != nil {
		le.logWriter = le.original
	}
}

func (le *LogEmitter) GetWriter() io.Writer {
	le.mu.Lock()
	defer le.mu.Unlock()
	return le.logWriter
}

type logPrefixWriter struct {
	module string
	parent io.Writer
}

func (lw *logPrefixWriter) Write(p []byte) (n int, err error) {
	line := string(p)
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	prefixed := fmt.Sprintf("[%s] %s\n", lw.module, line)
	return lw.parent.Write([]byte(prefixed))
}

var logEmitters = make(map[string]*LogEmitter)
var logEmitterMu sync.Mutex

func GetLogEmitter(module string) *LogEmitter {
	logEmitterMu.Lock()
	defer logEmitterMu.Unlock()
	if le, ok := logEmitters[module]; ok {
		return le
	}
	le := NewLogEmitter(module, os.Stdout)
	logEmitters[module] = le
	return le
}
