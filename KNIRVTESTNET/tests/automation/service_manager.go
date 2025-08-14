package automation

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// ServiceManager handles individual service lifecycle
type ServiceManager struct {
	Name     string
	Endpoint string
	Port     int
	Status   ServiceStatus
	Health   HealthMetrics
	Process  *ProcessManager
	Config   ServiceConfig
}

// ProcessManager handles process execution
type ProcessManager struct {
	Command string
	Args    []string
	WorkDir string
	Env     []string
	Process *exec.Cmd
}

// ServiceConfig holds service-specific configuration
type ServiceConfig struct {
	StartCommand    string
	StartArgs       []string
	WorkingDir      string
	HealthEndpoint  string
	StartupTimeout  time.Duration
	ShutdownTimeout time.Duration
	Environment     map[string]string
}

// HealthMetrics tracks service health information
type HealthMetrics struct {
	LastCheck    time.Time
	ResponseTime time.Duration
	Status       string
	ErrorCount   int
	Uptime       time.Duration
}

// NewServiceManager creates a new service manager
func NewServiceManager(serviceName string) *ServiceManager {
	config := getServiceConfig(serviceName)

	return &ServiceManager{
		Name:     serviceName,
		Endpoint: fmt.Sprintf("http://localhost:%d", config.Port),
		Port:     config.Port,
		Status:   ServiceStopped,
		Health:   HealthMetrics{},
		Process:  &ProcessManager{},
		Config:   config,
	}
}

// getServiceConfig returns configuration for each service
func getServiceConfig(serviceName string) ServiceConfig {
	configs := map[string]ServiceConfig{
		"knirv-root": {
			StartCommand:    "./scripts/start-knirvroot.sh",
			WorkingDir:      "../",
			HealthEndpoint:  "/health",
			Port:            1317,
			StartupTimeout:  30 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		"knirvchain": {
			StartCommand:    "./scripts/start-knirvchain.sh",
			WorkingDir:      "../",
			HealthEndpoint:  "/health",
			Port:            8090,
			StartupTimeout:  30 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		"knirvgraph": {
			StartCommand:    "./scripts/start-knirvgraph.sh",
			WorkingDir:      "../",
			HealthEndpoint:  "/height",
			Port:            8082,
			StartupTimeout:  25 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		"knirv-nexus": {
			StartCommand:    "./scripts/start-knirvnexus.sh",
			WorkingDir:      "../",
			HealthEndpoint:  "/health",
			Port:            8084,
			StartupTimeout:  20 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		"knirv-router": {
			StartCommand:    "./scripts/start-knirvrouter.sh",
			WorkingDir:      "../",
			HealthEndpoint:  "/health",
			Port:            5001,
			StartupTimeout:  15 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		"knirv-gateway": {
			StartCommand:    "./scripts/start-knirvgateway.sh",
			WorkingDir:      "../",
			HealthEndpoint:  "/health",
			Port:            8087,
			StartupTimeout:  20 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
	}

	if config, exists := configs[serviceName]; exists {
		return config
	}

	// Default configuration
	return ServiceConfig{
		StartCommand:    fmt.Sprintf("./scripts/start-%s.sh", serviceName),
		WorkingDir:      "../",
		HealthEndpoint:  "/health",
		Port:            8080,
		StartupTimeout:  30 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
}

// Start starts the service
func (sm *ServiceManager) Start(ctx context.Context) error {
	if sm.Status == ServiceRunning {
		return nil
	}

	sm.Status = ServiceStarting

	// Prepare command
	cmd := exec.CommandContext(ctx, sm.Config.StartCommand, sm.Config.StartArgs...)
	cmd.Dir = sm.Config.WorkingDir

	// Set environment variables
	if sm.Config.Environment != nil {
		for key, value := range sm.Config.Environment {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		sm.Status = ServiceFailed
		return fmt.Errorf("failed to start service %s: %w", sm.Name, err)
	}

	sm.Process.Process = cmd
	sm.Status = ServiceRunning

	return nil
}

// Stop stops the service
func (sm *ServiceManager) Stop(ctx context.Context) error {
	if sm.Status == ServiceStopped {
		return nil
	}

	if sm.Process.Process != nil {
		// Create timeout context for graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(ctx, sm.Config.ShutdownTimeout)
		defer cancel()

		// Try graceful shutdown first
		if err := sm.Process.Process.Process.Signal(os.Interrupt); err == nil {
			// Wait for graceful shutdown
			done := make(chan error, 1)
			go func() {
				done <- sm.Process.Process.Wait()
			}()

			select {
			case <-shutdownCtx.Done():
				// Force kill if graceful shutdown times out
				sm.Process.Process.Process.Kill()
				sm.Process.Process.Wait()
			case <-done:
				// Graceful shutdown completed
			}
		} else {
			// Force kill if signal fails
			sm.Process.Process.Process.Kill()
			sm.Process.Process.Wait()
		}
	}

	sm.Status = ServiceStopped
	return nil
}

// WaitForHealth waits for the service to become healthy
func (sm *ServiceManager) WaitForHealth(ctx context.Context, timeout time.Duration) error {
	healthCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-healthCtx.Done():
			return fmt.Errorf("service %s health check timeout", sm.Name)
		case <-ticker.C:
			if healthy, err := sm.CheckHealth(ctx); healthy {
				return nil
			} else if err != nil {
				// Log error but continue trying
				fmt.Printf("Health check error for %s: %v\n", sm.Name, err)
			}
		}
	}
}

// CheckHealth performs a health check on the service
func (sm *ServiceManager) CheckHealth(ctx context.Context) (bool, error) {
	startTime := time.Now()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	healthURL := fmt.Sprintf("%s%s", sm.Endpoint, sm.Config.HealthEndpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		sm.Health.ErrorCount++
		return false, err
	}
	defer resp.Body.Close()

	// Update health metrics
	sm.Health.LastCheck = time.Now()
	sm.Health.ResponseTime = time.Since(startTime)

	if resp.StatusCode == http.StatusOK {
		sm.Health.Status = "healthy"
		return true, nil
	}

	sm.Health.Status = "unhealthy"
	sm.Health.ErrorCount++
	return false, fmt.Errorf("health check failed with status %d", resp.StatusCode)
}

// GetMetrics returns current service metrics
func (sm *ServiceManager) GetMetrics() ServiceMetrics {
	return ServiceMetrics{
		Name:        sm.Name,
		Status:      sm.Status,
		Health:      sm.Health,
		Port:        sm.Port,
		Endpoint:    sm.Endpoint,
		LastUpdated: time.Now(),
	}
}

// ServiceMetrics holds service performance metrics
type ServiceMetrics struct {
	Name        string
	Status      ServiceStatus
	Health      HealthMetrics
	Port        int
	Endpoint    string
	LastUpdated time.Time
}

// IsHealthy returns true if the service is healthy
func (sm *ServiceManager) IsHealthy() bool {
	return sm.Status == ServiceRunning && sm.Health.Status == "healthy"
}

// GetUptime returns the service uptime
func (sm *ServiceManager) GetUptime() time.Duration {
	if sm.Status != ServiceRunning {
		return 0
	}
	return time.Since(sm.Health.LastCheck)
}

// Restart restarts the service
func (sm *ServiceManager) Restart(ctx context.Context) error {
	if err := sm.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop service %s: %w", sm.Name, err)
	}

	// Wait a moment before restarting
	time.Sleep(2 * time.Second)

	if err := sm.Start(ctx); err != nil {
		return fmt.Errorf("failed to start service %s: %w", sm.Name, err)
	}

	return sm.WaitForHealth(ctx, sm.Config.StartupTimeout)
}
