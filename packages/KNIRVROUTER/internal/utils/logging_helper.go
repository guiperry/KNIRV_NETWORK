// utils/logging_helper.go
package utils

import (
	"fmt"
	"log"
	"os"
	"time"
)

// LogInfo logs informational messages (goes to stdout via standard logger)
func LogInfo(format string, v ...interface{}) {
	log.Printf(format, v...) // Standard log writes to stdout by default
}

// LogError logs non-fatal errors (goes to stderr)
func LogError(format string, v ...interface{}) {
	// Manually write to stderr with a timestamp and clear prefix
	fmt.Fprintf(os.Stderr, "%s ERROR: %s\n", time.Now().Format("2006/01/02 15:04:05"), fmt.Sprintf(format, v...))
}

// LogCritical logs critical errors (goes to stderr)
func LogCritical(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s CRITICAL: %s\n", time.Now().Format("2006/01/02 15:04:05"), fmt.Sprintf(format, v...))
}

// LogFatal logs fatal errors (writes to stderr and exits)
func LogFatal(format string, v ...interface{}) {
	// log.Fatalf already writes to stderr and includes timestamp/prefix
	log.Fatalf("FATAL: "+format, v...)
}