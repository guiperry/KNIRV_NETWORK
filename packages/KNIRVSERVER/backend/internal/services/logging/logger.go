package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"
)

// LogLevel defines the severity level of a log entry
type LogLevel string

const (
	DEBUG  LogLevel = "DEBUG"
	INFO   LogLevel = "INFO"
	WARN   LogLevel = "WARN"
	ERROR  LogLevel = "ERROR"
	FATAL  LogLevel = "FATAL"
)

// Logger defines the interface for structured logging
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// JSONLogger implements Logger with JSON output
type JSONLogger struct {
	writer     io.Writer
	serviceName string
}

// NewJSONLogger creates a new JSON logger
func NewJSONLogger(serviceName string, writer io.Writer) *JSONLogger {
	if writer == nil {
		writer = os.Stdout
	}
	return &JSONLogger{
		writer:     writer,
		serviceName: serviceName,
	}
}

// log creates a JSON log entry
func (l *JSONLogger) log(level LogLevel, msg string, fields []Field) {
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"level":     string(level),
		"service":   l.serviceName,
		"message":   msg,
	}

	// Add custom fields
	for _, field := range fields {
		entry[field.Key] = field.Value
	}

	// Add correlation ID if present (would typically come from middleware/context)
	// This is just an example - in practice you'd get this from request context
	if correlationID, ok := l.getCorrelationID(); ok {
		entry["correlation_id"] = correlationID
	}

	// Add source location for errors
	if level == ERROR || level == FATAL {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			entry["source"] = fmt.Sprintf("%s:%d", file, line)
		}
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling log entry: %v\n", err)
		return
	}

	fmt.Fprintln(l.writer, string(jsonData))

	if level == FATAL {
		os.Exit(1)
	}
}

// Helper methods for each log level
func (l *JSONLogger) Debug(msg string, fields ...Field) {
	l.log(DEBUG, msg, fields)
}

func (l *JSONLogger) Info(msg string, fields ...Field) {
	l.log(INFO, msg, fields)
}

func (l *JSONLogger) Warn(msg string, fields ...Field) {
	l.log(WARN, msg, fields)
}

func (l *JSONLogger) Error(msg string, fields ...Field) {
	l.log(ERROR, msg, fields)
}

func (l *JSONLogger) Fatal(msg string, fields ...Field) {
	l.log(FATAL, msg, fields)
}

// getCorrelationID is a placeholder - in practice this would come from middleware
// that extracts correlation ID from headers or generates one
func (l *JSONLogger) getCorrelationID() (string, bool) {
	// Example implementation - in reality this would come from middleware
	// that extracts correlation ID from headers or generates one
	return "", false
}

// Helper functions to create fields
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

func ErrorField(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// HandlerLogger creates a logger with handler-specific fields pre-populated
func HandlerLogger(handlerName string) *JSONLogger {
	logger := NewJSONLogger("knirvserver", os.Stdout)
	return logger
}

// WithRequestFields adds common request-related fields to log entries
func WithRequestFields(logger *JSONLogger, method, path, userID string) []Field {
	return []Field{
		String("method", method),
		String("path", path),
		String("user_id", userID),
	}
}
