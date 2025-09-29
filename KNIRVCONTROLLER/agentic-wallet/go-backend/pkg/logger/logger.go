package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Logger levels
const (
	DEBUG = "DEBUG"
	INFO  = "INFO"
	WARN  = "WARN"
	ERROR = "ERROR"
	FATAL = "FATAL"
)

// Logger represents the application logger
type Logger struct {
	level  string
	logger *log.Logger
}

// Config holds logger configuration
type Config struct {
	Level  string
	Output string
}

// NewLogger creates a new logger instance
func NewLogger(config Config) *Logger {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	
	return &Logger{
		level:  config.Level,
		logger: logger,
	}
}

// Debug logs debug messages
func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.shouldLog(DEBUG) {
		l.log(DEBUG, msg, args...)
	}
}

// Info logs info messages
func (l *Logger) Info(msg string, args ...interface{}) {
	if l.shouldLog(INFO) {
		l.log(INFO, msg, args...)
	}
}

// Warn logs warning messages
func (l *Logger) Warn(msg string, args ...interface{}) {
	if l.shouldLog(WARN) {
		l.log(WARN, msg, args...)
	}
}

// Error logs error messages
func (l *Logger) Error(msg string, args ...interface{}) {
	if l.shouldLog(ERROR) {
		l.log(ERROR, msg, args...)
	}
}

// Fatal logs fatal messages and exits
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.log(FATAL, msg, args...)
	os.Exit(1)
}

// log performs the actual logging
func (l *Logger) log(level, msg string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(msg, args...)
	l.logger.Printf("[%s] %s: %s", timestamp, level, message)
}

// shouldLog determines if a message should be logged based on level
func (l *Logger) shouldLog(level string) bool {
	levels := map[string]int{
		DEBUG: 0,
		INFO:  1,
		WARN:  2,
		ERROR: 3,
		FATAL: 4,
	}
	
	currentLevel, exists := levels[l.level]
	if !exists {
		currentLevel = levels[INFO]
	}
	
	messageLevel, exists := levels[level]
	if !exists {
		return false
	}
	
	return messageLevel >= currentLevel
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level string) {
	l.level = level
}

// GetLevel returns the current logging level
func (l *Logger) GetLevel() string {
	return l.level
}
