package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

const (
	MaxLogBuffer  int = 1000
	MaxSSEClients int = 100
)

type LogEvent struct {
	Type      EventType              `json:"type"`
	Module    string                 `json:"module"` // "knirvchain", "knirvgateway", "knirvgraph"
	Level     string                 `json:"level"`  // "info", "warn", "error", "debug"
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type LogStreamHandler struct {
	clients    map[string]map[chan LogEvent]bool
	mu         sync.RWMutex
	logBuffer  []LogEvent
	bufferSize int
}

var logStreamHandler *LogStreamHandler

func GetLogStreamHandler() *LogStreamHandler {
	if logStreamHandler == nil {
		logStreamHandler = &LogStreamHandler{
			clients:    make(map[string]map[chan LogEvent]bool),
			logBuffer:  make([]LogEvent, 0, MaxLogBuffer),
			bufferSize: MaxLogBuffer,
		}
	}
	return logStreamHandler
}

func (h *LogStreamHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/logs/stream", h.handleLogStream).Methods("GET")
	r.HandleFunc("/api/logs/module/{module}", h.handleModuleLogStream).Methods("GET")
	r.HandleFunc("/api/logs/history", h.handleLogHistory).Methods("GET")
}

func (h *LogStreamHandler) handleLogStream(w http.ResponseWriter, r *http.Request) {
	h.serveSSE(w, r, "")
}

func (h *LogStreamHandler) handleModuleLogStream(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	module := vars["module"]
	h.serveSSE(w, r, module)
}

func (h *LogStreamHandler) serveSSE(w http.ResponseWriter, r *http.Request, module string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if module != "" {
		w.Header().Set("X-Target-Module", module)
	}

	clientChan := make(chan LogEvent, 100)

	h.mu.Lock()
	if len(h.clients) >= MaxSSEClients {
		h.mu.Unlock()
		http.Error(w, "Too many clients", http.StatusTooManyRequests)
		return
	}
	if h.clients[module] == nil {
		h.clients[module] = make(map[chan LogEvent]bool)
	}
	h.clients[module][clientChan] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients[module], clientChan)
		close(clientChan)
		h.mu.Unlock()
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	h.sendInitialLogs(w, flusher, module)

	notify := r.Context().Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-notify:
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case logEvent := <-clientChan:
			data, err := json.Marshal(logEvent)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (h *LogStreamHandler) sendInitialLogs(w http.ResponseWriter, flusher http.Flusher, module string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for i := len(h.logBuffer) - 1; i >= 0 && i >= len(h.logBuffer)-50; i-- {
		logEvent := h.logBuffer[i]
		if module == "" || logEvent.Module == module {
			data, _ := json.Marshal(logEvent)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
	}
	flusher.Flush()
}

func (h *LogStreamHandler) handleLogHistory(w http.ResponseWriter, r *http.Request) {
	module := r.URL.Query().Get("module")
	level := r.URL.Query().Get("level")
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	var logs []LogEvent
	for i := len(h.logBuffer) - 1; i >= 0 && len(logs) < limit; i-- {
		logEvent := h.logBuffer[i]
		if module != "" && logEvent.Module != module {
			continue
		}
		if level != "" && logEvent.Level != level {
			continue
		}
		logs = append(logs, logEvent)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":  logs,
		"total": len(logs),
	})
}

func (h *LogStreamHandler) EmitLog(module, level, message, source string, metadata map[string]interface{}) {
	logEvent := LogEvent{
		Type:      EventTypeModuleLog,
		Module:    module,
		Level:     level,
		Message:   message,
		Source:    source,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}

	h.mu.Lock()
	h.logBuffer = append(h.logBuffer, logEvent)
	if len(h.logBuffer) > h.bufferSize {
		h.logBuffer = h.logBuffer[1:]
	}
	h.mu.Unlock()

	h.broadcastLog(logEvent)
}

func (h *LogStreamHandler) broadcastLog(logEvent LogEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for module, clients := range h.clients {
		if module == "" || module == logEvent.Module {
			for clientChan := range clients {
				select {
				case clientChan <- logEvent:
				default:
				}
			}
		}
	}
}

func (h *LogStreamHandler) GetLogBuffer() []LogEvent {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]LogEvent, len(h.logBuffer))
	copy(result, h.logBuffer)
	return result
}
