package app

import (
	"context"
	"testing"
	"time"
)

func TestNewApp(t *testing.T) {
	tempDir := t.TempDir()

	app, err := NewApp(tempDir, 8080, false)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}
	t.Cleanup(func() {
		app.Stop(context.Background())
	})

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

	app, err := NewApp(tempDir, 8081, false)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	startErrChan := make(chan error, 1)
	go func() {
		startErrChan <- app.Start(appCtx)
	}()

	// Give the app a moment to start up
	time.Sleep(100 * time.Millisecond)

	// Now stop the app, which should cause app.Start to return
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	err = app.Stop(stopCtx)
	if err != nil {
		t.Errorf("Failed to stop app: %v", err)
	}
	appCancel() // Explicitly cancel the app's context to unblock app.Start

	// Wait for app.Start to return after being stopped
	select {
	case startErr := <-startErrChan:
		if startErr != nil && startErr != context.Canceled && startErr != context.DeadlineExceeded {
			t.Errorf("App.Start returned an unexpected error: %v", startErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("App.Start did not return after Stop was called")
	}
}

func TestAppConfiguration(t *testing.T) {
	tempDir := t.TempDir()

	app, err := NewApp(tempDir, 9000, false)
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
	app1, err := NewApp(tempDir1, 8083, false)
	if err != nil {
		t.Fatalf("Failed to create first app: %v", err)
	}

	app2, err := NewApp(tempDir2, 8084, false)
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
