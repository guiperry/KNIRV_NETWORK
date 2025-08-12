package app

import (
	"context"
	"testing"
	"time"
)

func TestNewApp(t *testing.T) {
	tempDir := t.TempDir()

	app, err := NewApp(tempDir, 8080)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	if app == nil {
		t.Fatal("Expected app to be created")
	}

	if app.graphchain == nil {
		t.Error("Expected graphchain to be initialized")
	}

	if app.storage == nil {
		t.Error("Expected storage to be set")
	}

	if app.nrvSystem == nil {
		t.Error("Expected NRV system to be initialized")
	}

	if app.rpc == nil {
		t.Error("Expected RPC server to be initialized")
	}

	if app.logger == nil {
		t.Error("Expected logger to be initialized")
	}
}

func TestAppStartAndStop(t *testing.T) {
	tempDir := t.TempDir()

	app, err := NewApp(tempDir, 8081)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Test start and stop cycle
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start the app in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- app.Start(ctx)
	}()

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Stop the app
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stopCancel()

	err = app.Stop(stopCtx)
	if err != nil {
		t.Errorf("Failed to stop app: %v", err)
	}

	// Wait for start to complete
	select {
	case err := <-done:
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("Unexpected error from Start: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Start did not complete in time")
	}
}

func TestAppConfiguration(t *testing.T) {
	tempDir := t.TempDir()

	app, err := NewApp(tempDir, 9000)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Test that the app was created with the correct configuration
	if app.GetConfig() == nil {
		t.Error("Expected config to be initialized")
	}

	// Test that storage directory was created
	if app.storage == nil {
		t.Error("Expected storage to be initialized")
	}
}

func TestAppMultipleInstances(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	// Create two app instances with different ports
	app1, err := NewApp(tempDir1, 8083)
	if err != nil {
		t.Fatalf("Failed to create first app: %v", err)
	}

	app2, err := NewApp(tempDir2, 8084)
	if err != nil {
		t.Fatalf("Failed to create second app: %v", err)
	}

	// Verify they are separate instances
	if app1 == app2 {
		t.Error("Expected different app instances")
	}

	if app1.storage == app2.storage {
		t.Error("Expected different storage instances")
	}
}
