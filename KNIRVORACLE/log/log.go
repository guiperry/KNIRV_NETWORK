package log

import (
	"log"
	"os"
)

// LogError logs an error message with context
func LogError(message string, err error) {
	log.Printf("[ERROR] %s: %v\n", message, err)
}

// LogWarning logs a warning message
func LogWarning(message string) {
	log.Printf("[WARNING] %s\n", message)
}

// LogInfo logs an informational message
func LogInfo(message string) {
	log.Printf("[INFO] %s\n", message)
}

// FatalError logs an error message and terminates the program
func FatalError(message string, err error) {
	LogError(message, err)
	os.Exit(1)
}
