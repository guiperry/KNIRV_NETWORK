package utils

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLogRelay(t *testing.T) {
	t.Run("creates new log relay with callback", func(t *testing.T) {
		var receivedLogs []string
		var mu sync.Mutex

		callback := func(logText string) {
			mu.Lock()
			receivedLogs = append(receivedLogs, logText)
			mu.Unlock()
		}

		relay := NewLogRelay(callback)
		assert.NotNil(t, relay)
		assert.NotNil(t, relay.logMessageChannel)
		assert.NotNil(t, relay.uiUpdateCallback)
		assert.NotNil(t, relay.logBuffer)
		assert.False(t, relay.active)
	})

	t.Run("creates new log relay with nil callback", func(t *testing.T) {
		relay := NewLogRelay(nil)
		assert.NotNil(t, relay)
		assert.Nil(t, relay.uiUpdateCallback)
	})
}

func TestLogRelayStartStop(t *testing.T) {
	t.Run("start and stop log relay", func(t *testing.T) {
		// Save original log output
		originalOutput := log.Writer()

		var receivedLogs []string
		var mu sync.Mutex

		callback := func(logText string) {
			mu.Lock()
			receivedLogs = append(receivedLogs, logText)
			mu.Unlock()
		}

		relay := NewLogRelay(callback)

		// Start the relay
		relay.Start()
		assert.True(t, relay.active)

		// Give some time for the goroutine to start
		time.Sleep(10 * time.Millisecond)

		// Stop the relay
		relay.Stop()
		assert.False(t, relay.active)

		// Verify original output is restored
		assert.Equal(t, originalOutput, log.Writer())
	})

	t.Run("start when already active", func(t *testing.T) {
		originalOutput := log.Writer()

		callback := func(logText string) {}
		relay := NewLogRelay(callback)

		// Start the relay
		relay.Start()
		assert.True(t, relay.active)

		// Start again (should not change state)
		relay.Start()
		assert.True(t, relay.active)

		// Stop the relay
		relay.Stop()
		assert.False(t, relay.active)

		// Restore original output
		log.SetOutput(originalOutput)
	})

	t.Run("stop when not active", func(t *testing.T) {
		callback := func(logText string) {}
		relay := NewLogRelay(callback)

		// Stop without starting (should not panic)
		relay.Stop()
		assert.False(t, relay.active)
	})
}

func TestLogRelayWrite(t *testing.T) {
	t.Run("write to log relay", func(t *testing.T) {
		// Create a buffer to capture original output
		var originalBuffer bytes.Buffer
		originalOutput := log.Writer()
		log.SetOutput(&originalBuffer)

		var receivedLogs []string
		var mu sync.Mutex

		callback := func(logText string) {
			mu.Lock()
			receivedLogs = append(receivedLogs, logText)
			mu.Unlock()
		}

		relay := NewLogRelay(callback)
		relay.originalLogOutput = &originalBuffer
		relay.active = true

		// Write some data
		testData := []byte("test log message\n")
		n, err := relay.Write(testData)

		assert.NoError(t, err)
		assert.Equal(t, len(testData), n)

		// Check that data was written to original output
		assert.Contains(t, originalBuffer.String(), "test log message")

		// Restore original output
		log.SetOutput(originalOutput)
	})

	t.Run("write when not active", func(t *testing.T) {
		var originalBuffer bytes.Buffer
		originalOutput := log.Writer()
		log.SetOutput(&originalBuffer)

		callback := func(logText string) {}
		relay := NewLogRelay(callback)
		relay.originalLogOutput = &originalBuffer
		relay.active = false

		testData := []byte("test log message\n")
		n, err := relay.Write(testData)

		assert.NoError(t, err)
		assert.Equal(t, len(testData), n)

		// Should still write to original output
		assert.Contains(t, originalBuffer.String(), "test log message")

		// Restore original output
		log.SetOutput(originalOutput)
	})
}

func TestLogRelayIntegration(t *testing.T) {
	t.Run("full integration test", func(t *testing.T) {
		// Save original log output
		originalOutput := log.Writer()

		var receivedLogs []string
		var mu sync.Mutex

		callback := func(logText string) {
			mu.Lock()
			receivedLogs = append(receivedLogs, logText)
			mu.Unlock()
		}

		relay := NewLogRelay(callback)

		// Start the relay
		relay.Start()

		// Give time for the goroutine to start
		time.Sleep(10 * time.Millisecond)

		// Log some messages
		log.Println("Test message 1")
		log.Println("Test message 2")

		// Give time for messages to be processed
		time.Sleep(50 * time.Millisecond)

		// Stop the relay
		relay.Stop()

		// Check that we received some logs
		mu.Lock()
		assert.True(t, len(receivedLogs) > 0, "Should have received some log messages")
		mu.Unlock()

		// Restore original output
		log.SetOutput(originalOutput)
	})
}

func TestLogRelayBufferManagement(t *testing.T) {
	t.Run("buffer management with multiple lines", func(t *testing.T) {
		var lastLogText string
		var mu sync.Mutex

		callback := func(logText string) {
			mu.Lock()
			lastLogText = logText
			mu.Unlock()
		}

		relay := NewLogRelay(callback)
		relay.active = true

		// Start the processing goroutine
		relay.wg.Add(1)
		go relay.processLogMessages()

		// Send multiple messages
		for i := 0; i < 25; i++ { // More than maxLogLinesForDialog (20)
			relay.logMessageChannel <- "Log line " + string(rune('A'+i%26)) + "\n"
		}

		// Close channel and wait for processing to complete
		close(relay.logMessageChannel)
		relay.wg.Wait()

		// Check that buffer was managed correctly
		mu.Lock()
		lines := strings.Split(lastLogText, "\n")
		assert.True(t, len(lines) <= maxLogLinesForDialog,
			"Buffer should not exceed maxLogLinesForDialog lines")
		mu.Unlock()
	})
}

func TestLogRelayConstants(t *testing.T) {
	t.Run("maxLogLinesForDialog constant", func(t *testing.T) {
		assert.Equal(t, 20, maxLogLinesForDialog)
	})
}
